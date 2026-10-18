// Package transport is content's HTTP layer (docs/api/rest.md §2): the
// catalogue, credits and the moderator question workflow.
package transport

import (
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/content/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/content/internal/service"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/i18n"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/idempotency"
)

// cacheSeconds is the client cache time of static-ish data (conventions.md §9).
const cacheSeconds = "300"

// Transport wires content's handlers.
type Transport struct {
	svc   *service.Service
	ident *identity.Module
	idem  *idempotency.Store
}

// New returns the transport.
func New(svc *service.Service, ident *identity.Module, idem *idempotency.Store) *Transport {
	return &Transport{svc: svc, ident: ident, idem: idem}
}

// TextBody is a question's wording in one locale.
type TextBody struct {
	Prompt      string   `json:"prompt"`
	Explanation *string  `json:"explanation,omitempty"`
	Options     []string `json:"options"`
	Origin      string   `json:"origin"`
}

// QuestionRequest is the body of POST /v1/questions. For true/false
// questions options may be omitted (Doğru/Yanlış, True/False); positions:
// 1 = true, 2 = false.
type QuestionRequest struct {
	SubcategoryID   uuid.UUID           `json:"subcategory_id"`
	QuestionType    string              `json:"question_type"`
	Difficulty      int                 `json:"difficulty"`
	OriginalLocale  string              `json:"original_locale"`
	SourceID        *uuid.UUID          `json:"source_id"`
	SourceRef       *string             `json:"source_ref,omitempty"`
	CorrectPosition int                 `json:"correct_position"`
	Translations    map[string]TextBody `json:"translations"`
}

// QuestionPatch is the body of PATCH /v1/questions/{id}; a translation
// replaces the whole text of its locale.
type QuestionPatch struct {
	SubcategoryID   *uuid.UUID          `json:"subcategory_id,omitempty"`
	Difficulty      *int                `json:"difficulty,omitempty"`
	CorrectPosition *int                `json:"correct_position,omitempty"`
	Translations    map[string]TextBody `json:"translations,omitempty"`
}

// ReviewRequest is the body of POST /v1/questions/{id}/review; locale set =
// translation-level decision (D-007).
type ReviewRequest struct {
	Decision string  `json:"decision"`
	Locale   *string `json:"locale,omitempty"`
	Reason   *string `json:"reason,omitempty"`
}

// Credits is the body of GET /v1/credits.
type Credits struct {
	Sources []service.Credit `json:"sources"`
}

var statusEnum = []string{string(domain.StatusDraft), string(domain.StatusInReview), string(domain.StatusVerified), string(domain.StatusRejected)}

