// Package service holds creator's use cases: player-made quizzes around
// a content question set, their review and discovery (schema-platform
// §5, category-modes §6, D-023 / D-031 / D-044).
package service

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/content"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/creator/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
)

// Questions is what creator needs from content.
type Questions interface {
	CreateCreatorQuestion(ctx context.Context, author, category uuid.UUID, locale string, q content.CreatorQuestion) (uuid.UUID, error)
	SaveQuestionSet(ctx context.Context, id *uuid.UUID, kind string, createdBy *uuid.UUID, questions []uuid.UUID) (uuid.UUID, error)
	SetQuestionIDs(ctx context.Context, set uuid.UUID) ([]uuid.UUID, error)
	Texts(ctx context.Context, ids []uuid.UUID, locales []string) (map[uuid.UUID]map[string]content.QuestionText, error)
}

// Users reads account tiers (identity).
type Users interface {
	User(ctx context.Context, id uuid.UUID) (identity.User, error)
}

// Cards reads authors' public cards (profile).
type Cards interface {
	Cards(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]profile.Card, error)
}

// Deps are the service's collaborators.
type Deps struct {
	DB        *pgxpool.Pool
	Questions Questions
	Users     Users
	Cards     Cards
	Clock     clock.Clock
}

// Service implements creator.
type Service struct {
	d Deps
}

// New builds the service.
func New(d Deps) *Service { return &Service{d: d} }

// Quiz rules (D-031; 1–30 questions since 2026-10-24).
const (
	MaxQuizzes   = 20
	MinQuestions = 1
	MaxQuestions = 30
)

// Quiz statuses.
const (
	StatusDraft       = "draft"
	StatusSubmitted   = "submitted"
	StatusPublished   = "published"
	StatusRejected    = "rejected"
	StatusUnpublished = "unpublished"
)

// QuizInput is the body of POST /v1/creator/quizzes and PATCH.
type QuizInput struct {
	Title          string                    `json:"title"`
	Description    *string                   `json:"description,omitempty"`
	Locale         string                    `json:"locale"`
	CategoryID     uuid.UUID                 `json:"category_id"`
	AnswerWindowMS int                       `json:"answer_window_ms"`
	Questions      []content.CreatorQuestion `json:"questions"`
}

// Quiz is a quiz; Questions (with the answers) are shown to its author
// and staff only.
type Quiz struct {
	ID              uuid.UUID                 `json:"id"`
	Author          profile.Card              `json:"author"`
	Title           string                    `json:"title"`
	Description     *string                   `json:"description"`
	Locale          string                    `json:"locale"`
	CategoryID      *uuid.UUID                `json:"category_id"`
	AnswerWindowMS  int32                     `json:"answer_window_ms"`
	QuestionCount   int16                     `json:"question_count"`
	Status          string                    `json:"status"`
	RejectionReason *string                   `json:"rejection_reason,omitempty"`
	LikeCount       int32                     `json:"like_count"`
	PlayCount       int32                     `json:"play_count"`
	Liked           bool                      `json:"liked"`
	Version         int32                     `json:"version"`
	UpdatedAt       time.Time                 `json:"updated_at"`
	Questions       []content.CreatorQuestion `json:"questions,omitempty"`
}

func invalid(field, rule string) error {
	return apierror.Validation(apierror.FieldError{Field: field, Rule: rule})
}

func (in *QuizInput) validate() error {
	in.Title = strings.TrimSpace(in.Title)
	switch n := len([]rune(in.Title)); {
	case n < 3 || n > 60:
		return invalid("title", "length")
	case in.Description != nil && len([]rune(*in.Description)) > 300:
		return invalid("description", "max_length")
	case in.Locale != "tr" && in.Locale != "en":
		return invalid("locale", "one_of")
	case in.CategoryID == uuid.Nil:
		return invalid("category_id", "required")
	case !slices.Contains([]int{10_000, 15_000, 20_000}, in.AnswerWindowMS):
		return invalid("answer_window_ms", "one_of")
	case len(in.Questions) < MinQuestions || len(in.Questions) > MaxQuestions:
		return apierror.New(apierror.QuizSizeInvalid).With("min", MinQuestions).With("max", MaxQuestions)
	}
	return nil
}

