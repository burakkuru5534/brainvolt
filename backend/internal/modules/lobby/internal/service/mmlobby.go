package service

import (
	"context"
	"errors"
	"math"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
)

// Matchmaking lobbies (D-094): a matchmade team match (2v2–4v4, couple)
// gathers its players in a waiting lobby before game creates the match.
// The matcher seats every ticket into an open lobby of its queue — a
// party, team or couple sits together on one side — so players watch the
// seats fill; once every seat is taken a leader is drawn at random, every
// player readies up and the leader starts the match.
const (
	KindMatchmaking = "matchmaking"

	// LobbyTick drives the lobby timers.
	LobbyTick = time.Second
	// LobbyReadyWindow: a full lobby whose players are not all ready
	// after this drops the players who are not, and waits for others.
	LobbyReadyWindow = 60 * time.Second
	// LobbyLeaderWindow: everyone is ready but the leader has not
	// started; the match starts on its own.
	LobbyLeaderWindow = 20 * time.Second
	// LobbyMaxAge closes a lobby that never filled (the ticket TTL).
	LobbyMaxAge = 10 * time.Minute
	// LobbyRatingRange is how far a Ranked group's rating may be from the
	// lobby's average (the widest matchmaking range, D-068).
	LobbyRatingRange = 600
)

// Close reasons of matchmaking lobbies.
const (
	CloseMatchStarted = "match_started"
	CloseNotReady     = "not_ready"
	CloseExpired      = "lobby_expired"
)

func errLobbyFixed() *apierror.Error {
	return apierror.New(apierror.MatchStateInvalid).With("reason", "matchmaking_lobby")
}

// MatchmakingGroup is one matchmaking ticket to seat (D-094).
type MatchmakingGroup struct {
	Queue    string
	Config   game.Config
	TicketID uuid.UUID
	Members  []game.Member
	TeamID   *uuid.UUID
	CoupleID *uuid.UUID
	Rating   float64
	Ranked   bool
	Blocks   []uuid.UUID
}

func (g *MatchmakingGroup) persistent() bool { return g.TeamID != nil || g.CoupleID != nil }

// starter reports whether user may start the room: the host of a private
// room, the leader of a matchmaking lobby.
func (r *room) starter(user uuid.UUID) bool {
	if r.Kind == KindMatchmaking {
		return r.Leader != nil && *r.Leader == user
	}
	return r.Host == user
}

// track keeps a matchmaking lobby's leader and timers in step with its
// seats and ready flags: a full lobby gets a random leader (kept while it
// stays), a lobby that is no longer full loses it.
func (r *room) track(now int64, pick func(int) int) {
	if len(r.Members) < r.capacity() {
		r.Leader, r.FullSinceMS, r.AllReadySinceMS = nil, 0, 0
		return
	}
	if r.Leader == nil || r.member(*r.Leader) == nil {
		id := r.Members[pick(len(r.Members))].UserID
		r.Leader = &id
	}
	if r.FullSinceMS == 0 {
		r.FullSinceMS = now
	}
	all := !slices.ContainsFunc(r.Members, func(m member) bool { return !m.Ready })
	switch {
	case !all:
		r.AllReadySinceMS = 0
	case r.AllReadySinceMS == 0:
		r.AllReadySinceMS = now
	}
}

// persistentSides are the team and couple of each side when one
// persistent ticket fills it.
func (r *room) persistentSides() (teams, couples [2]*uuid.UUID) {
	for i := range 2 {
		var side []member
		for _, m := range r.Members {
			if m.Side == i+1 {
				side = append(side, m)
			}
		}
		if len(side) == 0 || (side[0].TeamID == nil && side[0].CoupleID == nil) {
			continue
		}
		if slices.ContainsFunc(side, func(m member) bool { return m.Group != side[0].Group }) {
			continue
		}
		teams[i], couples[i] = side[0].TeamID, side[0].CoupleID
	}
	return teams, couples
}

func (r *room) sideCount(side int) int {
	n := 0
	for _, m := range r.Members {
		if m.Side == side {
			n++
		}
	}
	return n
}

