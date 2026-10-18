// Package transport exposes game's REST endpoints (rest.md §4).
package transport

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/service"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/i18n"
)

// Transport serves game's routes.
type Transport struct {
	svc   *service.Service
	ident *identity.Module
}

// New returns the transport.
func New(svc *service.Service, ident *identity.Module) *Transport {
	return &Transport{svc: svc, ident: ident}
}

// Register adds game's routes.
func (t *Transport) Register(s *httpserver.Server) {
	s.Handle("GET /v1/matches/current", t.ident.Auth(t.current), httpserver.WithDoc(httpserver.Doc{
		Tag: "game", Auth: "any",
		Summary:  "The caller's live match, if any, with the WebSocket stream to resume",
		Response: service.Current{},
	}))
	s.Handle("GET /v1/matches/{id}", t.ident.Auth(t.match), httpserver.WithDoc(httpserver.Doc{
		Tag: "game", Auth: "any",
		Summary:  "Result screen data for a participant: sides, participants and, once over, every round with answers",
		Response: service.MatchView{},
		Errors:   []apierror.Code{apierror.MatchNotFound, apierror.NotAParticipant},
	}))
}

func (t *Transport) current(w http.ResponseWriter, r *http.Request) error {
	p, _ := identity.PrincipalFrom(r.Context())
	c, err := t.svc.Current(r.Context(), p.UserID)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, c)
}

func (t *Transport) match(w http.ResponseWriter, r *http.Request) error {
	p, _ := identity.PrincipalFrom(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return apierror.New(apierror.MatchNotFound)
	}
	v, err := t.svc.Match(r.Context(), p.UserID, id, string(i18n.FromContext(r.Context())))
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, v)
}
