// Package service is the matchmaking queue (redis-keys §3, mode-matrix
// §3.4): one queue per matchmade 1v1 configuration, paired by rating with
// a search range that widens while a ticket waits, never pairing players
// who block each other. Matches cancelled before the start put the ready
// players back at the front of the queue; missed ready checks and leaving
// matches lead to short queue cooldowns (D-034, D-048). Team, couple and
// party queues arrive with Phase 6.
package service

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"slices"
	"strconv"
	"strings"
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

// Queue settings (redis-keys §3, D-032, D-068).
const (
	TicketTTL    = 10 * time.Minute
	MatchEvery   = 500 * time.Millisecond
	matcherLease = 5 * time.Second
	// matchMemory keeps the tickets of a new match until its ready check
	// is over, to re-queue the ready players.
	matchMemory = 2 * time.Minute
)

// Search range (D-068): base + RD/5, +50 every 5 s of waiting; Ranked
// stops at ±600, casual queues accept anyone from then on.
const (
	RangeBase   = 100
	RangeStep   = 50
	RangeEvery  = 5 * time.Second
	RangeMax    = 600
	unlimitedMS = math.MaxInt32
)

// Cooldowns (D-034, D-048).
const (
	MissedReadyWindow   = 30 * time.Minute
	MissedReadyLimit    = 3
	MissedReadyCooldown = 5 * time.Minute
	LeaveWindow         = 24 * time.Hour
	SecondLeaveCooldown = 5 * time.Minute
	ThirdLeaveCooldown  = 15 * time.Minute
)

// Ticket states.
const (
	StateSearching = "searching"
	StateFound     = "found"
)

// Rating is a player's rating and deviation (Glicko-2, D-003).
type Rating struct {
	Value float64
	RD    float64
}

// RatingReader returns players' ratings per bucket (ranking, Phase 7;
// backend-modules §2 upward read).
type RatingReader interface {
	Rating(ctx context.Context, user uuid.UUID, bucket string) (Rating, error)
}

// DefaultRatings is the RatingReader until ranking exists: every player
// is new (Glicko-2 defaults).
type DefaultRatings struct{}

// Rating implements RatingReader.
func (DefaultRatings) Rating(context.Context, uuid.UUID, string) (Rating, error) {
	return Rating{Value: 1500, RD: 350}, nil
}

// BlockReader lists users blocking or blocked by a user (social, wired in
// app).
type BlockReader interface {
	BlockedWith(ctx context.Context, user uuid.UUID) ([]uuid.UUID, error)
}

// Started is a match game created and handed to its live owner.
type Started struct {
	MatchID         uuid.UUID
	ReadyDeadlineMS int64
}

// Game is what matchmaking needs from game (adapted in matchmaking.go).
type Game interface {
	Start(ctx context.Context, r game.CreateRequest) (Started, error)
	Practice(ctx context.Context, user uuid.UUID, locale string) (Started, error)
	LiveMatch(ctx context.Context, user uuid.UUID) (uuid.UUID, bool)
	CategoryExists(ctx context.Context, id uuid.UUID) (bool, error)
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
	Ratings  RatingReader
	Blocks   BlockReader
	Clock    clock.Clock
	Logger   *slog.Logger
	Practice bool
}

// Service is the queue.
type Service struct{ d Deps }

// New returns the service.
func New(d Deps) *Service {
	if d.Ratings == nil {
		d.Ratings = DefaultRatings{}
	}
	return &Service{d: d}
}

// Request is the body of POST /v1/matchmaking/tickets.
type Request struct {
	MatchType string `json:"match_type"`
	Format    string `json:"format"`
	// CategoryMode: blind_challenge, category_master (with category_id, a
	// top-level category) or mixed_battle (the server picks 3); not in Ranked.
	CategoryMode string     `json:"category_mode,omitempty"`
	CategoryID   *uuid.UUID `json:"category_id,omitempty"`
	// Practice plays against the server's bot (development builds only).
	Practice bool `json:"practice,omitempty"`
}

