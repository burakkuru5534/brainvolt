// Package matchmaking is the public API of the matchmaking module
// (backend-modules.md §1): a queue per matchmade 1v1 configuration paired
// by rating with widening ranges (D-068), re-queueing and queue cooldowns,
// plus the development practice bot (D-066). Parties and team formats
// arrive with Phase 6.
package matchmaking

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/matchmaking/internal/service"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/matchmaking/internal/transport"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/realtime"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/idempotency"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/redis"
)

// RatingReader returns players' ratings (ranking implements it in Phase 7;
// nil uses the defaults for new players).
type RatingReader = service.RatingReader

// Rating is a rating and its deviation.
type Rating = service.Rating

// BlockReader lists the users blocking or blocked by a user (social).
type BlockReader = service.BlockReader

// Deps are the module's collaborators. Practice enables the development
// bot opponent; only the dev composition sets it.
type Deps struct {
	Redis    *redis.Client
	Ratings  RatingReader
	Blocks   BlockReader
	Game     *game.Module
	Realtime *realtime.Module
	Clock    clock.Clock
	Logger   *slog.Logger
	Practice bool
}

// Module is the matchmaking module; as an app component it runs the
// matcher loop.
type Module struct {
	svc *service.Service
}

// New builds the module and follows game's finished matches (re-queue,
// cooldowns).
func New(d Deps) *Module {
	svc := service.New(service.Deps{
		Redis: d.Redis, Game: games{d.Game}, Notifier: d.Realtime, Ratings: d.Ratings, Blocks: d.Blocks,
		Clock: d.Clock, Logger: d.Logger, Practice: d.Practice,
	})
	d.Game.OnEnd(svc.OnEnd)
	return &Module{svc: svc}
}

// games adapts game to what the queue needs.
type games struct{ *game.Module }

func (g games) Start(ctx context.Context, r game.CreateRequest) (service.Started, error) {
	c, err := g.Module.Start(ctx, r)
	if err != nil {
		return service.Started{}, err
	}
	return service.Started{MatchID: c.Match.ID, ReadyDeadlineMS: c.Match.ReadyDeadMS}, nil
}

func (g games) Practice(ctx context.Context, user uuid.UUID, locale string) (service.Started, error) {
	c, err := g.Module.Practice(ctx, user, locale)
	if err != nil {
		return service.Started{}, err
	}
	return service.Started{MatchID: c.Match.ID, ReadyDeadlineMS: c.Match.ReadyDeadMS}, nil
}

// Register adds the ticket routes.
func (m *Module) Register(s *httpserver.Server, ident *identity.Module, idem *idempotency.Store) {
	transport.Register(s, m.svc, ident, idem)
}

// Name implements the app component interface.
func (m *Module) Name() string { return "matchmaking" }

// Run pairs queued players until ctx ends.
func (m *Module) Run(ctx context.Context) error { return m.svc.Run(ctx) }
