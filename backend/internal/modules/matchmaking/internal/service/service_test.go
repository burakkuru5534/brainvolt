package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/realtime"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/redis"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/testenv"
)

type fakeGame struct {
	mu      sync.Mutex
	started [][2]uuid.UUID
	reqs    []game.CreateRequest
}

func (g *fakeGame) Start(_ context.Context, r game.CreateRequest) (Started, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.started = append(g.started, [2]uuid.UUID{r.Sides[0][0].UserID, r.Sides[1][0].UserID})
	g.reqs = append(g.reqs, r)
	return Started{MatchID: ids.New(), ReadyDeadlineMS: 1}, nil
}

func (g *fakeGame) Practice(context.Context, uuid.UUID, string) (Started, error) {
	return Started{}, errors.New("no practice here")
}

func (g *fakeGame) LiveMatch(context.Context, uuid.UUID) (uuid.UUID, bool)  { return uuid.Nil, false }
func (g *fakeGame) CategoryExists(context.Context, uuid.UUID) (bool, error) { return true, nil }

func (g *fakeGame) pairs() [][2]uuid.UUID {
	g.mu.Lock()
	defer g.mu.Unlock()
	return append([][2]uuid.UUID(nil), g.started...)
}

type frames struct {
	mu   sync.Mutex
	sent map[uuid.UUID][]realtime.Frame
}

func (f *frames) Send(_ context.Context, t realtime.Target, fr realtime.Frame) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent[t.User] = append(f.sent[t.User], fr)
	return nil
}

type ratings map[uuid.UUID]Rating

func (r ratings) Rating(_ context.Context, u uuid.UUID, _ string) (Rating, error) { return r[u], nil }

type blocks map[uuid.UUID][]uuid.UUID

func (b blocks) BlockedWith(_ context.Context, u uuid.UUID) ([]uuid.UUID, error) { return b[u], nil }

// AreFriends: everyone is friends in these tests.
func (b blocks) AreFriends(context.Context, uuid.UUID, uuid.UUID) (bool, error) { return true, nil }

type fixture struct {
	t     *testing.T
	svc   *Service
	game  *fakeGame
	clk   *clock.Fake
	rates ratings
	block blocks
	req   Request
}

