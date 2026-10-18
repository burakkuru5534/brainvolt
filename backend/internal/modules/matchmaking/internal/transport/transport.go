// Package transport exposes matchmaking's REST endpoints (rest.md §5).
package transport

import (
	"net/http"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/matchmaking/internal/service"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/i18n"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/idempotency"
)

// Register adds the ticket routes.
func Register(s *httpserver.Server, svc *service.Service, ident *identity.Module, idem *idempotency.Store) {
	s.Handle("POST /v1/matchmaking/tickets", ident.Auth(idem.Wrap(idempotency.Required, func(w http.ResponseWriter, r *http.Request) error {
		p, _ := identity.PrincipalFrom(r.Context())
		var req service.Request
		if err := httpserver.DecodeJSON(r, &req); err != nil {
			return err
		}
		t, err := svc.Enqueue(r.Context(), p.UserID, string(i18n.FromContext(r.Context())), req)
		if err != nil {
			return err
		}
		return httpserver.WriteJSON(w, http.StatusCreated, t)
	})), httpserver.WithDoc(httpserver.Doc{
		Tag: "matchmaking", Auth: "any", Idempotency: true, Status: http.StatusCreated,
		Summary: "Join the Quick Duel 1v1 queue (Phase 4: first come, first served); practice starts a match against the development bot",
		Request: service.Request{}, Response: service.Ticket{},
		Errors: []apierror.Code{apierror.ModeNotAllowed, apierror.AlreadyQueued, apierror.AlreadyInMatch, apierror.ValidationFailed,
			apierror.NotEnoughQuestions, apierror.IdempotencyKeyReused, apierror.IdempotencyInProgress},
	}))
	s.Handle("GET /v1/matchmaking/tickets/current", ident.Auth(func(w http.ResponseWriter, r *http.Request) error {
		p, _ := identity.PrincipalFrom(r.Context())
		t, err := svc.Current(r.Context(), p.UserID)
		if err != nil {
			return err
		}
		return httpserver.WriteJSON(w, http.StatusOK, t)
	}), httpserver.WithDoc(httpserver.Doc{
		Tag: "matchmaking", Auth: "any", Summary: "The caller's queue ticket and waiting time",
		Response: service.Ticket{}, Errors: []apierror.Code{apierror.NotFound},
	}))
	s.Handle("DELETE /v1/matchmaking/tickets/current", ident.Auth(func(w http.ResponseWriter, r *http.Request) error {
		p, _ := identity.PrincipalFrom(r.Context())
		if err := svc.Leave(r.Context(), p.UserID); err != nil {
			return err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	}), httpserver.WithDoc(httpserver.Doc{Tag: "matchmaking", Auth: "any", Summary: "Leave the queue"}))
}