// Ticket is a queue entry as the app sees it.
type Ticket struct {
	ID           uuid.UUID  `json:"id"`
	MatchType    string     `json:"match_type"`
	Format       string     `json:"format"`
	CategoryMode string     `json:"category_mode,omitempty"`
	CategoryID   *uuid.UUID `json:"category_id,omitempty"`
	State        string     `json:"state"`
	EnqueuedAt   time.Time  `json:"enqueued_at"`
	WaitMS       int64      `json:"wait_ms"`
	// Range is the current rating search range (±); null = anyone.
	Range   *int       `json:"range"`
	MatchID *uuid.UUID `json:"match_id,omitempty"`
}

// entry is a ticket as stored in mm:t:<id>; it is also what a new match
// remembers to re-queue its ready players.
type entry struct {
	ID         uuid.UUID   `json:"id"`
	User       uuid.UUID   `json:"user_id"`
	Locale     string      `json:"locale"`
	Queue      string      `json:"queue"`
	Request    Request     `json:"request"`
	Rating     float64     `json:"rating"`
	RD         float64     `json:"rd"`
	EnqueuedMS int64       `json:"enqueued_ms"`
	Blocks     []uuid.UUID `json:"blocks"`
}

var queuesKey = redis.Key("mm", "queues")

func queueKey(q string) string          { return redis.Key("mm", "q", q) }
func lockKey(q string) string           { return redis.Key("mm", "lock", q) }
func ticketKey(id uuid.UUID) string     { return redis.Key("mm", "t", id.String()) }
func userKey(user uuid.UUID) string     { return redis.Key("mm", "user", user.String()) }
func matchKey(id uuid.UUID) string      { return redis.Key("mm", "m", id.String()) }
func cooldownKey(user uuid.UUID) string { return redis.Key("mm", "cooldown", user.String()) }
func missedKey(user uuid.UUID) string   { return redis.Key("mm", "missed", user.String()) }
func leavesKey(user uuid.UUID) string   { return redis.Key("mm", "leaves", user.String()) }
func isRanked(r Request) bool           { return r.MatchType == "ranked" }
func ratingBucket(Request) string       { return "solo" } // 1v1 queues (mode-matrix §2.5)
func (r Request) config() game.Config   { return r.configWith(nil) }
func (s *Service) now() time.Time       { return s.d.Clock.Now().UTC() }
func (s *Service) nowMS() int64         { return s.d.Clock.Now().UnixMilli() }
func wait(e *entry, now int64) time.Duration {
	return time.Duration(now-e.EnqueuedMS) * time.Millisecond
}

func (r Request) configWith(cats []uuid.UUID) game.Config {
	c := game.Config{MatchType: game.MatchType(r.MatchType), Format: game.Format(r.Format), Origin: game.OriginMatchmaking,
		CategoryMode: game.CategoryMode(r.CategoryMode), CategoryIDs: cats}
	if r.CategoryID != nil {
		c.CategoryIDs = []uuid.UUID{*r.CategoryID}
	}
	return c
}

// queueName is mode-matrix §3.4's key. Questions come in both languages,
// so the locale group is always "any" (D-068).
func queueName(r Request) string {
	bucket := "casual"
	if isRanked(r) {
		bucket = ratingBucket(r)
	}
	mode := r.CategoryMode
	if mode == "" {
		mode = "default"
	}
	cat := "-"
	if r.CategoryID != nil {
		cat = r.CategoryID.String()
	}
	return strings.Join([]string{r.MatchType, r.Format, bucket, mode, cat, "any"}, ":")
}

// searchRange is a ticket's rating range after waiting w; unlimitedMS
// means anyone.
func searchRange(e *entry, w time.Duration) int {
	r := RangeBase + int(e.RD/5) + RangeStep*int(w/RangeEvery)
	if r >= RangeMax {
		if isRanked(e.Request) {
			return RangeMax
		}
		return unlimitedMS
	}
	return r
}

