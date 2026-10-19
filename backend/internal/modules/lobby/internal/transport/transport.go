// Package transport exposes lobby's REST endpoints (rest.md §6) and its
// WebSocket intents (room.*, rematch.respond; websocket.md §5.1).
package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/lobby/internal/service"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/realtime"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/i18n"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/idempotency"
)

// Transport serves lobby's routes.
type Transport struct {
	svc   *service.Service
	ident *identity.Module
	idem  *idempotency.Store
}

// New returns the transport.
func New(svc *service.Service, ident *identity.Module, idem *idempotency.Store) *Transport {
	return &Transport{svc: svc, ident: ident, idem: idem}
}

// CodeRequest is the body of POST /v1/invite-codes. Team and couple codes
// come with the team module.
type CodeRequest struct {
	Kind string `json:"kind"`
}

// ChallengeRequest is the body of POST /v1/challenges: exactly one of
// target_user_id (1v1) and target_couple_id (couple format, sent by a
// partner of an active couple).
type ChallengeRequest struct {
	TargetUserID   *uuid.UUID  `json:"target_user_id,omitempty"`
	TargetCoupleID *uuid.UUID  `json:"target_couple_id,omitempty"`
	MatchConfig    game.Config `json:"match_config"`
}

// Challenges is GET /v1/challenges.
type Challenges struct {
	Challenges []service.Challenge `json:"challenges"`
}

// RoomRequest is the body of POST /v1/rooms and PATCH /v1/rooms/{id}.
type RoomRequest struct {
	MatchConfig game.Config `json:"match_config"`
}

// JoinRequest is the body of POST /v1/rooms/join.
type JoinRequest struct {
	Code string `json:"code"`
}

