package transport

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/notification/internal/service"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/i18n"
)

// Inbox serves the inbox and preference routes (rest.md §12).
type Inbox struct {
	svc   *service.Service
	ident *identity.Module
}

// NewInbox returns the inbox transport.
func NewInbox(svc *service.Service, ident *identity.Module) *Inbox {
	return &Inbox{svc: svc, ident: ident}
}

// ReadInput is the body of POST /v1/me/inbox/read.
type ReadInput struct {
	IDs []uuid.UUID `json:"ids,omitempty"`
	All bool        `json:"all,omitempty"`
}

// Register adds the routes.
func (t *Inbox) Register(s *httpserver.Server) {
	a := t.ident.Auth
	doc := func(d httpserver.Doc) httpserver.RouteOption { d.Tag = "notification"; return httpserver.WithDoc(d) }
	s.Handle("GET /v1/me/inbox", a(t.list, identity.AllowSuspended()), doc(httpserver.Doc{
		Auth: "any", Summary: "In-app notifications, newest first, in the request language (30 days)", Response: service.Inbox{},
		Query: []httpserver.QueryParam{{Name: "cursor", Type: "string", Description: "next_cursor of the previous page"}},
	}))
	s.Handle("POST /v1/me/inbox/read", a(t.read, identity.AllowSuspended()), doc(httpserver.Doc{
		Auth: "any", Summary: "Mark notifications read: {ids} or {all: true}", Request: ReadInput{},
		Errors: []apierror.Code{apierror.ValidationFailed},
	}))
	s.Handle("GET /v1/me/notification-preferences", a(t.prefs), doc(httpserver.Doc{
		Auth: "any", Summary: "Per-category push / in-app channels and quiet hours", Response: service.Preferences{},
	}))
	s.Handle("PATCH /v1/me/notification-preferences", a(t.updatePrefs), doc(httpserver.Doc{
		Auth: "any", Summary: "Change categories and quiet hours (moderation cannot be turned off)",
		Request: service.PreferencesInput{}, Response: service.Preferences{},
		Errors: []apierror.Code{apierror.ValidationFailed},
	}))
}

func caller(r *http.Request) uuid.UUID {
	p, _ := identity.PrincipalFrom(r.Context())
	return p.UserID
}

func (t *Inbox) list(w http.ResponseWriter, r *http.Request) error {
	var cursor *time.Time
	if v := r.URL.Query().Get("cursor"); v != "" {
		c, err := time.Parse(time.RFC3339Nano, v)
		if err != nil {
			return apierror.Validation(apierror.FieldError{Field: "cursor", Rule: "format"})
		}
		cursor = &c
	}
	out, err := t.svc.List(r.Context(), caller(r), string(i18n.FromContext(r.Context())), cursor)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, out)
}

func (t *Inbox) read(w http.ResponseWriter, r *http.Request) error {
	var in ReadInput
	if err := httpserver.DecodeJSON(r, &in); err != nil {
		return err
	}
	if len(in.IDs) > 100 {
		return apierror.Validation(apierror.FieldError{Field: "ids", Rule: "max_items"})
	}
	if err := t.svc.MarkRead(r.Context(), caller(r), in.IDs, in.All); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (t *Inbox) prefs(w http.ResponseWriter, r *http.Request) error {
	out, err := t.svc.Preferences(r.Context(), caller(r))
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, out)
}

func (t *Inbox) updatePrefs(w http.ResponseWriter, r *http.Request) error {
	var in service.PreferencesInput
	if err := httpserver.DecodeJSON(r, &in); err != nil {
		return err
	}
	out, err := t.svc.UpdatePreferences(r.Context(), caller(r), in)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, out)
}
