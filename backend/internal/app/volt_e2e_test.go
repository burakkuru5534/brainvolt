package app

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/tournament"
)

// TestVolt covers D-105: Volt earned by playing, the store's items, the
// store's rewarded ad and Plus's monthly Volt.
func TestVolt(t *testing.T) {
	x := newE2E(t, func(d *moduleDeps) { d.rcSecret = "rc-secret" })
	tok, a, _ := x.fullUser()
	tokB, b, _ := x.fullUser()
	ctx := t.Context()
	balance := func(tok string) float64 {
		t.Helper()
		r := x.do(http.MethodGet, "/v1/me/wallet", tok, nil)
		if r.code != 200 {
			t.Fatalf("wallet %s", r.raw)
		}
		return r.get("balance").(float64)
	}
	if w := x.do(http.MethodGet, "/v1/me/wallet", tok, nil); w.get("balance") != 0.0 || len(w.get("items").([]any)) != 2 || w.get("ad_volt_left") != 3.0 {
		t.Fatalf("empty wallet %s", w.raw)
	}

	// A won duel reaches level 2: 25 Volt, once.
	x.progress(finishedDuel(a, b, false))
	if got := balance(tok); got != 25 {
		t.Fatalf("level-up volt %v", got)
	}
	w := x.do(http.MethodGet, "/v1/me/wallet", tok, nil)
	if h := w.get("history").([]any); len(h) != 1 || h[0].(map[string]any)["reason"] != "level_up" || h[0].(map[string]any)["ref"] != "2" {
		t.Fatalf("history %s", w.raw)
	}

	// A claimed daily quest: +10 (shown on the quest and in the claim).
	q := x.do(http.MethodGet, "/v1/me/quests", tok, nil)
	first := q.get("daily").([]any)[0].(map[string]any)
	if first["reward_volt"] != 10.0 || q.get("weekly").([]any)[0].(map[string]any)["reward_volt"] != 50.0 {
		t.Fatalf("quest volt %s", q.raw)
	}
	if _, err := x.db.Exec(ctx, `UPDATE progression.quest_assignments SET completed_at = now() WHERE id = $1`, first["id"]); err != nil {
		t.Fatal(err)
	}
	if c := x.post("/v1/me/quests/"+first["id"].(string)+"/claim", tok, nil); c.code != 200 || c.get("volt") != 10.0 {
		t.Fatalf("claim %s", c.raw)
	}

	// The Daily Challenge (+10) and a tournament win (+300), once each.
	run := game.DailyRunCompleted{RunID: uuid.New(), UserID: a, CorrectCount: 5, CompletedAt: time.Now().UTC()}
	x.deliver(x.mods.progression.OnDailyRunCompleted, game.EventDailyRunCompleted, run)
	x.deliver(x.mods.progression.OnTournamentFinished, tournament.EventFinished, map[string]any{
		"tournament_id": uuid.New(), "placements": []map[string]any{{"user_id": a, "place": 1, "xp": 0}, {"user_id": b, "place": 4, "xp": 0}},
	})
	// The quest and the run's XP also reach level 3 (+25).
	if got := balance(tok); got != 370 {
		t.Fatalf("after quest, daily and tournament %v", got)
	}
	if got := balance(tokB); got != 0 {
		t.Fatalf("4th place earned %v", got)
	}

	// Streak shields: 100 each, at most 2 held; the third is refused and
	// costs nothing.
	for i, left := range []float64{270, 170} {
		r := x.post("/v1/store/items/streak_shield/buy", tok, nil)
		if r.code != 200 || r.get("balance") != left {
			t.Fatalf("shield %d %s", i, r.raw)
		}
	}
	want(t, "shields full", x.post("/v1/store/items/streak_shield/buy", tok, nil), 409, "ITEM_UNAVAILABLE")
	if p := x.do(http.MethodGet, "/v1/me/progress", tok, nil); p.get("streak", "shields") != 2.0 || p.get("streak", "max_shields") != 2.0 {
		t.Fatalf("shields %s", p.raw)
	}
	// Two missed days: the shields keep the streak alive.
	if _, err := x.db.Exec(ctx, `UPDATE progression.play_streaks SET last_play_day = last_play_day - 3 WHERE user_id = $1`, a); err != nil {
		t.Fatal(err)
	}
	if p := x.do(http.MethodGet, "/v1/me/progress", tok, nil); p.get("streak", "current") != 1.0 {
		t.Fatalf("shielded streak %s", p.raw)
	}

	// An extra reroll: only after the free one; 30 Volt.
	want(t, "free reroll left", x.post("/v1/store/items/quest_reroll/buy", tok, nil), 409, "ITEM_UNAVAILABLE")
	open := q.get("daily").([]any)[1].(map[string]any)["id"].(string)
	if r := x.post("/v1/me/quests/"+open+"/reroll", tok, nil); r.code != 200 {
		t.Fatalf("free reroll %s", r.raw)
	}
	if r := x.post("/v1/store/items/quest_reroll/buy", tok, nil); r.code != 200 || r.get("balance") != 140.0 {
		t.Fatalf("reroll %s", r.raw)
	}
	if r := x.do(http.MethodGet, "/v1/me/quests", tok, nil); r.get("rerolls_left") != 1.0 {
		t.Fatalf("bought reroll %s", r.raw)
	}

	// Without Volt nothing is bought, and the item's effect is undone.
	want(t, "poor", x.post("/v1/store/items/streak_shield/buy", tokB, nil), 409, "VOLT_INSUFFICIENT")
	if p := x.do(http.MethodGet, "/v1/me/progress", tokB, nil); p.get("streak", "shields") != 0.0 {
		t.Fatalf("shield without paying %s", p.raw)
	}
	want(t, "unknown item", x.post("/v1/store/items/crown/buy", tokB, nil), 404, "NOT_FOUND")

	// The store's ad: 5 Volt, 3 a day, outside the other ads' limit.
	for i := range 3 {
		r := x.post("/v1/ads/rewards", tokB, map[string]any{"placement": "volt"})
		if r.code != 201 || r.get("reward", "volt") != 5.0 {
			t.Fatalf("reserve %d %s", i, r.raw)
		}
		if g := x.post("/v1/ads/rewards/"+r.get("id").(string)+"/dev-grant", tokB, nil); g.get("status") != "granted" {
			t.Fatalf("grant %s", g.raw)
		}
	}
	want(t, "4th volt ad", x.post("/v1/ads/rewards", tokB, map[string]any{"placement": "volt"}), 409, "AD_LIMIT_REACHED")
	if got := balance(tokB); got != 15 {
		t.Fatalf("ad volt %v", got)
	}
	if c := x.do(http.MethodGet, "/v1/ads/config", tokB, nil); c.get("rewarded", "used_today") != 0.0 {
		t.Fatalf("volt ads counted %s", c.raw)
	}

	// Plus: no Volt in a trial, 400 a month once paying, taken back on refund.
	month := time.Now().Add(30 * 24 * time.Hour)
	trial := plusEvent("INITIAL_PURCHASE", b, "tx-b", month)
	trial["period_type"] = "TRIAL"
	want(t, "trial", x.rcWebhook("rc-secret", trial), 200, "")
	if got := balance(tokB); got != 15 {
		t.Fatalf("trial volt %v", got)
	}
	paid := plusEvent("RENEWAL", b, "tx-b", month)
	paid["period_type"] = "NORMAL"
	want(t, "paid", x.rcWebhook("rc-secret", paid), 200, "")
	balance(tokB)
	if got := balance(tokB); got != 415 {
		t.Fatalf("plus volt %v", got)
	}
	if p := x.do(http.MethodGet, "/v1/me/progress", tokB, nil); p.get("streak", "shields") != 1.0 {
		t.Fatalf("plus shield %s", p.raw)
	}
	refund := plusEvent("CANCELLATION", b, "tx-b", month)
	refund["cancel_reason"] = "CUSTOMER_SUPPORT"
	want(t, "refund", x.rcWebhook("rc-secret", refund), 200, "")
	if got := balance(tokB); got != 15 {
		t.Fatalf("after refund %v", got)
	}
}
