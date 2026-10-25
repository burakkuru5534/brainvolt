// Package social is the public API of the social module
// (backend-modules.md §1, schema-social.md §2): friend requests,
// friendships and blocks. Other modules ask AreFriends, FriendIDs,
// IsBlocked and BlockedWith.
package social

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/social/internal/service"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/social/internal/transport"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/idempotency"
)

// Event names (schema-social.md §5).
const (
	EventFriendRequestSent = service.EventFriendRequestSent
	EventFriendshipCreated = service.EventFriendshipCreated
	EventFriendshipRemoved = service.EventFriendshipRemoved
	EventUserBlocked       = service.EventUserBlocked
)

// UserBlocked is the payload of social.UserBlocked.
type UserBlocked = service.UserBlocked

// Presence says whether a user is connected (realtime, wired in app).
type Presence = service.Presence

// Deps are the module's collaborators.
type Deps struct {
	DB       *pgxpool.Pool
	Identity *identity.Module
	Profiles *profile.Module
	Presence Presence
	Events   *events.Bus
	Clock    clock.Clock
}

// Module is the social module.
type Module struct {
	svc      *service.Service
	profiles *profile.Module
}

// New builds the module.
func New(d Deps) *Module {
	return &Module{profiles: d.Profiles, svc: service.New(service.Deps{
		DB: d.DB, Users: d.Identity, Profiles: d.Profiles, Presence: d.Presence, Events: d.Events, Clock: d.Clock,
	})}
}

// Register adds social's routes.
func (m *Module) Register(s *httpserver.Server, ident *identity.Module, idem *idempotency.Store) {
	transport.New(m.svc, ident, idem, func(r *http.Request, name string) (uuid.UUID, error) {
		return m.profiles.UserIDByUsername(r.Context(), name)
	}).Register(s)
}

// AreFriends reports a friendship.
func (m *Module) AreFriends(ctx context.Context, a, b uuid.UUID) (bool, error) {
	return m.svc.AreFriends(ctx, a, b)
}

// FriendIDs lists a user's friends.
func (m *Module) FriendIDs(ctx context.Context, user uuid.UUID) ([]uuid.UUID, error) {
	return m.svc.FriendIDs(ctx, user)
}

// Relation is viewer's tie to other (profile's RelationReader, D-101).
func (m *Module) Relation(ctx context.Context, viewer, other uuid.UUID) (string, error) {
	return m.svc.Relation(ctx, viewer, other)
}

// IsBlocked reports a block between a and b in either direction.
func (m *Module) IsBlocked(ctx context.Context, a, b uuid.UUID) (bool, error) {
	return m.svc.IsBlocked(ctx, a, b)
}

// BlockedWith lists every user blocking or blocked by user.
func (m *Module) BlockedWith(ctx context.Context, user uuid.UUID) ([]uuid.UUID, error) {
	return m.svc.BlockedWith(ctx, user)
}

// SendFriendRequest is a request created by another flow (post-match
// "add friend", friend invite codes).
func (m *Module) SendFriendRequest(ctx context.Context, from, to uuid.UUID, via string) error {
	_, err := m.svc.Send(ctx, from, to, via)
	return err
}
