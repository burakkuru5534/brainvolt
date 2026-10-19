// Package transport exposes social's REST endpoints (rest.md §7).
package transport

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/social/internal/service"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/idempotency"
)

// Transport serves social's routes.
type Transport struct {
	svc      *service.Service
	ident    *identity.Module
	idem     *idempotency.Store
	username func(r *http.Request, name string) (uuid.UUID, error)
}

// New returns the transport; username resolves usernames (profile).
func New(svc *service.Service, ident *identity.Module, idem *idempotency.Store, username func(r *http.Request, name string) (uuid.UUID, error)) *Transport {
	return &Transport{svc: svc, ident: ident, idem: idem, username: username}
}

// SendRequest is the body of POST /v1/friend-requests: exactly one of
// to_user_id and username.
type SendRequest struct {
	ToUserID *uuid.UUID `json:"to_user_id,omitempty"`
	Username *string    `json:"username,omitempty"`
	// Via: search (default), profile or post_match.
	Via string `json:"via,omitempty"`
}

// FriendRequests is GET /v1/friend-requests.
type FriendRequests struct {
	Requests []service.FriendRequest `json:"requests"`
}

// Blocks is GET /v1/blocks.
type Blocks struct {
	Blocks []service.BlockedUser `json:"blocks"`
}

// BlockRequest is the body of POST /v1/blocks.
type BlockRequest struct {
	UserID uuid.UUID `json:"user_id"`
}

var friendErrors = []apierror.Code{apierror.AccountUpgradeRequired, apierror.NotFound}

// Register adds social's routes. Friends need a full account (D-016);
// blocks are open to everyone.
func (t *Transport) Register(s *httpserver.Server) {
	full := identity.RequireFull("friends")
	s.Handle("GET /v1/friends", t.ident.Auth(t.friends, full), httpserver.WithDoc(httpserver.Doc{
		Tag: "social", Auth: "full", Summary: "Friends with presence (respecting show_online_status)",
		Response: service.FriendsPage{}, Errors: friendErrors,
		Query: []httpserver.QueryParam{{Name: "cursor", Type: "string", Format: "uuid", Description: "next_cursor of the previous page"}},
	}))
	s.Handle("DELETE /v1/friends/{user_id}", t.ident.Auth(t.unfriend, full), httpserver.WithDoc(httpserver.Doc{
		Tag: "social", Auth: "full", Summary: "Unfriend", Errors: friendErrors,
	}))
	s.Handle("POST /v1/friend-requests", t.ident.Auth(t.idem.Wrap(idempotency.Required, t.send), full), httpserver.WithDoc(httpserver.Doc{
		Tag: "social", Auth: "full", Idempotency: true, Status: http.StatusCreated,
		Summary: "Send a friend request by user id or username; sending back to a pending request accepts it",
		Request: SendRequest{}, Response: service.SendResult{},
		Errors: append([]apierror.Code{apierror.ValidationFailed, apierror.Blocked, apierror.AlreadyFriends, apierror.RequestPending,
			apierror.FriendLimitReached, apierror.Forbidden, apierror.RateLimited, apierror.IdempotencyKeyReused, apierror.IdempotencyInProgress}, friendErrors...),
	}))
	s.Handle("GET /v1/friend-requests", t.ident.Auth(t.requests, full), httpserver.WithDoc(httpserver.Doc{
		Tag: "social", Auth: "full", Summary: "Pending friend requests (box=in, default, or out)",
		Response: FriendRequests{}, Errors: friendErrors,
		Query: []httpserver.QueryParam{{Name: "box", Type: "string", Enum: []string{"in", "out"}}},
	}))
	for _, action := range []string{"accept", "decline", "cancel"} {
		s.Handle("POST /v1/friend-requests/{id}/"+action, t.ident.Auth(t.respond(action), full), httpserver.WithDoc(httpserver.Doc{
			Tag: "social", Auth: "full", Summary: "Friend request: " + action,
			Errors: append([]apierror.Code{apierror.Blocked, apierror.FriendLimitReached}, friendErrors...),
		}))
	}
	s.Handle("GET /v1/blocks", t.ident.Auth(t.blocks), httpserver.WithDoc(httpserver.Doc{
		Tag: "social", Auth: "any", Summary: "Users the caller blocked", Response: Blocks{},
	}))
	s.Handle("POST /v1/blocks", t.ident.Auth(t.block), httpserver.WithDoc(httpserver.Doc{
		Tag: "social", Auth: "any", Summary: "Block a user (ends the friendship and pending requests)",
		Request: BlockRequest{}, Errors: []apierror.Code{apierror.ValidationFailed, apierror.NotFound},
	}))
	s.Handle("DELETE /v1/blocks/{user_id}", t.ident.Auth(t.unblock), httpserver.WithDoc(httpserver.Doc{
		Tag: "social", Auth: "any", Summary: "Unblock a user",
	}))
}

