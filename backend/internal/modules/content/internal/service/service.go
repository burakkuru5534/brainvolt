// Package service holds content's use cases (docs/domain/schema-content.md):
// the localized catalogue with eligible-question counts, credits, the
// moderator question workflow (create, edit, submit, review, retire) with
// revisions and verification events, and the import used by cmd/seed.
package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/content/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/content/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
)

// Event names (backend-modules.md §3).
const (
	EventQuestionVerified = "content.QuestionVerified"
	EventQuestionRejected = "content.QuestionRejected"
)

// QuestionReviewed is the payload of EventQuestionVerified and
// EventQuestionRejected; Locale is empty for a question-level decision.
type QuestionReviewed struct {
	QuestionID uuid.UUID `json:"question_id"`
	Locale     string    `json:"locale,omitempty"`
	Revision   int32     `json:"revision"`
	Reason     string    `json:"reason,omitempty"`
}

// Service implements content's use cases.
type Service struct {
	db     *pgxpool.Pool
	clock  clock.Clock
	events *events.Bus
}

// New returns a service.
func New(db *pgxpool.Pool, c clock.Clock, bus *events.Bus) *Service {
	return &Service{db: db, clock: c, events: bus}
}

func (s *Service) now() time.Time { return s.clock.Now().UTC() }

// ---------- Catalogue ----------

// Counts are the questions a match may use (schema-content §3): ranked
// excludes true/false (D-024).
type Counts struct {
	Ranked int64 `json:"ranked"`
	Casual int64 `json:"casual"`
}

