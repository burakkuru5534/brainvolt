// Package transport exposes team's REST endpoints (rest.md §8).
package transport

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/team/internal/service"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/idempotency"
)

// Transport serves team's routes.
type Transport struct {
	svc   *service.Service
	ident *identity.Module
	idem  *idempotency.Store
}

// New returns the transport.
func New(svc *service.Service, ident *identity.Module, idem *idempotency.Store) *Transport {
	return &Transport{svc: svc, ident: ident, idem: idem}
}

// Teams is GET /v1/me/teams.
type Teams struct {
	Teams []service.Team `json:"teams"`
}

// Invites is GET /v1/me/team-invites.
type Invites struct {
	Invites []service.Invite `json:"invites"`
}

// InviteRequest is the body of POST /v1/teams/{id}/invites.
type InviteRequest struct {
	UserID uuid.UUID `json:"user_id"`
}

// InviteCreated is the answer to POST /v1/teams/{id}/invites.
type InviteCreated struct {
	ID uuid.UUID `json:"id"`
}

// MyCouple is GET /v1/me/couple.
type MyCouple struct {
	Couple *service.Couple `json:"couple"`
}

var (
	teamErrors   = []apierror.Code{apierror.NotFound, apierror.NotCaptain}
	upgrade      = []apierror.Code{apierror.AccountUpgradeRequired}
	createErrors = []apierror.Code{apierror.ValidationFailed, apierror.TextRejected, apierror.TeamLimitReached, apierror.AccountUpgradeRequired,
		apierror.IdempotencyKeyReused, apierror.IdempotencyInProgress}
)

// Register adds team's routes. Teams and couples need a full account
// (D-016a); reading a team is open to everyone.
func (t *Transport) Register(s *httpserver.Server) {
	full := identity.RequireFull("teams")
	doc := func(d httpserver.Doc) httpserver.RouteOption { d.Tag = "team"; return httpserver.WithDoc(d) }
	s.Handle("POST /v1/teams", t.ident.Auth(t.idem.Wrap(idempotency.Required, t.create), full), doc(httpserver.Doc{
		Auth: "full", Idempotency: true, Status: http.StatusCreated, Summary: "Create a team (the caller is captain; max 3 active teams)",
		Request: service.TeamInput{}, Response: service.Team{}, Errors: createErrors,
	}))
	s.Handle("GET /v1/teams/{id}", t.ident.Auth(t.get), doc(httpserver.Doc{
		Auth: "any", Summary: "Team profile and members", Response: service.Team{}, Errors: []apierror.Code{apierror.NotFound},
	}))
	s.Handle("PATCH /v1/teams/{id}", t.ident.Auth(t.update, full), doc(httpserver.Doc{
		Auth: "captain", Summary: "Edit name, tag or emblem", Request: service.TeamInput{}, Response: service.Team{},
		Errors: append([]apierror.Code{apierror.ValidationFailed, apierror.TextRejected}, teamErrors...),
	}))
	s.Handle("POST /v1/teams/{id}/invites", t.ident.Auth(t.idem.Wrap(idempotency.Required, t.invite), full), doc(httpserver.Doc{
		Auth: "member", Idempotency: true, Status: http.StatusCreated, Summary: "Invite a friend (full account) to the team; 7 days",
		Request: InviteRequest{}, Response: InviteCreated{},
		Errors: append([]apierror.Code{apierror.ValidationFailed, apierror.Forbidden, apierror.TeamFull, apierror.RequestPending,
			apierror.AccountUpgradeRequired, apierror.IdempotencyKeyReused, apierror.IdempotencyInProgress}, teamErrors...),
	}))
	s.Handle("GET /v1/me/team-invites", t.ident.Auth(t.invites, full), doc(httpserver.Doc{
		Auth: "full", Summary: "Pending team invites", Response: Invites{}, Errors: upgrade,
	}))
	for _, action := range []string{"accept", "decline"} {
		s.Handle("POST /v1/team-invites/{id}/"+action, t.ident.Auth(t.respondInvite(action == "accept"), full), doc(httpserver.Doc{
			Auth: "full", Summary: "Team invite: " + action,
			Errors: []apierror.Code{apierror.NotFound, apierror.TeamFull, apierror.TeamLimitReached, apierror.Blocked,
				apierror.ChallengeExpired, apierror.MatchStateInvalid, apierror.AccountUpgradeRequired},
		}))
	}
	s.Handle("POST /v1/teams/{id}/leave", t.ident.Auth(t.leave, full), doc(httpserver.Doc{
		Auth: "member", Summary: "Leave (a leaving captain hands over to the longest-standing member; the last member disbands)", Errors: teamErrors,
	}))
	s.Handle("POST /v1/teams/{id}/kick/{user_id}", t.ident.Auth(t.kick, full), doc(httpserver.Doc{
		Auth: "captain", Summary: "Remove a member", Errors: append([]apierror.Code{apierror.ValidationFailed}, teamErrors...),
	}))
	s.Handle("POST /v1/teams/{id}/captain/{user_id}", t.ident.Auth(t.captain, full), doc(httpserver.Doc{
		Auth: "captain", Summary: "Hand the captaincy to a member", Errors: teamErrors,
	}))
	s.Handle("POST /v1/teams/{id}/disband", t.ident.Auth(t.disband, full), doc(httpserver.Doc{
		Auth: "captain", Summary: "Disband the team", Errors: teamErrors,
	}))
	s.Handle("GET /v1/me/teams", t.ident.Auth(t.myTeams), doc(httpserver.Doc{
		Auth: "any", Summary: "The caller's active teams", Response: Teams{},
	}))

	coupleFull := identity.RequireFull("couples")
	s.Handle("POST /v1/couples", t.ident.Auth(t.idem.Wrap(idempotency.Required, t.requestCouple), coupleFull), doc(httpserver.Doc{
		Auth: "full", Idempotency: true, Status: http.StatusCreated,
		Summary: "Ask a friend to form a couple (the same pair reactivates its profile; 7-day cooldown for a new partner)",
		Request: service.CoupleRequest{}, Response: service.Couple{},
		Errors: []apierror.Code{apierror.ValidationFailed, apierror.TextRejected, apierror.Forbidden, apierror.Blocked, apierror.CoupleExists,
			apierror.CoupleCooldown, apierror.AccountUpgradeRequired, apierror.NotFound, apierror.IdempotencyKeyReused, apierror.IdempotencyInProgress},
	}))
	for _, action := range []string{service.CoupleAccept, service.CoupleDecline, service.CoupleDissolve} {
		s.Handle("POST /v1/couples/{id}/"+action, t.ident.Auth(t.respondCouple(action), coupleFull), doc(httpserver.Doc{
			Auth: "partner", Summary: "Couple: " + action, Response: service.Couple{},
			Errors: []apierror.Code{apierror.NotFound, apierror.ChallengeExpired, apierror.MatchStateInvalid, apierror.AccountUpgradeRequired},
		}))
	}
	s.Handle("GET /v1/couples/{id}", t.ident.Auth(t.getCouple), doc(httpserver.Doc{
		Auth: "any", Summary: "Couple profile", Response: service.Couple{}, Errors: []apierror.Code{apierror.NotFound},
	}))
	s.Handle("GET /v1/me/couple", t.ident.Auth(t.myCouple), doc(httpserver.Doc{
		Auth: "any", Summary: "The caller's pending or active couple (null when none)", Response: MyCouple{},
	}))
}

