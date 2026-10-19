package app

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// befriend makes a and b friends.
func (x *e2e) befriend(tokA string, b uuid.UUID, tokB string) {
	x.t.Helper()
	r := x.post("/v1/friend-requests", tokA, map[string]any{"to_user_id": b})
	if r.code != 201 {
		x.t.Fatalf("friend request %s", r.raw)
	}
	want(x.t, "accept friend", x.do(http.MethodPost, "/v1/friend-requests/"+str(r.get("request", "id"))+"/accept", tokB, nil), 204, "")
}

// Teams (schema-social §4): create, invites among friends, captaincy,
// leaving, limits and disbanding.
func TestTeams(t *testing.T) {
	x := newE2E(t)
	tokA, a, _ := x.fullUser()
	tokB, b, _ := x.fullUser()
	tokC, c, _ := x.fullUser()
	guest, _ := x.signedIn()
	volt := map[string]any{"name": "Volt  Squad", "tag": "bvx", "emblem_id": "bolt"}

	want(t, "guest team", x.post("/v1/teams", guest, volt), 403, "ACCOUNT_UPGRADE_REQUIRED")
	want(t, "bad emblem", x.post("/v1/teams", tokA, map[string]any{"name": "Some Team", "tag": "AB", "emblem_id": "x"}), 400, "VALIDATION_FAILED")
	r := x.post("/v1/teams", tokA, volt)
	want(t, "create", r, 201, "")
	id := str(r.get("id"))
	if r.get("name") != "Volt Squad" || r.get("tag") != "BVX" || r.get("captain_user_id") != a.String() || len(r.get("members").([]any)) != 1 {
		t.Fatalf("team %s", r.raw)
	}
	want(t, "taken", x.post("/v1/teams", tokB, map[string]any{"name": "volt squad", "tag": "BVX", "emblem_id": "owl"}), 400, "VALIDATION_FAILED")

	// Invites go to friends; any member may invite.
	want(t, "not a friend", x.post("/v1/teams/"+id+"/invites", tokA, map[string]any{"user_id": b}), 403, "FORBIDDEN")
	x.befriend(tokA, b, tokB)
	x.befriend(tokB, c, tokC)
	want(t, "invite B", x.post("/v1/teams/"+id+"/invites", tokA, map[string]any{"user_id": b}), 201, "")
	want(t, "again", x.post("/v1/teams/"+id+"/invites", tokA, map[string]any{"user_id": b}), 409, "REQUEST_PENDING")
	inv := x.do(http.MethodGet, "/v1/me/team-invites", tokB, nil)
	list := inv.get("invites").([]any)
	if len(list) != 1 || list[0].(map[string]any)["team_tag"] != "BVX" {
		t.Fatalf("invites %s", inv.raw)
	}
	invID := str(list[0].(map[string]any)["id"])
	want(t, "accept", x.do(http.MethodPost, "/v1/team-invites/"+invID+"/accept", tokB, nil), 204, "")
	want(t, "accept twice", x.do(http.MethodPost, "/v1/team-invites/"+invID+"/accept", tokB, nil), 409, "MATCH_STATE_INVALID")
	r = x.post("/v1/teams/"+id+"/invites", tokB, map[string]any{"user_id": c})
	want(t, "member invites", r, 201, "")
	want(t, "decline", x.do(http.MethodPost, "/v1/team-invites/"+str(r.get("id"))+"/decline", tokC, nil), 204, "")

	// Captaincy.
	want(t, "member kicks", x.do(http.MethodPost, "/v1/teams/"+id+"/kick/"+a.String(), tokB, nil), 403, "NOT_CAPTAIN")
	want(t, "member edits", x.do(http.MethodPatch, "/v1/teams/"+id, tokB, volt), 403, "NOT_CAPTAIN")
	want(t, "hand over", x.do(http.MethodPost, "/v1/teams/"+id+"/captain/"+b.String(), tokA, nil), 204, "")
	if g := x.do(http.MethodGet, "/v1/teams/"+id, guest, nil); g.get("captain_user_id") != b.String() {
		t.Fatalf("captain %s", g.raw)
	}
	// The captain leaves → the longest-standing member (A) is captain.
	want(t, "captain leaves", x.do(http.MethodPost, "/v1/teams/"+id+"/leave", tokB, nil), 204, "")
	if g := x.do(http.MethodGet, "/v1/teams/"+id, tokA, nil); g.get("captain_user_id") != a.String() || len(g.get("members").([]any)) != 1 {
		t.Fatalf("after leave %s", g.raw)
	}

	// Three active teams per user.
	for i, n := range []string{"Second One", "Third One"} {
		want(t, n, x.post("/v1/teams", tokA, map[string]any{"name": n, "tag": []string{"TWO", "THR"}[i], "emblem_id": "star"}), 201, "")
	}
	want(t, "fourth", x.post("/v1/teams", tokA, map[string]any{"name": "Fourth One", "tag": "FOU", "emblem_id": "star"}), 409, "TEAM_LIMIT_REACHED")

	// Disbanding frees the name and the slot.
	want(t, "disband", x.do(http.MethodPost, "/v1/teams/"+id+"/disband", tokA, nil), 204, "")
	if mine := x.do(http.MethodGet, "/v1/me/teams", tokA, nil); len(mine.get("teams").([]any)) != 2 {
		t.Fatalf("my teams %s", mine.raw)
	}
	want(t, "name free again", x.post("/v1/teams", tokC, volt), 201, "")
}

