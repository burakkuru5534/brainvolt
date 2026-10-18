// Package game is the public API of the game module (backend-modules.md
// §1): match configuration and the mode matrix, the match/round engine,
// match creation with its guards and the persistence of rounds and
// results. The live match owner and the WebSocket intents arrive in
// Phase 4; until then the engine runs inside this module's tests.
package game

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/content"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain/config"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain/engine"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/service"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
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

// MatchCancelled is the payload of game.MatchCancelled.
type MatchCancelled = service.MatchCancelled

// MatchStarted is the payload of game.MatchStarted.
type MatchStarted = service.MatchStarted

// SeasonProvider is implemented by ranking (Phase 7); until then ranked
// matches play in the preseason (D-064).
type SeasonProvider = service.SeasonProvider

// SanctionChecker is implemented by moderation (Phase 8); until then
// nobody is blocked (D-064).
type SanctionChecker = service.SanctionChecker

// ValidateConfig returns every mode-matrix violation of c after the
// transformations (no parent).
func ValidateConfig(c Config) []Violation { return config.Validate(config.Transform(c, nil)) }

// Deps are the module's collaborators.
type Deps struct {
	DB        *pgxpool.Pool
	Redis     *redis.Client
	Content   *content.Module
	Identity  *identity.Module
	Seasons   SeasonProvider
	Sanctions SanctionChecker
	Events    *events.Bus
	Clock     clock.Clock
	Logger    *slog.Logger
}

// Module is the game module.
type Module struct {
	svc *service.Service
}

// New builds the module.
func New(d Deps) *Module {
	return &Module{svc: service.New(service.Deps{
		DB: d.DB, Redis: d.Redis, Questions: d.Content, Users: d.Identity, Seasons: d.Seasons,
		Sanctions: d.Sanctions, Events: d.Events, Clock: d.Clock, Logger: d.Logger,
	})}
}

// Create checks the T1 guards and stores a new match in `waiting`.
func (m *Module) Create(ctx context.Context, r CreateRequest) (*Created, error) {
	return m.svc.Create(ctx, r)
}

// Apply persists one engine output (rounds, transitions, results, events).
func (m *Module) Apply(ctx context.Context, match *engine.Match, out engine.Output) error {
	return m.svc.Apply(ctx, match, out)
}
