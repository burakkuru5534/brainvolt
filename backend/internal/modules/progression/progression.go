// Package progression is the public API of the progression module
// (backend-modules.md §1, schema-ranking.md §5): XP and levels, play
// streaks, badges and daily / weekly quests, computed from completed
// matches in the worker.
package progression

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/progression/internal/service"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/progression/internal/transport"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/ranking"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/idempotency"
)

// Subscriptions: game.MatchCompleted and ranking.SeasonEnded.
const (
	SubscriberMatchAwards  = "progression.match_awards"
	SubscriberSeasonReward = "progression.season_rewards"
	SubscriberDailyRewards = "progression.daily_rewards"
	// SubscriberTournamentRewards gives tournament placement XP (D-090).
	SubscriberTournamentRewards = "progression.tournament_rewards"
	// SubscriberAdRewards gives verified rewarded ads their effect (D-103).
	SubscriberAdRewards = "progression.ad_rewards"
)

// MsgAwards is the personal match.awards message.
const MsgAwards = service.MsgAwards

// Pusher sends a personal WebSocket message (realtime, wired in app).
type Pusher = service.Pusher

// Deps are the module's collaborators.
type Deps struct {
	DB      *pgxpool.Pool
	Pusher  Pusher
	Ranking *ranking.Module
	// Plus reports BrainVolt Plus (commerce's entitlement reader, wired in
	// internal/app; D-104). nil = nobody has Plus.
	Plus PlusChecker
	// Wallet credits Volt rewards (commerce.VoltWriter, D-105); nil = none.
	Wallet Wallet
	Clock  clock.Clock
	Logger *slog.Logger
}

// PlusChecker reports entitlements.
type PlusChecker = service.PlusChecker

// Wallet writes Volt inside progression's transactions.
type Wallet = service.Wallet

// Module is the progression module.
type Module struct {
	svc     *service.Service
	ranking *ranking.Module
}

// New builds the module.
func New(d Deps) *Module {
	return &Module{
		svc:     service.New(service.Deps{DB: d.DB, Pusher: d.Pusher, Plus: d.Plus, Wallet: d.Wallet, Clock: d.Clock, Logger: d.Logger}),
		ranking: d.Ranking,
	}
}

// Register adds progression's routes.
func (m *Module) Register(s *httpserver.Server, ident *identity.Module, idem *idempotency.Store) {
	transport.New(m.svc, ident, idem).Register(s)
}

// OnMatchCompleted consumes game.MatchCompleted: XP, streaks, badges,
// quests and match.awards.
func (m *Module) OnMatchCompleted(ctx context.Context, e events.Envelope) error {
	return m.svc.OnMatchCompleted(ctx, e)
}

// OnSeasonEnded consumes ranking.SeasonEnded: season badges.
func (m *Module) OnSeasonEnded(ctx context.Context, e events.Envelope) error {
	return m.svc.OnSeasonEnded(ctx, e, seasons{m.ranking})
}

// OnTournamentFinished handles tournament.TournamentFinished.
func (m *Module) OnTournamentFinished(ctx context.Context, e events.Envelope) error {
	return m.svc.OnTournamentFinished(ctx, e)
}

// seasons adapts ranking's season results.
type seasons struct{ *ranking.Module }

func (s seasons) SeasonResults(ctx context.Context, season uuid.UUID) ([]service.SeasonResult, error) {
	rs, err := s.Module.SeasonResults(ctx, season)
	out := make([]service.SeasonResult, len(rs))
	for i, r := range rs {
		out[i] = service.SeasonResult{UserID: r.UserID, PeakLeague: r.PeakLeague, FinalRank: r.FinalRank}
	}
	return out, err
}

// OnDailyRunCompleted consumes game.DailyRunCompleted: XP, play streak
// and daily_challenge quests.
func (m *Module) OnDailyRunCompleted(ctx context.Context, e events.Envelope) error {
	return m.svc.OnDailyRunCompleted(ctx, e)
}

// PlayerStats is a player's public progress (profile's StatsReader, D-101).
func (m *Module) PlayerStats(ctx context.Context, user uuid.UUID) (profile.PlayerStats, error) {
	p, err := m.svc.Progress(ctx, user)
	if err != nil {
		return profile.PlayerStats{}, err
	}
	return profile.PlayerStats{Level: p.Level, Matches: p.Matches, Wins: p.Wins, Badges: p.Badges, Streak: p.Streak.Current}, nil
}

// AdOffer is what a rewarded ad placement would give user now (D-103).
type AdOffer = service.AdOffer

// AdOffer checks a rewarded-ad placement (commerce's RewardChecker,
// adapted in internal/app).
func (m *Module) AdOffer(ctx context.Context, user uuid.UUID, placement, ref string) (AdOffer, error) {
	return m.svc.AdOffer(ctx, user, placement, ref)
}

// OnAdRewardGranted consumes commerce.AdRewardGranted.
func (m *Module) OnAdRewardGranted(ctx context.Context, e events.Envelope) error {
	return m.svc.OnAdRewardGranted(ctx, e)
}

// ApplyItemTx gives a store item bought with Volt its effect inside the
// purchase's transaction (commerce's ItemApplier, D-105).
func (m *Module) ApplyItemTx(ctx context.Context, tx pgx.Tx, user uuid.UUID, item, ref string, now time.Time) error {
	return m.svc.ApplyItemTx(ctx, tx, user, item, ref, now)
}
