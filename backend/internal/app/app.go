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
	"sync/atomic"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/chat"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/content"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/creator"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/lobby"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/matchmaking"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/moderation"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/notification"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/progression"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/ranking"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/realtime"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/social"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/team"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/tournament"

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

// NewAPI builds the composition for cmd/api: Postgres
// connection, Firebase token verification, the event bus, the HTTP server
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

	sqlDB := stdlib.OpenDBFromPool(db)
	schema, err := migrations.NewProvider(sqlDB)
	if err != nil {
		_ = sqlDB.Close()
		db.Close()
		return nil, err
	}

	verifier, err := firebase.New(ctx, cfg.Firebase.ProjectID, cfg.Firebase.AuthEmulatorHost)
	if err != nil {
		_ = sqlDB.Close()
		db.Close()
		return nil, err
	}
	var (
		bus     *events.Bus
		extra   []Component
		closers []func()
		live    = newLivePusher(logger)
	)
	if cfg.EmbedWorker {
		// One process does it all (D-096): the API also works the jobs, on
		// a small pool of its own with the worker's statement timeout, so a
		// single Cloud Run service can scale to zero.
		jobsDB, err := postgres.Connect(ctx, postgres.Options{
			URL:              cfg.Postgres.URL,
			StatementTimeout: cfg.Postgres.WorkerStatementTimeout,
			ApplicationName:  "brainvolt-api-jobs",
		})
		if err != nil {
			_ = sqlDB.Close()
			db.Close()
			return nil, err
		}
		j, err := newJobs(ctx, cfg, jobsDB, logger, live)
		if err != nil {
			jobsDB.Close()
			_ = sqlDB.Close()
			db.Close()
			return nil, err
		}
		bus, extra, closers = j.bus, []Component{j.runner()}, []func(){jobsDB.Close}
	} else {
		// Insert-only River client: the API queues event deliveries, the worker runs them.
		bus = newBus(db, logger, nil, nil)
		inserter, err := river.NewClient(riverpgxv5.New(db), &river.Config{Logger: logger})
		if err != nil {
			_ = sqlDB.Close()
			db.Close()
			return nil, err
		}
		bus.Attach(inserter)
	}

	clk := clock.Real{}
	srv := httpserver.New(cfg.HTTP, cfg.Env, logger)
	limiter := ratelimit.New(clk, logger)
	srv.Use(limiter.Middleware(ratelimit.APIPerIP, ratelimit.ClientIP))
	health.Register(srv, logger,
		health.Check{Name: "postgres", Fn: db.Ping},
		health.Check{Name: "migrations", Fn: func(ctx context.Context) error { return migrations.Check(ctx, schema) }},
	)
	mods := registerModules(srv, moduleDeps{db: db, verifier: verifier, bus: bus, limiter: limiter,
		idem: idempotency.New(db, logger), clock: clk, logger: logger, instance: ids.New().String(), env: cfg.Env})

	// The jobs' personal messages go out through this process's hub.
	live.Bind(mods.realtime)
	a := New(RoleAPI, logger, append([]Component{srv, mods.realtime, mods.game, mods.matchmaking, mods.lobby, mods.moderation}, extra...)...)
	// Closers run in order: the jobs pool goes after the components stop.
	a.closers = append(a.closers, db.Close, func() { _ = sqlDB.Close() })
	a.closers = append(a.closers, closers...)
	return a, nil
}

// moduleDeps are the shared collaborators of the business modules.
type moduleDeps struct {
	db       *pgxpool.Pool
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
	ranking     *ranking.Module
	moderation  *moderation.Module
	progression *progression.Module
	game        *game.Module
	matchmaking *matchmaking.Module
	lobby       *lobby.Module
	tournament  *tournament.Module
}

