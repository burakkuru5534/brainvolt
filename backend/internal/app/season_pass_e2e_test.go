package app

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// TestSeasonPass covers D-108: tiers from season XP, the free track, the
// premium track bought with Volt, claims once and season-only cosmetics.
func TestSeasonPass(t *testing.T) {
	x := newE2E(t)
	tok, a, _ := x.fullUser()
	ctx := t.Context()
	addXP := func(n int) {
		t.Helper()
		if _, err := x.db.Exec(ctx, `INSERT INTO progression.xp_ledger (id, user_id, amount, source, source_ref, created_at)
			VALUES ($1, $2, $3, 'match', $4, now())`, uuid.New(), a, n, uuid.NewString()); err != nil {
			t.Fatal(err)
		}
	}
	balance := func() float64 {
		t.Helper()
		return x.do(http.MethodGet, "/v1/me/wallet", tok, nil).get("balance").(float64)
	}

	p := x.do(http.MethodGet, "/v1/me/season-pass", tok, nil)
	if p.code != 200 || p.get("tier") != 0.0 || len(p.get("tiers").([]any)) != 40 || p.get("price") != 900.0 || p.get("premium") != false {
		t.Fatalf("empty pass %s", p.raw)
	}

	// 5,200 XP: tier 10; the free track gives tier 5 (50 Volt) and 10 (a shield).
	addXP(5200)
	p = x.do(http.MethodGet, "/v1/me/season-pass", tok, nil)
	if p.get("tier") != 10.0 || p.get("tier_xp") != 200.0 || p.get("claimable") != 2.0 {
		t.Fatalf("tier 10 %s", p.raw)
	}
	c := x.post("/v1/me/season-pass/claim", tok, nil)
	if c.code != 200 || c.get("volt") != 50.0 || c.get("shields") != 1.0 || c.get("pass", "claimable") != 0.0 {
		t.Fatalf("free claim %s", c.raw)
	}
	if again := x.post("/v1/me/season-pass/claim", tok, nil); again.get("volt") != 0.0 {
		t.Fatalf("claimed twice %s", again.raw)
	}

	// The premium track: 900 Volt, once.
	want(t, "poor", x.post("/v1/me/season-pass/premium", tok, nil), 409, "VOLT_INSUFFICIENT")
	if _, err := x.db.Exec(ctx, `UPDATE commerce.volt_wallets SET balance = 1000 WHERE user_id = $1`, a); err != nil {
		t.Fatal(err)
	}
	if b := x.post("/v1/me/season-pass/premium", tok, nil); b.code != 200 || b.get("premium") != true || b.get("claimable") != 7.0 {
		t.Fatalf("premium %s", b.raw)
	}
	if got := balance(); got != 100 {
		t.Fatalf("after premium %v", got)
	}
	want(t, "twice", x.post("/v1/me/season-pass/premium", tok, nil), 409, "ITEM_UNAVAILABLE")

	// Premium tiers 1–10: the storm frame and avatar, 4 × 60 Volt, a shield.
	c = x.post("/v1/me/season-pass/claim", tok, nil)
	cos := c.get("cosmetics").([]any)
	if c.get("volt") != 240.0 || c.get("shields") != 1.0 || len(cos) != 2 || cos[0] != "frame_s1_storm" || cos[1] != "avatar_stormcloud" {
		t.Fatalf("premium claim %s", c.raw)
	}
	want(t, "season only", x.post("/v1/store/cosmetics/frame_s1_storm/buy", tok, nil), 409, "ITEM_UNAVAILABLE")
	if r := x.do(http.MethodPut, "/v1/me/cosmetics/frame", tok, map[string]any{"code": "frame_s1_storm"}); r.code != 200 {
		t.Fatalf("wear storm frame %s", r.raw)
	}
	if r := x.do(http.MethodPatch, "/v1/me/profile", tok, map[string]any{"avatar_id": "stormcloud"}); r.code != 200 {
		t.Fatalf("storm avatar %s", r.raw)
	}

	// Tier 15 with two shields held: the shield becomes 30 Volt.
	addXP(2500)
	c = x.post("/v1/me/season-pass/claim", tok, nil)
	if c.get("volt") != 225.0 || c.get("shields") != 0.0 {
		t.Fatalf("tier 15 claim %s", c.raw)
	}
	if got := balance(); got != 100+240+225 {
		t.Fatalf("balance %v", got)
	}
}
