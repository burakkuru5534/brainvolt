// Package service holds ranking's use cases: seasons, Glicko-2 ratings,
// season standings and the points ledger behind every leaderboard
// (schema-ranking.md, progression-formulas.md ruleset 1).
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/ranking/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/ranking/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/team"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/redis"
)

// Cards reads users' countries (profile).
type Cards interface {
	Cards(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]profile.Card, error)
}

// Teams reads a persistent team's captain (team country, D-029) and the
// labels of teams and couples on boards.
type Teams interface {
	TeamInfo(ctx context.Context, id uuid.UUID) (team.Info, error)
	TeamLabels(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]team.Label, error)
	CoupleLabels(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]team.Label, error)
}

// Deps are the service's collaborators.
type Deps struct {
	DB      *pgxpool.Pool
	Redis   *redis.Client
	Cards   Cards
	Teams   Teams
	Friends Friends
	Clock   clock.Clock
	Logger  *slog.Logger
}

// Service implements ranking.
type Service struct {
	d Deps
}

// New builds the service.
func New(d Deps) *Service { return &Service{d: d} }

// Season is a ranked season.
type Season struct {
	ID       uuid.UUID         `json:"id"`
	Number   int32             `json:"number"`
	StartsAt time.Time         `json:"starts_at"`
	EndsAt   time.Time         `json:"ends_at"`
	Status   string            `json:"status"`
	Names    map[string]string `json:"names"`
}

// resetPolicy is a season's soft reset (progression-formulas §9).
type resetPolicy struct {
	MinRD float64 `json:"min_rd"`
}

// ActiveSeason returns the active season, opening the first one when
// none exists yet (game's SeasonProvider).
func (s *Service) ActiveSeason(ctx context.Context) (uuid.UUID, bool, error) {
	se, err := s.ensureSeason(ctx)
	if err != nil {
		return uuid.Nil, false, err
	}
	return se.ID, true, nil
}

func (s *Service) ensureSeason(ctx context.Context) (sqlc.RankingSeason, error) {
	q := sqlc.New(s.d.DB)
	se, err := q.ActiveSeason(ctx)
	if !errors.Is(err, pgx.ErrNoRows) {
		return se, err
	}
	last, err := q.LastSeasonNumber(ctx)
	if err != nil {
		return se, err
	}
	if err := s.openSeason(ctx, last+1, domain.WeekStart(s.d.Clock.Now())); err != nil {
		var pg *pgconn.PgError
		if !errors.As(err, &pg) || pg.Code != "23505" {
			return se, err
		}
		// Another process opened it first.
	}
	return q.ActiveSeason(ctx)
}

// openSeason inserts season number n starting at start with its names.
func (s *Service) openSeason(ctx context.Context, n int32, start time.Time) error {
	policy, err := json.Marshal(resetPolicy{MinRD: 150})
	if err != nil {
		return err
	}
	id := ids.New()
	return postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		if err := q.InsertSeason(ctx, sqlc.InsertSeasonParams{
			ID: id, Number: n, StartsAt: start.UTC(), EndsAt: start.Add(domain.SeasonLength).UTC(), ResetPolicy: policy,
		}); err != nil {
			return err
		}
		for locale, name := range map[string]string{"tr": fmt.Sprintf("Sezon %d", n), "en": fmt.Sprintf("Season %d", n)} {
			if err := q.InsertSeasonName(ctx, sqlc.InsertSeasonNameParams{SeasonID: id, Locale: locale, Name: name}); err != nil {
				return err
			}
		}
		return nil
	})
}

// CurrentSeason returns the active season with its names.
func (s *Service) CurrentSeason(ctx context.Context) (Season, error) {
	se, err := s.ensureSeason(ctx)
	if err != nil {
		return Season{}, err
	}
	return s.season(ctx, se)
}

func (s *Service) season(ctx context.Context, se sqlc.RankingSeason) (Season, error) {
	names, err := sqlc.New(s.d.DB).SeasonNames(ctx, se.ID)
	if err != nil {
		return Season{}, err
	}
	out := Season{ID: se.ID, Number: se.Number, StartsAt: se.StartsAt, EndsAt: se.EndsAt, Status: se.Status, Names: map[string]string{}}
	for _, n := range names {
		out.Names[n.Locale] = n.Name
	}
	return out, nil
}

// Rating is a rating and its deviation (matchmaking's RatingReader).
type Rating struct {
	Value float64
	RD    float64
}

// Rating returns a user's rating of a bucket with RD grown for idle
// days; new players get the Glicko-2 defaults.
func (s *Service) Rating(ctx context.Context, user uuid.UUID, bucket string) (Rating, error) {
	r, _, err := loadRating(ctx, sqlc.New(s.d.DB), "user", user, bucket, s.d.Clock.Now())
	if err != nil {
		return Rating{}, err
	}
	return Rating{Value: r.Value, RD: r.RD}, nil
}

// loadRating reads a rating with inactivity applied (§1.2); ok = false
// for an entity never rated in the bucket (defaults returned).
func loadRating(ctx context.Context, q *sqlc.Queries, kind string, id uuid.UUID, bucket string, now time.Time) (domain.Rating, bool, error) {
	row, err := q.GetRating(ctx, sqlc.GetRatingParams{EntityKind: kind, EntityID: id, Bucket: bucket})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.NewRating(), false, nil
	}
	if err != nil {
		return domain.Rating{}, false, err
	}
	r := domain.Rating{Value: row.Rating, RD: row.Rd, Volatility: row.Volatility}
	if row.LastRatedAt != nil {
		r = r.Idle(int(now.Sub(*row.LastRatedAt) / (24 * time.Hour)))
	}
	return r, true, nil
}
