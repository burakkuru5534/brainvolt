package app

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/commerce"
)

// testKeys is an SSV key source with one key.
type testKeys struct {
	id  int64
	key *ecdsa.PublicKey
}

func (k testKeys) Key(_ context.Context, id int64) (*ecdsa.PublicKey, error) {
	if id != k.id {
		return nil, fmt.Errorf("unknown key %d", id)
	}
	return k.key, nil
}

// ssvQuery is an AdMob SSV callback query signed with priv.
func ssvQuery(t *testing.T, priv *ecdsa.PrivateKey, reward string, user uuid.UUID, tx string) string {
	t.Helper()
	msg := fmt.Sprintf("ad_network=5450213213286189855&ad_unit=7982840839&custom_data=%s&reward_amount=1&reward_item=reward&timestamp=%d&transaction_id=%s&user_id=%s",
		reward, time.Now().UnixMilli(), tx, user)
	sum := sha256.Sum256([]byte(msg))
	sig, err := ecdsa.SignASN1(rand.Reader, priv, sum[:])
	if err != nil {
		t.Fatal(err)
	}
	return msg + "&signature=" + base64.RawURLEncoding.EncodeToString(sig) + "&key_id=42"
}

// TestRewardedAds covers D-103: reserve → AdMob's signed callback →
// progression's effect, the limits and the interstitial rules.
func TestRewardedAds(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	x := newE2E(t, func(d *moduleDeps) { d.adKeys = testKeys{id: 42, key: &priv.PublicKey} })
	tok, a, _ := x.fullUser()
	_, b, _ := x.fullUser()
	duel := finishedDuel(a, b, false)
	x.progress(duel)
	ctx := t.Context()

	// A new player sees no interstitials; nothing is repairable.
	c := x.do(http.MethodGet, "/v1/ads/config", tok, nil)
	if c.code != 200 || c.get("interstitial", "enabled") != false || c.get("interstitial", "every_matches") != 2.0 ||
		c.get("rewarded", "daily_limit") != 5.0 || c.get("rewarded", "streak_repair", "available") != false {
		t.Fatalf("config %s", c.raw)
	}

	// Bonus XP: half the match XP (70 → 35); the same reservation again.
	r := x.post("/v1/ads/rewards", tok, map[string]any{"placement": "bonus_xp", "ref": duel.MatchID})
	if r.code != 201 || r.get("reward", "xp") != 35.0 || r.get("status") != "pending" {
		t.Fatalf("reserve %s", r.raw)
	}
	id := r.get("id").(string)
	if again := x.post("/v1/ads/rewards", tok, map[string]any{"placement": "bonus_xp", "ref": duel.MatchID}); again.get("id") != id {
		t.Fatalf("not reused %s", again.raw)
	}
	want(t, "no ref", x.post("/v1/ads/rewards", tok, map[string]any{"placement": "bonus_xp"}), 400, "VALIDATION_FAILED")
	want(t, "unknown match", x.post("/v1/ads/rewards", tok, map[string]any{"placement": "bonus_xp", "ref": uuid.New()}), 409, "AD_REWARD_UNAVAILABLE")

	// AdMob's callback: a forged signature is refused, the real one grants once.
	q := ssvQuery(t, priv, id, a, "tx-1")
	forged := q[:len(q)-len("&key_id=42")-10] + "AAAAAAAAAA&key_id=42"
	want(t, "forged", x.do(http.MethodGet, "/v1/ads/admob/ssv?"+forged, "", nil), 403, "FORBIDDEN")
	other, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	want(t, "other key", x.do(http.MethodGet, "/v1/ads/admob/ssv?"+ssvQuery(t, other, id, a, "tx-1"), "", nil), 403, "FORBIDDEN")
	want(t, "wrong user", x.do(http.MethodGet, "/v1/ads/admob/ssv?"+ssvQuery(t, priv, id, b, "tx-0"), "", nil), 200, "")
	if g := x.do(http.MethodGet, "/v1/ads/rewards/"+id, tok, nil); g.get("status") != "pending" {
		t.Fatalf("granted to the wrong user %s", g.raw)
	}
	for range 2 {
		want(t, "ssv", x.do(http.MethodGet, "/v1/ads/admob/ssv?"+q, "", nil), 200, "")
	}
	want(t, "verify-url test", x.do(http.MethodGet, "/v1/ads/admob/ssv?"+ssvQuery(t, priv, "", uuid.Nil, ""), "", nil), 200, "")
	if g := x.do(http.MethodGet, "/v1/ads/rewards/"+id, tok, nil); g.get("status") != "granted" {
		t.Fatalf("not granted %s", g.raw)
	}
	var events int
	if err := x.db.QueryRow(ctx, `SELECT count(*) FROM river_job WHERE args->>'event' = $1 OR args::text LIKE '%' || $1 || '%'`, commerce.EventAdRewardGranted).Scan(&events); err != nil || events != 1 {
		t.Fatalf("events %d %v", events, err)
	}
	x.deliver(x.mods.progression.OnAdRewardGranted, commerce.EventAdRewardGranted,
		commerce.AdRewardGranted{RewardID: uuid.MustParse(id), UserID: a, Placement: "bonus_xp", Ref: duel.MatchID.String(), GrantedAt: time.Now()})
	if aw := x.do(http.MethodGet, "/v1/me/awards/"+duel.MatchID.String(), tok, nil); aw.get("xp") == nil || aw.get("xp").(float64) < 105 {
		t.Fatalf("bonus XP missing %s", aw.raw)
	}
	r = x.post("/v1/ads/rewards", tok, map[string]any{"placement": "bonus_xp", "ref": duel.MatchID})
	if r.code != 409 || r.get("error", "details", "reason") != "already_given" {
		t.Fatalf("bonus twice %s", r.raw)
	}

	// Quest reroll: only after the free one is used; adds one reroll.
	qs := x.do(http.MethodGet, "/v1/me/quests", tok, nil)
	quest := func(i int) string { return qs.get("daily").([]any)[i].(map[string]any)["id"].(string) }
	r = x.post("/v1/ads/rewards", tok, map[string]any{"placement": "quest_reroll"})
	if r.code != 409 || r.get("error", "details", "reason") != "free_reroll_left" {
		t.Fatalf("reroll with free left %s", r.raw)
	}
	want(t, "free reroll", x.post("/v1/me/quests/"+quest(0)+"/reroll", tok, nil), 200, "")
	r = x.post("/v1/ads/rewards", tok, map[string]any{"placement": "quest_reroll"})
	if r.code != 201 {
		t.Fatalf("reroll reserve %s", r.raw)
	}
	rid := r.get("id").(string)
	if g := x.post("/v1/ads/rewards/"+rid+"/dev-grant", tok, nil); g.get("status") != "granted" {
		t.Fatalf("dev grant %s", g.raw)
	}
	x.deliver(x.mods.progression.OnAdRewardGranted, commerce.EventAdRewardGranted,
		commerce.AdRewardGranted{RewardID: uuid.MustParse(rid), UserID: a, Placement: "quest_reroll", GrantedAt: time.Now()})
	if qs = x.do(http.MethodGet, "/v1/me/quests", tok, nil); qs.get("rerolls_left") != 1.0 {
		t.Fatalf("reroll bonus %s", qs.raw)
	}

	// Streak repair: lost by missing only yesterday; once a week.
	setStreak := func() {
		t.Helper()
		if _, err := x.db.Exec(ctx, `UPDATE progression.play_streaks SET current_days = 4, last_play_day = (now() AT TIME ZONE 'Europe/Istanbul')::date - 2 WHERE user_id = $1`, a); err != nil {
			t.Fatal(err)
		}
	}
	setStreak()
	if c = x.do(http.MethodGet, "/v1/ads/config", tok, nil); c.get("rewarded", "streak_repair", "available") != true || c.get("rewarded", "streak_repair", "days") != 4.0 {
		t.Fatalf("streak offer %s", c.raw)
	}
	r = x.post("/v1/ads/rewards", tok, map[string]any{"placement": "streak_repair"})
	sid := r.get("id").(string)
	x.post("/v1/ads/rewards/"+sid+"/dev-grant", tok, nil)
	x.deliver(x.mods.progression.OnAdRewardGranted, commerce.EventAdRewardGranted,
		commerce.AdRewardGranted{RewardID: uuid.MustParse(sid), UserID: a, Placement: "streak_repair", GrantedAt: time.Now()})
	if p := x.do(http.MethodGet, "/v1/me/progress", tok, nil); p.get("streak", "current") != 4.0 {
		t.Fatalf("streak not repaired %s", p.raw)
	}
	setStreak()
	want(t, "second repair this week", x.post("/v1/ads/rewards", tok, map[string]any{"placement": "streak_repair"}), 409, "AD_LIMIT_REACHED")

	// Five rewarded ads a day in all.
	for range 2 {
		if _, err := x.db.Exec(ctx, `INSERT INTO commerce.ad_rewards (id, user_id, placement, status, created_at, expires_at, granted_at)
			VALUES ($1, $2, 'bonus_xp', 'granted', now(), now(), now())`, uuid.New(), a); err != nil {
			t.Fatal(err)
		}
	}
	x.progress(finishedDuel(a, b, false))
	r = x.post("/v1/ads/rewards", tok, map[string]any{"placement": "quest_reroll"})
	if r.code != 409 {
		t.Fatalf("over the daily limit %s", r.raw)
	}
	if c = x.do(http.MethodGet, "/v1/ads/config", tok, nil); c.get("rewarded", "used_today") != 5.0 || c.get("rewarded", "bonus_xp_left") != 0.0 {
		t.Fatalf("used %s", c.raw)
	}

	// Interstitials after three days, never for Plus.
	if _, err := x.db.Exec(ctx, `UPDATE identity.users SET created_at = now() - interval '4 days' WHERE id = $1`, a); err != nil {
		t.Fatal(err)
	}
	if c = x.do(http.MethodGet, "/v1/ads/config", tok, nil); c.get("interstitial", "enabled") != true {
		t.Fatalf("interstitial %s", c.raw)
	}
	if _, err := x.db.Exec(ctx, `INSERT INTO commerce.entitlements (user_id, code, source_kind, source_id, starts_at) VALUES ($1, 'plus', 'admin', 'test', now())`, a); err != nil {
		t.Fatal(err)
	}
	if ok, err := x.mods.commerce.HasEntitlement(ctx, a, "plus"); err != nil || !ok {
		t.Fatalf("entitlement %v %v", ok, err)
	}
	if c = x.do(http.MethodGet, "/v1/ads/config", tok, nil); c.get("interstitial", "enabled") != false {
		t.Fatalf("plus sees interstitials %s", c.raw)
	}
}