// registerModules builds the modules and their routes. The wiring follows
// backend-modules.md §2: profile implements identity's ProfileProvisioner;
// identity authenticates WebSocket sessions for realtime; game runs live
// matches on the gateway with content texts and profile cards.
func registerModules(srv *httpserver.Server, d moduleDeps) modules {
	prof := profile.New(d.db, d.clock)
	ident := identity.New(identity.Deps{
		DB: d.db, Verifier: d.verifier, Profiles: prof.Provisioner(), Events: d.bus,
		Limiter: d.limiter, Idempotency: d.idem, Clock: d.clock, Logger: d.logger,
	})
	ident.Register(srv)
	prof.Register(srv, ident, d.limiter)
	notification.New(notification.Deps{DB: d.db, Clock: d.clock, Logger: d.logger}).Register(srv, ident)
	cont := content.New(d.db, d.clock, d.bus)
	cont.Register(srv, ident, d.idem)
	rt := realtime.New(realtime.Deps{Auth: ident, Clock: d.clock, Logger: d.logger, Instance: d.instance})
	rt.Register(srv)
	soc := social.New(social.Deps{DB: d.db, Identity: ident, Profiles: prof, Presence: rt, Events: d.bus, Clock: d.clock})
	soc.Register(srv, ident, d.idem)
	tm := team.New(team.Deps{DB: d.db, Identity: ident, Profiles: prof, Social: soc, Events: d.bus, Clock: d.clock})
	tm.Register(srv, ident, d.idem)
	mod := moderation.New(moderation.Deps{DB: d.db, Identity: ident, Content: cont, Events: d.bus, Clock: d.clock, Logger: d.logger})
	mod.Register(srv, ident, d.idem)
	// Sockets closed for malformed messages or flooding are cheat signals (D-084).
	rt.OnViolation(mod.ProtocolViolation)
	rk := ranking.New(ranking.Deps{DB: d.db, Profiles: prof, Teams: tm, Social: soc, Clock: d.clock, Logger: d.logger})
	rk.Register(srv, ident)
	prog := progression.New(progression.Deps{DB: d.db, Ranking: rk, Clock: d.clock, Logger: d.logger})
	prog.Register(srv, ident, d.idem)
	g := game.New(game.Deps{
		DB: d.db, Content: cont, Identity: ident, Profiles: prof, Realtime: rt, Teams: tm, Seasons: rk, Sanctions: mod,
		Events: d.bus, Clock: d.clock, Logger: d.logger, Instance: d.instance,
	})
	g.Register(srv, ident, d.idem)
	mm := matchmaking.New(matchmaking.Deps{
		DB: d.db, Game: g, Realtime: rt, Ratings: ratings{rk}, Social: soc, Sanctions: mod, Teams: tm, Profiles: prof, Clock: d.clock, Logger: d.logger, Practice: d.env == config.EnvDev,
	})
	mm.Register(srv, ident, d.idem)
	chat.New(chat.Deps{DB: d.db, Teams: tm, Profiles: prof, Muter: mod, Realtime: rt, Events: d.bus, Clock: d.clock}).Register(srv, ident)
	cr := creator.New(creator.Deps{DB: d.db, Content: cont, Identity: ident, Profiles: prof, Clock: d.clock})
	cr.Register(srv, ident, d.idem)
	lob := lobby.New(lobby.Deps{
		DB: d.db, Identity: ident, Profiles: prof, Social: soc, Teams: tm, Game: g, Quizzes: quizzes{cr}, Realtime: rt,
		Events: d.bus, Clock: d.clock, Logger: d.logger,
	})
	lob.Register(srv, ident, d.idem)
	mm.SetLobbies(lobbySeats{lob}) // team formats gather in waiting lobbies (D-094)
	tour := tournament.New(tournament.Deps{
		DB: d.db, Game: g, Profiles: prof, Teams: tm, Pusher: pusher{rt}, Events: d.bus, Clock: d.clock, Logger: d.logger,
	})
	tour.Register(srv, ident, d.idem)
	return modules{
		identity: ident, profile: prof, content: cont, realtime: rt, social: soc, team: tm, ranking: rk, moderation: mod, progression: prog,
		game: g, matchmaking: mm, lobby: lob, tournament: tour,
	}
}

// lobbySeats adapts lobby's waiting lobbies to matchmaking (D-094);
// matchmaking may not import lobby (backend-modules.md §2).
type lobbySeats struct{ *lobby.Module }

func (l lobbySeats) SeatMatchmaking(ctx context.Context, g matchmaking.LobbyGroup) (uuid.UUID, error) {
	return l.Module.SeatMatchmaking(ctx, lobby.MatchmakingGroup(g))
}

// pusher sends personal messages through realtime (ranking and
// progression awards from the worker).
type pusher struct{ *realtime.Module }

