package service

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/content/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/testdb"
)

type fixture struct {
	t      *testing.T
	svc    *Service
	db     *pgxpool.Pool
	clock  *clock.Fake
	mod    uuid.UUID
	source uuid.UUID
	subs   map[string]uuid.UUID
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	db := testdb.New(t)
	ctx := t.Context()
	bus := events.NewBus()
	for _, ev := range []string{EventQuestionVerified, EventQuestionRejected} {
		bus.Subscribe(ev, "test.on_"+ev, func(context.Context, events.Envelope) error { return nil })
	}
	inserter, err := river.NewClient(riverpgxv5.New(db), &river.Config{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		t.Fatal(err)
	}
	bus.Attach(inserter)
	f := &fixture{t: t, db: db, clock: clock.NewFake(time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)), subs: map[string]uuid.UUID{}}
	f.svc = New(db, f.clock, bus)

	f.mod = ids.New()
	if _, err := db.Exec(ctx, `INSERT INTO identity.users (id, firebase_uid, age_verified_at, last_seen_at, created_at, updated_at)
		VALUES ($1, 'fb-mod', now(), now(), now(), now())`, f.mod); err != nil {
		t.Fatal(err)
	}
	f.source = ids.New()
	if err := f.svc.UpsertSources(ctx, []SourceSpec{{
		ID: f.source, Kind: "brainvolt_team", Name: "BrainVolt", LicenseCode: "LicenseRef-BrainVolt", AttributionText: "BrainVolt team",
	}}); err != nil {
		t.Fatal(err)
	}
	names := func(tr, en string) map[string]Name { return map[string]Name{"tr": {Name: tr}, "en": {Name: en}} }
	if err := f.svc.UpsertTaxonomy(ctx, []CategorySpec{
		{Slug: "science", IconKey: "science", Names: names("Bilim", "Science"), Subcategories: []SubcategorySpec{
			{Slug: "space", Names: names("Uzay", "Space")}, {Slug: "physics", Names: names("Fizik", "Physics")},
		}},
		{Slug: "history", IconKey: "history", Names: names("Tarih", "History"), Subcategories: []SubcategorySpec{
			{Slug: "world-history", Names: names("Dünya Tarihi", "World History")},
		}},
	}); err != nil {
		t.Fatal(err)
	}
	for _, k := range [][2]string{{"science", "space"}, {"science", "physics"}, {"history", "world-history"}} {
		id, err := f.svc.SubcategoryID(ctx, k[0], k[1])
		if err != nil {
			t.Fatal(err)
		}
		f.subs[k[1]] = id
	}
	return f
}

func (f *fixture) draft(sub, prompt string, opts ...string) domain.Draft {
	if len(opts) == 0 {
		opts = []string{"Jüpiter", "Satürn", "Mars", "Venüs"}
	}
	return domain.Draft{
		SubcategoryID: f.subs[sub], Type: domain.MultipleChoice, Difficulty: 2, OriginalLocale: "tr",
		SourceID: &f.source, Correct: 1,
		Texts: map[string]domain.Text{
			"tr": {Prompt: prompt, Options: opts, Origin: domain.OriginOriginal},
			"en": {Prompt: "EN: " + prompt, Options: []string{"Jupiter", "Saturn", "Mars", "Venus"}, Origin: domain.OriginHumanTranslation},
		},
	}
}

func (f *fixture) create(d domain.Draft) uuid.UUID {
	f.t.Helper()
	id, err := f.svc.Create(f.t.Context(), d, CreateOptions{Actor: &f.mod, Status: domain.StatusDraft, AuthorUserID: &f.mod})
	if err != nil {
		f.t.Fatal(err)
	}
	return id
}

func (f *fixture) get(id uuid.UUID) Question {
	f.t.Helper()
	q, err := f.svc.Get(f.t.Context(), id)
	if err != nil {
		f.t.Fatal(err)
	}
	return q
}

func (f *fixture) count(sql string, args ...any) int {
	f.t.Helper()
	var n int
	if err := f.db.QueryRow(f.t.Context(), sql, args...).Scan(&n); err != nil {
		f.t.Fatal(err)
	}
	return n
}

func code(err error) apierror.Code {
	if e := apierror.From(err); e != nil {
		return e.Code
	}
	return ""
}