// Subcategory is one entry of the catalogue.
type Subcategory struct {
	ID          uuid.UUID `json:"id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	Eligible    Counts    `json:"eligible"`
}

// Category is a top-level entry of the catalogue.
type Category struct {
	ID            uuid.UUID     `json:"id"`
	Slug          string        `json:"slug"`
	IconKey       string        `json:"icon_key"`
	Name          string        `json:"name"`
	Description   *string       `json:"description"`
	Eligible      Counts        `json:"eligible"`
	Subcategories []Subcategory `json:"subcategories"`
}

// Catalog is the body of GET /v1/categories.
type Catalog struct {
	Locale     string     `json:"locale"`
	Categories []Category `json:"categories"`
}

// CatalogETag is a strong validator of the catalogue in locale: it changes
// with any edit of the taxonomy, a question or a translation.
func (s *Service) CatalogETag(ctx context.Context, locale string) (string, error) {
	v, err := sqlc.New(s.db).ContentVersion(ctx)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(fmt.Appendf(nil, "%s|%d|%d", locale, v.Version.UnixNano(), v.Questions))
	return `"` + hex.EncodeToString(sum[:12]) + `"`, nil
}

// Catalog returns the active categories and sub-categories in locale with
// the number of eligible questions per mode.
func (s *Service) Catalog(ctx context.Context, locale string) (Catalog, error) {
	q := sqlc.New(s.db)
	cats, err := q.ListCategories(ctx, locale)
	if err != nil {
		return Catalog{}, err
	}
	subs, err := q.ListSubcategories(ctx, locale)
	if err != nil {
		return Catalog{}, err
	}
	counts, err := q.CountEligible(ctx, locale)
	if err != nil {
		return Catalog{}, err
	}
	bySub := map[uuid.UUID]Counts{}
	for _, c := range counts {
		bySub[c.SubcategoryID] = Counts{Ranked: c.Ranked, Casual: c.Casual}
	}
	out := Catalog{Locale: locale, Categories: make([]Category, 0, len(cats))}
	index := map[uuid.UUID]int{}
	for _, c := range cats {
		index[c.ID] = len(out.Categories)
		out.Categories = append(out.Categories, Category{
			ID: c.ID, Slug: c.Slug, IconKey: c.IconKey, Name: c.Name, Description: c.Description,
			Subcategories: []Subcategory{},
		})
	}
	for _, sc := range subs {
		i, ok := index[sc.CategoryID]
		if !ok {
			continue
		}
		n := bySub[sc.ID]
		cat := &out.Categories[i]
		cat.Subcategories = append(cat.Subcategories, Subcategory{
			ID: sc.ID, Slug: sc.Slug, Name: sc.Name, Description: sc.Description, Eligible: n,
		})
		cat.Eligible.Ranked += n.Ranked
		cat.Eligible.Casual += n.Casual
	}
	return out, nil
}

// ---------- Credits ----------

// Credit is one source on the in-app Credits screen (D-006, CC BY-SA).
type Credit struct {
	Name            string  `json:"name"`
	URL             *string `json:"url"`
	LicenseCode     string  `json:"license_code"`
	LicenseURL      *string `json:"license_url"`
	AttributionText string  `json:"attribution_text"`
	ShareAlike      bool    `json:"share_alike"`
}

// Credits lists every content source.
func (s *Service) Credits(ctx context.Context) ([]Credit, error) {
	rows, err := sqlc.New(s.db).ListSources(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Credit, 0, len(rows))
	for i := range rows {
		r := &rows[i]
		out = append(out, Credit{
			Name: r.Name, URL: r.Url, LicenseCode: r.LicenseCode, LicenseURL: r.LicenseUrl,
			AttributionText: r.AttributionText, ShareAlike: r.ShareAlike,
		})
	}
	return out, nil
}

// ---------- Moderator views ----------

// Option is an answer option with its correctness (moderators only).
type Option struct {
	ID        uuid.UUID `json:"id"`
	Position  int16     `json:"position"`
	IsCorrect bool      `json:"is_correct"`
}

// Translation is a question's text in one locale.
type Translation struct {
	Prompt          string        `json:"prompt"`
	Explanation     *string       `json:"explanation"`
	Options         []string      `json:"options"`
	Origin          domain.Origin `json:"origin"`
	Status          domain.Status `json:"status"`
	VerifiedBy      *uuid.UUID    `json:"verified_by"`
	VerifiedAt      *time.Time    `json:"verified_at"`
	RejectionReason *string       `json:"rejection_reason"`
}

// Revision is one stored snapshot (schema-content §4.3).
type Revision struct {
	Revision  int32      `json:"revision"`
	CreatedBy *uuid.UUID `json:"created_by"`
	CreatedAt time.Time  `json:"created_at"`
}

// VerificationEvent is one status change.
type VerificationEvent struct {
	Locale     *string    `json:"locale"`
	FromStatus string     `json:"from_status"`
	ToStatus   string     `json:"to_status"`
	ActorID    *uuid.UUID `json:"actor_id"`
	Reason     *string    `json:"reason"`
	Revision   int32      `json:"revision"`
	CreatedAt  time.Time  `json:"created_at"`
}

// Stats are the answer statistics of one locale (§5).
type Stats struct {
	TimesAsked    int64 `json:"times_asked"`
	TimesCorrect  int64 `json:"times_correct"`
	TimesTimedOut int64 `json:"times_timed_out"`
	SumResponseMS int64 `json:"sum_response_ms"`
}

// Question is the full moderator view (GET /v1/questions/{id}).
type Question struct {
	ID                   uuid.UUID              `json:"id"`
	SubcategoryID        uuid.UUID              `json:"subcategory_id"`
	QuestionType         domain.QuestionType    `json:"question_type"`
	Difficulty           int16                  `json:"difficulty"`
	DifficultyCalibrated *float64               `json:"difficulty_calibrated"`
	OriginalLocale       string                 `json:"original_locale"`
	Pool                 string                 `json:"pool"`
	SourceID             uuid.UUID              `json:"source_id"`
	SourceRef            *string                `json:"source_ref"`
	AuthorUserID         *uuid.UUID             `json:"author_user_id"`
	Status               domain.Status          `json:"status"`
	VerifiedBy           *uuid.UUID             `json:"verified_by"`
	VerifiedAt           *time.Time             `json:"verified_at"`
	RejectionReason      *string                `json:"rejection_reason"`
	RetiredAt            *time.Time             `json:"retired_at"`
	Revision             int32                  `json:"revision"`
	Used                 bool                   `json:"used"`
	CreatedAt            time.Time              `json:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at"`
	Options              []Option               `json:"options"`
	Translations         map[string]Translation `json:"translations"`
	Revisions            []Revision             `json:"revisions"`
	Events               []VerificationEvent    `json:"events"`
	Stats                map[string]Stats       `json:"stats"`
}

// QuestionSummary is one row of the moderator listing.
type QuestionSummary struct {
	ID             uuid.UUID                `json:"id"`
	SubcategoryID  uuid.UUID                `json:"subcategory_id"`
	QuestionType   domain.QuestionType      `json:"question_type"`
	Difficulty     int16                    `json:"difficulty"`
	OriginalLocale string                   `json:"original_locale"`
	Status         domain.Status            `json:"status"`
	Translations   map[string]domain.Status `json:"translations"`
	Prompt         string                   `json:"prompt"`
	Revision       int32                    `json:"revision"`
	RetiredAt      *time.Time               `json:"retired_at"`
	CreatedAt      time.Time                `json:"created_at"`
}

// QuestionPage is the body of GET /v1/questions.
type QuestionPage struct {
	Items      []QuestionSummary `json:"items"`
	NextCursor *string           `json:"next_cursor"`
}