func (p pusher) Push(ctx context.Context, user uuid.UUID, typ string, data any) error {
	return p.Send(ctx, realtime.Target{User: user}, realtime.Frame{T: typ, D: data})
}

// quizzes adapts creator to lobby's Quizzes.
type quizzes struct{ *creator.Module }

func (q quizzes) Playable(ctx context.Context, user, id uuid.UUID) (lobby.Quiz, error) {
	p, err := q.Module.Playable(ctx, user, id)
	return lobby.Quiz{SetID: p.SetID, Length: p.Length, WindowMS: p.WindowMS, Locale: p.Locale, Author: p.Author, Published: p.Published}, err
}

// ratings adapts ranking to matchmaking's RatingReader.
type ratings struct{ *ranking.Module }

func (r ratings) Rating(ctx context.Context, user uuid.UUID, bucket string) (matchmaking.Rating, error) {
	v, err := r.Module.Rating(ctx, user, bucket)
	return matchmaking.Rating{Value: v.Value, RD: v.RD}, err
}

// newBus returns the event bus with every subscription. The API and the
// worker build it the same way: the API to address deliveries, the worker
// to run them. Consumers subscribe here as their phases arrive
// (backend-modules.md §3.2 event catalogue).
func newBus(db *pgxpool.Pool, logger *slog.Logger, push notification.Sender, live *livePusher) *events.Bus {
	if live == nil {
		live = newLivePusher(logger)
	}
	bus := events.NewBus()
	// content counts answer statistics of completed matches (schema-content §5).
	stats := content.New(db, clock.Real{}, bus)
	bus.Subscribe(game.EventMatchCompleted, content.SubscriberMatchStats, stats.OnMatchCompleted)
	// lobby marks challenges whose match was cancelled as failed (schema-social §3.1).
	bus.Subscribe(game.EventMatchCancelled, lobby.SubscriberChallengeFailed, lobby.OnMatchCancelled(db))
	// ranking rates, ranks and scores completed matches (progression-formulas §1–4).
	rk := newRanking(db, bus, logger, live)
	bus.Subscribe(game.EventMatchCompleted, ranking.SubscriberMatchResults, rk.OnMatchCompleted)
	bus.Subscribe(identity.EventAccountTierChanged, ranking.SubscriberTierChanged, rk.OnTierChanged)
	// progression: XP, streaks, badges and quests (progression-formulas §5–9).
	prog := progression.New(progression.Deps{DB: db, Pusher: live, Ranking: rk, Clock: clock.Real{}, Logger: logger})
	bus.Subscribe(game.EventMatchCompleted, progression.SubscriberMatchAwards, prog.OnMatchCompleted)
	bus.Subscribe(ranking.EventSeasonEnded, progression.SubscriberSeasonReward, prog.OnSeasonEnded)
	// Daily Challenge: the daily board and the run's XP / streak / quest (D-040).
	bus.Subscribe(game.EventDailyRunCompleted, ranking.SubscriberDailyBoard, rk.OnDailyRunCompleted)
	bus.Subscribe(game.EventDailyRunCompleted, progression.SubscriberDailyRewards, prog.OnDailyRunCompleted)
	// moderation: anti-cheat signals from matches (schema-moderation §4, D-084).
	mod := moderation.New(moderation.Deps{DB: db, Clock: clock.Real{}, Logger: logger})
	bus.Subscribe(game.EventMatchCompleted, moderation.SubscriberMatchSignals, mod.OnMatchCompleted)
	bus.Subscribe(game.EventMatchCancelled, moderation.SubscriberVoidSignals, mod.OnMatchCancelled)
	// notification: inbox and push for what concerns a user (schema-platform §1).
	notif := newNotification(db, logger, push, live)
	bus.Subscribe(social.EventFriendRequestSent, notification.SubscriberFriendRequest, notif.OnFriendRequestSent)
	bus.Subscribe(social.EventFriendshipCreated, notification.SubscriberFriendship, notif.OnFriendshipCreated)
	bus.Subscribe(lobby.EventChallengeSent, notification.SubscriberChallengeSent, notif.OnChallengeSent)
	bus.Subscribe(lobby.EventChallengeAccepted, notification.SubscriberChallengeAccepts, notif.OnChallengeAccepted)
	bus.Subscribe(team.EventCoupleRequested, notification.SubscriberCoupleRequest, notif.OnCoupleRequested)
	bus.Subscribe(moderation.EventSanctionApplied, notification.SubscriberSanction, notif.OnSanctionApplied)
	bus.Subscribe(chat.EventMessageSent, notification.SubscriberChatMessage, notif.OnChatMessage)
	// tournament: results move the bracket on (schema-platform §3, D-090).
	tour := newTournament(db, bus, logger, live)
	bus.Subscribe(game.EventMatchCompleted, tournament.SubscriberMatchResults, tour.OnMatchCompleted)
	bus.Subscribe(game.EventMatchCancelled, tournament.SubscriberMatchCancelled, tour.OnMatchCancelled)
	bus.Subscribe(tournament.EventRoundScheduled, notification.SubscriberTournamentRound, notif.OnTournamentRound)
	bus.Subscribe(tournament.EventFinished, notification.SubscriberTournamentFinished, notif.OnTournamentFinished)
	bus.Subscribe(tournament.EventFinished, progression.SubscriberTournamentRewards, prog.OnTournamentFinished)
	return bus
}