func me(r *http.Request) uuid.UUID {
	p, _ := identity.PrincipalFrom(r.Context())
	return p.UserID
}

func pathUser(r *http.Request, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(r.PathValue(name))
	if err != nil {
		return uuid.Nil, apierror.New(apierror.NotFound)
	}
	return id, nil
}

func (t *Transport) friends(w http.ResponseWriter, r *http.Request) error {
	var after uuid.UUID
	if c := r.URL.Query().Get("cursor"); c != "" {
		id, err := uuid.Parse(c)
		if err != nil {
			return apierror.Validation(apierror.FieldError{Field: "cursor", Rule: "format"})
		}
		after = id
	}
	page, err := t.svc.Friends(r.Context(), me(r), after)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, page)
}

func (t *Transport) unfriend(w http.ResponseWriter, r *http.Request) error {
	id, err := pathUser(r, "user_id")
	if err != nil {
		return err
	}
	if err := t.svc.Unfriend(r.Context(), me(r), id); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (t *Transport) send(w http.ResponseWriter, r *http.Request) error {
	var req SendRequest
	if err := httpserver.DecodeJSON(r, &req); err != nil {
		return err
	}
	var f apierror.Fields
	if (req.ToUserID == nil) == (req.Username == nil) {
		f.Add("to_user_id", "one_of", "fields", []string{"to_user_id", "username"})
	}
	via := req.Via
	if via == "" {
		via = service.ViaSearch
	}
	if via != service.ViaSearch && via != service.ViaProfile && via != service.ViaPostMatch {
		f.Add("via", "enum")
	}
	if err := f.Err(); err != nil {
		return err
	}
	var to uuid.UUID
	if req.ToUserID != nil {
		to = *req.ToUserID
	} else {
		id, err := t.username(r, *req.Username)
		if err != nil {
			return err
		}
		to = id
	}
	res, err := t.svc.Send(r.Context(), me(r), to, via)
	if err != nil {
		return err
	}
	status := http.StatusCreated
	if res.Accepted {
		status = http.StatusOK
	}
	return httpserver.WriteJSON(w, status, res)
}

func (t *Transport) requests(w http.ResponseWriter, r *http.Request) error {
	box := r.URL.Query().Get("box")
	if box != "" && box != "in" && box != "out" {
		return apierror.Validation(apierror.FieldError{Field: "box", Rule: "enum"})
	}
	list, err := t.svc.Requests(r.Context(), me(r), box)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, FriendRequests{Requests: list})
}

func (t *Transport) respond(action string) httpserver.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		id, err := pathUser(r, "id")
		if err != nil {
			return err
		}
		switch action {
		case "accept":
			err = t.svc.Respond(r.Context(), me(r), id, true)
		case "decline":
			err = t.svc.Respond(r.Context(), me(r), id, false)
		default:
			err = t.svc.Cancel(r.Context(), me(r), id)
		}
		if err != nil {
			return err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	}
}

func (t *Transport) blocks(w http.ResponseWriter, r *http.Request) error {
	list, err := t.svc.Blocks(r.Context(), me(r))
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, Blocks{Blocks: list})
}

func (t *Transport) block(w http.ResponseWriter, r *http.Request) error {
	var req BlockRequest
	if err := httpserver.DecodeJSON(r, &req); err != nil {
		return err
	}
	if err := t.svc.Block(r.Context(), me(r), req.UserID); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (t *Transport) unblock(w http.ResponseWriter, r *http.Request) error {
	id, err := pathUser(r, "user_id")
	if err != nil {
		return err
	}
	if err := t.svc.Unblock(r.Context(), me(r), id); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
