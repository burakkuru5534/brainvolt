// Package service is the Phase 4 matchmaking queue: casual Quick Duel 1v1
// paired first come, first served (redis-keys §3). Phase 5 replaces the
// pairing with rating ranges, parties and the other modes; the endpoints
// and the ticket stay.
package service

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/realtime"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/redis"
)

// Queue settings (redis-keys §3, D-066).
const (
	QuickDuelQueue = "quick_duel:1v1:casual:any"
	TicketTTL      = 10 * time.Minute
	MatchEvery     = 500 * time.Millisecond
	matcherLease   = 5 * time.Second
)

// Ticket states.
const (
	StateSearching = "searching"
	StateFound     = "found"
)

// Game is what matchmaking needs from game.
type Game interface {
	Start(ctx context.Context, r game.CreateRequest) (*game.Created, error)
	Practice(ctx context.Context, user uuid.UUID, locale string) (*game.Created, error)
	LiveMatch(ctx context.Context, user uuid.UUID) (uuid.UUID, bool)
}

// Notifier pushes personal events (realtime).
type Notifier interface {
	Send(ctx context.Context, t realtime.Target, f realtime.Frame) error
}

// Deps are the service's collaborators. Practice enables the development
// bot opponent (D-066).
type Deps struct {
	Redis    *redis.Client
	Game     Game
	Notifier Notifier
	Clock    clock.Clock
	Logger   *slog.Logger
	Practice bool
}

// Service is the queue.
type Service struct{ d Deps }

// New returns the service.
func New(d Deps) *Service { return &Service{d: d} }

// Request is the body of POST /v1/matchmaking/tickets.
type Request struct {
	MatchType string `json:"match_type"`
	Format    string `json:"format"`
	// Practice plays against the server's bot (development builds only).
	Practice bool `json:"practice,omitempty"`
}

// Ticket is a queue entry as the app sees it.
type Ticket struct {
	ID         uuid.UUID  `json:"id"`
	MatchType  string     `json:"match_type"`
	Format     string     `json:"format"`
	State      string     `json:"state"`
	EnqueuedAt time.Time  `json:"enqueued_at"`
	WaitMS     int64      `json:"wait_ms"`
	MatchID    *uuid.UUID `json:"match_id,omitempty"`
}

// Phase 4 has one queue; Phase 5 keys queues by mode, format and bucket.
var (
	queueKey = redis.Key("mm", "q", QuickDuelQueue)
	lockKey  = redis.Key("mm", "lock", QuickDuelQueue)
)

func ticketKey(id uuid.UUID) string { return redis.Key("mm", "t", id.String()) }
func userKey(user uuid.UUID) string { return redis.Key("mm", "user", user.String()) }

// Enqueue puts the caller in the Quick Duel queue, or starts a practice
// match right away.
func (s *Service) Enqueue(ctx context.Context, user uuid.UUID, locale string, r Request) (Ticket, error) {
	if r.MatchType != "quick_duel" || r.Format != "1v1" {
		return Ticket{}, apierror.New(apierror.ModeNotAllowed).With("reason", "not_available_yet")
	}
	if r.Practice && !s.d.Practice {
		return Ticket{}, apierror.Validation(apierror.FieldError{Field: "practice", Rule: "not_available"})
	}
	if _, live := s.d.Game.LiveMatch(ctx, user); live {
		return Ticket{}, apierror.New(apierror.AlreadyInMatch)
	}
	now := s.d.Clock.Now().UTC()
	t := Ticket{ID: ids.New(), MatchType: r.MatchType, Format: r.Format, State: StateSearching, EnqueuedAt: now}
	if r.Practice {
		if exists, _ := s.d.Redis.Exists(ctx, userKey(user)).Result(); exists > 0 {
			return Ticket{}, apierror.New(apierror.AlreadyQueued)
		}
		c, err := s.d.Game.Practice(ctx, user, locale)
		if err != nil {
			return Ticket{}, err
		}
		t.State, t.MatchID = StateFound, &c.Match.ID
		return t, nil
	}
	ok, err := s.d.Redis.SetNX(ctx, userKey(user), t.ID.String(), TicketTTL).Result()
	if err != nil {
		return Ticket{}, err
	}
	if !ok {
		return Ticket{}, apierror.New(apierror.AlreadyQueued)
	}
	pipe := s.d.Redis.TxPipeline()
	pipe.HSet(ctx, ticketKey(t.ID), "user_id", user.String(), "locale", locale, "queue", QuickDuelQueue,
		"enqueued_ms", now.UnixMilli(), "match_type", r.MatchType, "format", r.Format)
	pipe.Expire(ctx, ticketKey(t.ID), TicketTTL)
	pipe.ZAdd(ctx, queueKey, goredis.Z{Score: float64(now.UnixMilli()), Member: t.ID.String()})
	if _, err := pipe.Exec(ctx); err != nil {
		s.d.Redis.Del(ctx, userKey(user))
		return Ticket{}, err
	}
	return t, nil
}

