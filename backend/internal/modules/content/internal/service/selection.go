package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/content/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/content/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
)

// SelectRequest asks for the questions of one match (match-types §2,
// schema-content §3: verified question, verified translation in every
// locale of the match).
type SelectRequest struct {
	Locales            []string
	MultipleChoiceOnly bool
	// Rules: one target difficulty per question, category cap, true/false
	// share, revenge home category.
	Rules domain.SelectRules
	// CategoryIDs limit the pool to these top-level categories.
	CategoryIDs   []uuid.UUID
	SubcategoryID *uuid.UUID
	// Exclude are questions the participants saw recently (no-repeat
	// window); if the pool is too small without them they are allowed back.
	Exclude []uuid.UUID
}

// SelectedQuestion is one selected question.
type SelectedQuestion struct {
	ID            uuid.UUID
	Revision      int32
	Type          domain.QuestionType
	Difficulty    int
	CategoryID    uuid.UUID
	SubcategoryID uuid.UUID
	// Options in canonical order; Correct is the correct option's id.
	Options []uuid.UUID
	Correct uuid.UUID
}

// candidatePool caps how many candidates one selection reads.
const candidatePool = 2_000

// Select returns the questions in round order, or NOT_ENOUGH_QUESTIONS.
func (s *Service) Select(ctx context.Context, r SelectRequest) ([]SelectedQuestion, error) {
	out, err := s.selectOnce(ctx, r)
	if err == nil || len(r.Exclude) == 0 {
		return out, err
	}
	// The pool is exhausted with the no-repeat window: allow repeats
	// rather than failing (schema-content §3).
	r.Exclude = nil
	return s.selectOnce(ctx, r)
}

func (s *Service) selectOnce(ctx context.Context, r SelectRequest) ([]SelectedQuestion, error) {
	q := sqlc.New(s.db)
	rows, err := q.SelectCandidates(ctx, sqlc.SelectCandidatesParams{
		McOnly: r.MultipleChoiceOnly, CategoryIds: nonNil(r.CategoryIDs), SubcategoryID: r.SubcategoryID,
		Exclude: nonNil(r.Exclude), Locales: r.Locales, RowLimit: candidatePool,
	})
	if err != nil {
		return nil, err
	}
	pool := make([]domain.Candidate, len(rows))
	revision := map[uuid.UUID]int32{}
	for i, row := range rows {
		pool[i] = domain.Candidate{
			ID: row.ID, Type: domain.QuestionType(row.QuestionType), Difficulty: int(row.DifficultyUsed),
			CategoryID: row.CategoryID, SubcategoryID: row.SubcategoryID,
		}
		revision[row.ID] = row.Revision
	}
	picked, ok := domain.Select(pool, r.Rules)
	if !ok {
		return nil, apierror.New(apierror.NotEnoughQuestions).With("available", len(pool)).With("needed", len(r.Rules.Difficulties))
	}
	ids := make([]uuid.UUID, len(picked))
	for i, c := range picked {
		ids[i] = c.ID
	}
	opts, err := q.OptionsOf(ctx, ids)
	if err != nil {
		return nil, err
	}
	byQuestion := map[uuid.UUID][]sqlc.OptionsOfRow{}
	for _, o := range opts {
		byQuestion[o.QuestionID] = append(byQuestion[o.QuestionID], o)
	}
	out := make([]SelectedQuestion, len(picked))
	for i, c := range picked {
		sq := SelectedQuestion{
			ID: c.ID, Revision: revision[c.ID], Type: c.Type, Difficulty: c.Difficulty,
			CategoryID: c.CategoryID, SubcategoryID: c.SubcategoryID,
		}
		for _, o := range byQuestion[c.ID] {
			sq.Options = append(sq.Options, o.ID)
			if o.IsCorrect {
				sq.Correct = o.ID
			}
		}
		out[i] = sq
	}
	return out, nil
}

// CategoryExists reports whether id is an active top-level category
// (Category Master in matchmaking, R12).
func (s *Service) CategoryExists(ctx context.Context, id uuid.UUID) (bool, error) {
	return sqlc.New(s.db).CategoryIsTopLevel(ctx, id)
}

// AnswerStat is one (question, locale) line of a completed match.
type AnswerStat struct {
	QuestionID    uuid.UUID `json:"question_id"`
	Locale        string    `json:"locale"`
	Asked         int64     `json:"asked"`
	Correct       int64     `json:"correct"`
	TimedOut      int64     `json:"timed_out"`
	SumResponseMS int64     `json:"sum_response_ms"`
}

// RecordStats adds a completed match's answers to the question statistics
// (schema-content §5) exactly once per match; from then on the questions
// count as used (§4.3).
func (s *Service) RecordStats(ctx context.Context, matchID uuid.UUID, stats []AnswerStat, at time.Time) error {
	return postgres.InTx(ctx, s.db, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		n, err := q.MarkStatsApplied(ctx, sqlc.MarkStatsAppliedParams{MatchID: matchID, AppliedAt: at})
		if err != nil || n == 0 {
			return err // already counted
		}
		for _, st := range stats {
			if err := q.AddQuestionStats(ctx, sqlc.AddQuestionStatsParams{
				QuestionID: st.QuestionID, Locale: st.Locale, Asked: st.Asked, Correct: st.Correct,
				TimedOut: st.TimedOut, SumResponseMs: st.SumResponseMS, At: &at,
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

func nonNil(ids []uuid.UUID) []uuid.UUID {
	if ids == nil {
		return []uuid.UUID{}
	}
	return ids
}
