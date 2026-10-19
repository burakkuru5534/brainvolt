// Package identity is the public API of the identity module
// (backend-modules.md §1): Firebase ID token → BrainVolt account, account
// tiers and statuses, session sync, authentication for every module's
// routes, the link-conflict exit and deletion requests.
//
// Other modules use only what this file exports.
package identity

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity/internal/service"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity/internal/transport"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/idempotency"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ratelimit"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/redis"
)

// Tier is the account tier: guest, unverified or full.
type Tier = domain.Tier

// Status is the account status: active, suspended or deleted.
type Status = domain.Status

// Principal is the authenticated caller of a request.
type Principal = domain.Principal

// User is the identity view of a user.
type User = service.User

// ProfileProvisioner is implemented by the profile module.
type ProfileProvisioner = domain.ProfileProvisioner

// NewProfile is passed to ProfileProvisioner.CreateTx.
type NewProfile = domain.NewProfile

// ProfileSummary is the profile part of the session response.
type ProfileSummary = domain.ProfileSummary

// UserRegistered is the payload of EventUserRegistered.
type UserRegistered = domain.UserRegistered

// AccountTierChanged is the payload of EventAccountTierChanged.
type AccountTierChanged = domain.AccountTierChanged

// GuestAbandoned is the payload of EventGuestAbandoned.
type GuestAbandoned = domain.GuestAbandoned

// AuthOption adjusts Auth for one route.
type AuthOption = transport.AuthOption

// Verifier verifies Firebase ID tokens (platform/firebase.Verifier).
type Verifier = service.Verifier

// Tiers, statuses and event names.
const (
	TierGuest      = domain.TierGuest
	TierUnverified = domain.TierUnverified
	TierFull       = domain.TierFull

	StatusActive    = domain.StatusActive
	StatusSuspended = domain.StatusSuspended
	StatusDeleted   = domain.StatusDeleted

	EventUserRegistered     = domain.EventUserRegistered
	EventAccountTierChanged = domain.EventAccountTierChanged
	EventGuestAbandoned     = domain.EventGuestAbandoned

	RoleModerator = domain.RoleModerator
	RoleAdmin     = domain.RoleAdmin
)

// Auth options for routes of any module.
var (
	AllowSuspended       = transport.AllowSuspended
	AllowDeletionPending = transport.AllowDeletionPending
	RequireFull          = transport.RequireFull
	RequireRole          = transport.RequireRole
)

// Deps are the module's collaborators, built in internal/app.
type Deps struct {
	DB          *pgxpool.Pool
	Redis       *redis.Client
	Verifier    Verifier
	Profiles    ProfileProvisioner
	Events      *events.Bus
	Limiter     *ratelimit.Limiter
	Idempotency *idempotency.Store
	Clock       clock.Clock
	Logger      *slog.Logger
}

// Module is the identity module.
type Module struct {
	svc      *service.Service
	tr       *transport.Transport
	verifier Verifier
}

// New builds the module.
func New(d Deps) *Module {
	svc := service.New(service.Deps{
		DB: d.DB, Redis: d.Redis, Verifier: d.Verifier, Profiles: d.Profiles,
		Events: d.Events, Clock: d.Clock, Logger: d.Logger,
	})
	return &Module{svc: svc, tr: transport.New(svc, d.Verifier, d.Limiter, d.Idempotency, d.Logger), verifier: d.Verifier}
}

// Register adds identity's routes to the server.
func (m *Module) Register(s *httpserver.Server) { m.tr.Register(s) }

// Auth wraps a handler of any module with authentication (see transport.Auth).
func (m *Module) Auth(h httpserver.HandlerFunc, opts ...AuthOption) httpserver.HandlerFunc {
	return m.tr.Auth(h, opts...)
}

// Authenticate verifies a raw Firebase ID token outside an HTTP request
// (the WebSocket handshake, websocket.md §2) and returns the principal and
// the token's expiry. Errors are those of Auth: TOKEN_EXPIRED,
// UNAUTHENTICATED (also for deleted accounts), UNAVAILABLE.
func (m *Module) Authenticate(ctx context.Context, raw string) (Principal, time.Time, error) {
	tok, err := m.verifier.Verify(ctx, raw)
	if err != nil {
		return Principal{}, time.Time{}, err
	}
	p, err := m.svc.Principal(ctx, tok)
	if err != nil {
		return Principal{}, time.Time{}, err
	}
	return p, tok.Expires, nil
}

// PrincipalFrom returns the caller of a request wrapped by Auth.
func PrincipalFrom(ctx context.Context) (Principal, bool) { return transport.PrincipalFrom(ctx) }

// SetRole grants or revokes a staff role (cmd/admin); user is a BrainVolt
// user id or a Firebase UID.
func (m *Module) SetRole(ctx context.Context, user, role string, grant bool) (uuid.UUID, error) {
	return m.svc.SetRole(ctx, user, role, grant)
}

// EnsureBot returns (creating once) a development practice bot user.
func (m *Module) EnsureBot(ctx context.Context, key, displayName, locale string) (uuid.UUID, error) {
	return m.svc.EnsureBot(ctx, key, displayName, locale)
}

// SetStatus suspends or restores an account (moderation).
func (m *Module) SetStatus(ctx context.Context, user uuid.UUID, status Status) error {
	return m.svc.SetStatus(ctx, user, status)
}

// User returns the identity view of a user (NOT_FOUND if missing).
func (m *Module) User(ctx context.Context, id uuid.UUID) (User, error) { return m.svc.User(ctx, id) }
