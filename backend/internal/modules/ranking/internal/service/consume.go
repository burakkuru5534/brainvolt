package service

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	goredis "github.com/redis/go-redis/v9"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/ranking/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/ranking/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
)

// matchCompleted is the part of game.MatchCompleted ranking reads (the
// event's JSON contract; ranking does not import game).
type matchCompleted struct {
	MatchID      uuid.UUID            `json:"match_id"`
	MatchType    string               `json:"match_type"`
	Origin       string               `json:"origin"`
	IsRated      bool                 `json:"is_rated"`
	RatingBucket string               `json:"rating_bucket"`
	SeasonID     *uuid.UUID           `json:"season_id"`
	EndedAt      time.Time            `json:"ended_at"`
	Sides        []sidePayload        `json:"sides"`
	Participants []participantPayload `json:"participants"`
	Rounds       []roundPayload       `json:"rounds"`
}

type sidePayload struct {
	SideNo   int        `json:"side_no"`
	Result   string     `json:"result"`
	TeamID   *uuid.UUID `json:"team_id"`
	CoupleID *uuid.UUID `json:"couple_id"`
}

type participantPayload struct {
	UserID      uuid.UUID `json:"user_id"`
	SideNo      int       `json:"side_no"`
	TierAtStart string    `json:"tier_at_start"`
	Status      string    `json:"status"`
}

type roundPayload struct {
	CategoryID     uuid.UUID   `json:"category_id"`
	SideCorrect    [2]bool     `json:"side_correct"`
	CorrectUserIDs []uuid.UUID `json:"correct_user_ids"`
}

// entity is a rated or ranked entity.
type entity struct {
	kind string
	id   uuid.UUID
}

// member is a participant with what ranking needs about them.
type member struct {
	participantPayload
	country string
}

// side is one side of a completed match seen by ranking.
type side struct {
	idx    int
	result domain.Result
	users  []member
	// entity is the persistent team or couple (nil for temporary sides).
	entity *entity
	// public: no member was restricted at match time (D-016).
	public bool
	// countries are the entity's country boards: the captain's country
	// for a team (D-029), both partners' for a couple (D-047).
	countries []string
}

// boardInc is one Redis leaderboard write.
type boardInc struct {
	key    string
	member string
	score  float64
	// set replaces the score (season SP boards); otherwise it is added.
	set      bool
	expireAt time.Time
}

