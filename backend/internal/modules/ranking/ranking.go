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

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/ranking/internal/service"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/ranking/internal/transport"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/social"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/team"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
)

// Subscriptions: game.MatchCompleted and identity.AccountTierChanged.
const (
	SubscriberMatchResults = "ranking.match_results"
	SubscriberTierChanged  = "ranking.tier_changed"
	SubscriberDailyBoard   = "ranking.daily_board"
)

// EventSeasonEnded is published when a season closes.
const EventSeasonEnded = service.EventSeasonEnded

// SeasonEnded is the payload of EventSeasonEnded.
type SeasonEnded = service.SeasonEnded

// SeasonResult is a user's outcome of a closed season.
type SeasonResult = service.SeasonResult

// Pusher sends a personal WebSocket message (realtime, wired in app).
type Pusher = service.Pusher

// Rating is a rating and its deviation.
type Rating = service.Rating

// Season is a ranked season.
type Season = service.Season

// Deps are the module's collaborators.
type Deps struct {
	DB       *pgxpool.Pool
	Profiles *profile.Module
	Teams    *team.Module
	Social   *social.Module
	Events   *events.Bus
	Pusher   Pusher
	Clock    clock.Clock
	Logger   *slog.Logger
}

// Module is the ranking module.
type Module struct {
	svc *service.Service
}

// New builds the module.
func New(d Deps) *Module {
	sd := service.Deps{DB: d.DB, Events: d.Events, Pusher: d.Pusher, Clock: d.Clock, Logger: d.Logger}
	if d.Profiles != nil {
		sd.Cards = d.Profiles
	}
	if d.Teams != nil {
		sd.Teams = d.Teams
	}
	if d.Social != nil {
		sd.Friends = d.Social
	}
	return &Module{svc: service.New(sd)}
}

// Register adds ranking's routes.
func (m *Module) Register(s *httpserver.Server, ident *identity.Module) {
	transport.New(m.svc, ident).Register(s)
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

// OnTierChanged consumes identity.AccountTierChanged: an upgraded
// account's hidden points join the running boards (D-016, D-017).
func (m *Module) OnTierChanged(ctx context.Context, e events.Envelope) error {
	return m.svc.OnTierChanged(ctx, e)
}

// Maintain archives boards of ended periods and closes an ended season
// (the worker runs it periodically).
func (m *Module) Maintain(ctx context.Context) error { return m.svc.Maintain(ctx) }

// SeasonResults lists users' peak leagues and final ranks of a season.
func (m *Module) SeasonResults(ctx context.Context, season uuid.UUID) ([]SeasonResult, error) {
	return m.svc.SeasonResults(ctx, season)
}

// OnDailyRunCompleted consumes game.DailyRunCompleted: the daily board.
func (m *Module) OnDailyRunCompleted(ctx context.Context, e events.Envelope) error {
	return m.svc.OnDailyRunCompleted(ctx, e)
}
