package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/progression/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
)

// SeasonResult is a user's outcome of a closed season (ranking).
type SeasonResult struct {
	UserID     uuid.UUID
	PeakLeague string
	FinalRank  int
}

// SeasonResults reads a closed season's results (ranking, wired in app).
type SeasonResults interface {
	SeasonResults(ctx context.Context, season uuid.UUID) ([]SeasonResult, error)
}

type seasonEnded struct {
	SeasonID uuid.UUID `json:"season_id"`
}

// OnSeasonEnded awards season badges (progression-formulas §9): the
// badge of the peak league and "Top 100" for the global top 100. Badges
// are earned once, so a later season adds only new ones. Every ranked
// player gets the peak league's Volt (D-105).
func (s *Service) OnSeasonEnded(ctx context.Context, e events.Envelope, seasons SeasonResults) error {
	var p seasonEnded
	if err := e.Decode(&p); err != nil {
		return err
	}
	results, err := seasons.SeasonResults(ctx, p.SeasonID)
	if err != nil {
		return err
	}
	ref := "season:" + p.SeasonID.String()
	now := s.d.Clock.Now()
	return postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		badges, err := activeBadges(ctx, q)
		if err != nil {
			return err
		}
		for _, r := range results {
			// The peak league's Volt (D-105), once per season.
			if err := s.volt(ctx, tx, r.UserID, voltSeason[r.PeakLeague], reasonSeason, p.SeasonID.String(), ref, now); err != nil {
				return err
			}
			for i := range badges {
				b := &badges[i]
				c := b.criteria
				if (c.Type == "season_peak" && c.League == r.PeakLeague) || (c.Type == "season_top" && r.FinalRank > 0 && r.FinalRank <= c.Rank) {
					if _, _, err := s.award(ctx, tx, q, r.UserID, b, ref, now); err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
}