// Register adds lobby's routes.
func (t *Transport) Register(s *httpserver.Server) {
	s.Handle("POST /v1/invite-codes", t.ident.Auth(t.idem.Wrap(idempotency.Required, t.createCode), identity.RequireFull("friends")), httpserver.WithDoc(httpserver.Doc{
		Tag: "lobby", Auth: "full", Idempotency: true, Summary: "The caller's friend invite code (kind=friend; one per user, 7 days)",
		Request: CodeRequest{}, Response: service.InviteCode{},
		Errors: []apierror.Code{apierror.ValidationFailed, apierror.AccountUpgradeRequired, apierror.IdempotencyKeyReused, apierror.IdempotencyInProgress},
	}))
	s.Handle("GET /v1/invite-codes/{code}", t.ident.Auth(t.resolveCode), httpserver.WithDoc(httpserver.Doc{
		Tag: "lobby", Auth: "any", Summary: "What an invite code opens (friend → user, room → room preview)",
		Response: service.ResolvedCode{}, Errors: []apierror.Code{apierror.InviteCodeInvalid, apierror.InviteCodeExpired, apierror.Blocked, apierror.RoomClosed},
	}))
	s.Handle("POST /v1/challenges", t.ident.Auth(t.idem.Wrap(idempotency.Required, t.challenge)), httpserver.WithDoc(httpserver.Doc{
		Tag: "lobby", Auth: "any", Idempotency: true, Status: http.StatusCreated,
		Summary: "Direct challenge: a user (1v1) or a couple (couple format); the targets get challenge.received",
		Request: ChallengeRequest{}, Response: service.Challenge{},
		Errors: []apierror.Code{apierror.ValidationFailed, apierror.NotFound, apierror.Blocked, apierror.ChallengesDisabled, apierror.RequestPending,
			apierror.RateLimited, apierror.ModeNotAllowed, apierror.IdempotencyKeyReused, apierror.IdempotencyInProgress},
	}))
	s.Handle("POST /v1/matches/{id}/revenge", t.ident.Auth(t.idem.Wrap(idempotency.Required, t.revenge)), httpserver.WithDoc(httpserver.Doc{
		Tag: "lobby", Auth: "losing side", Idempotency: true, Status: http.StatusCreated,
		Summary: "Revenge challenge to the winners of a completed 1v1 or couple match, with the challenger's home category (D-045)",
		Request: RevengeRequest{}, Response: service.Challenge{},
		Errors: []apierror.Code{apierror.ValidationFailed, apierror.MatchNotFound, apierror.RevengeNotAllowed, apierror.Blocked,
			apierror.RequestPending, apierror.RateLimited, apierror.ModeNotAllowed, apierror.IdempotencyKeyReused, apierror.IdempotencyInProgress},
	}))
	s.Handle("GET /v1/challenges", t.ident.Auth(t.challenges), httpserver.WithDoc(httpserver.Doc{
		Tag: "lobby", Auth: "any", Summary: "Pending challenges (box=in, default, or out)", Response: Challenges{},
		Query: []httpserver.QueryParam{{Name: "box", Type: "string", Enum: []string{"in", "out"}}},
	}))
	for _, action := range []string{service.ActionAccept, service.ActionDecline, service.ActionCancel} {
		s.Handle("POST /v1/challenges/{id}/"+action, t.ident.Auth(t.respond(action)), httpserver.WithDoc(httpserver.Doc{
			Tag: "lobby", Auth: "any", Summary: "Challenge: " + action + " (accept creates the match; both get challenge.updated)",
			Response: service.Challenge{},
			Errors: []apierror.Code{apierror.NotFound, apierror.ChallengeExpired, apierror.MatchStateInvalid, apierror.AlreadyInMatch,
				apierror.NotEnoughQuestions, apierror.ModeNotAllowed},
		}))
	}
	s.Handle("POST /v1/rooms", t.ident.Auth(t.idem.Wrap(idempotency.Required, t.createRoom)), httpserver.WithDoc(httpserver.Doc{
		Tag: "lobby", Auth: "any", Idempotency: true, Status: http.StatusCreated,
		Summary: "Create a private room (the caller hosts; leaves a previous room); returns the room and its invite code",
		Request: RoomRequest{}, Response: service.Room{},
		Errors: []apierror.Code{apierror.ValidationFailed, apierror.ModeNotAllowed, apierror.IdempotencyKeyReused, apierror.IdempotencyInProgress},
	}))
	s.Handle("POST /v1/rooms/join", t.ident.Auth(t.joinRoom), httpserver.WithDoc(httpserver.Doc{
		Tag: "lobby", Auth: "any", Summary: "Join a room by invite code (members get room.state)",
		Request: JoinRequest{}, Response: service.Room{},
		Errors: []apierror.Code{apierror.InviteCodeInvalid, apierror.InviteCodeExpired, apierror.RoomFull, apierror.RoomClosed,
			apierror.RoomFriendsOnly, apierror.Blocked},
	}))
	s.Handle("GET /v1/rooms/{id}", t.ident.Auth(t.getRoom), httpserver.WithDoc(httpserver.Doc{
		Tag: "lobby", Auth: "member", Summary: "Room state (live updates: room.state on the room:<id> stream)",
		Response: service.Room{}, Errors: []apierror.Code{apierror.NotFound, apierror.RoomClosed},
	}))
	s.Handle("PATCH /v1/rooms/{id}", t.ident.Auth(t.patchRoom), httpserver.WithDoc(httpserver.Doc{
		Tag: "lobby", Auth: "host", Summary: "Change the match configuration (If-Match: room version); ready flags reset",
		Request: RoomRequest{}, Response: service.Room{},
		Errors: []apierror.Code{apierror.NotFound, apierror.Forbidden, apierror.VersionConflict, apierror.ModeNotAllowed,
			apierror.MatchStateInvalid, apierror.RoomClosed, apierror.RoomFull},
	}))
	s.Handle("POST /v1/rooms/{id}/leave", t.ident.Auth(t.leaveRoom), httpserver.WithDoc(httpserver.Doc{
		Tag: "lobby", Auth: "member", Summary: "Leave a room; the host leaving closes it (room.closed)",
		Errors: []apierror.Code{apierror.NotFound, apierror.MatchStateInvalid},
	}))
}

func me(r *http.Request) uuid.UUID {
	p, _ := identity.PrincipalFrom(r.Context())
	return p.UserID
}

func pathID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return uuid.Nil, apierror.New(apierror.NotFound)
	}
	return id, nil
}

func locale(r *http.Request) string { return string(i18n.FromContext(r.Context())) }

func (t *Transport) createCode(w http.ResponseWriter, r *http.Request) error {
	var req CodeRequest
	if err := httpserver.DecodeJSON(r, &req); err != nil {
		return err
	}
	if req.Kind != service.KindFriend {
		return apierror.Validation(apierror.FieldError{Field: "kind", Rule: "enum"})
	}
	c, err := t.svc.FriendCode(r.Context(), me(r))
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, c)
}

func (t *Transport) resolveCode(w http.ResponseWriter, r *http.Request) error {
	c, err := t.svc.Resolve(r.Context(), me(r), r.PathValue("code"))
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, c)
}

func (t *Transport) challenge(w http.ResponseWriter, r *http.Request) error {
	var req ChallengeRequest
	if err := httpserver.DecodeJSON(r, &req); err != nil {
		return err
	}
	c, err := t.svc.Challenge(r.Context(), me(r), service.ChallengeTarget{User: req.TargetUserID, Couple: req.TargetCoupleID}, req.MatchConfig)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusCreated, c)
}

// RevengeRequest is the body of POST /v1/matches/{id}/revenge.
type RevengeRequest struct {
	HomeCategoryID uuid.UUID `json:"home_category_id"`
}

