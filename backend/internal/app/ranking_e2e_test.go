package app

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
)

// completed builds a ranked, matchmade MatchCompleted where side 1 wins:
// users[0] on side 1 answered both rounds right, users[1] on side 2 one.
func completed(t *testing.T, season uuid.UUID, format string, sides [2][]uuid.UUID, entities [2]*uuid.UUID) game.MatchCompleted {
	t.Helper()
	cat := uuid.New()
	p := game.MatchCompleted{
		MatchID: uuid.New(), MatchType: "ranked", Format: game.Format(format), Origin: game.OriginMatchmaking,
		IsRated: true, RatingBucket: "solo", SeasonID: &season, EndReason: "completed", WinnerSideNo: 1,
		RoundsPlayed: 2, EndedAt: time.Now().UTC(),
		Sides: []game.SidePayload{{SideNo: 1, Result: "win"}, {SideNo: 2, Result: "loss"}},
		Rounds: []game.RoundPayload{
			{RoundNo: 1, CategoryID: cat, SideCorrect: [2]bool{true, true}, CorrectUserIDs: []uuid.UUID{sides[0][0], sides[1][0]}},
			{RoundNo: 2, CategoryID: cat, SideCorrect: [2]bool{true, false}, CorrectUserIDs: []uuid.UUID{sides[0][0]}},
		},
	}
	if format == "couple" {
		p.RatingBucket = "couple"
		p.Sides[0].CoupleID, p.Sides[1].CoupleID = entities[0], entities[1]
	}
	for i, us := range sides {
		for _, u := range us {
			p.Participants = append(p.Participants, game.ParticipantPayload{UserID: u, SideNo: i + 1, TierAtStart: "full", Status: "finished"})
		}
	}
	return p
}

func (x *e2e) consume(p game.MatchCompleted) {
	x.t.Helper()
	payload, err := json.Marshal(p)
	if err != nil {
		x.t.Fatal(err)
	}
	// Twice: consumption is idempotent.
	for range 2 {
		if err := x.mods.ranking.OnMatchCompleted(x.t.Context(), events.Envelope{Name: game.EventMatchCompleted, Payload: payload}); err != nil {
			x.t.Fatal(err)
		}
	}
}

func (x *e2e) rating(kind string, id uuid.UUID, bucket string) (rating, rd float64) {
	x.t.Helper()
	if err := x.db.QueryRow(x.t.Context(), `SELECT rating, rd FROM ranking.ratings WHERE entity_kind = $1 AND entity_id = $2 AND bucket = $3`,
		kind, id, bucket).Scan(&rating, &rd); err != nil {
		x.t.Fatalf("rating %s %s: %v", kind, id, err)
	}
	return rating, rd
}

func (x *e2e) standing(season uuid.UUID, kind string, id uuid.UUID) (sp int, league string, matches int) {
	x.t.Helper()
	if err := x.db.QueryRow(x.t.Context(), `SELECT season_points, league_code, matches FROM ranking.season_standings
		WHERE season_id = $1 AND entity_kind = $2 AND entity_id = $3`, season, kind, id).Scan(&sp, &league, &matches); err != nil {
		x.t.Fatalf("standing %s %s: %v", kind, id, err)
	}
	return sp, league, matches
}

func (x *e2e) ledgerPoints(match uuid.UUID, kind string, id uuid.UUID) (overall, rows int) {
	x.t.Helper()
	if err := x.db.QueryRow(x.t.Context(), `SELECT coalesce(sum(points) FILTER (WHERE category_id IS NULL), 0), count(*)
		FROM ranking.points_ledger WHERE match_id = $1 AND entity_kind = $2 AND entity_id = $3`, match, kind, id).Scan(&overall, &rows); err != nil {
		x.t.Fatal(err)
	}
	return overall, rows
}