// full checks that only full accounts author and like (schema-platform §5).
func (s *Service) full(ctx context.Context, user uuid.UUID, feature string) error {
	u, err := s.d.Users.User(ctx, user)
	if err != nil {
		return err
	}
	if u.Tier != identity.TierFull {
		return apierror.New(apierror.AccountUpgradeRequired).With("feature", feature)
	}
	return nil
}

// storeQuestions writes the author's questions and the set holding them.
func (s *Service) storeQuestions(ctx context.Context, author uuid.UUID, set *uuid.UUID, in *QuizInput) (uuid.UUID, error) {
	qids := make([]uuid.UUID, len(in.Questions))
	for i, q := range in.Questions {
		id, err := s.d.Questions.CreateCreatorQuestion(ctx, author, in.CategoryID, in.Locale, q)
		if err != nil {
			if e := apierror.From(err); e.Code == apierror.ValidationFailed {
				return uuid.Nil, e.With("question", i+1)
			}
			return uuid.Nil, err
		}
		qids[i] = id
	}
	return s.d.Questions.SaveQuestionSet(ctx, set, "creator", &author, qids)
}

// Create stores a draft quiz (max 20 per author, 1–30 questions).
func (s *Service) Create(ctx context.Context, author uuid.UUID, in QuizInput) (Quiz, error) {
	if err := s.full(ctx, author, "creator_quiz"); err != nil {
		return Quiz{}, err
	}
	if err := in.validate(); err != nil {
		return Quiz{}, err
	}
	q := sqlc.New(s.d.DB)
	n, err := q.CountAuthorQuizzes(ctx, author)
	if err != nil {
		return Quiz{}, err
	}
	if n >= MaxQuizzes {
		return Quiz{}, apierror.New(apierror.QuizLimitReached)
	}
	set, err := s.storeQuestions(ctx, author, nil, &in)
	if err != nil {
		return Quiz{}, err
	}
	id := ids.New()
	if err := q.InsertQuiz(ctx, sqlc.InsertQuizParams{
		ID: id, Author: author, Locale: in.Locale, Title: in.Title, Description: in.Description, CategoryID: &in.CategoryID, SetID: set,
		WindowMs: int32(in.AnswerWindowMS), QuestionCount: int16(len(in.Questions)), Now: s.d.Clock.Now(), //nolint:gosec // G115: validated
	}); err != nil {
		return Quiz{}, err
	}
	return s.Get(ctx, author, id, false)
}

// owned loads the author's quiz for a change.
func owned(ctx context.Context, q *sqlc.Queries, author, id uuid.UUID) (sqlc.CreatorQuiz, error) {
	z, err := q.GetQuizForUpdate(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && z.AuthorUserID != author) {
		return z, apierror.New(apierror.NotFound)
	}
	return z, err
}

// Update replaces a quiz's details and questions (If-Match: version). A
// submitted or published quiz goes back to review (schema-platform §5).
func (s *Service) Update(ctx context.Context, author, id uuid.UUID, version *int32, in QuizInput) (Quiz, error) {
	if err := in.validate(); err != nil {
		return Quiz{}, err
	}
	z, err := sqlc.New(s.d.DB).GetQuiz(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && z.AuthorUserID != author) {
		return Quiz{}, apierror.New(apierror.NotFound)
	}
	if err != nil {
		return Quiz{}, err
	}
	if version != nil && *version != z.Version {
		return Quiz{}, apierror.New(apierror.VersionConflict).With("version", z.Version)
	}
	if in.Locale != z.Locale {
		return Quiz{}, invalid("locale", "immutable")
	}
	if _, err := s.storeQuestions(ctx, author, &z.QuestionSetID, &in); err != nil {
		return Quiz{}, err
	}
	now := s.d.Clock.Now()
	err = postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		cur, err := owned(ctx, q, author, id)
		if err != nil {
			return err
		}
		status, submitted := cur.Status, cur.SubmittedAt
		if status == StatusPublished || status == StatusSubmitted {
			status, submitted = StatusSubmitted, &now
		}
		return q.UpdateQuiz(ctx, sqlc.UpdateQuizParams{
			ID: id, Title: in.Title, Description: in.Description, CategoryID: &in.CategoryID, WindowMs: int32(in.AnswerWindowMS), //nolint:gosec // G115: validated
			QuestionCount: int16(len(in.Questions)), Status: status, SubmittedAt: submitted, Now: now, //nolint:gosec // G115: ≤ 30
		})
	})
	if err != nil {
		return Quiz{}, err
	}
	return s.Get(ctx, author, id, false)
}