func (f *fixture) eligible(locale, sub string) Counts {
	f.t.Helper()
	cat, err := f.svc.Catalog(f.t.Context(), locale)
	if err != nil {
		f.t.Fatal(err)
	}
	for _, c := range cat.Categories {
		for _, s := range c.Subcategories {
			if s.Slug == sub {
				return s.Eligible
			}
		}
	}
	f.t.Fatalf("sub-category %s not in catalogue", sub)
	return Counts{}
}

func TestQuestionLifecycle(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	id := f.create(f.draft("space", "Güneş Sistemi'nin en büyük gezegeni hangisidir?"))

	q := f.get(id)
	if q.Status != domain.StatusDraft || q.Revision != 1 || len(q.Options) != 4 || !q.Options[0].IsCorrect ||
		q.Translations["tr"].Options[0] != "Jüpiter" || q.Translations["en"].Status != domain.StatusDraft ||
		len(q.Revisions) != 1 || len(q.Events) != 1 || q.Used {
		t.Fatalf("created = %+v", q)
	}
	if got := f.eligible("tr", "space"); got != (Counts{}) {
		t.Fatalf("a draft is not eligible: %+v", got)
	}

	// Review before submit / translation of a draft: not allowed.
	if err := f.svc.Review(ctx, f.mod, id, Review{Decision: domain.DecisionApprove}); code(err) != apierror.QuestionStateInvalid {
		t.Fatalf("review of a draft: %v", err)
	}
	if err := f.svc.Review(ctx, f.mod, id, Review{Decision: domain.DecisionReject}); code(err) != apierror.ValidationFailed {
		t.Fatalf("reject without reason: %v", err)
	}

	if err := f.svc.Submit(ctx, f.mod, id); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.Submit(ctx, f.mod, id); code(err) != apierror.QuestionStateInvalid {
		t.Fatalf("second submit: %v", err)
	}
	q = f.get(id)
	if q.Status != domain.StatusInReview || q.Translations["tr"].Status != domain.StatusInReview || q.Translations["en"].Status != domain.StatusInReview {
		t.Fatalf("submitted = %+v", q)
	}

	// Approving the question verifies its original (tr) wording only (§2.1).
	if err := f.svc.Review(ctx, f.mod, id, Review{Decision: domain.DecisionApprove}); err != nil {
		t.Fatal(err)
	}
	q = f.get(id)
	if q.Status != domain.StatusVerified || q.VerifiedBy == nil || *q.VerifiedBy != f.mod ||
		q.Translations["tr"].Status != domain.StatusVerified || q.Translations["en"].Status != domain.StatusInReview {
		t.Fatalf("approved = %+v", q)
	}
	if got := f.eligible("tr", "space"); got != (Counts{Ranked: 1, Casual: 1}) {
		t.Fatalf("tr eligible = %+v", got)
	}
	if got := f.eligible("en", "space"); got != (Counts{}) {
		t.Fatalf("en translation not verified yet (D-007): %+v", got)
	}
	en := "en"
	if err := f.svc.Review(ctx, f.mod, id, Review{Decision: domain.DecisionApprove, Locale: &en}); err != nil {
		t.Fatal(err)
	}
	if got := f.eligible("en", "space"); got != (Counts{Ranked: 1, Casual: 1}) {
		t.Fatalf("en eligible = %+v", got)
	}
	if n := f.count(`SELECT count(*) FROM river_job WHERE args->'event'->>'name' = $1`, EventQuestionVerified); n != 2 {
		t.Fatalf("QuestionVerified events = %d, want 2 (question + en)", n)
	}

	// Editing a verified question sends it back to review (§4.1) with a new
	// revision; the edited translation waits for review too.
	f.clock.Advance(time.Minute)
	diff := 3
	if err := f.svc.Update(ctx, f.mod, id, Patch{Difficulty: &diff, Texts: map[string]domain.Text{
		"en": {Prompt: "Which is the largest planet in the Solar System?", Options: []string{"Jupiter", "Saturn", "Mars", "Venus"}, Origin: domain.OriginHumanTranslation},
	}}); err != nil {
		t.Fatal(err)
	}
	q = f.get(id)
	if q.Status != domain.StatusInReview || q.Revision != 2 || q.Difficulty != 3 || len(q.Revisions) != 2 ||
		q.Translations["en"].Status != domain.StatusInReview || q.Translations["tr"].Status != domain.StatusVerified {
		t.Fatalf("edited = %+v", q)
	}
	if got := f.eligible("tr", "space"); got != (Counts{}) {
		t.Fatalf("a question in review is not eligible: %+v", got)
	}

	// Changing the correct option of a used question is refused (§4.3).
	if _, err := f.db.Exec(ctx, `INSERT INTO content.question_stats (question_id, locale, times_asked) VALUES ($1, 'tr', 3)`, id); err != nil {
		t.Fatal(err)
	}
	two := 2
	if err := f.svc.Update(ctx, f.mod, id, Patch{Correct: &two}); code(err) != apierror.QuestionInUse {
		t.Fatalf("structural edit of a used question: %v", err)
	}

	// Reject, then retire.
	if err := f.svc.Review(ctx, f.mod, id, Review{Decision: domain.DecisionReject, Reason: "ambiguous"}); err != nil {
		t.Fatal(err)
	}
	if q = f.get(id); q.Status != domain.StatusRejected || q.RejectionReason == nil || *q.RejectionReason != "ambiguous" || !q.Used {
		t.Fatalf("rejected = %+v", q)
	}
	if err := f.svc.Retire(ctx, f.mod, id); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.Retire(ctx, f.mod, id); err != nil {
		t.Fatalf("retiring twice is harmless: %v", err)
	}
	if err := f.svc.Update(ctx, f.mod, id, Patch{Difficulty: &diff}); code(err) != apierror.QuestionStateInvalid {
		t.Fatalf("edit of a retired question: %v", err)
	}
	if q = f.get(id); q.RetiredAt == nil {
		t.Fatal("not retired")
	}
}

