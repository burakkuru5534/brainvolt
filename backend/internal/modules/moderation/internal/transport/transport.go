// Package transport exposes moderation's REST endpoints (rest.md §11).
package transport

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/moderation/internal/service"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/idempotency"
)

// Transport serves moderation's routes.
type Transport struct {
	svc   *service.Service
	ident *identity.Module
	idem  *idempotency.Store
}

// New returns the transport.
func New(svc *service.Service, ident *identity.Module, idem *idempotency.Store) *Transport {
	return &Transport{svc: svc, ident: ident, idem: idem}
}

// Sanctions is GET /v1/me/sanctions.
type Sanctions struct {
	Sanctions []service.Sanction `json:"sanctions"`
}

// AppealInput is the body of POST /v1/me/sanctions/{id}/appeal.
type AppealInput struct {
	Text string `json:"text"`
}

// Cases is GET /v1/admin/cases.
type Cases struct {
	Cases      []service.Case `json:"cases"`
	NextCursor *string        `json:"next_cursor"`
}

// Appeals is GET /v1/admin/appeals.
type Appeals struct {
	Appeals []service.AppealView `json:"appeals"`
}

// Decision is the body of POST /v1/admin/appeals/{id}/decide.
type Decision struct {
	Accept bool   `json:"accept"`
	Note   string `json:"note,omitempty"`
}

// Register adds moderation's routes; the admin routes need the
// moderator or admin role.
func (t *Transport) Register(s *httpserver.Server) {
	a := t.ident.Auth
	staff := identity.RequireRole(identity.RoleModerator, identity.RoleAdmin)
	doc := func(d httpserver.Doc) httpserver.RouteOption { d.Tag = "moderation"; return httpserver.WithDoc(d) }
	idem := []apierror.Code{apierror.IdempotencyKeyReused, apierror.IdempotencyInProgress}
	s.Handle("POST /v1/reports", a(t.idem.Wrap(idempotency.Required, t.report)), doc(httpserver.Doc{
		Auth: "any", Idempotency: true, Status: http.StatusCreated,
		Summary: "Report a question, translation, user, team or couple (one per target, 20 a day; D-030)",
		Request: service.ReportInput{}, Response: service.Filed{},
		Errors: append([]apierror.Code{apierror.ValidationFailed, apierror.ReportLimitReached, apierror.AlreadyReported}, idem...),
	}))
	s.Handle("GET /v1/me/sanctions", a(t.mySanctions), doc(httpserver.Doc{
		Auth: "any", Summary: "The caller's sanctions and warnings with their appeal state", Response: Sanctions{},
	}))
	s.Handle("POST /v1/me/sanctions/{id}/appeal", a(t.idem.Wrap(idempotency.Required, t.appeal), identity.AllowSuspended()), doc(httpserver.Doc{
		Auth: "any", Idempotency: true, Summary: "Appeal a sanction (one appeal per sanction; also while suspended)",
		Request: AppealInput{}, Errors: append([]apierror.Code{apierror.ValidationFailed, apierror.NotFound, apierror.AppealExists}, idem...),
	}))
	s.Handle("GET /v1/admin/cases", a(t.cases, staff), doc(httpserver.Doc{
		Auth: "mod", Summary: "Review queue, highest priority first (status=open, default, or resolved)", Response: Cases{},
		Query: []httpserver.QueryParam{
			{Name: "status", Type: "string", Enum: []string{"open", "resolved"}},
			{Name: "limit", Type: "integer", Description: "1–100, default 50"},
			{Name: "cursor", Type: "string"},
		},
	}))
	s.Handle("GET /v1/admin/cases/{id}", a(t.caseDetail, staff), doc(httpserver.Doc{
		Auth: "mod", Summary: "A case with its reports, cheat signals and the target's action history",
		Response: service.CaseDetail{}, Errors: []apierror.Code{apierror.NotFound},
	}))
	s.Handle("POST /v1/admin/cases/{id}/actions", a(t.idem.Wrap(idempotency.Required, t.act), staff), doc(httpserver.Doc{
		Auth: "mod", Idempotency: true,
		Summary: "Act on a case: case_dismiss, question_send_to_review, question_retire (admin), user_warn, user_sanction (permanent: admin), sanction_lift",
		Request: service.ActionInput{}, Response: service.CaseDetail{},
		Errors: append([]apierror.Code{apierror.ValidationFailed, apierror.NotFound, apierror.Forbidden, apierror.SelfReviewForbidden,
			apierror.MatchStateInvalid}, idem...),
	}))
	s.Handle("GET /v1/admin/appeals", a(t.appeals, staff), doc(httpserver.Doc{
		Auth: "mod", Summary: "Pending appeals, oldest first", Response: Appeals{},
	}))
	s.Handle("POST /v1/admin/appeals/{id}/decide", a(t.decide, staff), doc(httpserver.Doc{
		Auth: "mod", Summary: "Accept (lifts the sanction) or reject an appeal; not by the sanction's creator",
		Request: Decision{}, Errors: []apierror.Code{apierror.NotFound, apierror.SelfReviewForbidden, apierror.MatchStateInvalid},
	}))
}

