// Package service is the matchmaking queue (redis-keys §3, mode-matrix
// §3.4): one queue per matchmade 1v1 configuration, paired by rating with
// a search range that widens while a ticket waits, never pairing players
// who block each other. Matches cancelled before the start put the ready
// players back at the front of the queue; missed ready checks and leaving
// matches lead to short queue cooldowns (D-034, D-048). Team formats fill
// each side from solo tickets, parties of friends, a persistent team or a
// couple (Phase 6, D-072).
package service

import (
	"cmp"
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
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/team"
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

// Social is what matchmaking reads from social (wired in app): blocks and
// friendships (parties are friends of the leader).
type Social interface {
	BlockedWith(ctx context.Context, user uuid.UUID) ([]uuid.UUID, error)
	AreFriends(ctx context.Context, a, b uuid.UUID) (bool, error)
}

// Teams is what matchmaking needs from team.
type Teams interface {
	TeamInfo(ctx context.Context, id uuid.UUID) (team.Info, error)
	ActiveCoupleOf(ctx context.Context, user uuid.UUID) (team.CoupleInfo, bool, error)
}

// LocaleReader gives party members' content languages (profile).
type LocaleReader interface {
	LocalesOf(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error)
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
	Social   Social
	// Sanctions keeps sanctioned players out of the queues (moderation).
	Sanctions game.SanctionChecker
	Teams     Teams
	Locales   LocaleReader
	Clock     clock.Clock
	Logger    *slog.Logger
	Practice  bool
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

// SetTeams wires team (persistent teams and couples).
func (s *Service) SetTeams(t Teams) { s.d.Teams = t }

// SetLocales wires the party members' languages (profile).
func (s *Service) SetLocales(l LocaleReader) { s.d.Locales = l }

// Request is the body of POST /v1/matchmaking/tickets.
type Request struct {
	MatchType string `json:"match_type"`
	// Format: 1v1, 2v2, 3v3, 4v4 or couple (the caller's active couple).
	Format string `json:"format"`
	// CategoryMode: blind_challenge, category_master (with category_id, a
	// top-level category) or mixed_battle (the server picks 3); not in Ranked.
	CategoryMode string     `json:"category_mode,omitempty"`
	CategoryID   *uuid.UUID `json:"category_id,omitempty"`
	// PartyUserIDs are friends queueing with the caller on one side (team
	// formats); with TeamID they are the lineup of that persistent team.
	PartyUserIDs []uuid.UUID `json:"party_user_ids,omitempty"`
	TeamID       *uuid.UUID  `json:"team_id,omitempty"`
	// Practice plays against the server's bot (development builds only).
	Practice bool `json:"practice,omitempty"`
}

// Ticket is a queue entry as the app sees it.
type Ticket struct {
	ID           uuid.UUID   `json:"id"`
	MatchType    string      `json:"match_type"`
	Format       string      `json:"format"`
	CategoryMode string      `json:"category_mode,omitempty"`
	CategoryID   *uuid.UUID  `json:"category_id,omitempty"`
	UserIDs      []uuid.UUID `json:"user_ids"`
	State        string      `json:"state"`
	EnqueuedAt   time.Time   `json:"enqueued_at"`
	WaitMS       int64       `json:"wait_ms"`
	// Range is the current rating search range (±); null = anyone.
	Range   *int       `json:"range"`
	MatchID *uuid.UUID `json:"match_id,omitempty"`
}

// member is one player of a ticket.
type member struct {
	UserID uuid.UUID `json:"user_id"`
	Locale string    `json:"locale"`
}

// entry is a ticket as stored in mm:t:<id>: one player, a party, a team
// lineup or a couple; it is also what a new match remembers to re-queue
// its ready players.
type entry struct {
	ID         uuid.UUID   `json:"id"`
	Members    []member    `json:"members"`
	Queue      string      `json:"queue"`
	Request    Request     `json:"request"`
	Rating     float64     `json:"rating"`
	RD         float64     `json:"rd"`
	EnqueuedMS int64       `json:"enqueued_ms"`
	Blocks     []uuid.UUID `json:"blocks"`
	TeamID     *uuid.UUID  `json:"team_id,omitempty"`
	CoupleID   *uuid.UUID  `json:"couple_id,omitempty"`
}

func (e *entry) users() []uuid.UUID {
	out := make([]uuid.UUID, len(e.Members))
	for i, m := range e.Members {
		out[i] = m.UserID
	}
	return out
}

// persistent tickets fill a whole side on their own.
func (e *entry) persistent() bool { return e.TeamID != nil || e.CoupleID != nil }

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
func (r Request) config() game.Config   { return r.configWith(nil) }
func (r Request) sideSize() int         { return game.Format(r.Format).SideSize() }
func (s *Service) now() time.Time       { return s.d.Clock.Now().UTC() }
func (s *Service) nowMS() int64         { return s.d.Clock.Now().UnixMilli() }
func wait(e *entry, now int64) time.Duration {
	return time.Duration(now-e.EnqueuedMS) * time.Millisecond
}

// ratingBucket is mode-matrix §2.5's bucket of a format.
func ratingBucket(r Request) string {
	switch r.Format {
	case "1v1":
		return "solo"
	case "couple":
		return "couple"
	}
	return "team"
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

// validate checks a request against the mode matrix (Fast Team Round +
// average for team queues, D-053).
func (s *Service) validate(ctx context.Context, r Request) error {
	if r.sideSize() == 0 {
		return apierror.Validation(apierror.FieldError{Field: "format", Rule: "enum"})
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
	if len(r.PartyUserIDs)+1 > r.sideSize() || (r.Format == "1v1" && len(r.PartyUserIDs) > 0) {
		return apierror.Validation(apierror.FieldError{Field: "party_user_ids", Rule: "too_many"})
	}
	if r.TeamID != nil && (r.Format == "1v1" || r.Format == "couple" || len(r.PartyUserIDs)+1 != r.sideSize()) {
		return apierror.Validation(apierror.FieldError{Field: "team_id", Rule: "full_lineup_of_a_team_format"})
	}
	return nil
}

// members builds the ticket's players: the caller, their party (friends),
// their team lineup or their couple.
func (s *Service) members(ctx context.Context, user uuid.UUID, r Request) (users []uuid.UUID, teamID, coupleID *uuid.UUID, err error) {
	users = []uuid.UUID{user}
	switch {
	case r.Format == "couple":
		if s.d.Teams == nil {
			return nil, nil, nil, apierror.New(apierror.ModeNotAllowed).With("rule_id", "R16")
		}
		c, ok, err := s.d.Teams.ActiveCoupleOf(ctx, user)
		if err != nil {
			return nil, nil, nil, err
		}
		if !ok {
			return nil, nil, nil, apierror.New(apierror.ModeNotAllowed).With("rule_id", "R16").With("reason", "no_active_couple")
		}
		partner := c.Users[0]
		if partner == user {
			partner = c.Users[1]
		}
		id := c.ID
		return []uuid.UUID{user, partner}, nil, &id, nil
	case r.TeamID != nil:
		if s.d.Teams == nil {
			return nil, nil, nil, apierror.New(apierror.ModeNotAllowed).With("rule_id", "R16")
		}
		t, err := s.d.Teams.TeamInfo(ctx, *r.TeamID)
		if err != nil {
			return nil, nil, nil, err
		}
		for _, u := range append([]uuid.UUID{user}, r.PartyUserIDs...) {
			if !t.Active || !slices.Contains(t.Members, u) {
				return nil, nil, nil, apierror.New(apierror.ModeNotAllowed).With("rule_id", "R16").With("reason", "not_a_member").With("user_id", u)
			}
		}
		teamID = r.TeamID
	default:
		for _, u := range r.PartyUserIDs {
			ok, err := s.d.Social.AreFriends(ctx, user, u)
			if err != nil {
				return nil, nil, nil, err
			}
			if !ok {
				return nil, nil, nil, apierror.New(apierror.Forbidden).With("reason", "party_members_are_friends").With("user_id", u)
			}
		}
	}
	for _, u := range r.PartyUserIDs {
		if u == user || slices.Contains(users, u) {
			return nil, nil, nil, apierror.Validation(apierror.FieldError{Field: "party_user_ids", Rule: "unique"})
		}
		users = append(users, u)
	}
	return users, teamID, nil, nil
}

// Enqueue puts the caller — alone, with a party, a team lineup or their
// couple — in the queue of the requested configuration, or starts a
// practice match right away. Every member must be free (no ticket, no
// live match, no cooldown); party members confirm with the ready check.
func (s *Service) Enqueue(ctx context.Context, user uuid.UUID, locale string, r Request) (Ticket, error) {
	if r.Practice {
		if !s.d.Practice {
			return Ticket{}, apierror.Validation(apierror.FieldError{Field: "practice", Rule: "not_available"})
		}
		r = Request{MatchType: "quick_duel", Format: "1v1", Practice: true}
	}
	if err := s.validate(ctx, r); err != nil {
		return Ticket{}, err
	}
	users, teamID, coupleID, err := s.members(ctx, user, r)
	if err != nil {
		return Ticket{}, err
	}
	for _, u := range users {
		if ttl, _ := s.d.Redis.PTTL(ctx, cooldownKey(u)).Result(); ttl > 0 {
			return Ticket{}, apierror.New(apierror.QueueCooldown).WithRetryAfter(ttl).With("user_id", u)
		}
		if _, live := s.d.Game.LiveMatch(ctx, u); live {
			return Ticket{}, apierror.New(apierror.AlreadyInMatch).With("user_id", u)
		}
		if s.d.Sanctions != nil {
			kind, err := s.d.Sanctions.Blocking(ctx, u, game.MatchType(r.MatchType), game.OriginMatchmaking)
			if err != nil {
				return Ticket{}, err
			}
			if kind != "" {
				return Ticket{}, apierror.New(apierror.Sanctioned).With("kind", kind).With("user_id", u)
			}
		}
	}
	if r.Practice {
		if exists, _ := s.d.Redis.Exists(ctx, userKey(user)).Result(); exists > 0 {
			return Ticket{}, apierror.New(apierror.AlreadyQueued)
		}
		c, err := s.d.Game.Practice(ctx, user, locale)
		if err != nil {
			return Ticket{}, err
		}
		return Ticket{ID: ids.New(), MatchType: r.MatchType, Format: r.Format, UserIDs: users, State: StateFound, EnqueuedAt: s.now(), MatchID: &c.MatchID}, nil
	}
	e := entry{ID: ids.New(), Queue: queueName(r), Request: r, EnqueuedMS: s.nowMS(), TeamID: teamID, CoupleID: coupleID}
	locales := map[uuid.UUID]string{user: locale}
	if len(users) > 1 && s.d.Locales != nil {
		if locales, err = s.d.Locales.LocalesOf(ctx, users); err != nil {
			return Ticket{}, err
		}
		locales[user] = locale
	}
	for _, u := range users {
		rating, err := s.d.Ratings.Rating(ctx, u, ratingBucket(r))
		if err != nil {
			return Ticket{}, err
		}
		e.Rating += rating.Value / float64(len(users))
		e.RD += rating.RD / float64(len(users))
		if s.d.Social != nil {
			blocks, err := s.d.Social.BlockedWith(ctx, u)
			if err != nil {
				return Ticket{}, err
			}
			e.Blocks = append(e.Blocks, blocks...)
		}
		l := locales[u]
		if l == "" {
			l = locale
		}
		e.Members = append(e.Members, member{UserID: u, Locale: l})
	}
	for _, u := range users {
		if slices.Contains(e.Blocks, u) {
			return Ticket{}, apierror.New(apierror.Blocked).With("user_id", u)
		}
	}
	if err := s.put(ctx, &e); err != nil {
		return Ticket{}, err
	}
	return s.ticket(&e), nil
}

// put stores a ticket and adds it to its queue (one ticket per user).
func (s *Service) put(ctx context.Context, e *entry) error {
	var taken []uuid.UUID
	release := func() {
		for _, u := range taken {
			s.d.Redis.Del(ctx, userKey(u))
		}
	}
	for _, u := range e.users() {
		ok, err := s.d.Redis.SetNX(ctx, userKey(u), e.ID.String(), TicketTTL).Result()
		if err != nil {
			release()
			return err
		}
		if !ok {
			release()
			return apierror.New(apierror.AlreadyQueued).With("user_id", u)
		}
		taken = append(taken, u)
	}
	b, err := json.Marshal(e)
	if err != nil {
		release()
		return err
	}
	pipe := s.d.Redis.TxPipeline()
	pipe.Set(ctx, ticketKey(e.ID), b, TicketTTL)
	pipe.ZAdd(ctx, queueKey(e.Queue), goredis.Z{Score: e.Rating, Member: e.ID.String()})
	pipe.SAdd(ctx, queuesKey, e.Queue)
	if _, err := pipe.Exec(ctx); err != nil {
		release()
		return err
	}
	return nil
}

func (s *Service) ticket(e *entry) Ticket {
	w := wait(e, s.nowMS())
	t := Ticket{
		ID: e.ID, MatchType: e.Request.MatchType, Format: e.Request.Format, CategoryMode: e.Request.CategoryMode,
		CategoryID: e.Request.CategoryID, UserIDs: e.users(), State: StateSearching,
		EnqueuedAt: time.UnixMilli(e.EnqueuedMS).UTC(), WaitMS: w.Milliseconds(),
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

// Leave removes the caller's ticket — for the whole party; the others
// get mm.status cancelled. Leaving twice is harmless.
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
	if err != nil {
		s.d.Redis.Del(ctx, ticketKey(id), userKey(user))
		return nil //nolint:nilerr // expired ticket: nothing left to leave
	}
	s.d.Redis.ZRem(ctx, queueKey(e.Queue), id.String())
	s.drop(ctx, e)
	for _, u := range e.users() {
		if u != user {
			s.notify(ctx, u, map[string]any{"ticket_id": id, "state": "cancelled", "reason": "party_left"})
		}
	}
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

// claim removes tickets from a queue only if all of them are still there.
var claim = goredis.NewScript(`
for i = 1, #ARGV do
  if not redis.call('ZSCORE', KEYS[1], ARGV[i]) then return 0 end
end
redis.call('ZREM', KEYS[1], unpack(ARGV))
return 1`)

// side is the tickets forming one side of a new match.
type side []*entry

func (sd side) size() int {
	n := 0
	for _, e := range sd {
		n += len(e.Members)
	}
	return n
}

// fits reports whether ticket e may join side sd (with the other side
// other) on anchor a's terms: within both ranges of a, no block with
// anyone already chosen.
func fits(e, a *entry, chosen []*entry, now int64) bool {
	diff := math.Abs(a.Rating - e.Rating)
	if diff > float64(min(searchRange(a, wait(a, now)), searchRange(e, wait(e, now)))) {
		return false
	}
	for _, c := range chosen {
		for _, u := range c.users() {
			if slices.Contains(e.Blocks, u) {
				return false
			}
		}
		for _, u := range e.users() {
			if slices.Contains(c.Blocks, u) {
				return false
			}
		}
	}
	return true
}

// match forms matches from the waiting tickets of a queue: the oldest
// ticket anchors side 1, each side is a persistent team or couple ticket,
// or temporary tickets (solo and parties) filled closest rating first;
// side 2 starts from the closest rating that fits, never across a block.
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
	for _, a := range waiting {
		if taken[a.ID] {
			continue
		}
		n := a.Request.sideSize()
		var chosen []*entry
		var sides [2]side
		build := func(i int, first *entry) bool {
			sides[i] = side{first}
			chosen = append(chosen, first)
			if first.persistent() {
				return sides[i].size() == n
			}
			for sides[i].size() < n {
				var best *entry
				bestDiff := math.MaxFloat64
				for _, e := range waiting {
					if taken[e.ID] || slices.Contains(chosen, e) || e.persistent() || sides[i].size()+len(e.Members) > n || !fits(e, a, chosen, now) {
						continue
					}
					if d := math.Abs(first.Rating - e.Rating); d < bestDiff {
						best, bestDiff = e, d
					}
				}
				if best == nil {
					return false
				}
				sides[i] = append(sides[i], best)
				chosen = append(chosen, best)
			}
			return true
		}
		if !build(0, a) {
			continue
		}
		// Side 2 starts from the closest rating that fits (the oldest on ties).
		var candidates []*entry
		for _, b := range waiting {
			if !taken[b.ID] && !slices.Contains(chosen, b) && len(b.Members) <= n && fits(b, a, chosen, now) {
				candidates = append(candidates, b)
			}
		}
		slices.SortStableFunc(candidates, func(x, y *entry) int {
			return cmp.Compare(math.Abs(a.Rating-x.Rating), math.Abs(a.Rating-y.Rating))
		})
		formed := false
		for _, b := range candidates {
			before := len(chosen)
			if build(1, b) {
				formed = true
				break
			}
			chosen = chosen[:before]
		}
		if !formed {
			continue
		}
		claimed := make([]any, 0, len(chosen))
		for _, e := range chosen {
			claimed = append(claimed, e.ID.String())
		}
		if ok, err := claim.Run(ctx, s.d.Redis, []string{queueKey(q)}, claimed...).Int(); err != nil || ok != 1 {
			continue
		}
		for _, e := range chosen {
			taken[e.ID] = true
		}
		s.start(ctx, sides)
	}
}

func (s *Service) requeue(ctx context.Context, es ...*entry) {
	for _, e := range es {
		s.d.Redis.ZAdd(ctx, queueKey(e.Queue), goredis.Z{Score: e.Rating, Member: e.ID.String()})
	}
}

func (s *Service) drop(ctx context.Context, e *entry) {
	keys := make([]string, 0, 1+len(e.Members))
	keys = append(keys, ticketKey(e.ID))
	for _, u := range e.users() {
		keys = append(keys, userKey(u))
	}
	s.d.Redis.Del(ctx, keys...)
}

// start creates the match of two sides: every player gets
// mm.match_found; a ticket whose player meanwhile entered another match is
// dropped and the others go back to the queue.
func (s *Service) start(ctx context.Context, sides [2]side) {
	req := game.CreateRequest{Config: sides[0][0].Request.config()}
	var all []*entry
	for i, sd := range sides {
		for _, e := range sd {
			for _, m := range e.Members {
				req.Sides[i] = append(req.Sides[i], game.Member{UserID: m.UserID, Locale: m.Locale})
			}
			all = append(all, e)
		}
		if len(sd) == 1 {
			req.Teams[i], req.Couples[i] = sd[0].TeamID, sd[0].CoupleID
		}
	}
	c, err := s.d.Game.Start(ctx, req)
	if err != nil {
		var e *apierror.Error
		if errors.As(err, &e) && e.Code == apierror.AlreadyInMatch {
			busy, _ := e.Details["user_id"].(uuid.UUID)
			for _, x := range all {
				if slices.Contains(x.users(), busy) {
					s.drop(ctx, x)
					s.tell(ctx, x, map[string]any{"ticket_id": x.ID, "state": "cancelled", "reason": apierror.AlreadyInMatch})
				} else {
					s.requeue(ctx, x)
				}
			}
			return
		}
		s.d.Logger.ErrorContext(ctx, "matchmaking: starting a match failed", "queue", sides[0][0].Queue, "error", err)
		reason := apierror.Internal
		if e != nil {
			reason = e.Code
		}
		for _, x := range all {
			s.drop(ctx, x)
			s.tell(ctx, x, map[string]any{"ticket_id": x.ID, "state": "cancelled", "reason": reason})
		}
		return
	}
	// Remember the tickets until the ready check is over (re-queue, D-034).
	pipe := s.d.Redis.TxPipeline()
	for _, x := range all {
		if bs, err := json.Marshal(x); err == nil {
			for _, u := range x.users() {
				pipe.HSet(ctx, matchKey(c.MatchID), u.String(), bs)
			}
		}
	}
	pipe.Expire(ctx, matchKey(c.MatchID), matchMemory)
	if _, err := pipe.Exec(ctx); err != nil {
		s.d.Logger.WarnContext(ctx, "matchmaking: remembering a match failed", "error", err)
	}
	for _, x := range all {
		s.drop(ctx, x)
		for _, u := range x.users() {
			if err := s.d.Notifier.Send(ctx, realtime.Target{User: u}, realtime.Frame{T: "mm.match_found", D: map[string]any{
				"ticket_id": x.ID, "match_id": c.MatchID, "ready_deadline_at": c.ReadyDeadlineMS,
			}}); err != nil {
				s.d.Logger.WarnContext(ctx, "matchmaking: notifying a player failed", "error", err)
			}
		}
	}
}

// tell sends mm.status to every player of a ticket.
func (s *Service) tell(ctx context.Context, e *entry, d map[string]any) {
	for _, u := range e.users() {
		s.notify(ctx, u, d)
	}
}

func (s *Service) notify(ctx context.Context, user uuid.UUID, d map[string]any) {
	if err := s.d.Notifier.Send(ctx, realtime.Target{User: user}, realtime.Frame{T: "mm.status", D: d}); err != nil {
		s.d.Logger.WarnContext(ctx, "matchmaking: notifying a player failed", "error", err)
	}
}

// OnEnd reacts to a finished matchmade match: a match cancelled before the
// start re-queues each ticket whose players were all ready at its old
// place and counts a missed ready check for the others (3 in 30 min → 5
// min cooldown, D-034); leaving a match counts toward the leaver cooldown
// (2nd leave in 24 h → 5 min, 3rd and later → 15 min, D-048).
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
	ready := map[uuid.UUID]bool{}
	for _, p := range m.Participants {
		if p.Status == game.ParticipantReady {
			ready[p.UserID] = true
			continue
		}
		if s.count(ctx, missedKey(p.UserID), MissedReadyWindow) >= MissedReadyLimit {
			s.cooldown(ctx, p.UserID, MissedReadyCooldown)
		}
	}
	done := map[uuid.UUID]bool{}
	for _, p := range m.Participants {
		var e entry
		if err := json.Unmarshal([]byte(saved[p.UserID.String()]), &e); err != nil || done[e.ID] {
			continue
		}
		done[e.ID] = true
		if slices.ContainsFunc(e.users(), func(u uuid.UUID) bool { return !ready[u] }) {
			continue // the party goes back only together
		}
		e.ID = ids.New() // a new ticket, still at its old place in line
		if err := s.put(ctx, &e); err != nil {
			s.d.Logger.WarnContext(ctx, "matchmaking: re-queueing ready players failed", "error", err)
			continue
		}
		s.tell(ctx, &e, map[string]any{"ticket_id": e.ID, "state": StateSearching, "reason": "requeued", "ticket": s.ticket(&e)})
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