// validate checks a request against the mode matrix (1v1 until Phase 6).
func (s *Service) validate(ctx context.Context, r Request) error {
	if r.Format != "1v1" {
		return apierror.New(apierror.ModeNotAllowed).With("reason", "format_not_available_yet")
	}
	if (r.CategoryMode == "category_master") != (r.CategoryID != nil) {
		return apierror.Validation(apierror.FieldError{Field: "category_id", Rule: "required_for_category_master"})
	}
	if vs := game.ValidateConfig(r.config()); len(vs) > 0 {
		return apierror.New(apierror.ModeNotAllowed).With("rule_id", vs[0].RuleID).With("violations", vs)
	}
	if r.CategoryID != nil {
		ok, err := s.d.Game.CategoryExists(ctx, *r.CategoryID)
		if err != nil {
			return err
		}
		if !ok {
			return apierror.New(apierror.ModeNotAllowed).With("rule_id", "R12")
		}
	}
	return nil
}

// Enqueue puts the caller in the queue of the requested configuration, or
// starts a practice match right away.
func (s *Service) Enqueue(ctx context.Context, user uuid.UUID, locale string, r Request) (Ticket, error) {
	if r.Practice {
		if !s.d.Practice {
			return Ticket{}, apierror.Validation(apierror.FieldError{Field: "practice", Rule: "not_available"})
		}
		r.MatchType, r.Format, r.CategoryMode, r.CategoryID = "quick_duel", "1v1", "", nil
	}
	if err := s.validate(ctx, r); err != nil {
		return Ticket{}, err
	}
	if ttl, _ := s.d.Redis.PTTL(ctx, cooldownKey(user)).Result(); ttl > 0 {
		return Ticket{}, apierror.New(apierror.QueueCooldown).WithRetryAfter(ttl)
	}
	if _, live := s.d.Game.LiveMatch(ctx, user); live {
		return Ticket{}, apierror.New(apierror.AlreadyInMatch)
	}
	if r.Practice {
		if exists, _ := s.d.Redis.Exists(ctx, userKey(user)).Result(); exists > 0 {
			return Ticket{}, apierror.New(apierror.AlreadyQueued)
		}
		c, err := s.d.Game.Practice(ctx, user, locale)
		if err != nil {
			return Ticket{}, err
		}
		return Ticket{ID: ids.New(), MatchType: r.MatchType, Format: r.Format, State: StateFound, EnqueuedAt: s.now(), MatchID: &c.MatchID}, nil
	}
	rating, err := s.d.Ratings.Rating(ctx, user, ratingBucket(r))
	if err != nil {
		return Ticket{}, err
	}
	var blocks []uuid.UUID
	if s.d.Blocks != nil {
		if blocks, err = s.d.Blocks.BlockedWith(ctx, user); err != nil {
			return Ticket{}, err
		}
	}
	e := entry{ID: ids.New(), User: user, Locale: locale, Queue: queueName(r), Request: r, Rating: rating.Value, RD: rating.RD,
		EnqueuedMS: s.nowMS(), Blocks: blocks}
	if err := s.put(ctx, &e); err != nil {
		return Ticket{}, err
	}
	return s.ticket(&e), nil
}

// put stores a ticket and adds it to its queue (one ticket per user).
func (s *Service) put(ctx context.Context, e *entry) error {
	ok, err := s.d.Redis.SetNX(ctx, userKey(e.User), e.ID.String(), TicketTTL).Result()
	if err != nil {
		return err
	}
	if !ok {
		return apierror.New(apierror.AlreadyQueued)
	}
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	pipe := s.d.Redis.TxPipeline()
	pipe.Set(ctx, ticketKey(e.ID), b, TicketTTL)
	pipe.ZAdd(ctx, queueKey(e.Queue), goredis.Z{Score: e.Rating, Member: e.ID.String()})
	pipe.SAdd(ctx, queuesKey, e.Queue)
	if _, err := pipe.Exec(ctx); err != nil {
		s.d.Redis.Del(ctx, userKey(e.User))
		return err
	}
	return nil
}