func principal(r *http.Request) identity.Principal {
	p, _ := identity.PrincipalFrom(r.Context())
	return p
}

func actor(r *http.Request) service.Actor {
	p := principal(r)
	return service.Actor{ID: p.UserID, Admin: p.HasRole(identity.RoleAdmin)}
}

func pathID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return uuid.Nil, apierror.New(apierror.NotFound)
	}
	return id, nil
}

func (t *Transport) report(w http.ResponseWriter, r *http.Request) error {
	var in service.ReportInput
	if err := httpserver.DecodeJSON(r, &in); err != nil {
		return err
	}
	f, err := t.svc.Report(r.Context(), principal(r).UserID, in)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusCreated, f)
}

func (t *Transport) mySanctions(w http.ResponseWriter, r *http.Request) error {
	list, err := t.svc.MySanctions(r.Context(), principal(r).UserID)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, Sanctions{Sanctions: list})
}

func (t *Transport) appeal(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var in AppealInput
	if err := httpserver.DecodeJSON(r, &in); err != nil {
		return err
	}
	if err := t.svc.Appeal(r.Context(), principal(r).UserID, id, in.Text); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (t *Transport) cases(w http.ResponseWriter, r *http.Request) error {
	qs := r.URL.Query()
	status := qs.Get("status")
	if status != "" && status != "open" && status != "resolved" {
		return apierror.Validation(apierror.FieldError{Field: "status", Rule: "one_of"})
	}
	limit, offset := 50, 0
	if v := qs.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 100 {
			return apierror.Validation(apierror.FieldError{Field: "limit", Rule: "range"})
		}
		limit = n
	}
	if v := qs.Get("cursor"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 || n > 10_000 {
			return apierror.Validation(apierror.FieldError{Field: "cursor", Rule: "format"})
		}
		offset = n
	}
	list, err := t.svc.Cases(r.Context(), status == "resolved", limit+1, offset)
	if err != nil {
		return err
	}
	out := Cases{Cases: list}
	if len(list) > limit {
		out.Cases = list[:limit]
		next := strconv.Itoa(offset + limit)
		out.NextCursor = &next
	}
	return httpserver.WriteJSON(w, http.StatusOK, out)
}

func (t *Transport) caseDetail(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	c, err := t.svc.CaseDetail(r.Context(), id)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, c)
}

func (t *Transport) act(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var in service.ActionInput
	if err := httpserver.DecodeJSON(r, &in); err != nil {
		return err
	}
	c, err := t.svc.Act(r.Context(), actor(r), id, in)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, c)
}

func (t *Transport) appeals(w http.ResponseWriter, r *http.Request) error {
	list, err := t.svc.PendingAppeals(r.Context())
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, Appeals{Appeals: list})
}

func (t *Transport) decide(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var in Decision
	if err := httpserver.DecodeJSON(r, &in); err != nil {
		return err
	}
	if err := t.svc.DecideAppeal(r.Context(), actor(r), id, in.Accept, in.Note); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
