package service

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/ranking/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/ranking/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/team"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/redis"
)

// Friends lists a user's friends (social; friends boards, D-027).
type Friends interface {
	FriendIDs(ctx context.Context, user uuid.UUID) ([]uuid.UUID, error)
}

// Board filters (schema-ranking §4.1).
var (
	BoardPeriods  = []string{"daily", "weekly", "monthly", "all_time", "season"}
	BoardEntities = []string{"user", "team", "couple"}
	BoardScopes   = []string{"global", "country", "friends"}
)

// maxFriends caps a friends board (D-027).
const maxFriends = 300

// BoardQuery selects a leaderboard page.
type BoardQuery struct {
	Viewer     uuid.UUID
	Period     string
	Entity     string
	Scope      string
	Country    string
	CategoryID *uuid.UUID
	SeasonID   *uuid.UUID
	Limit      int
	Offset     int
}

// Board is a leaderboard page.
type Board struct {
	Period     string     `json:"period"`
	PeriodID   string     `json:"period_id"`
	Entity     string     `json:"entity"`
	Scope      string     `json:"scope"`
	Country    *string    `json:"country,omitempty"`
	CategoryID *uuid.UUID `json:"category_id,omitempty"`
	StartsAt   *time.Time `json:"starts_at,omitempty"`
	EndsAt     *time.Time `json:"ends_at,omitempty"`
	Entries    []Entry    `json:"entries"`
	// Me is the caller's own row on user boards (null when not ranked).
	Me         *Entry  `json:"me"`
	NextCursor *string `json:"next_cursor"`
}

// Entry is one ranked entity: user boards carry the user's card, team
// boards the team's label, couple boards the couple's name and partners.
type Entry struct {
	Rank     int           `json:"rank"`
	EntityID uuid.UUID     `json:"entity_id"`
	Score    int64         `json:"score"`
	User     *profile.Card `json:"user,omitempty"`
	Team     *team.Label   `json:"team,omitempty"`
	Couple   *CoupleLabel  `json:"couple,omitempty"`
}

// CoupleLabel is a couple on a board.
type CoupleLabel struct {
	Name     string         `json:"name,omitempty"`
	Partners []profile.Card `json:"partners"`
}

func invalid(field, rule string) error {
	return apierror.Validation(apierror.FieldError{Field: field, Rule: rule})
}

// Leaderboard returns a page of a board (schema-ranking §4.2): Redis
// sorted sets for global and country scopes, the caller's friends'
// scores on the global board for the friends scope.
func (s *Service) Leaderboard(ctx context.Context, q BoardQuery) (Board, error) {
	switch {
	case !slices.Contains(BoardPeriods, q.Period):
		return Board{}, invalid("period", "one_of")
	case !slices.Contains(BoardEntities, q.Entity):
		return Board{}, invalid("entity", "one_of")
	case !slices.Contains(BoardScopes, q.Scope):
		return Board{}, invalid("scope", "one_of")
	case q.Scope == "friends" && q.Entity != "user":
		return Board{}, invalid("scope", "friends_user_only")
	case q.Period == "season" && q.CategoryID != nil:
		return Board{}, invalid("category_id", "not_with_season")
	}
	b := Board{Period: q.Period, Entity: q.Entity, Scope: q.Scope, CategoryID: q.CategoryID, Entries: []Entry{}}
	if q.Period == "season" {
		se, err := s.seasonOrCurrent(ctx, q.SeasonID)
		if err != nil {
			return Board{}, err
		}
		b.PeriodID, b.StartsAt, b.EndsAt = domain.SeasonPeriodID(se.Number), &se.StartsAt, &se.EndsAt
	} else {
		ps := domain.Periods(s.d.Clock.Now())
		p := ps[slices.Index(BoardPeriods, q.Period)]
		b.PeriodID = p.ID
		if !p.Start.IsZero() {
			b.StartsAt, b.EndsAt = &p.Start, &p.End
		}
	}
	scope := "global"
	if q.Scope == "country" {
		country := q.Country
		if country == "" {
			cards, err := s.d.Cards.Cards(ctx, []uuid.UUID{q.Viewer})
			if err != nil {
				return Board{}, err
			}
			if c := cards[q.Viewer].CountryCode; c != nil {
				country = *c
			}
		}
		if len(country) != 2 {
			return Board{}, invalid("country", "required")
		}
		scope, b.Country = "country:"+country, &country
	}
	category := "all"
	if q.CategoryID != nil {
		category = "cat:" + q.CategoryID.String()
	}
	key := domain.BoardKey(b.PeriodID, q.Entity, scope, category)

	var rows []Entry
	var more bool
	if q.Scope == "friends" {
		all, err := s.friendRows(ctx, key, q.Viewer)
		if err != nil {
			return Board{}, err
		}
		for _, r := range all {
			if r.EntityID == q.Viewer {
				me := r
				b.Me = &me
			}
		}
		end := min(q.Offset+q.Limit, len(all))
		if q.Offset < end {
			rows = all[q.Offset:end]
		}
		more = end < len(all)
	} else {
		zs, err := s.d.Redis.ZRevRangeWithScores(ctx, key, int64(q.Offset), int64(q.Offset+q.Limit)).Result()
		if err != nil {
			return Board{}, err
		}
		more = len(zs) > q.Limit
		for i, z := range zs[:min(len(zs), q.Limit)] {
			id, _ := uuid.Parse(z.Member.(string)) //nolint:forcetypeassert // members are entity ids
			rows = append(rows, Entry{Rank: q.Offset + i + 1, EntityID: id, Score: int64(z.Score)})
		}
		if q.Entity == "user" {
			me, err := s.rankOf(ctx, key, q.Viewer)
			if err != nil {
				return Board{}, err
			}
			b.Me = me
		}
	}
	if more {
		next := strconv.Itoa(q.Offset + q.Limit)
		b.NextCursor = &next
	}
	all := rows
	if b.Me != nil {
		all = append(slices.Clone(rows), *b.Me)
	}
	if err := s.label(ctx, q.Entity, all); err != nil {
		return Board{}, err
	}
	if b.Me != nil {
		*b.Me = all[len(all)-1]
	}
	b.Entries = append(b.Entries, all[:len(rows)]...)
	return b, nil
}

