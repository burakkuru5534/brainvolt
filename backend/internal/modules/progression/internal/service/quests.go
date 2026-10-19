package service

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/progression/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/progression/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
)

// Quest rules (D-029, progression-formulas §8).
const (
	questsPerPeriod = 3
	freeRerolls     = 1
)

// cadences are the quest periods assigned to every user.
var cadences = []string{"daily", "weekly"}

// Quest is an assigned quest.
type Quest struct {
	ID          uuid.UUID  `json:"id"`
	Code        string     `json:"code"`
	Cadence     string     `json:"cadence"`
	Objective   string     `json:"objective"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Progress    int        `json:"progress"`
	Target      int        `json:"target"`
	RewardXP    int        `json:"reward_xp"`
	EndsAt      time.Time  `json:"ends_at"`
	CompletedAt *time.Time `json:"completed_at"`
	ClaimedAt   *time.Time `json:"claimed_at"`
}

// Quests is GET /v1/me/quests.
type Quests struct {
	Daily       []Quest `json:"daily"`
	Weekly      []Quest `json:"weekly"`
	RerollsLeft int     `json:"rerolls_left"`
}

// Claimed is the answer to a claim.
type Claimed struct {
	XP      int  `json:"xp"`
	Level   int  `json:"level"`
	LevelUp bool `json:"level_up"`
}

// filters are a quest's eligibility filters.
type filters struct {
	MatchType string `json:"match_type,omitempty"`
}

// Quests assigns the day's and week's quests when missing and returns
// them in a locale.
func (s *Service) Quests(ctx context.Context, user uuid.UUID, locale string) (Quests, error) {
	now := s.d.Clock.Now()
	out := Quests{Daily: []Quest{}, Weekly: []Quest{}}
	err := postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		if err := q.LockUser(ctx, user.String()); err != nil {
			return err
		}
		texts, err := q.QuestTexts(ctx, locale)
		if err != nil {
			return err
		}
		for _, c := range cadences {
			rows, err := ensureQuests(ctx, q, user, c, now)
			if err != nil {
				return err
			}
			for i := range rows {
				qu := questOf(&rows[i])
				if t := slices.IndexFunc(texts, func(t sqlc.QuestTextsRow) bool { return t.QuestID == rows[i].QuestID }); t >= 0 {
					qu.Title, qu.Description = texts[t].Title, texts[t].Description
				}
				if c == "daily" {
					out.Daily = append(out.Daily, qu)
				} else {
					out.Weekly = append(out.Weekly, qu)
				}
			}
		}
		used, err := q.RerollCount(ctx, sqlc.RerollCountParams{UserID: user, Day: domain.Day(now)})
		out.RerollsLeft = max(freeRerolls-int(used), 0)
		return err
	})
	return out, err
}

func questOf(r *sqlc.PeriodAssignmentsRow) Quest {
	return Quest{ID: r.ID, Code: r.Code, Cadence: r.Cadence, Objective: r.Objective, Progress: int(r.Progress), Target: int(r.Target),
		RewardXP: int(r.RewardXp), EndsAt: r.PeriodEnd, CompletedAt: r.CompletedAt, ClaimedAt: r.ClaimedAt}
}

// ensureQuests tops a period up to three quests with different
// objectives where the catalogue allows. The caller holds LockUser.
func ensureQuests(ctx context.Context, q *sqlc.Queries, user uuid.UUID, cadence string, t time.Time) ([]sqlc.PeriodAssignmentsRow, error) {
	start, end := domain.Period(cadence, t)
	rows, err := q.PeriodAssignments(ctx, sqlc.PeriodAssignmentsParams{UserID: user, Cadence: cadence, PeriodStart: start})
	if err != nil || len(rows) >= questsPerPeriod {
		return rows, err
	}
	pool, err := q.ActiveQuests(ctx, cadence)
	if err != nil {
		return nil, err
	}
	picked := pick(pool, rows, questsPerPeriod-len(rows), uuid.Nil)
	for _, p := range picked {
		if err := q.InsertAssignment(ctx, sqlc.InsertAssignmentParams{ID: ids.New(), UserID: user, QuestID: p.ID, PeriodStart: start, PeriodEnd: end}); err != nil {
			return nil, err
		}
	}
	if len(picked) == 0 {
		return rows, nil
	}
	return q.PeriodAssignments(ctx, sqlc.PeriodAssignmentsParams{UserID: user, Cadence: cadence, PeriodStart: start})
}

// pick draws n quests not yet assigned, new objectives first; skip is
// left out (the quest being rerolled).
func pick(pool []sqlc.ActiveQuestsRow, assigned []sqlc.PeriodAssignmentsRow, n int, skip uuid.UUID) []sqlc.ActiveQuestsRow {
	pool = slices.Clone(pool)
	rand.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] }) //nolint:gosec // G404: quest draw, not security relevant
	taken := map[uuid.UUID]bool{skip: true}
	objectives := map[string]bool{}
	for i := range assigned {
		taken[assigned[i].QuestID] = true
		if assigned[i].QuestID != skip {
			objectives[assigned[i].Objective] = true
		}
	}
	var out []sqlc.ActiveQuestsRow
	for _, distinct := range []bool{true, false} {
		for _, p := range pool {
			if len(out) == n {
				return out
			}
			if taken[p.ID] || (distinct && objectives[p.Objective]) {
				continue
			}
			taken[p.ID], objectives[p.Objective] = true, true
			out = append(out, p)
		}
	}
	return out
}

// QuestProgress is a quest moved by a match (match.awards).
type QuestProgress struct {
	ID        uuid.UUID `json:"id"`
	Code      string    `json:"code"`
	Progress  int       `json:"progress"`
	Target    int       `json:"target"`
	Completed bool      `json:"completed"`
}

// advanceQuests applies a player's match to their open quests of the
// periods holding the match end.
func advanceQuests(ctx context.Context, q *sqlc.Queries, user uuid.UUID, p *matchCompleted, me *participant, won bool) ([]QuestProgress, error) {
	var out []QuestProgress
	for _, c := range cadences {
		rows, err := ensureQuests(ctx, q, user, c, p.EndedAt)
		if err != nil {
			return nil, err
		}
		for i := range rows {
			r := &rows[i]
			if r.CompletedAt != nil {
				continue
			}
			var f filters
			if err := json.Unmarshal(r.Filters, &f); err != nil {
				return nil, err
			}
			progress := int(r.Progress)
			switch r.Objective {
			case "play_matches":
				progress++
			case "win_matches":
				if won {
					progress++
				}
			case "correct_answers":
				progress += me.Correct
			case "play_mode":
				if f.MatchType == "" || f.MatchType == p.MatchType {
					progress++
				}
			case "play_team_match":
				if p.Format != "1v1" && p.Format != "couple" {
					progress++
				}
			case "win_streak":
				if won {
					progress++
				} else {
					progress = 0
				}
			}
			if progress == int(r.Progress) {
				continue
			}
			progress = min(progress, int(r.Target))
			var done *time.Time
			if progress >= int(r.Target) {
				done = &p.EndedAt
			}
			if err := q.SetAssignmentProgress(ctx, sqlc.SetAssignmentProgressParams{Progress: int32(progress), CompletedAt: done, ID: r.ID}); err != nil { //nolint:gosec // G115: ≤ target
				return nil, err
			}
			out = append(out, QuestProgress{ID: r.ID, Code: r.Code, Progress: progress, Target: int(r.Target), Completed: done != nil})
		}
	}
	return out, nil
}

// Claim grants a completed quest's XP once.
func (s *Service) Claim(ctx context.Context, user, id uuid.UUID) (Claimed, error) {
	var out Claimed
	err := postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		if err := q.LockUser(ctx, user.String()); err != nil {
			return err
		}
		a, err := q.GetAssignmentForUpdate(ctx, sqlc.GetAssignmentForUpdateParams{ID: id, UserID: user})
		if errors.Is(err, pgx.ErrNoRows) {
			return apierror.New(apierror.NotFound)
		}
		switch {
		case err != nil:
			return err
		case a.ClaimedAt != nil:
			return apierror.New(apierror.QuestAlreadyClaimed)
		case a.CompletedAt == nil:
			return apierror.New(apierror.QuestNotCompleted)
		}
		now := s.d.Clock.Now()
		if err := q.ClaimAssignment(ctx, sqlc.ClaimAssignmentParams{Now: &now, ID: id}); err != nil {
			return err
		}
		added, before, after, err := grant(ctx, q, user, int(a.RewardXp), sourceQuest, id.String(), now)
		out = Claimed{XP: added, Level: after, LevelUp: after > before}
		return err
	})
	return out, err
}

// Reroll swaps an open quest for another of its period, once a day for
// free (progression-formulas §8).
func (s *Service) Reroll(ctx context.Context, user, id uuid.UUID, locale string) (Quests, error) {
	err := postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		if err := q.LockUser(ctx, user.String()); err != nil {
			return err
		}
		a, err := q.GetAssignmentForUpdate(ctx, sqlc.GetAssignmentForUpdateParams{ID: id, UserID: user})
		if errors.Is(err, pgx.ErrNoRows) {
			return apierror.New(apierror.NotFound)
		}
		if err != nil {
			return err
		}
		now := s.d.Clock.Now()
		if a.CompletedAt != nil || a.ClaimedAt != nil || !a.PeriodEnd.After(now) {
			return apierror.New(apierror.QuestNotRerollable)
		}
		used, err := q.UseReroll(ctx, sqlc.UseRerollParams{UserID: user, Day: domain.Day(now)})
		if err != nil {
			return err
		}
		if int(used) > freeRerolls {
			return apierror.New(apierror.QuestRerollLimit)
		}
		rows, err := q.PeriodAssignments(ctx, sqlc.PeriodAssignmentsParams{UserID: user, Cadence: a.Cadence, PeriodStart: a.PeriodStart})
		if err != nil {
			return err
		}
		pool, err := q.ActiveQuests(ctx, a.Cadence)
		if err != nil {
			return err
		}
		next := pick(pool, rows, 1, a.QuestID)
		if len(next) == 0 {
			return apierror.New(apierror.QuestNotRerollable)
		}
		if err := q.DeleteAssignment(ctx, id); err != nil {
			return err
		}
		return q.InsertAssignment(ctx, sqlc.InsertAssignmentParams{
			ID: ids.New(), UserID: user, QuestID: next[0].ID, PeriodStart: a.PeriodStart, PeriodEnd: a.PeriodEnd,
		})
	})
	if err != nil {
		return Quests{}, err
	}
	return s.Quests(ctx, user, locale)
}
