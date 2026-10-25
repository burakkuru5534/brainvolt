package app

import (
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
)

// fullUser signs up a guest and upgrades it to a full account (verified
// email); returns its token, user id and username.
func (x *e2e) fullUser() (token string, user uuid.UUID, username string) {
	x.t.Helper()
	token, uid := x.emu.signUpAnonymous()
	r := x.session(token, nil)
	if r.code != 200 {
		x.t.Fatalf("session: %s", r.raw)
	}
	email := "u" + strings.ReplaceAll(ids.NewString(), "-", "") + "@example.com"
	x.emu.linkPassword(token, email)
	x.emu.verifyEmail(uid)
	token = x.emu.signIn(email)
	r = x.session(token, nil)
	if r.get("tier") != "full" {
		x.t.Fatalf("not full: %s", r.raw)
	}
	id, _ := r.get("user", "id").(string)
	name, _ := r.get("profile", "username").(string)
	return token, uuid.MustParse(id), name
}

func TestFriendsAndBlocks(t *testing.T) {
	x := newE2E(t)
	tokA, a, _ := x.fullUser()
	tokB, b, nameB := x.fullUser()
	tokC, c, _ := x.fullUser()
	guest, _ := x.signedIn()

	// Guests cannot have friends (D-016).
	want(t, "guest friends", x.do(http.MethodGet, "/v1/friends", guest, nil), 403, "ACCOUNT_UPGRADE_REQUIRED")

	// Lookup by username, then a request; sending twice is REQUEST_PENDING.
	p := x.do(http.MethodGet, "/v1/profiles/by-username/"+strings.ToUpper(nameB), tokA, nil)
	if p.code != 200 || p.get("user_id") != b.String() {
		t.Fatalf("by-username %s", p.raw)
	}
	want(t, "unknown username", x.do(http.MethodGet, "/v1/profiles/by-username/nobody_here_x", tokA, nil), 404, "NOT_FOUND")
	r := x.post("/v1/friend-requests", tokA, map[string]any{"username": nameB})
	want(t, "request", r, 201, "")
	reqID, _ := r.get("request", "id").(string)
	if r.get("request", "user", "user_id") != b.String() {
		t.Fatalf("request %s", r.raw)
	}
	want(t, "again", x.post("/v1/friend-requests", tokA, map[string]any{"to_user_id": b}), 409, "REQUEST_PENDING")
	want(t, "self", x.post("/v1/friend-requests", tokA, map[string]any{"to_user_id": a}), 400, "VALIDATION_FAILED")
	want(t, "to a guest", x.post("/v1/friend-requests", tokA, map[string]any{"to_user_id": x.userOf(guest)}), 403, "FORBIDDEN")

	in := x.do(http.MethodGet, "/v1/friend-requests", tokB, nil)
	if reqs := in.get("requests").([]any); len(reqs) != 1 || reqs[0].(map[string]any)["id"] != reqID {
		t.Fatalf("inbox %s", in.raw)
	}
	want(t, "A cannot accept own request", x.do(http.MethodPost, "/v1/friend-requests/"+reqID+"/accept", tokA, nil), 404, "NOT_FOUND")
	want(t, "accept", x.do(http.MethodPost, "/v1/friend-requests/"+reqID+"/accept", tokB, nil), 204, "")
	want(t, "already friends", x.post("/v1/friend-requests", tokB, map[string]any{"to_user_id": a}), 409, "ALREADY_FRIENDS")
	f := x.do(http.MethodGet, "/v1/friends", tokA, nil)
	friends := f.get("friends").([]any)
	if len(friends) != 1 || friends[0].(map[string]any)["user_id"] != b.String() || friends[0].(map[string]any)["online"] != false {
		t.Fatalf("friends %s", f.raw)
	}
	if ok, err := x.mods.social.AreFriends(t.Context(), b, a); err != nil || !ok {
		t.Fatalf("AreFriends %v %v", ok, err)
	}

	// Crossing requests: C asks A, A asks C → accepted at once.
	want(t, "C→A", x.post("/v1/friend-requests", tokC, map[string]any{"to_user_id": a}), 201, "")
	r = x.post("/v1/friend-requests", tokA, map[string]any{"to_user_id": c})
	if r.code != 200 || r.get("accepted") != true {
		t.Fatalf("crossing %s", r.raw)
	}

	// Blocking ends the friendship and stops new requests both ways.
	want(t, "block", x.do(http.MethodPost, "/v1/blocks", tokB, map[string]any{"user_id": a}), 204, "")
	if ok, _ := x.mods.social.AreFriends(t.Context(), a, b); ok {
		t.Fatal("still friends after block")
	}
	want(t, "blocked request", x.post("/v1/friend-requests", tokA, map[string]any{"to_user_id": b}), 403, "BLOCKED")
	bl := x.do(http.MethodGet, "/v1/blocks", tokB, nil)
	if list := bl.get("blocks").([]any); len(list) != 1 || list[0].(map[string]any)["user_id"] != a.String() {
		t.Fatalf("blocks %s", bl.raw)
	}
	want(t, "unblock", x.do(http.MethodDelete, "/v1/blocks/"+a.String(), tokB, nil), 204, "")
	want(t, "request after unblock", x.post("/v1/friend-requests", tokA, map[string]any{"to_user_id": b}), 201, "")
	out := x.do(http.MethodGet, "/v1/friend-requests?box=out", tokA, nil)
	id := out.get("requests").([]any)[0].(map[string]any)["id"].(string)
	want(t, "cancel", x.do(http.MethodPost, "/v1/friend-requests/"+id+"/cancel", tokA, nil), 204, "")
	want(t, "unfriend", x.do(http.MethodDelete, "/v1/friends/"+c.String(), tokA, nil), 204, "")
	if f := x.do(http.MethodGet, "/v1/friends", tokA, nil); len(f.get("friends").([]any)) != 0 {
		t.Fatalf("friends after unfriend %s", f.raw)
	}
	// Guests may block.
	want(t, "guest block", x.do(http.MethodPost, "/v1/blocks", guest, map[string]any{"user_id": a}), 204, "")
}

