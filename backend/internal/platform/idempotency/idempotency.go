// Package idempotency implements the Idempotency-Key contract of
// docs/api/conventions.md §9 for POSTs that create something or spend a
// limited resource.
//
// The first request with a key runs the handler; its response (any status
// below 500) is stored in Redis under idem:<user_id>:<key> for 24 h and
// replayed for every repeat with header Idempotent-Replayed: true. A repeat
// with a different method, path or body gets 409 IDEMPOTENCY_KEY_REUSED; a
// repeat while the first is still running gets 409
// IDEMPOTENT_REQUEST_IN_PROGRESS. 5xx responses are not stored, so the
// client can retry. If Redis is unreachable the request is refused with 503
// (fail closed): running it without the guarantee could spend twice.
package idempotency

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/logging"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/redis"
)

const (
	// Header is the request header carrying the client-generated UUID.
	Header = "Idempotency-Key"
	// ReplayedHeader marks a stored response sent again.
	ReplayedHeader = "Idempotent-Replayed"
	// Retention is how long a completed response is kept (conventions.md §9).
	Retention = 24 * time.Hour
	// pendingTTL frees a key whose first request died mid-way (crash);
	// it is well above the handler deadline.
	pendingTTL = time.Minute
)

// replayedHeaders are the response headers stored and replayed.
var replayedHeaders = []string{"Content-Type", "Content-Language", "Location", "ETag"}

// Mode says whether a route requires the header.
type Mode bool

const (
	// Required means requests without Idempotency-Key fail with VALIDATION_FAILED
	// (friend requests, challenges, room creation, report, purchases, …).
	Required Mode = true
	// Optional means the header is honoured when present.
	Optional Mode = false
)

type record struct {
	State   string            `json:"state"` // "pending" or "done"
	Hash    string            `json:"hash"`
	Status  int               `json:"status,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    []byte            `json:"body,omitempty"`
}

// Store keeps idempotency records in Redis.
type Store struct {
	rdb    *redis.Client
	logger *slog.Logger
}

// New returns a store.
func New(rdb *redis.Client, logger *slog.Logger) *Store {
	return &Store{rdb: rdb, logger: logger}
}

// Wrap guards h. The caller is identified by the user id in the request
// context (set by authentication), so it must wrap authenticated routes.
func (s *Store) Wrap(mode Mode, h httpserver.HandlerFunc) httpserver.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		key := r.Header.Get(Header)
		if key == "" {
			if mode == Required {
				return apierror.Validation(apierror.FieldError{Field: Header, Rule: "required"})
			}
			return h(w, r)
		}
		if _, err := ids.Parse(key); err != nil {
			return apierror.Validation(apierror.FieldError{Field: Header, Rule: "uuid"})
		}
		ctx := r.Context()
		user := logging.UserID(ctx)
		if user == "" {
			return errors.New("idempotency: route is not authenticated (no user id in context)")
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			return err // over the body limit → PAYLOAD_TOO_LARGE
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		hash := fingerprint(r, body)
		rkey := redis.Key("idem", user, key)

		claimed, existing, err := s.claim(r, rkey, hash)
		if err != nil {
			return apierror.Wrap(apierror.Unavailable, err).WithRetryAfter(time.Second)
		}
		if !claimed {
			return s.replay(w, existing, hash)
		}

		rec := newCapture()
		if err := h(rec, r); err != nil {
			httpserver.WriteError(rec, r, s.logger, err)
		}
		s.finish(r, rkey, hash, rec)
		rec.copyTo(w)
		return nil
	}
}

// fingerprint identifies "the same request": method, path and body.
func fingerprint(r *http.Request, body []byte) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s %s\n", r.Method, r.URL.Path)
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}

// claim stores a pending record if the key is new. Otherwise it returns the
// existing record. A record that expires between SET and GET is retried.
func (s *Store) claim(r *http.Request, rkey, hash string) (bool, record, error) {
	ctx := r.Context()
	pending, err := json.Marshal(record{State: "pending", Hash: hash})
	if err != nil {
		return false, record{}, err
	}
	for range 3 {
		ok, err := s.rdb.SetNX(ctx, rkey, pending, pendingTTL).Result()
		if err != nil {
			return false, record{}, err
		}
		if ok {
			return true, record{}, nil
		}
		raw, err := s.rdb.Get(ctx, rkey).Bytes()
		if errors.Is(err, redis.ErrNil) {
			continue
		}
		if err != nil {
			return false, record{}, err
		}
		var existing record
		if err := json.Unmarshal(raw, &existing); err != nil {
			return false, record{}, fmt.Errorf("idempotency: corrupt record: %w", err)
		}
		return false, existing, nil
	}
	return false, record{}, errors.New("idempotency: key kept expiring while claiming")
}

func (s *Store) replay(w http.ResponseWriter, rec record, hash string) error {
	if rec.Hash != hash {
		return apierror.New(apierror.IdempotencyKeyReused)
	}
	if rec.State != "done" {
		return apierror.New(apierror.IdempotencyInProgress).WithRetryAfter(time.Second)
	}
	for k, v := range rec.Headers {
		w.Header().Set(k, v)
	}
	w.Header().Set(ReplayedHeader, "true")
	w.WriteHeader(rec.Status)
	_, err := w.Write(rec.Body)
	return err
}

// finish stores a completed response for Retention, or frees the key after
// a 5xx so that a retry runs again.
func (s *Store) finish(r *http.Request, rkey, hash string, c *capture) {
	ctx := r.Context()
	if c.status >= http.StatusInternalServerError {
		if err := s.rdb.Del(ctx, rkey).Err(); err != nil {
			s.logger.WarnContext(ctx, "idempotency: releasing key failed", "error", err)
		}
		return
	}
	headers := map[string]string{}
	for _, k := range replayedHeaders {
		if v := c.header.Get(k); v != "" {
			headers[k] = v
		}
	}
	done, err := json.Marshal(record{State: "done", Hash: hash, Status: c.status, Headers: headers, Body: c.body.Bytes()})
	if err == nil {
		err = s.rdb.Set(ctx, rkey, done, Retention).Err()
	}
	if err != nil {
		// The response is still sent; a repeat may then run the handler again.
		s.logger.ErrorContext(ctx, "idempotency: storing response failed", "error", err)
	}
}

// capture buffers a response so it can be stored before it is sent.
type capture struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func newCapture() *capture { return &capture{header: http.Header{}} }

func (c *capture) Header() http.Header { return c.header }

func (c *capture) WriteHeader(code int) {
	if c.status == 0 {
		c.status = code
	}
}

func (c *capture) Write(b []byte) (int, error) {
	if c.status == 0 {
		c.status = http.StatusOK
	}
	return c.body.Write(b)
}

func (c *capture) copyTo(w http.ResponseWriter) {
	for k, v := range c.header {
		w.Header()[k] = v
	}
	if c.status == 0 {
		c.status = http.StatusOK
	}
	w.WriteHeader(c.status)
	_, _ = w.Write(c.body.Bytes())
}
