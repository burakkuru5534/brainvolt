package app

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/lobby"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
)

var quickDuel = map[string]any{"match_type": "quick_duel", "format": "1v1"}

// leaveMatch cancels a match in waiting: every participant joins and leaves.
func leaveMatch(t *testing.T, id string, ws ...*wsClient) {
	t.Helper()
	for _, w := range ws {
		w.answer(w.send("match.join", map[string]any{"match_id": id}))
	}
	if f := ws[0].answer(ws[0].send("match.leave", map[string]any{"match_id": id})); f["t"] != "sys.ack" {
		t.Fatalf("leave %v", f)
	}
	ws[0].find("match.cancelled", func(f map[string]any) bool { return f["t"] == "match.cancelled" && data(f)["match_id"] == id })
}

// Invite codes and direct challenges (schema-social §3.1–§3.2).
func TestCodesAndChallenges(t *testing.T) {
	x := newE2E(t)
	tokA, a, _ := x.fullUser()
	tokB, b, _ := x.fullUser()
	tokC, c, _ := x.fullUser()
	guest, _ := x.signedIn()
	x.seedQuestions(a)

	// Friend codes: full accounts only, one per user; resolving shows the user.
	want(t, "guest code", x.post("/v1/invite-codes", guest, map[string]any{"kind": "friend"}), 403, "ACCOUNT_UPGRADE_REQUIRED")
	want(t, "bad kind", x.post("/v1/invite-codes", tokA, map[string]any{"kind": "team"}), 400, "VALIDATION_FAILED")
	r := x.post("/v1/invite-codes", tokA, map[string]any{"kind": "friend"})
	code := str(r.get("code"))
	if r.code != 200 || len(code) != 6 {
		t.Fatalf("friend code %s", r.raw)
	}
	if again := x.post("/v1/invite-codes", tokA, map[string]any{"kind": "friend"}); again.get("code") != code {
		t.Fatalf("second code %s", again.raw)
	}
	res := x.do(http.MethodGet, "/v1/invite-codes/"+strings.ToLower(code), tokB, nil)
	if res.code != 200 || res.get("kind") != "friend" || res.get("user", "user_id") != a.String() {
		t.Fatalf("resolve %s", res.raw)
	}
	want(t, "unknown code", x.do(http.MethodGet, "/v1/invite-codes/ZZZZZ2", tokB, nil), 404, "INVITE_CODE_INVALID")
	want(t, "malformed code", x.do(http.MethodGet, "/v1/invite-codes/O0O0O0", tokB, nil), 404, "INVITE_CODE_INVALID")
	want(t, "friend by code", x.post("/v1/friend-requests", tokB, map[string]any{"to_user_id": a, "via": "invite_code"}), 201, "")

	wa, _ := x.auth(tokA, "inst-a", nil)
	wb, _ := x.auth(tokB, "inst-b", nil)

	// Sending: rules, one pending per pair, the target is told.
	want(t, "self", x.post("/v1/challenges", tokA, map[string]any{"target_user_id": a, "match_config": quickDuel}), 400, "VALIDATION_FAILED")
	want(t, "team format", x.post("/v1/challenges", tokA, map[string]any{"target_user_id": b, "match_config": map[string]any{"match_type": "quick_duel", "format": "2v2"}}), 422, "MODE_NOT_ALLOWED")
	want(t, "challenges off", x.do(http.MethodPatch, "/v1/me/settings", tokC, map[string]any{"challenges_from": "nobody"}), 200, "")
	want(t, "to nobody", x.post("/v1/challenges", tokA, map[string]any{"target_user_id": c, "match_config": quickDuel}), 403, "CHALLENGES_DISABLED")
	want(t, "friends only", x.do(http.MethodPatch, "/v1/me/settings", tokC, map[string]any{"challenges_from": "friends"}), 200, "")
	want(t, "not a friend", x.post("/v1/challenges", tokA, map[string]any{"target_user_id": c, "match_config": quickDuel}), 403, "CHALLENGES_DISABLED")
	want(t, "C blocks A", x.do(http.MethodPost, "/v1/blocks", tokC, map[string]any{"user_id": a}), 204, "")
	want(t, "blocked", x.post("/v1/challenges", tokA, map[string]any{"target_user_id": c, "match_config": quickDuel}), 403, "BLOCKED")

	r = x.post("/v1/challenges", tokA, map[string]any{"target_user_id": b, "match_config": quickDuel})
	want(t, "challenge", r, 201, "")
	first := str(r.get("id"))
	if r.get("status") != "pending" || r.get("target", "user_id") != b.String() || r.get("match_config", "origin") != "challenge" {
		t.Fatalf("challenge %s", r.raw)
	}
	if got := data(wb.next("challenge.received")); got["challenge"].(map[string]any)["id"] != first {
		t.Fatalf("challenge.received %v", got)
	}
	want(t, "pending pair", x.post("/v1/challenges", tokB, map[string]any{"target_user_id": a, "match_config": quickDuel}), 409, "REQUEST_PENDING")
	in := x.do(http.MethodGet, "/v1/challenges?box=in", tokB, nil)
	if list := in.get("challenges").([]any); len(list) != 1 || list[0].(map[string]any)["id"] != first {
		t.Fatalf("inbox %s", in.raw)
	}
	want(t, "challenger cannot accept", x.do(http.MethodPost, "/v1/challenges/"+first+"/accept", tokA, nil), 404, "NOT_FOUND")
	r = x.do(http.MethodPost, "/v1/challenges/"+first+"/cancel", tokA, nil)
	if r.code != 200 || r.get("status") != "cancelled" {
		t.Fatalf("cancel %s", r.raw)
	}
	if u := data(wb.next("challenge.updated")); u["status"] != "cancelled" {
		t.Fatalf("updated %v", u)
	}
	want(t, "accept cancelled", x.do(http.MethodPost, "/v1/challenges/"+first+"/accept", tokB, nil), 409, "MATCH_STATE_INVALID")

	// An expired challenge cannot be accepted.
	r = x.post("/v1/challenges", tokA, map[string]any{"target_user_id": b, "match_config": quickDuel})
	old := str(r.get("id"))
	if _, err := x.db.Exec(t.Context(), `UPDATE lobby.challenges SET expires_at = now() - interval '1 second' WHERE id = $1`, old); err != nil {
		t.Fatal(err)
	}
	want(t, "expired", x.do(http.MethodPost, "/v1/challenges/"+old+"/accept", tokB, nil), 410, "CHALLENGE_EXPIRED")

	// Accepting creates the match; both players get it.
	r = x.post("/v1/challenges", tokA, map[string]any{"target_user_id": b, "match_config": quickDuel})
	id := str(r.get("id"))
	r = x.do(http.MethodPost, "/v1/challenges/"+id+"/accept", tokB, nil)
	matchID := str(r.get("match_id"))
	if r.code != 200 || r.get("status") != "started" || matchID == "" {
		t.Fatalf("accept %s", r.raw)
	}
	for _, w := range []*wsClient{wa, wb} {
		w.find("match.created", func(f map[string]any) bool { return f["t"] == "match.created" && data(f)["match_id"] == matchID })
		w.find("challenge.updated", func(f map[string]any) bool { return f["t"] == "challenge.updated" && data(f)["match_id"] == matchID })
	}

	// The match is cancelled before it starts → the challenge failed.
	leaveMatch(t, matchID, wa, wb)
	payload, _ := json.Marshal(game.MatchCancelled{MatchID: uuid.MustParse(matchID), Reason: "participant_left_before_start", Origin: game.OriginChallenge})
	if err := lobby.OnMatchCancelled(x.db)(t.Context(), events.Envelope{Name: game.EventMatchCancelled, Payload: payload}); err != nil {
		t.Fatal(err)
	}
	var status string
	if err := x.db.QueryRow(t.Context(), `SELECT status FROM lobby.challenges WHERE id = $1`, id).Scan(&status); err != nil || status != "failed" {
		t.Fatalf("challenge after cancel %q %v", status, err)
	}
}

