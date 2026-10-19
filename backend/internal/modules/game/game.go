// Package game is the public API of the game module (backend-modules.md
// §1): match configuration and the mode matrix, the match/round engine,
// match creation with its guards, the live match owners behind the
// WebSocket intents (match.*), and the persistence of rounds and results.
package game

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/content"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain/config"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain/engine"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/live"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/service"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/transport"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/realtime"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/team"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/redis"
)

// Event names other modules subscribe to.
const (
	EventMatchStarted   = service.EventMatchStarted
	EventMatchCompleted = service.EventMatchCompleted
	EventMatchCancelled = service.EventMatchCancelled
)

// Config is a match configuration (mode-matrix.md).
type Config = config.Config

// MatchType is a configuration's match type.
type MatchType = domain.MatchType

// Format is a configuration's format.
type Format = domain.Format

// CategoryMode is a configuration's category mode.
type CategoryMode = domain.CategoryMode

// Violation is a broken mode-matrix rule.
type Violation = config.Violation

// CreateRequest creates a match.
type CreateRequest = service.CreateRequest

// Member is one participant of a new match.
type Member = service.Member

// Created is a new match in `waiting` with the engine output of T1.
type Created = service.Created

// MatchCompleted is the payload of game.MatchCompleted.
type MatchCompleted = service.MatchCompleted

// SidePayload is a side of MatchCompleted.
type SidePayload = service.SidePayload

// ParticipantPayload is a participant of MatchCompleted.
type ParticipantPayload = service.ParticipantPayload

// RoundPayload is a played round of MatchCompleted.
type RoundPayload = service.RoundPayload

// MatchCancelled is the payload of game.MatchCancelled.
type MatchCancelled = service.MatchCancelled

// MatchStarted is the payload of game.MatchStarted.
type MatchStarted = service.MatchStarted

// Ended is a finished match as its live owner saw it.
type Ended = live.Ended

// EndedParticipant is a participant's final state.
type EndedParticipant = live.EndedParticipant

// Summary is a match's state and seating.
type Summary = service.Summary

// Seated is one participant of a Summary.
type Seated = service.Seated

// Values other modules compare with.
const (
	OriginMatchmaking     = domain.Matchmaking
	StatusCancelled       = domain.StatusCancelled
	StatusCompleted       = domain.StatusCompleted
	CancelNotAllReady     = domain.CancelNotAllReady
	CancelLeftBeforeStart = domain.CancelLeftBeforeStart
	ParticipantReady      = domain.PReady
	ParticipantLeft       = domain.PLeft
	ParticipantForfeited  = domain.PForfeited
)

// Origins of matches created by other modules.
const (
	OriginChallenge   = domain.Challenge
	OriginPrivateRoom = domain.PrivateRoom
	OriginRematch     = domain.Rematch
)

// SeasonProvider is implemented by ranking (Phase 7); until then ranked
// matches play in the preseason (D-064).
type SeasonProvider = service.SeasonProvider

// SanctionChecker is implemented by moderation (Phase 8); until then
// nobody is blocked (D-064).
type SanctionChecker = service.SanctionChecker

// ValidateConfig returns every mode-matrix violation of c after the
// transformations (no parent).
func ValidateConfig(c Config) []Violation { return config.Validate(config.Transform(c, nil)) }

// Deps are the module's collaborators. Realtime, Profiles and Instance
// are needed for live matches (the API process); tools that only create
// or persist matches may leave them nil.
type Deps struct {
	DB        *pgxpool.Pool
	Redis     *redis.Client
	Content   *content.Module
	Identity  *identity.Module
	Profiles  *profile.Module
	Realtime  *realtime.Module
	Seasons   SeasonProvider
	Sanctions SanctionChecker
	Teams     *team.Module
	Events    *events.Bus
	Clock     clock.Clock
	Logger    *slog.Logger
	Instance  string
}

// Module is the game module. In the API it is also an app component: Run
// receives match commands forwarded from other instances.
type Module struct {
	svc      *service.Service
	live     *live.Manager
	identity *identity.Module
}

