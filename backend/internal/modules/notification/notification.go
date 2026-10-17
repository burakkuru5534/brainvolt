// Package notification is the public API of the notification module
// (backend-modules.md §1). Phase 1 stores devices and FCM tokens; push
// delivery, preferences and the inbox arrive in Phase 9.
package notification

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/notification/internal/transport"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
)

// Module is the notification module.
type Module struct {
	db    *pgxpool.Pool
	clock clock.Clock
}

// New builds the module.
func New(db *pgxpool.Pool, c clock.Clock) *Module { return &Module{db: db, clock: c} }

// Register adds the device routes; ident authenticates them.
func (m *Module) Register(s *httpserver.Server, ident *identity.Module) {
	transport.New(m.db, m.clock, ident).Register(s)
}