// Current returns the caller's ticket (NOT_FOUND when not queued).
func (s *Service) Current(ctx context.Context, user uuid.UUID) (Ticket, error) {
	id, err := s.ticketOf(ctx, user)
	if err != nil {
		return Ticket{}, err
	}
	v, err := s.d.Redis.HGetAll(ctx, ticketKey(id)).Result()
	if err != nil {
		return Ticket{}, err
	}
	if len(v) == 0 {
		return Ticket{}, apierror.New(apierror.NotFound)
	}
	ms, _ := strconv.ParseInt(v["enqueued_ms"], 10, 64)
	at := time.UnixMilli(ms).UTC()
	return Ticket{
		ID: id, MatchType: v["match_type"], Format: v["format"], State: StateSearching, EnqueuedAt: at,
		WaitMS: s.d.Clock.Now().Sub(at).Milliseconds(),
	}, nil
}

// Leave removes the caller's ticket; leaving twice is harmless.
func (s *Service) Leave(ctx context.Context, user uuid.UUID) error {
	id, err := s.ticketOf(ctx, user)
	if err != nil {
		var e *apierror.Error
		if errors.As(err, &e) && e.Code == apierror.NotFound {
			return nil
		}
		return err
	}
	pipe := s.d.Redis.TxPipeline()
	pipe.ZRem(ctx, queueKey, id.String())
	pipe.Del(ctx, ticketKey(id), userKey(user))
	_, err = pipe.Exec(ctx)
	return err
}

func (s *Service) ticketOf(ctx context.Context, user uuid.UUID) (uuid.UUID, error) {
	v, err := s.d.Redis.Get(ctx, userKey(user)).Result()
	if errors.Is(err, goredis.Nil) {
		return uuid.Nil, apierror.New(apierror.NotFound)
	}
	if err != nil {
		return uuid.Nil, err
	}
	id, err := uuid.Parse(v)
	if err != nil {
		return uuid.Nil, apierror.New(apierror.NotFound)
	}
	return id, nil
}

// popPair removes the two oldest tickets of a queue atomically.
var popPair = goredis.NewScript(`
local ids = redis.call('ZRANGE', KEYS[1], 0, 1, 'WITHSCORES')
if #ids < 4 then return {} end
redis.call('ZREM', KEYS[1], ids[1], ids[3])
return ids`)

// Run pairs tickets every 500 ms while it holds the queue's matcher lease.
func (s *Service) Run(ctx context.Context) error {
	tick := s.d.Clock.NewTicker(MatchEvery)
	defer tick.Stop()
	owner := ids.NewString()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tick.C():
			ok, err := s.d.Redis.SetNX(ctx, lockKey, owner, matcherLease).Result()
			if err != nil || !ok {
				if v, _ := s.d.Redis.Get(ctx, lockKey).Result(); v != owner {
					continue
				}
				s.d.Redis.Expire(ctx, lockKey, matcherLease)
			}
			s.match(ctx)
		}
	}
}

