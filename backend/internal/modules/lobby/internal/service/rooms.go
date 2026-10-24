package service

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/lobby/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/realtime"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
)

// Room settings (D-027).
const (
	// RoomIdle closes a room without activity (the sweep runs every minute).
	RoomIdle   = 15 * time.Minute
	SweepEvery = time.Minute
	// roomTTL keeps live state a while past the idle limit; the sweep, not
	// the TTL, closes rooms.
	roomTTL     = time.Hour
	roomCodeTTL = 24 * time.Hour
	updateTries = 10
)

// Room statuses.
const (
	RoomOpen     = "open"
	RoomStarting = "starting"
	RoomInMatch  = "in_match"
)

// Close reasons.
const (
	CloseHost    = "host_closed"
	CloseIdle    = "idle"
	CloseAllLeft = "all_left"
)

// member is one room member in the live state.
type member struct {
	UserID uuid.UUID `json:"user_id"`
	Side   int       `json:"side_no"`
	Seat   int       `json:"seat_no"`
	Ready  bool      `json:"ready"`
	Locale string    `json:"locale"`
}

// room is the live state of a private room (redis-keys §5), versioned for
// optimistic concurrency: every change increments Version, which is also
// the room stream's seq.
type room struct {
	ID            uuid.UUID   `json:"id"`
	Host          uuid.UUID   `json:"host_user_id"`
	Status        string      `json:"status"`
	Config        game.Config `json:"match_config"`
	Version       int64       `json:"version"`
	FriendsOnly   bool        `json:"friends_of_host_only"`
	Code          string      `json:"code"`
	MatchID       *uuid.UUID  `json:"match_id,omitempty"`
	MatchesPlayed int         `json:"matches_played"`
	Members       []member    `json:"members"`
}

func (r *room) capacity() int { return 2 * r.Config.Format.SideSize() }

func (r *room) member(user uuid.UUID) *member {
	for i := range r.Members {
		if r.Members[i].UserID == user {
			return &r.Members[i]
		}
	}
	return nil
}

func (r *room) seatTaken(side, seat int) bool {
	return slices.ContainsFunc(r.Members, func(m member) bool { return m.Side == side && m.Seat == seat })
}

// freeSeat is the first empty seat, side 1 first.
func (r *room) freeSeat() (side, seat int, ok bool) {
	size := r.Config.Format.SideSize()
	for side := 1; side <= 2; side++ {
		for seat := 1; seat <= size; seat++ {
			if !r.seatTaken(side, seat) {
				return side, seat, true
			}
		}
	}
	return 0, 0, false
}

func (r *room) unreadyAll() {
	for i := range r.Members {
		r.Members[i].Ready = false
	}
}

func roomStream(id uuid.UUID) string { return "room:" + id.String() }
func errRoomClosed() *apierror.Error { return apierror.New(apierror.RoomClosed) }
func errNotHost() *apierror.Error    { return apierror.New(apierror.Forbidden).With("reason", "not_host") }
func errRoomBusy(st string) *apierror.Error {
	return apierror.New(apierror.MatchStateInvalid).With("reason", "room_"+st)
}

// RoomMember is a member as the app shows it.
type RoomMember struct {
	profile.Card
	SideNo int  `json:"side_no"`
	SeatNo int  `json:"seat_no"`
	Ready  bool `json:"ready"`
	IsHost bool `json:"is_host"`
}

// Room is a room as its members see it (GET /v1/rooms/{id}, room.state).
type Room struct {
	ID            uuid.UUID    `json:"id"`
	Status        string       `json:"status"`
	Version       int64        `json:"version"`
	HostUserID    uuid.UUID    `json:"host_user_id"`
	Code          string       `json:"code"`
	MatchConfig   game.Config  `json:"match_config"`
	FriendsOnly   bool         `json:"friends_of_host_only"`
	MatchID       *uuid.UUID   `json:"match_id,omitempty"`
	MatchesPlayed int          `json:"matches_played"`
	Capacity      int          `json:"capacity"`
	Members       []RoomMember `json:"members"`
}