// Delete removes the author's quiz.
func (s *Service) Delete(ctx context.Context, author, id uuid.UUID) error {
	return postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		if _, err := owned(ctx, q, author, id); err != nil {
			return err
		}
		return q.DeleteQuiz(ctx, sqlc.DeleteQuizParams{ID: id, Now: ptr(s.d.Clock.Now())})
	})
}

func ptr[T any](v T) *T { return &v }

// Submit sends a draft, rejected or unpublished quiz to review.
func (s *Service) Submit(ctx context.Context, author, id uuid.UUID) (Quiz, error) {
	now := s.d.Clock.Now()
	err := postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		z, err := owned(ctx, q, author, id)
		if err != nil {
			return err
		}
		if z.Status == StatusSubmitted || z.Status == StatusPublished {
			return apierror.New(apierror.MatchStateInvalid).With("reason", "quiz_"+z.Status)
		}
		return q.SetQuizStatus(ctx, sqlc.SetQuizStatusParams{ID: id, Status: StatusSubmitted, SubmittedAt: &now, Now: now})
	})
	if err != nil {
		return Quiz{}, err
	}
	return s.Get(ctx, author, id, false)
}

// Review publishes or rejects a submitted quiz; staff never review their
// own quiz.
func (s *Service) Review(ctx context.Context, staff, id uuid.UUID, approve bool, reason string) (Quiz, error) {
	if !approve && strings.TrimSpace(reason) == "" {
		return Quiz{}, invalid("reason", "required")
	}
	now := s.d.Clock.Now()
	err := postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		z, err := q.GetQuizForUpdate(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return apierror.New(apierror.NotFound)
		}
		switch {
		case err != nil:
			return err
		case z.AuthorUserID == staff:
			return apierror.New(apierror.SelfReviewForbidden)
		case z.Status != StatusSubmitted:
			return apierror.New(apierror.MatchStateInvalid).With("reason", "quiz_"+z.Status)
		}
		p := sqlc.SetQuizStatusParams{ID: id, Status: StatusPublished, SubmittedAt: z.SubmittedAt, PublishedAt: &now, ReviewedBy: &staff, Now: now}
		if !approve {
			p.Status, p.PublishedAt, p.RejectionReason = StatusRejected, nil, &reason
		}
		return q.SetQuizStatus(ctx, p)
	})
	if err != nil {
		return Quiz{}, err
	}
	return s.Get(ctx, staff, id, true)
}

// Get returns a quiz: everything to its author and staff, published
// quizzes to everyone else.
func (s *Service) Get(ctx context.Context, viewer, id uuid.UUID, staff bool) (Quiz, error) {
	q := sqlc.New(s.d.DB)
	z, err := q.GetQuiz(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Quiz{}, apierror.New(apierror.NotFound)
	}
	if err != nil {
		return Quiz{}, err
	}
	full := staff || z.AuthorUserID == viewer
	if !full && z.Status != StatusPublished {
		return Quiz{}, apierror.New(apierror.NotFound)
	}
	out, err := s.views(ctx, viewer, []sqlc.CreatorQuiz{z})
	if err != nil {
		return Quiz{}, err
	}
	v := out[0]
	if full {
		if v.Questions, err = s.questions(ctx, &z); err != nil {
			return Quiz{}, err
		}
	}
	return v, nil
}

// questions rebuilds the author's questions from content.
func (s *Service) questions(ctx context.Context, z *sqlc.CreatorQuiz) ([]content.CreatorQuestion, error) {
	qids, err := s.d.Questions.SetQuestionIDs(ctx, z.QuestionSetID)
	if err != nil {
		return nil, err
	}
	texts, err := s.d.Questions.Texts(ctx, qids, []string{z.Locale})
	if err != nil {
		return nil, err
	}
	out := make([]content.CreatorQuestion, 0, len(qids))
	for _, id := range qids {
		t := texts[id][z.Locale]
		cq := content.CreatorQuestion{Type: "multiple_choice", Prompt: t.Prompt}
		if len(t.Order) == 2 {
			cq.Type = "true_false"
		}
		for i, o := range t.Order {
			cq.Options = append(cq.Options, t.Options[o])
			if o == t.Correct {
				cq.Correct = i + 1
			}
		}
		out = append(out, cq)
	}
	return out, nil
}

