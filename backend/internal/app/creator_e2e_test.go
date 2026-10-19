package app

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
)

// quizBody is a quiz of n multiple-choice questions in category cat.
func quizBody(cat string, n int) map[string]any {
	qs := make([]map[string]any, n)
	for i := range qs {
		qs[i] = map[string]any{
			"type": "multiple_choice", "prompt": fmt.Sprintf("Creator question number %d?", i+1),
			"options": []string{"Alpha", "Beta", "Gamma", "Delta"}, "correct": 1 + i%4,
		}
	}
	return map[string]any{"title": "My first quiz", "locale": "en", "category_id": cat, "answer_window_ms": 20_000, "questions": qs}
}

// Creator Quiz (D-023, D-031, D-044): authoring, review, discovery,
// likes, quiz rooms (unpublished: host's friends only), challenges
// (published only) and awards (author 0 XP, unpublished nothing).
func TestCreatorQuiz(t *testing.T) {
	x := newE2E(t)
	tokA, a, _ := x.fullUser()
	tokF, f, _ := x.fullUser()
	tokS, s, _ := x.fullUser()
	tokM, m, _ := x.fullUser()
	guest, _ := x.signedIn()
	x.seedQuestions(a)
	if _, err := x.mods.identity.SetRole(t.Context(), m.String(), identity.RoleModerator, true); err != nil {
		t.Fatal(err)
	}
	x.befriend(tokA, f, tokF)
	var cat string
	if err := x.db.QueryRow(t.Context(), `SELECT id FROM content.categories ORDER BY sort_order LIMIT 1`).Scan(&cat); err != nil {
		t.Fatal(err)
	}

	// Authoring rules.
	want(t, "guest", x.post("/v1/creator/quizzes", guest, quizBody(cat, 5)), 403, "ACCOUNT_UPGRADE_REQUIRED")
	want(t, "too short", x.post("/v1/creator/quizzes", tokA, quizBody(cat, 4)), 422, "QUIZ_SIZE_INVALID")
	bad := quizBody(cat, 5)
	bad["answer_window_ms"] = 12_000
	want(t, "window", x.post("/v1/creator/quizzes", tokA, bad), 400, "VALIDATION_FAILED")
	bad = quizBody(cat, 5)
	bad["questions"].([]map[string]any)[2]["options"] = []string{"Only one"}
	r := x.post("/v1/creator/quizzes", tokA, bad)
	if r.code != 400 || r.get("error", "details", "question") != 3.0 {
		t.Fatalf("bad question %s", r.raw)
	}
	r = x.post("/v1/creator/quizzes", tokA, quizBody(cat, 5))
	want(t, "create", r, 201, "")
	quiz := str(r.get("id"))
	if r.get("status") != "draft" || r.get("question_count") != 5.0 || len(r.get("questions").([]any)) != 5 ||
		r.get("questions").([]any)[1].(map[string]any)["correct"] != 2.0 {
		t.Fatalf("quiz %s", r.raw)
	}
	want(t, "draft to others", x.do(http.MethodGet, "/v1/creator/quizzes/"+quiz, tokS, nil), 404, "NOT_FOUND")
	if r := x.do(http.MethodGet, "/v1/me/quizzes", tokA, nil); len(r.get("quizzes").([]any)) != 1 {
		t.Fatalf("mine %s", r.raw)
	}

	// An unpublished quiz plays in a room of the host's friends only.
	cfg := map[string]any{"match_type": "quick_duel", "format": "1v1", "category_mode": "creator_quiz", "creator_quiz_id": quiz}
	want(t, "no quiz id", x.post("/v1/rooms", tokA, map[string]any{"match_config": map[string]any{
		"match_type": "quick_duel", "format": "1v1", "category_mode": "creator_quiz"}}), 400, "VALIDATION_FAILED")
	want(t, "other's draft", x.post("/v1/rooms", tokS, map[string]any{"match_config": cfg}), 404, "NOT_FOUND")
	want(t, "draft challenge", x.post("/v1/challenges", tokA, map[string]any{"target_user_id": f, "match_config": cfg}), 422, "MODE_NOT_ALLOWED")
	r = x.post("/v1/rooms", tokA, map[string]any{"match_config": cfg})
	want(t, "quiz room", r, 201, "")
	roomID, code := str(r.get("id")), str(r.get("code"))
	if r.get("friends_of_host_only") != true || r.get("match_config", "quiz_length") != 5.0 || r.get("match_config", "quiz_window_ms") != 20_000.0 ||
		r.get("match_config", "question_set_id") == nil {
		t.Fatalf("quiz room %s", r.raw)
	}
	want(t, "stranger joins", x.do(http.MethodPost, "/v1/rooms/join", tokS, map[string]any{"code": code}), 403, "ROOM_FRIENDS_ONLY")
	want(t, "friend joins", x.do(http.MethodPost, "/v1/rooms/join", tokF, map[string]any{"code": code}), 200, "")
	wa, _ := x.auth(tokA, "inst-a", nil)
	wf, _ := x.auth(tokF, "inst-f", nil)
	wf.answer(wf.send("room.ready", map[string]any{"room_id": roomID}))
	if fr := wa.answer(wa.send("room.start", map[string]any{"room_id": roomID})); fr["t"] != "sys.ack" {
		t.Fatalf("start %v", fr)
	}
	matchID := str(data(wf.next("room.match_starting"))["match_id"])
	var count int
	var mode string
	var locales []string
	if err := x.db.QueryRow(t.Context(), `SELECT question_count, category_mode, locales FROM game.matches WHERE id = $1`, matchID).Scan(&count, &mode, &locales); err != nil ||
		count != 5 || mode != "creator_quiz" || len(locales) != 1 || locales[0] != "en" {
		t.Fatalf("match %d %q %v %v", count, mode, locales, err)
	}
	leaveMatch(t, matchID, wa, wf)

	// Review: submit, no self review, publish.
	want(t, "stale edit", x.do(http.MethodPatch, "/v1/creator/quizzes/"+quiz, tokA, quizBody(cat, 6), "If-Match", "7"), 409, "VERSION_CONFLICT")
	r = x.do(http.MethodPatch, "/v1/creator/quizzes/"+quiz, tokA, quizBody(cat, 6), "If-Match", "1")
	if r.code != 200 || r.get("question_count") != 6.0 || r.get("version") != 2.0 {
		t.Fatalf("edit %s", r.raw)
	}
	want(t, "submit", x.post("/v1/creator/quizzes/"+quiz+"/submit", tokA, nil), 200, "")
	want(t, "submit twice", x.post("/v1/creator/quizzes/"+quiz+"/submit", tokA, nil), 409, "MATCH_STATE_INVALID")
	want(t, "player queue", x.do(http.MethodGet, "/v1/admin/quizzes", tokA, nil), 403, "FORBIDDEN")
	r = x.post("/v1/creator/quizzes", tokM, quizBody(cat, 5))
	own := str(r.get("id"))
	x.post("/v1/creator/quizzes/"+own+"/submit", tokM, nil)
	want(t, "self review", x.post("/v1/admin/quizzes/"+own+"/review", tokM, map[string]any{"approve": true}), 403, "SELF_REVIEW_FORBIDDEN")
	if r := x.do(http.MethodGet, "/v1/admin/quizzes", tokM, nil); len(r.get("quizzes").([]any)) != 2 {
		t.Fatalf("review queue %s", r.raw)
	}
	want(t, "reject without reason", x.post("/v1/admin/quizzes/"+quiz+"/review", tokM, map[string]any{"approve": false}), 400, "VALIDATION_FAILED")
	r = x.post("/v1/admin/quizzes/"+quiz+"/review", tokM, map[string]any{"approve": true})
	if r.code != 200 || r.get("status") != "published" {
		t.Fatalf("publish %s", r.raw)
	}

	// Discovery and likes.
	r = x.do(http.MethodGet, "/v1/creator/quizzes?locale=en", tokS, nil)
	if list := r.get("quizzes").([]any); len(list) != 1 || list[0].(map[string]any)["id"] != quiz || list[0].(map[string]any)["play_count"] != 1.0 {
		t.Fatalf("discover %s", r.raw)
	}
	if r := x.do(http.MethodGet, "/v1/creator/quizzes/"+quiz, tokS, nil); r.code != 200 || r.get("questions") != nil {
		t.Fatalf("published to others %s", r.raw)
	}
	want(t, "guest like", x.post("/v1/creator/quizzes/"+quiz+"/like", guest, nil), 403, "ACCOUNT_UPGRADE_REQUIRED")
	want(t, "like", x.post("/v1/creator/quizzes/"+quiz+"/like", tokS, nil), 204, "")
	want(t, "like twice", x.post("/v1/creator/quizzes/"+quiz+"/like", tokS, nil), 204, "")
	if r := x.do(http.MethodGet, "/v1/creator/quizzes/"+quiz, tokS, nil); r.get("like_count") != 1.0 || r.get("liked") != true {
		t.Fatalf("liked %s", r.raw)
	}

	// Published: anyone's room, open to everyone; challenges allowed.
	r = x.post("/v1/rooms", tokS, map[string]any{"match_config": cfg})
	if r.code != 201 || r.get("friends_of_host_only") != false || r.get("match_config", "quiz_length") != 6.0 {
		t.Fatalf("published room %s", r.raw)
	}
	want(t, "leave room", x.do(http.MethodPost, "/v1/rooms/"+str(r.get("id"))+"/leave", tokS, nil), 204, "")
	want(t, "challenge", x.post("/v1/challenges", tokA, map[string]any{"target_user_id": f, "match_config": cfg}), 201, "")

	// Awards: players of a published quiz earn, its author does not;
	// an unpublished quiz gives nothing.
	p := finishedDuel(a, s, false)
	p.CategoryMode, p.Origin, p.QuizAuthorID, p.QuizPublished = "creator_quiz", "private_room", &a, true
	x.progress(p)
	if r := x.do(http.MethodGet, "/v1/me/progress", tokA, nil); r.get("xp_total") != 0.0 {
		t.Fatalf("author xp %s", r.raw)
	}
	if r := x.do(http.MethodGet, "/v1/me/progress", tokS, nil); r.get("xp_total") == 0.0 {
		t.Fatalf("player xp %s", r.raw)
	}
	p = finishedDuel(f, s, false)
	p.CategoryMode, p.Origin, p.QuizAuthorID = "creator_quiz", "private_room", &a
	before := x.do(http.MethodGet, "/v1/me/progress", tokS, nil).get("xp_total")
	x.progress(p)
	if r := x.do(http.MethodGet, "/v1/me/progress", tokF, nil); r.get("xp_total") != 0.0 {
		t.Fatalf("unpublished xp %s", r.raw)
	}
	if r := x.do(http.MethodGet, "/v1/me/progress", tokS, nil); r.get("xp_total") != before {
		t.Fatalf("unpublished xp %s", r.raw)
	}

	// Deleting removes the quiz everywhere.
	want(t, "delete by other", x.do(http.MethodDelete, "/v1/creator/quizzes/"+quiz, tokS, nil), 404, "NOT_FOUND")
	want(t, "delete", x.do(http.MethodDelete, "/v1/creator/quizzes/"+quiz, tokA, nil), 204, "")
	want(t, "deleted", x.do(http.MethodGet, "/v1/creator/quizzes/"+quiz, tokA, nil), 404, "NOT_FOUND")
	want(t, "deleted room", x.post("/v1/rooms", tokS, map[string]any{"match_config": cfg}), 404, "NOT_FOUND")
}