// rankOf is an entity's row on a board (nil when absent).
func (s *Service) rankOf(ctx context.Context, key string, id uuid.UUID) (*Entry, error) {
	rank, err := s.d.Redis.ZRevRank(ctx, key, id.String()).Result()
	if errors.Is(err, redis.ErrNil) {
		return nil, nil //nolint:nilnil // not ranked
	}
	if err != nil {
		return nil, err
	}
	score, err := s.d.Redis.ZScore(ctx, key, id.String()).Result()
	if err != nil {
		return nil, err
	}
	return &Entry{Rank: int(rank) + 1, EntityID: id, Score: int64(score)}, nil
}

// friendRows ranks the viewer and up to 300 friends by their global
// scores (redis-keys §6.3).
func (s *Service) friendRows(ctx context.Context, key string, viewer uuid.UUID) ([]Entry, error) {
	friends, err := s.d.Friends.FriendIDs(ctx, viewer)
	if err != nil {
		return nil, err
	}
	ids := append([]uuid.UUID{viewer}, friends[:min(len(friends), maxFriends)]...)
	members := make([]string, len(ids))
	for i, id := range ids {
		members[i] = id.String()
	}
	scores, err := s.d.Redis.ZMScore(ctx, key, members...).Result()
	if err != nil {
		return nil, err
	}
	out := []Entry{}
	for i, sc := range scores {
		// go-redis reports a missing member as 0; keep only real members.
		if sc == 0 {
			if _, err := s.d.Redis.ZScore(ctx, key, members[i]).Result(); errors.Is(err, redis.ErrNil) {
				continue
			} else if err != nil {
				return nil, err
			}
		}
		out = append(out, Entry{EntityID: ids[i], Score: int64(sc)})
	}
	slices.SortStableFunc(out, func(a, b Entry) int {
		if a.Score != b.Score {
			if a.Score > b.Score {
				return -1
			}
			return 1
		}
		return slices.Compare(a.EntityID[:], b.EntityID[:])
	})
	for i := range out {
		out[i].Rank = i + 1
	}
	return out, nil
}

// label adds cards, team labels or couple labels to rows.
func (s *Service) label(ctx context.Context, entity string, rows []Entry) error {
	if len(rows) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(rows))
	for i, r := range rows {
		ids[i] = r.EntityID
	}
	switch entity {
	case "user":
		cards, err := s.d.Cards.Cards(ctx, ids)
		if err != nil {
			return err
		}
		for i := range rows {
			if c, ok := cards[rows[i].EntityID]; ok {
				rows[i].User = &c
			}
		}
	case "team":
		labels, err := s.d.Teams.TeamLabels(ctx, ids)
		if err != nil {
			return err
		}
		for i := range rows {
			if l, ok := labels[rows[i].EntityID]; ok {
				rows[i].Team = &l
			}
		}
	case "couple":
		labels, err := s.d.Teams.CoupleLabels(ctx, ids)
		if err != nil {
			return err
		}
		var users []uuid.UUID
		for _, l := range labels {
			users = append(users, l.Users...)
		}
		cards, err := s.d.Cards.Cards(ctx, users)
		if err != nil {
			return err
		}
		for i := range rows {
			l, ok := labels[rows[i].EntityID]
			if !ok {
				continue
			}
			c := &CoupleLabel{Name: l.Name, Partners: []profile.Card{}}
			for _, u := range l.Users {
				if card, ok := cards[u]; ok {
					c.Partners = append(c.Partners, card)
				}
			}
			rows[i].Couple = c
		}
	}
	return nil
}