// RoomPreview is what an invite code shows before joining.
type RoomPreview struct {
	ID          uuid.UUID    `json:"id"`
	Status      string       `json:"status"`
	Host        profile.Card `json:"host"`
	MatchConfig game.Config  `json:"match_config"`
	Members     int          `json:"member_count"`
	Capacity    int          `json:"capacity"`
}

func (s *Service) render(ctx context.Context, r *room) (Room, error) {
	users := make([]uuid.UUID, len(r.Members))
	for i, m := range r.Members {
		users[i] = m.UserID
	}
	cards, err := s.d.Profiles.Cards(ctx, users)
	if err != nil {
		return Room{}, err
	}
	out := Room{
		ID: r.ID, Status: r.Status, Version: r.Version, HostUserID: r.Host, Code: r.Code, MatchConfig: r.Config, FriendsOnly: r.FriendsOnly,
		MatchID: r.MatchID, MatchesPlayed: r.MatchesPlayed, Capacity: r.capacity(), Members: make([]RoomMember, 0, len(r.Members)),
	}
	for _, m := range r.Members {
		out.Members = append(out.Members, RoomMember{Card: cards[m.UserID], SideNo: m.Side, SeatNo: m.Seat, Ready: m.Ready, IsHost: m.UserID == r.Host})
	}
	slices.SortFunc(out.Members, func(a, b RoomMember) int { return (a.SideNo*10 + a.SeatNo) - (b.SideNo*10 + b.SeatNo) })
	return out, nil
}

// load reads a room's live state; a missing room is closed.
func (s *Service) load(ctx context.Context, id uuid.UUID) (*room, error) {
	s.roomsMu.RLock()
	defer s.roomsMu.RUnlock()
	r, ok := s.rooms[id]
	if !ok {
		return nil, errRoomClosed()
	}
	rcopy := *r
	rcopy.Members = slices.Clone(r.Members)
	return &rcopy, nil
}

// errNoChange ends an update without writing.
var errNoChange = errors.New("no change")

// update applies fn to the room atomically, mirrors it to Postgres and sends room.state to the members.
// fn returning errNoChange leaves the room as it is.
func (s *Service) update(ctx context.Context, id uuid.UUID, fn func(r *room) error) (*room, error) {
	s.roomsMu.Lock()
	r, ok := s.rooms[id]
	if !ok {
		s.roomsMu.Unlock()
		return nil, errRoomClosed()
	}
	before := make([]uuid.UUID, len(r.Members))
	for i, m := range r.Members {
		before[i] = m.UserID
	}
	if err := fn(r); err != nil {
		s.roomsMu.Unlock()
		if errors.Is(err, errNoChange) {
			return s.load(ctx, id)
		}
		return nil, err
	}
	r.Version++
	for _, m := range r.Members {
		s.userRooms[m.UserID] = r.ID
	}
	for _, u := range before {
		if r.member(u) == nil {
			delete(s.userRooms, u)
		}
	}
	out := *r
	out.Members = slices.Clone(r.Members)
	s.roomsMu.Unlock()

	s.persist(ctx, &out)
	s.broadcast(ctx, &out, before)
	return &out, nil
}

// persist mirrors the durable part of a room to lobby.rooms.
func (s *Service) persist(ctx context.Context, r *room) {
	cfg, err := json.Marshal(r.Config)
	if err == nil {
		err = sqlc.New(s.d.DB).UpdateRoom(ctx, sqlc.UpdateRoomParams{
			ID: r.ID, HostUserID: r.Host, Status: r.Status, MatchConfig: cfg, CurrentMatchID: r.MatchID,
			CreatorQuizID: r.Config.CreatorQuizID, FriendsOfHostOnly: r.FriendsOnly,
			MatchesPlayed: int32(r.MatchesPlayed), Now: s.d.Clock.Now(), //nolint:gosec // G115: a handful of matches per room
		})
	}
	if err != nil {
		s.d.Logger.WarnContext(ctx, "lobby: saving a room failed", "room_id", r.ID, "error", err)
	}
}

