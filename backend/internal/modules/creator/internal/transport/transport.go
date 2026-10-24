// Package transport exposes creator's REST endpoints (rest.md §16).
package transport

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/creator/internal/service"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/idempotency"
)

// Transport serves creator's routes.
type Transport struct {
	svc   *service.Service
	ident *identity.Module
	idem  *idempotency.Store
}

// New returns the transport.
func New(svc *service.Service, ident *identity.Module, idem *idempotency.Store) *Transport {
	return &Transport{svc: svc, ident: ident, idem: idem}
}

// Quizzes is a list of quizzes.
type Quizzes struct {
	Quizzes    []service.Quiz `json:"quizzes"`
	NextCursor *string        `json:"next_cursor,omitempty"`
}

// ReviewInput is the body of POST /v1/admin/quizzes/{id}/review.
type ReviewInput struct {
	Approve bool   `json:"approve"`
	Reason  string `json:"reason,omitempty"`
}

// Register adds creator's routes.
func (t *Transport) Register(s *httpserver.Server) {
	a := t.ident.Auth
	staff := identity.RequireRole(identity.RoleModerator, identity.RoleAdmin)
	doc := func(d httpserver.Doc) httpserver.RouteOption { d.Tag = "creator"; return httpserver.WithDoc(d) }
	idem := []apierror.Code{apierror.IdempotencyKeyReused, apierror.IdempotencyInProgress}
	s.Handle("POST /v1/creator/quizzes", a(t.idem.Wrap(idempotency.Required, t.create)), doc(httpserver.Doc{
		Auth: "full", Idempotency: true, Status: http.StatusCreated,
		Summary: "Create a draft quiz with its questions (1–30, 20 quizzes per author; D-031)",
		Request: service.QuizInput{}, Response: service.Quiz{},
		Errors: append([]apierror.Code{apierror.ValidationFailed, apierror.AccountUpgradeRequired, apierror.QuizLimitReached,
			apierror.QuizSizeInvalid}, idem...),
	}))
	s.Handle("GET /v1/creator/quizzes", a(t.discover), doc(httpserver.Doc{
		Auth: "any", Summary: "Published quizzes, most liked first", Response: Quizzes{},
		Query: []httpserver.QueryParam{
			{Name: "locale", Type: "string", Enum: []string{"tr", "en"}},
			{Name: "category_id", Type: "string", Format: "uuid"},
			{Name: "limit", Type: "integer", Description: "1–50, default 20"},
			{Name: "cursor", Type: "string"},
		},
	}))
	s.Handle("GET /v1/me/quizzes", a(t.mine), doc(httpserver.Doc{
		Auth: "any", Summary: "The caller's quizzes", Response: Quizzes{},
	}))
	s.Handle("GET /v1/creator/quizzes/{id}", a(t.get), doc(httpserver.Doc{
		Auth: "any", Summary: "A quiz; its questions and answers only for the author", Response: service.Quiz{},
		Errors: []apierror.Code{apierror.NotFound},
	}))
	s.Handle("PATCH /v1/creator/quizzes/{id}", a(t.update), doc(httpserver.Doc{
		Auth: "author", Summary: "Replace a quiz's details and questions (If-Match: version); a published quiz goes back to review",
		Request: service.QuizInput{}, Response: service.Quiz{},
		Errors: []apierror.Code{apierror.ValidationFailed, apierror.NotFound, apierror.QuizSizeInvalid, apierror.VersionConflict},
	}))
	s.Handle("DELETE /v1/creator/quizzes/{id}", a(t.delete), doc(httpserver.Doc{
		Auth: "author", Summary: "Delete a quiz", Errors: []apierror.Code{apierror.NotFound},
	}))
	s.Handle("POST /v1/creator/quizzes/{id}/submit", a(t.submit), doc(httpserver.Doc{
		Auth: "author", Summary: "Send a quiz to review", Response: service.Quiz{},
		Errors: []apierror.Code{apierror.NotFound, apierror.MatchStateInvalid},
	}))
	for _, like := range []bool{true, false} {
		method, summary := "POST", "Like a published quiz"
		if !like {
			method, summary = "DELETE", "Remove a like"
		}
		s.Handle(method+" /v1/creator/quizzes/{id}/like", a(t.like(like)), doc(httpserver.Doc{
			Auth: "full", Summary: summary, Errors: []apierror.Code{apierror.NotFound, apierror.AccountUpgradeRequired},
		}))
	}
	s.Handle("GET /v1/admin/quizzes", a(t.submitted, staff), doc(httpserver.Doc{
		Auth: "mod", Summary: "Quizzes waiting for review, oldest first", Response: Quizzes{},
	}))
	s.Handle("POST /v1/admin/quizzes/{id}/review", a(t.review, staff), doc(httpserver.Doc{
		Auth: "mod", Summary: "Publish or reject a submitted quiz (not one's own)", Request: ReviewInput{}, Response: service.Quiz{},
		Errors: []apierror.Code{apierror.ValidationFailed, apierror.NotFound, apierror.SelfReviewForbidden, apierror.MatchStateInvalid},
	}))
}