// Couples (D-047): request among friends, one live couple, accept,
// dissolve, cooldown for a new partner, reactivation of the same pair.
func TestCouples(t *testing.T) {
	x := newE2E(t)
	tokA, a, _ := x.fullUser()
	tokB, b, _ := x.fullUser()
	tokC, c, _ := x.fullUser()
	x.befriend(tokA, b, tokB)
	x.befriend(tokA, c, tokC)
	x.befriend(tokB, c, tokC)

	r := x.post("/v1/couples", tokA, map[string]any{"partner_user_id": c, "name": "Bilgi Çifti"})
	want(t, "request", r, 201, "")
	id := str(r.get("id"))
	if r.get("status") != "pending" || r.get("expires_at") == nil {
		t.Fatalf("couple %s", r.raw)
	}
	want(t, "C is taken", x.post("/v1/couples", tokB, map[string]any{"partner_user_id": c}), 409, "COUPLE_EXISTS")
	want(t, "requester accepts", x.do(http.MethodPost, "/v1/couples/"+id+"/accept", tokA, nil), 409, "MATCH_STATE_INVALID")
	want(t, "outsider", x.do(http.MethodPost, "/v1/couples/"+id+"/accept", tokB, nil), 404, "NOT_FOUND")
	r = x.do(http.MethodPost, "/v1/couples/"+id+"/accept", tokC, nil)
	if r.code != 200 || r.get("status") != "active" || r.get("activated_at") == nil {
		t.Fatalf("accept %s", r.raw)
	}
	if mine := x.do(http.MethodGet, "/v1/me/couple", tokA, nil); mine.get("couple", "id") != id {
		t.Fatalf("my couple %s", mine.raw)
	}
	want(t, "dissolve", x.do(http.MethodPost, "/v1/couples/"+id+"/dissolve", tokA, nil), 200, "")
	if mine := x.do(http.MethodGet, "/v1/me/couple", tokA, nil); mine.get("couple") != nil {
		t.Fatalf("after dissolve %s", mine.raw)
	}

	// A new partner within 7 days → cooldown (for A and for C).
	r = x.post("/v1/couples", tokA, map[string]any{"partner_user_id": b})
	want(t, "cooldown", r, 409, "COUPLE_COOLDOWN")
	if r.get("error", "details", "available_at") == nil {
		t.Fatalf("cooldown details %s", r.raw)
	}
	want(t, "C cooldown too", x.post("/v1/couples", tokB, map[string]any{"partner_user_id": c}), 409, "COUPLE_COOLDOWN")
	// The same pair again reactivates the old profile.
	r = x.post("/v1/couples", tokC, map[string]any{"partner_user_id": a})
	if r.code != 201 || r.get("id") != id || r.get("requested_by") != c.String() || r.get("name") != "Bilgi Çifti" {
		t.Fatalf("reactivation %s", r.raw)
	}
	want(t, "withdraw", x.do(http.MethodPost, "/v1/couples/"+id+"/decline", tokC, nil), 200, "")
	_ = b
}