type queued struct {
	id     uuid.UUID
	score  string
	user   uuid.UUID
	locale string
}

// match pairs tickets until fewer than two are waiting.
func (s *Service) match(ctx context.Context) {
	for {
		res, err := popPair.Run(ctx, s.d.Redis, []string{queueKey}).StringSlice()
		if err != nil || len(res) < 4 {
			return
		}
		pair := []queued{{score: res[1]}, {score: res[3]}}
		pair[0].id, _ = uuid.Parse(res[0])
		pair[1].id, _ = uuid.Parse(res[2])
		var alive []queued
		for _, q := range pair {
			v, err := s.d.Redis.HGetAll(ctx, ticketKey(q.id)).Result()
			if err != nil || len(v) == 0 {
				continue // expired or left
			}
			q.user, _ = uuid.Parse(v["user_id"])
			q.locale = v["locale"]
			alive = append(alive, q)
		}
		if len(alive) < 2 {
			s.requeue(ctx, alive)
			continue
		}
		s.start(ctx, alive[0], alive[1])
	}
}

func (s *Service) requeue(ctx context.Context, qs []queued) {
	for _, q := range qs {
		score, _ := strconv.ParseFloat(q.score, 64)
		s.d.Redis.ZAdd(ctx, queueKey, goredis.Z{Score: score, Member: q.id.String()})
	}
}

func (s *Service) drop(ctx context.Context, q queued) {
	s.d.Redis.Del(ctx, ticketKey(q.id), userKey(q.user))
}

// start creates the match of a pair: both players get mm.match_found; a
// player who meanwhile entered another match loses the ticket and the
// other goes back to the front of the queue.
func (s *Service) start(ctx context.Context, a, b queued) {
	c, err := s.d.Game.Start(ctx, game.CreateRequest{
		Config: game.Config{MatchType: "quick_duel", Format: "1v1", Origin: "matchmaking"},
		Sides:  [2][]game.Member{{{UserID: a.user, Locale: a.locale}}, {{UserID: b.user, Locale: b.locale}}},
	})
	if err != nil {
		var e *apierror.Error
		if errors.As(err, &e) && e.Code == apierror.AlreadyInMatch {
			busy, _ := e.Details["user_id"].(uuid.UUID)
			for _, q := range []queued{a, b} {
				if q.user == busy {
					s.drop(ctx, q)
					s.notify(ctx, q, map[string]any{"ticket_id": q.id, "state": "cancelled", "reason": apierror.AlreadyInMatch})
				} else {
					s.requeue(ctx, []queued{q})
				}
			}
			return
		}
		s.d.Logger.ErrorContext(ctx, "matchmaking: starting a match failed", "error", err)
		for _, q := range []queued{a, b} {
			s.drop(ctx, q)
			reason := apierror.Internal
			if e != nil {
				reason = e.Code
			}
			s.notify(ctx, q, map[string]any{"ticket_id": q.id, "state": "cancelled", "reason": reason})
		}
		return
	}
	for _, q := range []queued{a, b} {
		s.drop(ctx, q)
		if err := s.d.Notifier.Send(ctx, realtime.Target{User: q.user}, realtime.Frame{T: "mm.match_found", D: map[string]any{
			"ticket_id": q.id, "match_id": c.Match.ID, "ready_deadline_at": c.Match.ReadyDeadMS,
		}}); err != nil {
			s.d.Logger.WarnContext(ctx, "matchmaking: notifying a player failed", "error", err)
		}
	}
}

func (s *Service) notify(ctx context.Context, q queued, d map[string]any) {
	if err := s.d.Notifier.Send(ctx, realtime.Target{User: q.user}, realtime.Frame{T: "mm.status", D: d}); err != nil {
		s.d.Logger.WarnContext(ctx, "matchmaking: notifying a player failed", "error", err)
	}
}
