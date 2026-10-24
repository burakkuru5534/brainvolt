// Package lobby is the public API of the lobby module
// (backend-modules.md §1, schema-social.md §3): invite codes, direct
// challenges, private rooms and rematch — everything that gathers players
// before game creates the match.
package lobby

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/lobby/internal/service"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/lobby/internal/transport"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/realtime"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/social"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/team"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/idempotency"
)

// Event names (schema-social.md §5).
const (
	EventChallengeSent     = service.EventChallengeSent
	EventChallengeAccepted = service.EventChallengeAccepted
	EventRoomClosed        = service.EventRoomClosed
)

// SubscriberChallengeFailed is lobby's subscription to game.MatchCancelled.
const SubscriberChallengeFailed = "lobby.on_match_cancelled"

// Quiz is a Creator Quiz as a room or challenge plays it.
type Quiz = service.Quiz

// Quizzes is implemented over creator in app (lobby does not import it).
type Quizzes = service.Quizzes

// Deps are the module's collaborators.
type Deps struct {
	DB       *pgxpool.Pool
	Identity *identity.Module
	Profiles *profile.Module
	Social   *social.Module
	Teams    *team.Module
	Game     *game.Module
	Quizzes  Quizzes
	Realtime *realtime.Module
	Events   *events.Bus
	Clock    clock.Clock
	Logger   *slog.Logger
}

// Module is the lobby module; as an app component it closes idle rooms.
type Module struct {
	svc *service.Service
	t   *transport.Transport
}

// New builds the module and registers the room.* and rematch.* intents
// and the room stream's resume.
func New(d Deps) *Module {
	svc := service.New(service.Deps{
		DB: d.DB, Users: d.Identity, Profiles: d.Profiles, Social: d.Social, Teams: d.Teams, Game: d.Game, Quizzes: d.Quizzes,
		RT: d.Realtime, Events: d.Events, Clock: d.Clock, Logger: d.Logger,
	})
	m := &Module{svc: svc}
	if d.Realtime != nil {
		m.t = transport.New(svc, d.Identity, nil)
		d.Realtime.Handle("room", m.t.HandleRoom)
		d.Realtime.Handle("rematch", m.t.HandleRematch)
		d.Realtime.OnResume("room", func(ctx context.Context, c realtime.Client, stream string, _ int64) error {
			id, err := service.RoomOfStream(stream)
			if err != nil {
				return err
			}
			return svc.SnapshotRoom(ctx, c, id)
		})
		d.Realtime.OnLive(func(ctx context.Context, user uuid.UUID) (map[string]any, error) {
			if id, ok := svc.CurrentRoom(ctx, user); ok {
				return map[string]any{"room_id": id}, nil
			}
			return nil, nil
		})
	}
	return m
}

// Register adds lobby's routes.
func (m *Module) Register(s *httpserver.Server, ident *identity.Module, idem *idempotency.Store) {
	transport.New(m.svc, ident, idem).Register(s)
}

// Name implements the app component interface.
func (m *Module) Name() string { return "lobby" }

// Run closes idle rooms and deletes old invite codes every minute until
// ctx ends. Every API instance sweeps; closing is idempotent.
func (m *Module) Run(ctx context.Context) error { return m.svc.Run(ctx) }

// OnMatchCancelled marks the challenge of a match cancelled before it
// started as failed (schema-social §3.1).
func OnMatchCancelled(db *pgxpool.Pool) events.Handler {
	svc := service.New(service.Deps{DB: db})
	return func(ctx context.Context, e events.Envelope) error {
		var p game.MatchCancelled
		if err := json.Unmarshal(e.Payload, &p); err != nil {
			return fmt.Errorf("lobby: decoding %s: %w", e.Name, err)
		}
		if p.Origin != game.OriginChallenge {
			return nil
		}
		return svc.FailChallengeOfMatch(ctx, p.MatchID)
	}
}

// Sweep closes idle rooms once (tests; Run sweeps every minute).
func (m *Module) Sweep(ctx context.Context) { m.svc.SweepRooms(ctx) }
