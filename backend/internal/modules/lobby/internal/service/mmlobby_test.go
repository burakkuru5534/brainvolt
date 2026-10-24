package service

import (
	"context"
	"io"
	"log/slog"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/realtime"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
)

type fakeCards struct{ Profiles }

func (fakeCards) Cards(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]profile.Card, error) {
	out := map[uuid.UUID]profile.Card{}
	for _, id := range ids {
		out[id] = profile.Card{}
	}
	return out, nil
}

type fakeRT struct {
	mu     sync.Mutex
	frames map[uuid.UUID][]realtime.Frame
}

func (f *fakeRT) Send(_ context.Context, t realtime.Target, fr realtime.Frame) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.frames[t.User] = append(f.frames[t.User], fr)
	return nil
}

func (f *fakeRT) last(u uuid.UUID, typ string) (realtime.Frame, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	fs := f.frames[u]
	for i := len(fs) - 1; i >= 0; i-- {
		if fs[i].T == typ {
			return fs[i], true
		}
	}
	return realtime.Frame{}, false
}

type fakeGame struct {
	Game
	started []game.CreateRequest
}

func (g *fakeGame) Start(_ context.Context, r game.CreateRequest) (*game.Created, error) {
	g.started = append(g.started, r)
	// engine is internal to game: build its Match through reflection.
	c := &game.Created{}
	m := reflect.New(reflect.TypeOf(c.Match).Elem())
	m.Elem().FieldByName("ID").Set(reflect.ValueOf(uuid.New()))
	reflect.ValueOf(c).Elem().FieldByName("Match").Set(m)
	return c, nil
}

func newLobbyService(t *testing.T) (*Service, *fakeRT, *fakeGame, *clock.Fake) {
	t.Helper()
	rt := &fakeRT{frames: map[uuid.UUID][]realtime.Frame{}}
	g := &fakeGame{}
	clk := clock.NewFake(time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC))
	s := New(Deps{Profiles: fakeCards{}, RT: rt, Game: g, Clock: clk, Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Random: func(n int) int { return n - 1 }})
	return s, rt, g, clk
}

func group(queue string, n int, team *uuid.UUID) MatchmakingGroup {
	g := MatchmakingGroup{Queue: queue, TicketID: uuid.New(), TeamID: team,
		Config: game.Config{MatchType: "quick_duel", Format: "2v2", Origin: game.OriginMatchmaking}}
	for range n {
		g.Members = append(g.Members, game.Member{UserID: uuid.New(), Locale: "tr"})
	}
	return g
}

