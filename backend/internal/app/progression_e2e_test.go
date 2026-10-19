package app

import (
	"encoding/json"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/progression"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/ranking"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
)

// finishedDuel is a finished Quick Duel: a answers all 7 rounds right and
// wins, b answers the first one and left when bLeft.
func finishedDuel(a, b uuid.UUID, bLeft bool) game.MatchCompleted {
	cat := uuid.New()
	p := game.MatchCompleted{
		MatchID: uuid.New(), MatchType: "quick_duel", Format: "1v1", Origin: game.OriginMatchmaking, EndReason: "completed",
		WinnerSideNo: 1, RoundsPlayed: 7, EndedAt: time.Now().UTC(),
		Sides: []game.SidePayload{{SideNo: 1, Result: "win", Correct: 7}, {SideNo: 2, Result: "loss", Correct: 1}},
		Participants: []game.ParticipantPayload{
			{UserID: a, SideNo: 1, TierAtStart: "full", Status: "finished", Correct: 7},
			{UserID: b, SideNo: 2, TierAtStart: "full", Status: "finished", Correct: 1},
		},
	}
	if bLeft {
		p.Participants[1].Status = "left"
	}
	for i := 1; i <= 7; i++ {
		r := game.RoundPayload{RoundNo: i, CategoryID: cat, SideCorrect: [2]bool{true, i == 1}, CorrectUserIDs: []uuid.UUID{a}}
		if i == 1 {
			r.CorrectUserIDs = append(r.CorrectUserIDs, b)
		}
		p.Rounds = append(p.Rounds, r)
	}
	return p
}

func (x *e2e) progress(p game.MatchCompleted) {
	x.t.Helper()
	payload, err := json.Marshal(p)
	if err != nil {
		x.t.Fatal(err)
	}
	for range 2 { // idempotent
		if err := x.mods.progression.OnMatchCompleted(x.t.Context(), events.Envelope{Name: game.EventMatchCompleted, Payload: payload}); err != nil {
			x.t.Fatal(err)
		}
	}
}