func (s *Service) ticket(e *entry) Ticket {
	w := wait(e, s.nowMS())
	t := Ticket{
		ID: e.ID, MatchType: e.Request.MatchType, Format: e.Request.Format, CategoryMode: e.Request.CategoryMode,
		CategoryID: e.Request.CategoryID, State: StateSearching, EnqueuedAt: time.UnixMilli(e.EnqueuedMS).UTC(), WaitMS: w.Milliseconds(),
	}
	if r := searchRange(e, w); r != unlimitedMS {
		t.Range = &r
	}
	return t
}

func (s *Service) load(ctx context.Context, id uuid.UUID) (*entry, error) {
	b, err := s.d.Redis.Get(ctx, ticketKey(id)).Bytes()
	if errors.Is(err, goredis.Nil) {
		return nil, apierror.New(apierror.NotFound)
	}
	if err != nil {
		return nil, err
	}
	var e entry
	return &e, json.Unmarshal(b, &e)
}

// Current returns the caller's ticket (NOT_FOUND when not queued).
func (s *Service) Current(ctx context.Context, user uuid.UUID) (Ticket, error) {
	id, err := s.ticketOf(ctx, user)
	if err != nil {
		return Ticket{}, err
	}
	e, err := s.load(ctx, id)
	if err != nil {
		return Ticket{}, err
	}
	return s.ticket(e), nil
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
	e, err := s.load(ctx, id)
	if err == nil {
		s.d.Redis.ZRem(ctx, queueKey(e.Queue), id.String())
	}
	s.d.Redis.Del(ctx, ticketKey(id), userKey(user))
	return nil
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

// Run pairs tickets every 500 ms in every queue whose matcher lease this
// instance holds.
func (s *Service) Run(ctx context.Context) error {
	tick := s.d.Clock.NewTicker(MatchEvery)
	defer tick.Stop()
	owner := ids.NewString()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tick.C():
			queues, err := s.d.Redis.SMembers(ctx, queuesKey).Result()
			if err != nil {
				continue
			}
			for _, q := range queues {
				if s.lease(ctx, q, owner) {
					s.match(ctx, q)
				}
			}
		}
	}
}

func (s *Service) lease(ctx context.Context, q, owner string) bool {
	ok, err := s.d.Redis.SetNX(ctx, lockKey(q), owner, matcherLease).Result()
	if err == nil && ok {
		return true
	}
	if v, _ := s.d.Redis.Get(ctx, lockKey(q)).Result(); v == owner {
		s.d.Redis.Expire(ctx, lockKey(q), matcherLease)
		return true
	}
	return false
}

// claim removes two tickets from a queue only if both are still there.
var claim = goredis.NewScript(`
if redis.call('ZSCORE', KEYS[1], ARGV[1]) and redis.call('ZSCORE', KEYS[1], ARGV[2]) then
  redis.call('ZREM', KEYS[1], ARGV[1], ARGV[2])
  return 1
end
return 0`)