// broadcast sends room.state to every member and drops the user keys of
// members who left.
func (s *Service) broadcast(ctx context.Context, r *room, before []uuid.UUID) {
	v, err := s.render(ctx, r)
	if err != nil {
		s.d.Logger.WarnContext(ctx, "lobby: rendering a room failed", "room_id", r.ID, "error", err)
		return
	}
	f := realtime.Frame{T: "room.state", S: roomStream(r.ID), Seq: r.Version, TS: s.d.Clock.Now().UnixMilli(), D: map[string]any{"room": v}}
	for _, m := range r.Members {
		s.send(ctx, m.UserID, f)
	}
}

// refresh returns a room to `open` once its match is over (rooms host
// several matches in a row).
func (s *Service) refresh(ctx context.Context, id uuid.UUID) (*room, error) {
	r, err := s.load(ctx, id)
	if err != nil || r.Status != RoomInMatch || r.MatchID == nil {
		return r, err
	}
	sum, err := s.d.Game.Summary(ctx, *r.MatchID)
	if err != nil {
		return nil, err
	}
	if sum.Status != "completed" && sum.Status != "cancelled" {
		return r, nil
	}
	return s.update(ctx, id, func(r *room) error {
		if r.Status != RoomInMatch || r.MatchID == nil || *r.MatchID != sum.ID {
			return errNoChange
		}
		r.Status, r.MatchID = RoomOpen, nil
		if sum.Status == "completed" {
			r.MatchesPlayed++
		}
		r.unreadyAll()
		return nil
	})
}

// CreateRoom opens a private room with the caller as host on side 1 and
// returns it with its invite code. A room the caller was in is left.
func (s *Service) CreateRoom(ctx context.Context, user uuid.UUID, locale string, cfg game.Config) (Room, error) {
	cfg.Origin = game.OriginPrivateRoom
	quiz, err := s.withQuiz(ctx, user, &cfg)
	if err != nil {
		return Room{}, err
	}
	if err := checkConfig(cfg); err != nil {
		return Room{}, err
	}
	if err := s.leaveCurrent(ctx, user); err != nil {
		return Room{}, err
	}
	now := s.d.Clock.Now().UTC()
	body, err := json.Marshal(cfg)
	if err != nil {
		return Room{}, err
	}
	r := &room{ID: ids.New(), Host: user, Status: RoomOpen, Config: cfg, Version: 1, FriendsOnly: friendsOnly(quiz),
		Members: []member{{UserID: user, Side: 1, Seat: 1, Locale: locale}}}
	err = postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		if err := q.InsertRoom(ctx, sqlc.InsertRoomParams{
			ID: r.ID, HostUserID: user, MatchConfig: body, CreatorQuizID: cfg.CreatorQuizID, FriendsOfHostOnly: r.FriendsOnly, Now: now,
		}); err != nil {
			return err
		}
		r.Code, err = s.newCode(ctx, q, KindRoom, r.ID, user, now.Add(roomCodeTTL), now)
		return err
	})
	if err != nil {
		return Room{}, err
	}
	s.roomsMu.Lock()
	s.rooms[r.ID] = r
	s.userRooms[user] = r.ID
	s.roomsMu.Unlock()
	return s.render(ctx, r)
}

// GetRoom returns a room to one of its members.
func (s *Service) GetRoom(ctx context.Context, user, id uuid.UUID) (Room, error) {
	r, err := s.refresh(ctx, id)
	if err != nil {
		return Room{}, err
	}
	if r.member(user) == nil {
		return Room{}, apierror.New(apierror.NotFound)
	}
	return s.render(ctx, r)
}

// CurrentRoom is the caller's room id, if any.
func (s *Service) CurrentRoom(ctx context.Context, user uuid.UUID) (uuid.UUID, bool) {
	s.roomsMu.RLock()
	defer s.roomsMu.RUnlock()
	id, ok := s.userRooms[user]
	return id, ok
}

