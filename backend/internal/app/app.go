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
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/content"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/lobby"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/matchmaking"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/notification"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/realtime"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/social"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/team"

	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/config"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/firebase"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/health"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/idempotency"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/logging"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ratelimit"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/redis"
	"github.com/burakkuru5534/brainvolt/backend/migrations"
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
// connections, Firebase token verification, the event bus, the HTTP server
// with the per-IP rate limit, /healthz, /readyz and the identity, profile
// and notification routes and the WebSocket gateway. Matchmaking loops
// (Phase 5) join it later.
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

	sqlDB := stdlib.OpenDBFromPool(db)
	schema, err := migrations.NewProvider(sqlDB)
	if err != nil {
		_ = sqlDB.Close()
		db.Close()
		_ = rdb.Close()
		return nil, err
	}

	verifier, err := firebase.New(ctx, cfg.Firebase.ProjectID, cfg.Firebase.AuthEmulatorHost)
	if err != nil {
		_ = sqlDB.Close()
		db.Close()
		_ = rdb.Close()
		return nil, err
	}
	// Insert-only River client: the API queues event deliveries, the worker runs them.
	bus := newBus(db)
	inserter, err := river.NewClient(riverpgxv5.New(db), &river.Config{Logger: logger})
	if err != nil {
		_ = sqlDB.Close()
		db.Close()
		_ = rdb.Close()
		return nil, err
	}
	bus.Attach(inserter)

	clk := clock.Real{}
	srv := httpserver.New(cfg.HTTP, cfg.Env, logger)
	limiter := ratelimit.New(rdb, clk, logger)
	srv.Use(limiter.Middleware(ratelimit.APIPerIP, ratelimit.ClientIP))
	health.Register(srv, logger,
		health.Check{Name: "postgres", Fn: db.Ping},
		health.Check{Name: "redis", Fn: func(ctx context.Context) error { return redis.Ping(ctx, rdb) }},
		health.Check{Name: "migrations", Fn: func(ctx context.Context) error { return migrations.Check(ctx, schema) }},
	)
	mods := registerModules(srv, moduleDeps{db: db, rdb: rdb, verifier: verifier, bus: bus, limiter: limiter,
		idem: idempotency.New(rdb, logger), clock: clk, logger: logger, instance: ids.New().String(), env: cfg.Env})

	a := New(RoleAPI, logger, srv, mods.realtime, mods.game, mods.matchmaking, mods.lobby)
	a.closers = append(a.closers, db.Close, func() { _ = sqlDB.Close() }, func() { _ = rdb.Close() })
	return a, nil
}

// moduleDeps are the shared collaborators of the business modules.
type moduleDeps struct {
	db       *pgxpool.Pool
	rdb      *redis.Client
	verifier identity.Verifier
	bus      *events.Bus
	limiter  *ratelimit.Limiter
	idem     *idempotency.Store
	clock    clock.Clock
	logger   *slog.Logger
	// instance identifies this API process (realtime delivery, match owners).
	instance string
	// env enables development-only features (practice bot, D-066).
	env config.Environment
}

// modules are the built modules a process runs besides the HTTP server.
type modules struct {
	identity    *identity.Module
	profile     *profile.Module
	content     *content.Module
	realtime    *realtime.Module
	social      *social.Module
	team        *team.Module
	game        *game.Module
	matchmaking *matchmaking.Module
	lobby       *lobby.Module
}

// registerModules builds the modules and their routes. The wiring follows
// backend-modules.md §2: profile implements identity's ProfileProvisioner;
// identity authenticates WebSocket sessions for realtime; game runs live
// matches on the gateway with content texts and profile cards.
func registerModules(srv *httpserver.Server, d moduleDeps) modules {
	prof := profile.New(d.db, d.clock)
	ident := identity.New(identity.Deps{
		DB: d.db, Redis: d.rdb, Verifier: d.verifier, Profiles: prof.Provisioner(), Events: d.bus,
		Limiter: d.limiter, Idempotency: d.idem, Clock: d.clock, Logger: d.logger,
	})
	ident.Register(srv)
	prof.Register(srv, ident, d.limiter)
	notification.New(d.db, d.clock).Register(srv, ident)
	cont := content.New(d.db, d.clock, d.bus)
	cont.Register(srv, ident, d.idem)
	rt := realtime.New(realtime.Deps{Redis: d.rdb, Auth: ident, Clock: d.clock, Logger: d.logger, Instance: d.instance})
	rt.Register(srv)
	soc := social.New(social.Deps{DB: d.db, Identity: ident, Profiles: prof, Presence: rt, Events: d.bus, Clock: d.clock})
	soc.Register(srv, ident, d.idem)
	tm := team.New(team.Deps{DB: d.db, Identity: ident, Profiles: prof, Social: soc, Events: d.bus, Clock: d.clock})
	tm.Register(srv, ident, d.idem)
	g := game.New(game.Deps{
		DB: d.db, Redis: d.rdb, Content: cont, Identity: ident, Profiles: prof, Realtime: rt,
		Events: d.bus, Clock: d.clock, Logger: d.logger, Instance: d.instance,
	})
	g.Register(srv, ident)
	mm := matchmaking.New(matchmaking.Deps{
		Redis: d.rdb, Game: g, Realtime: rt, Blocks: soc, Clock: d.clock, Logger: d.logger, Practice: d.env == config.EnvDev,
	})
	mm.Register(srv, ident, d.idem)
	lob := lobby.New(lobby.Deps{
		DB: d.db, Redis: d.rdb, Identity: ident, Profiles: prof, Social: soc, Game: g, Realtime: rt,
		Events: d.bus, Clock: d.clock, Logger: d.logger,
	})
	lob.Register(srv, ident, d.idem)
	return modules{identity: ident, profile: prof, content: cont, realtime: rt, social: soc, team: tm, game: g, matchmaking: mm, lobby: lob}
}

// newBus returns the event bus with every subscription. The API and the
// worker build it the same way: the API to address deliveries, the worker
// to run them. Consumers subscribe here as their phases arrive
// (backend-modules.md §3.2 event catalogue).
func newBus(db *pgxpool.Pool) *events.Bus {
	bus := events.NewBus()
	// content counts answer statistics of completed matches (schema-content §5).
	stats := content.New(db, clock.Real{}, bus)
	bus.Subscribe(game.EventMatchCompleted, content.SubscriberMatchStats, stats.OnMatchCompleted)
	// lobby marks challenges whose match was cancelled as failed (schema-social §3.1).
	bus.Subscribe(game.EventMatchCancelled, lobby.SubscriberChallengeFailed, lobby.OnMatchCancelled(db))
	return bus
}

// NewWorker builds the composition for cmd/worker: a Postgres pool with the
// longer worker statement timeout and the River job runner (event
// deliveries now, scheduled jobs in later phases).
func NewWorker(ctx context.Context, cfg config.Config, logger *slog.Logger) (*App, error) {
	db, err := postgres.Connect(ctx, postgres.Options{
		URL:              cfg.Postgres.URL,
		StatementTimeout: cfg.Postgres.WorkerStatementTimeout,
		ApplicationName:  "brainvolt-worker",
	})
	if err != nil {
		return nil, err
	}
	bus := newBus(db)
	workers := river.NewWorkers()
	bus.AddWorker(workers)
	client, err := river.NewClient(riverpgxv5.New(db), &river.Config{
		Queues:  map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: 10}},
		Workers: workers,
		Logger:  logger,
	})
	if err != nil {
		db.Close()
		return nil, err
	}
	bus.Attach(client)
	a := New(RoleWorker, logger, events.Runner{Client: client, StopTimeout: 30 * time.Second})
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
