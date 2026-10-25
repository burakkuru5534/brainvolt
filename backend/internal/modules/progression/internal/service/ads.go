package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/progression/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/progression/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
)

// Rewarded-ad placements (D-103); commerce verifies the ad, progression
// gives what it earns.
const (
	AdBonusXP      = "bonus_xp"
	AdQuestReroll  = "quest_reroll"
	AdStreakRepair = "streak_repair"

	sourceAdBonus = "ad_bonus"
	// adBonusMax caps a match's bonus XP; the bonus is half the match XP
	// (progression-formulas §5.1, D-031).
	adBonusMax = 50
	// adBonusWindow is how long after a match its bonus can be earned.
	adBonusWindow = time.Hour
)

// AdOffer is what a rewarded ad would give right now.
type AdOffer struct {
	XP         int `json:"xp,omitempty"`
	StreakDays int `json:"streak_days,omitempty"`
}

func unavailable(placement, reason string) error {
	return apierror.New(apierror.AdRewardUnavailable).With("placement", placement).With("reason", reason)
}

// AdOffer checks that placement can be earned by user now and says what
// it gives: bonus_xp needs a match (ref) that gave XP in the last hour
// and no bonus yet; quest_reroll needs the free reroll used and an open
// quest; streak_repair needs a streak lost by missing exactly yesterday.
func (s *Service) AdOffer(ctx context.Context, user uuid.UUID, placement, ref string) (AdOffer, error) {
	return s.adOffer(ctx, sqlc.New(s.d.DB), user, placement, ref, s.d.Clock.Now(), true)
}

func (s *Service) adOffer(ctx context.Context, q *sqlc.Queries, user uuid.UUID, placement, ref string, now time.Time, fresh bool) (AdOffer, error) {
	switch placement {
	case AdBonusXP:
		e, err := q.MatchXPEntry(ctx, sqlc.MatchXPEntryParams{UserID: user, MatchRef: ref})
		if errors.Is(err, pgx.ErrNoRows) {
			return AdOffer{}, unavailable(placement, "no_match_xp")
		}
		if err != nil {
			return AdOffer{}, err
		}
		if fresh && now.Sub(e.CreatedAt) > adBonusWindow {
			return AdOffer{}, unavailable(placement, "too_late")
		}
		given, err := q.AdBonusGiven(ctx, sqlc.AdBonusGivenParams{UserID: user, MatchRef: ref})
		if err != nil {
			return AdOffer{}, err
		}
		if given {
			return AdOffer{}, unavailable(placement, "already_given")
		}
		return AdOffer{XP: min((int(e.Amount)+1)/2, adBonusMax)}, nil
	case AdQuestReroll:
		st, err := q.RerollState(ctx, sqlc.RerollStateParams{UserID: user, Day: domain.Day(now)})
		if err != nil {
			return AdOffer{}, err
		}
		if int(st.Used) < freeRerolls+int(st.Bonus) {
			return AdOffer{}, unavailable(placement, "free_reroll_left")
		}
		open := false
		for _, c := range cadences {
			start, _ := domain.Period(c, now)
			rows, err := q.PeriodAssignments(ctx, sqlc.PeriodAssignmentsParams{UserID: user, Cadence: c, PeriodStart: start})
			if err != nil {
				return AdOffer{}, err
			}
			for i := range rows {
				if rows[i].CompletedAt == nil && rows[i].ClaimedAt == nil && rows[i].PeriodEnd.After(now) {
					open = true
				}
			}
		}
		if !open {
			return AdOffer{}, unavailable(placement, "no_open_quest")
		}
		return AdOffer{}, nil
	case AdStreakRepair:
		st, err := loadStreak(ctx, q, user)
		if err != nil {
			return AdOffer{}, err
		}
		if !repairable(st, now) {
			return AdOffer{}, unavailable(placement, "not_repairable")
		}
		return AdOffer{StreakDays: st.Current}, nil
	}
	return AdOffer{}, apierror.Validation(apierror.FieldError{Field: "placement", Rule: "enum"})
}

// repairable: the last play day was the day before yesterday, so only
// yesterday was missed and nothing was played today.
func repairable(st domain.Streak, now time.Time) bool {
	today := domain.Day(now)
	return st.Current > 0 && !st.LastDay.IsZero() && domain.Day(st.LastDay.AddDate(0, 0, 2)).Equal(today)
}

// adRewardGranted is commerce.AdRewardGranted's payload (D-103).
type adRewardGranted struct {
	RewardID  uuid.UUID `json:"reward_id"`
	UserID    uuid.UUID `json:"user_id"`
	Placement string    `json:"placement"`
	Ref       string    `json:"ref"`
	GrantedAt time.Time `json:"granted_at"`
}

// OnAdRewardGranted gives a verified rewarded ad its effect, once per
// reward. A reward whose condition no longer holds (the bonus was given,
// the streak was played on) is used up without effect.
func (s *Service) OnAdRewardGranted(ctx context.Context, e events.Envelope) error {
	var p adRewardGranted
	if err := e.Decode(&p); err != nil {
		return err
	}
	now := p.GrantedAt
	if now.IsZero() {
		now = s.d.Clock.Now()
	}
	return postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		if err := q.LockUser(ctx, p.UserID.String()); err != nil {
			return err
		}
		n, err := q.MarkAdRewardApplied(ctx, sqlc.MarkAdRewardAppliedParams{RewardID: p.RewardID, UserID: p.UserID, Placement: p.Placement, Now: now})
		if err != nil || n == 0 {
			return err
		}
		offer, err := s.adOffer(ctx, q, p.UserID, p.Placement, p.Ref, now, false)
		var ae *apierror.Error
		if errors.As(err, &ae) {
			return nil // nothing left to give
		}
		if err != nil {
			return err
		}
		switch p.Placement {
		case AdBonusXP:
			_, _, _, err = grant(ctx, q, p.UserID, offer.XP, sourceAdBonus, p.Ref, now)
		case AdQuestReroll:
			err = q.AddRerollBonus(ctx, sqlc.AddRerollBonusParams{UserID: p.UserID, Day: domain.Day(now)})
		case AdStreakRepair:
			err = q.SetStreakLastDay(ctx, sqlc.SetStreakLastDayParams{UserID: p.UserID, Day: ptr(domain.Day(now).AddDate(0, 0, -1)), Now: now})
		}
		return err
	})
}

func ptr[T any](v T) *T { return &v }
