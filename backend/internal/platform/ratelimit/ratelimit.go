// Package ratelimit implements the fixed-window rate limiter of
// docs/api/limits.md §1 on Redis counters rl:<name>:<subject>:<window_id>
// (docs/domain/redis-keys.md §7.1).
//
// Each window is a counter incremented with INCR and given the window's
// length as TTL on first use, so keys clean themselves up. Exceeding a limit
// yields 429 RATE_LIMITED with Retry-After = time left in the window.
//
// If Redis is unreachable the limiter fails open (allows the request and
// logs): rate limits protect the service, they must not take it down;
// /readyz already reports Redis as failed (D-056).
package ratelimit

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/redis"
)

// Limit is at most Max events per Window for one subject.
type Limit struct {
	Name   string
	Max    int
	Window time.Duration
}

// General limits (limits.md §1, D-032). Feature limits (friend requests,
// reports, …) are declared by the module that owns the feature.
var (
	// APIPerUser applies to every authenticated request.
	APIPerUser = Limit{Name: "http_user", Max: 60, Window: 10 * time.Second}
	// APIPerIP applies to every request.
	APIPerIP = Limit{Name: "http_ip", Max: 300, Window: time.Minute}
	// Unauthenticated applies to public endpoints (/v1/meta, /v1/credits).
	Unauthenticated = Limit{Name: "public_ip", Max: 60, Window: time.Minute}
	// Session applies to POST /v1/session per installation.
	Session = Limit{Name: "session", Max: 20, Window: time.Hour}
)

// Result describes one counted event.
type Result struct {
	Allowed    bool
	Count      int
	Remaining  int
	RetryAfter time.Duration // time until the window resets; set when not allowed
}

// Limiter counts events in Redis.
type Limiter struct {
	rdb    *redis.Client
	clock  clock.Clock
	logger *slog.Logger
}

// New returns a limiter.
func New(rdb *redis.Client, c clock.Clock, logger *slog.Logger) *Limiter {
	return &Limiter{rdb: rdb, clock: c, logger: logger}
}

// Allow counts one event for subject under l.
func (lim *Limiter) Allow(ctx context.Context, l Limit, subject string) (Result, error) {
	now := lim.clock.Now()
	window := now.UnixMilli() / l.Window.Milliseconds()
	resetAt := time.UnixMilli((window + 1) * l.Window.Milliseconds())
	key := redis.Key("rl", l.Name, subject, strconv.FormatInt(window, 10))

	pipe := lim.rdb.TxPipeline()
	incr := pipe.Incr(ctx, key)
	pipe.ExpireNX(ctx, key, l.Window)
	if _, err := pipe.Exec(ctx); err != nil {
		return Result{}, fmt.Errorf("ratelimit %s: %w", l.Name, err)
	}
	count := int(incr.Val())
	res := Result{Allowed: count <= l.Max, Count: count, Remaining: max(l.Max-count, 0)}
	if !res.Allowed {
		res.RetryAfter = resetAt.Sub(now)
	}
	return res, nil
}

// Check is Allow for handlers: nil when allowed, RATE_LIMITED when not.
// A Redis failure is logged and the event allowed (fail open).
func (lim *Limiter) Check(ctx context.Context, l Limit, subject string) error {
	res, err := lim.Allow(ctx, l, subject)
	if err != nil {
		lim.logger.WarnContext(ctx, "rate limiter unavailable, allowing request", "limit", l.Name, "error", err)
		return nil
	}
	if !res.Allowed {
		return apierror.New(apierror.RateLimited).With("limit", l.Name).WithRetryAfter(res.RetryAfter)
	}
	return nil
}

// Middleware applies l to every request whose subject is non-empty.
func (lim *Limiter) Middleware(l Limit, subject func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if s := subject(r); s != "" {
				if err := lim.Check(r.Context(), l, s); err != nil {
					httpserver.WriteError(w, r, lim.logger, err)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ClientIP is the subject for per-IP limits: the TCP peer address. Health
// probes are exempt. Behind a load balancer this becomes the trusted
// X-Forwarded-For hop, configured with the deployment (Phase 9).
func ClientIP(r *http.Request) string {
	if strings.HasPrefix(r.URL.Path, "/healthz") || strings.HasPrefix(r.URL.Path, "/readyz") {
		return ""
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
