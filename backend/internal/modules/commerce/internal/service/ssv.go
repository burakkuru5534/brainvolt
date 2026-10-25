package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

// AdMobKeysURL lists the public keys AdMob signs SSV callbacks with
// (developers.google.com/admob/android/ssv).
const AdMobKeysURL = "https://www.gstatic.com/admob/reward/verifier-keys.json"

// KeySource returns AdMob's verifying key by id.
type KeySource interface {
	Key(ctx context.Context, id int64) (*ecdsa.PublicKey, error)
}

// HTTPKeys fetches AdMob's keys and caches them for a day; an unknown id
// refreshes the list at most once a minute.
type HTTPKeys struct {
	URL    string
	Client *http.Client

	mu      sync.Mutex
	keys    map[int64]*ecdsa.PublicKey
	fetched time.Time
}

// NewHTTPKeys returns a KeySource for AdMob's published keys.
func NewHTTPKeys() *HTTPKeys {
	return &HTTPKeys{URL: AdMobKeysURL, Client: &http.Client{Timeout: 5 * time.Second}}
}

// Key implements KeySource.
func (k *HTTPKeys) Key(ctx context.Context, id int64) (*ecdsa.PublicKey, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	age := time.Since(k.fetched)
	if key, ok := k.keys[id]; ok && age < 24*time.Hour {
		return key, nil
	}
	if k.keys != nil && age < time.Minute {
		if key, ok := k.keys[id]; ok {
			return key, nil
		}
		return nil, fmt.Errorf("ssv: unknown key %d", id)
	}
	keys, err := k.fetch(ctx)
	if err != nil {
		if key, ok := k.keys[id]; ok {
			return key, nil // keep serving the cached key while gstatic is down
		}
		return nil, err
	}
	k.keys, k.fetched = keys, time.Now()
	if key, ok := keys[id]; ok {
		return key, nil
	}
	return nil, fmt.Errorf("ssv: unknown key %d", id)
}

func (k *HTTPKeys) fetch(ctx context.Context) (map[int64]*ecdsa.PublicKey, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, k.URL, http.NoBody)
	if err != nil {
		return nil, err
	}
	res, err := k.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ssv: keys: HTTP %d", res.StatusCode)
	}
	var body struct {
		Keys []struct {
			KeyID int64  `json:"keyId"`
			PEM   string `json:"pem"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return nil, err
	}
	out := make(map[int64]*ecdsa.PublicKey, len(body.Keys))
	for _, kk := range body.Keys {
		key, err := ParsePublicKey(kk.PEM)
		if err != nil {
			return nil, err
		}
		out[kk.KeyID] = key
	}
	return out, nil
}

// ParsePublicKey reads an ECDSA public key in PEM.
func ParsePublicKey(p string) (*ecdsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(p))
	if block == nil {
		return nil, errors.New("ssv: bad PEM")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	key, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("ssv: not an ECDSA key")
	}
	return key, nil
}

// ErrBadSignature: the callback is not signed by AdMob.
var ErrBadSignature = errors.New("ssv: bad signature")

// VerifySSV checks an SSV callback's raw query: the signed content is
// everything before "&signature=", signature and key_id come last.
func (s *Service) VerifySSV(ctx context.Context, rawQuery string) (url.Values, error) {
	i := strings.Index(rawQuery, "&signature=")
	if i < 0 {
		return nil, ErrBadSignature
	}
	msg := rawQuery[:i]
	vals, err := url.ParseQuery(rawQuery)
	if err != nil {
		return nil, ErrBadSignature
	}
	sig, err := decodeB64(vals.Get("signature"))
	if err != nil {
		return nil, ErrBadSignature
	}
	keyID, err := strconv.ParseInt(vals.Get("key_id"), 10, 64)
	if err != nil {
		return nil, ErrBadSignature
	}
	if s.d.Keys == nil {
		return nil, errors.New("ssv: no key source")
	}
	key, err := s.d.Keys.Key(ctx, keyID)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256([]byte(msg))
	if !ecdsa.VerifyASN1(key, sum[:], sig) {
		return nil, ErrBadSignature
	}
	return vals, nil
}

func decodeB64(s string) ([]byte, error) {
	for _, enc := range []*base64.Encoding{base64.RawURLEncoding, base64.URLEncoding, base64.StdEncoding, base64.RawStdEncoding} {
		if b, err := enc.DecodeString(s); err == nil {
			return b, nil
		}
	}
	return nil, errors.New("ssv: bad base64")
}

// HandleSSV verifies a callback and grants the reward it names
// (custom_data = reward id, user_id = our user id). Callbacks without a
// reward (AdMob's "verify URL" test) and unknown rewards are accepted and
// ignored, so AdMob does not retry them; only a bad signature is refused.
func (s *Service) HandleSSV(ctx context.Context, rawQuery string) error {
	vals, err := s.VerifySSV(ctx, rawQuery)
	if err != nil {
		return err
	}
	id, err1 := uuid.Parse(vals.Get("custom_data"))
	user, err2 := uuid.Parse(vals.Get("user_id"))
	tx := vals.Get("transaction_id")
	if err1 != nil || err2 != nil || tx == "" {
		s.d.Logger.InfoContext(ctx, "commerce: ssv callback without a reward", "custom_data", vals.Get("custom_data"))
		return nil
	}
	ok, err := s.Grant(ctx, id, user, tx)
	if err != nil {
		return err
	}
	if !ok {
		s.d.Logger.WarnContext(ctx, "commerce: ssv callback for an unknown reward", "reward", id, "user", user)
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "23505"
}
