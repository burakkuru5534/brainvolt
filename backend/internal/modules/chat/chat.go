// Package chat is the public API of the chat module (backend-modules.md
// §1, schema-platform §2): channels of persistent teams and couples with
// filtered, rate-limited messages over the WebSocket.
package chat

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/chat/internal/service"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/chat/internal/transport"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/realtime"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/team"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
)

// Muter is implemented by moderation (chat_mute, suspension).
type Muter = service.Muter

// Deps are the module's collaborators; Realtime is nil in the worker.
type Deps struct {
	DB       *pgxpool.Pool
	Teams    *team.Module
	Profiles *profile.Module
	Muter    Muter
	Realtime *realtime.Module
	Events   *events.Bus
	Clock    clock.Clock
}

// EventMessageSent is published with every message (D-089).
const EventMessageSent = service.EventMessageSent

// Module is the chat module.
type Module struct {
	svc *service.Service
	rt  *realtime.Module
}

// New builds the module and registers the chat.* intents.
func New(d Deps) *Module {
	deps := service.Deps{DB: d.DB, Teams: d.Teams, Cards: d.Profiles, Muter: d.Muter, Events: d.Events, Clock: d.Clock}
	if d.Realtime != nil {
		deps.RT = d.Realtime
	}
	return &Module{svc: service.New(deps), rt: d.Realtime}
}

// Register adds the REST routes and the WebSocket intents.
func (m *Module) Register(s *httpserver.Server, ident *identity.Module) {
	t := transport.New(m.svc, ident)
	t.Register(s)
	if m.rt != nil {
		m.rt.Handle("chat", t.Handle)
	}
}

// Purge deletes messages older than six months.
func (m *Module) Purge(ctx context.Context) error { return m.svc.Purge(ctx) }
