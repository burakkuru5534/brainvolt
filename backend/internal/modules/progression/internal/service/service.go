// Package service holds progression's use cases: XP and levels, play
// streaks, badges and quests (schema-ranking.md §5, progression-formulas
// §5–8).
package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/progression/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/progression/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
)

// Pusher sends a personal WebSocket message (realtime, wired in app).
type Pusher interface {
	Push(ctx context.Context, user uuid.UUID, typ string, data any) error
}

// Deps are the service's collaborators.
type Deps struct {
	DB     *pgxpool.Pool
	Pusher Pusher
	Clock  clock.Clock
	Logger *slog.Logger
}

// Service implements progression.
type Service struct {
	d Deps
}

// New builds the service.
func New(d Deps) *Service { return &Service{d: d} }

// XP sources (xp_ledger.source).
const (
	sourceMatch  = "match"
	sourceQuest  = "quest"
	sourceBadge  = "badge"
	sourceSeason = "season_reward"
)

// grant writes an XP ledger row once per (user, source, ref) and moves
// the user's total and level; it returns the XP added (0 when the row
// existed) and the level before and after.
func grant(ctx context.Context, q *sqlc.Queries, user uuid.UUID, amount int, source, ref string, now time.Time) (added, before, after int, err error) {
	p, err := q.GetProgress(ctx, user)
	if errors.Is(err, pgx.ErrNoRows) {
		p = sqlc.ProgressionUserProgress{Level: 1}
	} else if err != nil {
		return 0, 0, 0, err
	}
	before = int(p.Level)
	if amount <= 0 {
		return 0, before, before, nil
	}
	n, err := q.InsertXP(ctx, sqlc.InsertXPParams{
		ID: ids.New(), UserID: user, Amount: int32(amount), Source: source, SourceRef: ref, Now: now, //nolint:gosec // G115: XP amounts are small
	})
	if err != nil || n == 0 {
		return 0, before, before, err
	}
	total := p.XpTotal + int64(amount)
	after = domain.Level(total)
	if err := q.UpsertProgress(ctx, sqlc.UpsertProgressParams{UserID: user, XpTotal: total, Level: int32(after), Now: now}); err != nil { //nolint:gosec // G115: ≤ 100
		return 0, 0, 0, err
	}
	return amount, before, after, nil
}

// Progress is GET /v1/me/progress.
type Progress struct {
	XPTotal int64 `json:"xp_total"`
	Level   int   `json:"level"`
	// LevelXP is the XP earned inside the current level, LevelSize the
	// XP the level takes (0 at the cap).
	LevelXP   int64  `json:"level_xp"`
	LevelSize int64  `json:"level_size"`
	Streak    Streak `json:"streak"`
	Badges    int    `json:"badges"`
	// Matches and Wins count every completed match (Home stats).
	Matches int `json:"matches"`
	Wins    int `json:"wins"`
}

// Streak is the play streak as shown today.
type Streak struct {
	Current int `json:"current"`
	Longest int `json:"longest"`
	// PlayedToday says whether today already counts.
	PlayedToday bool `json:"played_today"`
}

// Progress returns a user's XP, level, play streak and badge count.
func (s *Service) Progress(ctx context.Context, user uuid.UUID) (Progress, error) {
	q := sqlc.New(s.d.DB)
	var out Progress
	p, err := q.GetProgress(ctx, user)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		out.Level = 1
	case err != nil:
		return Progress{}, err
	default:
		out.XPTotal, out.Level = p.XpTotal, int(p.Level)
	}
	out.LevelXP = out.XPTotal - domain.LevelStart(out.Level)
	if out.Level < domain.MaxLevel {
		out.LevelSize = domain.LevelStart(out.Level+1) - domain.LevelStart(out.Level)
	}
	stats, err := q.GetStats(ctx, user)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
	case err != nil:
		return Progress{}, err
	default:
		out.Matches, out.Wins = int(stats.Matches), int(stats.Wins)
	}
	st, err := loadStreak(ctx, q, user)
	if err != nil {
		return Progress{}, err
	}
	today := domain.Day(s.d.Clock.Now())
	out.Streak = Streak{Current: st.Live(today), Longest: st.Longest, PlayedToday: !st.LastDay.IsZero() && st.LastDay.Equal(today)}
	badges, err := q.BadgeCatalog(ctx, sqlc.BadgeCatalogParams{Locale: "en", UserID: user})
	if err != nil {
		return Progress{}, err
	}
	for _, b := range badges {
		if b.EarnedAt != nil {
			out.Badges++
		}
	}
	return out, nil
}

func loadStreak(ctx context.Context, q *sqlc.Queries, user uuid.UUID) (domain.Streak, error) {
	row, err := q.GetStreak(ctx, user)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Streak{}, nil
	}
	if err != nil {
		return domain.Streak{}, err
	}
	st := domain.Streak{Current: int(row.CurrentDays), Longest: int(row.LongestDays)}
	if row.LastPlayDay != nil {
		d := row.LastPlayDay
		st.LastDay = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, domain.Istanbul)
	}
	return st, nil
}