// newFixture uses a queue of its own (a fresh Category Master category)
// so tests sharing Redis do not meet.
func newFixture(t *testing.T) *fixture {
	rdb, err := redis.Connect(t.Context(), testenv.RedisURL(t), "brainvolt-test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = rdb.Close() })
	cat := ids.New()
	f := &fixture{t: t, game: &fakeGame{}, clk: clock.NewFake(time.Now()), rates: ratings{}, block: blocks{},
		req: Request{MatchType: "quick_duel", Format: "1v1", CategoryMode: "category_master", CategoryID: &cat}}
	f.svc = New(Deps{Redis: rdb, Game: f.game, Notifier: &frames{sent: map[uuid.UUID][]realtime.Frame{}}, Ratings: f.rates,
		Social: f.block, Clock: f.clk, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	return f
}

func (f *fixture) player(rating float64) uuid.UUID {
	u := ids.New()
	f.rates[u] = Rating{Value: rating}
	return u
}

func (f *fixture) enqueue(u uuid.UUID) Ticket {
	f.t.Helper()
	t, err := f.svc.Enqueue(f.t.Context(), u, "tr", f.req)
	if err != nil {
		f.t.Fatal(err)
	}
	return t
}

func (f *fixture) match() { f.svc.match(f.t.Context(), queueName(f.req)) }

func TestSearchRangeWidens(t *testing.T) {
	casual := &entry{Request: Request{MatchType: "quick_duel"}, RD: 350}
	ranked := &entry{Request: Request{MatchType: "ranked"}}
	for _, c := range []struct {
		e    *entry
		wait time.Duration
		want int
	}{
		{ranked, 0, 100}, {ranked, 4 * time.Second, 100}, {ranked, 5 * time.Second, 150}, {ranked, 50 * time.Second, 600},
		{ranked, 10 * time.Minute, 600}, {casual, 0, 170}, {casual, 40 * time.Second, 570}, {casual, 45 * time.Second, unlimitedMS},
	} {
		if got := searchRange(c.e, c.wait); got != c.want {
			t.Errorf("%s after %s: %d, want %d", c.e.Request.MatchType, c.wait, got, c.want)
		}
	}
}

func TestPairsWithinRangeAfterWaiting(t *testing.T) {
	f := newFixture(t)
	a, b := f.player(1500), f.player(1800)
	f.enqueue(a)
	f.enqueue(b)
	f.match()
	if len(f.game.pairs()) != 0 {
		t.Fatal("paired 300 apart at once")
	}
	cur, err := f.svc.Current(t.Context(), a)
	if err != nil || cur.Range == nil || *cur.Range != 100 {
		t.Fatalf("current %+v %v", cur, err)
	}
	f.clk.Advance(30 * time.Second) // ±400
	f.match()
	if p := f.game.pairs(); len(p) != 1 || p[0] != [2]uuid.UUID{a, b} {
		t.Fatalf("pairs %v", p)
	}
	if _, err := f.svc.Current(t.Context(), a); err == nil {
		t.Fatal("ticket left after the match")
	}
}

func TestClosestRatingAndBlocks(t *testing.T) {
	f := newFixture(t)
	a, far, blocked, near := f.player(1500), f.player(1590), f.player(1500), f.player(1520)
	f.block[a] = []uuid.UUID{blocked}
	f.block[blocked] = []uuid.UUID{a}
	for _, u := range []uuid.UUID{a, far, blocked, near} {
		f.enqueue(u)
		f.clk.Advance(time.Millisecond)
	}
	f.match()
	p := f.game.pairs()
	if len(p) != 2 || p[0] != [2]uuid.UUID{a, near} || p[1] != [2]uuid.UUID{far, blocked} {
		t.Fatalf("pairs %v (a=%s far=%s blocked=%s near=%s)", p, a, far, blocked, near)
	}
}

func code(err error) apierror.Code {
	var e *apierror.Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}

func TestReadyCheckRequeueAndCooldowns(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	ready, absent := f.player(1500), f.player(1500)
	first := f.enqueue(ready)
	f.enqueue(absent)
	f.clk.Advance(3 * time.Second)
	f.match()
	if len(f.game.pairs()) != 1 {
		t.Fatal("not paired")
	}
	// The match id is only known to the fake; find the remembered tickets.
	keys, _ := f.svc.d.Redis.Keys(ctx, "mm:m:*").Result()
	var matchID uuid.UUID
	for _, k := range keys {
		if ok, _ := f.svc.d.Redis.HExists(ctx, k, ready.String()).Result(); ok {
			matchID = uuid.MustParse(k[len("mm:m:"):])
		}
	}
	cancelled := func(id uuid.UUID) game.Ended {
		return game.Ended{MatchID: id, Origin: game.OriginMatchmaking, Status: game.StatusCancelled, CancelReason: game.CancelNotAllReady,
			Participants: []game.EndedParticipant{{UserID: ready, Status: game.ParticipantReady}, {UserID: absent, Status: "joined"}}}
	}
	f.svc.OnEnd(ctx, cancelled(matchID))
	cur, err := f.svc.Current(ctx, ready)
	if err != nil || !cur.EnqueuedAt.Equal(first.EnqueuedAt) || cur.ID == first.ID {
		t.Fatalf("ready player not back at the front: %+v %v", cur, err)
	}
	if _, err := f.svc.Current(ctx, absent); code(err) != apierror.NotFound {
		t.Fatalf("absent player queued again: %v", err)
	}
	// Three missed ready checks in 30 minutes → 5 minute cooldown.
	f.svc.OnEnd(ctx, cancelled(ids.New()))
	if _, err := f.svc.Enqueue(ctx, absent, "tr", f.req); err != nil {
		t.Fatalf("cooldown after 2 misses: %v", err)
	}
	_ = f.svc.Leave(ctx, absent)
	f.svc.OnEnd(ctx, cancelled(ids.New()))
	if _, err := f.svc.Enqueue(ctx, absent, "tr", f.req); code(err) != apierror.QueueCooldown {
		t.Fatalf("3 misses: %v", err)
	}

	// Leaving: the 2nd leave in 24 h → 5 min, the 3rd → 15 min.
	leaver := f.player(1500)
	left := game.Ended{MatchID: ids.New(), Origin: game.OriginMatchmaking, Status: game.StatusCompleted,
		Participants: []game.EndedParticipant{{UserID: leaver, Status: game.ParticipantLeft}}}
	f.svc.OnEnd(ctx, left)
	if ttl, _ := f.svc.d.Redis.PTTL(ctx, cooldownKey(leaver)).Result(); ttl > 0 {
		t.Fatal("cooldown after the first leave")
	}
	f.svc.OnEnd(ctx, left)
	if ttl, _ := f.svc.d.Redis.PTTL(ctx, cooldownKey(leaver)).Result(); ttl <= 4*time.Minute || ttl > SecondLeaveCooldown {
		t.Fatalf("2nd leave cooldown %s", ttl)
	}
	f.svc.OnEnd(ctx, left)
	if ttl, _ := f.svc.d.Redis.PTTL(ctx, cooldownKey(leaver)).Result(); ttl <= 14*time.Minute {
		t.Fatalf("3rd leave cooldown %s", ttl)
	}
	// Other origins do not count.
	other := ids.New()
	for range 3 {
		f.svc.OnEnd(ctx, game.Ended{MatchID: ids.New(), Origin: game.OriginChallenge, Status: game.StatusCompleted,
			Participants: []game.EndedParticipant{{UserID: other, Status: game.ParticipantLeft}}})
	}
	if ttl, _ := f.svc.d.Redis.PTTL(ctx, cooldownKey(other)).Result(); ttl > 0 {
		t.Fatal("challenge leaves count")
	}
}

func sideUsers(ms []game.Member) []uuid.UUID {
	out := make([]uuid.UUID, len(ms))
	for i, m := range ms {
		out[i] = m.UserID
	}
	return out
}

// Team formats: a party of two and two solos make a 2v2; the party stays
// on one side, a leaver takes the party out of the queue.
func TestPartiesFillTemporarySides(t *testing.T) {
	f := newFixture(t)
	f.req.Format = "2v2" // still a Category Master queue of its own
	ctx := t.Context()
	lead, mate, s1, s2 := f.player(1500), f.player(1500), f.player(1510), f.player(1490)
	party := f.req
	party.PartyUserIDs = []uuid.UUID{mate}
	if _, err := f.svc.Enqueue(ctx, lead, "tr", party); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Enqueue(ctx, mate, "tr", f.req); code(err) != apierror.AlreadyQueued {
		t.Fatalf("party member queued twice: %v", err)
	}
	f.clk.Advance(time.Millisecond)
	f.enqueue(s1)
	f.match()
	if len(f.game.pairs()) != 0 {
		t.Fatal("3 players made a 2v2")
	}
	f.enqueue(s2)
	f.match()
	reqs := f.game.reqs
	if len(reqs) != 1 {
		t.Fatalf("%d matches", len(reqs))
	}
	r := reqs[0]
	if got := sideUsers(r.Sides[0]); len(got) != 2 || got[0] != lead || got[1] != mate {
		t.Fatalf("party side %v", got)
	}
	if got := sideUsers(r.Sides[1]); len(got) != 2 || !slices.Contains(got, s1) || !slices.Contains(got, s2) {
		t.Fatalf("solo side %v", got)
	}
	if r.Teams[0] != nil || r.Teams[1] != nil {
		t.Fatal("temporary sides got a team")
	}

	// Leaving with a party drops the whole ticket.
	a, b := f.player(1500), f.player(1500)
	party.PartyUserIDs = []uuid.UUID{b}
	if _, err := f.svc.Enqueue(ctx, a, "tr", party); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.Leave(ctx, b); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Current(ctx, a); code(err) != apierror.NotFound {
		t.Fatalf("leader still queued: %v", err)
	}
	if _, err := f.svc.Enqueue(ctx, a, "tr", f.req); err != nil {
		t.Fatalf("queue again: %v", err)
	}
}

// Requests that cannot form a side are refused.
func TestPartyRules(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	u, v := f.player(1500), f.player(1500)
	duo := f.req
	duo.PartyUserIDs = []uuid.UUID{v}
	if _, err := f.svc.Enqueue(ctx, u, "tr", duo); code(err) != apierror.ValidationFailed {
		t.Fatalf("party in 1v1: %v", err)
	}
	f.req.Format = "2v2"
	big := f.req
	big.PartyUserIDs = []uuid.UUID{v, f.player(1500)}
	if _, err := f.svc.Enqueue(ctx, u, "tr", big); code(err) != apierror.ValidationFailed {
		t.Fatalf("party of 3 in 2v2: %v", err)
	}
	f.block[u] = []uuid.UUID{v}
	if _, err := f.svc.Enqueue(ctx, u, "tr", duo); code(err) != apierror.ValidationFailed {
		t.Fatalf("1v1 party: %v", err)
	}
	two := f.req
	two.PartyUserIDs = []uuid.UUID{v}
	if _, err := f.svc.Enqueue(ctx, u, "tr", two); code(err) != apierror.Blocked {
		t.Fatalf("blocked party: %v", err)
	}
}
