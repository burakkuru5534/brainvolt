package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/ranking/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/ranking/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
)

// tierChanged is the part of identity.AccountTierChanged ranking reads.
type tierChanged struct {
	UserID uuid.UUID `json:"user_id"`
	To     string    `json:"to"`
}

// OnTierChanged replays a user's hidden ledger rows into the boards of
// periods still running once the account becomes full (schema-ranking
// §4.2, D-016, D-017), and adds the current season standing.
func (s *Service) OnTierChanged(ctx context.Context, e events.Envelope) error {
	var p tierChanged
	if err := e.Decode(&p); err != nil {
		return err
	}
	if p.To != "full" {
		return nil
	}
	season, err := s.ensureSeason(ctx)
	if err != nil {
		return err
	}
	now := s.d.Clock.Now()
	var incs []boardInc
	err = postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		incs = nil
		q := sqlc.New(tx)
		rows, err := q.PublishUserLedger(ctx, p.UserID)
		if err != nil {
			return err
		}
		for _, r := range rows {
			var countries []string
			if r.CountryCode != nil {
				countries = []string{*r.CountryCode}
			}
			incs = append(incs, ledgerIncs("user", p.UserID, r.CategoryID, int(r.Points), countries, r.OccurredAt, now)...)
		}
		st, err := q.GetStanding(ctx, sqlc.GetStandingParams{SeasonID: season.ID, EntityKind: "user", EntityID: p.UserID})
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		var countries []string
		if s.d.Cards != nil {
			cards, err := s.d.Cards.Cards(ctx, []uuid.UUID{p.UserID})
			if err != nil {
				return err
			}
			if c := cards[p.UserID].CountryCode; c != nil {
				countries = []string{*c}
			}
		}
		for _, scope := range domain.Scopes(countries) {
			incs = append(incs, boardInc{key: domain.BoardKey(domain.SeasonPeriodID(season.Number), "user", scope, "all"),
				member: p.UserID.String(), score: float64(st.SeasonPoints), set: true})
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.applyBoards(ctx, incs)
	return nil
}

// ledgerIncs are the board writes of one ledger row: every period
// holding occurredAt that has not ended by now, global and country scope.
func ledgerIncs(kind string, id uuid.UUID, cat *uuid.UUID, points int, countries []string, occurredAt, now time.Time) []boardInc {
	category := "all"
	if cat != nil {
		category = "cat:" + cat.String()
	}
	var out []boardInc
	for _, per := range domain.Periods(occurredAt) {
		var exp time.Time
		if per.TTL > 0 {
			if !per.End.After(now) {
				continue
			}
			exp = per.End.Add(per.TTL)
		}
		for _, scope := range domain.Scopes(countries) {
			out = append(out, boardInc{key: domain.BoardKey(per.ID, kind, scope, category), member: id.String(), score: float64(points), expireAt: exp})
		}
	}
	return out
}