func TestCorrectOptionChangeAndTrueFalse(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	id := f.create(f.draft("physics", "Hangisi bir gezegendir?"))
	two := 2
	if err := f.svc.Update(ctx, f.mod, id, Patch{Correct: &two}); err != nil {
		t.Fatalf("an unused question may change its correct option: %v", err)
	}
	if q := f.get(id); q.Options[0].IsCorrect || !q.Options[1].IsCorrect || q.Status != domain.StatusDraft {
		t.Fatalf("options = %+v", q.Options)
	}

	tf := domain.Draft{
		SubcategoryID: f.subs["physics"], Type: domain.TrueFalse, Difficulty: 1, OriginalLocale: "tr", SourceID: &f.source, Correct: 2,
		Texts: map[string]domain.Text{"tr": {Prompt: "Ses boşlukta yayılır.", Origin: domain.OriginOriginal}},
	}
	tfID := f.create(tf)
	if err := f.svc.ApproveAll(ctx, f.mod, tfID); err != nil {
		t.Fatal(err)
	}
	q := f.get(tfID)
	if q.Status != domain.StatusVerified || q.Translations["tr"].Options[1] != "Yanlış" || !q.Options[1].IsCorrect {
		t.Fatalf("true/false = %+v", q)
	}
	// Ranked excludes true/false (D-024).
	if got := f.eligible("tr", "physics"); got != (Counts{Ranked: 0, Casual: 1}) {
		t.Fatalf("physics eligible = %+v", got)
	}
}

func TestDuplicatesSourcesAndImport(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	d := f.draft("space", "Kızıl Gezegen hangisidir?")
	id := f.create(d)

	again := f.draft("space", "  kızıl gezegen HANGİSİDİR ", "Venüs", "Mars", "Satürn", "Jüpiter")
	_, err := f.svc.Create(ctx, again, CreateOptions{Status: domain.StatusDraft})
	if e := apierror.From(err); e == nil || e.Code != apierror.QuestionDuplicate || e.Details["question_id"] != id {
		t.Fatalf("duplicate: %v", err)
	}

	unknown := uuid.New()
	other := f.draft("space", "Başka bir soru metni?")
	other.SourceID = &unknown
	if _, err := f.svc.Create(ctx, other, CreateOptions{Status: domain.StatusDraft}); code(err) != apierror.QuestionSourceRequired {
		t.Fatalf("unknown source (D-006): %v", err)
	}
	other.SourceID = &f.source
	other.SubcategoryID = uuid.New()
	if _, err := f.svc.Create(ctx, other, CreateOptions{Status: domain.StatusDraft}); code(err) != apierror.ValidationFailed {
		t.Fatalf("unknown sub-category: %v", err)
	}

	imp := f.draft("world-history", "Fransız İhtilali hangi yıl başladı?", "1789", "1776", "1815", "1848")
	res, err := f.svc.Import(ctx, imp)
	if err != nil || !res.Created {
		t.Fatalf("import = %+v, %v", res, err)
	}
	if q := f.get(res.ID); q.Status != domain.StatusInReview || q.Translations["en"].Status != domain.StatusInReview {
		t.Fatalf("imported questions start in review (§6): %+v", q)
	}
	res2, err := f.svc.Import(ctx, imp)
	if err != nil || res2.Created || res2.ID != res.ID {
		t.Fatalf("re-import = %+v, %v", res2, err)
	}

	credits, err := f.svc.Credits(ctx)
	if err != nil || len(credits) != 1 || credits[0].LicenseCode != "LicenseRef-BrainVolt" {
		t.Fatalf("credits = %+v, %v", credits, err)
	}
}

