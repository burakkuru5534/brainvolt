package app

import (
	"net/http"
	"testing"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
)

// Team chat (schema-platform §2, D-031, D-088): members only, delivery
// over the WebSocket, masking and refused slurs, quick phrases, the
// 5-in-10-seconds limit, chat_mute, history, removal and read cursors.
func TestTeamChat(t *testing.T) {
	x := newE2E(t)
	tokA, a, _ := x.fullUser()
	tokB, b, _ := x.fullUser()
	tokC, _, _ := x.fullUser()
	tokM, m, _ := x.fullUser()
	if _, err := x.mods.identity.SetRole(t.Context(), m.String(), identity.RoleModerator, true); err != nil {
		t.Fatal(err)
	}
	x.befriend(tokA, b, tokB)
	team := str(x.post("/v1/teams", tokA, map[string]any{"name": "Chat Squad", "tag": "CHT", "emblem_id": "owl"}).get("id"))
	inv := x.post("/v1/teams/"+team+"/invites", tokA, map[string]any{"user_id": b})
	want(t, "join", x.do(http.MethodPost, "/v1/team-invites/"+str(inv.get("id"))+"/accept", tokB, nil), 204, "")

	want(t, "outsider", x.post("/v1/chat/channels", tokC, map[string]any{"team_id": team}), 404, "NOT_FOUND")
	want(t, "both", x.post("/v1/chat/channels", tokA, map[string]any{}), 400, "VALIDATION_FAILED")
	r := x.post("/v1/chat/channels", tokA, map[string]any{"team_id": team})
	if r.code != 200 || len(r.get("members").([]any)) != 2 || r.get("kind") != "team" {
		t.Fatalf("open %s", r.raw)
	}
	ch := str(r.get("id"))
	if again := x.post("/v1/chat/channels", tokB, map[string]any{"team_id": team}); str(again.get("id")) != ch {
		t.Fatalf("same channel %s", again.raw)
	}

	wa, _ := x.auth(tokA, "inst-a", nil)
	wb, _ := x.auth(tokB, "inst-b", nil)
	wc, _ := x.auth(tokC, "inst-c", nil)
	send := func(w *wsClient, d map[string]any) map[string]any {
		d["channel_id"] = ch
		return w.answer(w.send("chat.send", d))
	}
	if f := send(wa, map[string]any{"body": "  bu soru amk zordu  "}); f["t"] != "sys.ack" {
		t.Fatalf("send %v", f)
	}
	got := data(wb.next("chat.message"))["message"].(map[string]any)
	if got["body"] != "bu soru *** zordu" || got["sender"].(map[string]any)["user_id"] != a.String() {
		t.Fatalf("delivered %v", got)
	}
	wa.next("chat.message") // the sender's own sessions get it too
	if f := send(wa, map[string]any{"body": "sen bir orospu"}); data(f)["code"] != "TEXT_REJECTED" {
		t.Fatalf("slur %v", f)
	}
	if f := send(wa, map[string]any{"quick_id": "gg"}); f["t"] != "sys.ack" {
		t.Fatalf("quick %v", f)
	}
	if f := send(wa, map[string]any{"quick_id": "spam"}); data(f)["code"] != "VALIDATION_FAILED" {
		t.Fatalf("bad quick %v", f)
	}
	if f := send(wc, map[string]any{"body": "hello"}); data(f)["code"] != "NOT_FOUND" {
		t.Fatalf("outsider send %v", f)
	}
	if f := wb.answer(wb.send("chat.typing", map[string]any{"channel_id": ch})); f["t"] != "sys.ack" {
		t.Fatalf("typing %v", f)
	}
	if d := data(wa.next("chat.typing")); d["user_id"] != b.String() {
		t.Fatalf("typing %v", d)
	}

	// Five messages in ten seconds.
	for i := range 3 {
		if f := send(wa, map[string]any{"body": "mesaj"}); f["t"] != "sys.ack" {
			t.Fatalf("message %d %v", i, f)
		}
	}
	if f := send(wa, map[string]any{"body": "one too many"}); data(f)["code"] != "RATE_LIMITED" {
		t.Fatalf("burst %v", f)
	}

	// History newest first; unread and the read cursor.
	r = x.do(http.MethodGet, "/v1/chat/channels/"+ch+"/messages", tokB, nil)
	msgs := r.get("messages").([]any)
	if r.code != 200 || len(msgs) != 5 || msgs[4].(map[string]any)["body"] != "bu soru *** zordu" || msgs[3].(map[string]any)["quick_id"] != "gg" {
		t.Fatalf("history %s", r.raw)
	}
	want(t, "outsider history", x.do(http.MethodGet, "/v1/chat/channels/"+ch+"/messages", tokC, nil), 404, "NOT_FOUND")
	if o := x.post("/v1/chat/channels", tokB, map[string]any{"team_id": team}); o.get("unread") != 5.0 {
		t.Fatalf("unread %s", o.raw)
	}
	want(t, "read", x.post("/v1/chat/channels/"+ch+"/read", tokB, map[string]any{"last_read_id": msgs[0].(map[string]any)["id"]}), 204, "")
	if o := x.post("/v1/chat/channels", tokB, map[string]any{"team_id": team}); o.get("unread") != 0.0 {
		t.Fatalf("after read %s", o.raw)
	}

	// Removal: the sender or staff.
	first := str(msgs[4].(map[string]any)["id"])
	want(t, "other removes", x.do(http.MethodDelete, "/v1/chat/messages/"+first, tokB, nil), 404, "NOT_FOUND")
	want(t, "sender removes", x.do(http.MethodDelete, "/v1/chat/messages/"+first, tokA, nil), 204, "")
	if d := data(wb.find("removed", func(f map[string]any) bool { return f["t"] == "chat.message_removed" })); d["id"] != first {
		t.Fatalf("removed %v", d)
	}
	want(t, "staff removes", x.do(http.MethodDelete, "/v1/chat/messages/"+str(msgs[3].(map[string]any)["id"]), tokM, nil), 204, "")
	r = x.do(http.MethodGet, "/v1/chat/channels/"+ch+"/messages", tokA, nil)
	if last := r.get("messages").([]any)[4].(map[string]any); last["removed"] != true || last["body"] != nil {
		t.Fatalf("blanked %s", r.raw)
	}

	// A chat_mute stops sending.
	rep := x.post("/v1/reports", tokA, map[string]any{"target_kind": "user", "target_id": b, "reason": "harassment"})
	want(t, "mute", x.post("/v1/admin/cases/"+str(rep.get("case_id"))+"/actions", tokM, map[string]any{
		"action": "user_sanction", "sanction": map[string]any{"kind": "chat_mute", "hours": 1, "reason_code": "harassment"}}), 200, "")
	if f := send(wb, map[string]any{"body": "hello?"}); data(f)["code"] != "SANCTIONED" || data(f)["details"].(map[string]any)["kind"] != "chat_mute" {
		t.Fatalf("muted %v", f)
	}
}
