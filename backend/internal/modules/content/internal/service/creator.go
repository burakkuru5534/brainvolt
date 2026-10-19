package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/content/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/content/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
)

// CreatorQuestion is one question an author writes for a quiz.
type CreatorQuestion struct {
	Type    domain.QuestionType `json:"type"`
	Prompt  string              `json:"prompt"`
	Options []string            `json:"options"`
	// Correct is the 1-based position of the correct option.
	Correct int `json:"correct"`
}

// creatorDifficulty is the difficulty creator questions are stored with
// (quizzes keep the author's order, so it only feeds scoring).
const creatorDifficulty = 3

// CreateCreatorQuestion stores an author's question in the creator_only
// pool under the quiz's category, unreviewed (schema-platform §5).
func (s *Service) CreateCreatorQuestion(ctx context.Context, author, category uuid.UUID, locale string, cq CreatorQuestion) (uuid.UUID, error) {
	q := sqlc.New(s.db)
	sub, err := q.FirstSubcategory(ctx, category)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, apierror.Validation(apierror.FieldError{Field: "category_id", Rule: "active_category"})
	}
	if err != nil {
		return uuid.Nil, err
	}
	source, err := q.CreatorSource(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	return s.Create(ctx, domain.Draft{
		SubcategoryID: sub, Type: cq.Type, Difficulty: creatorDifficulty, OriginalLocale: locale, SourceID: &source,
		Correct: cq.Correct, Texts: map[string]domain.Text{locale: {Prompt: cq.Prompt, Options: cq.Options, Origin: domain.OriginOriginal}},
	}, CreateOptions{Actor: &author, Status: domain.StatusDraft, AuthorUserID: &author, Creator: true})
}

// SaveQuestionSet stores the questions of a set in order: a new set when
// id is nil, otherwise its items are replaced. It returns the set id.
func (s *Service) SaveQuestionSet(ctx context.Context, id *uuid.UUID, kind string, createdBy *uuid.UUID, questions []uuid.UUID) (uuid.UUID, error) {
	setID := ids.New()
	if id != nil {
		setID = *id
	}
	err := postgres.InTx(ctx, s.db, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		if id == nil {
			if err := q.InsertQuestionSet(ctx, sqlc.InsertQuestionSetParams{ID: setID, Kind: kind, CreatedBy: createdBy, Now: time.Now().UTC()}); err != nil {
				return err
			}
		} else if err := q.DeleteSetItems(ctx, setID); err != nil {
			return err
		}
		for i, qid := range questions {
			if err := q.InsertSetItem(ctx, sqlc.InsertSetItemParams{SetID: setID, Position: int16(i + 1), QuestionID: qid}); err != nil {
				return err
			}
		}
		return nil
	})
	return setID, err
}

// SetQuestionIDs lists a set's questions in order.
func (s *Service) SetQuestionIDs(ctx context.Context, set uuid.UUID) ([]uuid.UUID, error) {
	return sqlc.New(s.db).SetQuestionIDs(ctx, set)
}
