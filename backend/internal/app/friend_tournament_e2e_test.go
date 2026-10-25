package app

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/tournament"
)

// TestFriendTournaments covers D-109: set-up fee (free with Plus), only
// the owner's friends see and join, the owner cancels with a refund, an
// empty one is cancelled with a refund, no rewards, trophies on profiles.
func TestFriendTournaments(t *testing.T) {
	x := newE2E(t, func(d *moduleDeps) { d.rcSecret = "rc-secret" })
	tokA, a, _ := x.fullUser()
	tokB, b, _ := x.fullUser()
	tokC, _, _ := x.fullUser()
	guest, _ := x.signedIn()
	ctx := t.Context()
	x.befriend(tokA, b, tokB)
	setVolt := func(u uuid.UUID, v int) {
		t.Helper()
		if _, err := x.db.Exec(ctx, `INSERT INTO commerce.volt_wallets (user_id, balance, updated_at) VALUES ($1, $2, now())
			ON CONFLICT (user_id) DO UPDATE SET balance = EXCLUDED.balance`, u, v); err != nil {
			t.Fatal(err)
		}
	}
	balance := func(tok string) float64 {
		t.Helper()
		return x.do(http.MethodGet, "/v1/me/wallet", tok, nil).get("balance").(float64)
	}
	body := map[string]any{"name": "Cuma Kapışması", "match_type": "quick_duel", "max_entrants": 8,
		"starts_at": time.Now().Add(2 * time.Hour)}

	want(t, "guest", x.post("/v1/tournaments/friends", guest, body), 403, "ACCOUNT_UPGRADE_REQUIRED")
	want(t, "size", x.post("/v1/tournaments/friends", tokA, merge(body, map[string]any{"max_entrants": 6})), 400, "VALIDATION_FAILED")
	want(t, "too soon", x.post("/v1/tournaments/friends", tokA, merge(body, map[string]any{"starts_at": time.Now().Add(5 * time.Minute)})), 400, "VALIDATION_FAILED")
	want(t, "ranked", x.post("/v1/tournaments/friends", tokA, merge(body, map[string]any{"match_type": "ranked"})), 400, "VALIDATION_FAILED")
	want(t, "no volt", x.post("/v1/tournaments/friends", tokA, body), 409, "VOLT_INSUFFICIENT")
	setVolt(a, 500)
	r := x.post("/v1/tournaments/friends", tokA, body)
	if r.code != 201 || r.get("kind") != "friends" || r.get("is_owner") != true || r.get("status") != "registration" || r.get("max_entrants") != 8.0 {
		t.Fatalf("create %s", r.raw)
	}
	id := str(r.get("id"))
	path := "/v1/tournaments/" + id
	if got := balance(tokA); got != 200 {
		t.Fatalf("fee %v", got)
	}

	// Friends see and join it; others do not.
	listed := func(tok string) bool {
		for _, it := range x.do(http.MethodGet, "/v1/tournaments", tok, nil).get("tournaments").([]any) {
			if it.(map[string]any)["id"] == id {
				return true
			}
		}
		return false
	}
	if !listed(tokA) || !listed(tokB) || listed(tokC) {
		t.Fatal("friend tournament visibility")
	}
	want(t, "stranger get", x.do(http.MethodGet, path, tokC, nil), 404, "NOT_FOUND")
	want(t, "stranger join", x.post(path+"/entries", tokC, nil), 404, "NOT_FOUND")
	if r := x.post(path+"/entries", tokB, nil); r.code != 201 {
		t.Fatalf("friend joins %s", r.raw)
	}
	if r := x.post(path+"/entries", tokA, nil); r.code != 201 {
		t.Fatalf("owner joins %s", r.raw)
	}
	if d := x.do(http.MethodGet, path, tokB, nil); d.get("is_owner") != nil || d.get("entrants") != 2.0 {
		t.Fatalf("friend view %s", d.raw)
	}

	// Only the owner cancels, before the start; the fee comes back once.
	want(t, "not owner", x.do(http.MethodDelete, path, tokB, nil), 404, "NOT_FOUND")
	want(t, "cancel", x.do(http.MethodDelete, path, tokA, nil), 204, "")
	want(t, "cancel twice", x.do(http.MethodDelete, path, tokA, nil), 409, "MATCH_STATE_INVALID")
	if got := balance(tokA); got != 500 {
		t.Fatalf("refund %v", got)
	}

	// Plus sets up for free; two open ones at most; one nobody checks in to
	// is cancelled at the start and costs nothing.
	want(t, "plus", x.rcWebhook("rc-secret", plusEvent("INITIAL_PURCHASE", a, "tx-ft", time.Now().Add(30*24*time.Hour))), 200, "")
	first := x.post("/v1/tournaments/friends", tokA, body)
	second := x.post("/v1/tournaments/friends", tokA, body)
	// Plus also brings the month's 400 Volt.
	if first.code != 201 || second.code != 201 || balance(tokA) != 900 {
		t.Fatalf("plus set-up %s %s", first.raw, second.raw)
	}
	want(t, "third", x.post("/v1/tournaments/friends", tokA, body), 409, "ITEM_UNAVAILABLE")
	if _, err := x.db.Exec(ctx, `UPDATE tournament.tournaments SET fee = 300 WHERE id = $1`, str(first.get("id"))); err != nil {
		t.Fatal(err) // as if paid before Plus
	}
	if _, err := x.db.Exec(ctx, `UPDATE tournament.tournaments SET registration_opens_at = now() - interval '3 seconds',
		registration_closes_at = now() - interval '2 seconds', starts_at = now() - interval '1 second' WHERE id = $1`, str(first.get("id"))); err != nil {
		t.Fatal(err)
	}
	if err := x.mods.tournament.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if d := x.do(http.MethodGet, "/v1/tournaments/"+str(first.get("id")), tokA, nil); d.get("status") != "cancelled" || balance(tokA) != 1200 {
		t.Fatalf("auto-cancel %s", d.raw)
	}

	// No rewards from a friend tournament; the winner's trophy is on the profile.
	x.deliver(x.mods.progression.OnTournamentFinished, tournament.EventFinished, map[string]any{
		"tournament_id": second.get("id"), "kind": "friends",
		"placements": []map[string]any{{"user_id": b, "place": 1, "xp": 0}},
	})
	if got := balance(tokB); got != 0 {
		t.Fatalf("friend tournament volt %v", got)
	}
	if _, err := x.db.Exec(ctx, `UPDATE tournament.tournaments SET status = 'finished' WHERE id = $1`, str(second.get("id"))); err != nil {
		t.Fatal(err)
	}
	if _, err := x.db.Exec(ctx, `INSERT INTO tournament.entries (id, tournament_id, entity_kind, entity_id, lineup, status, final_place, registered_at)
		VALUES ($1, $2, 'user', $3, ARRAY[$3::uuid], 'winner', 1, now())`, uuid.New(), str(second.get("id")), b); err != nil {
		t.Fatal(err)
	}
	if p := x.do(http.MethodGet, "/v1/profiles/"+b.String(), tokA, nil); p.get("stats", "trophies") != 1.0 {
		t.Fatalf("trophies %s", p.raw)
	}
}
