package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/progression/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/progression/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
)

// Daily Challenge XP (progression-formulas §5.1).
const (
	xpDailyBase    = 30
	xpDailyCorrect = 5
	sourceDaily    = "daily_challenge"
)

type dailyRunCompleted struct {
	RunID        uuid.UUID `json:"run_id"`
	UserID       uuid.UUID `json:"user_id"`
	CorrectCount int       `json:"correct_count"`
	CompletedAt  time.Time `json:"completed_at"`
}

// OnDailyRunCompleted gives an official daily run its XP, counts the
// play day and moves daily_challenge quests, once per run.
func (s *Service) OnDailyRunCompleted(ctx context.Context, e events.Envelope) error {
	var p dailyRunCompleted
	if err := e.Decode(&p); err != nil {
		return err
	}
	return postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		if err := q.LockUser(ctx, p.UserID.String()); err != nil {
			return err
		}
		run := p.RunID.String()
		added, _, _, err := s.grant(ctx, tx, q, p.UserID, xpDailyBase+xpDailyCorrect*p.CorrectCount, sourceDaily, run, p.CompletedAt)
		if err != nil || added == 0 {
			return err // already consumed
		}
		if err := s.volt(ctx, tx, p.UserID, voltDaily, reasonDaily, run, "daily:"+run, p.CompletedAt); err != nil {
			return err
		}
		if err := s.plusShield(ctx, q, p.UserID, p.CompletedAt); err != nil {
			return err
		}
		st, err := loadStreak(ctx, q, p.UserID)
		if err != nil {
			return err
		}
		if err := saveStreak(ctx, q, p.UserID, st.Play(domain.Day(p.CompletedAt)), p.CompletedAt); err != nil {
			return err
		}
		for _, c := range cadences {
			rows, err := ensureQuests(ctx, q, p.UserID, c, p.CompletedAt, s.extraQuests(ctx, p.UserID))
			if err != nil {
				return err
			}
			for i := range rows {
				r := &rows[i]
				if r.Objective != "daily_challenge" || r.CompletedAt != nil {
					continue
				}
				progress := min(int(r.Progress)+1, int(r.Target))
				var done *time.Time
				if progress >= int(r.Target) {
					done = &p.CompletedAt
				}
				if err := q.SetAssignmentProgress(ctx, sqlc.SetAssignmentProgressParams{Progress: int32(progress), CompletedAt: done, ID: r.ID}); err != nil { //nolint:gosec // G115: ≤ target
					return err
				}
			}
		}
		return nil
	})
}