func TestRankingConsumesRankedDuel(t *testing.T) {
	x := newE2E(t)
	season, ok, err := x.mods.ranking.ActiveSeason(t.Context())
	if err != nil || !ok {
		t.Fatalf("active season %v %v", ok, err)
	}
	again, _, _ := x.mods.ranking.ActiveSeason(t.Context())
	if again != season {
		t.Fatal("a second season was opened")
	}
	cur, err := x.mods.ranking.CurrentSeason(t.Context())
	if err != nil || cur.Names["tr"] == "" || cur.EndsAt.Sub(cur.StartsAt) != 56*24*time.Hour || cur.StartsAt.Weekday() != time.Sunday {
		// Monday 00:00 Istanbul is Sunday 21:00 UTC.
		t.Fatalf("season %+v %v", cur, err)
	}

	a, b := uuid.New(), uuid.New()
	p := completed(t, season, "1v1", [2][]uuid.UUID{{a}, {b}}, [2]*uuid.UUID{})
	x.consume(p)

	ra, rda := x.rating("user", a, "solo")
	rb, _ := x.rating("user", b, "solo")
	if ra <= 1500 || rb >= 1500 || rda >= 350 {
		t.Fatalf("ratings a %v (rd %v) b %v", ra, rda, rb)
	}
	var changes int
	if err := x.db.QueryRow(t.Context(), `SELECT count(*) FROM ranking.rating_changes WHERE match_id = $1`, p.MatchID).Scan(&changes); err != nil || changes != 2 {
		t.Fatalf("rating changes %d %v", changes, err)
	}
	if sp, league, n := x.standing(season, "user", a); sp != 30 || league != "bronze" || n != 1 {
		t.Fatalf("winner standing %d %s %d", sp, league, n)
	}
	if sp, _, _ := x.standing(season, "user", b); sp != 0 {
		t.Fatalf("loser SP %d (floor 0)", sp)
	}
	// Leaderboard points: win 20 + 2 × 2 correct; loss 0 + 2 × 1.
	if pts, rows := x.ledgerPoints(p.MatchID, "user", a); pts != 24 || rows != 2 {
		t.Fatalf("winner ledger %d (%d rows)", pts, rows)
	}
	if pts, _ := x.ledgerPoints(p.MatchID, "user", b); pts != 2 {
		t.Fatalf("loser ledger %d", pts)
	}
	week := weekBoard("user", "all")
	if score, err := x.rdb.ZScore(t.Context(), week, a.String()).Result(); err != nil || score != 24 {
		t.Fatalf("weekly board %s: %v %v", week, score, err)
	}
	cat := "cat:" + p.Rounds[0].CategoryID.String()
	if score, err := x.rdb.ZScore(t.Context(), "rank:lb:a:user:global:"+cat, a.String()).Result(); err != nil || score != 24 {
		t.Fatalf("all-time category board: %v %v", score, err)
	}
	if score, err := x.rdb.ZScore(t.Context(), "rank:lb:s:"+jsonNum(float64(cur.Number))+":user:global:all", a.String()).Result(); err != nil || score != 30 {
		t.Fatalf("season board: %v %v", score, err)
	}

	// Three wins in a row: the third gets the +5 streak bonus.
	for range 2 {
		x.consume(completed(t, season, "1v1", [2][]uuid.UUID{{a}, {uuid.New()}}, [2]*uuid.UUID{}))
	}
	if sp, _, n := x.standing(season, "user", a); sp != 30+30+35 || n != 3 {
		t.Fatalf("streak SP %d after %d", sp, n)
	}

	// matchmaking reads the stored rating.
	r, err := x.mods.ranking.Rating(t.Context(), a, "solo")
	if err != nil || r.Value <= 1500 {
		t.Fatalf("rating reader %+v %v", r, err)
	}
	if r, _ := x.mods.ranking.Rating(t.Context(), uuid.New(), "solo"); r.Value != 1500 || r.RD != 350 {
		t.Fatalf("new player rating %+v", r)
	}
}

func TestRankingConsumesCoupleContest(t *testing.T) {
	x := newE2E(t)
	season, _, err := x.mods.ranking.ActiveSeason(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	c1, c2 := uuid.New(), uuid.New()
	s1, s2 := []uuid.UUID{uuid.New(), uuid.New()}, []uuid.UUID{uuid.New(), uuid.New()}
	p := completed(t, season, "couple", [2][]uuid.UUID{s1, s2}, [2]*uuid.UUID{&c1, &c2})
	x.consume(p)

	// New couple entities start at the partners' mean with RD 200.
	if r, rd := x.rating("couple", c1, "couple"); r <= 1500 || rd >= 200 {
		t.Fatalf("couple rating %v rd %v", r, rd)
	}
	if r, _ := x.rating("user", s2[1], "couple"); r >= 1500 {
		t.Fatalf("losing partner's couple rating %v", r)
	}
	if sp, _, _ := x.standing(season, "couple", c1); sp != 30 {
		t.Fatalf("couple SP %d", sp)
	}
	// Side-correct rounds: side 1 both, side 2 one.
	if pts, _ := x.ledgerPoints(p.MatchID, "couple", c1); pts != 24 {
		t.Fatalf("couple ledger %d", pts)
	}
	if pts, _ := x.ledgerPoints(p.MatchID, "couple", c2); pts != 2 {
		t.Fatalf("losing couple ledger %d", pts)
	}
	if _, err := x.rdb.ZScore(t.Context(), weekBoard("couple", "all"), c1.String()).Result(); err != nil {
		t.Fatalf("couple weekly board: %v", err)
	}
}

// weekBoard is this week's global board key.
func weekBoard(entity, category string) string {
	loc, _ := time.LoadLocation("Europe/Istanbul")
	y, w := time.Now().In(loc).ISOWeek()
	return "rank:lb:w:" + jsonNum(float64(y)) + "-W" + pad2(w) + ":" + entity + ":global:" + category
}

func pad2(n int) string {
	if n < 10 {
		return "0" + jsonNum(float64(n))
	}
	return jsonNum(float64(n))
}
