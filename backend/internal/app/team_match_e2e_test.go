package app

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game"
)

// Shared Answer live (D-035): teammates see each other's proposals, the
// other side only "redacted"; a lock is the side's answer, shown to the
// side; both locks end the round.
func TestLiveSharedAnswer(t *testing.T) {
	x := newE2E(t)
	var toks [4]string
	var users [4]uuid.UUID
	for i := range 4 {
		toks[i], users[i] = x.signedIn()
	}
	x.seedQuestions(users[0])
	var ws [4]*wsClient
	for i := range 4 {
		ws[i], _ = x.auth(toks[i], "inst-"+string(rune('a'+i)), nil)
	}
	c, err := x.mods.game.Start(t.Context(), game.CreateRequest{
		Config: game.Config{MatchType: "quick_duel", Format: "2v2", TeamMode: "shared_answer", Origin: "private_room"},
		Sides: [2][]game.Member{
			{{UserID: users[0], Locale: "tr"}, {UserID: users[1], Locale: "tr"}},
			{{UserID: users[2], Locale: "en"}, {UserID: users[3], Locale: "en"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	id := c.Match.ID.String()
	for _, w := range ws {
		w.answer(w.send("match.join", map[string]any{"match_id": id}))
		w.send("match.ready", map[string]any{"match_id": id})
	}
	started := data(ws[0].findIn(10*time.Second, "round.started", func(f map[string]any) bool { return f["t"] == "round.started" }))
	if wait := time.Until(time.UnixMilli(int64(started["show_at"].(float64)))); wait > 0 {
		time.Sleep(wait)
	}
	opts := started["options"].([]any)
	opt := func(i int) any { return opts[i].(map[string]any)["id"] }

	// A proposes; teammate B sees it, C gets a redacted frame.
	if f := ws[0].answer(ws[0].send("match.propose", map[string]any{"match_id": id, "round_no": 1, "option_id": opt(0)})); f["t"] != "sys.ack" {
		t.Fatalf("propose %v", f)
	}
	props := data(ws[1].find("proposals", func(f map[string]any) bool { return f["t"] == "round.proposals" }))
	if list := props["proposals"].([]any); len(list) != 1 || list[0].(map[string]any)["user_id"] != users[0].String() {
		t.Fatalf("teammate proposals %v", props)
	}
	if red := data(ws[2].find("redacted proposals", func(f map[string]any) bool { return f["t"] == "round.proposals" })); red["redacted"] != true || red["proposals"] != nil {
		t.Fatalf("opponent saw proposals %v", red)
	}
	if f := ws[1].answer(ws[1].send("match.lock", map[string]any{"match_id": id, "round_no": 1, "option_id": opt(1)})); f["t"] != "sys.ack" {
		t.Fatalf("lock %v", f)
	}
	locked := func(w *wsClient) map[string]any {
		return data(w.find("lock", func(f map[string]any) bool {
			return f["t"] == "round.answer_received" && data(f)["user_id"] == users[1].String()
		}))
	}
	if d := locked(ws[0]); d["option_id"] != opt(1) {
		t.Fatalf("teammate does not see the lock %v", d)
	}
	if d := locked(ws[3]); d["option_id"] != nil {
		t.Fatalf("opponent sees the lock %v", d)
	}
	if f := ws[0].answer(ws[0].send("match.lock", map[string]any{"match_id": id, "round_no": 1, "option_id": opt(0)})); data(f)["code"] != "ANSWER_DUPLICATE" {
		t.Fatalf("second lock %v", f)
	}
	ws[2].answer(ws[2].send("match.lock", map[string]any{"match_id": id, "round_no": 1, "option_id": opt(0)}))
	rev := data(ws[3].find("revealed", func(f map[string]any) bool { return f["t"] == "round.revealed" }))
	if len(rev["sides"].([]any)) != 2 {
		t.Fatalf("revealed %v", rev)
	}
	ws[0].answer(ws[0].send("match.leave", map[string]any{"match_id": id}))
}

// Persistent sides: a team side fields current members (R16); couple
// matches are played by two active couples.
func TestTeamAndCoupleSides(t *testing.T) {
	x := newE2E(t)
	var toks [8]string
	var users [8]uuid.UUID
	for i := range 8 {
		toks[i], users[i], _ = x.fullUser()
	}
	x.seedQuestions(users[0])
	for i := 0; i < 8; i += 2 {
		x.befriend(toks[i], users[i+1], toks[i+1])
	}
	team := func(owner, member int, name, tag string) uuid.UUID {
		r := x.post("/v1/teams", toks[owner], map[string]any{"name": name, "tag": tag, "emblem_id": "bolt"})
		id := str(r.get("id"))
		inv := x.post("/v1/teams/"+id+"/invites", toks[owner], map[string]any{"user_id": users[member]})
		want(t, "accept invite", x.do(http.MethodPost, "/v1/team-invites/"+str(inv.get("id"))+"/accept", toks[member], nil), 204, "")
		return uuid.MustParse(id)
	}
	t1, t2 := team(0, 1, "Alpha Team", "ALF"), team(2, 3, "Beta Team", "BET")
	sides := [2][]game.Member{
		{{UserID: users[0], Locale: "tr"}, {UserID: users[1], Locale: "tr"}},
		{{UserID: users[2], Locale: "tr"}, {UserID: users[3], Locale: "tr"}},
	}
	ftr := game.Config{MatchType: "quick_duel", Format: "2v2", TeamMode: "fast_team_round", Origin: "challenge"}

	// Wrong lineup: a non-member on team 2's side.
	bad := [2][]game.Member{sides[0], {{UserID: users[2], Locale: "tr"}, {UserID: users[0], Locale: "tr"}}}
	if _, err := x.mods.game.Create(t.Context(), game.CreateRequest{Config: ftr, Sides: bad, Teams: [2]*uuid.UUID{&t1, &t2}}); err == nil {
		t.Fatal("non-member lineup accepted")
	}
	c, err := x.mods.game.Create(t.Context(), game.CreateRequest{Config: ftr, Sides: sides, Teams: [2]*uuid.UUID{&t1, &t2}})
	if err != nil {
		t.Fatal(err)
	}
	var team1 uuid.UUID
	if err := x.db.QueryRow(t.Context(), `SELECT team_id FROM game.match_sides WHERE match_id = $1 AND side_no = 1`, c.Match.ID).Scan(&team1); err != nil || team1 != t1 {
		t.Fatalf("side team %v %v", team1, err)
	}

	// Couples (other players: the first four wait in their match).
	couple := func(a, b int) uuid.UUID {
		r := x.post("/v1/couples", toks[a], map[string]any{"partner_user_id": users[b]})
		id := str(r.get("id"))
		want(t, "accept couple", x.do(http.MethodPost, "/v1/couples/"+id+"/accept", toks[b], nil), 200, "")
		return uuid.MustParse(id)
	}
	c1, c2 := couple(4, 5), couple(6, 7)
	pairs := [2][]game.Member{
		{{UserID: users[4], Locale: "tr"}, {UserID: users[5], Locale: "tr"}},
		{{UserID: users[6], Locale: "tr"}, {UserID: users[7], Locale: "tr"}},
	}
	cc := game.Config{MatchType: "quick_duel", Format: "couple", TeamMode: "fast_team_round", Origin: "challenge"}
	if _, err := x.mods.game.Create(t.Context(), game.CreateRequest{Config: cc, Sides: pairs}); err == nil {
		t.Fatal("couple match without couples accepted")
	}
	swapped := [2][]game.Member{{pairs[0][0], pairs[1][0]}, {pairs[0][1], pairs[1][1]}}
	if _, err := x.mods.game.Create(t.Context(), game.CreateRequest{Config: cc, Sides: swapped, Couples: [2]*uuid.UUID{&c1, &c2}}); err == nil {
		t.Fatal("mixed-up couples accepted")
	}
	c2m, err := x.mods.game.Create(t.Context(), game.CreateRequest{Config: cc, Sides: pairs, Couples: [2]*uuid.UUID{&c1, &c2}})
	if err != nil {
		t.Fatal(err)
	}
	var couple2 uuid.UUID
	if err := x.db.QueryRow(t.Context(), `SELECT couple_id FROM game.match_sides WHERE match_id = $1 AND side_no = 2`, c2m.Match.ID).Scan(&couple2); err != nil || couple2 != c2 {
		t.Fatalf("side couple %v %v", couple2, err)
	}
}
