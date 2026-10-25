package app

import (
	"net/http"
	"testing"
	"time"
)

// TestCosmetics covers D-106: buying cosmetics with Volt, wearing them,
// special avatars, Plus-only items and what other players see.
func TestCosmetics(t *testing.T) {
	x := newE2E(t, func(d *moduleDeps) { d.rcSecret = "rc-secret" })
	tok, a, _ := x.fullUser()
	tokB, _, _ := x.fullUser()
	ctx := t.Context()
	if _, err := x.db.Exec(ctx, `INSERT INTO commerce.volt_wallets (user_id, balance, updated_at) VALUES ($1, 1000, now())`, a); err != nil {
		t.Fatal(err)
	}
	put := func(tok, kind string, code any) response {
		return x.do(http.MethodPut, "/v1/me/cosmetics/"+kind, tok, map[string]any{"code": code})
	}
	card := func() response {
		r := x.do(http.MethodGet, "/v1/profiles/"+a.String(), tokB, nil)
		if r.code != 200 {
			t.Fatalf("profile %s", r.raw)
		}
		return r
	}

	c := x.do(http.MethodGet, "/v1/store/cosmetics", tok, nil)
	if c.code != 200 || len(c.get("items").([]any)) != 22 || c.get("plus") != false {
		t.Fatalf("catalogue %s", c.raw)
	}

	// Buying: once, with enough Volt, never Plus-only items.
	if r := x.post("/v1/store/cosmetics/frame_neon/buy", tok, nil); r.code != 200 || r.get("balance") != 700.0 {
		t.Fatalf("buy %s", r.raw)
	}
	want(t, "twice", x.post("/v1/store/cosmetics/frame_neon/buy", tok, nil), 409, "ITEM_UNAVAILABLE")
	want(t, "plus only", x.post("/v1/store/cosmetics/frame_plus/buy", tok, nil), 409, "ITEM_UNAVAILABLE")
	want(t, "unknown", x.post("/v1/store/cosmetics/frame_gold/buy", tok, nil), 404, "NOT_FOUND")
	want(t, "poor", x.post("/v1/store/cosmetics/frame_crown/buy", tokB, nil), 409, "VOLT_INSUFFICIENT")
	if r := x.do(http.MethodGet, "/v1/store/cosmetics", tokB, nil); r.get("items").([]any)[3].(map[string]any)["owned"] != false {
		t.Fatalf("owned without paying %s", r.raw)
	}

	// Wearing: owned items of the right kind; others see frames and name colours.
	want(t, "not owned", put(tok, "frame", "frame_crown"), 409, "ITEM_UNAVAILABLE")
	want(t, "wrong kind", put(tok, "name_color", "frame_neon"), 400, "VALIDATION_FAILED")
	want(t, "unknown kind", put(tok, "hat", "frame_neon"), 404, "NOT_FOUND")
	if r := put(tok, "frame", "frame_neon"); r.code != 200 || r.get("equipped", "frame") != "frame_neon" {
		t.Fatalf("equip %s", r.raw)
	}
	if p := card(); p.get("frame") != "frame_neon" || p.get("name_color") != nil {
		t.Fatalf("card %s", p.raw)
	}

	// A special avatar is worn through the profile once bought.
	want(t, "avatar not owned", x.do(http.MethodPatch, "/v1/me/profile", tok, map[string]any{"avatar_id": "dragon"}), 400, "VALIDATION_FAILED")
	if r := x.post("/v1/store/cosmetics/avatar_dragon/buy", tok, nil); r.code != 200 || r.get("balance") != 450.0 {
		t.Fatalf("buy avatar %s", r.raw)
	}
	if r := x.do(http.MethodPatch, "/v1/me/profile", tok, map[string]any{"avatar_id": "dragon"}); r.code != 200 {
		t.Fatalf("wear avatar %s", r.raw)
	}
	if p := card(); p.get("avatar_id") != "dragon" {
		t.Fatalf("avatar card %s", p.raw)
	}

	// Plus-only items: worn while Plus lasts, hidden once it ends.
	want(t, "rainbow without plus", put(tok, "name_color", "name_rainbow"), 409, "ITEM_UNAVAILABLE")
	month := time.Now().Add(30 * 24 * time.Hour)
	want(t, "plus", x.rcWebhook("rc-secret", plusEvent("INITIAL_PURCHASE", a, "tx-c", month)), 200, "")
	if r := put(tok, "name_color", "name_rainbow"); r.code != 200 || r.get("equipped", "name_color") != "name_rainbow" {
		t.Fatalf("rainbow %s", r.raw)
	}
	if p := card(); p.get("name_color") != "name_rainbow" {
		t.Fatalf("rainbow card %s", p.raw)
	}
	want(t, "expire", x.rcWebhook("rc-secret", plusEvent("EXPIRATION", a, "tx-c", time.Now().Add(-time.Second))), 200, "")
	if p := card(); p.get("name_color") != nil || p.get("frame") != "frame_neon" {
		t.Fatalf("after plus %s", p.raw)
	}

	// Taking a frame off.
	if r := put(tok, "frame", nil); r.code != 200 || r.get("equipped", "frame") != nil {
		t.Fatalf("unequip %s", r.raw)
	}
	if p := card(); p.get("frame") != nil {
		t.Fatalf("frame left on %s", p.raw)
	}
}