// ListFilter selects questions for the moderator listing. Locale also picks
// the prompt shown (else the original wording).
type ListFilter struct {
	Status            *domain.Status
	TranslationStatus *domain.Status
	SubcategoryID     *uuid.UUID
	Locale            *string
	IncludeRetired    bool
	Cursor            *uuid.UUID
	Limit             int
}

// List returns one page of questions, newest first.
func (s *Service) List(ctx context.Context, f ListFilter) (QuestionPage, error) {
	q := sqlc.New(s.db)
	rows, err := q.ListQuestions(ctx, sqlc.ListQuestionsParams{
		Status: (*string)(f.Status), TranslationStatus: (*string)(f.TranslationStatus),
		SubcategoryID: f.SubcategoryID, Locale: f.Locale, IncludeRetired: f.IncludeRetired,
		Cursor: f.Cursor, RowLimit: int32(f.Limit + 1), //nolint:gosec // G115: limit is validated to 1–100
	})
	if err != nil {
		return QuestionPage{}, err
	}
	page := QuestionPage{Items: make([]QuestionSummary, 0, len(rows))}
	if len(rows) > f.Limit {
		rows = rows[:f.Limit]
		next := rows[len(rows)-1].ID.String()
		page.NextCursor = &next
	}
	idList := make([]uuid.UUID, len(rows))
	for i := range rows {
		idList[i] = rows[i].ID
	}
	statuses, err := q.ListQuestionTranslationStatuses(ctx, idList)
	if err != nil {
		return QuestionPage{}, err
	}
	byQuestion := map[uuid.UUID]map[string]domain.Status{}
	for _, st := range statuses {
		if byQuestion[st.QuestionID] == nil {
			byQuestion[st.QuestionID] = map[string]domain.Status{}
		}
		byQuestion[st.QuestionID][st.Locale] = domain.Status(st.VerificationStatus)
	}
	for i := range rows {
		r := &rows[i]
		page.Items = append(page.Items, QuestionSummary{
			ID: r.ID, SubcategoryID: r.SubcategoryID, QuestionType: domain.QuestionType(r.QuestionType),
			Difficulty: r.Difficulty, OriginalLocale: r.OriginalLocale, Status: domain.Status(r.VerificationStatus),
			Translations: byQuestion[r.ID], Prompt: r.Prompt, Revision: r.Revision, RetiredAt: r.RetiredAt,
			CreatedAt: r.CreatedAt,
		})
	}
	return page, nil
}

// Get returns the full moderator view of a question.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (Question, error) {
	q := sqlc.New(s.db)
	row, err := q.GetQuestion(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Question{}, apierror.New(apierror.NotFound)
	}
	if err != nil {
		return Question{}, err
	}
	out := questionView(row)
	if out.Used, err = q.QuestionUsed(ctx, id); err != nil {
		return Question{}, err
	}
	snap, err := load(ctx, q, id)
	if err != nil {
		return Question{}, err
	}
	out.Options = snap.options
	out.Translations = snap.translations
	revs, err := q.ListRevisions(ctx, id)
	if err != nil {
		return Question{}, err
	}
	out.Revisions = make([]Revision, 0, len(revs))
	for _, r := range revs {
		out.Revisions = append(out.Revisions, Revision{Revision: r.Revision, CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt})
	}
	evs, err := q.ListVerificationEvents(ctx, id)
	if err != nil {
		return Question{}, err
	}
	out.Events = make([]VerificationEvent, 0, len(evs))
	for _, e := range evs {
		out.Events = append(out.Events, VerificationEvent{
			Locale: e.Locale, FromStatus: e.FromStatus, ToStatus: e.ToStatus, ActorID: e.ActorID,
			Reason: e.Reason, Revision: e.Revision, CreatedAt: e.CreatedAt,
		})
	}
	stats, err := q.ListQuestionStats(ctx, id)
	if err != nil {
		return Question{}, err
	}
	out.Stats = map[string]Stats{}
	for _, st := range stats {
		out.Stats[st.Locale] = Stats{
			TimesAsked: st.TimesAsked, TimesCorrect: st.TimesCorrect,
			TimesTimedOut: st.TimesTimedOut, SumResponseMS: st.SumResponseMs,
		}
	}
	return out, nil
}

