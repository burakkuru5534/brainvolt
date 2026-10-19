package app

import (
	"net/http"
	"testing"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
)

// Reports, the staff queue, sanctions with their effects and appeals
// (schema-moderation, D-030).
func TestModeration(t *testing.T) {
	x := newE2E(t)
	tok1, u1, _ := x.fullUser()
	tok2, _, _ := x.fullUser()
	tok3, _, _ := x.fullUser()
	tokB, b, _ := x.fullUser()
	tokM, m, _ := x.fullUser()
	tokA, adm, _ := x.fullUser()
	x.seedQuestions(u1)
	for _, g := range []struct {
		id   string
		role string
	}{{m.String(), identity.RoleModerator}, {adm.String(), identity.RoleAdmin}} {
		if _, err := x.mods.identity.SetRole(t.Context(), g.id, g.role, true); err != nil {
			t.Fatal(err)
		}
	}
	var question string
	if err := x.db.QueryRow(t.Context(), `SELECT id FROM content.questions WHERE verification_status = 'verified' LIMIT 1`).Scan(&question); err != nil {
		t.Fatal(err)
	}

	// Three wrong_answer reports pull the question back to review.
	report := func(tok string, body map[string]any) response { return x.post("/v1/reports", tok, body) }
	q := map[string]any{"target_kind": "question", "target_id": question, "reason": "wrong_answer"}
	r := report(tok1, q)
	want(t, "report", r, 201, "")
	qcase := str(r.get("case_id"))
	want(t, "again", report(tok1, q), 409, "ALREADY_REPORTED")
	want(t, "bad reason", report(tok1, map[string]any{"target_kind": "question", "target_id": question, "reason": "cheating"}), 400, "VALIDATION_FAILED")
	want(t, "self", report(tok1, map[string]any{"target_kind": "user", "target_id": u1, "reason": "spam"}), 400, "VALIDATION_FAILED")
	want(t, "second", report(tok2, q), 201, "")
	want(t, "third", report(tok3, q), 201, "")
	var status string
	if err := x.db.QueryRow(t.Context(), `SELECT verification_status FROM content.questions WHERE id = $1`, question).Scan(&status); err != nil || status != "in_review" {
		t.Fatalf("question after 3 reports %q %v", status, err)
	}

	// The staff queue.
	want(t, "player queue", x.do(http.MethodGet, "/v1/admin/cases", tok1, nil), 403, "FORBIDDEN")
	r = x.do(http.MethodGet, "/v1/admin/cases", tokM, nil)
	if cases, _ := r.get("cases").([]any); r.code != 200 || len(cases) != 1 || cases[0].(map[string]any)["report_count"] != 3.0 {
		t.Fatalf("queue %s", r.raw)
	}
	r = x.do(http.MethodGet, "/v1/admin/cases/"+qcase, tokM, nil)
	if reps, _ := r.get("reports").([]any); len(reps) != 3 || len(r.get("history").([]any)) != 1 {
		t.Fatalf("case %s", r.raw)
	}
	want(t, "retire by mod", x.post("/v1/admin/cases/"+qcase+"/actions", tokM, map[string]any{"action": "question_retire"}), 403, "FORBIDDEN")
	r = x.post("/v1/admin/cases/"+qcase+"/actions", tokM, map[string]any{"action": "case_dismiss", "note": "answer is right"})
	if r.code != 200 || r.get("case", "status") != "resolved" || r.get("case", "resolution") != "dismissed" {
		t.Fatalf("dismiss %s", r.raw)
	}

	// A matchmaking ban keeps B out of the queues.
	r = report(tok1, map[string]any{"target_kind": "user", "target_id": b, "reason": "cheating"})
	ucase := str(r.get("case_id"))
	want(t, "permanent by mod", x.post("/v1/admin/cases/"+ucase+"/actions", tokM, map[string]any{
		"action": "user_sanction", "sanction": map[string]any{"kind": "matchmaking_ban", "reason_code": "cheating_confirmed"}}), 403, "FORBIDDEN")
	want(t, "ban", x.post("/v1/admin/cases/"+ucase+"/actions", tokM, map[string]any{
		"action": "user_sanction", "sanction": map[string]any{"kind": "matchmaking_ban", "hours": 24, "reason_code": "cheating_confirmed"}}), 200, "")
	r = x.do(http.MethodGet, "/v1/me/sanctions", tokB, nil)
	sanctions, _ := r.get("sanctions").([]any)
	if len(sanctions) != 1 || sanctions[0].(map[string]any)["active"] != true || sanctions[0].(map[string]any)["kind"] != "matchmaking_ban" {
		t.Fatalf("sanctions %s", r.raw)
	}
	sanction := str(sanctions[0].(map[string]any)["id"])
	r = x.post("/v1/matchmaking/tickets", tokB, map[string]any{"match_type": "quick_duel", "format": "1v1"})
	if r.code != 403 || r.errCode() != "SANCTIONED" || r.get("error", "details", "kind") != "matchmaking_ban" {
		t.Fatalf("queue while banned %s", r.raw)
	}

	// Appeals: one per sanction, not decided by the sanction's creator.
	want(t, "short appeal", x.post("/v1/me/sanctions/"+sanction+"/appeal", tokB, map[string]any{"text": "no"}), 400, "VALIDATION_FAILED")
	want(t, "appeal", x.post("/v1/me/sanctions/"+sanction+"/appeal", tokB, map[string]any{"text": "I did not cheat, please check."}), 204, "")
	want(t, "appeal twice", x.post("/v1/me/sanctions/"+sanction+"/appeal", tokB, map[string]any{"text": "I did not cheat, please check."}), 409, "APPEAL_EXISTS")
	r = x.do(http.MethodGet, "/v1/admin/appeals", tokM, nil)
	appeals, _ := r.get("appeals").([]any)
	if len(appeals) != 1 {
		t.Fatalf("appeals %s", r.raw)
	}
	appeal := str(appeals[0].(map[string]any)["id"])
	want(t, "creator decides", x.post("/v1/admin/appeals/"+appeal+"/decide", tokM, map[string]any{"accept": true}), 403, "SELF_REVIEW_FORBIDDEN")
	want(t, "admin decides", x.post("/v1/admin/appeals/"+appeal+"/decide", tokA, map[string]any{"accept": true}), 204, "")
	if r := x.do(http.MethodGet, "/v1/me/sanctions", tokB, nil); r.get("sanctions").([]any)[0].(map[string]any)["active"] != false {
		t.Fatalf("after appeal %s", r.raw)
	}
	r = x.post("/v1/matchmaking/tickets", tokB, map[string]any{"match_type": "quick_duel", "format": "1v1", "category_mode": "blind_challenge"})
	want(t, "queue after appeal", r, 201, "")
	want(t, "leave queue", x.do(http.MethodDelete, "/v1/matchmaking/tickets/current", tokB, nil), 204, "")

	// A suspension locks the account until lifted.
	r = report(tok2, map[string]any{"target_kind": "user", "target_id": b, "reason": "harassment"})
	scase := str(r.get("case_id"))
	want(t, "suspend", x.post("/v1/admin/cases/"+scase+"/actions", tokA, map[string]any{
		"action": "user_sanction", "sanction": map[string]any{"kind": "suspension", "hours": 1}}), 200, "")
	want(t, "suspended", x.do(http.MethodGet, "/v1/me/progress", tokB, nil), 403, "ACCOUNT_SUSPENDED")
	r = x.do(http.MethodGet, "/v1/admin/cases/"+scase, tokA, nil)
	var susp string
	for _, h := range r.get("history").([]any) {
		if p, _ := h.(map[string]any)["payload"].(map[string]any); p["sanction_id"] != nil {
			susp = str(p["sanction_id"])
		}
	}
	want(t, "lift", x.post("/v1/admin/cases/"+scase+"/actions", tokA, map[string]any{"action": "sanction_lift", "sanction_id": susp}), 200, "")
	want(t, "restored", x.do(http.MethodGet, "/v1/me/progress", tokB, nil), 200, "")

	// Nobody acts on a case about themselves.
	r = report(tok3, map[string]any{"target_kind": "user", "target_id": m, "reason": "spam"})
	want(t, "self case", x.post("/v1/admin/cases/"+str(r.get("case_id"))+"/actions", tokM, map[string]any{"action": "case_dismiss"}), 403, "SELF_REVIEW_FORBIDDEN")
}
