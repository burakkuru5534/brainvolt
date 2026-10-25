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
)

// Store items progression applies (commerce sells them for Volt, D-105).
const (
	ItemStreakShield = "streak_shield"
	ItemQuestReroll  = "quest_reroll"
)

func itemUnavailable(item, reason string) error {
	return apierror.New(apierror.ItemUnavailable).With("item", item).With("reason", reason)
}

// ApplyItemTx gives a bought item its effect inside commerce's purchase
// transaction: a streak shield (at most domain.MaxShields held) or an
// extra quest reroll for today (only once the free ones are used and a
// quest is open). It refuses with ITEM_UNAVAILABLE.
func (s *Service) ApplyItemTx(ctx context.Context, tx pgx.Tx, user uuid.UUID, item, _ string, now time.Time) error {
	q := sqlc.New(tx)
	if err := q.LockUser(ctx, user.String()); err != nil {
		return err
	}
	switch item {
	case ItemStreakShield:
		_, err := q.AddShield(ctx, sqlc.AddShieldParams{UserID: user, Now: now})
		if errors.Is(err, pgx.ErrNoRows) {
			return itemUnavailable(item, "shields_full")
		}
		return err
	case ItemQuestReroll:
		reason, err := s.rerollBlocked(ctx, q, user, now)
		if err != nil {
			return err
		}
		if reason != "" {
			return itemUnavailable(item, reason)
		}
		return q.AddRerollBonus(ctx, sqlc.AddRerollBonusParams{UserID: user, Day: domain.Day(now)})
	}
	return itemUnavailable(item, "unknown_item")
}
