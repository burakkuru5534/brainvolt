package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

// rcWebhook posts a RevenueCat event with auth as the Authorization header.
func (x *e2e) rcWebhook(auth string, event map[string]any) response {
	x.t.Helper()
	if _, ok := event["id"]; !ok {
		event["id"] = uuid.NewString()
	}
	if _, ok := event["event_timestamp_ms"]; !ok {
		event["event_timestamp_ms"] = time.Now().UnixMilli()
	}
	body, _ := json.Marshal(map[string]any{"api_version": "1.0", "event": event})
	req := httptest.NewRequest(http.MethodPost, "/v1/store/revenuecat/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	rec := httptest.NewRecorder()
	x.srv.Handler().ServeHTTP(rec, req)
	out := response{code: rec.Code, raw: rec.Body.String()}
	_ = json.Unmarshal(rec.Body.Bytes(), &out.body)
	return out
}

func plusEvent(kind string, user uuid.UUID, tx string, exp time.Time) map[string]any {
	return map[string]any{
		"type": kind, "app_user_id": user.String(), "product_id": "brainvolt_plus:monthly",
		"entitlement_ids": []string{"plus"}, "original_transaction_id": tx, "environment": "SANDBOX",
		"purchased_at_ms": time.Now().Add(-time.Minute).UnixMilli(), "expiration_at_ms": exp.UnixMilli(),
	}
}

// TestPlusSubscription covers D-104: RevenueCat's webhook turns the plus
// entitlement on and off, and Plus changes ads and quests.
func TestPlusSubscription(t *testing.T) {
	x := newE2E(t, func(d *moduleDeps) { d.rcSecret = "rc-secret" })
	tok, a, _ := x.fullUser()
	tokB, b, _ := x.fullUser()
	ctx := t.Context()
	if _, err := x.db.Exec(ctx, `UPDATE identity.users SET created_at = now() - interval '4 days' WHERE id = ANY($1)`, []uuid.UUID{a, b}); err != nil {
		t.Fatal(err)
	}
	plus := func(tok string) bool {
		t.Helper()
		r := x.do(http.MethodGet, "/v1/me/entitlements", tok, nil)
		if r.code != 200 {
			t.Fatalf("entitlements %s", r.raw)
		}
		return r.get("plus") == true
	}
	month := time.Now().Add(30 * 24 * time.Hour)

	// Before: interstitials on, three daily quests, one free reroll.
	if c := x.do(http.MethodGet, "/v1/ads/config", tok, nil); c.get("interstitial", "enabled") != true {
		t.Fatalf("config %s", c.raw)
	}
	q := x.do(http.MethodGet, "/v1/me/quests", tok, nil)
	if len(q.get("daily").([]any)) != 3 || q.get("rerolls_left") != 1.0 {
		t.Fatalf("quests %s", q.raw)
	}

	want(t, "no auth", x.rcWebhook("", plusEvent("INITIAL_PURCHASE", a, "tx-a", month)), 401, "UNAUTHENTICATED")
	want(t, "wrong auth", x.rcWebhook("Bearer nope", plusEvent("INITIAL_PURCHASE", a, "tx-a", month)), 401, "UNAUTHENTICATED")
	ev := plusEvent("INITIAL_PURCHASE", a, "tx-a", month)
	for range 2 { // redelivered
		want(t, "purchase", x.rcWebhook("rc-secret", ev), 200, "")
	}
	if !plus(tok) {
		t.Fatal("no plus after purchase")
	}
	r := x.do(http.MethodGet, "/v1/me/entitlements", tok, nil)
	if until, _ := time.Parse(time.RFC3339Nano, r.get("plus_until").(string)); until.Sub(month).Abs() > time.Second {
		t.Fatalf("plus_until %s", r.raw)
	}

	// With Plus: no interstitials, a fourth daily quest, two free rerolls,
	// rewarded-ad rewards without the ad.
	if c := x.do(http.MethodGet, "/v1/ads/config", tok, nil); c.get("interstitial", "enabled") != false {
		t.Fatalf("plus config %s", c.raw)
	}
	q = x.do(http.MethodGet, "/v1/me/quests", tok, nil)
	if len(q.get("daily").([]any)) != 4 || q.get("rerolls_left") != 2.0 {
		t.Fatalf("plus quests %s", q.raw)
	}
	duel := finishedDuel(a, b, false)
	x.progress(duel)
	rw := x.post("/v1/ads/rewards", tok, map[string]any{"placement": "bonus_xp", "ref": duel.MatchID})
	if rw.code != 201 || rw.get("status") != "granted" {
		t.Fatalf("plus reward %s", rw.raw)
	}

	// A cancellation keeps Plus until the period ends; the expiration ends it.
	want(t, "cancel", x.rcWebhook("rc-secret", plusEvent("CANCELLATION", a, "tx-a", month)), 200, "")
	if !plus(tok) {
		t.Fatal("cancellation ended plus early")
	}
	want(t, "expire", x.rcWebhook("rc-secret", plusEvent("EXPIRATION", a, "tx-a", time.Now().Add(-time.Second))), 200, "")
	if plus(tok) {
		t.Fatal("plus after expiration")
	}

	// A refund ends a new subscription at once.
	want(t, "again", x.rcWebhook("rc-secret", plusEvent("INITIAL_PURCHASE", a, "tx-a2", month)), 200, "")
	refund := plusEvent("CANCELLATION", a, "tx-a2", month)
	refund["cancel_reason"] = "CUSTOMER_SUPPORT"
	want(t, "refund", x.rcWebhook("rc-secret", refund), 200, "")
	if plus(tok) {
		t.Fatal("plus after refund")
	}

	// A transfer (restore on another account) moves Plus.
	want(t, "buy 3", x.rcWebhook("rc-secret", plusEvent("RENEWAL", a, "tx-a3", month)), 200, "")
	want(t, "transfer", x.rcWebhook("rc-secret", map[string]any{
		"type": "TRANSFER", "app_user_id": b.String(), "transferred_from": []string{a.String()}, "transferred_to": []string{b.String()},
	}), 200, "")
	if plus(tok) || !plus(tokB) {
		t.Fatal("transfer did not move plus")
	}

	// Unknown users and other products are recorded and ignored.
	anon := plusEvent("INITIAL_PURCHASE", a, "tx-x", month)
	anon["app_user_id"] = "$RCAnonymousID:1234"
	want(t, "anonymous", x.rcWebhook("rc-secret", anon), 200, "")
	var n int
	if err := x.db.QueryRow(ctx, `SELECT count(*) FROM commerce.store_events`).Scan(&n); err != nil || n != 8 {
		t.Fatalf("store events %d %v", n, err)
	}
}

func TestPlusWebhookOff(t *testing.T) {
	x := newE2E(t)
	_, a, _ := x.fullUser()
	want(t, "off", x.rcWebhook("anything", plusEvent("INITIAL_PURCHASE", a, "tx", time.Now().Add(time.Hour))), 503, "UNAVAILABLE")
}