// seatFor is the side a group fits on (the fuller side first, to complete
// sides), or 0.
func (r *room) seatFor(g *MatchmakingGroup) int {
	if r.Kind != KindMatchmaking || r.Queue != g.Queue || r.Status != RoomOpen {
		return 0
	}
	if r.Ranked && len(r.Members) > 0 {
		sum := 0.0
		for _, m := range r.Members {
			sum += m.Rating
		}
		if math.Abs(sum/float64(len(r.Members))-g.Rating) > LobbyRatingRange {
			return 0
		}
	}
	for _, m := range r.Members {
		if slices.Contains(g.Blocks, m.UserID) {
			return 0
		}
		for _, gm := range g.Members {
			if slices.Contains(m.Blocks, gm.UserID) || m.UserID == gm.UserID {
				return 0
			}
		}
	}
	size := r.Config.Format.SideSize()
	sides := []int{1, 2}
	if r.sideCount(2) > r.sideCount(1) {
		sides = []int{2, 1}
	}
	for _, side := range sides {
		used := r.sideCount(side)
		if g.persistent() && used > 0 {
			continue
		}
		if used+len(g.Members) <= size {
			return side
		}
	}
	return 0
}

// add seats a group on side, in that side's first free seats.
func (r *room) add(g *MatchmakingGroup, side int) {
	size := r.Config.Format.SideSize()
	for _, gm := range g.Members {
		for seat := 1; seat <= size; seat++ {
			if !r.seatTaken(side, seat) {
				r.Members = append(r.Members, member{
					UserID: gm.UserID, Side: side, Seat: seat, Locale: gm.Locale, Group: g.TicketID,
					TeamID: g.TeamID, CoupleID: g.CoupleID, Rating: g.Rating, Blocks: g.Blocks,
				})
				break
			}
		}
	}
}

// SeatMatchmaking puts a matchmaking ticket into the fullest open lobby
// of its queue that it fits — or a new one — and returns the lobby; every
// member gets room.state. A room a member was in is left first.
func (s *Service) SeatMatchmaking(ctx context.Context, g MatchmakingGroup) (uuid.UUID, error) {
	if len(g.Members) == 0 || len(g.Members) > g.Config.Format.SideSize() {
		return uuid.Nil, apierror.New(apierror.MatchStateInvalid).With("reason", "group_size")
	}
	for _, m := range g.Members {
		if err := s.leaveCurrent(ctx, m.UserID); err != nil {
			return uuid.Nil, err
		}
	}
	for range updateTries {
		id := s.bestLobby(&g)
		if id == uuid.Nil {
			return s.newLobby(ctx, &g)
		}
		_, err := s.update(ctx, id, func(r *room) error {
			side := r.seatFor(&g)
			if side == 0 {
				return errLobbyGone
			}
			r.add(&g, side)
			r.track(s.d.Clock.Now().UnixMilli(), s.d.Random)
			return nil
		})
		switch {
		case err == nil:
			return id, nil
		case errors.Is(err, errLobbyGone) || isCode(err, apierror.RoomClosed):
			continue // someone else took the seats: look again
		default:
			return uuid.Nil, err
		}
	}
	return s.newLobby(ctx, &g)
}

var errLobbyGone = errors.New("lobby no longer fits")

// bestLobby is the fullest open lobby the group fits (the oldest on ties).
func (s *Service) bestLobby(g *MatchmakingGroup) uuid.UUID {
	s.roomsMu.RLock()
	defer s.roomsMu.RUnlock()
	var best *room
	for _, r := range s.rooms {
		if r.seatFor(g) == 0 {
			continue
		}
		if best == nil || len(r.Members) > len(best.Members) ||
			(len(r.Members) == len(best.Members) && r.CreatedMS < best.CreatedMS) {
			best = r
		}
	}
	if best == nil {
		return uuid.Nil
	}
	return best.ID
}