// D-094: seats fill across tickets; a full lobby draws a leader; only the
// leader starts, and only once everyone is ready; the lobby then closes.
func TestMatchmakingLobbyFillsReadiesAndStarts(t *testing.T) {
	ctx := t.Context()
	s, rt, g, _ := newLobbyService(t)
	team := uuid.New()
	party := group("q", 2, &team)
	solo1, solo2, other := group("q", 1, nil), group("q", 1, nil), group("other", 1, nil)

	room, err := s.SeatMatchmaking(ctx, party)
	if err != nil {
		t.Fatal(err)
	}
	if id, _ := s.SeatMatchmaking(ctx, other); id == room {
		t.Fatal("another queue joined the lobby")
	}
	if id, _ := s.SeatMatchmaking(ctx, solo1); id != room {
		t.Fatal("solo player not seated in the open lobby")
	}
	r, _ := s.load(ctx, room)
	if r.Leader != nil || len(r.Members) != 3 {
		t.Fatalf("not full yet: leader %v, %d members", r.Leader, len(r.Members))
	}
	for _, m := range r.Members {
		if m.Group == party.TicketID && m.Side != 1 {
			t.Fatalf("party split: %+v", r.Members)
		}
	}
	if id, _ := s.SeatMatchmaking(ctx, solo2); id != room {
		t.Fatal("last seat not filled")
	}
	r, _ = s.load(ctx, room)
	if r.Leader == nil || r.FullSinceMS == 0 {
		t.Fatalf("full lobby without a leader: %+v", r)
	}
	leader := *r.Leader
	var notLeader uuid.UUID
	for _, m := range r.Members {
		if m.UserID != leader {
			notLeader = m.UserID
		}
	}
	if _, ok := rt.last(leader, "room.state"); !ok {
		t.Fatal("members get room.state")
	}
	if _, err := s.StartRoom(ctx, leader, room); !isCode(err, "MATCH_STATE_INVALID") {
		t.Fatalf("start before everyone is ready: %v", err)
	}
	for _, m := range r.Members {
		if err := s.SetReady(ctx, m.UserID, room, true); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.StartRoom(ctx, notLeader, room); !isCode(err, "FORBIDDEN") {
		t.Fatalf("start by a non-leader: %v", err)
	}
	if _, err := s.StartRoom(ctx, leader, room); err != nil {
		t.Fatal(err)
	}
	if len(g.started) != 1 || g.started[0].Teams[0] == nil || *g.started[0].Teams[0] != team || g.started[0].Teams[1] != nil {
		t.Fatalf("match request %+v", g.started)
	}
	if g.started[0].Config.Origin != game.OriginMatchmaking || g.started[0].OriginRef != nil {
		t.Fatalf("origin %+v", g.started[0].Config)
	}
	if f, ok := rt.last(notLeader, "room.closed"); !ok || f.D.(map[string]any)["reason"] != CloseMatchStarted {
		t.Fatalf("lobby not closed after the start: %+v", f)
	}
	if _, ok := s.CurrentRoom(ctx, leader); ok {
		t.Fatal("players still in the lobby")
	}
}

// D-094 timers: unready players are dropped after the ready window; an
// idle leader's lobby starts on its own; a leaver frees the seat and the
// leader is re-drawn when the lobby fills again.
func TestMatchmakingLobbyTimers(t *testing.T) {
	ctx := t.Context()
	s, rt, g, clk := newLobbyService(t)
	gs := []MatchmakingGroup{group("q", 1, nil), group("q", 1, nil), group("q", 1, nil), group("q", 1, nil)}
	var room uuid.UUID
	for _, gr := range gs {
		room, _ = s.SeatMatchmaking(ctx, gr)
	}
	lazy := gs[3].Members[0].UserID
	for _, gr := range gs[:3] {
		if err := s.SetReady(ctx, gr.Members[0].UserID, room, true); err != nil {
			t.Fatal(err)
		}
	}
	clk.Advance(LobbyReadyWindow)
	s.TickLobbies(ctx)
	if f, ok := rt.last(lazy, "room.closed"); !ok || f.D.(map[string]any)["reason"] != CloseNotReady {
		t.Fatalf("unready player not dropped: %+v", f)
	}
	r, _ := s.load(ctx, room)
	if len(r.Members) != 3 || r.Leader != nil {
		t.Fatalf("after the drop: %d members, leader %v", len(r.Members), r.Leader)
	}
	// A new player fills the seat and readies; nobody presses start.
	late := group("q", 1, nil)
	if id, _ := s.SeatMatchmaking(ctx, late); id != room {
		t.Fatal("freed seat not offered")
	}
	if err := s.SetReady(ctx, late.Members[0].UserID, room, true); err != nil {
		t.Fatal(err)
	}
	clk.Advance(LobbyLeaderWindow)
	s.TickLobbies(ctx)
	if len(g.started) != 1 {
		t.Fatalf("idle leader: %d matches started", len(g.started))
	}

	// A lobby that never fills expires.
	alone := group("q", 1, nil)
	solo, _ := s.SeatMatchmaking(ctx, alone)
	if err := s.LeaveRoom(ctx, alone.Members[0].UserID, solo); err != nil {
		t.Fatal(err)
	}
	if _, err := s.load(ctx, solo); !isCode(err, "ROOM_CLOSED") {
		t.Fatalf("empty lobby still open: %v", err)
	}
	waiting := group("q", 1, nil)
	solo, _ = s.SeatMatchmaking(ctx, waiting)
	clk.Advance(LobbyMaxAge)
	s.TickLobbies(ctx)
	if f, ok := rt.last(waiting.Members[0].UserID, "room.closed"); !ok || f.D.(map[string]any)["reason"] != CloseExpired {
		t.Fatalf("old lobby not closed: %+v", f)
	}
}
