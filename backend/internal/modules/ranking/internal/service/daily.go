package service

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/ranking/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
)

// dailyRunCompleted is the part of game.DailyRunCompleted ranking reads.
type dailyRunCompleted struct {
	UserID       uuid.UUID `json:"user_id"`
	Day          string    `json:"day"`
	CorrectCount int       `json:"correct_count"`
	TotalTimeMS  int       `json:"total_time_ms"`
	CompletedAt  time.Time `json:"completed_at"`
	TierAtStart  string    `json:"tier_at_start"`
}

// OnDailyRunCompleted adds an official run to its day's board once
// (category-modes §5.2); restricted accounts stay off public boards
// until they upgrade (D-016).
func (s *Service) OnDailyRunCompleted(ctx context.Context, e events.Envelope) error {
	var p dailyRunCompleted
	if err := e.Decode(&p); err != nil {
		return err
	}
	day, err := time.Parse(time.DateOnly, p.Day)
	if err != nil {
		return err
	}
	var country *string
	if s.d.Cards != nil {
		cards, err := s.d.Cards.Cards(ctx, []uuid.UUID{p.UserID})
		if err != nil {
			return err
		}
		country = cards[p.UserID].CountryCode
	}
	return sqlc.New(s.d.DB).InsertDailyEntry(ctx, sqlc.InsertDailyEntryParams{
		Day: day, UserID: p.UserID, CorrectCount: int16(p.CorrectCount), TotalTimeMs: int32(p.TotalTimeMS), //nolint:gosec // G115: ≤ 10 / ≤ 150 000
		CompletedAt: p.CompletedAt, CountryCode: country, Public: p.TierAtStart == "full",
	})
}

// DailyEntry is one row of the Daily Challenge board.
type DailyEntry struct {
	Rank        int           `json:"rank"`
	User        *profile.Card `json:"user"`
	UserID      uuid.UUID     `json:"user_id"`
	Correct     int           `json:"correct_count"`
	TotalTimeMS int           `json:"total_time_ms"`
	CompletedAt time.Time     `json:"completed_at"`
}

// DailyBoard is GET /v1/leaderboards/daily-challenge/{date}.
type DailyBoard struct {
	Day        string       `json:"day"`
	Scope      string       `json:"scope"`
	Country    *string      `json:"country,omitempty"`
	Entries    []DailyEntry `json:"entries"`
	Me         *DailyEntry  `json:"me"`
	NextCursor *string      `json:"next_cursor"`
}

// DailyBoard returns a page of a day's board: global, the caller's
// country or the caller and their friends (D-040).
func (s *Service) DailyBoard(ctx context.Context, viewer uuid.UUID, day time.Time, scope string, limit, offset int) (DailyBoard, error) {
	out := DailyBoard{Day: day.Format(time.DateOnly), Scope: scope, Entries: []DailyEntry{}}
	var country *string
	users := []uuid.UUID{}
	switch scope {
	case "global":
	case "country":
		cards, err := s.d.Cards.Cards(ctx, []uuid.UUID{viewer})
		if err != nil {
			return DailyBoard{}, err
		}
		if country = cards[viewer].CountryCode; country == nil {
			return DailyBoard{}, invalid("country", "required")
		}
		out.Country = country
	case "friends":
		friends, err := s.d.Friends.FriendIDs(ctx, viewer)
		if err != nil {
			return DailyBoard{}, err
		}
		users = append([]uuid.UUID{viewer}, friends[:min(len(friends), maxFriends)]...)
	default:
		return DailyBoard{}, invalid("scope", "one_of")
	}
	q := sqlc.New(s.d.DB)
	rows, err := q.DailyBoard(ctx, sqlc.DailyBoardParams{Day: day, Country: country, Users: users, Lim: int32(limit + 1), Off: int32(offset)}) //nolint:gosec // G115: ≤ 101 / ≤ 10 000
	if err != nil {
		return DailyBoard{}, err
	}
	if len(rows) > limit {
		rows = rows[:limit]
		next := offset + limit
		cursor := strconv.Itoa(next)
		out.NextCursor = &cursor
	}
	for i, r := range rows {
		out.Entries = append(out.Entries, DailyEntry{Rank: offset + i + 1, UserID: r.UserID, Correct: int(r.CorrectCount),
			TotalTimeMS: int(r.TotalTimeMs), CompletedAt: r.CompletedAt})
	}
	mine, err := q.DailyEntry(ctx, sqlc.DailyEntryParams{Day: day, UserID: viewer})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
	case err != nil:
		return DailyBoard{}, err
	case mine.Public:
		rank, err := q.DailyRank(ctx, sqlc.DailyRankParams{Day: day, Country: country, Users: users, CorrectCount: mine.CorrectCount,
			TotalTimeMs: mine.TotalTimeMs, CompletedAt: mine.CompletedAt})
		if err != nil {
			return DailyBoard{}, err
		}
		out.Me = &DailyEntry{Rank: int(rank), UserID: viewer, Correct: int(mine.CorrectCount), TotalTimeMS: int(mine.TotalTimeMs), CompletedAt: mine.CompletedAt}
	}
	ids := make([]uuid.UUID, 0, len(out.Entries)+1)
	for _, e := range out.Entries {
		ids = append(ids, e.UserID)
	}
	ids = append(ids, viewer)
	cards, err := s.d.Cards.Cards(ctx, ids)
	if err != nil {
		return DailyBoard{}, err
	}
	for i := range out.Entries {
		if c, ok := cards[out.Entries[i].UserID]; ok {
			out.Entries[i].User = &c
		}
	}
	if out.Me != nil {
		if c, ok := cards[viewer]; ok {
			out.Me.User = &c
		}
	}
	return out, nil
}