// UpdateRoomConfig changes the match configuration (host, room open);
// version, when given, must be the current one. Ready flags reset.
func (s *Service) UpdateRoomConfig(ctx context.Context, user, id uuid.UUID, cfg game.Config, version *int64) (Room, error) {
	// A finished match returns the room to open first.
	if _, err := s.refresh(ctx, id); err != nil {
		return Room{}, err
	}
	cfg.Origin = game.OriginPrivateRoom
	quiz, err := s.withQuiz(ctx, user, &cfg)
	if err != nil {
		return Room{}, err
	}
	if err := checkConfig(cfg); err != nil {
		return Room{}, err
	}
	if friendsOnly(quiz) {
		if err := s.allFriendsOfHost(ctx, id, user); err != nil {
			return Room{}, err
		}
	}
	r, err := s.update(ctx, id, func(r *room) error {
		if r.member(user) == nil {
			return apierror.New(apierror.NotFound)
		}
		if r.Host != user {
			return errNotHost()
		}
		if version != nil && *version != r.Version {
			return apierror.New(apierror.VersionConflict).With("version", r.Version)
		}
		if r.Status != RoomOpen {
			return errRoomBusy(r.Status)
		}
		if 2*cfg.Format.SideSize() < len(r.Members) {
			return apierror.New(apierror.RoomFull)
		}
		r.Config, r.FriendsOnly = cfg, friendsOnly(quiz)
		r.unreadyAll()
		return nil
	})
	if err != nil {
		return Room{}, err
	}
	return s.render(ctx, r)
}

// friendsOnly reports whether a room playing quiz is limited to the
// host's friends: an unpublished quiz (D-023, R14).
func friendsOnly(quiz *Quiz) bool { return quiz != nil && !quiz.Published }

// allFriendsOfHost fails with ROOM_FRIENDS_ONLY unless every member of
// the room is the host or a friend of the host (R14).
func (s *Service) allFriendsOfHost(ctx context.Context, id, host uuid.UUID) error {
	r, err := s.load(ctx, id)
	if err != nil {
		return err
	}
	for _, m := range r.Members {
		if m.UserID == host {
			continue
		}
		ok, err := s.d.Social.AreFriends(ctx, m.UserID, host)
		if err != nil {
			return err
		}
		if !ok {
			return apierror.New(apierror.RoomFriendsOnly)
		}
	}
	return nil
}

// countPlay counts a started Creator Quiz match; a failure only loses a
// count.
func (s *Service) countPlay(ctx context.Context, cfg *game.Config) {
	if cfg.CreatorQuizID == nil || s.d.Quizzes == nil {
		return
	}
	if err := s.d.Quizzes.CountPlay(ctx, *cfg.CreatorQuizID); err != nil {
		s.d.Logger.WarnContext(ctx, "lobby: counting a quiz play failed", "quiz_id", *cfg.CreatorQuizID, "error", err)
	}
}

// JoinRoom joins the room of an invite code: blocks with any member,
// friends-of-host rooms (D-023) and the format size apply. Joining the
// room the caller is in returns it; another room is left first.
func (s *Service) JoinRoom(ctx context.Context, user uuid.UUID, locale, code string) (Room, error) {
	c, err := s.lookup(ctx, code)
	if err != nil {
		return Room{}, err
	}
	if c.TargetKind != KindRoom {
		return Room{}, apierror.New(apierror.InviteCodeInvalid).With("reason", "not_a_room_code")
	}
	cur, err := s.refresh(ctx, c.TargetID)
	if err != nil {
		return Room{}, err
	}
	if cur.member(user) != nil {
		return s.render(ctx, cur)
	}
	for _, m := range cur.Members {
		blocked, err := s.d.Social.IsBlocked(ctx, user, m.UserID)
		if err != nil {
			return Room{}, err
		}
		if blocked {
			return Room{}, apierror.New(apierror.Blocked)
		}
	}
	if cur.FriendsOnly {
		friends, err := s.d.Social.AreFriends(ctx, user, cur.Host)
		if err != nil {
			return Room{}, err
		}
		if !friends {
			return Room{}, apierror.New(apierror.RoomFriendsOnly)
		}
	}
	if old, ok := s.CurrentRoom(ctx, user); ok && old != c.TargetID {
		if err := s.leave(ctx, user, old); err != nil && !isCode(err, apierror.RoomClosed) && !isCode(err, apierror.NotFound) {
			return Room{}, err
		}
	}
	r, err := s.update(ctx, c.TargetID, func(r *room) error {
		if r.member(user) != nil {
			return errNoChange
		}
		side, seat, ok := r.freeSeat()
		if !ok {
			return apierror.New(apierror.RoomFull)
		}
		r.Members = append(r.Members, member{UserID: user, Side: side, Seat: seat, Locale: locale})
		return nil
	})
	if err != nil {
		return Room{}, err
	}
	if err := sqlc.New(s.d.DB).UseCode(ctx, c.Code); err != nil {
		s.d.Logger.WarnContext(ctx, "lobby: counting a code use failed", "error", err)
	}
	return s.render(ctx, r)
}

