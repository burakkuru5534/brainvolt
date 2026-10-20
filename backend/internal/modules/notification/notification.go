// Package notification is the public API of the notification module
// (backend-modules.md §1, schema-platform §1): devices and FCM tokens,
// the in-app inbox, per-category preferences with quiet hours, and push
// delivery for other modules' events.
package notification

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/notification/internal/service"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/notification/internal/transport"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
)

// Push is one message to one device.
type Push = service.Push

// Sender delivers pushes; FCM in production (platform/firebase).
type Sender = service.Sender

// Pusher sends a realtime message to a user's open sessions.
type Pusher = service.Pusher

// ErrTokenInvalid is what a Sender returns for a token FCM rejects.
var ErrTokenInvalid = service.ErrTokenInvalid

// Notice is something to tell a user.
type Notice = service.Notice

// Subscriptions (backend-modules §3.2).
const (
	SubscriberFriendRequest    = "notification.friend_request"
	SubscriberFriendship       = "notification.friendship"
	SubscriberChallengeSent    = "notification.challenge_sent"
	SubscriberChallengeAccepts = "notification.challenge_accepted"
	SubscriberCoupleRequest    = "notification.couple_request"
	SubscriberSanction         = "notification.sanction"
)

// Deps are the module's collaborators; Sender, Pusher and Profiles are
// needed only where notices are delivered (the worker).
type Deps struct {
	DB       *pgxpool.Pool
	Profiles *profile.Module
	Sender   Sender
	Pusher   Pusher
	Clock    clock.Clock
	Logger   *slog.Logger
}

// Module is the notification module.
type Module struct {
	d   Deps
	svc *service.Service
}

// New builds the module.
func New(d Deps) *Module {
	return &Module{d: d, svc: service.New(service.Deps{DB: d.DB, Sender: d.Sender, Pusher: d.Pusher, Clock: d.Clock, Logger: d.Logger})}
}

// Register adds the device, inbox and preference routes.
func (m *Module) Register(s *httpserver.Server, ident *identity.Module) {
	transport.New(m.d.DB, m.d.Clock, ident).Register(s)
	transport.NewInbox(m.svc, ident).Register(s)
}

// Notify delivers a notice (inbox, open app, push).
func (m *Module) Notify(ctx context.Context, n Notice) error { return m.svc.Notify(ctx, n) }

// Purge deletes expired inbox items and old delivery records.
func (m *Module) Purge(ctx context.Context) error { return m.svc.Purge(ctx) }

// Event handlers, subscribed in app (backend-modules §3.2): notification
// does not import the producers.
func (m *Module) handlers() service.Handlers {
	return service.Handlers{Svc: m.svc, Cards: m.d.Profiles}
}

// OnFriendRequestSent handles social.FriendRequestSent.
func (m *Module) OnFriendRequestSent(ctx context.Context, e events.Envelope) error {
	return m.handlers().OnFriendRequestSent(ctx, e)
}

// OnFriendshipCreated handles social.FriendshipCreated.
func (m *Module) OnFriendshipCreated(ctx context.Context, e events.Envelope) error {
	return m.handlers().OnFriendshipCreated(ctx, e)
}

// OnChallengeSent handles lobby.ChallengeSent.
func (m *Module) OnChallengeSent(ctx context.Context, e events.Envelope) error {
	return m.handlers().OnChallengeSent(ctx, e)
}

// OnChallengeAccepted handles lobby.ChallengeAccepted.
func (m *Module) OnChallengeAccepted(ctx context.Context, e events.Envelope) error {
	return m.handlers().OnChallengeAccepted(ctx, e)
}

// OnCoupleRequested handles team.CoupleRequested.
func (m *Module) OnCoupleRequested(ctx context.Context, e events.Envelope) error {
	return m.handlers().OnCoupleRequested(ctx, e)
}

// OnSanctionApplied handles moderation.SanctionApplied.
func (m *Module) OnSanctionApplied(ctx context.Context, e events.Envelope) error {
	return m.handlers().OnSanctionApplied(ctx, e)
}