// match pairs the waiting tickets of a queue: oldest first, each with the
// closest rating inside both players' ranges, never across a block.
func (s *Service) match(ctx context.Context, q string) {
	members, err := s.d.Redis.ZRange(ctx, queueKey(q), 0, -1).Result()
	if err != nil || len(members) < 2 {
		return
	}
	var waiting []*entry
	for _, m := range members {
		id, err := uuid.Parse(m)
		if err != nil {
			s.d.Redis.ZRem(ctx, queueKey(q), m)
			continue
		}
		e, err := s.load(ctx, id)
		if err != nil {
			s.d.Redis.ZRem(ctx, queueKey(q), m) // expired
			continue
		}
		waiting = append(waiting, e)
	}
	slices.SortFunc(waiting, func(a, b *entry) int { return int(a.EnqueuedMS - b.EnqueuedMS) })
	now := s.nowMS()
	taken := map[uuid.UUID]bool{}
	for i, a := range waiting {
		if taken[a.ID] {
			continue
		}
		var best *entry
		bestDiff := math.MaxFloat64
		for _, b := range waiting[i+1:] {
			if taken[b.ID] || a.User == b.User || slices.Contains(a.Blocks, b.User) || slices.Contains(b.Blocks, a.User) {
				continue
			}
			diff := math.Abs(a.Rating - b.Rating)
			if diff > float64(min(searchRange(a, wait(a, now)), searchRange(b, wait(b, now)))) || diff >= bestDiff {
				continue
			}
			best, bestDiff = b, diff
		}
		if best == nil {
			continue
		}
		if ok, err := claim.Run(ctx, s.d.Redis, []string{queueKey(q)}, a.ID.String(), best.ID.String()).Int(); err != nil || ok != 1 {
			continue
		}
		taken[a.ID], taken[best.ID] = true, true
		s.start(ctx, a, best)
	}
}

func (s *Service) requeue(ctx context.Context, es ...*entry) {
	for _, e := range es {
		s.d.Redis.ZAdd(ctx, queueKey(e.Queue), goredis.Z{Score: e.Rating, Member: e.ID.String()})
	}
}

func (s *Service) drop(ctx context.Context, e *entry) {
	s.d.Redis.Del(ctx, ticketKey(e.ID), userKey(e.User))
}

// start creates the match of a pair: both players get mm.match_found; a
// player who meanwhile entered another match loses the ticket and the
// other goes back to the queue.
func (s *Service) start(ctx context.Context, a, b *entry) {
	c, err := s.d.Game.Start(ctx, game.CreateRequest{
		Config: a.Request.config(),
		Sides:  [2][]game.Member{{{UserID: a.User, Locale: a.Locale}}, {{UserID: b.User, Locale: b.Locale}}},
	})
	if err != nil {
		var e *apierror.Error
		if errors.As(err, &e) && e.Code == apierror.AlreadyInMatch {
			busy, _ := e.Details["user_id"].(uuid.UUID)
			for _, x := range []*entry{a, b} {
				if x.User == busy {
					s.drop(ctx, x)
					s.notify(ctx, x.User, map[string]any{"ticket_id": x.ID, "state": "cancelled", "reason": apierror.AlreadyInMatch})
				} else {
					s.requeue(ctx, x)
				}
			}
			return
		}
		s.d.Logger.ErrorContext(ctx, "matchmaking: starting a match failed", "queue", a.Queue, "error", err)
		reason := apierror.Internal
		if e != nil {
			reason = e.Code
		}
		for _, x := range []*entry{a, b} {
			s.drop(ctx, x)
			s.notify(ctx, x.User, map[string]any{"ticket_id": x.ID, "state": "cancelled", "reason": reason})
		}
		return
	}
	// Remember the tickets until the ready check is over (re-queue, D-034).
	pipe := s.d.Redis.TxPipeline()
	for _, x := range []*entry{a, b} {
		if bs, err := json.Marshal(x); err == nil {
			pipe.HSet(ctx, matchKey(c.MatchID), x.User.String(), bs)
		}
	}
	pipe.Expire(ctx, matchKey(c.MatchID), matchMemory)
	if _, err := pipe.Exec(ctx); err != nil {
		s.d.Logger.WarnContext(ctx, "matchmaking: remembering a match failed", "error", err)
	}
	for _, x := range []*entry{a, b} {
		s.drop(ctx, x)
		if err := s.d.Notifier.Send(ctx, realtime.Target{User: x.User}, realtime.Frame{T: "mm.match_found", D: map[string]any{
			"ticket_id": x.ID, "match_id": c.MatchID, "ready_deadline_at": c.ReadyDeadlineMS,
		}}); err != nil {
			s.d.Logger.WarnContext(ctx, "matchmaking: notifying a player failed", "error", err)
		}
	}
}

