package service

import (
	"context"
	"encoding/json"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/lobby/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/realtime"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
)

// RevengeWindow is how long after a match the losing side may ask for
// revenge (D-027).
const RevengeWindow = 24 * time.Hour

// Revenge sends a revenge challenge from the losing side of a completed
// 1v1 or Couple Contest match to the winners (D-045): the challenger
// picks a home category; the match copies the parent's configuration
// when accepted (game applies R15: losing side, 24 h, chain < 3). Only
// blocks stop it — the players just met, so challenges_from does not
// apply. One pending revenge per match.
func (s *Service) Revenge(ctx context.Context, from, parent, homeCategory uuid.UUID) (Challenge, error) {
	sum, err := s.d.Game.Summary(ctx, parent)
	if err != nil {
		return Challenge{}, err
	}
	now := s.d.Clock.Now().UTC()
	if sum.Status != game.StatusCompleted || sum.EndedAt == nil || sum.WinnerSide == 0 || now.Sub(*sum.EndedAt) >= RevengeWindow {
		return Challenge{}, apierror.New(apierror.RevengeNotAllowed).With("reason", "window")
	}
	var losers, winners []uuid.UUID
	for _, p := range sum.Participants {
		if p.Side == sum.WinnerSide {
			winners = append(winners, p.UserID)
		} else {
			losers = append(losers, p.UserID)
		}
	}
	if !slices.Contains(losers, from) {
		return Challenge{}, apierror.New(apierror.RevengeNotAllowed).With("reason", "not_losing_side")
	}
	target, couples := ChallengeTarget{User: &winners[0]}, [2]*uuid.UUID{}
	switch {
	case len(winners) == 1 && len(losers) == 1:
	case len(winners) == 2 && sum.Couples[0] != nil && sum.Couples[1] != nil:
		couples = [2]*uuid.UUID{sum.Couples[2-sum.WinnerSide], sum.Couples[sum.WinnerSide-1]}
		target = ChallengeTarget{Couple: couples[1]}
	default:
		return Challenge{}, apierror.New(apierror.ModeNotAllowed).With("reason", "revenge_1v1_or_couple")
	}
	ok, err := s.d.Game.CategoryExists(ctx, homeCategory)
	if err != nil {
		return Challenge{}, err
	}
	if !ok {
		return Challenge{}, apierror.Validation(apierror.FieldError{Field: "home_category_id", Rule: "active_category"})
	}
	for _, w := range winners {
		for _, l := range losers {
			if blocked, err := s.d.Social.IsBlocked(ctx, l, w); err != nil || blocked {
				if err != nil {
					return Challenge{}, err
				}
				return Challenge{}, apierror.New(apierror.Blocked)
			}
		}
	}
	body, err := json.Marshal(game.Config{Origin: game.OriginRevenge, CategoryIDs: []uuid.UUID{homeCategory}})
	if err != nil {
		return Challenge{}, err
	}
	id := ids.New()
	err = postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		pending, err := q.PendingRevengeOf(ctx, sqlc.PendingRevengeOfParams{ParentMatchID: &parent, Now: now})
		if err != nil {
			return err
		}
		if pending {
			return apierror.New(apierror.RequestPending)
		}
		sent, err := q.ChallengesSentSince(ctx, sqlc.ChallengesSentSinceParams{UserID: from, Since: now.Add(-time.Hour)})
		if err != nil {
			return err
		}
		if sent >= ChallengesPerHour {
			return apierror.New(apierror.RateLimited).WithRetryAfter(time.Hour)
		}
		if err := q.InsertChallenge(ctx, sqlc.InsertChallengeParams{
			ID: id, Kind: "revenge", ChallengerUserID: from, ChallengerCoupleID: couples[0], TargetUserID: target.User,
			TargetCoupleID: target.Couple, MatchConfig: body, ParentMatchID: &parent, Now: now, ExpiresAt: sum.EndedAt.Add(RevengeWindow),
		}); err != nil {
			return err
		}
		return s.d.Events.PublishTx(ctx, tx, EventChallengeSent, now, ChallengeSent{ChallengeID: id, ChallengerID: from, TargetID: winners[0], Kind: "revenge", TargetIDs: winners})
	})
	if err != nil {
		return Challenge{}, err
	}
	c, err := s.challenge(ctx, id)
	if err != nil {
		return Challenge{}, err
	}
	for _, w := range winners {
		s.send(ctx, w, realtime.Frame{T: "challenge.received", D: map[string]any{"challenge": c}})
	}
	return c, nil
}