func questionView(r sqlc.ContentQuestion) Question {
	var calibrated *float64
	if f, err := r.DifficultyCalibrated.Float64Value(); err == nil && f.Valid {
		calibrated = &f.Float64
	}
	return Question{
		ID: r.ID, SubcategoryID: r.SubcategoryID, QuestionType: domain.QuestionType(r.QuestionType),
		Difficulty: r.Difficulty, DifficultyCalibrated: calibrated, OriginalLocale: r.OriginalLocale,
		Pool: r.Pool, SourceID: r.SourceID, SourceRef: r.SourceRef, AuthorUserID: r.AuthorUserID,
		Status: domain.Status(r.VerificationStatus), VerifiedBy: r.VerifiedBy, VerifiedAt: r.VerifiedAt,
		RejectionReason: r.RejectionReason, RetiredAt: r.RetiredAt, Revision: r.Revision,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

// state is a question's options and translations as stored.
type state struct {
	options      []Option
	translations map[string]Translation
}

func (st state) correct() int {
	for _, o := range st.options {
		if o.IsCorrect {
			return int(o.Position)
		}
	}
	return 0
}

func load(ctx context.Context, q *sqlc.Queries, id uuid.UUID) (state, error) {
	opts, err := q.ListOptions(ctx, id)
	if err != nil {
		return state{}, err
	}
	st := state{options: make([]Option, 0, len(opts)), translations: map[string]Translation{}}
	position := map[uuid.UUID]int{}
	for _, o := range opts {
		position[o.ID] = int(o.Position)
		st.options = append(st.options, Option{ID: o.ID, Position: o.Position, IsCorrect: o.IsCorrect})
	}
	trs, err := q.ListQuestionTranslations(ctx, id)
	if err != nil {
		return state{}, err
	}
	for i := range trs {
		t := &trs[i]
		st.translations[t.Locale] = Translation{
			Prompt: t.Prompt, Explanation: t.Explanation, Options: make([]string, len(opts)),
			Origin: domain.Origin(t.Origin), Status: domain.Status(t.VerificationStatus),
			VerifiedBy: t.VerifiedBy, VerifiedAt: t.VerifiedAt, RejectionReason: t.RejectionReason,
		}
	}
	ots, err := q.ListOptionTranslations(ctx, id)
	if err != nil {
		return state{}, err
	}
	for _, ot := range ots {
		if t, ok := st.translations[ot.Locale]; ok {
			t.Options[position[ot.OptionID]-1] = ot.Text
		}
	}
	return st, nil
}

// ---------- Create ----------

// CreateOptions control who and what creates a question.
type CreateOptions struct {
	Actor *uuid.UUID
	// Status is draft (moderator create) or in_review (import, §6).
	Status       domain.Status
	AuthorUserID *uuid.UUID
}

// Create stores a new question with its options, translations, revision 1
// and the creation event; it returns the new id.
func (s *Service) Create(ctx context.Context, d domain.Draft, o CreateOptions) (uuid.UUID, error) {
	d.Normalize()
	if err := d.Validate(); err != nil {
		return uuid.Nil, err
	}
	if err := s.checkRefs(ctx, d.SubcategoryID, *d.SourceID); err != nil {
		return uuid.Nil, err
	}
	fp := domain.Fingerprint(d.OriginalLocale, d.Texts[d.OriginalLocale])
	if err := s.checkDuplicate(ctx, fp, uuid.Nil); err != nil {
		return uuid.Nil, err
	}
	id := ids.New()
	now := s.now()
	err := postgres.InTx(ctx, s.db, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		var ref *string
		if d.SourceRef != "" {
			ref = &d.SourceRef
		}
		if err := q.InsertQuestion(ctx, sqlc.InsertQuestionParams{
			ID: id, SubcategoryID: d.SubcategoryID, QuestionType: string(d.Type),
			Difficulty:     int16(d.Difficulty), //nolint:gosec // G115: validated 1–5
			OriginalLocale: d.OriginalLocale, SourceID: *d.SourceID, SourceRef: ref,
			AuthorUserID: o.AuthorUserID, Fingerprint: fp, VerificationStatus: string(o.Status), Now: now,
		}); err != nil {
			return err
		}
		optionIDs := make([]uuid.UUID, d.Type.OptionCount())
		for i := range optionIDs {
			optionIDs[i] = ids.New()
			if err := q.InsertOption(ctx, sqlc.InsertOptionParams{
				ID: optionIDs[i], QuestionID: id, Position: int16(i + 1), IsCorrect: i+1 == d.Correct,
			}); err != nil {
				return err
			}
		}
		for _, l := range slices.Sorted(maps.Keys(d.Texts)) {
			if err := writeText(ctx, q, id, optionIDs, l, d.Texts[l], o.Status, now); err != nil {
				return err
			}
		}
		if err := q.InsertVerificationEvent(ctx, event(id, nil, "new", string(o.Status), o.Actor, nil, 1, now)); err != nil {
			return err
		}
		return snapshot(ctx, q, id, 1, o.Actor, now)
	})
	if isUniqueViolation(err) {
		return uuid.Nil, s.checkDuplicate(ctx, fp, uuid.Nil)
	}
	return id, err
}

func writeText(ctx context.Context, q *sqlc.Queries, id uuid.UUID, optionIDs []uuid.UUID, locale string, t domain.Text, status domain.Status, now time.Time) error {
	var expl *string
	if t.Explanation != "" {
		expl = &t.Explanation
	}
	if err := q.UpsertQuestionTranslation(ctx, sqlc.UpsertQuestionTranslationParams{
		QuestionID: id, Locale: locale, Prompt: t.Prompt, Explanation: expl, Origin: string(t.Origin),
		VerificationStatus: string(status), Now: now,
	}); err != nil {
		return err
	}
	for i, text := range t.Options {
		if err := q.UpsertOptionTranslation(ctx, sqlc.UpsertOptionTranslationParams{
			OptionID: optionIDs[i], Locale: locale, Text: text,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) checkRefs(ctx context.Context, sub, source uuid.UUID) error {
	q := sqlc.New(s.db)
	ok, err := q.SourceExists(ctx, source)
	if err != nil {
		return err
	}
	if !ok {
		return apierror.New(apierror.QuestionSourceRequired).With("source_id", source)
	}
	if ok, err = q.SubcategoryExists(ctx, sub); err != nil {
		return err
	}
	if !ok {
		var f apierror.Fields
		f.Add("subcategory_id", "not_found")
		return f.Err()
	}
	return nil
}

// checkDuplicate fails when another question (not self) has fingerprint fp.
func (s *Service) checkDuplicate(ctx context.Context, fp string, self uuid.UUID) error {
	other, err := sqlc.New(s.db).QuestionIDByFingerprint(ctx, fp)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && other == self) {
		return nil
	}
	if err != nil {
		return err
	}
	return apierror.New(apierror.QuestionDuplicate).With("question_id", other)
}

func isUniqueViolation(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "23505"
}

// ---------- Update ----------

// Patch is an edit of a question; nil / empty fields stay as they are.
// Texts replace the whole translation of their locale.
type Patch struct {
	SubcategoryID *uuid.UUID
	Difficulty    *int
	Correct       *int
	Texts         map[string]domain.Text
}

// Update applies p (schema-content §4.3): every edit bumps the revision and
// stores a snapshot; a verified question goes back to review; changing the
// correct option of a used question is QUESTION_IN_USE.
func (s *Service) Update(ctx context.Context, actor, id uuid.UUID, p Patch) error {
	if p.SubcategoryID == nil && p.Difficulty == nil && p.Correct == nil && len(p.Texts) == 0 {
		var f apierror.Fields
		f.Add("body", "empty")
		return f.Err()
	}
	now := s.now()
	err := postgres.InTx(ctx, s.db, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		row, err := q.GetQuestionForUpdate(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return apierror.New(apierror.NotFound)
		}
		if err != nil {
			return err
		}
		if row.RetiredAt != nil {
			return apierror.New(apierror.QuestionStateInvalid).With("status", "retired")
		}
		cur, err := load(ctx, q, id)
		if err != nil {
			return err
		}
		qt := domain.QuestionType(row.QuestionType)
		d := domain.Draft{
			SubcategoryID: row.SubcategoryID, Type: qt, Difficulty: int(row.Difficulty),
			OriginalLocale: row.OriginalLocale, SourceID: &row.SourceID, Correct: cur.correct(),
			Texts: map[string]domain.Text{},
		}
		for l, t := range cur.translations {
			expl := ""
			if t.Explanation != nil {
				expl = *t.Explanation
			}
			d.Texts[l] = domain.Text{Prompt: t.Prompt, Explanation: expl, Options: t.Options, Origin: t.Origin}
		}
		if p.SubcategoryID != nil {
			d.SubcategoryID = *p.SubcategoryID
		}
		if p.Difficulty != nil {
			d.Difficulty = *p.Difficulty
		}
		structural := p.Correct != nil && *p.Correct != d.Correct
		if p.Correct != nil {
			d.Correct = *p.Correct
		}
		maps.Copy(d.Texts, p.Texts)
		d.Normalize()
		if err := d.Validate(); err != nil {
			return err
		}
		if structural {
			used, err := q.QuestionUsed(ctx, id)
			if err != nil {
				return err
			}
			if used {
				return apierror.New(apierror.QuestionInUse)
			}
		}
		if d.SubcategoryID != row.SubcategoryID {
			if ok, err := q.SubcategoryExists(ctx, d.SubcategoryID); err != nil || !ok {
				if err != nil {
					return err
				}
				var f apierror.Fields
				f.Add("subcategory_id", "not_found")
				return f.Err()
			}
		}
		fp := domain.Fingerprint(d.OriginalLocale, d.Texts[d.OriginalLocale])
		if fp != row.Fingerprint {
			if err := s.checkDuplicate(ctx, fp, id); err != nil {
				return err
			}
		}

		from := domain.Status(row.VerificationStatus)
		to := domain.AfterEdit(from)
		rev := row.Revision + 1
		upd := sqlc.UpdateQuestionParams{
			ID: id, SubcategoryID: d.SubcategoryID, Difficulty: int16(d.Difficulty), //nolint:gosec // G115: validated 1–5
			Fingerprint: fp, VerificationStatus: string(to), Revision: rev, Now: now,
		}
		if to == domain.StatusVerified {
			upd.VerifiedBy, upd.VerifiedAt = row.VerifiedBy, row.VerifiedAt
		}
		if to == domain.StatusRejected {
			upd.RejectionReason = row.RejectionReason
		}
		if err := q.UpdateQuestion(ctx, upd); err != nil {
			return err
		}
		if to != from {
			if err := q.InsertVerificationEvent(ctx, event(id, nil, string(from), string(to), &actor, nil, rev, now)); err != nil {
				return err
			}
		}
		if structural {
			if err := q.ClearOptionCorrect(ctx, id); err != nil {
				return err
			}
			if err := q.SetOptionCorrect(ctx, sqlc.SetOptionCorrectParams{
				QuestionID: id, CorrectPosition: int16(d.Correct), //nolint:gosec // G115: validated ≤ 4
			}); err != nil {
				return err
			}
		}
		optionIDs := make([]uuid.UUID, len(cur.options))
		for i, o := range cur.options {
			optionIDs[i] = o.ID
		}
		// Edited or new translations wait for review unless the whole
		// question is still a draft.
		textStatus := domain.StatusInReview
		if to == domain.StatusDraft {
			textStatus = domain.StatusDraft
		}
		for _, l := range slices.Sorted(maps.Keys(p.Texts)) {
			old, existed := cur.translations[l]
			if err := writeText(ctx, q, id, optionIDs, l, d.Texts[l], textStatus, now); err != nil {
				return err
			}
			fromText := "new"
			if existed {
				fromText = string(old.Status)
			}
			if fromText != string(textStatus) {
				if err := q.InsertVerificationEvent(ctx, event(id, &l, fromText, string(textStatus), &actor, nil, rev, now)); err != nil {
					return err
				}
			}
		}
		return snapshot(ctx, q, id, rev, &actor, now)
	})
	if isUniqueViolation(err) {
		return apierror.New(apierror.QuestionDuplicate)
	}
	return err
}

// ---------- Workflow ----------

// Submit moves a draft question and its draft translations to review.
func (s *Service) Submit(ctx context.Context, actor, id uuid.UUID) error {
	now := s.now()
	return postgres.InTx(ctx, s.db, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		row, err := lockActive(ctx, q, id)
		if err != nil {
			return err
		}
		if !domain.CanSubmit(domain.Status(row.VerificationStatus)) {
			return domain.StateError(domain.Status(row.VerificationStatus))
		}
		if err := q.SetQuestionStatus(ctx, sqlc.SetQuestionStatusParams{ID: id, VerificationStatus: string(domain.StatusInReview), Now: now}); err != nil {
			return err
		}
		if err := q.InsertVerificationEvent(ctx, event(id, nil, row.VerificationStatus, string(domain.StatusInReview), &actor, nil, row.Revision, now)); err != nil {
			return err
		}
		trs, err := q.ListQuestionTranslations(ctx, id)
		if err != nil {
			return err
		}
		for i := range trs {
			t := &trs[i]
			if t.VerificationStatus != string(domain.StatusDraft) {
				continue
			}
			if err := q.SetTranslationStatus(ctx, sqlc.SetTranslationStatusParams{
				QuestionID: id, Locale: t.Locale, VerificationStatus: string(domain.StatusInReview), Now: now,
			}); err != nil {
				return err
			}
			if err := q.InsertVerificationEvent(ctx, event(id, &t.Locale, t.VerificationStatus, string(domain.StatusInReview), &actor, nil, row.Revision, now)); err != nil {
				return err
			}
		}
		return nil
	})
}

// Review is a moderator decision; Locale nil = the question itself.
type Review struct {
	Decision string
	Locale   *string
	Reason   string
}

// Review approves or rejects a question or one translation (D-007). A
// question-level approval also verifies the original-locale translation
// (§2.1); translations are approved separately.
func (s *Service) Review(ctx context.Context, actor, id uuid.UUID, r Review) error {
	var f apierror.Fields
	if r.Decision != domain.DecisionApprove && r.Decision != domain.DecisionReject {
		f.Add("decision", "enum", "allowed", []string{domain.DecisionApprove, domain.DecisionReject})
	}
	if r.Decision == domain.DecisionReject && r.Reason == "" {
		f.Add("reason", "required")
	}
	if r.Locale != nil && !domain.ValidLocale(*r.Locale) {
		f.Add("locale", "enum", "allowed", domain.Locales)
	}
	if err := f.Err(); err != nil {
		return err
	}
	now := s.now()
	return postgres.InTx(ctx, s.db, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		row, err := lockActive(ctx, q, id)
		if err != nil {
			return err
		}
		if r.Locale == nil {
			return s.reviewQuestion(ctx, tx, q, row, actor, r, now)
		}
		return s.reviewTranslation(ctx, tx, q, row, actor, r, now)
	})
}