func isCode(err error, code apierror.Code) bool {
	var e *apierror.Error
	return errors.As(err, &e) && e.Code == code
}

// LeaveRoom removes the caller; the host leaving closes the room, and so
// does the last member.
func (s *Service) LeaveRoom(ctx context.Context, user, id uuid.UUID) error {
	err := s.leave(ctx, user, id)
	if isCode(err, apierror.RoomClosed) {
		return nil
	}
	return err
}

func (s *Service) leaveCurrent(ctx context.Context, user uuid.UUID) error {
	id, ok := s.CurrentRoom(ctx, user)
	if !ok {
		return nil
	}
	if err := s.leave(ctx, user, id); err != nil && !isCode(err, apierror.RoomClosed) && !isCode(err, apierror.NotFound) {
		return err
	}
	return nil
}

func (s *Service) leave(ctx context.Context, user, id uuid.UUID) error {
	closeReason := ""
	_, err := s.update(ctx, id, func(r *room) error {
		if r.member(user) == nil {
			return apierror.New(apierror.NotFound)
		}
		if r.Status == RoomStarting {
			return errRoomBusy(r.Status)
		}
		switch {
		case r.Host == user:
			closeReason = CloseHost
			return errNoChange
		case len(r.Members) == 1:
			closeReason = CloseAllLeft
			return errNoChange
		}
		r.Members = slices.DeleteFunc(r.Members, func(m member) bool { return m.UserID == user })
		r.unreadyAll()
		return nil
	})
	if err != nil {
		return err
	}
	if closeReason != "" {
		return s.closeRoom(ctx, id, closeReason)
	}
	s.roomsMu.Lock()
	if s.userRooms[user] == id {
		delete(s.userRooms, user)
	}
	s.roomsMu.Unlock()
	return nil
}

// closeRoom ends a room: members get room.closed, the live state and the
// code go away. Closing twice is harmless.
func (s *Service) closeRoom(ctx context.Context, id uuid.UUID, reason string) error {
	r, err := s.load(ctx, id)
	if err != nil && !isCode(err, apierror.RoomClosed) {
		return err
	}
	now := s.d.Clock.Now().UTC()
	err = postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		n, err := q.CloseRoom(ctx, sqlc.CloseRoomParams{ID: id, Reason: reason, Now: now})
		if err != nil || n == 0 {
			return err
		}
		if err := q.RevokeCodes(ctx, sqlc.RevokeCodesParams{TargetKind: KindRoom, TargetID: id, Now: now}); err != nil {
			return err
		}
		return s.d.Events.PublishTx(ctx, tx, EventRoomClosed, now, RoomClosed{RoomID: id, Reason: reason})
	})
	if err != nil {
		return err
	}
	if r == nil {
		return nil
	}
	s.roomsMu.Lock()
	delete(s.rooms, id)
	for _, m := range r.Members {
		if cur, ok := s.userRooms[m.UserID]; ok && cur == id {
			delete(s.userRooms, m.UserID)
		}
	}
	s.roomsMu.Unlock()
	for _, m := range r.Members {
		s.send(ctx, m.UserID, realtime.Frame{T: "room.closed", S: roomStream(id), Seq: r.Version + 1, TS: now.UnixMilli(), D: map[string]any{"room_id": id, "reason": reason}})
	}
	return nil
}

// RoomClosed is the payload of lobby.RoomClosed.
type RoomClosed struct {
	RoomID uuid.UUID `json:"room_id"`
	Reason string    `json:"reason"`
}