func (x *e2e) userOf(token string) uuid.UUID {
	x.t.Helper()
	r := x.do(http.MethodGet, "/v1/me", token, nil)
	id, _ := r.get("user", "id").(string)
	return uuid.MustParse(id)
}

// TestPlayerProfile covers D-101: the relation of the viewer, the stats
// and a private profile that only friends can open.
func TestPlayerProfile(t *testing.T) {
	x := newE2E(t)
	tokA, a, _ := x.fullUser()
	tokB, b, _ := x.fullUser()
	tokC, _, _ := x.fullUser()
	profileOf := func(tok string, id uuid.UUID) response {
		t.Helper()
		r := x.do(http.MethodGet, "/v1/profiles/"+id.String(), tok, nil)
		if r.code != 200 {
			t.Fatalf("profile %s", r.raw)
		}
		return r
	}
	relation := func(tok string, id uuid.UUID, rel string, stats bool) {
		t.Helper()
		r := profileOf(tok, id)
		if r.get("relation") != rel || (r.get("stats") != nil) != stats {
			t.Fatalf("want %s stats=%v: %s", rel, stats, r.raw)
		}
	}

	r := profileOf(tokA, a)
	if r.get("relation") != "self" || r.get("private") != false || r.get("stats", "level") != float64(1) {
		t.Fatalf("self %s", r.raw)
	}
	relation(tokA, b, "none", true)

	want(t, "request", x.post("/v1/friend-requests", tokA, map[string]any{"to_user_id": b}), 201, "")
	relation(tokA, b, "request_sent", true)
	relation(tokB, a, "request_received", true)

	// B goes private: strangers lose the stats, B still sees its own.
	st := x.do(http.MethodPatch, "/v1/me/settings", tokB, map[string]any{"profile_private": true})
	if st.code != 200 || st.get("profile_private") != true {
		t.Fatalf("settings %s", st.raw)
	}
	if r := profileOf(tokC, b); r.get("private") != true || r.get("stats") != nil {
		t.Fatalf("stranger sees private stats %s", r.raw)
	}
	relation(tokB, b, "self", true)
	relation(tokA, b, "request_sent", false)

	// Friends see a private profile.
	in := x.do(http.MethodGet, "/v1/friend-requests", tokB, nil)
	id := in.get("requests").([]any)[0].(map[string]any)["id"].(string)
	want(t, "accept", x.do(http.MethodPost, "/v1/friend-requests/"+id+"/accept", tokB, nil), 204, "")
	relation(tokA, b, "friend", true)

	// A block hides the stats both ways.
	want(t, "block", x.do(http.MethodPost, "/v1/blocks", tokA, map[string]any{"user_id": b}), 204, "")
	relation(tokB, a, "blocked", false)
}
