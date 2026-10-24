package app

import (
	"context"
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

// Team queues (D-072, D-094): a persistent team lineup meets two solo
// players on a temporary side; two couples meet in the couple queue. Both
// gather in a waiting lobby first.
func TestTeamAndCoupleQueues(t *testing.T) {
	x := newE2E(t)
	var toks [6]string
	var users [6]uuid.UUID
	var ws [6]*wsClient
	for i := range 4 {
		toks[i], users[i], _ = x.fullUser()
	}
	for i := 4; i < 6; i++ {
		toks[i], users[i] = x.signedIn()
	}
	for i := range 6 {
		ws[i], _ = x.auth(toks[i], "inst-"+string(rune('a'+i)), nil)
	}
	x.seedQuestions(users[0])
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() { _ = x.mods.matchmaking.Run(ctx) }()

	x.befriend(toks[0], users[1], toks[1])
	r := x.post("/v1/teams", toks[0], map[string]any{"name": "Queue Team", "tag": "QTM", "emblem_id": "star"})
	teamID := str(r.get("id"))
	inv := x.post("/v1/teams/"+teamID+"/invites", toks[0], map[string]any{"user_id": users[1]})
	want(t, "join team", x.do(http.MethodPost, "/v1/team-invites/"+str(inv.get("id"))+"/accept", toks[1], nil), 204, "")

	duo := map[string]any{"match_type": "sudden_death", "format": "2v2"}
	want(t, "lineup not full", x.post("/v1/matchmaking/tickets", toks[0], map[string]any{"match_type": "sudden_death", "format": "2v2", "team_id": teamID}), 400, "VALIDATION_FAILED")
	r = x.post("/v1/matchmaking/tickets", toks[0], map[string]any{"match_type": "sudden_death", "format": "2v2", "team_id": teamID, "party_user_ids": []any{users[1]}})
	if r.code != 201 || len(r.get("user_ids").([]any)) != 2 {
		t.Fatalf("team ticket %s", r.raw)
	}
	want(t, "solo 1", x.post("/v1/matchmaking/tickets", toks[4], duo), 201, "")
	want(t, "solo 2", x.post("/v1/matchmaking/tickets", toks[5], duo), 201, "")
	id := gatherLobby(t, []*wsClient{ws[0], ws[1], ws[4], ws[5]}, []uuid.UUID{users[0], users[1], users[4], users[5]})
	var mode, team string
	if err := x.db.QueryRow(t.Context(), `SELECT m.team_answer_mode, coalesce(s.team_id::text, '') FROM game.matches m
		JOIN game.match_sides s ON s.match_id = m.id AND s.side_no = 1 WHERE m.id = $1`, id).Scan(&mode, &team); err != nil || mode != "fast_team_round" || team != teamID {
		t.Fatalf("team match %q %q %v", mode, team, err)
	}
	leaveMatch(t, id, ws[0], ws[1], ws[4], ws[5])

	// Couples: guests cannot (no couple), full accounts with a couple can.
	couple := map[string]any{"match_type": "quick_duel", "format": "couple"}
	want(t, "guest couple", x.post("/v1/matchmaking/tickets", toks[4], couple), 422, "MODE_NOT_ALLOWED")
	x.befriend(toks[2], users[3], toks[3])
	pair := func(a, b int) {
		r := x.post("/v1/couples", toks[a], map[string]any{"partner_user_id": users[b]})
		want(t, "accept couple", x.do(http.MethodPost, "/v1/couples/"+str(r.get("id"))+"/accept", toks[b], nil), 200, "")
	}
	pair(0, 1)
	pair(2, 3)
	// Leaving the first match gave player 0 a leave — not a cooldown yet.
	want(t, "couple 1", x.post("/v1/matchmaking/tickets", toks[1], couple), 201, "")
	want(t, "couple 2", x.post("/v1/matchmaking/tickets", toks[2], couple), 201, "")
	id = gatherLobby(t, []*wsClient{ws[0], ws[1], ws[2], ws[3]}, users[:4])
	var couples int
	if err := x.db.QueryRow(t.Context(), `SELECT count(*) FROM game.match_sides WHERE match_id = $1 AND couple_id IS NOT NULL`, id).Scan(&couples); err != nil || couples != 2 {
		t.Fatalf("couple sides %d %v", couples, err)
	}
	leaveMatch(t, id, ws[0], ws[1], ws[2], ws[3])
}

// gatherLobby is the waiting lobby of a matchmade team match (D-094):
// every player is seated in the same lobby (mm.lobby), everyone readies
// up, and the leader drawn once the lobby is full starts the match.
func gatherLobby(t *testing.T, ws []*wsClient, users []uuid.UUID) string {
	t.Helper()
	room := str(data(ws[0].findWithin(5*time.Second, "mm.lobby"))["room_id"])
	for i := 1; i < len(ws); i++ {
		if got := str(data(ws[i].findWithin(5*time.Second, "mm.lobby"))["room_id"]); got != room {
			t.Fatalf("player %d in lobby %s, want %s", i, got, room)
		}
	}
	for i, w := range ws {
		if f := w.answer(w.send("room.ready", map[string]any{"room_id": room})); f["t"] != "sys.ack" {
			t.Fatalf("player %d ready %v", i, f)
		}
	}
	ready := ws[0].find("full and ready lobby", func(f map[string]any) bool {
		if f["t"] != "room.state" {
			return false
		}
		rm := data(f)["room"].(map[string]any)
		if rm["kind"] != "matchmaking" || rm["leader_user_id"] == nil {
			return false
		}
		members := rm["members"].([]any)
		for _, m := range members {
			if m.(map[string]any)["ready"] != true {
				return false
			}
		}
		return len(members) == len(ws)
	})
	leader := data(ready)["room"].(map[string]any)["leader_user_id"].(string)
	started := false
	for i, u := range users {
		if u.String() != leader {
			if f := ws[i].answer(ws[i].send("room.start", map[string]any{"room_id": room})); f["t"] == "sys.ack" {
				t.Fatalf("player %d started without being the leader", i)
			}
			continue
		}
		if f := ws[i].answer(ws[i].send("room.start", map[string]any{"room_id": room})); f["t"] != "sys.ack" {
			t.Fatalf("leader start %v", f)
		}
		started = true
	}
	if !started {
		t.Fatalf("leader %s is not a player", leader)
	}
	return str(data(ws[0].find("room.match_starting", func(f map[string]any) bool { return f["t"] == "room.match_starting" }))["match_id"])
}