func (s *Service) newLobby(ctx context.Context, g *MatchmakingGroup) (uuid.UUID, error) {
	cfg := g.Config
	cfg.Origin = game.OriginMatchmaking
	now := s.d.Clock.Now().UnixMilli()
	r := &room{ID: ids.New(), Status: RoomOpen, Config: cfg, Version: 1, Kind: KindMatchmaking, Queue: g.Queue,
		Ranked: g.Ranked, CreatedMS: now}
	r.add(g, 1)
	r.track(now, s.d.Random)
	s.roomsMu.Lock()
	s.rooms[r.ID] = r
	for _, m := range r.Members {
		s.userRooms[m.UserID] = r.ID
	}
	out := *r
	out.Members = slices.Clone(r.Members)
	s.roomsMu.Unlock()
	s.broadcast(ctx, &out, nil)
	return r.ID, nil
}

// TickLobbies runs the matchmaking lobby timers (every LobbyTick): a full
// lobby drops the players still not ready after LobbyReadyWindow; a ready
// lobby whose leader waits LobbyLeaderWindow starts on its own; a lobby
// that never filled closes after LobbyMaxAge.
func (s *Service) TickLobbies(ctx context.Context) {
	now := s.d.Clock.Now().UnixMilli()
	type due struct {
		id      uuid.UUID
		leader  uuid.UUID
		version int64
		action  int
	}
	const (
		dropUnready = iota + 1
		autoStart
		expire
	)
	var todo []due
	s.roomsMu.RLock()
	for _, r := range s.rooms {
		if r.Kind != KindMatchmaking || r.Status != RoomOpen {
			continue
		}
		switch {
		case r.AllReadySinceMS > 0 && r.Leader != nil && now-r.AllReadySinceMS >= LobbyLeaderWindow.Milliseconds():
			todo = append(todo, due{id: r.ID, leader: *r.Leader, action: autoStart})
		case r.FullSinceMS > 0 && r.AllReadySinceMS == 0 && now-r.FullSinceMS >= LobbyReadyWindow.Milliseconds():
			todo = append(todo, due{id: r.ID, version: r.Version, action: dropUnready})
		case r.FullSinceMS == 0 && now-r.CreatedMS >= LobbyMaxAge.Milliseconds():
			todo = append(todo, due{id: r.ID, action: expire})
		}
	}
	s.roomsMu.RUnlock()

	for _, d := range todo {
		var err error
		switch d.action {
		case autoStart:
			_, err = s.StartRoom(ctx, d.leader, d.id)
		case expire:
			err = s.closeRoom(ctx, d.id, CloseExpired)
		case dropUnready:
			err = s.dropUnready(ctx, d.id)
		}
		if err != nil && !isCode(err, apierror.RoomClosed) {
			s.d.Logger.WarnContext(ctx, "lobby: matchmaking lobby timer failed", "room_id", d.id, "error", err)
		}
	}
}

// dropUnready removes the players who are not ready from a full lobby;
// each gets room.closed with reason not_ready, the rest wait for others.
func (s *Service) dropUnready(ctx context.Context, id uuid.UUID) error {
	var dropped []uuid.UUID
	r, err := s.update(ctx, id, func(r *room) error {
		dropped = dropped[:0]
		if r.FullSinceMS == 0 || r.AllReadySinceMS != 0 {
			return errNoChange
		}
		for _, m := range r.Members {
			if !m.Ready {
				dropped = append(dropped, m.UserID)
			}
		}
		r.Members = slices.DeleteFunc(r.Members, func(m member) bool { return !m.Ready })
		r.track(s.d.Clock.Now().UnixMilli(), s.d.Random)
		return nil
	})
	if err != nil {
		return err
	}
	s.roomsMu.Lock()
	for _, u := range dropped {
		if s.userRooms[u] == id {
			delete(s.userRooms, u)
		}
	}
	s.roomsMu.Unlock()
	for _, u := range dropped {
		s.sendClosed(ctx, u, id, r.Version, CloseNotReady)
	}
	if len(r.Members) == 0 {
		return s.closeRoom(ctx, id, CloseAllLeft)
	}
	return nil
}