func (s *Service) notify(ctx context.Context, user uuid.UUID, d map[string]any) {
	if err := s.d.Notifier.Send(ctx, realtime.Target{User: user}, realtime.Frame{T: "mm.status", D: d}); err != nil {
		s.d.Logger.WarnContext(ctx, "matchmaking: notifying a player failed", "error", err)
	}
}

// OnEnd reacts to a finished matchmade match: a match cancelled before the
// start re-queues its ready players at their old place and counts a
// missed ready check for the others (3 in 30 min → 5 min cooldown,
// D-034); leaving a match counts toward the leaver cooldown (2nd leave in
// 24 h → 5 min, 3rd and later → 15 min, D-048).
func (s *Service) OnEnd(ctx context.Context, m game.Ended) {
	if m.Origin != game.OriginMatchmaking {
		return
	}
	beforeStart := m.Status == game.StatusCancelled &&
		(m.CancelReason == game.CancelNotAllReady || m.CancelReason == game.CancelLeftBeforeStart)
	if !beforeStart {
		for _, p := range m.Participants {
			if p.Status == game.ParticipantLeft || p.Status == game.ParticipantForfeited {
				n := s.count(ctx, leavesKey(p.UserID), LeaveWindow)
				switch {
				case n >= 3:
					s.cooldown(ctx, p.UserID, ThirdLeaveCooldown)
				case n == 2:
					s.cooldown(ctx, p.UserID, SecondLeaveCooldown)
				}
			}
		}
		return
	}
	saved, _ := s.d.Redis.HGetAll(ctx, matchKey(m.MatchID)).Result()
	s.d.Redis.Del(ctx, matchKey(m.MatchID))
	for _, p := range m.Participants {
		if p.Status != game.ParticipantReady {
			if s.count(ctx, missedKey(p.UserID), MissedReadyWindow) >= MissedReadyLimit {
				s.cooldown(ctx, p.UserID, MissedReadyCooldown)
			}
			continue
		}
		var e entry
		if err := json.Unmarshal([]byte(saved[p.UserID.String()]), &e); err != nil {
			continue
		}
		e.ID = ids.New() // a new ticket, still at its old place in line
		if err := s.put(ctx, &e); err != nil {
			s.d.Logger.WarnContext(ctx, "matchmaking: re-queueing a ready player failed", "error", err)
			continue
		}
		s.notify(ctx, e.User, map[string]any{"ticket_id": e.ID, "state": StateSearching, "reason": "requeued", "ticket": s.ticket(&e)})
	}
}

// count records one occurrence now and returns the occurrences in window.
func (s *Service) count(ctx context.Context, key string, window time.Duration) int64 {
	now := s.nowMS()
	pipe := s.d.Redis.TxPipeline()
	pipe.ZAdd(ctx, key, goredis.Z{Score: float64(now), Member: ids.NewString()})
	pipe.ZRemRangeByScore(ctx, key, "-inf", strconv.FormatInt(now-window.Milliseconds(), 10))
	n := pipe.ZCard(ctx, key)
	pipe.Expire(ctx, key, window)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0
	}
	return n.Val()
}

// cooldown blocks queueing for d, keeping a longer running cooldown.
func (s *Service) cooldown(ctx context.Context, user uuid.UUID, d time.Duration) {
	if ttl, _ := s.d.Redis.PTTL(ctx, cooldownKey(user)).Result(); ttl >= d {
		return
	}
	s.d.Redis.Set(ctx, cooldownKey(user), "1", d)
	s.notify(ctx, user, map[string]any{"state": "cooldown", "ends_at": s.d.Clock.Now().Add(d).UnixMilli()})
}