// newTournament builds tournament for the worker: no game (matches are
// created from the API), bracket changes pushed through realtime.
func newTournament(db *pgxpool.Pool, bus *events.Bus, logger *slog.Logger, live *livePusher) *tournament.Module {
	return tournament.New(tournament.Deps{
		DB: db, Profiles: profile.New(db, clock.Real{}), Teams: team.New(team.Deps{DB: db, Clock: clock.Real{}}),
		Pusher: live, Events: bus, Clock: clock.Real{}, Logger: logger,
	})
}

// newNotification builds notification for delivery in the worker: names
// from profile, inbox.new through realtime, push through push.
func newNotification(db *pgxpool.Pool, logger *slog.Logger, push notification.Sender, live *livePusher) *notification.Module {
	return notification.New(notification.Deps{
		DB: db, Profiles: profile.New(db, clock.Real{}), Sender: push, Pusher: live, Clock: clock.Real{}, Logger: logger,
	})
}

// logSender stands in for FCM when no credentials are configured: the
// push is only logged (development).
type logSender struct{ logger *slog.Logger }

func (l logSender) Send(ctx context.Context, p notification.Push) (string, error) {
	l.logger.InfoContext(ctx, "push (not sent: no FCM credentials)", "title", p.Title, "body", p.Body, "kind", p.Data["kind"])
	return "", nil
}

// fcmSender adapts platform/firebase to notification.Sender.
type fcmSender struct{ m *firebase.Messenger }

func (f fcmSender) Send(ctx context.Context, p notification.Push) (string, error) {
	id, err := f.m.Send(ctx, p.Token, p.Title, p.Body, p.Data)
	if errors.Is(err, firebase.ErrTokenInvalid) {
		return "", fmt.Errorf("%w: %w", notification.ErrTokenInvalid, err)
	}
	return id, err
}

// newRanking builds ranking for worker use: country and team reads,
// SeasonEnded on bus and awards pushed through realtime; no routes.
func newRanking(db *pgxpool.Pool, bus *events.Bus, logger *slog.Logger, live *livePusher) *ranking.Module {
	return ranking.New(ranking.Deps{
		DB: db, Profiles: profile.New(db, clock.Real{}), Teams: team.New(team.Deps{DB: db, Clock: clock.Real{}}),
		Events: bus, Pusher: live, Clock: clock.Real{}, Logger: logger,
	})
}

// livePusher delivers the jobs' personal messages (match.awards, inbox.new,
// tournament frames) to connected players. Since Redis is gone there is no
// hub shared between processes: it works once Bind hands it the API's
// realtime module, i.e. when the jobs run inside the API process (D-096).
// Unbound (a separate worker process) it drops them with a debug log; the
// app then reads awards and the inbox over REST.
type livePusher struct {
	rt     atomic.Pointer[realtime.Module]
	logger *slog.Logger
}

func newLivePusher(logger *slog.Logger) *livePusher { return &livePusher{logger: logger} }

// Bind starts delivering through rt.
func (p *livePusher) Bind(rt *realtime.Module) { p.rt.Store(rt) }

func (p *livePusher) Push(ctx context.Context, user uuid.UUID, typ string, data any) error {
	if rt := p.rt.Load(); rt != nil {
		return pusher{rt}.Push(ctx, user, typ, data)
	}
	p.logger.DebugContext(ctx, "no realtime hub in this process; personal message dropped", "type", typ, "user_id", user)
	return nil
}