// MoveSeat moves a member to an empty seat: the host moves anyone, others
// only themselves. The moved member is no longer ready.
func (s *Service) MoveSeat(ctx context.Context, user, id, who uuid.UUID, side, seat int) error {
	// A finished match returns the room to open first.
	if _, err := s.refresh(ctx, id); err != nil {
		return err
	}
	_, err := s.update(ctx, id, func(r *room) error {
		if r.member(user) == nil {
			return apierror.New(apierror.NotFound)
		}
		if who != user && r.Host != user {
			return errNotHost()
		}
		m := r.member(who)
		if m == nil {
			return apierror.New(apierror.NotFound)
		}
		if r.Status != RoomOpen {
			return errRoomBusy(r.Status)
		}
		if side < 1 || side > 2 || seat < 1 || seat > r.Config.Format.SideSize() {
			return apierror.Validation(apierror.FieldError{Field: "seat_no", Rule: "range"})
		}
		if m.Side == side && m.Seat == seat {
			return errNoChange
		}
		if r.seatTaken(side, seat) {
			return apierror.Validation(apierror.FieldError{Field: "seat_no", Rule: "taken"})
		}
		m.Side, m.Seat, m.Ready = side, seat, false
		return nil
	})
	return err
}

// SetReady sets the caller's ready flag (room open).
func (s *Service) SetReady(ctx context.Context, user, id uuid.UUID, ready bool) error {
	// A finished match returns the room to open first.
	if _, err := s.refresh(ctx, id); err != nil {
		return err
	}
	_, err := s.update(ctx, id, func(r *room) error {
		m := r.member(user)
		if m == nil {
			return apierror.New(apierror.NotFound)
		}
		if r.Status != RoomOpen {
			return errRoomBusy(r.Status)
		}
		if m.Ready == ready {
			return errNoChange
		}
		m.Ready = ready
		return nil
	})
	return err
}

// StartRoom creates the room's match (host; every seat taken, everyone
// but the host ready): the room is `starting` while game creates the
// match, then `in_match`; members get room.match_starting and the match
// stream's match.created.
func (s *Service) StartRoom(ctx context.Context, user, id uuid.UUID) (uuid.UUID, error) {
	// A finished match returns the room to open first.
	if _, err := s.refresh(ctx, id); err != nil {
		return uuid.Nil, err
	}
	r, err := s.update(ctx, id, func(r *room) error {
		if r.member(user) == nil {
			return apierror.New(apierror.NotFound)
		}
		if r.Host != user {
			return errNotHost()
		}
		if r.Status != RoomOpen {
			return errRoomBusy(r.Status)
		}
		if len(r.Members) < r.capacity() {
			return apierror.New(apierror.MatchStateInvalid).With("reason", "room_not_full")
		}
		for _, m := range r.Members {
			if !m.Ready && m.UserID != r.Host {
				return apierror.New(apierror.MatchStateInvalid).With("reason", "not_all_ready")
			}
		}
		r.Status = RoomStarting
		return nil
	})
	if err != nil {
		return uuid.Nil, err
	}
	var startErr error
	if r.FriendsOnly {
		startErr = s.allFriendsOfHost(ctx, id, r.Host)
	}
	size := r.Config.Format.SideSize()
	var sides [2][]game.Member
	for side := range 2 {
		sides[side] = make([]game.Member, size)
	}
	for _, m := range r.Members {
		sides[m.Side-1][m.Seat-1] = game.Member{UserID: m.UserID, Locale: quizLocale(&r.Config, m.Locale)}
	}
	req := game.CreateRequest{Config: r.Config, Sides: sides, OriginRef: &r.ID}
	if startErr == nil && r.Config.Format == "couple" {
		req.Couples, startErr = s.couplesOf(ctx, sides)
	}
	var c *game.Created
	if startErr == nil {
		c, startErr = s.d.Game.Start(ctx, req)
	}
	_, err = s.update(context.WithoutCancel(ctx), id, func(r *room) error {
		if r.Status != RoomStarting {
			return errNoChange
		}
		if startErr == nil {
			r.Status, r.MatchID = RoomInMatch, &c.Match.ID
		} else {
			r.Status = RoomOpen // the start failed: back to the lobby
		}
		return nil
	})
	if startErr != nil {
		return uuid.Nil, startErr
	}
	if err != nil {
		s.d.Logger.WarnContext(ctx, "lobby: marking a room in match failed", "room_id", id, "error", err)
	}
	s.countPlay(ctx, &r.Config)
	for _, m := range r.Members {
		s.send(ctx, m.UserID, realtime.Frame{T: "room.match_starting", S: roomStream(id), D: map[string]any{"room_id": id, "match_id": c.Match.ID}})
	}
	return c.Match.ID, nil
}

