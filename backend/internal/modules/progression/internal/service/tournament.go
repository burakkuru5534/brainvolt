package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/progression/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
)

const sourceTournament = "tournament"

type tournamentFinished struct {
	TournamentID uuid.UUID `json:"tournament_id"`
	Placements   []struct {
		UserID uuid.UUID `json:"user_id"`
		XP     int       `json:"xp"`
	} `json:"placements"`
}

// OnTournamentFinished gives each player the XP of their final place
// (progression-formulas §5, D-090), once per tournament.
func (s *Service) OnTournamentFinished(ctx context.Context, e events.Envelope) error {
	var p tournamentFinished
	if err := e.Decode(&p); err != nil {
		return err
	}
	return postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		for _, pl := range p.Placements {
			if err := q.LockUser(ctx, pl.UserID.String()); err != nil {
				return err
			}
			if _, _, _, err := grant(ctx, q, pl.UserID, pl.XP, sourceTournament, p.TournamentID.String(), e.OccurredAt); err != nil {
				return err
			}
		}
		return nil
	})
}
