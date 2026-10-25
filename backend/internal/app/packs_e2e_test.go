package app

import (
	"net/http"
	"slices"
	"testing"

	"github.com/google/uuid"
)

func packEvent(kind string, user uuid.UUID, product, tx string) map[string]any {
	return map[string]any{
		"type": kind, "app_user_id": user.String(), "product_id": product,
		"transaction_id": tx, "environment": "SANDBOX",
	}
}

// TestVoltPacks covers D-107: Volt packs bought through RevenueCat, the
// once-only starter pack with its frame, and refunds.
func TestVoltPacks(t *testing.T) {
	x := newE2E(t, func(d *moduleDeps) { d.rcSecret = "rc-secret" })
	tok, a, _ := x.fullUser()
	wallet := func() response {
		t.Helper()
		r := x.do(http.MethodGet, "/v1/me/wallet", tok, nil)
		if r.code != 200 {
			t.Fatalf("wallet %s", r.raw)
		}
		return r
	}
	packs := func(r response) []string {
		var out []string
		for _, p := range r.get("packs").([]any) {
			out = append(out, p.(map[string]any)["product_id"].(string))
		}
		return out
	}
	ownsStarter := func() bool {
		r := x.do(http.MethodGet, "/v1/store/cosmetics", tok, nil)
		for _, it := range r.get("items").([]any) {
			if m := it.(map[string]any); m["code"] == "frame_starter" {
				return m["owned"] == true
			}
		}
		t.Fatal("no starter frame in the catalogue")
		return false
	}
	if p := packs(wallet()); !slices.Equal(p, []string{"volt_starter", "volt_100", "volt_350", "volt_800", "volt_2000"}) {
		t.Fatalf("packs %v", p)
	}

	// A pack: once per store transaction, even under another event id.
	ev := packEvent("NON_RENEWING_PURCHASE", a, "volt_350", "GPA.1")
	for range 2 {
		want(t, "buy", x.rcWebhook("rc-secret", ev), 200, "")
	}
	want(t, "same tx", x.rcWebhook("rc-secret", packEvent("NON_RENEWING_PURCHASE", a, "volt_350", "GPA.1")), 200, "")
	if b := wallet().get("balance"); b != 350.0 {
		t.Fatalf("after a pack %v", b)
	}

	// The starter pack: 250 Volt and the starter frame, then no longer offered.
	want(t, "starter", x.rcWebhook("rc-secret", packEvent("NON_RENEWING_PURCHASE", a, "volt_starter", "GPA.2")), 200, "")
	w := wallet()
	if w.get("balance") != 600.0 || slices.Contains(packs(w), "volt_starter") || !ownsStarter() {
		t.Fatalf("starter %s", w.raw)
	}
	want(t, "starter frame for volt", x.post("/v1/store/cosmetics/frame_starter/buy", tok, nil), 409, "ITEM_UNAVAILABLE")
	if r := x.do(http.MethodPut, "/v1/me/cosmetics/frame", tok, map[string]any{"code": "frame_starter"}); r.code != 200 {
		t.Fatalf("wear starter %s", r.raw)
	}

	// Refunds take the Volt (and the frame) back; unknown ones change nothing.
	refund := func(product, tx string) map[string]any {
		e := packEvent("CANCELLATION", a, product, tx)
		e["cancel_reason"] = "CUSTOMER_SUPPORT"
		return e
	}
	want(t, "refund", x.rcWebhook("rc-secret", refund("volt_350", "GPA.1")), 200, "")
	want(t, "unknown refund", x.rcWebhook("rc-secret", refund("volt_800", "GPA.9")), 200, "")
	if b := wallet().get("balance"); b != 250.0 {
		t.Fatalf("after refund %v", b)
	}
	want(t, "starter refund", x.rcWebhook("rc-secret", refund("volt_starter", "GPA.2")), 200, "")
	w = wallet()
	if w.get("balance") != 0.0 || ownsStarter() || !slices.Contains(packs(w), "volt_starter") {
		t.Fatalf("after starter refund %s", w.raw)
	}
	if r := x.do(http.MethodGet, "/v1/store/cosmetics", tok, nil); r.get("equipped", "frame") != nil {
		t.Fatalf("refunded frame still worn %s", r.raw)
	}
	h := w.get("history").([]any)
	if h[0].(map[string]any)["reason"] != "purchase_refund" {
		t.Fatalf("history %s", w.raw)
	}
}
