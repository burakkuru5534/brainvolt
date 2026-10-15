// Package health serves the liveness and readiness endpoints
// (docs/api/conventions.md §12): GET /healthz answers while the process is
// up; GET /readyz answers 200 only when every dependency check passes
// (Postgres, Redis, and from roadmap 1.5.1 on, applied migrations).
package health

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
)

// checkTimeout bounds every dependency check, so /readyz answers quickly
// even when a dependency hangs.
const checkTimeout = 2 * time.Second

// Check is one readiness dependency.
type Check struct {
	Name string
	Fn   func(ctx context.Context) error
}

// Register adds /healthz and /readyz to the server.
func Register(s *httpserver.Server, logger *slog.Logger, checks ...Check) {
	s.Handle("GET /healthz", func(w http.ResponseWriter, _ *http.Request) error {
		return httpserver.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	s.Handle("GET /readyz", func(w http.ResponseWriter, r *http.Request) error {
		results, ok := run(r.Context(), logger, checks)
		if !ok {
			// Only check names and ok/failed go out; causes are logged.
			return apierror.New(apierror.Unavailable).With("checks", results).WithRetryAfter(5 * time.Second)
		}
		return httpserver.WriteJSON(w, http.StatusOK, map[string]any{"status": "ready", "checks": results})
	})
}

// run executes all checks concurrently.
func run(ctx context.Context, logger *slog.Logger, checks []Check) (map[string]string, bool) {
	results := make(map[string]string, len(checks))
	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)
	ok := true
	for _, c := range checks {
		wg.Go(func() {
			cctx, cancel := context.WithTimeout(ctx, checkTimeout)
			defer cancel()
			status := "ok"
			if err := c.Fn(cctx); err != nil {
				status = "failed"
				logger.WarnContext(ctx, "readiness check failed", "check", c.Name, "error", err)
			}
			mu.Lock()
			defer mu.Unlock()
			results[c.Name] = status
			ok = ok && status == "ok"
		})
	}
	wg.Wait()
	return results, ok
}
