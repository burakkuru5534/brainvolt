package app

import (
	"context"
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/contentfiles"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/content"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
)

// seedQuestions imports the repository's questions, verified (dev flow).
func (x *e2e) seedQuestions(approver uuid.UUID) {
	x.t.Helper()
	bundle, err := contentfiles.Load(filepath.Join("..", "..", "..", "content"))
	if err != nil {
		x.t.Fatal(err)
	}
	if err := importBundle(x.t.Context(), content.New(x.db, clock.Real{}, events.NewBus()), bundle, approver, io.Discard, discard); err != nil {
		x.t.Fatal(err)
	}
}

func str(v any) string { s, _ := v.(string); return s }

// A live quick duel over WebSockets: creation, join, ready, countdown,
// every round with texts and reveal, completion, and the result in
// Postgres; a reconnect in the middle replays the missed events.
func TestLiveQuickDuel(t *testing.T) {
	x := newE2E(t)
	tokA, a := x.signedIn()
	tokB, b := x.signedIn()
	x.seedQuestions(a)
	wa, welA := x.auth(tokA, "inst-a", nil)
	wb, welB := x.auth(tokB, "inst-b", nil)
	if welA["user_id"] != a.String() || welB["user_id"] != b.String() {
		t.Fatalf("welcome users %v %v", welA["user_id"], welB["user_id"])
	}

	c, err := x.mods.game.Start(t.Context(), game.CreateRequest{
		Config: game.Config{MatchType: "quick_duel", Format: "1v1", Origin: "challenge"},
		Sides:  [2][]game.Member{{{UserID: a, Locale: "tr"}}, {{UserID: b, Locale: "en"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	id := c.Match.ID.String()
	created := data(wa.next("match.created"))
	if created["match_id"] != id || created["question_count"] != float64(7) || len(created["participants"].([]any)) != 2 {
		t.Fatalf("match.created %v", created)
	}
	wb.next("match.created")

	for _, w := range []*wsClient{wa, wb} {
		re := w.send("match.join", map[string]any{"match_id": id})
		if f := w.answer(re); f["t"] != "sys.ack" {
			t.Fatalf("join ack %v", f)
		}
		if s := data(w.next("match.snapshot")); s["status"] != "waiting" {
			t.Fatalf("snapshot %v", s)
		}
	}
	// Not a participant → error.
	tokC, _ := x.signedIn()
	wc, _ := x.auth(tokC, "inst-c", nil)
	re := wc.send("match.ready", map[string]any{"match_id": id})
	if f := wc.answer(re); f["t"] != "sys.error" || data(f)["code"] != "NOT_A_PARTICIPANT" {
		t.Fatalf("outsider ready %v", f)
	}

	wa.send("match.ready", map[string]any{"match_id": id})
	wb.send("match.ready", map[string]any{"match_id": id})
	starting := data(wa.next("match.starting"))
	if starting["starts_at"] == nil {
		t.Fatal("no starts_at")
	}
	wa.next("match.started")

	for round := 1; round <= 7; round++ {
		sa := data(wa.next("round.started"))
		sb := data(wb.next("round.started"))
		qa, qb := sa["question"].(map[string]any), sb["question"].(map[string]any)
		if qa["id"] != qb["id"] || str(qa["prompt"]) == "" || str(qb["prompt"]) == "" || sa["round_no"] != float64(round) {
			t.Fatalf("round %d started: %v / %v", round, sa, sb)
		}
		opts := sa["options"].([]any)
		if len(opts) < 2 || str(opts[0].(map[string]any)["text"]) == "" || sa["correct_option_id"] != nil {
			t.Fatalf("round %d options %v", round, opts)
		}
		// Wait until show_at so the answers are on time.
		if wait := time.Until(time.UnixMilli(int64(sa["show_at"].(float64)))); wait > 0 {
			time.Sleep(wait)
		}
		first := opts[0].(map[string]any)["id"]
		re := wa.send("match.answer", map[string]any{"match_id": id, "round_no": round, "option_id": first})
		if f := wa.answer(re); f["t"] != "sys.ack" {
			t.Fatalf("answer ack %v", f)
		}
		answerOf := func(w *wsClient, user uuid.UUID) map[string]any {
			return data(w.find("answer_received", func(f map[string]any) bool {
				d := data(f)
				return f["t"] == "round.answer_received" && d["user_id"] == user.String() && d["round_no"] == float64(round)
			}))
		}
		own, other := answerOf(wa, a), answerOf(wb, a)
		if own["option_id"] != first || other["option_id"] != nil {
			t.Fatalf("answer_received own %v other %v (a=%s b=%s)", own, other, a, b)
		}
		if round == 4 {
			// B drops before answering; A sees the disconnect. B resumes its
			// session with the last sequence number it saw and gets the missed
			// events replayed, then takes its seat again.
			last := wb.lastSeq.Load()
			_ = wb.c.CloseNow()
			wb.waitClose()
			if st := data(wa.next("match.participant_status")); st["status"] != "disconnected" || st["reconnect_deadline_at"] == nil {
				t.Fatalf("disconnect status %v", st)
			}
			wb, welB = x.auth(tokB, "inst-b", map[string]any{
				"session_id": welB["session_id"], "resume_token": welB["resume_token"], "streams": map[string]any{"match:" + id: last},
			})
			replayed := wb.next("match.participant_status")
			if int64(replayed["seq"].(float64)) != last+1 || data(replayed)["status"] != "disconnected" {
				t.Fatalf("replay %v after seq %d", replayed, last)
			}
			re := wb.send("match.join", map[string]any{"match_id": id})
			wb.answer(re)
			snap := data(wb.next("match.snapshot"))
			rd, _ := snap["round"].(map[string]any)
			if snap["phase"] != "question" || rd == nil || rd["round_no"] != float64(4) || rd["own_option_id"] != nil {
				t.Fatalf("snapshot after rejoin %v (re %s)", snap, re)
			}
			if st := data(wa.next("match.participant_status")); st["status"] != "playing" {
				t.Fatalf("back status %v", st)
			}
		}
		wb.send("match.answer", map[string]any{"match_id": id, "round_no": round, "option_id": opts[1].(map[string]any)["id"]})
		rev := wa.next("round.revealed")
		wb.next("round.revealed")
		if d := data(rev); d["correct_option_id"] == nil || d["category"] == nil || len(d["sides"].([]any)) != 2 {
			t.Fatalf("revealed %v", d)
		}
	}
	done := data(wa.next("match.completed"))
	wb.next("match.completed")
	if done["end_reason"] != "all_rounds_played" {
		t.Fatalf("completed %v", done)
	}
	var status string
	if err := x.db.QueryRow(t.Context(), `SELECT status FROM game.matches WHERE id = $1`, c.Match.ID).Scan(&status); err != nil || status != "completed" {
		t.Fatalf("db status %q %v", status, err)
	}
	var rounds int
	_ = x.db.QueryRow(t.Context(), `SELECT count(*) FROM game.match_rounds WHERE match_id = $1`, c.Match.ID).Scan(&rounds)
	if rounds != 7 {
		t.Fatalf("%d rounds stored", rounds)
	}
	if v, _ := x.rdb.Exists(t.Context(), "game:user:"+a.String()+":live").Result(); v != 0 {
		t.Fatal("live lock not released")
	}
}

// findWithin is find with a longer wait (takeover needs the lease to expire).
func (w *wsClient) findWithin(d time.Duration, typ string) map[string]any {
	w.t.Helper()
	deadline := time.After(d)
	for {
		for i, f := range w.held {
			if f["t"] == typ {
				w.held = append(w.held[:i], w.held[i+1:]...)
				return f
			}
		}
		select {
		case f, ok := <-w.frames:
			if !ok {
				w.t.Fatalf("connection closed while waiting for %s", typ)
			}
			w.held = append(w.held, f)
		case <-deadline:
			w.t.Fatalf("no %s within %s", typ, d)
		}
	}
}

// Edge-cases §4: the owning instance dies during an open round; another
// instance takes the match over after the lease expires, voids the round,
// replays it with a new question and the match completes.
func TestLiveTakeover(t *testing.T) {
	x := newE2E(t)
	tokA, a := x.signedIn()
	tokB, b := x.signedIn()
	x.seedQuestions(a)
	wa, _ := x.auth(tokA, "inst-a", nil)
	wb, _ := x.auth(tokB, "inst-b", nil)
	c, err := x.mods.game.Start(t.Context(), game.CreateRequest{
		Config: game.Config{MatchType: "quick_duel", Format: "1v1", Origin: "challenge"},
		Sides:  [2][]game.Member{{{UserID: a, Locale: "tr"}}, {{UserID: b, Locale: "tr"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	id := c.Match.ID.String()
	for _, w := range []*wsClient{wa, wb} {
		w.answer(w.send("match.join", map[string]any{"match_id": id}))
		w.answer(w.send("match.ready", map[string]any{"match_id": id}))
	}
	first := data(wa.findWithin(10*time.Second, "round.started"))
	q1 := first["question"].(map[string]any)["id"]

	// The owner instance dies (lease kept, as after a crash).
	x.mods.game.Close()
	second := game.New(game.Deps{
		DB: x.db, Redis: x.rdb, Content: x.mods.content, Identity: x.mods.identity, Profiles: x.mods.profile,
		Realtime: x.mods.realtime, Events: newBus(x.db), Clock: clock.Real{}, Logger: discard, Instance: "test-second-" + id,
	})
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() { _ = second.Run(ctx) }()

	resumed := data(wa.findWithin(20*time.Second, "match.resumed"))
	if resumed["reason"] != "takeover" || resumed["voided_round_no"] != float64(1) {
		t.Fatalf("resumed %v", resumed)
	}
	again := data(wa.findWithin(5*time.Second, "round.started"))
	if again["round_no"] != float64(1) || again["question"].(map[string]any)["id"] == q1 {
		t.Fatalf("replacement round %v", again)
	}
	for round := 1; ; round++ {
		var started map[string]any
		if round == 1 {
			started = again
		} else {
			f := wa.findWithin(10*time.Second, "round.started")
			started = data(f)
		}
		if wait := time.Until(time.UnixMilli(int64(started["show_at"].(float64)))); wait > 0 {
			time.Sleep(wait)
		}
		opt := started["options"].([]any)[0].(map[string]any)["id"]
		wa.send("match.answer", map[string]any{"match_id": id, "round_no": round, "option_id": opt})
		wb.send("match.answer", map[string]any{"match_id": id, "round_no": round, "option_id": opt})
		if round == 7 {
			break
		}
	}
	done := data(wa.findWithin(10*time.Second, "match.completed"))
	if done["end_reason"] != "all_rounds_played" {
		t.Fatalf("completed %v", done)
	}
	var rounds, voided int
	_ = x.db.QueryRow(t.Context(), `SELECT count(*) FROM game.match_rounds WHERE match_id = $1`, c.Match.ID).Scan(&rounds)
	_ = x.db.QueryRow(t.Context(), `SELECT count(*) FROM game.match_transitions WHERE match_id = $1 AND reason = 'round_voided_takeover'`, c.Match.ID).Scan(&voided)
	if rounds != 7 || voided != 1 {
		t.Fatalf("rounds %d, voided transitions %d", rounds, voided)
	}
}

func (x *e2e) post(path, token string, body any) response {
	x.t.Helper()
	return x.do("POST", path, token, body, "Idempotency-Key", ids.NewString())
}

// Phase 4 entry: the Quick Duel queue pairs two players, both get
// mm.match_found; a player leaving before the start cancels the match;
// the result endpoint and /v1/matches/current follow.
func TestQueuePairsPlayers(t *testing.T) {
	x := newE2E(t)
	tokA, a := x.signedIn()
	tokB, _ := x.signedIn()
	x.seedQuestions(a)
	wa, _ := x.auth(tokA, "inst-a", nil)
	wb, _ := x.auth(tokB, "inst-b", nil)

	want(t, "unsupported mode", x.post("/v1/matchmaking/tickets", tokA, map[string]any{"match_type": "ranked", "format": "1v1"}), 422, "MODE_NOT_ALLOWED")
	r := x.post("/v1/matchmaking/tickets", tokA, map[string]any{"match_type": "quick_duel", "format": "1v1"})
	want(t, "ticket", r, 201, "")
	if r.get("state") != "searching" {
		t.Fatalf("ticket %s", r.raw)
	}
	want(t, "second ticket", x.post("/v1/matchmaking/tickets", tokA, map[string]any{"match_type": "quick_duel", "format": "1v1"}), 409, "ALREADY_QUEUED")
	want(t, "current ticket", x.do("GET", "/v1/matchmaking/tickets/current", tokA, nil), 200, "")
	want(t, "leave", x.do("DELETE", "/v1/matchmaking/tickets/current", tokA, nil), 204, "")
	want(t, "no ticket", x.do("GET", "/v1/matchmaking/tickets/current", tokA, nil), 404, "NOT_FOUND")

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() { _ = x.mods.matchmaking.Run(ctx) }()
	want(t, "ticket A", x.post("/v1/matchmaking/tickets", tokA, map[string]any{"match_type": "quick_duel", "format": "1v1"}), 201, "")
	want(t, "ticket B", x.post("/v1/matchmaking/tickets", tokB, map[string]any{"match_type": "quick_duel", "format": "1v1"}), 201, "")
	found := data(wa.findWithin(5*time.Second, "mm.match_found"))
	if data(wb.findWithin(5*time.Second, "mm.match_found"))["match_id"] != found["match_id"] {
		t.Fatal("players matched into different matches")
	}
	id := str(found["match_id"])
	cur := x.do("GET", "/v1/matches/current", tokA, nil)
	if cur.get("match", "id") != id || cur.get("match", "stream") != "match:"+id {
		t.Fatalf("current %s", cur.raw)
	}
	want(t, "queue busy while in a match", x.post("/v1/matchmaking/tickets", tokA, map[string]any{"match_type": "quick_duel", "format": "1v1"}), 409, "ALREADY_IN_MATCH")

	wb.answer(wb.send("match.join", map[string]any{"match_id": id}))
	wb.answer(wb.send("match.leave", map[string]any{"match_id": id}))
	if c := data(wa.findWithin(5*time.Second, "match.cancelled")); c["reason"] != "participant_left_before_start" {
		t.Fatalf("cancelled %v", c)
	}
	v := x.do("GET", "/v1/matches/"+id, tokA, nil)
	want(t, "result", v, 200, "")
	if v.get("status") != "cancelled" || len(v.get("participants").([]any)) != 2 {
		t.Fatalf("match view %s", v.raw)
	}
	tokC, _ := x.signedIn()
	want(t, "outsider", x.do("GET", "/v1/matches/"+id, tokC, nil), 403, "NOT_A_PARTICIPANT")
	if x.do("GET", "/v1/matches/current", tokA, nil).get("match") != nil {
		t.Fatal("cancelled match still current")
	}
}

// The development practice bot readies and answers on its own.
func TestPracticeAgainstBot(t *testing.T) {
	x := newE2E(t)
	tok, user := x.signedIn()
	x.seedQuestions(user)
	w, _ := x.auth(tok, "inst-p", nil)
	r := x.post("/v1/matchmaking/tickets", tok, map[string]any{"match_type": "quick_duel", "format": "1v1", "practice": true})
	want(t, "practice", r, 201, "")
	id := str(r.get("match_id"))
	if r.get("state") != "found" || id == "" {
		t.Fatalf("practice ticket %s", r.raw)
	}
	w.answer(w.send("match.join", map[string]any{"match_id": id}))
	w.answer(w.send("match.ready", map[string]any{"match_id": id}))
	started := data(w.findWithin(10*time.Second, "round.started"))
	bot := ""
	for _, p := range data(w.next("match.created"))["participants"].([]any) {
		if pm := p.(map[string]any); pm["user_id"] != user.String() {
			bot = str(pm["user_id"])
			if pm["display_name"] != "Practice Bot" {
				t.Fatalf("bot card %v", pm)
			}
		}
	}
	// The bot answers round 1 by itself, 1.5–6 s after show_at.
	w.findIn(10*time.Second, "bot answer", func(f map[string]any) bool {
		return f["t"] == "round.answer_received" && data(f)["user_id"] == bot
	})
	if wait := time.Until(time.UnixMilli(int64(started["show_at"].(float64)))); wait > 0 {
		time.Sleep(wait)
	}
	w.send("match.answer", map[string]any{"match_id": id, "round_no": 1, "option_id": started["options"].([]any)[0].(map[string]any)["id"]})
	w.findWithin(5*time.Second, "round.revealed")
	w.answer(w.send("match.leave", map[string]any{"match_id": id}))
	done := data(w.findWithin(5*time.Second, "match.completed"))
	if done["end_reason"] != "all_opponents_left" {
		t.Fatalf("completed %v", done)
	}
	v := x.do("GET", "/v1/matches/"+id, tok, nil)
	rounds, _ := v.get("rounds").([]any)
	if v.get("status") != "completed" || len(rounds) != 1 || str(rounds[0].(map[string]any)["prompt"]) == "" ||
		len(rounds[0].(map[string]any)["options"].([]any)) < 2 {
		t.Fatalf("result %s", v.raw)
	}
}

// Mixed Battle (D-043): before each round the picker chooses one of the
// three categories; the others may not, and the round's question comes
// from the picked category.
func TestLiveMixedBattlePick(t *testing.T) {
	x := newE2E(t)
	tokA, a := x.signedIn()
	tokB, b := x.signedIn()
	x.seedQuestions(a)
	rows, err := x.db.Query(t.Context(), `SELECT id FROM content.categories ORDER BY sort_order LIMIT 3`)
	if err != nil {
		t.Fatal(err)
	}
	var cats []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		cats = append(cats, id)
	}
	rows.Close()
	wa, _ := x.auth(tokA, "inst-a", nil)
	wb, _ := x.auth(tokB, "inst-b", nil)
	c, err := x.mods.game.Start(t.Context(), game.CreateRequest{
		Config: game.Config{MatchType: "quick_duel", Format: "1v1", Origin: "challenge", CategoryMode: "mixed_battle", CategoryIDs: cats},
		Sides:  [2][]game.Member{{{UserID: a, Locale: "tr"}}, {{UserID: b, Locale: "en"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	id := c.Match.ID.String()
	for _, w := range []*wsClient{wa, wb} {
		w.answer(w.send("match.join", map[string]any{"match_id": id}))
		w.send("match.ready", map[string]any{"match_id": id})
	}
	pre := data(wa.findIn(10*time.Second, "round.pre_round", func(f map[string]any) bool { return f["t"] == "round.pre_round" }))
	wb.next("round.pre_round")
	named := pre["categories"].([]any)
	if len(named) != 3 || str(named[0].(map[string]any)["name"]) == "" || len(pre["allowed_category_ids"].([]any)) != 3 {
		t.Fatalf("pre_round %v", pre)
	}
	picker, other := wa, wb
	if pre["picker_user_id"] != a.String() {
		picker, other = wb, wa
	}
	pick := map[string]any{"match_id": id, "round_no": 1, "category_id": cats[1]}
	if f := other.answer(other.send("match.pick_category", pick)); data(f)["code"] != "PICK_NOT_ALLOWED" {
		t.Fatalf("pick by the other player %v", f)
	}
	if f := picker.answer(picker.send("match.pick_category", pick)); f["t"] != "sys.ack" {
		t.Fatalf("pick %v", f)
	}
	if d := data(other.next("round.category_picked")); d["category_id"] != cats[1].String() {
		t.Fatalf("category_picked %v", d)
	}
	started := data(other.next("round.started"))
	if cat, _ := started["question"].(map[string]any)["category"].(map[string]any); cat == nil || cat["id"] != cats[1].String() {
		t.Fatalf("round.started %v", started)
	}
	wa.answer(wa.send("match.leave", map[string]any{"match_id": id}))
}
