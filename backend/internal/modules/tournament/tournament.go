// Package tournament is the public API of the tournament module
// (backend-modules.md §1, schema-platform §3): admin-made single-elimination
// tournaments of users, couples or teams, with registration, check-in,
// brackets, matches through game, walkovers and XP rewards (D-090).
package tournament

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/team"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/tournament/internal/service"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/tournament/internal/transport"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/idempotency"
)

// Events (backend-modules §3.2).
const (
	EventRoundScheduled = service.EventRoundScheduled
	EventFinished       = service.EventFinished
)

// Subscriptions to game's events.
const (
	SubscriberMatchResults   = "tournament.match_results"
	SubscriberMatchCancelled = "tournament.match_cancelled"
)

// Pusher sends a realtime message to a user's open sessions.
type Pusher = service.Pusher

// Deps are the module's collaborators; Game is nil in the worker.
type Deps struct {
	DB       *pgxpool.Pool
	Game     *game.Module
	Profiles *profile.Module
	Teams    *team.Module
	Pusher   Pusher
	Events   *events.Bus
	// Friend tournaments (D-109): friendships (social), the fee (commerce's
	// VoltWriter) and Plus; Wallet is also needed in the worker (refunds).
	Friends Friends
	Wallet  Wallet
	Plus    PlusChecker
	Clock   clock.Clock
	Logger  *slog.Logger
}

// Friends answers friendship questions (social).
type Friends = service.Friends

// Wallet spends and refunds Volt (commerce.VoltWriter).
type Wallet = service.Wallet

// PlusChecker reports entitlements (commerce).
type PlusChecker = service.PlusChecker

// KindFriends marks friend tournaments (D-109).
const KindFriends = service.KindFriends

// Module is the tournament module.
type Module struct {
	svc *service.Service
}

// New builds the module.
func New(d Deps) *Module {
	deps := service.Deps{DB: d.DB, Cards: d.Profiles, Teams: d.Teams, Pusher: d.Pusher, Events: d.Events,
		Friends: d.Friends, Wallet: d.Wallet, Plus: d.Plus, Clock: d.Clock, Logger: d.Logger}
	if d.Game != nil {
		deps.Games = d.Game
	}
	return &Module{svc: service.New(deps)}
}

// Register adds tournament's routes.
func (m *Module) Register(s *httpserver.Server, ident *identity.Module, idem *idempotency.Store) {
	transport.New(m.svc, ident, idem).Register(s)
}

// OnMatchCompleted handles game.MatchCompleted.
func (m *Module) OnMatchCompleted(ctx context.Context, e events.Envelope) error {
	return m.svc.OnMatchCompleted(ctx, e)
}

// OnMatchCancelled handles game.MatchCancelled.
func (m *Module) OnMatchCancelled(ctx context.Context, e events.Envelope) error {
	return m.svc.OnMatchCancelled(ctx, e)
}

// Tick starts due tournaments and decides closed ready windows (worker).
func (m *Module) Tick(ctx context.Context) error { return m.svc.Tick(ctx) }

// FriendTrophies counts the friend tournaments user won (player profiles).
func (m *Module) FriendTrophies(ctx context.Context, user uuid.UUID) (int, error) {
	return m.svc.FriendTrophies(ctx, user)
}