func (s *Service) reviewQuestion(ctx context.Context, tx pgx.Tx, q *sqlc.Queries, row sqlc.ContentQuestion, actor uuid.UUID, r Review, now time.Time) error {
	from := domain.Status(row.VerificationStatus)
	if !domain.CanReview(from) {
		return domain.StateError(from)
	}
	payload := QuestionReviewed{QuestionID: row.ID, Revision: row.Revision}
	if r.Decision == domain.DecisionReject {
		payload.Reason = r.Reason
		if err := q.SetQuestionStatus(ctx, sqlc.SetQuestionStatusParams{
			ID: row.ID, VerificationStatus: string(domain.StatusRejected), RejectionReason: &r.Reason, Now: now,
		}); err != nil {
			return err
		}
		if err := q.InsertVerificationEvent(ctx, event(row.ID, nil, string(from), string(domain.StatusRejected), &actor, &r.Reason, row.Revision, now)); err != nil {
			return err
		}
		return s.events.PublishTx(ctx, tx, EventQuestionRejected, now, payload)
	}
	// The facts are reviewed against the original wording, so that
	// translation is verified with the question (§2.1).
	trs, err := q.ListQuestionTranslations(ctx, row.ID)
	if err != nil {
		return err
	}
	var original *sqlc.ContentQuestionTranslation
	for i := range trs {
		if trs[i].Locale == row.OriginalLocale {
			original = &trs[i]
		}
	}
	if original == nil || original.VerificationStatus == string(domain.StatusRejected) {
		return apierror.New(apierror.QuestionStateInvalid).With("status", from).With("locale", row.OriginalLocale)
	}
	if err := q.SetQuestionStatus(ctx, sqlc.SetQuestionStatusParams{
		ID: row.ID, VerificationStatus: string(domain.StatusVerified), VerifiedBy: &actor, VerifiedAt: &now, Now: now,
	}); err != nil {
		return err
	}
	if err := q.InsertVerificationEvent(ctx, event(row.ID, nil, string(from), string(domain.StatusVerified), &actor, nil, row.Revision, now)); err != nil {
		return err
	}
	if original.VerificationStatus != string(domain.StatusVerified) {
		if err := q.SetTranslationStatus(ctx, sqlc.SetTranslationStatusParams{
			QuestionID: row.ID, Locale: original.Locale, VerificationStatus: string(domain.StatusVerified),
			VerifiedBy: &actor, VerifiedAt: &now, Now: now,
		}); err != nil {
			return err
		}
		if err := q.InsertVerificationEvent(ctx, event(row.ID, &original.Locale, original.VerificationStatus, string(domain.StatusVerified), &actor, nil, row.Revision, now)); err != nil {
			return err
		}
	}
	return s.events.PublishTx(ctx, tx, EventQuestionVerified, now, payload)
}