func (s *Service) views(ctx context.Context, viewer uuid.UUID, rows []sqlc.CreatorQuiz) ([]Quiz, error) {
	authors := make([]uuid.UUID, len(rows))
	for i := range rows {
		authors[i] = rows[i].AuthorUserID
	}
	cards, err := s.d.Cards.Cards(ctx, authors)
	if err != nil {
		return nil, err
	}
	q := sqlc.New(s.d.DB)
	out := make([]Quiz, len(rows))
	for i := range rows {
		z := &rows[i]
		liked, err := q.Liked(ctx, sqlc.LikedParams{QuizID: z.ID, UserID: viewer})
		if err != nil {
			return nil, err
		}
		out[i] = Quiz{ID: z.ID, Author: cards[z.AuthorUserID], Title: z.Title, Description: z.Description, Locale: z.Locale,
			CategoryID: z.CategoryID, AnswerWindowMS: z.AnswerWindowMs, QuestionCount: z.QuestionCount, Status: z.Status,
			RejectionReason: z.RejectionReason, LikeCount: z.LikeCount, PlayCount: z.PlayCount, Liked: liked, Version: z.Version,
			UpdatedAt: z.UpdatedAt}
	}
	return out, nil
}

// Mine lists the author's quizzes, newest change first.
func (s *Service) Mine(ctx context.Context, author uuid.UUID) ([]Quiz, error) {
	rows, err := sqlc.New(s.d.DB).AuthorQuizzes(ctx, author)
	if err != nil {
		return nil, err
	}
	return s.views(ctx, author, rows)
}

// Published is public discovery: published quizzes, most liked first.
func (s *Service) Published(ctx context.Context, viewer uuid.UUID, locale *string, category *uuid.UUID, limit, offset int) ([]Quiz, error) {
	rows, err := sqlc.New(s.d.DB).PublishedQuizzes(ctx, sqlc.PublishedQuizzesParams{
		Locale: locale, CategoryID: category, Lim: int32(limit), Off: int32(offset), //nolint:gosec // G115: bounded
	})
	if err != nil {
		return nil, err
	}
	return s.views(ctx, viewer, rows)
}

// Submitted is the staff review queue, oldest first.
func (s *Service) Submitted(ctx context.Context, staff uuid.UUID) ([]Quiz, error) {
	rows, err := sqlc.New(s.d.DB).SubmittedQuizzes(ctx)
	if err != nil {
		return nil, err
	}
	return s.views(ctx, staff, rows)
}

// Like likes (or with like = false unlikes) a published quiz.
func (s *Service) Like(ctx context.Context, user, id uuid.UUID, like bool) error {
	if err := s.full(ctx, user, "creator_quiz"); err != nil {
		return err
	}
	return postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		z, err := q.GetQuizForUpdate(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && z.Status != StatusPublished) {
			return apierror.New(apierror.NotFound)
		}
		if err != nil {
			return err
		}
		var n int64
		delta := int32(1)
		if like {
			n, err = q.Like(ctx, sqlc.LikeParams{QuizID: id, UserID: user, Now: s.d.Clock.Now()})
		} else {
			n, err = q.Unlike(ctx, sqlc.UnlikeParams{QuizID: id, UserID: user})
			delta = -1
		}
		if err != nil || n == 0 {
			return err
		}
		return q.AddLikes(ctx, sqlc.AddLikesParams{ID: id, Delta: delta})
	})
}

// Playable is what a room needs to play a quiz.
type Playable struct {
	SetID     uuid.UUID
	Length    int
	WindowMS  int
	Locale    string
	Author    uuid.UUID
	Published bool
}

// Playable returns a quiz for a room: the author may always use it, others
// only a published one.
func (s *Service) Playable(ctx context.Context, user, id uuid.UUID) (Playable, error) {
	z, err := sqlc.New(s.d.DB).GetQuiz(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && z.AuthorUserID != user && z.Status != StatusPublished) {
		return Playable{}, apierror.New(apierror.NotFound)
	}
	if err != nil {
		return Playable{}, err
	}
	return Playable{SetID: z.QuestionSetID, Length: int(z.QuestionCount), WindowMS: int(z.AnswerWindowMs), Locale: z.Locale,
		Author: z.AuthorUserID, Published: z.Status == StatusPublished}, nil
}

// CountPlay counts a started quiz match.
func (s *Service) CountPlay(ctx context.Context, id uuid.UUID) error {
	return sqlc.New(s.d.DB).CountPlay(ctx, id)
}
