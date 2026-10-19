// Package team is the public API of the team module (backend-modules.md
// §1, schema-social.md §4): persistent teams (2–4 members, a captain,
// invites among friends) and couple profiles (D-018, D-047). Team and
// couple matches use them from Phase 6.
package team

import (
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