func (t *Transport) revenge(w http.ResponseWriter, r *http.Request) error {
	parent, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return apierror.New(apierror.MatchNotFound)
	}
	var req RevengeRequest
	if err := httpserver.DecodeJSON(r, &req); err != nil {
		return err
	}
	c, err := t.svc.Revenge(r.Context(), me(r), parent, req.HomeCategoryID)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusCreated, c)
}

func (t *Transport) challenges(w http.ResponseWriter, r *http.Request) error {
	box := r.URL.Query().Get("box")
	if box != "" && box != "in" && box != "out" {
		return apierror.Validation(apierror.FieldError{Field: "box", Rule: "enum"})
	}
	list, err := t.svc.Challenges(r.Context(), me(r), box)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, Challenges{Challenges: list})
}

func (t *Transport) respond(action string) httpserver.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		c, err := t.svc.RespondChallenge(r.Context(), me(r), id, action)
		if err != nil {
			return err
		}
		return httpserver.WriteJSON(w, http.StatusOK, c)
	}
}

func (t *Transport) createRoom(w http.ResponseWriter, r *http.Request) error {
	var req RoomRequest
	if err := httpserver.DecodeJSON(r, &req); err != nil {
		return err
	}
	room, err := t.svc.CreateRoom(r.Context(), me(r), locale(r), req.MatchConfig)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusCreated, room)
}

func (t *Transport) joinRoom(w http.ResponseWriter, r *http.Request) error {
	var req JoinRequest
	if err := httpserver.DecodeJSON(r, &req); err != nil {
		return err
	}
	room, err := t.svc.JoinRoom(r.Context(), me(r), locale(r), req.Code)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, room)
}

func (t *Transport) getRoom(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	room, err := t.svc.GetRoom(r.Context(), me(r), id)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, room)
}

func (t *Transport) patchRoom(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var version *int64
	if h := strings.Trim(r.Header.Get("If-Match"), `"W/ `); h != "" {
		v, err := strconv.ParseInt(h, 10, 64)
		if err != nil {
			return apierror.Validation(apierror.FieldError{Field: "If-Match", Rule: "format"})
		}
		version = &v
	}
	var req RoomRequest
	if err := httpserver.DecodeJSON(r, &req); err != nil {
		return err
	}
	room, err := t.svc.UpdateRoomConfig(r.Context(), me(r), id, req.MatchConfig, version)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, room)
}

func (t *Transport) leaveRoom(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	if err := t.svc.LeaveRoom(r.Context(), me(r), id); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// roomIntent is the payload of room.* intents.
type roomIntent struct {
	RoomID uuid.UUID `json:"room_id"`
	SideNo int       `json:"side_no"`
	SeatNo int       `json:"seat_no"`
	// UserID is the member the host moves (default: the caller).
	UserID uuid.UUID `json:"user_id"`
}

// HandleRoom routes room.* intents.
func (t *Transport) HandleRoom(ctx context.Context, c realtime.Client, msg realtime.Message) error {
	var p roomIntent
	if err := json.Unmarshal(msg.Data, &p); err != nil || p.RoomID == uuid.Nil {
		return apierror.New(apierror.MalformedRequest)
	}
	switch msg.Type {
	case "room.join":
		return t.svc.SnapshotRoom(ctx, c, p.RoomID)
	case "room.update_seat":
		who := p.UserID
		if who == uuid.Nil {
			who = c.UserID
		}
		return t.svc.MoveSeat(ctx, c.UserID, p.RoomID, who, p.SideNo, p.SeatNo)
	case "room.ready", "room.unready":
		return t.svc.SetReady(ctx, c.UserID, p.RoomID, msg.Type == "room.ready")
	case "room.start":
		_, err := t.svc.StartRoom(ctx, c.UserID, p.RoomID)
		return err
	}
	return apierror.New(apierror.MalformedRequest).With("reason", "unknown_intent")
}

// rematchIntent is the payload of rematch.respond.
type rematchIntent struct {
	MatchID uuid.UUID `json:"match_id"`
	Accept  *bool     `json:"accept"`
}

// HandleRematch routes rematch.respond.
func (t *Transport) HandleRematch(ctx context.Context, c realtime.Client, msg realtime.Message) error {
	var p rematchIntent
	if msg.Type != "rematch.respond" {
		return apierror.New(apierror.MalformedRequest).With("reason", "unknown_intent")
	}
	if err := json.Unmarshal(msg.Data, &p); err != nil || p.MatchID == uuid.Nil || p.Accept == nil {
		return apierror.New(apierror.MalformedRequest)
	}
	return t.svc.RespondRematch(ctx, c.UserID, p.MatchID, *p.Accept)
}