// OnMatchCompleted consumes game.MatchCompleted once per match: Glicko-2
// ratings of rated matches, Season Points of ranked ones and leaderboard
// points of matchmade ones (progression-formulas §1–4). Redis boards are
// updated after the commit; they can be rebuilt from the ledger.
func (s *Service) OnMatchCompleted(ctx context.Context, e events.Envelope) error {
	var p matchCompleted
	if err := e.Decode(&p); err != nil {
		return err
	}
	ranked, matchmade := p.MatchType == "ranked", p.Origin == "matchmaking"
	if (!p.IsRated && !ranked && !matchmade) || len(p.Sides) != 2 {
		return nil
	}
	sides, err := s.sides(ctx, &p)
	if err != nil {
		return err
	}
	season, err := s.matchSeason(ctx, p.SeasonID)
	if err != nil {
		return err
	}
	var incs []boardInc
	err = postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		incs = nil
		q := sqlc.New(tx)
		n, err := q.MarkProcessed(ctx, sqlc.MarkProcessedParams{MatchID: p.MatchID, Now: s.d.Clock.Now()})
		if err != nil || n == 0 {
			return err
		}
		if p.IsRated && p.RatingBucket != "" {
			if err := rate(ctx, q, &p, sides, season.ID); err != nil {
				return err
			}
		}
		if ranked {
			out, err := standings(ctx, q, &p, sides, season)
			if err != nil {
				return err
			}
			incs = append(incs, out...)
		}
		if matchmade {
			out, err := ledger(ctx, q, &p, sides, season.ID)
			if err != nil {
				return err
			}
			incs = append(incs, out...)
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.applyBoards(ctx, incs)
	return nil
}

// matchSeason is the match's season (ranked) or the active one; matches
// of the preseason before ranking existed (D-064) count for the active one.
func (s *Service) matchSeason(ctx context.Context, id *uuid.UUID) (sqlc.RankingSeason, error) {
	if id != nil {
		se, err := sqlc.New(s.d.DB).GetSeason(ctx, *id)
		if !errors.Is(err, pgx.ErrNoRows) {
			return se, err
		}
	}
	return s.ensureSeason(ctx)
}

// sides groups participants by side with their countries.
func (s *Service) sides(ctx context.Context, p *matchCompleted) ([2]side, error) {
	var out [2]side
	users := make([]uuid.UUID, 0, len(p.Participants)+2)
	for _, u := range p.Participants {
		users = append(users, u.UserID)
	}
	captains := [2]uuid.UUID{}
	for _, sd := range p.Sides {
		if sd.TeamID != nil && s.d.Teams != nil {
			t, err := s.d.Teams.TeamInfo(ctx, *sd.TeamID)
			if err != nil {
				return out, err
			}
			captains[sd.SideNo-1] = t.Captain
			users = append(users, t.Captain)
		}
	}
	countries := map[uuid.UUID]string{}
	if s.d.Cards != nil {
		cards, err := s.d.Cards.Cards(ctx, users)
		if err != nil {
			return out, err
		}
		for id, c := range cards {
			if c.CountryCode != nil {
				countries[id] = *c.CountryCode
			}
		}
	}
	for _, sd := range p.Sides {
		i := sd.SideNo - 1
		out[i] = side{idx: i, result: domain.Result(sd.Result), public: true}
		switch {
		case sd.TeamID != nil:
			out[i].entity = &entity{kind: "team", id: *sd.TeamID}
			out[i].countries = []string{countries[captains[i]]}
		case sd.CoupleID != nil:
			out[i].entity = &entity{kind: "couple", id: *sd.CoupleID}
		}
	}
	for _, u := range p.Participants {
		sd := &out[u.SideNo-1]
		sd.users = append(sd.users, member{participantPayload: u, country: countries[u.UserID]})
		sd.public = sd.public && u.TierAtStart == "full"
		if sd.entity != nil && sd.entity.kind == "couple" {
			sd.countries = append(sd.countries, countries[u.UserID])
		}
	}
	return out, nil
}

// rate applies one Glicko-2 rating period per rated entity
// (progression-formulas §1.3): users against the opposing side's
// composite, a team or couple entity against the opposing entity when
// there is one, otherwise against the composite.
func rate(ctx context.Context, q *sqlc.Queries, p *matchCompleted, sides [2]side, season uuid.UUID) error {
	now := p.EndedAt
	users := map[uuid.UUID]domain.Rating{}
	var composite, ent [2]domain.Rating
	for i, sd := range sides {
		rs := make([]domain.Rating, 0, len(sd.users))
		for _, u := range sd.users {
			r, _, err := loadRating(ctx, q, "user", u.UserID, p.RatingBucket, now)
			if err != nil {
				return err
			}
			users[u.UserID] = r
			rs = append(rs, r)
		}
		composite[i] = domain.Composite(rs)
		if sd.entity != nil {
			r, ok, err := loadRating(ctx, q, sd.entity.kind, sd.entity.id, p.RatingBucket, now)
			if err != nil {
				return err
			}
			if !ok {
				r = domain.Rating{Value: composite[i].Value, RD: domain.NewEntityRD, Volatility: domain.InitialVolatility}
			}
			ent[i] = r
		}
	}
	write := func(e entity, before domain.Rating, opp domain.Rating, score float64) error {
		after := before.Update([]domain.Opponent{{Rating: opp, Score: score}})
		if err := q.UpsertRating(ctx, sqlc.UpsertRatingParams{
			EntityKind: e.kind, EntityID: e.id, Bucket: p.RatingBucket,
			Rating: round2(after.Value), Rd: round2(after.RD), Volatility: after.Volatility, Now: &now,
		}); err != nil {
			return err
		}
		return q.InsertRatingChange(ctx, sqlc.InsertRatingChangeParams{
			MatchID: p.MatchID, EntityKind: e.kind, EntityID: e.id, Bucket: p.RatingBucket, SeasonID: season,
			RatingBefore: round2(before.Value), RatingAfter: round2(after.Value), RdBefore: round2(before.RD),
			RdAfter: round2(after.RD), VolBefore: before.Volatility, VolAfter: after.Volatility, Now: now,
		})
	}
	for i, sd := range sides {
		score := sd.result.Score()
		for _, u := range sd.users {
			if err := write(entity{kind: "user", id: u.UserID}, users[u.UserID], composite[1-i], score); err != nil {
				return err
			}
		}
		if sd.entity != nil {
			opp := composite[1-i]
			if sides[1-i].entity != nil {
				opp = ent[1-i]
			}
			if err := write(*sd.entity, ent[i], opp, score); err != nil {
				return err
			}
		}
	}
	return nil
}

func round2(v float64) float64 { return float64(int64(v*100+0.5)) / 100 }

// standings applies Season Points (progression-formulas §2–3) to every
// user and persistent entity of a ranked match and returns the season
// board writes.
func standings(ctx context.Context, q *sqlc.Queries, p *matchCompleted, sides [2]side, season sqlc.RankingSeason) ([]boardInc, error) {
	var incs []boardInc
	apply := func(e entity, sd *side, protected, public bool, countries []string) error {
		st, err := q.GetStanding(ctx, sqlc.GetStandingParams{SeasonID: season.ID, EntityKind: e.kind, EntityID: e.id})
		if errors.Is(err, pgx.ErrNoRows) {
			start := domain.Place(0)
			st = sqlc.RankingSeasonStanding{LeagueCode: start.League, Division: int16(start.Division), //nolint:gosec // G115: 1–3
				PeakLeague: start.League, PeakDivision: int16(start.Division)} //nolint:gosec // G115: 1–3
		} else if err != nil {
			return err
		}
		streak := 0
		switch sd.result {
		case domain.Win:
			st.Wins++
			streak = int(st.CurrentWinStreak) + 1
		case domain.Draw:
			st.Draws++
		case domain.Loss:
			st.Losses++
		}
		sp := domain.ApplySP(int(st.SeasonPoints), domain.SPDelta(sd.result, streak, protected))
		pl := domain.Place(sp)
		peak := domain.LeagueIndex(st.PeakLeague)
		if now := domain.LeagueIndex(pl.League); now > peak || (now == peak && pl.Division < int(st.PeakDivision)) {
			st.PeakLeague, st.PeakDivision = pl.League, int16(pl.Division) //nolint:gosec // G115: 1–3
		}
		if err := q.UpsertStanding(ctx, sqlc.UpsertStandingParams{
			SeasonID: season.ID, EntityKind: e.kind, EntityID: e.id, SeasonPoints: int32(sp), //nolint:gosec // G115: SP is small
			LeagueCode: pl.League, Division: int16(pl.Division), PeakLeague: st.PeakLeague, PeakDivision: st.PeakDivision, //nolint:gosec // G115: 1–3
			Matches: st.Matches + 1, Wins: st.Wins, Losses: st.Losses, Draws: st.Draws,
			BestWinStreak: max(st.BestWinStreak, int32(streak)), CurrentWinStreak: int32(streak), Now: p.EndedAt,
		}); err != nil {
			return err
		}
		if public {
			for _, scope := range domain.Scopes(countries) {
				incs = append(incs, boardInc{
					key:    domain.BoardKey(domain.SeasonPeriodID(season.Number), e.kind, scope, "all"),
					member: e.id.String(), score: float64(sp), set: true,
				})
			}
		}
		return nil
	}
	for i := range sides {
		sd := &sides[i]
		for _, u := range sd.users {
			protected := sd.result == domain.Loss && slices.ContainsFunc(sd.users, func(o member) bool {
				return o.UserID != u.UserID && (o.Status == "left" || o.Status == "forfeited")
			})
			if err := apply(entity{kind: "user", id: u.UserID}, sd, protected, u.TierAtStart == "full", []string{u.country}); err != nil {
				return nil, err
			}
		}
		if sd.entity != nil {
			if err := apply(*sd.entity, sd, false, sd.public, sd.countries); err != nil {
				return nil, err
			}
		}
	}
	return incs, nil
}

// ledger writes leaderboard points (progression-formulas §4) of a
// matchmade match: an overall row and one row per played category for
// every user and persistent entity, and returns the board writes.
func ledger(ctx context.Context, q *sqlc.Queries, p *matchCompleted, sides [2]side, season uuid.UUID) ([]boardInc, error) {
	var cats []uuid.UUID
	inCat := map[uuid.UUID]int{}
	for _, r := range p.Rounds {
		if inCat[r.CategoryID] == 0 {
			cats = append(cats, r.CategoryID)
		}
		inCat[r.CategoryID]++
	}
	var incs []boardInc
	write := func(e entity, res domain.Result, public bool, countries []string, correct func(r *roundPayload) bool) error {
		total, byCat := 0, map[uuid.UUID]int{}
		for i := range p.Rounds {
			if correct(&p.Rounds[i]) {
				total++
				byCat[p.Rounds[i].CategoryID]++
			}
		}
		var country *string
		if len(countries) > 0 && countries[0] != "" {
			country = &countries[0]
		}
		row := func(cat *uuid.UUID, points int) error {
			n, err := q.InsertLedger(ctx, sqlc.InsertLedgerParams{
				ID: ids.New(), MatchID: p.MatchID, EntityKind: e.kind, EntityID: e.id, SeasonID: season, CategoryID: cat,
				Points: int32(points), CountryCode: country, Public: public, OccurredAt: p.EndedAt, //nolint:gosec // G115: points are small
			})
			if err != nil || n == 0 || !public {
				return err
			}
			incs = append(incs, ledgerIncs(e.kind, e.id, cat, points, countries, p.EndedAt, p.EndedAt)...)
			return nil
		}
		if err := row(nil, domain.LeaderboardPoints(res, total)); err != nil {
			return err
		}
		for _, c := range cats {
			if err := row(&c, domain.CategoryPoints(res, inCat[c], len(p.Rounds), byCat[c])); err != nil {
				return err
			}
		}
		return nil
	}
	for i := range sides {
		sd := &sides[i]
		for _, u := range sd.users {
			own := func(r *roundPayload) bool { return slices.Contains(r.CorrectUserIDs, u.UserID) }
			if err := write(entity{kind: "user", id: u.UserID}, sd.result, u.TierAtStart == "full", []string{u.country}, own); err != nil {
				return nil, err
			}
		}
		if sd.entity != nil {
			idx := sd.idx
			if err := write(*sd.entity, sd.result, sd.public, sd.countries, func(r *roundPayload) bool { return r.SideCorrect[idx] }); err != nil {
				return nil, err
			}
		}
	}
	return incs, nil
}

// dirtyBoards lists period boards touched since the last snapshot run
// (redis-keys §6.2).
const dirtyBoards = "rank:lbdirty"

// applyBoards writes leaderboard changes to Redis. A failure is logged:
// the ledger is committed and boards are rebuilt from it.
func (s *Service) applyBoards(ctx context.Context, incs []boardInc) {
	if len(incs) == 0 || s.d.Redis == nil {
		return
	}
	pipe := s.d.Redis.Pipeline()
	var dirty []any
	for _, b := range incs {
		if b.set {
			pipe.ZAdd(ctx, b.key, goredis.Z{Score: b.score, Member: b.member})
		} else {
			pipe.ZIncrBy(ctx, b.key, b.score, b.member)
		}
		if !b.expireAt.IsZero() {
			pipe.ExpireAt(ctx, b.key, b.expireAt)
			dirty = append(dirty, b.key)
		}
	}
	if len(dirty) > 0 {
		pipe.SAdd(ctx, dirtyBoards, dirty...)
	}
	if _, err := pipe.Exec(ctx); err != nil && s.d.Logger != nil {
		s.d.Logger.WarnContext(ctx, "ranking: leaderboard update failed", "error", err)
	}
}
