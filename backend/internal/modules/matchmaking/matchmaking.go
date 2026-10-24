// Package matchmaking is the public API of the matchmaking module
// (backend-modules.md §1): a queue per matchmade 1v1 configuration paired
// by rating with widening ranges (D-068), re-queueing and queue cooldowns,
// team formats with parties, persistent teams and couples (D-072), plus
// the development practice bot (D-066).
package matchmaking

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/matchmaking/internal/service"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/matchmaking/internal/transport"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/realtime"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/team"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/idempotency"
)

// RatingReader returns players' ratings (ranking implements it in Phase 7;
// nil uses the defaults for new players).
type RatingReader = service.RatingReader

// Rating is a rating and its deviation.
type Rating = service.Rating

// Social reads blocks and friendships (social, wired in app).
type Social = service.Social

// Deps are the module's collaborators. Practice enables the development
// bot opponent; only the dev composition sets it.
type Deps struct {
	DB      *pgxpool.Pool
	Ratings RatingReader
	Social  Social
	// Sanctions keeps sanctioned players out of the queues (moderation).
	Sanctions game.SanctionChecker
	Teams     *team.Module
	Profiles  *profile.Module
	Game      *game.Module
	Realtime  *realtime.Module
	Clock     clock.Clock
	Logger    *slog.Logger
	Practice  bool
}

// Module is the matchmaking module; as an app component it runs the
// matcher loop.
type Module struct {
	svc *service.Service
}

// New builds the module and follows game's finished matches (re-queue,
// cooldowns).
func New(d Deps) *Module {
	svc := service.New(service.Deps{
		DB: d.DB, Game: games{d.Game}, Notifier: d.Realtime, Ratings: d.Ratings, Social: d.Social, Sanctions: d.Sanctions,
		Clock: d.Clock, Logger: d.Logger, Practice: d.Practice,
	})
	if d.Teams != nil {
		svc.SetTeams(d.Teams)
	}
	if d.Profiles != nil {
		svc.SetLocales(d.Profiles)
	}
	d.Game.OnEnd(svc.OnEnd)
	if d.Realtime != nil {
		// A searching player counts as busy: the idle close (D-097) leaves
		// their socket alone, and sys.welcome.live names the ticket.
		d.Realtime.OnLive(func(ctx context.Context, user uuid.UUID) (map[string]any, error) {
			t, err := svc.Current(ctx, user)
			if err != nil {
				if apierror.From(err) != nil && apierror.From(err).Code == apierror.NotFound {
					return nil, nil
				}
				return nil, err
			}
			return map[string]any{"ticket_id": t.ID}, nil
		})
	}
	return &Module{svc: svc}
}

// LobbyGroup is a team ticket to seat in a waiting lobby (D-094).
type LobbyGroup = service.LobbyGroup

// Lobbies seats team tickets in waiting lobbies (D-094). lobby implements
// it; app adapts it, since matchmaking may not import lobby
// (backend-modules.md §2).
type Lobbies = service.Lobbies

// SetLobbies takes team formats to waiting lobbies (D-094).
func (m *Module) SetLobbies(l Lobbies) { m.svc.SetLobbies(l) }

// games adapts game to what the queue needs.
type games struct{ *game.Module }

func (g games) Start(ctx context.Context, r game.CreateRequest) (service.Started, error) {
	c, err := g.Module.Start(ctx, r)
	if err != nil {
		return service.Started{}, err
	}
	return service.Started{MatchID: c.Match.ID, ReadyDeadlineMS: c.Match.ReadyDeadMS}, nil
}

func (g games) Practice(ctx context.Context, user uuid.UUID, locale string) (service.Started, error) {
	c, err := g.Module.Practice(ctx, user, locale)
	if err != nil {
		return service.Started{}, err
	}
	return service.Started{MatchID: c.Match.ID, ReadyDeadlineMS: c.Match.ReadyDeadMS}, nil
}

// Register adds the ticket routes.
func (m *Module) Register(s *httpserver.Server, ident *identity.Module, idem *idempotency.Store) {
	transport.Register(s, m.svc, ident, idem)
}

// Name implements the app component interface.
func (m *Module) Name() string { return "matchmaking" }

// Run pairs queued players until ctx ends.
func (m *Module) Run(ctx context.Context) error { return m.svc.Run(ctx) }
