// Package moderation is the public API of the moderation module
// (backend-modules.md §1, schema-moderation.md): player reports grouped
// into cases, the staff review queue and actions, sanctions and appeals.
// Other modules see it only through interfaces wired in app (game's
// SanctionChecker, matchmaking's sanction check).
package moderation

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/content"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/moderation/internal/service"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/moderation/internal/transport"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/idempotency"
)

// Deps are the module's collaborators.
type Deps struct {
	DB       *pgxpool.Pool
	Identity *identity.Module
	Content  *content.Module
	Events   *events.Bus
	Clock    clock.Clock
	Logger   *slog.Logger
}

// Events moderation publishes (schema-moderation §6).
const (
	EventSanctionApplied = service.EventSanctionApplied
	EventSanctionLifted  = service.EventSanctionLifted
)

// Subscriptions to game events (anti-cheat signals, D-084).
const (
	SubscriberMatchSignals = "moderation.match_signals"
	SubscriberVoidSignals  = "moderation.void_signals"
)

// Module is the moderation module.
type Module struct {
	svc    *service.Service
	logger *slog.Logger
}

// New builds the module.
func New(d Deps) *Module {
	return &Module{
		svc: service.New(service.Deps{
			DB: d.DB, Users: d.Identity, Questions: d.Content, Events: d.Events, Clock: d.Clock,
			ServerVoids: func(ctx context.Context, user uuid.UUID, since time.Time) (int, error) {
				return game.ServerVoids(ctx, d.DB, user, since)
			},
		}),
		logger: d.Logger,
	}
}

// Register adds moderation's routes.
func (m *Module) Register(s *httpserver.Server, ident *identity.Module, idem *idempotency.Store) {
	transport.New(m.svc, ident, idem).Register(s)
}

// Blocking reports an active sanction keeping a user out of a match of
// this type and origin ("" when none): game's SanctionChecker.
func (m *Module) Blocking(ctx context.Context, user uuid.UUID, t game.MatchType, origin game.Origin) (string, error) {
	return m.svc.Blocking(ctx, user, t, origin)
}

// OnMatchCompleted records a completed match's cheat signals.
func (m *Module) OnMatchCompleted(ctx context.Context, e events.Envelope) error {
	return m.svc.OnMatchCompleted(ctx, e)
}

// OnMatchCancelled records repeated server cancellations of a player.
func (m *Module) OnMatchCancelled(ctx context.Context, e events.Envelope) error {
	return m.svc.OnMatchCancelled(ctx, e)
}

// ProtocolViolation records a socket the gateway closed for a malformed
// message or flooding (at most one signal an hour per user).
func (m *Module) ProtocolViolation(ctx context.Context, user uuid.UUID, reason string) {
	err := m.svc.RecordSignal(ctx, service.Signal{User: user, Kind: service.SignalProtocol, Severity: 1,
		Details: map[string]any{"reason": reason}})
	if err != nil && m.logger != nil {
		m.logger.WarnContext(ctx, "moderation: recording a protocol violation failed", "user_id", user, "error", err)
	}
}

// Name implements the app component.
func (m *Module) Name() string { return "moderation" }

// Run restores accounts whose timed suspension ended, every minute, until
// ctx ends (repeating it on several instances is harmless).
func (m *Module) Run(ctx context.Context) error {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			if err := m.svc.ExpireSuspensions(ctx); err != nil && ctx.Err() == nil && m.logger != nil {
				m.logger.WarnContext(ctx, "moderation: suspension expiry failed", "error", err)
			}
		}
	}
}

// ExpireSuspensions restores accounts whose timed suspension ended.
func (m *Module) ExpireSuspensions(ctx context.Context) error { return m.svc.ExpireSuspensions(ctx) }
