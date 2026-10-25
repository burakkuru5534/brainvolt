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
	// Kind "friends" earns nothing (D-109).
	Kind       string `json:"kind"`
	Placements []struct {
		UserID uuid.UUID `json:"user_id"`
		Place  int       `json:"place"`
		XP     int       `json:"xp"`
	} `json:"placements"`
}

// OnTournamentFinished gives each player the XP of their final place
// (progression-formulas §5, D-090) and places 1–3 their Volt (D-105),
// once per tournament.
func (s *Service) OnTournamentFinished(ctx context.Context, e events.Envelope) error {
	var p tournamentFinished
	if err := e.Decode(&p); err != nil {
		return err
	}
	if p.Kind == "friends" {
		return nil
	}
	return postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		for _, pl := range p.Placements {
			if err := q.LockUser(ctx, pl.UserID.String()); err != nil {
				return err
			}
			id := p.TournamentID.String()
			if _, _, _, err := s.grant(ctx, tx, q, pl.UserID, pl.XP, sourceTournament, id, e.OccurredAt); err != nil {
				return err
			}
			// Places 1–3 earn Volt (D-105).
			if err := s.volt(ctx, tx, pl.UserID, voltTournament[pl.Place], reasonTournament, id, "tournament:"+id, e.OccurredAt); err != nil {
				return err
			}
		}
		return nil
	})
}