// NewWorker builds the composition for cmd/worker: a Postgres pool with the
// longer worker statement timeout and the River job runner (event
// deliveries now, scheduled jobs in later phases).
//
// Note: the health HTTP server is NOT included here. cmd/worker/main.go binds
// $PORT immediately at start-up (before this wiring runs) so that Cloud Run
// health probes pass within the startup timeout.
func NewWorker(ctx context.Context, cfg config.Config, logger *slog.Logger) (*App, error) {
	db, err := postgres.Connect(ctx, postgres.Options{
		URL:              cfg.Postgres.URL,
		StatementTimeout: cfg.Postgres.WorkerStatementTimeout,
		ApplicationName:  "brainvolt-worker",
	})
	if err != nil {
		return nil, err
	}
	j, err := newJobs(ctx, cfg, db, logger, newLivePusher(logger))
	if err != nil {
		db.Close()
		return nil, err
	}
	a := New(RoleWorker, logger, j.runner())
	a.closers = append(a.closers, db.Close)
	return a, nil
}

// jobs is the worker role's River client: outbox deliveries to every
// subscriber and the periodic jobs. cmd/worker runs it alone; with
// BRAINVOLT_EMBED_WORKER the API process runs it too (single Cloud Run
// service that scales to zero, D-096).
type jobs struct {
	bus    *events.Bus
	client *river.Client[pgx.Tx]
}

func (j jobs) runner() events.Runner {
	return events.Runner{Client: j.client, StopTimeout: 30 * time.Second}
}

// newJobs builds the push sender, the event bus with every subscriber,
// the periodic jobs and the River client that works them, on db.
func newJobs(ctx context.Context, cfg config.Config, db *pgxpool.Pool, logger *slog.Logger, live *livePusher) (jobs, error) {
	push, err := newPushSender(ctx, cfg, logger)
	if err != nil {
		return jobs{}, err
	}
	bus := newBus(db, logger, push, live)
	workers := river.NewWorkers()
	bus.AddWorker(workers)
	periodic := addJobs(workers, newRanking(db, bus, logger, live), content.New(db, clock.Real{}, bus), newNotification(db, logger, push, live),
		chat.New(chat.Deps{DB: db, Clock: clock.Real{}}), newTournament(db, bus, logger, live))
	client, err := river.NewClient(riverpgxv5.New(db), &river.Config{
		Queues:       map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: 10}},
		Workers:      workers,
		PeriodicJobs: periodic,
		Logger:       logger,
	})
	if err != nil {
		return jobs{}, err
	}
	bus.Attach(client)
	return jobs{bus: bus, client: client}, nil
}

// newPushSender is FCM when credentials (or, in prod, Application Default
// Credentials) are available, otherwise a sender that only logs.
func newPushSender(ctx context.Context, cfg config.Config, logger *slog.Logger) (notification.Sender, error) {
	var push notification.Sender = logSender{logger}
	cred := cfg.Firebase.MessagingCredentialsJSON
	if cred == "" {
		cred = cfg.Firebase.MessagingCredentialsFile
	}
	if cred != "" {
		m, err := firebase.NewMessenger(ctx, cfg.Firebase.ProjectID, cred)
		if err == nil {
			return fcmSender{m}, nil
		}
		if cfg.Env != config.EnvProd {
			return nil, err
		}
		logger.WarnContext(ctx, "failed to load configured FCM credentials, falling back to Application Default Credentials (ADC)", "error", err)
		mADC, errADC := firebase.NewMessenger(ctx, cfg.Firebase.ProjectID, "")
		if errADC != nil {
			return nil, fmt.Errorf("firebase messaging (both configured credentials and ADC failed): %w", errADC)
		}
		return fcmSender{mADC}, nil
	}
	if cfg.Env == config.EnvProd {
		m, err := firebase.NewMessenger(ctx, cfg.Firebase.ProjectID, "")
		if err != nil {
			logger.WarnContext(ctx, "firebase messaging ADC init failed, falling back to log sender", "error", err)
		} else {
			logger.InfoContext(ctx, "firebase messaging initialized with Application Default Credentials (ADC)")
			push = fcmSender{m}
		}
	}
	return push, nil
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
