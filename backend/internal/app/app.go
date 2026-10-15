// Package app is the composition root of the backend (docs/architecture/repo-layout.md).
//
// It is the only place that knows every module: it builds configuration,
// infrastructure (platform/*) and modules, wires cross-module interfaces
// (e.g. ranking.RatingReader for matchmaking, backend-modules.md §2) and
// decides which long-running components each process runs (§5).
// cmd/* binaries only call NewAPI / NewWorker and Run.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/config"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/health"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/logging"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ratelimit"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/redis"
)

// Role names the process a composition is built for (backend-modules.md §5).
type Role string

const (
	// RoleAPI serves REST + WebSocket and owns live matches and matchmaking loops.
	RoleAPI Role = "api"
	// RoleWorker runs the outbox dispatcher and scheduled jobs.
	RoleWorker Role = "worker"
)

// Component is a long-running part of a process: an HTTP server, a job
// runner, a matchmaking loop… Run blocks until ctx is cancelled (clean stop,
// returns nil) or the component fails (returns the error).
type Component interface {
	Name() string
	Run(ctx context.Context) error
}

// App runs the components of one process and stops them together.
type App struct {
	role       Role
	logger     *slog.Logger
	components []Component
	closers    []func()
}

// New builds an App for role running the given components.
func New(role Role, logger *slog.Logger, components ...Component) *App {
	return &App{role: role, logger: logger, components: components}
}

// NewAPI builds the composition for cmd/api: Postgres and Redis
// connections, the HTTP server with the per-IP rate limit, /healthz and
// /readyz. The WebSocket
// gateway (Phase 4) and matchmaking loops (Phase 5) join it later.
func NewAPI(ctx context.Context, cfg config.Config, logger *slog.Logger) (*App, error) {
	db, err := postgres.Connect(ctx, postgres.Options{
		URL:              cfg.Postgres.URL,
		StatementTimeout: cfg.Postgres.StatementTimeout,
		ApplicationName:  "brainvolt-api",
	})
	if err != nil {
		return nil, err
	}
	rdb, err := redis.Connect(ctx, cfg.Redis.URL, "brainvolt-api")
	if err != nil {
		db.Close()
		return nil, err
	}

	srv := httpserver.New(cfg.HTTP, cfg.Env, logger)
	limiter := ratelimit.New(rdb, clock.Real{}, logger)
	srv.Use(limiter.Middleware(ratelimit.APIPerIP, ratelimit.ClientIP))
	health.Register(srv, logger,
		health.Check{Name: "postgres", Fn: db.Ping},
		health.Check{Name: "redis", Fn: func(ctx context.Context) error { return redis.Ping(ctx, rdb) }},
	)

	a := New(RoleAPI, logger, srv)
	a.closers = append(a.closers, db.Close, func() { _ = rdb.Close() })
	return a, nil
}

// NewWorker builds the composition for cmd/worker: a Postgres pool with the
// longer worker statement timeout. The River job runner joins it in 1.5.2.
func NewWorker(ctx context.Context, cfg config.Config, logger *slog.Logger) (*App, error) {
	db, err := postgres.Connect(ctx, postgres.Options{
		URL:              cfg.Postgres.URL,
		StatementTimeout: cfg.Postgres.WorkerStatementTimeout,
		ApplicationName:  "brainvolt-worker",
	})
	if err != nil {
		return nil, err
	}
	a := New(RoleWorker, logger)
	a.closers = append(a.closers, db.Close)
	return a, nil
}

// Role reports which process this App was built for.
func (a *App) Role() Role { return a.role }

// Run starts every component and blocks until ctx is cancelled or one
// component fails. A failure cancels all other components; Run waits for all
// of them to return before it returns. A stop caused by ctx returns nil.
func (a *App) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	a.logger.InfoContext(ctx, "starting", "role", a.role, "components", len(a.components))

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		failures []error
	)
	for _, c := range a.components {
		wg.Go(func() {
			if err := c.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
				err = fmt.Errorf("%s: %w", c.Name(), err)
				mu.Lock()
				failures = append(failures, err)
				mu.Unlock()
				cancel(err)
			}
		})
	}

	<-ctx.Done()
	a.logger.InfoContext(ctx, "stopping", "role", a.role)
	wg.Wait()
	// Connections close only after every component has stopped using them.
	for i := len(a.closers) - 1; i >= 0; i-- {
		a.closers[i]()
	}

	if err := errors.Join(failures...); err != nil {
		a.logger.ErrorContext(ctx, "stopped with error", "role", a.role, "error", err)
		return err
	}
	a.logger.InfoContext(ctx, "stopped", "role", a.role)
	return nil
}

// Main is the shared body of the cmd/* entry points: it loads and validates
// the configuration, builds the logger and the App with build, runs it until
// SIGINT/SIGTERM, and returns the process exit code (0 = clean stop,
// 1 = configuration, start-up or run failure).
func Main(build func(context.Context, config.Config, *slog.Logger) (*App, error)) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		// No logger options yet: report in the default JSON format.
		logging.New(os.Stderr, logging.Options{Format: logging.FormatJSON}).
			Error("invalid configuration", "error", err)
		return 1
	}
	logger := logging.New(os.Stderr, logging.Options{Format: cfg.Log.Format, Level: cfg.Log.Level})
	logger.Info("configuration loaded", "config", cfg)

	a, err := build(ctx, cfg, logger)
	if err != nil {
		logger.Error("start-up failed", "error", err)
		return 1
	}
	if err := a.Run(ctx); err != nil {
		return 1
	}
	return 0
}