// Register adds content's routes.
func (t *Transport) Register(s *httpserver.Server) {
	a := t.ident.Auth
	mod := identity.RequireRole(identity.RoleModerator, identity.RoleAdmin)
	adm := identity.RequireRole(identity.RoleAdmin)
	staffErrors := []apierror.Code{apierror.Forbidden, apierror.AccountSuspended, apierror.AccountDeletionPending}

	s.Handle("GET /v1/categories", a(t.categories), httpserver.WithDoc(httpserver.Doc{
		Tag: "content", Auth: "any",
		Summary:  "Categories and sub-categories in the caller's language with eligible question counts (ranked / casual); ETag + If-None-Match",
		Response: service.Catalog{},
		Errors:   []apierror.Code{apierror.AccountSuspended, apierror.AccountDeletionPending},
	}))
	s.Handle("GET /v1/credits", t.credits, httpserver.WithDoc(httpserver.Doc{
		Tag: "content", Auth: "public",
		Summary:  "Content sources and attribution for the Credits screen (D-006, CC BY-SA)",
		Response: Credits{},
	}))
	s.Handle("GET /v1/questions", a(t.list, mod), httpserver.WithDoc(httpserver.Doc{
		Tag: "content", Auth: "mod", Summary: "Question listing for review tools, newest first",
		Response: service.QuestionPage{}, Errors: append([]apierror.Code{apierror.ValidationFailed}, staffErrors...),
		Query: []httpserver.QueryParam{
			{Name: "status", Type: "string", Enum: statusEnum, Description: "question status"},
			{Name: "translation_status", Type: "string", Enum: statusEnum, Description: "status of the `locale` translation (needs locale)"},
			{Name: "subcategory_id", Type: "string", Format: "uuid"},
			{Name: "locale", Type: "string", Enum: domain.Locales, Description: "only questions with this translation; its prompt is shown"},
			{Name: "include_retired", Type: "boolean"},
			{Name: "limit", Type: "integer", Description: "1–100, default 20"},
			{Name: "cursor", Type: "string", Description: "next_cursor of the previous page"},
		},
	}))
	s.Handle("GET /v1/questions/{id}", a(t.get, mod), httpserver.WithDoc(httpserver.Doc{
		Tag: "content", Auth: "mod", Summary: "Full question: translations, options with the correct one, revisions, events, stats",
		Response: service.Question{}, Errors: append([]apierror.Code{apierror.NotFound}, staffErrors...),
	}))
	s.Handle("POST /v1/questions", a(t.idem.Wrap(idempotency.Required, t.create), mod), httpserver.WithDoc(httpserver.Doc{
		Tag: "content", Auth: "mod", Idempotency: true, Summary: "Create a draft question with its source (D-006)",
		Request: QuestionRequest{}, Response: service.Question{}, Status: http.StatusCreated,
		Errors: append([]apierror.Code{apierror.ValidationFailed, apierror.QuestionSourceRequired, apierror.QuestionDuplicate,
			apierror.IdempotencyKeyReused, apierror.IdempotencyInProgress}, staffErrors...),
	}))
	s.Handle("PATCH /v1/questions/{id}", a(t.patch, mod), httpserver.WithDoc(httpserver.Doc{
		Tag: "content", Auth: "mod", Summary: "Edit a question: bumps the revision; a verified question goes back to review; the correct option of a used question is fixed",
		Request: QuestionPatch{}, Response: service.Question{},
		Errors: append([]apierror.Code{apierror.ValidationFailed, apierror.NotFound, apierror.QuestionInUse,
			apierror.QuestionStateInvalid, apierror.QuestionDuplicate}, staffErrors...),
	}))
	s.Handle("POST /v1/questions/{id}/submit", a(t.submit, mod), httpserver.WithDoc(httpserver.Doc{
		Tag: "content", Auth: "mod", Summary: "Submit a draft (and its draft translations) for review",
		Response: service.Question{}, Errors: append([]apierror.Code{apierror.NotFound, apierror.QuestionStateInvalid}, staffErrors...),
	}))
	s.Handle("POST /v1/questions/{id}/review", a(t.review, mod), httpserver.WithDoc(httpserver.Doc{
		Tag: "content", Auth: "mod", Summary: "Approve or reject a question (with its original wording) or one translation (D-007)",
		Request: ReviewRequest{}, Response: service.Question{},
		Errors: append([]apierror.Code{apierror.ValidationFailed, apierror.NotFound, apierror.QuestionStateInvalid}, staffErrors...),
	}))
	s.Handle("POST /v1/questions/{id}/retire", a(t.retire, adm), httpserver.WithDoc(httpserver.Doc{
		Tag: "content", Auth: "adm", Summary: "Retire a question: never selected again, kept for match history; repeating it is harmless",
		Response: service.Question{}, Errors: append([]apierror.Code{apierror.NotFound}, staffErrors...),
	}))
}

func actor(r *http.Request) uuid.UUID {
	p, _ := identity.PrincipalFrom(r.Context())
	return p.UserID
}

func (t *Transport) categories(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	locale := string(i18n.FromContext(ctx))
	etag, err := t.svc.CatalogETag(ctx, locale)
	if err != nil {
		return err
	}
	h := w.Header()
	h.Set("ETag", etag)
	h.Set("Cache-Control", "private, max-age="+cacheSeconds)
	h.Add("Vary", "Accept-Language")
	if match := r.Header.Get("If-None-Match"); match != "" && strings.Contains(match, etag) {
		w.WriteHeader(http.StatusNotModified)
		return nil
	}
	cat, err := t.svc.Catalog(ctx, locale)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, cat)
}

func (t *Transport) credits(w http.ResponseWriter, r *http.Request) error {
	credits, err := t.svc.Credits(r.Context())
	if err != nil {
		return err
	}
	w.Header().Set("Cache-Control", "public, max-age="+cacheSeconds)
	return httpserver.WriteJSON(w, http.StatusOK, Credits{Sources: credits})
}

func (t *Transport) list(w http.ResponseWriter, r *http.Request) error {
	f, err := listFilter(r)
	if err != nil {
		return err
	}
	page, err := t.svc.List(r.Context(), f)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, page)
}

