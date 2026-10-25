// Package service holds progression's use cases: XP and levels, play
// streaks, badges and quests (schema-ranking.md §5, progression-formulas
// §5–8).
package service

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
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
	// Plus tells whether a user holds BrainVolt Plus (commerce, D-104);
	// nil = nobody does.
	Plus PlusChecker
	// Wallet credits Volt in progression's transactions (commerce, D-105);
	// nil = no Volt.
	Wallet Wallet
	// Season is the current ranked season (the Season Pass, D-108).
	Season SeasonSource
	Clock  clock.Clock
	Logger *slog.Logger
}

// Wallet writes Volt and cosmetics to commerce inside tx, once per key.
type Wallet interface {
	CreditTx(ctx context.Context, tx pgx.Tx, user uuid.UUID, amount int, reason, ref, key string, now time.Time) error
	// DebitTx refuses with VOLT_INSUFFICIENT when the balance is short.
	DebitTx(ctx context.Context, tx pgx.Tx, user uuid.UUID, amount int, reason, ref, key string, now time.Time) error
	GrantCosmeticTx(ctx context.Context, tx pgx.Tx, user uuid.UUID, code string, now time.Time) error
}

// Volt rewards (D-105): quests by cadence, the Daily Challenge, each level
// reached, tournament places 1–3 and a season's peak league.
const (
	voltQuestDaily  = 10
	voltQuestWeekly = 50
	voltDaily       = 10
	voltLevel       = 25
)

var (
	voltTournament = map[int]int{1: 300, 2: 150, 3: 75}
	voltSeason     = map[string]int{"bronze": 50, "silver": 100, "gold": 200, "platinum": 300, "diamond": 500}
)

// Volt ledger reasons (commerce's).
const (
	reasonQuest      = "quest"
	reasonDaily      = "daily_challenge"
	reasonLevelUp    = "level_up"
	reasonTournament = "tournament"
	reasonSeason     = "season"
)

// volt credits Volt to user inside tx once per key.
func (s *Service) volt(ctx context.Context, tx pgx.Tx, user uuid.UUID, amount int, reason, ref, key string, now time.Time) error {
	if s.d.Wallet == nil || amount <= 0 {
		return nil
	}
	return s.d.Wallet.CreditTx(ctx, tx, user, amount, reason, ref, key, now)
}

func questVolt(cadence string) int {
	if cadence == "weekly" {
		return voltQuestWeekly
	}
	return voltQuestDaily
}

// PlusChecker reports entitlements (commerce's reader).
type PlusChecker interface {
	HasEntitlement(ctx context.Context, user uuid.UUID, codes ...string) (bool, error)
}

// plus reports whether user holds Plus; a failed lookup counts as no.
func (s *Service) plus(ctx context.Context, user uuid.UUID) bool {
	if s.d.Plus == nil {
		return false
	}
	ok, err := s.d.Plus.HasEntitlement(ctx, user, "plus")
	if err != nil && s.d.Logger != nil {
		s.d.Logger.WarnContext(ctx, "progression: plus lookup failed", "user", user, "error", err)
	}
	return ok && err == nil
}

// extraQuests is the extra daily quest of Plus (D-104).
func (s *Service) extraQuests(ctx context.Context, user uuid.UUID) int {
	if s.plus(ctx, user) {
		return 1
	}
	return 0
}

// freeRerolls is the day's free quest rerolls: one, two with Plus (D-104).
func (s *Service) freeRerolls(ctx context.Context, user uuid.UUID) int {
	return freeRerolls + s.extraQuests(ctx, user)
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
// existed) and the level before and after. Each level reached gives
// voltLevel Volt (D-105).
func (s *Service) grant(ctx context.Context, tx pgx.Tx, q *sqlc.Queries, user uuid.UUID, amount int, source, ref string, now time.Time) (added, before, after int, err error) {
	added, before, after, err = grant(ctx, q, user, amount, source, ref, now)
	for l := before + 1; err == nil && l <= after; l++ {
		lv := strconv.Itoa(l)
		err = s.volt(ctx, tx, user, voltLevel, reasonLevelUp, lv, "level:"+lv, now)
	}
	return added, before, after, err
}

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
	// Shields cover missed days (D-105); at most MaxShields.
	Shields    int `json:"shields"`
	MaxShields int `json:"max_shields"`
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
	now := s.d.Clock.Now()
	if err := s.plusShield(ctx, q, user, now); err != nil {
		return Progress{}, err
	}
	st, err := loadStreak(ctx, q, user)
	if err != nil {
		return Progress{}, err
	}
	today := domain.Day(now)
	out.Streak = Streak{Current: st.Live(today), Longest: st.Longest, PlayedToday: !st.LastDay.IsZero() && st.LastDay.Equal(today),
		Shields: st.Shields, MaxShields: domain.MaxShields}
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
	st := domain.Streak{Current: int(row.CurrentDays), Longest: int(row.LongestDays), Shields: int(row.Shields)}
	if row.LastPlayDay != nil {
		d := row.LastPlayDay
		st.LastDay = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, domain.Istanbul)
	}
	return st, nil
}

// plusShield gives a Plus subscriber the week's streak shield (D-105).
func (s *Service) plusShield(ctx context.Context, q *sqlc.Queries, user uuid.UUID, now time.Time) error {
	if !s.plus(ctx, user) {
		return nil
	}
	return q.PlusShield(ctx, sqlc.PlusShieldParams{UserID: user, Week: ptr(domain.WeekStart(now)), Now: now})
}

// saveStreak stores a streak after play.
func saveStreak(ctx context.Context, q *sqlc.Queries, user uuid.UUID, st domain.Streak, now time.Time) error {
	day := st.LastDay
	return q.UpsertStreak(ctx, sqlc.UpsertStreakParams{
		UserID: user, CurrentDays: int32(st.Current), LongestDays: int32(st.Longest), LastPlayDay: &day, Shields: int32(st.Shields), Now: now, //nolint:gosec // G115: days
	})
}