// couplesOf finds the active couple of each side of a couple room: the
// two seats of a side must be partners (R16).
func (s *Service) couplesOf(ctx context.Context, sides [2][]game.Member) ([2]*uuid.UUID, error) {
	var out [2]*uuid.UUID
	for i, side := range sides {
		if len(side) != 2 {
			return out, apierror.New(apierror.ModeNotAllowed).With("rule_id", "R16")
		}
		c, ok, err := s.d.Teams.ActiveCoupleOf(ctx, side[0].UserID)
		if err != nil {
			return out, err
		}
		if !ok || !slices.Contains(c.Users[:], side[1].UserID) {
			return out, apierror.New(apierror.ModeNotAllowed).With("rule_id", "R16").With("reason", "side_not_a_couple").With("side_no", i+1)
		}
		id := c.ID
		out[i] = &id
	}
	return out, nil
}

// SweepRooms closes rooms idle for 15 minutes; a room whose match is
// still running counts as active.
func (s *Service) SweepRooms(ctx context.Context) {
	rows, err := sqlc.New(s.d.DB).IdleRooms(ctx, s.d.Clock.Now().Add(-RoomIdle))
	if err != nil {
		s.d.Logger.WarnContext(ctx, "lobby: listing idle rooms failed", "error", err)
		return
	}
	for _, row := range rows {
		if row.Status == RoomInMatch && row.CurrentMatchID != nil {
			if sum, err := s.d.Game.Summary(ctx, *row.CurrentMatchID); err == nil && sum.Status != "completed" && sum.Status != "cancelled" {
				if r, err := s.load(ctx, row.ID); err == nil {
					s.persist(ctx, r)
				}
				continue
			}
		}
		if err := s.closeRoom(ctx, row.ID, CloseIdle); err != nil {
			s.d.Logger.WarnContext(ctx, "lobby: closing an idle room failed", "room_id", row.ID, "error", err)
		}
	}
}

// preview is a room as its invite code shows it.
func (s *Service) preview(ctx context.Context, id uuid.UUID) (RoomPreview, error) {
	r, err := s.refresh(ctx, id)
	if err != nil {
		return RoomPreview{}, err
	}
	cards, err := s.cards(ctx, r.Host)
	if err != nil {
		return RoomPreview{}, err
	}
	return RoomPreview{ID: r.ID, Status: r.Status, Host: cards[r.Host], MatchConfig: r.Config, Members: len(r.Members), Capacity: r.capacity()}, nil
}

// SnapshotRoom sends room.state to one session (room.join, resume).
func (s *Service) SnapshotRoom(ctx context.Context, c realtime.Client, id uuid.UUID) error {
	r, err := s.refresh(ctx, id)
	if err != nil {
		return err
	}
	if r.member(c.UserID) == nil {
		return apierror.New(apierror.NotFound)
	}
	v, err := s.render(ctx, r)
	if err != nil {
		return err
	}
	return s.d.RT.Send(ctx, realtime.Target{User: c.UserID, Session: c.SessionID}, realtime.Frame{
		T: "room.state", S: roomStream(id), Seq: r.Version, TS: s.d.Clock.Now().UnixMilli(), D: map[string]any{"room": v},
	})
}

// RoomOfStream parses "room:<id>".
func RoomOfStream(stream string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimPrefix(stream, "room:"))
	if err != nil {
		return uuid.Nil, errRoomClosed()
	}
	return id, nil
}

// Run sweeps idle rooms and old invite codes every minute until ctx ends.
func (s *Service) Run(ctx context.Context) error {
	tick := s.d.Clock.NewTicker(SweepEvery)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tick.C():
			s.SweepRooms(ctx)
			if err := s.SweepCodes(ctx); err != nil {
				s.d.Logger.WarnContext(ctx, "lobby: deleting old invite codes failed", "error", err)
			}
		}
	}
}