// Private rooms (schema-social §3.3) and rematch (D-027).
func TestRoomsAndRematch(t *testing.T) {
	x := newE2E(t)
	tokH, h := x.signedIn()
	tokP, p := x.signedIn()
	tokQ, _ := x.signedIn()
	x.seedQuestions(h)
	wh, _ := x.auth(tokH, "inst-h", nil)
	wp, _ := x.auth(tokP, "inst-p", nil)

	r := x.post("/v1/rooms", tokH, map[string]any{"match_config": quickDuel})
	want(t, "create", r, 201, "")
	roomID, code := str(r.get("id")), str(r.get("code"))
	if r.get("status") != "open" || r.get("capacity") != float64(2) || len(r.get("members").([]any)) != 1 || len(code) != 6 {
		t.Fatalf("room %s", r.raw)
	}
	res := x.do(http.MethodGet, "/v1/invite-codes/"+code, tokP, nil)
	if res.get("kind") != "room" || res.get("room", "member_count") != float64(1) || res.get("room", "host", "user_id") != h.String() {
		t.Fatalf("resolve %s", res.raw)
	}
	r = x.do(http.MethodPost, "/v1/rooms/join", tokP, map[string]any{"code": strings.ToLower(code)})
	if r.code != 200 || len(r.get("members").([]any)) != 2 {
		t.Fatalf("join %s", r.raw)
	}
	st := data(wh.find("room.state with 2", func(f map[string]any) bool {
		return f["t"] == "room.state" && len(data(f)["room"].(map[string]any)["members"].([]any)) == 2
	}))
	version := st["room"].(map[string]any)["version"].(float64)
	want(t, "full", x.do(http.MethodPost, "/v1/rooms/join", tokQ, map[string]any{"code": code}), 409, "ROOM_FULL")
	want(t, "outsider get", x.do(http.MethodGet, "/v1/rooms/"+roomID, tokQ, nil), 404, "NOT_FOUND")
	want(t, "member get", x.do(http.MethodGet, "/v1/rooms/"+roomID, tokP, nil), 200, "")

	// Config: host only, If-Match version.
	sd := map[string]any{"match_config": map[string]any{"match_type": "marathon", "format": "1v1"}}
	want(t, "patch by member", x.do(http.MethodPatch, "/v1/rooms/"+roomID, tokP, sd), 403, "FORBIDDEN")
	want(t, "stale version", x.do(http.MethodPatch, "/v1/rooms/"+roomID, tokH, sd, "If-Match", "1"), 409, "VERSION_CONFLICT")
	want(t, "patch", x.do(http.MethodPatch, "/v1/rooms/"+roomID, tokH, map[string]any{"match_config": quickDuel}, "If-Match", jsonNum(version)), 200, "")

	// Seats: members move themselves to free seats only.
	if f := wp.answer(wp.send("room.update_seat", map[string]any{"room_id": roomID, "side_no": 1, "seat_no": 1})); data(f)["code"] != "VALIDATION_FAILED" {
		t.Fatalf("taken seat %v", f)
	}

	// Start: everyone but the host ready.
	if f := wh.answer(wh.send("room.start", map[string]any{"room_id": roomID})); data(f)["code"] != "MATCH_STATE_INVALID" {
		t.Fatalf("start unready %v", f)
	}
	if f := wp.answer(wp.send("room.ready", map[string]any{"room_id": roomID})); f["t"] != "sys.ack" {
		t.Fatalf("ready %v", f)
	}
	if f := wp.answer(wp.send("room.start", map[string]any{"room_id": roomID})); data(f)["code"] != "FORBIDDEN" {
		t.Fatalf("member start %v", f)
	}
	if f := wh.answer(wh.send("room.start", map[string]any{"room_id": roomID})); f["t"] != "sys.ack" {
		t.Fatalf("start %v", f)
	}
	matchID := str(data(wp.next("room.match_starting"))["match_id"])
	wh.next("room.match_starting")
	if g := x.do(http.MethodGet, "/v1/rooms/"+roomID, tokH, nil); g.get("status") != "in_match" || g.get("match_id") != matchID {
		t.Fatalf("in match %s", g.raw)
	}

	// After the match the room is open again for the next one.
	leaveMatch(t, matchID, wh, wp)
	if f := wh.answer(wh.send("room.join", map[string]any{"room_id": roomID})); f["t"] != "sys.ack" {
		t.Fatalf("room.join %v", f)
	}
	open := wh.find("open room.state", func(f map[string]any) bool {
		return f["t"] == "room.state" && data(f)["room"].(map[string]any)["status"] == "open"
	})
	if rm := data(open)["room"].(map[string]any); rm["match_id"] != nil {
		t.Fatalf("reset %v", rm)
	}

	// Rematch works on completed non-room matches within 30 s.
	re := wh.send("rematch.respond", map[string]any{"match_id": matchID, "accept": true})
	if f := wh.answer(re); data(f)["code"] != "MATCH_STATE_INVALID" {
		t.Fatalf("rematch of a cancelled room match %v", f)
	}
	c, err := x.mods.game.Start(t.Context(), game.CreateRequest{
		Config: game.Config{MatchType: "quick_duel", Format: "1v1", Origin: "challenge"},
		Sides:  [2][]game.Member{{{UserID: h, Locale: "tr"}}, {{UserID: p, Locale: "en"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	parent := c.Match.ID.String()
	leaveMatch(t, parent, wh, wp)
	// Pretend it was played to the end.
	if _, err := x.db.Exec(t.Context(), `UPDATE game.matches SET status = 'completed', cancel_reason = NULL, end_reason = 'all_rounds_played', ended_at = now() WHERE id = $1`, c.Match.ID); err != nil {
		t.Fatal(err)
	}
	tokQws, _ := x.auth(tokQ, "inst-q", nil)
	if f := tokQws.answer(tokQws.send("rematch.respond", map[string]any{"match_id": parent, "accept": true})); data(f)["code"] != "NOT_A_PARTICIPANT" {
		t.Fatalf("outsider rematch %v", f)
	}
	if f := wh.answer(wh.send("rematch.respond", map[string]any{"match_id": parent, "accept": true})); f["t"] != "sys.ack" {
		t.Fatalf("rematch accept %v", f)
	}
	offer := data(wp.next("rematch.offer"))
	if ids := offer["accepted_user_ids"].([]any); len(ids) != 1 || ids[0] != h.String() {
		t.Fatalf("offer %v", offer)
	}
	wp.answer(wp.send("rematch.respond", map[string]any{"match_id": parent, "accept": true}))
	result := data(wh.find("rematch.result", func(f map[string]any) bool { return f["t"] == "rematch.result" }))
	next := str(result["new_match_id"])
	if result["accepted"] != true || next == "" {
		t.Fatalf("rematch result %v", result)
	}
	var origin, parentID string
	if err := x.db.QueryRow(t.Context(), `SELECT origin, parent_match_id::text FROM game.matches WHERE id = $1`, next).Scan(&origin, &parentID); err != nil || origin != "rematch" || parentID != parent {
		t.Fatalf("rematch row %q %q %v", origin, parentID, err)
	}
	leaveMatch(t, next, wh, wp)

	// Too late: the window closed.
	if _, err := x.db.Exec(t.Context(), `UPDATE game.matches SET ended_at = now() - interval '1 minute' WHERE id = $1`, c.Match.ID); err != nil {
		t.Fatal(err)
	}
	if f := wh.answer(wh.send("rematch.respond", map[string]any{"match_id": parent, "accept": true})); data(f)["code"] != "CHALLENGE_EXPIRED" {
		t.Fatalf("late rematch %v", f)
	}

	// The host leaving closes the room; the code stops working.
	want(t, "leave", x.do(http.MethodPost, "/v1/rooms/"+roomID+"/leave", tokH, nil), 204, "")
	wp.find("room.closed", func(f map[string]any) bool { return f["t"] == "room.closed" && data(f)["reason"] == "host_closed" })
	want(t, "closed code", x.do(http.MethodPost, "/v1/rooms/join", tokQ, map[string]any{"code": code}), 410, "INVITE_CODE_EXPIRED")
	want(t, "closed room", x.do(http.MethodGet, "/v1/rooms/"+roomID, tokP, nil), 410, "ROOM_CLOSED")
}

func jsonNum(v float64) string { b, _ := json.Marshal(v); return string(b) }

// Idle rooms close after 15 minutes (D-027).
func TestIdleRoomCloses(t *testing.T) {
	x := newE2E(t)
	tok, _ := x.signedIn()
	r := x.post("/v1/rooms", tok, map[string]any{"match_config": quickDuel})
	id := str(r.get("id"))
	if _, err := x.db.Exec(t.Context(), `UPDATE lobby.rooms SET last_activity_at = $2 WHERE id = $1`, id, time.Now().Add(-20*time.Minute)); err != nil {
		t.Fatal(err)
	}
	x.mods.lobby.Sweep(t.Context())
	var status, reason string
	if err := x.db.QueryRow(t.Context(), `SELECT status, close_reason FROM lobby.rooms WHERE id = $1`, id).Scan(&status, &reason); err != nil || status != "closed" || reason != "idle" {
		t.Fatalf("idle room %q %q %v", status, reason, err)
	}
}

// Team rooms (Phase 6): a 2v2 room fills, readies and starts; a couple
// room needs real couples on both sides.
func TestTeamRooms(t *testing.T) {
	x := newE2E(t)
	var toks [4]string
	var users [4]uuid.UUID
	var ws [4]*wsClient
	for i := range 4 {
		toks[i], users[i] = x.signedIn()
	}
	x.seedQuestions(users[0])
	for i := range 4 {
		ws[i], _ = x.auth(toks[i], "inst-"+string(rune('a'+i)), nil)
	}
	cfg := map[string]any{"match_type": "quick_duel", "format": "2v2", "team_answer_mode": "fast_team_round"}
	want(t, "team mode needed", x.post("/v1/rooms", toks[0], map[string]any{"match_config": map[string]any{"match_type": "quick_duel", "format": "2v2"}}), 422, "MODE_NOT_ALLOWED")
	r := x.post("/v1/rooms", toks[0], map[string]any{"match_config": cfg})
	want(t, "create", r, 201, "")
	roomID, code := str(r.get("id")), str(r.get("code"))
	if r.get("capacity") != float64(4) {
		t.Fatalf("capacity %s", r.raw)
	}
	for i := 1; i < 4; i++ {
		want(t, "join", x.do(http.MethodPost, "/v1/rooms/join", toks[i], map[string]any{"code": code}), 200, "")
		ws[i].answer(ws[i].send("room.ready", map[string]any{"room_id": roomID}))
	}
	if f := ws[0].answer(ws[0].send("room.start", map[string]any{"room_id": roomID})); f["t"] != "sys.ack" {
		t.Fatalf("start %v", f)
	}
	matchID := str(data(ws[3].next("room.match_starting"))["match_id"])
	var format, mode string
	if err := x.db.QueryRow(t.Context(), `SELECT format, team_answer_mode FROM game.matches WHERE id = $1`, matchID).Scan(&format, &mode); err != nil || format != "2v2" || mode != "fast_team_round" {
		t.Fatalf("match %q %q %v", format, mode, err)
	}
	leaveMatch(t, matchID, ws[0], ws[1], ws[2], ws[3])

	// Couple format with players who are not couples → R16 at start.
	want(t, "to couple", x.do(http.MethodPatch, "/v1/rooms/"+roomID, toks[0], map[string]any{
		"match_config": map[string]any{"match_type": "quick_duel", "format": "couple", "team_answer_mode": "fast_team_round"},
	}), 200, "")
	ws[0].answer(ws[0].send("room.join", map[string]any{"room_id": roomID}))
	for i := 1; i < 4; i++ {
		ws[i].answer(ws[i].send("room.ready", map[string]any{"room_id": roomID}))
	}
	if f := ws[0].answer(ws[0].send("room.start", map[string]any{"room_id": roomID})); data(f)["code"] != "MODE_NOT_ALLOWED" {
		t.Fatalf("couple start %v", f)
	}
}

// Couple challenges (Couple Contest): a partner challenges another
// couple; both target partners see it, either accepts, the match is a
// couple match.
func TestCoupleChallenge(t *testing.T) {
	x := newE2E(t)
	var toks [4]string
	var users [4]uuid.UUID
	var ws [4]*wsClient
	for i := range 4 {
		toks[i], users[i], _ = x.fullUser()
		ws[i], _ = x.auth(toks[i], "inst-"+string(rune('a'+i)), nil)
	}
	x.seedQuestions(users[0])
	couple := func(a, b int) string {
		x.befriend(toks[a], users[b], toks[b])
		r := x.post("/v1/couples", toks[a], map[string]any{"partner_user_id": users[b]})
		want(t, "accept couple", x.do(http.MethodPost, "/v1/couples/"+str(r.get("id"))+"/accept", toks[b], nil), 200, "")
		return str(r.get("id"))
	}
	couple(0, 1)
	theirs := couple(2, 3)
	cfg := map[string]any{"match_type": "quick_duel", "format": "couple", "team_answer_mode": "fast_team_round"}
	want(t, "user target couple format", x.post("/v1/challenges", toks[0], map[string]any{"target_user_id": users[2], "match_config": cfg}), 422, "MODE_NOT_ALLOWED")
	r := x.post("/v1/challenges", toks[0], map[string]any{"target_couple_id": theirs, "match_config": cfg})
	want(t, "couple challenge", r, 201, "")
	id := str(r.get("id"))
	if len(r.get("target_side").([]any)) != 2 || len(r.get("challenger_side").([]any)) != 2 {
		t.Fatalf("challenge %s", r.raw)
	}
	for _, i := range []int{2, 3} {
		ws[i].next("challenge.received")
	}
	in := x.do(http.MethodGet, "/v1/challenges?box=in", toks[3], nil)
	if list := in.get("challenges").([]any); len(list) != 1 {
		t.Fatalf("partner inbox %s", in.raw)
	}
	want(t, "challenger's partner cannot accept", x.do(http.MethodPost, "/v1/challenges/"+id+"/accept", toks[1], nil), 404, "NOT_FOUND")
	r = x.do(http.MethodPost, "/v1/challenges/"+id+"/accept", toks[3], nil)
	matchID := str(r.get("match_id"))
	if r.code != 200 || matchID == "" {
		t.Fatalf("accept %s", r.raw)
	}
	var n int
	if err := x.db.QueryRow(t.Context(), `SELECT count(*) FROM game.match_sides WHERE match_id = $1 AND couple_id IS NOT NULL`, matchID).Scan(&n); err != nil || n != 2 {
		t.Fatalf("couple sides %d %v", n, err)
	}
	for i := range 4 {
		ws[i].find("challenge.updated", func(f map[string]any) bool {
			return f["t"] == "challenge.updated" && data(f)["match_id"] == matchID
		})
	}
	leaveMatch(t, matchID, ws[0], ws[1], ws[2], ws[3])
}

// Revenge (D-045): the losing side of a completed 1v1 challenges the
// winner with a home category; the result shows the series.
func TestRevengeChallenge(t *testing.T) {
	x := newE2E(t)
	tokA, a, _ := x.fullUser()
	tokB, _, _ := x.fullUser()
	x.seedQuestions(a)
	wa, _ := x.auth(tokA, "inst-a", nil)
	wb, _ := x.auth(tokB, "inst-b", nil)

	// A completed match that B won.
	r := x.post("/v1/challenges", tokA, map[string]any{"target_user_id": x.userOf(tokB), "match_config": quickDuel})
	r = x.do(http.MethodPost, "/v1/challenges/"+str(r.get("id"))+"/accept", tokB, nil)
	parent := str(r.get("match_id"))
	leaveMatch(t, parent, wa, wb)
	if _, err := x.db.Exec(t.Context(), `UPDATE game.matches SET status = 'completed', ended_at = now(), winner_side_no = 2, is_draw = false,
		cancel_reason = NULL, end_reason = 'all_rounds_played' WHERE id = $1`, parent); err != nil {
		t.Fatal(err)
	}
	var cat string
	if err := x.db.QueryRow(t.Context(), `SELECT id FROM content.categories ORDER BY sort_order LIMIT 1`).Scan(&cat); err != nil {
		t.Fatal(err)
	}

	r = x.do(http.MethodGet, "/v1/matches/"+parent, tokA, nil)
	if r.get("revenge", "available") != true || r.get("revenge", "losses") != 1.0 {
		t.Fatalf("loser's revenge view %s", r.raw)
	}
	if r := x.do(http.MethodGet, "/v1/matches/"+parent, tokB, nil); r.get("revenge", "available") != false {
		t.Fatalf("winner's revenge view %s", r.raw)
	}
	path := "/v1/matches/" + parent + "/revenge"
	want(t, "winner asks", x.post(path, tokB, map[string]any{"home_category_id": cat}), 403, "REVENGE_NOT_ALLOWED")
	want(t, "bad category", x.post(path, tokA, map[string]any{"home_category_id": uuid.NewString()}), 400, "VALIDATION_FAILED")
	r = x.post(path, tokA, map[string]any{"home_category_id": cat})
	want(t, "revenge", r, 201, "")
	id := str(r.get("id"))
	if r.get("kind") != "revenge" || r.get("parent_match_id") != parent || r.get("target", "user_id") != x.userOf(tokB).String() {
		t.Fatalf("revenge challenge %s", r.raw)
	}
	wb.find("challenge.received", func(f map[string]any) bool {
		return f["t"] == "challenge.received" && data(f)["challenge"].(map[string]any)["id"] == id
	})
	want(t, "second revenge", x.post(path, tokA, map[string]any{"home_category_id": cat}), 409, "REQUEST_PENDING")

	r = x.do(http.MethodPost, "/v1/challenges/"+id+"/accept", tokB, nil)
	next := str(r.get("match_id"))
	if r.code != 200 || next == "" {
		t.Fatalf("accept revenge %s", r.raw)
	}
	r = x.do(http.MethodGet, "/v1/matches/"+next, tokA, nil)
	if r.get("origin") != "revenge" || r.get("category_mode") != "revenge_match" || r.get("match_type") != "quick_duel" {
		t.Fatalf("revenge match %s", r.raw)
	}
	leaveMatch(t, next, wa, wb)
}
