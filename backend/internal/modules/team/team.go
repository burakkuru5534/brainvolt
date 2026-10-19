// Package team is the public API of the team module (backend-modules.md
// §1, schema-social.md §4): persistent teams (2–4 members, a captain,
// invites among friends) and couple profiles (D-018, D-047). Team and
// couple matches use them from Phase 6.
package team

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/social"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/team/internal/service"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/team/internal/transport"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/idempotency"
)

// Event names (schema-social.md §5).
const (
	EventTeamCreated     = service.EventTeamCreated
	EventTeamDisbanded   = service.EventTeamDisbanded
	EventMemberJoined    = service.EventMemberJoined
	EventMemberLeft      = service.EventMemberLeft
	EventCaptainChanged  = service.EventCaptainChanged
	EventCoupleRequested = service.EventCoupleRequested
	EventCoupleFormed    = service.EventCoupleFormed
	EventCoupleDissolved = service.EventCoupleDissolved
)

// Deps are the module's collaborators.
type Deps struct {
	DB       *pgxpool.Pool
	Identity *identity.Module
	Profiles *profile.Module
	Social   *social.Module
	Events   *events.Bus
	Clock    clock.Clock
}

// Module is the team module.
type Module struct {
	svc *service.Service
}

// New builds the module.
func New(d Deps) *Module {
	return &Module{svc: service.New(service.Deps{
		DB: d.DB, Users: d.Identity, Profiles: d.Profiles, Social: d.Social, Events: d.Events, Clock: d.Clock,
	})}
}

// Register adds team's routes.
func (m *Module) Register(s *httpserver.Server, ident *identity.Module, idem *idempotency.Store) {
	transport.New(m.svc, ident, idem).Register(s)
}

// Info is a team's state and current members.
type Info = service.TeamInfo

// CoupleInfo is a couple's state and partners.
type CoupleInfo = service.CoupleInfo

// TeamInfo returns a team's state and current members (game lineups).
func (m *Module) TeamInfo(ctx context.Context, id uuid.UUID) (Info, error) {
	return m.svc.TeamInfo(ctx, id)
}

// CoupleInfo returns a couple's state and partners.
func (m *Module) CoupleInfo(ctx context.Context, id uuid.UUID) (CoupleInfo, error) {
	return m.svc.CoupleInfo(ctx, id)
}

// ActiveCoupleOf returns a user's active couple.
func (m *Module) ActiveCoupleOf(ctx context.Context, user uuid.UUID) (CoupleInfo, bool, error) {
	return m.svc.ActiveCoupleOf(ctx, user)
}

// Label is how a leaderboard shows a team or couple.
type Label = service.Label

// TeamLabels returns the names, tags and emblems of teams.
func (m *Module) TeamLabels(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]Label, error) {
	return m.svc.TeamLabels(ctx, ids)
}

// CoupleLabels returns the names and partners of couples.
func (m *Module) CoupleLabels(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]Label, error) {
	return m.svc.CoupleLabels(ctx, ids)
}