func me(r *http.Request) uuid.UUID {
	p, _ := identity.PrincipalFrom(r.Context())
	return p.UserID
}

func pathID(r *http.Request, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(r.PathValue(name))
	if err != nil {
		return uuid.Nil, apierror.New(apierror.NotFound)
	}
	return id, nil
}

func noContent(w http.ResponseWriter, err error) error {
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (t *Transport) create(w http.ResponseWriter, r *http.Request) error {
	var in service.TeamInput
	if err := httpserver.DecodeJSON(r, &in); err != nil {
		return err
	}
	team, err := t.svc.CreateTeam(r.Context(), me(r), in)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusCreated, team)
}

func (t *Transport) get(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	team, err := t.svc.Team(r.Context(), id)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, team)
}

func (t *Transport) update(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	var in service.TeamInput
	if err := httpserver.DecodeJSON(r, &in); err != nil {
		return err
	}
	team, err := t.svc.UpdateTeam(r.Context(), me(r), id, in)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, team)
}

func (t *Transport) invite(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	var in InviteRequest
	if err := httpserver.DecodeJSON(r, &in); err != nil {
		return err
	}
	inv, err := t.svc.InviteToTeam(r.Context(), me(r), id, in.UserID)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusCreated, InviteCreated{ID: inv})
}

func (t *Transport) invites(w http.ResponseWriter, r *http.Request) error {
	list, err := t.svc.Invites(r.Context(), me(r))
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, Invites{Invites: list})
}

func (t *Transport) respondInvite(accept bool) httpserver.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		id, err := pathID(r, "id")
		if err != nil {
			return err
		}
		return noContent(w, t.svc.RespondInvite(r.Context(), me(r), id, accept))
	}
}

func (t *Transport) leave(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	return noContent(w, t.svc.LeaveTeam(r.Context(), me(r), id))
}

func (t *Transport) kick(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	user, err := pathID(r, "user_id")
	if err != nil {
		return err
	}
	return noContent(w, t.svc.Kick(r.Context(), me(r), id, user))
}

func (t *Transport) captain(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	user, err := pathID(r, "user_id")
	if err != nil {
		return err
	}
	return noContent(w, t.svc.TransferCaptain(r.Context(), me(r), id, user))
}

func (t *Transport) disband(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	return noContent(w, t.svc.Disband(r.Context(), me(r), id))
}

func (t *Transport) myTeams(w http.ResponseWriter, r *http.Request) error {
	list, err := t.svc.MyTeams(r.Context(), me(r))
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, Teams{Teams: list})
}

func (t *Transport) requestCouple(w http.ResponseWriter, r *http.Request) error {
	var in service.CoupleRequest
	if err := httpserver.DecodeJSON(r, &in); err != nil {
		return err
	}
	c, err := t.svc.RequestCouple(r.Context(), me(r), in)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusCreated, c)
}

func (t *Transport) respondCouple(action string) httpserver.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		id, err := pathID(r, "id")
		if err != nil {
			return err
		}
		c, err := t.svc.RespondCouple(r.Context(), me(r), id, action)
		if err != nil {
			return err
		}
		return httpserver.WriteJSON(w, http.StatusOK, c)
	}
}

func (t *Transport) getCouple(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	c, err := t.svc.Couple(r.Context(), id)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, c)
}

func (t *Transport) myCouple(w http.ResponseWriter, r *http.Request) error {
	c, err := t.svc.MyCouple(r.Context(), me(r))
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, MyCouple{Couple: c})
}