func (s *Service) reviewTranslation(ctx context.Context, tx pgx.Tx, q *sqlc.Queries, row sqlc.ContentQuestion, actor uuid.UUID, r Review, now time.Time) error {
	locale := *r.Locale
	if row.VerificationStatus == string(domain.StatusRejected) {
		return domain.StateError(domain.StatusRejected)
	}
	trs, err := q.ListQuestionTranslations(ctx, row.ID)
	if err != nil {
		return err
	}
	idx := slices.IndexFunc(trs, func(t sqlc.ContentQuestionTranslation) bool { return t.Locale == locale })
	if idx < 0 {
		return apierror.New(apierror.NotFound).With("locale", locale)
	}
	from := domain.Status(trs[idx].VerificationStatus)
	if !domain.CanReview(from) {
		return domain.StateError(from).With("locale", locale)
	}
	payload := QuestionReviewed{QuestionID: row.ID, Locale: locale, Revision: row.Revision}
	params := sqlc.SetTranslationStatusParams{QuestionID: row.ID, Locale: locale, Now: now}
	name := EventQuestionVerified
	var reason *string
	if r.Decision == domain.DecisionReject {
		params.VerificationStatus = string(domain.StatusRejected)
		params.RejectionReason = &r.Reason
		reason, payload.Reason, name = &r.Reason, r.Reason, EventQuestionRejected
	} else {
		params.VerificationStatus = string(domain.StatusVerified)
		params.VerifiedBy, params.VerifiedAt = &actor, &now
	}
	if err := q.SetTranslationStatus(ctx, params); err != nil {
		return err
	}
	if err := q.InsertVerificationEvent(ctx, event(row.ID, &locale, string(from), params.VerificationStatus, &actor, reason, row.Revision, now)); err != nil {
		return err
	}
	return s.events.PublishTx(ctx, tx, name, now, payload)
}

