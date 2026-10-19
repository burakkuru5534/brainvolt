// Package transport exposes progression's REST endpoints (rest.md §10).
package transport

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/progression/internal/service"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/i18n"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/idempotency"
)

// Transport serves progression's routes.
type Transport struct {
	svc   *service.Service
	ident *identity.Module
	idem  *idempotency.Store
}

// New returns the transport.
func New(svc *service.Service, ident *identity.Module, idem *idempotency.Store) *Transport {
	return &Transport{svc: svc, ident: ident, idem: idem}
}

// Badges is GET /v1/badges and /v1/me/badges.
type Badges struct {
	Badges []service.Badge `json:"badges"`
}

// Register adds progression's routes; every account has progression.
func (t *Transport) Register(s *httpserver.Server) {
	a := t.ident.Auth
	doc := func(d httpserver.Doc) httpserver.RouteOption {
		d.Tag = "progression"
		d.Auth = "any"
		return httpserver.WithDoc(d)
	}
	s.Handle("GET /v1/me/progress", a(t.progress), doc(httpserver.Doc{
		Summary: "XP, level with the XP inside it, play streak and badge count", Response: service.Progress{},
	}))
	s.Handle("GET /v1/me/quests", a(t.quests), doc(httpserver.Doc{
		Summary: "Today's and this week's quests (3 each, assigned on first read) and the rerolls left today", Response: service.Quests{},
	}))
	questErrors := []apierror.Code{apierror.NotFound, apierror.IdempotencyKeyReused, apierror.IdempotencyInProgress}
	s.Handle("POST /v1/me/quests/{id}/claim", a(t.idem.Wrap(idempotency.Required, t.claim)), doc(httpserver.Doc{
		Idempotency: true, Summary: "Claim a completed quest's XP", Response: service.Claimed{},
		Errors: append([]apierror.Code{apierror.QuestNotCompleted, apierror.QuestAlreadyClaimed}, questErrors...),
	}))
	s.Handle("POST /v1/me/quests/{id}/reroll", a(t.idem.Wrap(idempotency.Required, t.reroll)), doc(httpserver.Doc{
		Idempotency: true, Summary: "Swap an open quest for another one (one free reroll a day)", Response: service.Quests{},
		Errors: append([]apierror.Code{apierror.QuestRerollLimit, apierror.QuestNotRerollable}, questErrors...),
	}))
	s.Handle("GET /v1/badges", a(t.badges(false)), doc(httpserver.Doc{
		Summary: "Badge catalogue with the caller's earn dates (hidden badges only once earned)", Response: Badges{},
	}))
	s.Handle("GET /v1/me/badges", a(t.badges(true)), doc(httpserver.Doc{
		Summary: "The caller's earned badges", Response: Badges{},
	}))
	s.Handle("GET /v1/me/awards/{match_id}", a(t.awards), doc(httpserver.Doc{
		Summary:  "Progression awards of a finished match (the match.awards content) for clients that missed the message",
		Response: service.Awards{}, Errors: []apierror.Code{apierror.NotFound},
	}))
}

func me(r *http.Request) uuid.UUID {
	p, _ := identity.PrincipalFrom(r.Context())
	return p.UserID
}

func locale(r *http.Request) string { return string(i18n.FromContext(r.Context())) }

func pathID(r *http.Request, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(r.PathValue(name))
	if err != nil {
		return uuid.Nil, apierror.New(apierror.NotFound)
	}
	return id, nil
}

func (t *Transport) progress(w http.ResponseWriter, r *http.Request) error {
	p, err := t.svc.Progress(r.Context(), me(r))
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, p)
}

func (t *Transport) quests(w http.ResponseWriter, r *http.Request) error {
	q, err := t.svc.Quests(r.Context(), me(r), locale(r))
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, q)
}

func (t *Transport) claim(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	c, err := t.svc.Claim(r.Context(), me(r), id)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, c)
}

func (t *Transport) reroll(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	q, err := t.svc.Reroll(r.Context(), me(r), id, locale(r))
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, q)
}

func (t *Transport) badges(earned bool) httpserver.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		b, err := t.svc.Badges(r.Context(), me(r), locale(r), earned)
		if err != nil {
			return err
		}
		return httpserver.WriteJSON(w, http.StatusOK, Badges{Badges: b})
	}
}

func (t *Transport) awards(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "match_id")
	if err != nil {
		return err
	}
	a, err := t.svc.MatchAwards(r.Context(), me(r), id, locale(r))
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, a)
}
