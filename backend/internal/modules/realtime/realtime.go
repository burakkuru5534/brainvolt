// Package realtime is the public API of the realtime module
// (backend-modules.md §1, §3.3): the WebSocket gateway of
// docs/api/websocket.md. Other modules register intent handlers by type
// prefix and deliver server events through Send; realtime never imports
// them.
package realtime

import (
	"context"
	"log/slog"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/realtime/internal/service"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/realtime/internal/transport"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/redis"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/wsserver"
)

// Client is the authenticated side of a session.
type Client = service.Client

// Message is a routed client intent.
type Message = service.Message

// Handler processes an intent: nil → sys.ack, error → sys.error.
type Handler = service.Handler

// Target addresses a delivery (a user's sessions, or one session).
type Target = service.Target

// Frame is a server → client message.
type Frame = wsserver.Outbound

// DisconnectFunc runs when a session's socket closes.
type DisconnectFunc = service.DisconnectFunc

// PongFunc receives a session's RTT estimate after every pong.
type PongFunc = service.PongFunc

// LiveFunc contributes to sys.welcome.live.
type LiveFunc = service.LiveFunc

// ResumeFunc replays a stream to a resuming session.
type ResumeFunc = service.ResumeFunc

// Authenticator verifies sys.auth tokens (identity).
type Authenticator = service.Authenticator

// Deps are the module's collaborators; Instance identifies this API
// process for cross-instance delivery.
type Deps struct {
	Redis    *redis.Client
	Auth     Authenticator
	Clock    clock.Clock
	Logger   *slog.Logger
	Instance string
}

// Module is the realtime module. It is also an app component: Run
// receives deliveries addressed to this instance.
type Module struct {
	hub *service.Hub
}

// New builds the module.
func New(d Deps) *Module {
	return &Module{hub: service.New(service.Deps{Redis: d.Redis, Auth: d.Auth, Clock: d.Clock, Logger: d.Logger, Instance: d.Instance})}
}

// Register adds GET /v1/ws.
func (m *Module) Register(s *httpserver.Server) { transport.Register(s, m.hub) }

// Handle routes intents "<prefix>.*" to h.
func (m *Module) Handle(prefix string, h Handler) { m.hub.Handle(prefix, h) }

// OnResume registers the resumer of streams "<prefix>:<id>".
func (m *Module) OnResume(prefix string, fn ResumeFunc) { m.hub.OnResume(prefix, fn) }

// OnDisconnect registers a disconnect listener.
func (m *Module) OnDisconnect(fn DisconnectFunc) { m.hub.OnDisconnect(fn) }

// OnPong registers an RTT listener.
func (m *Module) OnPong(fn PongFunc) { m.hub.OnPong(fn) }

// OnLive registers a contributor to sys.welcome.live.
func (m *Module) OnLive(fn LiveFunc) { m.hub.OnLive(fn) }

// Send delivers a frame to a user's sessions (or one session) on any instance.
func (m *Module) Send(ctx context.Context, t Target, f Frame) error { return m.hub.Send(ctx, t, f) }

// SetPlaying switches a session's heartbeat to the in-match rate (2 s).
func (m *Module) SetPlaying(ctx context.Context, t Target, playing bool) error {
	return m.hub.SetPlaying(ctx, t, playing)
}

// ErrDeferred is returned by a handler that answers later with Reply.
var ErrDeferred = service.ErrDeferred

// Reply answers an intent after ErrDeferred: sys.ack for nil, else sys.error.
func (m *Module) Reply(ctx context.Context, c Client, re string, err error) error {
	return m.hub.Reply(ctx, c, re, err)
}

// Name implements the app component interface.
func (m *Module) Name() string { return "realtime" }

// Run receives cross-instance deliveries until ctx ends, then closes
// every local session with 1001 (clients resume elsewhere).
func (m *Module) Run(ctx context.Context) error { return m.hub.Run(ctx) }