func listFilter(r *http.Request) (service.ListFilter, error) {
	v := r.URL.Query()
	f := service.ListFilter{Limit: 20}
	var fe apierror.Fields
	status := func(name string) *domain.Status {
		s := v.Get(name)
		if s == "" {
			return nil
		}
		st := domain.Status(s)
		if !slices.Contains(statusEnum, s) {
			fe.Add(name, "enum", "allowed", statusEnum)
		}
		return &st
	}
	f.Status = status("status")
	f.TranslationStatus = status("translation_status")
	if s := v.Get("locale"); s != "" {
		if !domain.ValidLocale(s) {
			fe.Add("locale", "enum", "allowed", domain.Locales)
		}
		f.Locale = &s
	} else if f.TranslationStatus != nil {
		fe.Add("translation_status", "requires", "field", "locale")
	}
	if s := v.Get("subcategory_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			fe.Add("subcategory_id", "uuid")
		}
		f.SubcategoryID = &id
	}
	if s := v.Get("include_retired"); s != "" {
		b, err := strconv.ParseBool(s)
		if err != nil {
			fe.Add("include_retired", "boolean")
		}
		f.IncludeRetired = b
	}
	if s := v.Get("limit"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 || n > 100 {
			fe.Add("limit", "range", "min", 1, "max", 100)
		}
		f.Limit = n
	}
	if s := v.Get("cursor"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			fe.Add("cursor", "invalid")
		}
		f.Cursor = &id
	}
	return f, fe.Err()
}

func pathID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return uuid.Nil, apierror.New(apierror.NotFound)
	}
	return id, nil
}

// respond writes the question's current full view.
func (t *Transport) respond(w http.ResponseWriter, r *http.Request, id uuid.UUID, status int) error {
	q, err := t.svc.Get(r.Context(), id)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, status, q)
}

func (t *Transport) get(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	return t.respond(w, r, id, http.StatusOK)
}

func texts(in map[string]TextBody) map[string]domain.Text {
	out := make(map[string]domain.Text, len(in))
	for l, b := range in {
		t := domain.Text{Prompt: b.Prompt, Options: b.Options, Origin: domain.Origin(b.Origin)}
		if b.Explanation != nil {
			t.Explanation = *b.Explanation
		}
		out[l] = t
	}
	return out
}

func (t *Transport) create(w http.ResponseWriter, r *http.Request) error {
	var req QuestionRequest
	if err := httpserver.DecodeJSON(r, &req); err != nil {
		return err
	}
	d := domain.Draft{
		SubcategoryID: req.SubcategoryID, Type: domain.QuestionType(req.QuestionType), Difficulty: req.Difficulty,
		OriginalLocale: req.OriginalLocale, SourceID: req.SourceID, Correct: req.CorrectPosition,
		Texts: texts(req.Translations),
	}
	if req.SourceRef != nil {
		d.SourceRef = *req.SourceRef
	}
	me := actor(r)
	id, err := t.svc.Create(r.Context(), d, service.CreateOptions{Actor: &me, Status: domain.StatusDraft, AuthorUserID: &me})
	if err != nil {
		return err
	}
	return t.respond(w, r, id, http.StatusCreated)
}

func (t *Transport) patch(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var req QuestionPatch
	if err := httpserver.DecodeJSON(r, &req); err != nil {
		return err
	}
	p := service.Patch{
		SubcategoryID: req.SubcategoryID, Difficulty: req.Difficulty, Correct: req.CorrectPosition,
		Texts: texts(req.Translations),
	}
	if err := t.svc.Update(r.Context(), actor(r), id, p); err != nil {
		return err
	}
	return t.respond(w, r, id, http.StatusOK)
}

func (t *Transport) submit(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	if err := t.svc.Submit(r.Context(), actor(r), id); err != nil {
		return err
	}
	return t.respond(w, r, id, http.StatusOK)
}

func (t *Transport) review(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var req ReviewRequest
	if err := httpserver.DecodeJSON(r, &req); err != nil {
		return err
	}
	rv := service.Review{Decision: req.Decision, Locale: req.Locale}
	if req.Reason != nil {
		rv.Reason = strings.TrimSpace(*req.Reason)
	}
	if err := t.svc.Review(r.Context(), actor(r), id, rv); err != nil {
		return err
	}
	return t.respond(w, r, id, http.StatusOK)
}

func (t *Transport) retire(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	if err := t.svc.Retire(r.Context(), actor(r), id); err != nil {
		return err
	}
	return t.respond(w, r, id, http.StatusOK)
}