// PullToReview moves a verified question back to review (moderation:
// enough wrong_answer reports or a moderator, D-030), which takes it out
// of every selection until it is approved again. actor nil = system.
// Other states are left alone.
func (s *Service) PullToReview(ctx context.Context, actor *uuid.UUID, id uuid.UUID, reason string) error {
	now := s.now()
	return postgres.InTx(ctx, s.db, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		row, err := q.GetQuestionForUpdate(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return apierror.New(apierror.NotFound)
		}
		if err != nil || row.VerificationStatus != string(domain.StatusVerified) || row.RetiredAt != nil {
			return err
		}
		if err := q.SetQuestionStatus(ctx, sqlc.SetQuestionStatusParams{ID: id, VerificationStatus: string(domain.StatusInReview), Now: now}); err != nil {
			return err
		}
		return q.InsertVerificationEvent(ctx, event(id, nil, row.VerificationStatus, string(domain.StatusInReview), actor, &reason, row.Revision, now))
	})
}

// Retire removes a question from all future selection; repeating it is
// harmless.
func (s *Service) Retire(ctx context.Context, actor, id uuid.UUID) error {
	now := s.now()
	return postgres.InTx(ctx, s.db, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		row, err := q.GetQuestionForUpdate(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return apierror.New(apierror.NotFound)
		}
		if err != nil || row.RetiredAt != nil {
			return err
		}
		if err := q.RetireQuestion(ctx, sqlc.RetireQuestionParams{ID: id, Now: &now}); err != nil {
			return err
		}
		return q.InsertVerificationEvent(ctx, event(id, nil, row.VerificationStatus, "retired", &actor, nil, row.Revision, now))
	})
}