func TestCatalogListAndETag(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	cat, err := f.svc.Catalog(ctx, "tr")
	if err != nil || len(cat.Categories) != 2 || cat.Categories[0].Name != "Bilim" || len(cat.Categories[0].Subcategories) != 2 ||
		cat.Categories[0].Subcategories[0].Slug != "space" {
		t.Fatalf("catalog = %+v, %v", cat, err)
	}
	en, _ := f.svc.Catalog(ctx, "en")
	if en.Categories[1].Subcategories[0].Name != "World History" {
		t.Fatalf("en names: %+v", en)
	}
	tag1, _ := f.svc.CatalogETag(ctx, "tr")
	tagEN, _ := f.svc.CatalogETag(ctx, "en")
	if tag1 == tagEN {
		t.Fatal("ETag must differ per locale")
	}

	var created []uuid.UUID
	for i, p := range []string{"Birinci soru metni?", "İkinci soru metni?", "Üçüncü soru metni?"} {
		f.clock.Advance(time.Second)
		created = append(created, f.create(f.draft([]string{"space", "physics", "space"}[i], p)))
	}
	tag2, _ := f.svc.CatalogETag(ctx, "tr")
	if tag2 == tag1 {
		t.Fatal("ETag must change with content")
	}
	if err := f.svc.ApproveAll(ctx, f.mod, created[0]); err != nil {
		t.Fatal(err)
	}
	if got := f.eligible("en", "space"); got != (Counts{Ranked: 1, Casual: 1}) {
		t.Fatalf("ApproveAll verifies every translation: %+v", got)
	}
	if cat, _ = f.svc.Catalog(ctx, "tr"); cat.Categories[0].Eligible != (Counts{Ranked: 1, Casual: 1}) {
		t.Fatalf("category totals = %+v", cat.Categories[0].Eligible)
	}

	page, err := f.svc.List(ctx, ListFilter{Limit: 2})
	if err != nil || len(page.Items) != 2 || page.NextCursor == nil || page.Items[0].ID != created[2] {
		t.Fatalf("page 1 = %+v, %v", page, err)
	}
	cursor := uuid.MustParse(*page.NextCursor)
	page, _ = f.svc.List(ctx, ListFilter{Limit: 2, Cursor: &cursor})
	if len(page.Items) != 1 || page.NextCursor != nil || page.Items[0].ID != created[0] {
		t.Fatalf("page 2 = %+v", page)
	}
	verified := domain.StatusVerified
	page, _ = f.svc.List(ctx, ListFilter{Limit: 10, Status: &verified})
	if len(page.Items) != 1 || page.Items[0].Translations["en"] != domain.StatusVerified {
		t.Fatalf("status filter = %+v", page)
	}
	en2, draft := "en", domain.StatusDraft
	page, _ = f.svc.List(ctx, ListFilter{Limit: 10, Locale: &en2, TranslationStatus: &draft, SubcategoryID: ptr(f.subs["space"])})
	if len(page.Items) != 1 || page.Items[0].ID != created[2] || page.Items[0].Prompt != "EN: Üçüncü soru metni?" {
		t.Fatalf("locale filter = %+v", page)
	}
	if err := f.svc.Retire(ctx, f.mod, created[1]); err != nil {
		t.Fatal(err)
	}
	if page, _ = f.svc.List(ctx, ListFilter{Limit: 10}); len(page.Items) != 2 {
		t.Fatalf("retired questions are hidden by default: %d", len(page.Items))
	}
	if page, _ = f.svc.List(ctx, ListFilter{Limit: 10, IncludeRetired: true}); len(page.Items) != 3 {
		t.Fatalf("include_retired: %d", len(page.Items))
	}
}

func ptr[T any](v T) *T { return &v }
