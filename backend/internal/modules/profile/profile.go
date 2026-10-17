// Package profile is the public API of the profile module
// (backend-modules.md §1): display name, username, avatar, country,
// locale, settings and the public profile read model.
package profile

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile/internal/service"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile/internal/transport"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ratelimit"
)

// Module is the profile module.
type Module struct {
	svc *service.Service
}

// New builds the module.
func New(db *pgxpool.Pool, c clock.Clock) *Module { return &Module{svc: service.New(db, c)} }

// Provisioner creates and summarises profiles for identity's session flow
// (wired in internal/app, schema-identity §5.1).
func (m *Module) Provisioner() identity.ProfileProvisioner { return m.svc }

// Register adds profile's routes; ident authenticates them.
func (m *Module) Register(s *httpserver.Server, ident *identity.Module, limiter *ratelimit.Limiter) {
	transport.New(m.svc, ident, limiter).Register(s)
}
