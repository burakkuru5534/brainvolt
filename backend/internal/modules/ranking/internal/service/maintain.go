package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/ranking/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/ranking/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
)

// snapshotTop boards entries are archived per board (D-029).
const snapshotTop = 1000

// Maintain is ranking's periodic worker job: it archives boards whose
// period ended and closes an ended season (schema-ranking §4.2, §6).
func (s *Service) Maintain(ctx context.Context) error {
	if err := s.snapshotEnded(ctx); err != nil {
		return err
	}
	return s.closeEndedSeason(ctx)
}

// snapshotEnded archives every touched daily, weekly or monthly board
// whose period is over; Redis expires the boards on their own.
func (s *Service) snapshotEnded(ctx context.Context) error {
	keys, err := s.d.Redis.SMembers(ctx, dirtyBoards).Result()
	if err != nil {
		return err
	}
	now := s.d.Clock.Now()
	for _, key := range keys {
		parts := strings.Split(key, ":")
		if len(parts) < 5 {
			s.d.Redis.SRem(ctx, dirtyBoards, key)
			continue
		}
		per, ok := domain.PeriodByID(parts[2] + ":" + parts[3])
		if !ok {
			s.d.Redis.SRem(ctx, dirtyBoards, key)
			continue
		}
		if per.End.After(now) {
			continue
		}
		if err := s.snapshot(ctx, key, parts[4], per.Start, per.End); err != nil {
			return err
		}
		if err := s.d.Redis.SRem(ctx, dirtyBoards, key).Err(); err != nil {
			return err
		}
	}
	return nil
}

// snapshot copies a board's top entries to leaderboard_snapshots under
// the key without its "rank:lb:" prefix (idempotent).
func (s *Service) snapshot(ctx context.Context, key, kind string, start, end time.Time) error {
	zs, err := s.d.Redis.ZRevRangeWithScores(ctx, key, 0, snapshotTop-1).Result()
	if err != nil || len(zs) == 0 {
		return err
	}
	board := strings.TrimPrefix(key, "rank:lb:")
	return postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		for i, z := range zs {
			member, _ := z.Member.(string)
			id, err := uuid.Parse(member)
			if err != nil {
				continue
			}
			if err := q.InsertSnapshot(ctx, sqlc.InsertSnapshotParams{
				BoardKey: board, PeriodStart: start, PeriodEnd: end, Rank: int32(i + 1),
				EntityKind: kind, EntityID: id, Score: int64(z.Score),
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

// closeEndedSeason closes the active season once it ended
// (progression-formulas §9): season boards archived and kept 30 days,
// final ranks, the rating soft reset recorded under the season id, and
// the next season opened at the old end (skipping seasons the worker
// missed entirely).
func (s *Service) closeEndedSeason(ctx context.Context) error {
	q := sqlc.New(s.d.DB)
	se, err := q.ActiveSeason(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	now := s.d.Clock.Now()
	if se.EndsAt.After(now) {
		return nil
	}
	period := domain.SeasonPeriodID(se.Number)
	for _, kind := range BoardEntities {
		var cursor uint64
		for {
			keys, next, err := s.d.Redis.Scan(ctx, cursor, domain.BoardKey(period, kind, "*", "*"), 200).Result()
			if err != nil {
				return err
			}
			for _, key := range keys {
				if err := s.snapshot(ctx, key, kind, se.StartsAt, se.EndsAt); err != nil {
					return err
				}
				if err := s.d.Redis.ExpireAt(ctx, key, now.Add(domain.SeasonTTL)).Err(); err != nil {
					return err
				}
			}
			if cursor = next; cursor == 0 {
				break
			}
		}
	}
	var policy resetPolicy
	if err := json.Unmarshal(se.ResetPolicy, &policy); err != nil {
		return err
	}
	start := se.EndsAt
	for !start.Add(domain.SeasonLength).After(now) {
		start = start.Add(domain.SeasonLength)
	}
	return postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		if err := q.SetFinalRanks(ctx, se.ID); err != nil {
			return err
		}
		if policy.MinRD > 0 {
			if _, err := q.SoftReset(ctx, sqlc.SoftResetParams{ResetID: se.ID, SeasonID: se.ID, MinRd: policy.MinRD, Now: now}); err != nil {
				return err
			}
		}
		if err := q.SetSeasonStatus(ctx, sqlc.SetSeasonStatusParams{Status: "closed", ClosedAt: &now, ID: se.ID}); err != nil {
			return err
		}
		next, err := insertSeason(ctx, q, se.Number+1, start.In(domain.Istanbul))
		if err != nil || s.d.Events == nil {
			return err
		}
		return s.d.Events.PublishTx(ctx, tx, EventSeasonEnded, now, SeasonEnded{SeasonID: se.ID, Number: se.Number, NextSeason: next})
	})
}

// EventSeasonEnded is published when a season closes (schema-ranking §7).
const EventSeasonEnded = "ranking.SeasonEnded"

// SeasonEnded is the payload of ranking.SeasonEnded.
type SeasonEnded struct {
	SeasonID   uuid.UUID `json:"season_id"`
	Number     int32     `json:"number"`
	NextSeason uuid.UUID `json:"next_season_id"`
}

// SeasonResult is a user's outcome of a closed season.
type SeasonResult struct {
	UserID     uuid.UUID
	PeakLeague string
	FinalRank  int
}

// SeasonResults lists the users' peak leagues and final ranks of a season.
func (s *Service) SeasonResults(ctx context.Context, season uuid.UUID) ([]SeasonResult, error) {
	rows, err := sqlc.New(s.d.DB).SeasonUserResults(ctx, season)
	if err != nil {
		return nil, err
	}
	out := make([]SeasonResult, len(rows))
	for i, r := range rows {
		out[i] = SeasonResult{UserID: r.UserID, PeakLeague: r.PeakLeague}
		if r.FinalRankGlobal != nil {
			out[i].FinalRank = int(*r.FinalRankGlobal)
		}
	}
	return out, nil
}
