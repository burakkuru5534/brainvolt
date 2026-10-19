// Package creator is the public API of the creator module
// (backend-modules.md §1, schema-platform §5): player-made quizzes, their
// review and discovery. Rooms play them through Playable (wired in app).
package creator

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/content"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/creator/internal/service"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/creator/internal/transport"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/idempotency"
)

// Playable is what a room needs to play a quiz.
type Playable = service.Playable

// Deps are the module's collaborators.
type Deps struct {
	DB       *pgxpool.Pool
	Content  *content.Module
	Identity *identity.Module
	Profiles *profile.Module
	Clock    clock.Clock
}

// Module is the creator module.
type Module struct {
	svc *service.Service
}

// New builds the module.
func New(d Deps) *Module {
	return &Module{svc: service.New(service.Deps{DB: d.DB, Questions: d.Content, Users: d.Identity, Cards: d.Profiles, Clock: d.Clock})}
}

// Register adds creator's routes.
func (m *Module) Register(s *httpserver.Server, ident *identity.Module, idem *idempotency.Store) {
	transport.New(m.svc, ident, idem).Register(s)
}

// Playable returns a quiz a user may put in a room: their own, or a
// published one.
func (m *Module) Playable(ctx context.Context, user, id uuid.UUID) (Playable, error) {
	return m.svc.Playable(ctx, user, id)
}

// CountPlay counts a started quiz match.
func (m *Module) CountPlay(ctx context.Context, id uuid.UUID) error { return m.svc.CountPlay(ctx, id) }
