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

// QuestionText is a question's wording in one locale as a match shows it.
// Order lists the options in canonical position order and Correct is the
// correct one: servers only reveal it after a round (round-lifecycle §4).
type QuestionText struct {
	Prompt      string               `json:"prompt"`
	Explanation string               `json:"explanation,omitempty"`
	Options     map[uuid.UUID]string `json:"options"`
	Order       []uuid.UUID          `json:"order"`
	Correct     uuid.UUID            `json:"correct"`
	Category    string               `json:"category"`
}

// Texts returns the wording of questions per question and locale, with
// the name of each question's category (the match screens, websocket.md
// round.started / round.revealed).
func (s *Service) Texts(ctx context.Context, ids []uuid.UUID, locales []string) (map[uuid.UUID]map[string]QuestionText, error) {
	q := sqlc.New(s.db)
	qs, err := q.QuestionTexts(ctx, sqlc.QuestionTextsParams{Ids: nonNil(ids), Locales: locales})
	if err != nil {
		return nil, err
	}
	opts, err := q.OptionTexts(ctx, sqlc.OptionTextsParams{Ids: nonNil(ids), Locales: locales})
	if err != nil {
		return nil, err
	}
	catIDs := map[uuid.UUID]bool{}
	for _, r := range qs {
		catIDs[r.CategoryID] = true
	}
	cats := make([]uuid.UUID, 0, len(catIDs))
	for id := range catIDs {
		cats = append(cats, id)
	}
	names, err := q.CategoryNames(ctx, sqlc.CategoryNamesParams{Ids: cats, Locales: locales})
	if err != nil {
		return nil, err
	}
	catName := map[uuid.UUID]map[string]string{}
	for _, n := range names {
		if catName[n.CategoryID] == nil {
			catName[n.CategoryID] = map[string]string{}
		}
		catName[n.CategoryID][n.Locale] = n.Name
	}
	out := map[uuid.UUID]map[string]QuestionText{}
	for _, r := range qs {
		if out[r.QuestionID] == nil {
			out[r.QuestionID] = map[string]QuestionText{}
		}
		t := QuestionText{Prompt: r.Prompt, Options: map[uuid.UUID]string{}, Category: catName[r.CategoryID][r.Locale]}
		if r.Explanation != nil {
			t.Explanation = *r.Explanation
		}
		out[r.QuestionID][r.Locale] = t
	}
	for _, o := range opts {
		if t, ok := out[o.QuestionID][o.Locale]; ok {
			t.Options[o.OptionID] = o.Text
			t.Order = append(t.Order, o.OptionID)
			if o.IsCorrect {
				t.Correct = o.OptionID
			}
			out[o.QuestionID][o.Locale] = t
		}
	}
	return out, nil
}