func lockActive(ctx context.Context, q *sqlc.Queries, id uuid.UUID) (sqlc.ContentQuestion, error) {
	row, err := q.GetQuestionForUpdate(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return row, apierror.New(apierror.NotFound)
	}
	if err != nil {
		return row, err
	}
	if row.RetiredAt != nil {
		return row, apierror.New(apierror.QuestionStateInvalid).With("status", "retired")
	}
	return row, nil
}

func event(id uuid.UUID, locale *string, from, to string, actor *uuid.UUID, reason *string, rev int32, now time.Time) sqlc.InsertVerificationEventParams {
	return sqlc.InsertVerificationEventParams{
		ID: ids.New(), QuestionID: id, Locale: locale, FromStatus: from, ToStatus: to,
		ActorID: actor, Reason: reason, Revision: rev, CreatedAt: now,
	}
}

// snapshotBody is what players saw at one revision (§4.3).
type snapshotBody struct {
	Revision        int32                  `json:"revision"`
	QuestionType    string                 `json:"question_type"`
	Difficulty      int16                  `json:"difficulty"`
	SubcategoryID   uuid.UUID              `json:"subcategory_id"`
	CorrectOptionID *uuid.UUID             `json:"correct_option_id"`
	Translations    map[string]snapshotTxt `json:"translations"`
}

type snapshotTxt struct {
	Prompt      string               `json:"prompt"`
	Explanation *string              `json:"explanation"`
	Options     map[uuid.UUID]string `json:"options"`
}

func snapshot(ctx context.Context, q *sqlc.Queries, id uuid.UUID, rev int32, actor *uuid.UUID, now time.Time) error {
	row, err := q.GetQuestion(ctx, id)
	if err != nil {
		return err
	}
	st, err := load(ctx, q, id)
	if err != nil {
		return err
	}
	body := snapshotBody{
		Revision: rev, QuestionType: row.QuestionType, Difficulty: row.Difficulty,
		SubcategoryID: row.SubcategoryID, Translations: map[string]snapshotTxt{},
	}
	for _, o := range st.options {
		if o.IsCorrect {
			body.CorrectOptionID = &o.ID
		}
	}
	for l, t := range st.translations {
		opts := map[uuid.UUID]string{}
		for i, o := range st.options {
			opts[o.ID] = t.Options[i]
		}
		body.Translations[l] = snapshotTxt{Prompt: t.Prompt, Explanation: t.Explanation, Options: opts}
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return q.InsertRevision(ctx, sqlc.InsertRevisionParams{QuestionID: id, Revision: rev, Snapshot: raw, CreatedBy: actor, CreatedAt: now})
}