// New builds the module and, with a realtime gateway, registers the
// match.* intents, match stream resume, disconnect and RTT hooks.
func New(d Deps) *Module {
	sd := service.Deps{
		DB: d.DB, Redis: d.Redis, Questions: d.Content, Users: d.Identity, Cards: d.Profiles, Seasons: d.Seasons,
		Sanctions: d.Sanctions, Events: d.Events, Clock: d.Clock, Logger: d.Logger,
	}
	if d.Teams != nil {
		sd.Teams = d.Teams
	}
	svc := service.New(sd)
	m := &Module{svc: svc, identity: d.Identity}
	if d.Realtime != nil {
		m.live = live.New(live.Deps{
			Redis: d.Redis, Persister: svc, RT: d.Realtime, Texts: d.Content, Cards: d.Profiles,
			Clock: d.Clock, Logger: d.Logger, Instance: d.Instance, Random: service.CryptoRandom{},
		})
		d.Realtime.Handle("match", m.live.Handle)
		d.Realtime.OnResume("match", m.live.Resume)
		d.Realtime.OnDisconnect(m.live.OnDisconnect)
		d.Realtime.OnPong(m.live.OnPong)
		d.Realtime.OnLive(m.live.OnLive)
	}
	return m
}

// Create checks the T1 guards and stores a new match in `waiting`.
func (m *Module) Create(ctx context.Context, r CreateRequest) (*Created, error) {
	return m.svc.Create(ctx, r)
}

// Start creates a match and hands it to a live owner on this instance:
// participants receive match.created and join with match.join.
func (m *Module) Start(ctx context.Context, r CreateRequest) (*Created, error) {
	c, err := m.svc.Create(ctx, r)
	if err != nil {
		return nil, err
	}
	if err := m.live.Start(ctx, c.Match, c.Output); err != nil {
		m.svc.Abandon(context.WithoutCancel(ctx), c.Match)
		return nil, err
	}
	return c, nil
}

// Practice starts a development Quick Duel against a bot played by the
// server (D-066; only the dev composition exposes it).
func (m *Module) Practice(ctx context.Context, user uuid.UUID, locale string) (*Created, error) {
	name := "Pratik Botu"
	if locale == "en" {
		name = "Practice Bot"
	}
	bot, err := m.identity.EnsureBot(ctx, "practice-"+user.String(), name, locale)
	if err != nil {
		return nil, err
	}
	c, err := m.svc.Create(ctx, CreateRequest{
		Config: Config{MatchType: domain.QuickDuel, Format: domain.Format1v1, Origin: domain.Challenge},
		Sides:  [2][]Member{{{UserID: user, Locale: locale}}, {{UserID: bot, Locale: locale}}},
	})
	if err != nil {
		return nil, err
	}
	if err := m.live.Start(ctx, c.Match, c.Output, bot); err != nil {
		m.svc.Abandon(context.WithoutCancel(ctx), c.Match)
		return nil, err
	}
	return c, nil
}

// OnEnd registers fn for matches finishing on this instance (completed or
// cancelled), called after the live state is cleaned up.
func (m *Module) OnEnd(fn func(ctx context.Context, e Ended)) {
	if m.live != nil {
		m.live.OnEnd(fn)
	}
}

// CategoryExists reports an active top-level category (R12).
func (m *Module) CategoryExists(ctx context.Context, id uuid.UUID) (bool, error) {
	return m.svc.CategoryExists(ctx, id)
}

// Summary returns a match's state and participants.
func (m *Module) Summary(ctx context.Context, id uuid.UUID) (Summary, error) {
	return m.svc.Summary(ctx, id)
}

// LiveMatch returns the user's live match, if any.
func (m *Module) LiveMatch(ctx context.Context, user uuid.UUID) (uuid.UUID, bool) {
	return m.live.LiveMatch(ctx, user)
}

// Apply persists one engine output (rounds, transitions, results, events).
func (m *Module) Apply(ctx context.Context, match *engine.Match, out engine.Output) error {
	return m.svc.Apply(ctx, match, out)
}

// Register adds game's REST routes; ident authenticates them.
func (m *Module) Register(s *httpserver.Server, ident *identity.Module) {
	transport.New(m.svc, ident).Register(s)
}

// Close stops this instance's live owners without releasing their leases,
// as a crashed process would (tests); Run's end does the same.
func (m *Module) Close() { m.live.Close() }

// Name implements the app component interface.
func (m *Module) Name() string { return "game" }

// Run receives forwarded match commands until ctx ends, then stops the
// local owners (their leases expire; edge-cases §4).
func (m *Module) Run(ctx context.Context) error { return m.live.Run(ctx) }
