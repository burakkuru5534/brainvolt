package app

import (
	"net/http"
	"testing"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
)

// showNow moves a run's open question so it is shown now.
func (x *e2e) showNow(run string, pos float64) {
	x.t.Helper()
	if _, err := x.db.Exec(x.t.Context(), `UPDATE game.daily_run_answers
		SET show_at = now() - interval '1 second', deadline_at = now() + interval '14 seconds' WHERE run_id = $1 AND position = $2`, run, int(pos)); err != nil {
		x.t.Fatal(err)
	}
}

// correctOption is the right answer of a run's question.
func (x *e2e) correctOption(run string, pos float64) string {
	x.t.Helper()
	var id string
	if err := x.db.QueryRow(x.t.Context(), `SELECT o.id FROM game.daily_run_answers a JOIN content.options o ON o.question_id = a.question_id AND o.is_correct
		WHERE a.run_id = $1 AND a.position = $2`, run, int(pos)).Scan(&id); err != nil {
		x.t.Fatal(err)
	}
	return id
}

// Daily Challenge (category-modes §5): one official run a day, a solo
// run over the WebSocket, the board and the XP from the completed event.
func TestDailyChallenge(t *testing.T) {
	x := newE2E(t)
	tokA, a, _ := x.fullUser()
	x.seedQuestions(a)

	r := x.do(http.MethodGet, "/v1/daily/today", tokA, nil)
	if r.code != 200 || r.get("available") != true || r.get("official") != nil {
		t.Fatalf("today %s", r.raw)
	}
	day := str(r.get("day"))
	want(t, "practice first", x.post("/v1/daily/runs", tokA, map[string]any{"kind": "practice"}), 409, "DAILY_ATTEMPT_IN_PROGRESS")
	want(t, "bad kind", x.post("/v1/daily/runs", tokA, map[string]any{"kind": "ranked"}), 400, "VALIDATION_FAILED")
	r = x.post("/v1/daily/runs", tokA, map[string]any{"kind": "official"})
	run := str(r.get("id"))
	if r.code != 201 || r.get("status") != "in_progress" {
		t.Fatalf("start %s", r.raw)
	}
	want(t, "second official", x.post("/v1/daily/runs", tokA, map[string]any{"kind": "official"}), 409, "DAILY_ATTEMPT_IN_PROGRESS")

	w, _ := x.auth(tokA, "inst-a", nil)
	w.send("daily.join", map[string]any{"run_id": run})
	q := data(w.next("daily.question"))
	if q["position"] != 1.0 || q["total"] != 10.0 || len(q["options"].([]any)) < 2 || q["prompt"] == "" {
		t.Fatalf("question %v", q)
	}
	// Answering before the question is shown is refused.
	w.send("daily.answer", map[string]any{"run_id": run, "position": 1, "option_id": x.correctOption(run, 1)})
	if e := data(w.next("sys.error")); e["code"] != "MATCH_STATE_INVALID" {
		t.Fatalf("blind answer %v", e)
	}
	for pos := 1.0; pos <= 10; pos++ {
		x.showNow(run, pos)
		option := x.correctOption(run, pos)
		if pos == 10 {
			// A wrong last answer: any other option.
			for _, o := range q["options"].([]any) {
				if id := str(o.(map[string]any)["id"]); id != option {
					option = id
					break
				}
			}
		}
		w.send("daily.answer", map[string]any{"run_id": run, "position": pos, "option_id": option})
		rv := data(w.next("daily.revealed"))
		if rv["position"] != pos || rv["correct"] != (pos < 10) {
			t.Fatalf("revealed %v", rv)
		}
		// D-100: never the correct option during the day.
		if _, leaked := rv["correct_option_id"]; leaked {
			t.Fatalf("daily reveal leaks the answer %v", rv)
		}
		if pos < 10 {
			q = data(w.next("daily.question"))
			if q["position"] != pos+1 {
				t.Fatalf("next question %v", q)
			}
		}
	}
	done := data(w.next("daily.completed"))
	if done["correct_count"] != 9.0 || done["kind"] != "official" || done["total_time_ms"].(float64) > 20_000 {
		t.Fatalf("completed %v", done)
	}
	r = x.do(http.MethodGet, "/v1/daily/today", tokA, nil)
	if r.get("official", "status") != "completed" || r.get("practice_allowed") != true || len(r.get("official", "answers").([]any)) != 10 {
		t.Fatalf("today after %s", r.raw)
	}
	want(t, "official again", x.post("/v1/daily/runs", tokA, map[string]any{"kind": "official"}), 409, "DAILY_ATTEMPT_USED")
	want(t, "practice", x.post("/v1/daily/runs", tokA, map[string]any{"kind": "practice"}), 201, "")

	// The completed event feeds the board and progression.
	var payload []byte
	if err := x.db.QueryRow(t.Context(), `SELECT args->'event'->'payload' FROM river_job WHERE args->'event'->>'name' = $1
		AND args->>'subscriber' = 'ranking.daily_board' AND args->'event'->'payload'->>'run_id' = $2`, game.EventDailyRunCompleted, run).Scan(&payload); err != nil {
		t.Fatal(err)
	}
	env := events.Envelope{Name: game.EventDailyRunCompleted, Payload: payload}
	for range 2 {
		if err := x.mods.ranking.OnDailyRunCompleted(t.Context(), env); err != nil {
			t.Fatal(err)
		}
		if err := x.mods.progression.OnDailyRunCompleted(t.Context(), env); err != nil {
			t.Fatal(err)
		}
	}
	r = x.do(http.MethodGet, "/v1/leaderboards/daily-challenge/"+day, tokA, nil)
	if r.code != 200 || r.get("me", "rank") != 1.0 || r.get("me", "correct_count") != 9.0 {
		t.Fatalf("daily board %s", r.raw)
	}
	if r := x.do(http.MethodGet, "/v1/me/progress", tokA, nil); r.get("xp_total") != 75.0 || r.get("streak", "current") != 1.0 {
		t.Fatalf("daily XP %s", r.raw)
	}
	want(t, "bad scope", x.do(http.MethodGet, "/v1/leaderboards/daily-challenge/"+day+"?scope=team", tokA, nil), 400, "VALIDATION_FAILED")

	// D-100: today's answers stay closed; once the day is over they open
	// (the run and the set are moved to yesterday to simulate it).
	want(t, "answers today", x.do(http.MethodGet, "/v1/daily/answers?day="+day, tokA, nil), 409, "DAILY_ANSWERS_NOT_OPEN")
	want(t, "answers bad day", x.do(http.MethodGet, "/v1/daily/answers?day=yesterday", tokA, nil), 400, "VALIDATION_FAILED")
	for _, stmt := range []string{
		`UPDATE content.daily_sets SET day = day - 1 WHERE day = $1::date`,
		`UPDATE game.daily_runs SET day = day - 1 WHERE day = $1::date`,
	} {
		if _, err := x.db.Exec(t.Context(), stmt, day); err != nil {
			t.Fatal(err)
		}
	}
	r = x.do(http.MethodGet, "/v1/daily/answers", tokA, nil)
	qs, _ := r.get("questions").([]any)
	if r.code != 200 || len(qs) != 10 {
		t.Fatalf("yesterday's answers %s", r.raw)
	}
	right := 0
	for _, v := range qs {
		q := v.(map[string]any)
		if q["correct_option_id"] == nil || q["prompt"] == "" || len(q["options"].([]any)) < 2 {
			t.Fatalf("answer review %v", q)
		}
		if q["your_correct"] == true && q["your_option_id"] == q["correct_option_id"] {
			right++
		}
	}
	if right != 9 {
		t.Fatalf("own answers in the review: %d right, want 9 (%s)", right, r.raw)
	}
	// Move them back: the rest of the test plays today.
	for _, stmt := range []string{
		`UPDATE content.daily_sets SET day = day + 1 WHERE day = $1::date - 1`,
		`UPDATE game.daily_runs SET day = day + 1 WHERE day = $1::date - 1`,
	} {
		if _, err := x.db.Exec(t.Context(), stmt, day); err != nil {
			t.Fatal(err)
		}
	}

	// The clock keeps running: a missed deadline is a timeout on resume.
	tokB, _, _ := x.fullUser()
	r = x.post("/v1/daily/runs", tokB, map[string]any{"kind": "official"})
	runB := str(r.get("id"))
	wb, _ := x.auth(tokB, "inst-b", nil)
	wb.send("daily.join", map[string]any{"run_id": runB})
	wb.next("daily.question")
	if _, err := x.db.Exec(t.Context(), `UPDATE game.daily_run_answers SET show_at = now() - interval '20 seconds',
		deadline_at = now() - interval '5 seconds' WHERE run_id = $1`, runB); err != nil {
		t.Fatal(err)
	}
	wb.send("daily.join", map[string]any{"run_id": runB})
	if rv := data(wb.next("daily.revealed")); rv["position"] != 1.0 || rv["correct"] != false || rv["option_id"] != nil {
		t.Fatalf("timeout %v", rv)
	}
	if q := data(wb.next("daily.question")); q["position"] != 2.0 {
		t.Fatalf("after timeout %v", q)
	}

	// A run never started is voided and frees the official attempt.
	tokC, _, _ := x.fullUser()
	r = x.post("/v1/daily/runs", tokC, map[string]any{"kind": "official"})
	if _, err := x.db.Exec(t.Context(), `UPDATE game.daily_runs SET started_at = now() - interval '11 minutes' WHERE id = $1`, str(r.get("id"))); err != nil {
		t.Fatal(err)
	}
	if r := x.do(http.MethodGet, "/v1/daily/today", tokC, nil); r.get("official", "status") != "void_not_started" {
		t.Fatalf("void %s", r.raw)
	}
	want(t, "official after void", x.post("/v1/daily/runs", tokC, map[string]any{"kind": "official"}), 201, "")
}
