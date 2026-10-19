package app

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
)

func (x *e2e) signals(user uuid.UUID, kind string) int {
	x.t.Helper()
	var n int
	if err := x.db.QueryRow(x.t.Context(), `SELECT count(*) FROM moderation.cheat_signals WHERE user_id = $1 AND kind = $2`, user, kind).Scan(&n); err != nil {
		x.t.Fatal(err)
	}
	return n
}

func (x *e2e) deliver(handler events.Handler, name string, payload any) {
	x.t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		x.t.Fatal(err)
	}
	for range 2 { // de-duplicated
		if err := handler(x.t.Context(), events.Envelope{Name: name, Payload: body}); err != nil {
			x.t.Fatal(err)
		}
	}
}

// Anti-cheat signals (schema-moderation §4, D-084): superhuman answers,
// refused-submission bursts, protocol violations and repeated server
// cancellations open a case on the player; nothing is sanctioned.
func TestCheatSignals(t *testing.T) {
	x := newE2E(t)
	tokA, a, _ := x.fullUser()
	_, b, _ := x.fullUser()
	tokM, m, _ := x.fullUser()
	x.seedQuestions(m)
	if _, err := x.mods.identity.SetRole(t.Context(), m.String(), identity.RoleModerator, true); err != nil {
		t.Fatal(err)
	}

	p := finishedDuel(a, b, false)
	p.Participants[0].FastCorrect, p.Participants[0].FastestMS = 4, 120
	p.Participants[1].FastCorrect, p.Participants[1].Rejects = 1, 7
	x.deliver(x.mods.moderation.OnMatchCompleted, game.EventMatchCompleted, p)
	if x.signals(a, "superhuman_response_time") != 1 || x.signals(b, "superhuman_response_time") != 0 ||
		x.signals(b, "rejected_submission_burst") != 1 || x.signals(a, "rejected_submission_burst") != 0 {
		t.Fatal("match signals")
	}

	// The queue shows A's case with its signal and measures.
	r := x.do(http.MethodGet, "/v1/admin/cases", tokM, nil)
	var caseA string
	for _, c := range r.get("cases").([]any) {
		cm := c.(map[string]any)
		if cm["target_id"] == a.String() {
			caseA = str(cm["id"])
			if cm["signal_count"] != 1.0 || cm["target_kind"] != "user" {
				t.Fatalf("case %v", cm)
			}
		}
	}
	r = x.do(http.MethodGet, "/v1/admin/cases/"+caseA, tokM, nil)
	if sg := r.get("signals").([]any); len(sg) != 1 || sg[0].(map[string]any)["details"].(map[string]any)["fast_correct"] != 4.0 {
		t.Fatalf("case detail %s", r.raw)
	}

	// A malformed WebSocket message closes the socket and records one
	// signal an hour.
	for range 2 {
		w, _ := x.auth(tokA, "inst-a", nil)
		if err := w.c.Write(t.Context(), 1, []byte("not json")); err != nil {
			t.Fatal(err)
		}
		w.waitClose()
	}
	deadline := time.Now().Add(3 * time.Second)
	for x.signals(a, "protocol_violation") == 0 && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if n := x.signals(a, "protocol_violation"); n != 1 {
		t.Fatalf("protocol signals %d", n)
	}

	// Three server cancellations in a week.
	var last uuid.UUID
	for range 3 {
		c, err := x.mods.game.Create(t.Context(), game.CreateRequest{
			Config: game.Config{MatchType: "quick_duel", Format: "1v1", Origin: "challenge"},
			Sides:  [2][]game.Member{{{UserID: a, Locale: "tr"}}, {{UserID: b, Locale: "tr"}}},
		})
		if err != nil {
			t.Fatal(err)
		}
		last = c.Match.ID
		// The server gives the match up (T7 server_error).
		if err := x.mods.game.Apply(t.Context(), c.Match, c.Match.Cancel("server_error", uuid.Nil, time.Now().UnixMilli())); err != nil {
			t.Fatal(err)
		}
	}
	x.deliver(x.mods.moderation.OnMatchCancelled, game.EventMatchCancelled, game.MatchCancelled{
		MatchID: last, Reason: "server_error", Origin: "challenge", ParticipantIDs: []uuid.UUID{a, b},
	})
	if x.signals(a, "repeated_server_voids") != 1 || x.signals(b, "repeated_server_voids") != 1 {
		t.Fatal("void signals")
	}
	var sanctions int
	_ = x.db.QueryRow(t.Context(), `SELECT count(*) FROM moderation.sanctions`).Scan(&sanctions)
	if sanctions != 0 {
		t.Fatal("signals sanctioned")
	}
}
