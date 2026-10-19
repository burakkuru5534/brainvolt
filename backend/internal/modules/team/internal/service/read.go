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

// Label is how a leaderboard shows a team or couple.
type Label struct {
	Name     string      `json:"name"`
	Tag      string      `json:"tag,omitempty"`
	EmblemID string      `json:"emblem_id,omitempty"`
	Users    []uuid.UUID `json:"-"`
}

// TeamLabels returns the names, tags and emblems of teams.
func (s *Service) TeamLabels(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]Label, error) {
	rows, err := sqlc.New(s.d.DB).TeamsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]Label, len(rows))
	for _, r := range rows {
		out[r.ID] = Label{Name: r.Name, Tag: r.Tag, EmblemID: r.EmblemID}
	}
	return out, nil
}

// CoupleLabels returns the names and partners of couples.
func (s *Service) CoupleLabels(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]Label, error) {
	rows, err := sqlc.New(s.d.DB).CouplesByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]Label, len(rows))
	for _, r := range rows {
		out[r.ID] = Label{Name: r.Name, Users: []uuid.UUID{r.UserLowID, r.UserHighID}}
	}
	return out, nil
}
