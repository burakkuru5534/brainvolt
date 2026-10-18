// Package matchmaking is the public API of the matchmaking module
// (backend-modules.md §1). Phase 4 runs one casual Quick Duel 1v1 queue
// paired first come, first served, plus the development practice bot
// (D-066); Phase 5 adds rating ranges, parties and the other modes.
package matchmaking

import (
	"context"
	"log/slog"

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

// Deps are the module's collaborators. Practice enables the development
// bot opponent; only the dev composition sets it.
type Deps struct {
	Redis    *redis.Client
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

// New builds the module.
func New(d Deps) *Module {
	return &Module{svc: service.New(service.Deps{
		Redis: d.Redis, Game: d.Game, Notifier: d.Realtime, Clock: d.Clock, Logger: d.Logger, Practice: d.Practice,
	})}
}

// Register adds the ticket routes.
func (m *Module) Register(s *httpserver.Server, ident *identity.Module, idem *idempotency.Store) {
	transport.Register(s, m.svc, ident, idem)
}

// Name implements the app component interface.
func (m *Module) Name() string { return "matchmaking" }

// Run pairs queued players until ctx ends.
func (m *Module) Run(ctx context.Context) error { return m.svc.Run(ctx) }