func TestProgressionFromMatches(t *testing.T) {
	x := newE2E(t)
	tokA, a, _ := x.fullUser()
	tokB, b, _ := x.fullUser()
	m := finishedDuel(a, b, false)
	x.progress(m)

	// 20 + 7 × 5 + 15 = 70 match XP, + 25 first_win + 25 perfect_quick_duel.
	r := x.do(http.MethodGet, "/v1/me/progress", tokA, nil)
	if r.code != 200 || r.get("xp_total") != 120.0 || r.get("level") != 2.0 || r.get("level_xp") != 20.0 || r.get("level_size") != 125.0 ||
		r.get("streak", "current") != 1.0 || r.get("streak", "played_today") != true || r.get("badges") != 2.0 {
		t.Fatalf("progress %s", r.raw)
	}
	if r := x.do(http.MethodGet, "/v1/me/progress", tokA, nil); r.get("wins") != 1.0 || r.get("matches") != 1.0 {
		t.Fatalf("stats %s", r.raw)
	}
	if r := x.do(http.MethodGet, "/v1/me/progress", tokB, nil); r.get("xp_total") != 25.0 || r.get("level") != 1.0 {
		t.Fatalf("loser progress %s", r.raw)
	}
	r = x.do(http.MethodGet, "/v1/me/awards/"+m.MatchID.String(), tokA, nil, "Accept-Language", "tr")
	badges, _ := r.get("badges").([]any)
	if r.code != 200 || r.get("xp") != 120.0 || len(badges) != 2 {
		t.Fatalf("awards %s", r.raw)
	}
	r = x.do(http.MethodGet, "/v1/me/badges", tokA, nil)
	if earned, _ := r.get("badges").([]any); len(earned) != 2 {
		t.Fatalf("earned badges %s", r.raw)
	}
	r = x.do(http.MethodGet, "/v1/badges", tokB, nil, "Accept-Language", "tr")
	all, _ := r.get("badges").([]any)
	if len(all) < 20 {
		t.Fatalf("catalogue %s", r.raw)
	}
	for _, b := range all {
		if b, _ := b.(map[string]any); b["code"] == "first_win" && (b["earned_at"] != nil || b["name"] != "İlk Galibiyet") {
			t.Fatalf("first_win for the loser %v", b)
		}
	}

	// Quests: assigned by the match; at least one daily objective moved.
	r = x.do(http.MethodGet, "/v1/me/quests", tokA, nil)
	daily, _ := r.get("daily").([]any)
	weekly, _ := r.get("weekly").([]any)
	if r.code != 200 || len(daily) != 3 || len(weekly) != 3 || r.get("rerolls_left") != 1.0 {
		t.Fatalf("quests %s", r.raw)
	}
	// The day's quests are drawn per day: every quest a Quick Duel win
	// counts for has moved, the others have not.
	for _, q := range append(slices.Clone(daily), weekly...) {
		q, _ := q.(map[string]any)
		counts := slices.Contains([]any{"play_matches", "win_matches", "correct_answers", "win_streak"}, q["objective"])
		if moved := q["progress"].(float64) > 0; moved != counts {
			t.Fatalf("quest %v progress %v", q["code"], q["progress"])
		}
	}
	first, _ := daily[0].(map[string]any)
	second, _ := daily[1].(map[string]any)
	id := str(first["id"])
	want(t, "claim early", x.post("/v1/me/quests/"+id+"/claim", tokA, nil), 409, "QUEST_NOT_COMPLETED")
	if _, err := x.db.Exec(t.Context(), `UPDATE progression.quest_assignments SET progress = 99, completed_at = now() WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}
	r = x.post("/v1/me/quests/"+id+"/claim", tokA, nil)
	if r.code != 200 || r.get("xp") != first["reward_xp"] {
		t.Fatalf("claim %s", r.raw)
	}
	want(t, "claim twice", x.post("/v1/me/quests/"+id+"/claim", tokA, nil), 409, "QUEST_ALREADY_CLAIMED")
	want(t, "reroll claimed", x.post("/v1/me/quests/"+id+"/reroll", tokA, nil), 409, "QUEST_NOT_REROLLABLE")
	want(t, "someone else's", x.post("/v1/me/quests/"+id+"/claim", tokB, nil), 404, "NOT_FOUND")

	r = x.post("/v1/me/quests/"+str(second["id"])+"/reroll", tokA, nil)
	daily, _ = r.get("daily").([]any)
	if r.code != 200 || r.get("rerolls_left") != 0.0 || len(daily) != 3 {
		t.Fatalf("reroll %s", r.raw)
	}
	for _, q := range daily {
		if q, _ := q.(map[string]any); q["id"] == second["id"] {
			t.Fatal("rerolled quest still assigned")
		}
	}
	third, _ := daily[2].(map[string]any)
	want(t, "second reroll", x.post("/v1/me/quests/"+str(third["id"])+"/reroll", tokA, nil), 429, "QUEST_REROLL_LIMIT")

	// A leaver gets 5 XP per correct answer only and no streak day.
	tokC, c, _ := x.fullUser()
	x.progress(finishedDuel(a, c, true))
	if r := x.do(http.MethodGet, "/v1/me/progress", tokC, nil); r.get("xp_total") != 5.0 || r.get("streak", "current") != 0.0 {
		t.Fatalf("leaver progress %s", r.raw)
	}

	// Season badges: the peak league's badge and the global top 100.
	season, _, err := x.mods.ranking.ActiveSeason(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	x.consume(completed(t, season, "1v1", [2][]uuid.UUID{{a}, {b}}, [2]*uuid.UUID{}))
	cur, _ := x.mods.ranking.CurrentSeason(t.Context())
	later := ranking.New(ranking.Deps{DB: x.db, Redis: x.rdb, Clock: clock.NewFake(cur.EndsAt.Add(time.Minute)), Logger: discard})
	if err := later.Maintain(t.Context()); err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(ranking.SeasonEnded{SeasonID: season, Number: cur.Number})
	for range 2 {
		if err := x.mods.progression.OnSeasonEnded(t.Context(), events.Envelope{Name: ranking.EventSeasonEnded, Payload: payload}); err != nil {
			t.Fatal(err)
		}
	}
	r = x.do(http.MethodGet, "/v1/me/badges", tokA, nil)
	codes := map[string]bool{}
	earned, _ := r.get("badges").([]any)
	for _, b := range earned {
		b, _ := b.(map[string]any)
		codes[str(b["code"])] = true
	}
	if !codes["season_bronze"] || !codes["season_top_100"] {
		t.Fatalf("season badges %s", r.raw)
	}
}

// The worker pushes both halves of match.awards through realtime.
func TestMatchAwardsPushed(t *testing.T) {
	x := newE2E(t)
	tokA, a, _ := x.fullUser()
	_, b, _ := x.fullUser()
	w, _ := x.auth(tokA, "inst-awards", nil)
	push := workerPusher(x.rdb, discard)
	prog := progression.New(progression.Deps{DB: x.db, Pusher: push, Clock: clock.Real{}, Logger: discard})
	rk := ranking.New(ranking.Deps{DB: x.db, Redis: x.rdb, Pusher: push, Clock: clock.Real{}, Logger: discard})
	season, _, err := rk.ActiveSeason(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	p := completed(t, season, "1v1", [2][]uuid.UUID{{a}, {b}}, [2]*uuid.UUID{})
	payload, _ := json.Marshal(p)
	env := events.Envelope{Name: game.EventMatchCompleted, Payload: payload}
	if err := prog.OnMatchCompleted(t.Context(), env); err != nil {
		t.Fatal(err)
	}
	if err := rk.OnMatchCompleted(t.Context(), env); err != nil {
		t.Fatal(err)
	}
	got := map[string]map[string]any{}
	for range 2 {
		d := data(w.findWithin(5*time.Second, "match.awards"))
		if d["xp"] != nil {
			got["progression"] = d
		} else {
			got["ranking"] = d
		}
	}
	if got["progression"]["match_id"] != p.MatchID.String() || got["progression"]["xp"].(float64) < 49 {
		t.Fatalf("progression awards %v", got["progression"])
	}
	if got["ranking"]["sp_delta"] != 30.0 || got["ranking"]["league"] != "bronze" {
		t.Fatalf("ranking awards %v", got["ranking"])
	}
}
