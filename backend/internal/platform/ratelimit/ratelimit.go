// Package ratelimit implements the fixed-window rate limiter of
// docs/api/limits.md §1 using in-memory counters.
//
// Each window is a counter incremented on each request. Exceeding a limit
// yields 429 RATE_LIMITED with Retry-After = time left in the window.
package ratelimit

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
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

type windowEntry struct {
	count     int
	expiresAt time.Time
}

// Limiter counts events in memory.
type Limiter struct {
	mu      sync.Mutex
	windows map[string]*windowEntry
	clock   clock.Clock
	logger  *slog.Logger
}

// New returns an in-memory limiter.
func New(c clock.Clock, logger *slog.Logger) *Limiter {
	lim := &Limiter{
		windows: make(map[string]*windowEntry),
		clock:   c,
		logger:  logger,
	}
	return lim
}

// Allow counts one event for subject under l.
func (lim *Limiter) Allow(ctx context.Context, l Limit, subject string) (Result, error) {
	now := lim.clock.Now()
	window := now.UnixMilli() / l.Window.Milliseconds()
	resetAt := time.UnixMilli((window + 1) * l.Window.Milliseconds())
	key := fmt.Sprintf("%s:%s:%d", l.Name, subject, window)

	lim.mu.Lock()
	defer lim.mu.Unlock()

	// Opportunistic cleanup of expired windows if map gets large
	if len(lim.windows) > 1000 {
		for k, w := range lim.windows {
			if now.After(w.expiresAt) {
				delete(lim.windows, k)
			}
		}
	}

	entry, ok := lim.windows[key]
	if !ok {
		entry = &windowEntry{
			count:     0,
			expiresAt: resetAt.Add(l.Window),
		}
		lim.windows[key] = entry
	}
	entry.count++
	count := entry.count

	res := Result{Allowed: count <= l.Max, Count: count, Remaining: max(l.Max-count, 0)}
	if !res.Allowed {
		res.RetryAfter = resetAt.Sub(now)
	}
	return res, nil
}

// Check is Allow for handlers: nil when allowed, RATE_LIMITED when not.
// Any failure is logged and the event allowed (fail open).
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