func (s *Service) seasonOrCurrent(ctx context.Context, id *uuid.UUID) (sqlc.RankingSeason, error) {
	if id == nil {
		return s.ensureSeason(ctx)
	}
	se, err := sqlc.New(s.d.DB).GetSeason(ctx, *id)
	if errors.Is(err, pgx.ErrNoRows) {
		return se, apierror.New(apierror.NotFound)
	}
	return se, err
}

// SeasonByID returns a season with its names.
func (s *Service) SeasonByID(ctx context.Context, id uuid.UUID) (Season, error) {
	se, err := s.seasonOrCurrent(ctx, &id)
	if err != nil {
		return Season{}, err
	}
	return s.season(ctx, se)
}

// Standing is an entity's season standing (rest.md §9).
type Standing struct {
	SeasonID         uuid.UUID `json:"season_id"`
	EntityKind       string    `json:"entity_kind"`
	EntityID         uuid.UUID `json:"entity_id"`
	SeasonPoints     int       `json:"season_points"`
	League           string    `json:"league"`
	Division         int       `json:"division"`
	PeakLeague       string    `json:"peak_league"`
	PeakDivision     int       `json:"peak_division"`
	Matches          int       `json:"matches"`
	Wins             int       `json:"wins"`
	Losses           int       `json:"losses"`
	Draws            int       `json:"draws"`
	BestWinStreak    int       `json:"best_win_streak"`
	CurrentWinStreak int       `json:"current_win_streak"`
	// NextAt is the SP of the next division (null in Diamond).
	NextAt *int `json:"next_at"`
	// Rank is the place on the global season board (null when unranked).
	Rank      *int `json:"rank"`
	FinalRank *int `json:"final_rank"`
}

// Standing returns an entity's standing in a season (default the
// current one); an entity without matches stands at 0 SP in Bronze III.
func (s *Service) Standing(ctx context.Context, kind string, id uuid.UUID, season *uuid.UUID) (Standing, error) {
	if !slices.Contains(BoardEntities, kind) {
		return Standing{}, apierror.New(apierror.NotFound)
	}
	se, err := s.seasonOrCurrent(ctx, season)
	if err != nil {
		return Standing{}, err
	}
	out := Standing{SeasonID: se.ID, EntityKind: kind, EntityID: id}
	row, err := sqlc.New(s.d.DB).GetStanding(ctx, sqlc.GetStandingParams{SeasonID: se.ID, EntityKind: kind, EntityID: id})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		start := domain.Place(0)
		out.League, out.Division, out.PeakLeague, out.PeakDivision = start.League, start.Division, start.League, start.Division
	case err != nil:
		return Standing{}, err
	default:
		out.SeasonPoints, out.League, out.Division = int(row.SeasonPoints), row.LeagueCode, int(row.Division)
		out.PeakLeague, out.PeakDivision, out.Matches = row.PeakLeague, int(row.PeakDivision), int(row.Matches)
		out.Wins, out.Losses, out.Draws = int(row.Wins), int(row.Losses), int(row.Draws)
		out.BestWinStreak, out.CurrentWinStreak = int(row.BestWinStreak), int(row.CurrentWinStreak)
		if row.FinalRankGlobal != nil {
			r := int(*row.FinalRankGlobal)
			out.FinalRank = &r
		}
	}
	if next, ok := domain.NextThreshold(out.SeasonPoints); ok {
		out.NextAt = &next
	}
	me, err := s.rankOf(ctx, domain.BoardKey(domain.SeasonPeriodID(se.Number), kind, "global", "all"), id)
	if err != nil {
		return Standing{}, err
	}
	if me != nil {
		out.Rank = &me.Rank
	}
	return out, nil
}

// League is a league with its name and division floors (III … I).
type League struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
	Floors    []int  `json:"floors"`
}

// Leagues returns the league table in a locale.
func (s *Service) Leagues(ctx context.Context, locale string) ([]League, error) {
	rows, err := sqlc.New(s.d.DB).Leagues(ctx, locale)
	if err != nil {
		return nil, err
	}
	out := make([]League, 0, len(rows))
	for _, r := range rows {
		l := League{Code: r.Code, Name: r.Name, SortOrder: int(r.SortOrder)}
		for _, d := range domain.Leagues {
			if d.Code == r.Code {
				l.Floors = d.Divisions
			}
		}
		out = append(out, l)
	}
	return out, nil
}
