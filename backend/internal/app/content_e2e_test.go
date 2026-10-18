package app

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/contentfiles"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/content"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
)

// TestContentFlows imports the repository's content/ files the way
// `seed import -approve-as` does, then drives the Phase 2 endpoints: the
// catalogue with ETag, credits, role checks and the moderator workflow.
func TestContentFlows(t *testing.T) {
	x := newE2E(t)
	ctx := t.Context()
	token, uid := x.emu.signUpAnonymous()
	r := x.session(token, nil)
	want(t, "session", r, 200, "")
	userID := r.get("user", "id").(string)

	r = x.do(http.MethodGet, "/v1/categories", token, nil)
	want(t, "empty catalogue", r, 200, "")
	if n := len(r.get("categories").([]any)); n != 0 {
		t.Fatalf("categories before import = %d", n)
	}

	// --- seed import (dev approval) ---------------------------------------
	bundle, err := contentfiles.Load(filepath.Join("..", "..", "..", "content"))
	if err != nil {
		t.Fatal(err)
	}
	mod := content.New(x.db, clock.Real{}, events.NewBus())
	var out bytes.Buffer
	if err := importBundle(ctx, mod, bundle, uuid.MustParse(userID), &out, discard); err != nil {
		t.Fatal(err)
	}
	if want := fmt.Sprintf("imported: %d new, 0 already present, %d approved (dev)\n", len(bundle.Questions), len(bundle.Questions)); out.String() != want {
		t.Fatalf("import output %q, want %q", out.String(), want)
	}
	out.Reset()
	if err := importBundle(ctx, mod, bundle, uuid.Nil, &out, discard); err != nil || !strings.Contains(out.String(), "0 new") {
		t.Fatalf("second import must skip everything: %q, %v", out.String(), err)
	}

	// --- catalogue ----------------------------------------------------------
	r = x.do(http.MethodGet, "/v1/categories", token, nil)
	want(t, "catalogue", r, 200, "")
	cats := r.get("categories").([]any)
	if len(cats) != len(bundle.Categories) || r.get("locale") != "en" {
		t.Fatalf("catalogue = %s", r.raw)
	}
	first := cats[0].(map[string]any)
	if first["name"] != "History" || first["eligible"].(map[string]any)["casual"].(float64) < 1 {
		t.Fatalf("first category = %v", first)
	}
	subID := first["subcategories"].([]any)[0].(map[string]any)["id"].(string)
	etag := x.lastHeader(http.MethodGet, "/v1/categories", token, "ETag")
	if r := x.do(http.MethodGet, "/v1/categories", token, nil, "If-None-Match", etag); r.code != http.StatusNotModified {
		t.Fatalf("If-None-Match: %d", r.code)
	}
	if r := x.do(http.MethodGet, "/v1/categories", token, nil, "If-None-Match", etag, "Accept-Language", "tr"); r.code != 200 || r.get("categories").([]any)[0].(map[string]any)["name"] != "Tarih" {
		t.Fatalf("tr catalogue has its own ETag: %d %s", r.code, r.raw)
	}

	r = x.do(http.MethodGet, "/v1/credits", "", nil)
	want(t, "credits (public)", r, 200, "")
	if n := len(r.get("sources").([]any)); n != len(bundle.Sources) {
		t.Fatalf("credits = %s", r.raw)
	}

	// --- roles ----------------------------------------------------------------
	want(t, "player lists questions", x.do(http.MethodGet, "/v1/questions", token, nil), 403, "FORBIDDEN")
	ident := identity.New(identity.Deps{DB: x.db, Redis: x.rdb, Events: events.NewBus(), Clock: clock.Real{}, Logger: discard})
	if _, err := ident.SetRole(ctx, uid, identity.RoleModerator, true); err != nil { // by Firebase UID
		t.Fatal(err)
	}
	if _, err := ident.SetRole(ctx, uid, "superuser", true); err == nil {
		t.Fatal("unknown role accepted")
	}

	r = x.do(http.MethodGet, "/v1/questions?limit=5&status=verified", token, nil)
	want(t, "moderator lists questions", r, 200, "")
	if len(r.get("items").([]any)) != 5 || r.get("next_cursor") == nil {
		t.Fatalf("page = %s", r.raw)
	}
	r = x.do(http.MethodGet, "/v1/questions?limit=500&status=done&translation_status=verified", token, nil)
	want(t, "invalid listing query", r, 400, "VALIDATION_FAILED")
	if n := len(r.get("error", "details", "fields").([]any)); n != 3 {
		t.Fatalf("want 3 field errors (limit, status, translation_status without locale): %s", r.raw)
	}
	want(t, "unknown question", x.do(http.MethodGet, "/v1/questions/"+ids.NewString(), token, nil), 404, "NOT_FOUND")

	// --- moderator workflow -------------------------------------------------
	body := map[string]any{
		"subcategory_id": subID, "question_type": "multiple_choice", "difficulty": 2, "original_locale": "tr",
		"source_id": bundle.Sources[0].ID.String(), "correct_position": 2,
		"translations": map[string]any{
			"tr": map[string]any{"prompt": "Kurtuluş Savaşı hangi antlaşmayla sona erdi?", "options": []string{"Sevr", "Lozan", "Mondros", "Ankara"}, "origin": "original"},
		},
	}
	want(t, "create without Idempotency-Key", x.do(http.MethodPost, "/v1/questions", token, body), 400, "VALIDATION_FAILED")
	noSource := map[string]any{}
	for k, v := range body {
		noSource[k] = v
	}
	delete(noSource, "source_id")
	want(t, "create without source (D-006)", x.do(http.MethodPost, "/v1/questions", token, noSource, "Idempotency-Key", ids.NewString()), 422, "QUESTION_SOURCE_REQUIRED")
	r = x.do(http.MethodPost, "/v1/questions", token, body, "Idempotency-Key", ids.NewString())
	want(t, "create", r, 201, "")
	qid := r.get("id").(string)
	if r.get("status") != "draft" || r.get("options").([]any)[1].(map[string]any)["is_correct"] != true || r.get("author_user_id") != userID {
		t.Fatalf("created = %s", r.raw)
	}
	r = x.do(http.MethodPost, "/v1/questions", token, body, "Idempotency-Key", ids.NewString())
	want(t, "duplicate", r, 409, "QUESTION_DUPLICATE")
	if r.get("error", "details", "question_id") != qid {
		t.Fatalf("duplicate details: %s", r.raw)
	}

	r = x.do(http.MethodPatch, "/v1/questions/"+qid, token, map[string]any{"translations": map[string]any{
		"en": map[string]any{"prompt": "Which treaty ended the Turkish War of Independence?", "options": []string{"Sèvres", "Lausanne", "Mudros", "Ankara"}, "origin": "human_translation"},
	}})
	want(t, "add translation", r, 200, "")
	if r.get("revision").(float64) != 2 || r.get("translations", "en", "status") != "draft" {
		t.Fatalf("patched = %s", r.raw)
	}
	want(t, "review a draft", x.do(http.MethodPost, "/v1/questions/"+qid+"/review", token, map[string]any{"decision": "approve"}), 409, "QUESTION_STATE_INVALID")
	want(t, "submit", x.do(http.MethodPost, "/v1/questions/"+qid+"/submit", token, nil), 200, "")
	r = x.do(http.MethodPost, "/v1/questions/"+qid+"/review", token, map[string]any{"decision": "approve"})
	want(t, "approve", r, 200, "")
	if r.get("status") != "verified" || r.get("translations", "tr", "status") != "verified" || r.get("translations", "en", "status") != "in_review" {
		t.Fatalf("approved = %s", r.raw)
	}
	r = x.do(http.MethodPost, "/v1/questions/"+qid+"/review", token, map[string]any{"decision": "reject", "locale": "en", "reason": "wrong treaty name"})
	want(t, "reject translation", r, 200, "")
	if r.get("translations", "en", "status") != "rejected" || r.get("status") != "verified" {
		t.Fatalf("translation rejected = %s", r.raw)
	}

	want(t, "moderator retires", x.do(http.MethodPost, "/v1/questions/"+qid+"/retire", token, nil), 403, "FORBIDDEN")
	if _, err := ident.SetRole(ctx, userID, identity.RoleAdmin, true); err != nil { // by user id
		t.Fatal(err)
	}
	r = x.do(http.MethodPost, "/v1/questions/"+qid+"/retire", token, nil)
	want(t, "admin retires", r, 200, "")
	if r.get("retired_at") == nil {
		t.Fatalf("retired = %s", r.raw)
	}
	if _, err := ident.SetRole(ctx, userID, identity.RoleAdmin, false); err != nil {
		t.Fatal(err)
	}
	want(t, "revoked admin", x.do(http.MethodPost, "/v1/questions/"+qid+"/retire", token, nil), 403, "FORBIDDEN")
}

// lastHeader performs a request and returns one response header.
func (x *e2e) lastHeader(method, path, token, header string) string {
	x.t.Helper()
	req, _ := http.NewRequestWithContext(x.t.Context(), method, path, http.NoBody)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept-Language", "en")
	rec := httptest.NewRecorder()
	x.srv.Handler().ServeHTTP(rec, req)
	return rec.Header().Get(header)
}