func me(r *http.Request) uuid.UUID {
	p, _ := identity.PrincipalFrom(r.Context())
	return p.UserID
}

func isStaff(r *http.Request) bool {
	p, _ := identity.PrincipalFrom(r.Context())
	return p.HasRole(identity.RoleModerator) || p.HasRole(identity.RoleAdmin)
}

func pathID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return uuid.Nil, apierror.New(apierror.NotFound)
	}
	return id, nil
}

func (t *Transport) create(w http.ResponseWriter, r *http.Request) error {
	var in service.QuizInput
	if err := httpserver.DecodeJSON(r, &in); err != nil {
		return err
	}
	z, err := t.svc.Create(r.Context(), me(r), in)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusCreated, z)
}

func (t *Transport) discover(w http.ResponseWriter, r *http.Request) error {
	qs := r.URL.Query()
	var locale *string
	if v := qs.Get("locale"); v != "" {
		if v != "tr" && v != "en" {
			return apierror.Validation(apierror.FieldError{Field: "locale", Rule: "one_of"})
		}
		locale = &v
	}
	var category *uuid.UUID
	if v := qs.Get("category_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return apierror.Validation(apierror.FieldError{Field: "category_id", Rule: "format"})
		}
		category = &id
	}
	limit, offset := 20, 0
	if v := qs.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 50 {
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
	list, err := t.svc.Published(r.Context(), me(r), locale, category, limit+1, offset)
	if err != nil {
		return err
	}
	out := Quizzes{Quizzes: list}
	if len(list) > limit {
		out.Quizzes = list[:limit]
		next := strconv.Itoa(offset + limit)
		out.NextCursor = &next
	}
	return httpserver.WriteJSON(w, http.StatusOK, out)
}

func (t *Transport) mine(w http.ResponseWriter, r *http.Request) error {
	list, err := t.svc.Mine(r.Context(), me(r))
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, Quizzes{Quizzes: list})
}

func (t *Transport) get(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	z, err := t.svc.Get(r.Context(), me(r), id, isStaff(r))
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, z)
}

func (t *Transport) update(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var version *int32
	if h := strings.Trim(r.Header.Get("If-Match"), `"W/ `); h != "" {
		v, err := strconv.ParseInt(h, 10, 32)
		if err != nil {
			return apierror.Validation(apierror.FieldError{Field: "If-Match", Rule: "format"})
		}
		v32 := int32(v)
		version = &v32
	}
	var in service.QuizInput
	if err := httpserver.DecodeJSON(r, &in); err != nil {
		return err
	}
	z, err := t.svc.Update(r.Context(), me(r), id, version, in)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, z)
}

func (t *Transport) delete(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	if err := t.svc.Delete(r.Context(), me(r), id); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (t *Transport) submit(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	z, err := t.svc.Submit(r.Context(), me(r), id)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, z)
}

func (t *Transport) like(like bool) httpserver.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		if err := t.svc.Like(r.Context(), me(r), id, like); err != nil {
			return err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	}
}

func (t *Transport) submitted(w http.ResponseWriter, r *http.Request) error {
	list, err := t.svc.Submitted(r.Context(), me(r))
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, Quizzes{Quizzes: list})
}

func (t *Transport) review(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var in ReviewInput
	if err := httpserver.DecodeJSON(r, &in); err != nil {
		return err
	}
	z, err := t.svc.Review(r.Context(), me(r), id, in.Approve, in.Reason)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, z)
}
