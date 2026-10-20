// Package transport exposes tournament's REST endpoints (rest.md §14).
package transport

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/tournament/internal/service"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/i18n"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/idempotency"
)

// Transport serves tournament's routes.
type Transport struct {
	svc   *service.Service
	ident *identity.Module
	idem  *idempotency.Store
}

// New returns the transport.
func New(svc *service.Service, ident *identity.Module, idem *idempotency.Store) *Transport {
	return &Transport{svc: svc, ident: ident, idem: idem}
}

// Tournaments is a list of tournaments.
type Tournaments struct {
	Tournaments []service.Tournament `json:"tournaments"`
}

// Register adds tournament's routes.
func (t *Transport) Register(s *httpserver.Server) {
	a := t.ident.Auth
	adm := identity.RequireRole(identity.RoleAdmin)
	doc := func(d httpserver.Doc) httpserver.RouteOption { d.Tag = "tournament"; return httpserver.WithDoc(d) }
	idem := []apierror.Code{apierror.IdempotencyKeyReused, apierror.IdempotencyInProgress}
	s.Handle("GET /v1/tournaments", a(t.list), doc(httpserver.Doc{
		Auth: "any", Summary: "Tournaments, newest start first, with the caller's entry", Response: Tournaments{},
		Query: []httpserver.QueryParam{{Name: "status", Type: "string", Enum: []string{"registration", "running", "finished", "cancelled", "draft"}}},
	}))
	s.Handle("GET /v1/tournaments/{id}", a(t.get), doc(httpserver.Doc{
		Auth: "any", Summary: "A tournament with its entries, bracket and the caller's current match", Response: service.TournamentDetail{},
		Errors: []apierror.Code{apierror.NotFound},
	}))
	s.Handle("POST /v1/tournaments/{id}/entries", a(t.idem.Wrap(idempotency.Optional, t.register)), doc(httpserver.Doc{
		Auth: "full", Idempotency: true, Status: http.StatusCreated,
		Summary: "Register as a user, the caller's couple, or a team (captain, with its lineup; D-090)",
		Request: service.TournamentRegistration{}, Response: service.TournamentEntry{},
		Errors: append([]apierror.Code{apierror.ValidationFailed, apierror.NotFound, apierror.AccountUpgradeRequired, apierror.TournamentRegistrationClosed,
			apierror.TournamentFull, apierror.TournamentRequirementsNotMet, apierror.NotCaptain}, idem...),
	}))
	s.Handle("DELETE /v1/tournaments/{id}/entries/mine", a(t.withdraw), doc(httpserver.Doc{
		Auth: "full", Summary: "Withdraw before the start", Errors: []apierror.Code{apierror.NotFound, apierror.MatchStateInvalid, apierror.NotCaptain},
	}))
	s.Handle("POST /v1/tournaments/{id}/check-in", a(t.checkIn), doc(httpserver.Doc{
		Auth: "full", Summary: "Check in during the hour before the start; only checked-in entries play", Response: service.TournamentEntry{},
		Errors: []apierror.Code{apierror.NotFound, apierror.MatchStateInvalid},
	}))
	s.Handle("POST /v1/tournaments/{id}/play", a(t.play), doc(httpserver.Doc{
		Auth: "full", Summary: "Show up for the current bracket match; the match starts when both sides are present", Response: service.TournamentPlay{},
		Errors: []apierror.Code{apierror.NotFound, apierror.MatchStateInvalid, apierror.AlreadyInMatch, apierror.ModeNotAllowed, apierror.NotEnoughQuestions},
	}))
	s.Handle("POST /v1/admin/tournaments", a(t.idem.Wrap(idempotency.Optional, t.create), adm), doc(httpserver.Doc{
		Auth: "adm", Idempotency: true, Status: http.StatusCreated, Summary: "Create a tournament (draft or open for registration; D-031, D-090)",
		Request: service.TournamentInput{}, Response: service.Tournament{},
		Errors: append([]apierror.Code{apierror.ValidationFailed, apierror.ModeNotAllowed}, idem...),
	}))
	s.Handle("PATCH /v1/admin/tournaments/{id}", a(t.update, adm), doc(httpserver.Doc{
		Auth: "adm", Summary: "Publish a draft or cancel a tournament", Request: service.TournamentUpdate{}, Response: service.Tournament{},
		Errors: []apierror.Code{apierror.ValidationFailed, apierror.NotFound, apierror.MatchStateInvalid},
	}))
}

func principal(r *http.Request) identity.Principal {
	p, _ := identity.PrincipalFrom(r.Context())
	return p
}

func isStaff(r *http.Request) bool {
	p := principal(r)
	return p.HasRole(identity.RoleModerator) || p.HasRole(identity.RoleAdmin)
}

func locale(r *http.Request) string { return string(i18n.FromContext(r.Context())) }

func pathID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return uuid.Nil, apierror.New(apierror.NotFound)
	}
	return id, nil
}

func (t *Transport) list(w http.ResponseWriter, r *http.Request) error {
	var status *string
	if v := r.URL.Query().Get("status"); v != "" {
		status = &v
	}
	list, err := t.svc.List(r.Context(), principal(r).UserID, status, locale(r), isStaff(r))
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, Tournaments{Tournaments: list})
}

func (t *Transport) get(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	d, err := t.svc.Get(r.Context(), principal(r).UserID, id, locale(r), isStaff(r))
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, d)
}

func (t *Transport) register(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var in service.TournamentRegistration
	if r.ContentLength != 0 {
		if err := httpserver.DecodeJSON(r, &in); err != nil {
			return err
		}
	}
	p := principal(r)
	e, err := t.svc.Register(r.Context(), p.UserID, p.Tier == identity.TierFull, id, in)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusCreated, e)
}

func (t *Transport) withdraw(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	if err := t.svc.Withdraw(r.Context(), principal(r).UserID, id); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (t *Transport) checkIn(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	e, err := t.svc.CheckIn(r.Context(), principal(r).UserID, id)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, e)
}

func (t *Transport) play(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	res, err := t.svc.Play(r.Context(), principal(r).UserID, id)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, res)
}

func (t *Transport) create(w http.ResponseWriter, r *http.Request) error {
	var in service.TournamentInput
	if err := httpserver.DecodeJSON(r, &in); err != nil {
		return err
	}
	out, err := t.svc.Create(r.Context(), principal(r).UserID, in)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusCreated, out)
}

func (t *Transport) update(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var in service.TournamentUpdate
	if err := httpserver.DecodeJSON(r, &in); err != nil {
		return err
	}
	out, err := t.svc.Update(r.Context(), principal(r).UserID, id, in)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, out)
}
