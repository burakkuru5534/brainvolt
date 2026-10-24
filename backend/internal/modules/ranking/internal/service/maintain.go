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

// snapshotEnded archives every daily, weekly or monthly board whose
// period is over, once, and drops its entries when the board's TTL after
// the period has passed (the Redis-era expiry, redis-keys §6.1). All-time
// and season boards are not periodic: season boards are archived by
// closeEndedSeason and dropped by dropOldSeasonBoards; all-time boards stay.
func (s *Service) snapshotEnded(ctx context.Context) error {
	rows, err := s.d.DB.Query(ctx, `SELECT DISTINCT board_key FROM ranking.leaderboard_entries
		WHERE board_key LIKE 'rank:lb:d:%' OR board_key LIKE 'rank:lb:w:%' OR board_key LIKE 'rank:lb:m:%'`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err == nil {
			keys = append(keys, k)
		}
	}
	rows.Close()

	now := s.d.Clock.Now()
	for _, key := range keys {
		parts := strings.Split(key, ":")
		if len(parts) < 5 {
			continue // not a board key this code writes: leave it alone
		}
		per, ok := domain.PeriodByID(parts[2] + ":" + parts[3])
		if !ok || per.End.After(now) {
			continue
		}
		board := strings.TrimPrefix(key, "rank:lb:")
		var archived bool
		if err := s.d.DB.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM ranking.leaderboard_snapshots WHERE board_key = $1)`, board).Scan(&archived); err != nil {
			return err
		}
		if !archived {
			if err := s.snapshot(ctx, key, parts[4], per.Start, per.End); err != nil {
				return err
			}
		}
		if !per.End.Add(per.TTL).After(now) {
			if _, err := s.d.DB.Exec(ctx, `DELETE FROM ranking.leaderboard_entries WHERE board_key = $1`, key); err != nil {
				return err
			}
		}
	}
	return s.dropOldSeasonBoards(ctx, now)
}

// dropOldSeasonBoards deletes the entries of season boards whose season
// closed more than domain.SeasonTTL ago (they were archived at the close).
func (s *Service) dropOldSeasonBoards(ctx context.Context, now time.Time) error {
	_, err := s.d.DB.Exec(ctx, `
		DELETE FROM ranking.leaderboard_entries e
		USING ranking.seasons se
		WHERE se.status = 'closed' AND se.ends_at < $1
		  AND e.board_key LIKE 'rank:lb:s:' || se.number || ':%'
	`, now.Add(-domain.SeasonTTL))
	return err
}

// snapshot copies a board's top entries to leaderboard_snapshots under
// the key without its "rank:lb:" prefix (idempotent).
func (s *Service) snapshot(ctx context.Context, key, kind string, start, end time.Time) error {
	rows, err := s.d.DB.Query(ctx, `
		SELECT member, score FROM ranking.leaderboard_entries
		WHERE board_key = $1
		ORDER BY score DESC, member ASC
		LIMIT $2
	`, key, snapshotTop)
	if err != nil {
		return err
	}
	defer rows.Close()

	type item struct {
		member string
		score  float64
	}
	var items []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.member, &it.score); err == nil {
			items = append(items, it)
		}
	}
	rows.Close()

	if len(items) == 0 {
		return nil
	}

	board := strings.TrimPrefix(key, "rank:lb:")
	return postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		for i, it := range items {
			id, err := uuid.Parse(it.member)
			if err != nil {
				continue
			}
			if err := q.InsertSnapshot(ctx, sqlc.InsertSnapshotParams{
				BoardKey: board, PeriodStart: start, PeriodEnd: end, Rank: int32(i + 1),
				EntityKind: kind, EntityID: id, Score: int64(it.score),
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
		prefix := "rank:lb:" + period + ":" + kind + ":%"
		rows, err := s.d.DB.Query(ctx, `
			SELECT DISTINCT board_key FROM ranking.leaderboard_entries
			WHERE board_key LIKE $1
		`, prefix)
		if err == nil {
			var keys []string
			for rows.Next() {
				var k string
				if err := rows.Scan(&k); err == nil {
					keys = append(keys, k)
				}
			}
			rows.Close()
			for _, key := range keys {
				if err := s.snapshot(ctx, key, kind, se.StartsAt, se.EndsAt); err != nil {
					return err
				}
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
