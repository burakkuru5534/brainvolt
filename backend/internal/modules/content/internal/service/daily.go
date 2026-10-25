package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/content/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/content/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
)

// Daily set rules (category-modes §5.1): 10 multiple-choice questions on
// the Ranked curve, verified in both locales, at most 3 per category.
var dailyCurve = []int{2, 3, 3, 3, 4, 4, 4, 4, 5, 5}

// dailyNoRepeat keeps a question out of daily sets for this long.
const dailyNoRepeat = 90 * 24 * time.Hour

// DailySet is a question set in order: a day's Daily Challenge or a
// creator quiz.
type DailySet struct {
	ID        uuid.UUID
	Questions []SelectedQuestion
}

func date(t time.Time) pgtype.Date {
	return pgtype.Date{Time: time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), Valid: true}
}

// DailySet returns day's set, publishing it on first use (the worker
// publishes ahead; concurrent publishers agree on the first insert).
// day is a calendar day (its Y-M-D is used).
func (s *Service) DailySet(ctx context.Context, day time.Time) (DailySet, error) {
	q := sqlc.New(s.db)
	id, err := q.DailySetOf(ctx, date(day))
	if errors.Is(err, pgx.ErrNoRows) {
		if err := s.publishDaily(ctx, day); err != nil && !errors.Is(err, errDailyExists) {
			return DailySet{}, err
		}
		id, err = q.DailySetOf(ctx, date(day))
	}
	if err != nil {
		return DailySet{}, err
	}
	return s.questionSet(ctx, q, id)
}

// PublishedDailySet returns day's set only if it was published; unlike
// DailySet it never publishes one (looking back at past days, D-100).
func (s *Service) PublishedDailySet(ctx context.Context, day time.Time) (DailySet, bool, error) {
	q := sqlc.New(s.db)
	id, err := q.DailySetOf(ctx, date(day))
	if errors.Is(err, pgx.ErrNoRows) {
		return DailySet{}, false, nil
	}
	if err != nil {
		return DailySet{}, false, err
	}
	set, err := s.questionSet(ctx, q, id)
	return set, err == nil, err
}

// QuestionSet returns a question set's questions in order whatever their
// verification state (creator quizzes are played among friends before
// review).
func (s *Service) QuestionSet(ctx context.Context, id uuid.UUID) (DailySet, error) {
	return s.questionSet(ctx, sqlc.New(s.db), id)
}

func (s *Service) questionSet(ctx context.Context, q *sqlc.Queries, id uuid.UUID) (DailySet, error) {
	rows, err := q.SetQuestions(ctx, id)
	if err != nil {
		return DailySet{}, err
	}
	out := DailySet{ID: id}
	qids := make([]uuid.UUID, len(rows))
	for i, r := range rows {
		qids[i] = r.ID
		out.Questions = append(out.Questions, SelectedQuestion{
			ID: r.ID, Revision: r.Revision, Type: domain.QuestionType(r.QuestionType), Difficulty: int(r.DifficultyUsed),
			CategoryID: r.CategoryID, SubcategoryID: r.SubcategoryID,
		})
	}
	opts, err := q.OptionsOf(ctx, qids)
	if err != nil {
		return DailySet{}, err
	}
	for i := range out.Questions {
		for _, o := range opts {
			if o.QuestionID == out.Questions[i].ID {
				out.Questions[i].Options = append(out.Questions[i].Options, o.ID)
				if o.IsCorrect {
					out.Questions[i].Correct = o.ID
				}
			}
		}
	}
	return out, nil
}

// publishDaily selects and stores a day's set.
func (s *Service) publishDaily(ctx context.Context, day time.Time) error {
	recent, err := sqlc.New(s.db).RecentDailyQuestions(ctx, date(day.Add(-dailyNoRepeat)))
	if err != nil {
		return err
	}
	picked, err := s.Select(ctx, SelectRequest{
		Locales: []string{"tr", "en"}, MultipleChoiceOnly: true, Exclude: recent,
		Rules: domain.SelectRules{Difficulties: dailyCurve, MaxPerCategory: 3},
	})
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	setID := ids.New()
	return postgres.InTx(ctx, s.db, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		if err := q.InsertQuestionSet(ctx, sqlc.InsertQuestionSetParams{ID: setID, Kind: "daily", Now: now}); err != nil {
			return err
		}
		for i, p := range picked {
			if err := q.InsertSetItem(ctx, sqlc.InsertSetItemParams{SetID: setID, Position: int16(i + 1), QuestionID: p.ID}); err != nil {
				return err
			}
		}
		n, err := q.InsertDailySet(ctx, sqlc.InsertDailySetParams{Day: date(day), SetID: setID, Now: now})
		if err != nil {
			return err
		}
		if n == 0 {
			return errDailyExists // another publisher won: drop this set
		}
		return nil
	})
}

var errDailyExists = errors.New("daily set exists")
