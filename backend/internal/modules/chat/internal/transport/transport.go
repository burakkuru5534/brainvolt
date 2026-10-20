// Package transport exposes chat's REST routes (rest.md §13) and its
// WebSocket intents chat.send and chat.typing (websocket.md).
package transport

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/chat/internal/service"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/realtime"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
)

// Transport serves chat.
type Transport struct {
	svc   *service.Service
	ident *identity.Module
}

// New returns the transport.
func New(svc *service.Service, ident *identity.Module) *Transport {
	return &Transport{svc: svc, ident: ident}
}

// ChatReadInput is the body of POST /v1/chat/channels/{id}/read.
type ChatReadInput struct {
	LastReadID uuid.UUID `json:"last_read_id"`
}

// Register adds the routes.
func (t *Transport) Register(s *httpserver.Server) {
	a := t.ident.Auth
	doc := func(d httpserver.Doc) httpserver.RouteOption { d.Tag = "chat"; return httpserver.WithDoc(d) }
	s.Handle("POST /v1/chat/channels", a(t.open), doc(httpserver.Doc{
		Auth: "member", Summary: "Open the chat of one's team or couple ({team_id} or {couple_id}); created on first use",
		Request: service.OpenInput{}, Response: service.Channel{},
		Errors: []apierror.Code{apierror.ValidationFailed, apierror.NotFound},
	}))
	s.Handle("GET /v1/chat/channels/{id}/messages", a(t.history), doc(httpserver.Doc{
		Auth: "member", Summary: "Messages newest first (sending is the WebSocket intent chat.send)", Response: service.History{},
		Query:  []httpserver.QueryParam{{Name: "cursor", Type: "string", Format: "uuid", Description: "next_cursor of the previous page"}},
		Errors: []apierror.Code{apierror.NotFound},
	}))
	s.Handle("POST /v1/chat/channels/{id}/read", a(t.read), doc(httpserver.Doc{
		Auth: "member", Summary: "Move the read cursor", Request: ChatReadInput{}, Errors: []apierror.Code{apierror.NotFound},
	}))
	s.Handle("DELETE /v1/chat/messages/{id}", a(t.remove), doc(httpserver.Doc{
		Auth: "sender / mod", Summary: "Remove a message (its sender, or staff)", Errors: []apierror.Code{apierror.NotFound},
	}))
}

func principal(r *http.Request) identity.Principal {
	p, _ := identity.PrincipalFrom(r.Context())
	return p
}

func pathID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return uuid.Nil, apierror.New(apierror.NotFound)
	}
	return id, nil
}

func (t *Transport) open(w http.ResponseWriter, r *http.Request) error {
	var in service.OpenInput
	if err := httpserver.DecodeJSON(r, &in); err != nil {
		return err
	}
	c, err := t.svc.Open(r.Context(), principal(r).UserID, in)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, c)
}

func (t *Transport) history(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var before *uuid.UUID
	if v := r.URL.Query().Get("cursor"); v != "" {
		c, err := uuid.Parse(v)
		if err != nil {
			return apierror.Validation(apierror.FieldError{Field: "cursor", Rule: "format"})
		}
		before = &c
	}
	h, err := t.svc.History(r.Context(), principal(r).UserID, id, before)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, h)
}

func (t *Transport) read(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var in ChatReadInput
	if err := httpserver.DecodeJSON(r, &in); err != nil {
		return err
	}
	if err := t.svc.Read(r.Context(), principal(r).UserID, id, in.LastReadID); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (t *Transport) remove(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	p := principal(r)
	staff := p.HasRole(identity.RoleModerator) || p.HasRole(identity.RoleAdmin)
	if err := t.svc.Remove(r.Context(), p.UserID, id, staff); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// typingIntent is the payload of chat.typing.
type typingIntent struct {
	ChannelID uuid.UUID `json:"channel_id"`
}

// Handle routes chat.send and chat.typing.
func (t *Transport) Handle(ctx context.Context, c realtime.Client, msg realtime.Message) error {
	switch msg.Type {
	case "chat.send":
		var in service.SendInput
		if err := json.Unmarshal(msg.Data, &in); err != nil || in.ChannelID == uuid.Nil {
			return apierror.New(apierror.MalformedRequest)
		}
		_, err := t.svc.Send(ctx, c.UserID, in)
		return err
	case "chat.typing":
		var in typingIntent
		if err := json.Unmarshal(msg.Data, &in); err != nil || in.ChannelID == uuid.Nil {
			return apierror.New(apierror.MalformedRequest)
		}
		return t.svc.Typing(ctx, c.UserID, in.ChannelID)
	}
	return apierror.New(apierror.MalformedRequest).With("reason", "unknown_intent")
}
