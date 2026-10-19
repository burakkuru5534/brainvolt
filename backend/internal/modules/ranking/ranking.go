// Package ranking is the public API of the ranking module
// (backend-modules.md §1, schema-ranking.md): Glicko-2 ratings per bucket,
// 8-week seasons, Season Points and leagues, and the points ledger behind
// the daily / weekly / monthly / all-time and season leaderboards.
package ranking

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/ranking/internal/service"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/team"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/redis"
)

// SubscriberMatchResults is ranking's game.MatchCompleted subscription.
const SubscriberMatchResults = "ranking.match_results"

// Rating is a rating and its deviation.
type Rating = service.Rating

// Season is a ranked season.
type Season = service.Season

// Deps are the module's collaborators.
type Deps struct {
	DB       *pgxpool.Pool
	Redis    *redis.Client
	Profiles *profile.Module
	Teams    *team.Module
	Clock    clock.Clock
	Logger   *slog.Logger
}

// Module is the ranking module.
type Module struct {
	svc *service.Service
}

// New builds the module.
func New(d Deps) *Module {
	sd := service.Deps{DB: d.DB, Redis: d.Redis, Clock: d.Clock, Logger: d.Logger}
	if d.Profiles != nil {
		sd.Cards = d.Profiles
	}
	if d.Teams != nil {
		sd.Teams = d.Teams
	}
	return &Module{svc: service.New(sd)}
}

// ActiveSeason returns the active season, opening the first one when
// none exists (game's SeasonProvider).
func (m *Module) ActiveSeason(ctx context.Context) (uuid.UUID, bool, error) {
	return m.svc.ActiveSeason(ctx)
}

// CurrentSeason returns the active season with its names.
func (m *Module) CurrentSeason(ctx context.Context) (Season, error) { return m.svc.CurrentSeason(ctx) }

// Rating returns a user's rating of a bucket (matchmaking's RatingReader).
func (m *Module) Rating(ctx context.Context, user uuid.UUID, bucket string) (Rating, error) {
	return m.svc.Rating(ctx, user, bucket)
}

// OnMatchCompleted consumes game.MatchCompleted: ratings, Season Points
// and leaderboard points.
func (m *Module) OnMatchCompleted(ctx context.Context, e events.Envelope) error {
	return m.svc.OnMatchCompleted(ctx, e)
}
