package ratelimit

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/config"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/redis"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/testenv"
)

var discard = slog.New(slog.DiscardHandler)

// windowStart is aligned to a 10 s and a 1 min window boundary.
var windowStart = time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)

func newLimiter(t *testing.T) (*Limiter, *clock.Fake, *redis.Client) {
	t.Helper()
	rdb, err := redis.Connect(t.Context(), testenv.RedisURL(t), "brainvolt-test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = rdb.Close() })
	fake := clock.NewFake(windowStart)
	return New(rdb, fake, discard), fake, rdb
}

func TestAllowCountsPerWindowAndSubject(t *testing.T) {
	lim, fake, rdb := newLimiter(t)
	ctx := t.Context()
	l := Limit{Name: "test_" + ids.NewString(), Max: 3, Window: 10 * time.Second}

	for i := 1; i <= 3; i++ {
		res, err := lim.Allow(ctx, l, "user-1")
		if err != nil || !res.Allowed || res.Count != i || res.Remaining != 3-i {
			t.Fatalf("event %d: %+v, %v", i, res, err)
		}
	}
	fake.Advance(4 * time.Second)
	res, err := lim.Allow(ctx, l, "user-1")
	if err != nil || res.Allowed || res.RetryAfter != 6*time.Second || res.Remaining != 0 {
		t.Fatalf("4th event: %+v, %v; want blocked with 6s left", res, err)
	}

	// Another subject has its own counter.
	if res, _ := lim.Allow(ctx, l, "user-2"); !res.Allowed || res.Count != 1 {
		t.Fatalf("other subject: %+v", res)
	}

	// The next window starts fresh.
	fake.Advance(6 * time.Second)
	if res, _ := lim.Allow(ctx, l, "user-1"); !res.Allowed || res.Count != 1 {
		t.Fatalf("next window: %+v", res)
	}

	// Keys expire on their own after one window.
	key := redis.Key("rl", l.Name, "user-1", "")
	keys, err := rdb.Keys(ctx, key+"*").Result()
	if err != nil || len(keys) == 0 {
		t.Fatalf("keys: %v %v", keys, err)
	}
	for _, k := range keys {
		ttl, err := rdb.PTTL(ctx, k).Result()
		if err != nil || ttl <= 0 || ttl > l.Window {
			t.Errorf("%s TTL = %v, want 0 < ttl <= %v", k, ttl, l.Window)
		}
		rdb.Del(ctx, k)
	}
}

func TestCheckReturnsRateLimited(t *testing.T) {
	lim, _, _ := newLimiter(t)
	l := Limit{Name: "test_" + ids.NewString(), Max: 1, Window: time.Minute}
	if err := lim.Check(t.Context(), l, "s"); err != nil {
		t.Fatalf("first: %v", err)
	}
	err := lim.Check(t.Context(), l, "s")
	e := apierror.From(err)
	if e == nil || e.Code != apierror.RateLimited || e.RetryAfter != time.Minute || e.Details["limit"] != l.Name {
		t.Fatalf("second: %v", err)
	}
}

func TestFailsOpenWhenRedisDown(t *testing.T) {
	rdb, err := redis.Connect(t.Context(), testenv.RedisURL(t), "brainvolt-test")
	if err != nil {
		t.Fatal(err)
	}
	_ = rdb.Close() // every command now fails
	lim := New(rdb, clock.NewFake(windowStart), discard)

	if _, err := lim.Allow(t.Context(), APIPerIP, "1.2.3.4"); err == nil {
		t.Fatal("Allow on a closed client succeeded")
	}
	if err := lim.Check(t.Context(), APIPerIP, "1.2.3.4"); err != nil {
		t.Fatalf("Check must fail open, got %v", err)
	}
}

func TestMiddleware(t *testing.T) {
	lim, _, _ := newLimiter(t)
	l := Limit{Name: "test_" + ids.NewString(), Max: 2, Window: time.Minute}

	cfg := config.HTTP{ReadTimeout: time.Second, WriteTimeout: 2 * time.Second, HandlerTimeout: time.Second, ShutdownTimeout: time.Second}
	srv := httpserver.New(cfg, config.EnvDev, discard)
	srv.Use(lim.Middleware(l, ClientIP))
	srv.Handle("GET /v1/ping", func(w http.ResponseWriter, _ *http.Request) error {
		w.WriteHeader(http.StatusNoContent)
		return nil
	})

	call := func(path, ip string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
		req.RemoteAddr = ip + ":51234"
		req.Header.Set("Accept-Language", "en")
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		return rec
	}

	for range 2 {
		if rec := call("/v1/ping", "10.0.0.1"); rec.Code != http.StatusNoContent {
			t.Fatalf("allowed request = %d", rec.Code)
		}
	}
	rec := call("/v1/ping", "10.0.0.1")
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") != "60" {
		t.Fatalf("over limit = %d Retry-After=%q", rec.Code, rec.Header().Get("Retry-After"))
	}
	var body struct {
		Error struct {
			Code    string         `json:"code"`
			Message string         `json:"message"`
			Details map[string]any `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Error.Code != "RATE_LIMITED" || body.Error.Details["retry_after_s"] != float64(60) ||
		body.Error.Message != "Too many requests. Please wait a moment and try again." {
		t.Fatalf("body = %s", rec.Body.String())
	}

	// Unmatched paths count too (scanners), other IPs and health probes do not.
	if rec := call("/nope", "10.0.0.1"); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("unmatched path over limit = %d", rec.Code)
	}
	if rec := call("/v1/ping", "10.0.0.2"); rec.Code != http.StatusNoContent {
		t.Fatalf("other IP = %d", rec.Code)
	}
	if got := ClientIP(httptest.NewRequest(http.MethodGet, "/readyz", http.NoBody)); got != "" {
		t.Fatalf("ClientIP(/readyz) = %q, want exempt", got)
	}
}

func TestLimitsMatchDocument(t *testing.T) {
	// limits.md §1 / D-032 values; changing them needs the document first.
	want := map[string]Limit{
		"http_user": {Name: "http_user", Max: 60, Window: 10 * time.Second},
		"http_ip":   {Name: "http_ip", Max: 300, Window: time.Minute},
		"public_ip": {Name: "public_ip", Max: 60, Window: time.Minute},
		"session":   {Name: "session", Max: 20, Window: time.Hour},
	}
	for _, l := range []Limit{APIPerUser, APIPerIP, Unauthenticated, Session} {
		if want[l.Name] != l {
			t.Errorf("%s = %+v, want %+v", l.Name, l, want[l.Name])
		}
	}
}
