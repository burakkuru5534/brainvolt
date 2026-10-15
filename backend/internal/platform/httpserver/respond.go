package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strconv"

	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/i18n"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/logging"
)

// HandlerFunc is the signature of every API handler: it writes a success
// response itself and returns an error otherwise. A returned *apierror.Error
// becomes the matching envelope; anything else becomes 500 INTERNAL.
type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

// envelope is the error body (docs/api/conventions.md §6).
type envelope struct {
	Error envelopeError `json:"error"`
}

type envelopeError struct {
	Code      apierror.Code  `json:"code"`
	Message   string         `json:"message"`
	Details   map[string]any `json:"details,omitempty"`
	RequestID string         `json:"request_id"`
}

// WriteJSON writes v as a JSON response with the given status.
func WriteJSON(w http.ResponseWriter, status int, v any) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, err := w.Write(buf.Bytes())
	return err
}

// WriteError writes err as an error envelope in the request's locale and
// logs it: 5xx at error level with the cause, others at debug level.
func WriteError(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	ctx := r.Context()
	e := classify(err)
	status := e.Code.HTTPStatus()

	if status >= http.StatusInternalServerError {
		logger.ErrorContext(ctx, "request failed", "code", e.Code, "error", err)
	} else {
		logger.DebugContext(ctx, "request rejected", "code", e.Code, "error", err)
	}

	if e.RetryAfter > 0 {
		secs := int(math.Ceil(e.RetryAfter.Seconds()))
		w.Header().Set("Retry-After", strconv.Itoa(secs))
		if e.Code == apierror.RateLimited {
			e = e.With("retry_after_s", secs)
		}
	}
	locale := i18n.FromContext(ctx)
	w.Header().Set("Content-Language", string(locale))
	body := envelope{Error: envelopeError{
		Code:      e.Code,
		Message:   e.Message(locale),
		Details:   e.Details,
		RequestID: logging.RequestID(ctx),
	}}
	if werr := WriteJSON(w, status, body); werr != nil {
		logger.DebugContext(ctx, "writing error response failed", "error", werr)
	}
}

// classify maps any error to an *apierror.Error.
func classify(err error) *apierror.Error {
	if e := apierror.From(err); e != nil {
		if !e.Code.Known() || (e.Code.HTTPStatus() == http.StatusInternalServerError && e.Code != apierror.Internal) {
			// Unregistered or WebSocket-only code on REST: a server bug.
			return apierror.Wrap(apierror.Internal, err)
		}
		return e
	}
	var tooLarge *http.MaxBytesError
	switch {
	case errors.As(err, &tooLarge):
		return apierror.Wrap(apierror.PayloadTooLarge, err)
	case errors.Is(err, context.DeadlineExceeded):
		return apierror.Wrap(apierror.Unavailable, err).WithRetryAfter(retryAfterTimeout)
	default:
		return apierror.Wrap(apierror.Internal, err)
	}
}

// DecodeJSON reads one JSON value from the request body into v. Unknown
// fields are ignored (older servers must accept newer clients); a missing,
// malformed or trailing-garbage body is MALFORMED_REQUEST and a body over the
// route's limit is PAYLOAD_TOO_LARGE.
func DecodeJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(v); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return apierror.Wrap(apierror.PayloadTooLarge, err)
		}
		return apierror.Wrap(apierror.MalformedRequest, err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return apierror.Wrap(apierror.PayloadTooLarge, err)
		}
		return apierror.New(apierror.MalformedRequest).With("reason", "trailing data after JSON value")
	}
	return nil
}
