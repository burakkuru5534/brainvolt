package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/team/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
)

// TeamInfo is what other modules read about a team (game lineups).
type TeamInfo struct {
	ID      uuid.UUID
	Active  bool
	Captain uuid.UUID
	Members []uuid.UUID
}

// TeamInfo returns a team's state and current members.
func (s *Service) TeamInfo(ctx context.Context, id uuid.UUID) (TeamInfo, error) {
	q := sqlc.New(s.d.DB)
	t, err := q.GetTeam(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return TeamInfo{}, apierror.New(apierror.NotFound)
	}
	if err != nil {
		return TeamInfo{}, err
	}
	ms, err := q.Members(ctx, id)
	if err != nil {
		return TeamInfo{}, err
	}
	out := TeamInfo{ID: id, Active: t.Status == "active", Captain: t.CaptainUserID}
	for _, m := range ms {
		out.Members = append(out.Members, m.UserID)
	}
	return out, nil
}

// CoupleInfo is what other modules read about a couple.
type CoupleInfo struct {
	ID     uuid.UUID
	Active bool
	Users  [2]uuid.UUID
}

func coupleInfo(c *sqlc.TeamCouple) CoupleInfo {
	return CoupleInfo{ID: c.ID, Active: c.Status == "active", Users: [2]uuid.UUID{c.UserLowID, c.UserHighID}}
}

// CoupleInfo returns a couple's state and partners.
func (s *Service) CoupleInfo(ctx context.Context, id uuid.UUID) (CoupleInfo, error) {
	c, err := sqlc.New(s.d.DB).GetCouple(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return CoupleInfo{}, apierror.New(apierror.NotFound)
	}
	if err != nil {
		return CoupleInfo{}, err
	}
	return coupleInfo(&c), nil
}

// ActiveCoupleOf returns a user's active couple (ok = false when none).
func (s *Service) ActiveCoupleOf(ctx context.Context, user uuid.UUID) (CoupleInfo, bool, error) {
	c, err := s.liveCouple(ctx, sqlc.New(s.d.DB), user)
	if err != nil || c == nil || c.Status != "active" {
		return CoupleInfo{}, false, err
	}
	return coupleInfo(c), true, nil
}
