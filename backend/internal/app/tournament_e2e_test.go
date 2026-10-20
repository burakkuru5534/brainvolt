package app

import (
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/notification"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/progression"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/tournament"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
)

// publishedTo reads the payloads of an event delivered to a subscriber.
func (x *e2e) publishedTo(event, subscriber string) []events.Envelope {
	x.t.Helper()
	rows, err := x.db.Query(x.t.Context(), `SELECT args->'event' FROM river_job WHERE args->'event'->>'name' = $1 AND args->>'subscriber' = $2 ORDER BY id`,
		event, subscriber)
	if err != nil {
		x.t.Fatal(err)
	}
	envs, err := pgx.CollectRows(rows, pgx.RowTo[events.Envelope])
	if err != nil {
		x.t.Fatal(err)
	}
	return envs
}

// A user tournament end to end (schema-platform §3, D-090): admin
// creation, registration rules, check-in, seeding at the start, a match
// through game and its result, walkovers when the ready window closes,
// final places, XP and notices.
func TestTournament(t *testing.T) {
	x := newE2E(t)
	tokAdm, adm, _ := x.fullUser()
	if _, err := x.mods.identity.SetRole(t.Context(), adm.String(), identity.RoleAdmin, true); err != nil {
		t.Fatal(err)
	}
	x.seedQuestions(adm)
	toks := map[uuid.UUID]string{}
	var users []uuid.UUID
	for range 5 {
		tok, u, _ := x.fullUser()
		toks[u] = tok
		users = append(users, u)
	}
	tokGuest, _ := x.signedIn()

	now := time.Now()
	body := map[string]any{
		"texts":                  map[string]any{"tr": map[string]any{"name": "Bilgi Kupası"}, "en": map[string]any{"name": "Quiz Cup"}},
		"entity_kind":            "user",
		"match_config":           map[string]any{"match_type": "quick_duel", "format": "1v1"},
		"min_entrants":           2,
		"max_entrants":           4,
		"registration_opens_at":  now.Add(-time.Minute),
		"registration_closes_at": now.Add(30 * time.Minute),
		"starts_at":              now.Add(40 * time.Minute),
		"rewards":                map[string]any{"places": []any{map[string]any{"place": 1, "xp": 300}, map[string]any{"place": 2, "xp": 150}}, "participation_xp": 20},
	}
	want(t, "not admin", x.post("/v1/admin/tournaments", toks[users[0]], body), 403, "FORBIDDEN")
	want(t, "ranked", x.post("/v1/admin/tournaments", tokAdm, merge(body, map[string]any{"match_config": map[string]any{"match_type": "ranked", "format": "1v1"}})), 422, "MODE_NOT_ALLOWED")
	want(t, "format", x.post("/v1/admin/tournaments", tokAdm, merge(body, map[string]any{"entity_kind": "couple"})), 400, "VALIDATION_FAILED")
	r := x.post("/v1/admin/tournaments", tokAdm, body)
	if r.code != 201 || r.get("status") != "draft" || r.get("name") != "Bilgi Kupası" {
		t.Fatalf("create %s", r.raw)
	}
	id := str(r.get("id"))
	path := "/v1/tournaments/" + id
	want(t, "draft hidden", x.do(http.MethodGet, path, toks[users[0]], nil), 404, "NOT_FOUND")
	want(t, "publish", x.do(http.MethodPatch, "/v1/admin/tournaments/"+id, tokAdm, map[string]any{"status": "registration"}), 200, "")
	if r := x.do(http.MethodGet, "/v1/tournaments?status=registration", toks[users[0]], nil); len(r.get("tournaments").([]any)) != 1 {
		t.Fatalf("list %s", r.raw)
	}

	// Registration: full accounts, at most max_entrants, once.
	want(t, "guest", x.post(path+"/entries", tokGuest, nil), 403, "ACCOUNT_UPGRADE_REQUIRED")
	for _, u := range users[:4] {
		if r := x.post(path+"/entries", toks[u], nil); r.code != 201 || r.get("status") != "registered" {
			t.Fatalf("register %s", r.raw)
		}
	}
	want(t, "again", x.post(path+"/entries", toks[users[0]], nil), 403, "TOURNAMENT_REQUIREMENTS_NOT_MET")
	want(t, "full", x.post(path+"/entries", toks[users[4]], nil), 409, "TOURNAMENT_FULL")
	want(t, "withdraw", x.do(http.MethodDelete, path+"/entries/mine", toks[users[3]], nil), 204, "")
	if r := x.post(path+"/entries", toks[users[4]], nil); r.code != 201 {
		t.Fatalf("free place %s", r.raw)
	}
	players := []uuid.UUID{users[0], users[1], users[2], users[4]}

	// Check-in in the last hour; users[2] forgets and drops out.
	for _, u := range []uuid.UUID{users[0], users[1], users[4]} {
		if r := x.post(path+"/check-in", toks[u], nil); r.code != 200 || r.get("status") != "checked_in" {
			t.Fatalf("check-in %s", r.raw)
		}
	}
	want(t, "withdrawn", x.post(path+"/check-in", toks[users[3]], nil), 404, "NOT_FOUND")

	// The start: three seeded entries, the top seed has a bye.
	if _, err := x.db.Exec(t.Context(), `UPDATE tournament.tournaments SET registration_closes_at = now() - interval '2 seconds',
		starts_at = now() - interval '1 second' WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}
	if err := x.mods.tournament.Tick(t.Context()); err != nil {
		t.Fatal(err)
	}
	want(t, "late", x.post(path+"/entries", toks[users[3]], nil), 409, "TOURNAMENT_REGISTRATION_CLOSED")
	d := x.do(http.MethodGet, path, toks[users[0]], nil)
	if d.get("status") != "running" || d.get("rounds") != 2.0 || d.get("entrants") != 3.0 {
		t.Fatalf("started %s", d.raw)
	}
	entryOf := map[string]uuid.UUID{} // entry id → user
	seedOf := map[uuid.UUID]float64{}
	for _, e := range d.get("entries").([]any) {
		e := e.(map[string]any)
		u := uuid.MustParse(str(e["entity_id"]))
		entryOf[str(e["id"])] = u
		if e["status"] == "withdrawn" {
			if u != users[2] && u != users[3] {
				t.Fatalf("withdrawn %v", e)
			}
			continue
		}
		seedOf[u], _ = e["seed"].(float64)
	}
	bySeed := func(s float64) uuid.UUID {
		for u, v := range seedOf {
			if v == s {
				return u
			}
		}
		t.Fatalf("no seed %v", s)
		return uuid.Nil
	}
	top, second, third := bySeed(1), bySeed(2), bySeed(3)
	slots := d.get("bracket").([]any)
	if len(slots) != 3 || slots[0].(map[string]any)["status"] != "walkover" || slots[1].(map[string]any)["status"] != "ready" ||
		entryOf[str(slots[0].(map[string]any)["winner_entry_id"])] != top {
		t.Fatalf("bracket %s", d.raw)
	}
	if n := len(x.publishedTo(tournament.EventRoundScheduled, notification.SubscriberTournamentRound)); n != 1 {
		t.Fatalf("round scheduled %d", n)
	}

	// Semi-final: both show up, the match is created through game; the
	// lower seed wins it.
	if r := x.post(path+"/play", toks[second], nil); r.code != 200 || r.get("match_id") != nil {
		t.Fatalf("first present %s", r.raw)
	}
	want(t, "no match yet", x.post(path+"/play", toks[top], nil), 409, "MATCH_STATE_INVALID")
	r = x.post(path+"/play", toks[third], nil)
	match := str(r.get("match_id"))
	if r.code != 200 || match == "" {
		t.Fatalf("match %s", r.raw)
	}
	var origin string
	if err := x.db.QueryRow(t.Context(), `SELECT origin FROM game.matches WHERE id = $1`, match).Scan(&origin); err != nil || origin != "tournament" {
		t.Fatalf("game match %q %v", origin, err)
	}
	completed := map[string]any{"match_id": match, "origin": "tournament", "winner_side_no": 2, "is_draw": false}
	x.deliver(x.mods.tournament.OnMatchCompleted, game.EventMatchCompleted, completed)
	d = x.do(http.MethodGet, path, toks[top], nil)
	final := d.get("bracket").([]any)[2].(map[string]any)
	if final["status"] != "ready" || entryOf[str(final["entry_a_id"])] != top || entryOf[str(final["entry_b_id"])] != third ||
		d.get("my_match") == nil {
		t.Fatalf("final %s", d.raw)
	}

	// Final: only the top seed shows up before the window closes.
	if r := x.post(path+"/play", toks[top], nil); r.code != 200 {
		t.Fatalf("present %s", r.raw)
	}
	if _, err := x.db.Exec(t.Context(), `UPDATE tournament.bracket_matches SET ready_until = now() - interval '1 second' WHERE tournament_id = $1 AND status = 'ready'`, id); err != nil {
		t.Fatal(err)
	}
	if err := x.mods.tournament.Tick(t.Context()); err != nil {
		t.Fatal(err)
	}
	want(t, "closed", x.post(path+"/play", toks[third], nil), 409, "MATCH_STATE_INVALID")
	d = x.do(http.MethodGet, path, toks[top], nil)
	places := map[uuid.UUID]any{}
	for _, e := range d.get("entries").([]any) {
		e := e.(map[string]any)
		places[uuid.MustParse(str(e["entity_id"]))] = e["final_place"]
	}
	if d.get("status") != "finished" || places[top] != 1.0 || places[third] != 2.0 || places[second] != 3.0 {
		t.Fatalf("finished %s", d.raw)
	}

	// Rewards: XP per place, a notice each.
	fin := x.publishedTo(tournament.EventFinished, progression.SubscriberTournamentRewards)
	if len(fin) != 1 {
		t.Fatalf("finished events %d", len(fin))
	}
	for range 2 { // idempotent
		if err := x.mods.progression.OnTournamentFinished(t.Context(), fin[0]); err != nil {
			t.Fatal(err)
		}
	}
	xp := func(u uuid.UUID) int {
		var n int
		if err := x.db.QueryRow(t.Context(), `SELECT coalesce(sum(amount), 0) FROM progression.xp_ledger WHERE user_id = $1 AND source = 'tournament'`, u).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if xp(top) != 300 || xp(third) != 150 || xp(second) != 20 || xp(users[2]) != 0 {
		t.Fatalf("xp %d %d %d", xp(top), xp(third), xp(second))
	}
	notif := notification.New(notification.Deps{DB: x.db, Profiles: x.mods.profile, Clock: clock.Real{}, Logger: discard})
	if err := notif.OnTournamentFinished(t.Context(), x.publishedTo(tournament.EventFinished, notification.SubscriberTournamentFinished)[0]); err != nil {
		t.Fatal(err)
	}
	r = x.do(http.MethodGet, "/v1/me/inbox", toks[top], nil)
	if items := r.get("items").([]any); len(items) != 1 || items[0].(map[string]any)["body"] != "You finished in place 1. +300 XP!" ||
		items[0].(map[string]any)["deep_link"] != "/tournaments/"+id {
		t.Fatalf("inbox %s", r.raw)
	}
	if !slices.Contains(players, top) {
		t.Fatal("top seed is a player")
	}
}

func merge(a, b map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}

// No-shows (D-090): a tournament match cancelled at its ready check goes
// to the side whose players readied; too few check-ins cancel a
// tournament at its start.
func TestTournamentNoShows(t *testing.T) {
	x := newE2E(t)
	tokAdm, adm, _ := x.fullUser()
	if _, err := x.mods.identity.SetRole(t.Context(), adm.String(), identity.RoleAdmin, true); err != nil {
		t.Fatal(err)
	}
	x.seedQuestions(adm)
	tokA, a, _ := x.fullUser()
	tokB, b, _ := x.fullUser()
	now := time.Now()
	create := func() string {
		r := x.post("/v1/admin/tournaments", tokAdm, map[string]any{
			"texts": map[string]any{"tr": map[string]any{"name": "Gece Turnuvası"}}, "entity_kind": "user",
			"match_config": map[string]any{"match_type": "quick_duel", "format": "1v1"}, "min_entrants": 2, "max_entrants": 8,
			"registration_opens_at": now.Add(-time.Minute), "registration_closes_at": now.Add(10 * time.Minute), "starts_at": now.Add(20 * time.Minute),
			"rewards": map[string]any{"places": []any{}}, "publish": true,
		})
		if r.code != 201 || r.get("status") != "registration" {
			t.Fatalf("create %s", r.raw)
		}
		return str(r.get("id"))
	}
	start := func(id string) {
		if _, err := x.db.Exec(t.Context(), `UPDATE tournament.tournaments SET registration_closes_at = now() - interval '2 seconds',
			starts_at = now() - interval '1 second' WHERE id = $1`, id); err != nil {
			t.Fatal(err)
		}
		if err := x.mods.tournament.Tick(t.Context()); err != nil {
			t.Fatal(err)
		}
	}

	// One check-in of two: cancelled.
	lonely := create()
	for _, tok := range []string{tokA, tokB} {
		want(t, "register", x.post("/v1/tournaments/"+lonely+"/entries", tok, nil), 201, "")
	}
	want(t, "check-in", x.post("/v1/tournaments/"+lonely+"/check-in", tokA, nil), 200, "")
	start(lonely)
	if r := x.do(http.MethodGet, "/v1/tournaments/"+lonely, tokA, nil); r.get("status") != "cancelled" {
		t.Fatalf("too few %s", r.raw)
	}

	// The final's match is cancelled at the ready check; only A readied.
	id := create()
	for _, tok := range []string{tokA, tokB} {
		want(t, "register", x.post("/v1/tournaments/"+id+"/entries", tok, nil), 201, "")
		want(t, "check-in", x.post("/v1/tournaments/"+id+"/check-in", tok, nil), 200, "")
	}
	start(id)
	want(t, "present", x.post("/v1/tournaments/"+id+"/play", tokA, nil), 200, "")
	match := str(x.post("/v1/tournaments/"+id+"/play", tokB, nil).get("match_id"))
	if match == "" {
		t.Fatal("no match")
	}
	x.deliver(x.mods.tournament.OnMatchCancelled, game.EventMatchCancelled, map[string]any{
		"match_id": match, "origin": "tournament", "reason": "not_all_ready", "participant_ids": []uuid.UUID{a, b}, "ready_ids": []uuid.UUID{a},
	})
	d := x.do(http.MethodGet, "/v1/tournaments/"+id, tokB, nil)
	slot := d.get("bracket").([]any)[0].(map[string]any)
	if d.get("status") != "finished" || slot["status"] != "walkover" || d.get("my_entry").(map[string]any)["final_place"] != 2.0 {
		t.Fatalf("walkover %s", d.raw)
	}
}
