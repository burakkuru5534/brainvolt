package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/content"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain/config"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain/engine"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/redis"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/testdb"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/testenv"
)

// users is the identity port: every known user is active.
type users map[uuid.UUID]identity.User

func (u users) User(_ context.Context, id uuid.UUID) (identity.User, error) {
	if v, ok := u[id]; ok {
		return v, nil
	}
	return identity.User{}, apierror.New(apierror.NotFound)
}

type fixture struct {
	t       *testing.T
	svc     *Service
	db      *pgxpool.Pool
	rdb     *redis.Client
	content *content.Module
	clock   *clock.Fake
	users   users
	cats    map[string]uuid.UUID
}

// Five categories of 2×4 questions (difficulties cycling 1–5, every fourth
// one true/false) and a "tiny" category of 2 questions.
func newFixture(t *testing.T) *fixture {
	t.Helper()
	ctx := t.Context()
	db := testdb.New(t)
	rdb, err := redis.Connect(ctx, testenv.RedisURL(t), "brainvolt-test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = rdb.Close() })
	f := &fixture{t: t, db: db, rdb: rdb, users: users{}, cats: map[string]uuid.UUID{},
		clock: clock.NewFake(time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC))}

	bus := events.NewBus()
	f.content = content.New(db, f.clock, bus)
	bus.Subscribe(EventMatchCompleted, content.SubscriberMatchStats, f.content.OnMatchCompleted)
	for _, ev := range []string{EventMatchStarted, EventMatchCancelled, content.EventQuestionVerified} {
		bus.Subscribe(ev, "test.on_"+ev, func(context.Context, events.Envelope) error { return nil })
	}
	inserter, err := river.NewClient(riverpgxv5.New(db), &river.Config{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		t.Fatal(err)
	}
	bus.Attach(inserter)
	f.svc = New(Deps{DB: db, Redis: rdb, Questions: f.content, Users: f.users, Events: bus, Clock: f.clock,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})

	mod := f.user()
	source := ids.New()
	if err := f.content.UpsertSources(ctx, []content.SourceSpec{{
		ID: source, Kind: "brainvolt_team", Name: "BrainVolt", LicenseCode: "LicenseRef-BrainVolt", AttributionText: "BrainVolt team",
	}}); err != nil {
		t.Fatal(err)
	}
	names := func(s string) map[string]content.Name {
		return map[string]content.Name{"tr": {Name: s}, "en": {Name: s}}
	}
	slugs := []string{"c1", "c2", "c3", "c4", "c5", "tiny"}
	var specs []content.CategorySpec
	for _, c := range slugs {
		specs = append(specs, content.CategorySpec{Slug: c, IconKey: c, Names: names(c), Subcategories: []content.SubcategorySpec{
			{Slug: c + "-a", Names: names(c + "-a")}, {Slug: c + "-b", Names: names(c + "-b")},
		}})
	}
	if err := f.content.UpsertTaxonomy(ctx, specs); err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, c := range slugs {
		for _, s := range []string{"a", "b"} {
			sub, err := f.content.SubcategoryID(ctx, c, c+"-"+s)
			if err != nil {
				t.Fatal(err)
			}
			per := 4
			if c == "tiny" {
				per = 1
			}
			for range per {
				n++
				typ, tr, en := content.MultipleChoice, []string{"A", "B", "C", "D"}, []string{"A", "B", "C", "D"}
				if n%4 == 0 {
					typ, tr, en = content.TrueFalse, []string{"Doğru", "Yanlış"}, []string{"True", "False"}
				}
				res, err := f.content.Import(ctx, content.Draft{
					SubcategoryID: sub, Type: typ, Difficulty: 1 + n%5, OriginalLocale: "tr", SourceID: &source, Correct: 1,
					Texts: map[string]content.Text{
						"tr": {Prompt: fmt.Sprintf("Soru %d?", n), Options: tr, Origin: content.OriginOriginal},
						"en": {Prompt: fmt.Sprintf("Question %d?", n), Options: en, Origin: content.OriginHumanTranslation},
					},
				})
				if err != nil {
					t.Fatal(err)
				}
				if err := f.content.ApproveAll(ctx, mod, res.ID); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	rows, err := db.Query(ctx, `SELECT slug, id FROM content.categories`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var slug string
		var id uuid.UUID
		if err := rows.Scan(&slug, &id); err != nil {
			t.Fatal(err)
		}
		f.cats[slug] = id
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return f
}

// user inserts an active full account.
func (f *fixture) user() uuid.UUID {
	f.t.Helper()
	id := ids.New()
	if _, err := f.db.Exec(f.t.Context(), `INSERT INTO identity.users (id, firebase_uid, age_verified_at, last_seen_at, created_at, updated_at)
		VALUES ($1, $2, now(), now(), now(), now())`, id, "fb-"+id.String()); err != nil {
		f.t.Fatal(err)
	}
	f.users[id] = identity.User{ID: id, Tier: identity.TierFull, Status: identity.StatusActive}
	return id
}

func (f *fixture) duel(c config.Config, a, b uuid.UUID) (*Created, error) {
	return f.svc.Create(f.t.Context(), CreateRequest{Config: c, Sides: [2][]Member{{{UserID: a, Locale: "tr"}}, {{UserID: b, Locale: "tr"}}}})
}

func (f *fixture) lock(u uuid.UUID) string {
	v, err := f.rdb.Get(f.t.Context(), liveKey(u)).Result()
	if err != nil {
		return ""
	}
	return v
}

func (f *fixture) count(sql string, args ...any) int {
	f.t.Helper()
	var n int
	if err := f.db.QueryRow(f.t.Context(), sql, args...).Scan(&n); err != nil {
		f.t.Fatal(err)
	}
	return n
}

func (f *fixture) apply(m *engine.Match, o engine.Output) {
	f.t.Helper()
	if err := f.svc.Apply(f.t.Context(), m, o); err != nil {
		f.t.Fatal(err)
	}
}

func code(err error) apierror.Code {
	if e := apierror.From(err); e != nil {
		return e.Code
	}
	return ""
}

var quickDuel = config.Config{MatchType: domain.QuickDuel, Format: domain.Format1v1, Origin: domain.Challenge}

// A full quick duel: creation with its locks, every round persisted, the
// result, the outbox event, the statistics and the lock release.
func TestQuickDuelIsPersisted(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	a, b := f.user(), f.user()
	c, err := f.duel(quickDuel, a, b)
	if err != nil {
		t.Fatal(err)
	}
	m := c.Match
	f.apply(m, c.Output)
	if f.lock(a) != m.ID.String() || f.lock(b) != m.ID.String() {
		t.Fatal("live locks not taken")
	}
	if got := f.count(`SELECT count(*) FROM game.matches WHERE id = $1 AND status = 'waiting' AND question_count = 7`, m.ID); got != 1 {
		t.Fatal("match not stored in waiting")
	}

	now := m.CreatedMS
	for _, u := range []uuid.UUID{a, b} {
		o, err := m.Ready(u, now)
		if err != nil {
			t.Fatal(err)
		}
		f.apply(m, o)
	}
	now = m.StartsAtMS
	f.apply(m, m.Fire(engine.Timer{Kind: engine.TimerStart}, now))
	asked := map[uuid.UUID]bool{}
	for m.Status == domain.StatusInProgress {
		r := m.Round
		asked[r.Question.ID] = true
		now = r.ShowAtMS + 2_000
		o, err := m.Answer(a, r.No, r.Question.Correct, now)
		if err != nil {
			t.Fatal(err)
		}
		f.apply(m, o)
		wrong := r.Question.Options[0]
		if wrong == r.Question.Correct {
			wrong = r.Question.Options[1]
		}
		if o, err = m.Answer(b, r.No, wrong, now); err != nil {
			t.Fatal(err)
		}
		if o.Round == nil {
			t.Fatal("round not locked after both answers")
		}
		f.apply(m, o)
		now += engine.RevealMS
		f.apply(m, m.Fire(engine.Timer{Kind: engine.TimerReveal, Round: r.No}, now))
	}

	res := m.Result()
	if m.Status != domain.StatusCompleted || res.WinnerSide != 1 || len(asked) != 7 {
		t.Fatalf("status %s, result %+v, %d questions", m.Status, res, len(asked))
	}
	if got := f.count(`SELECT count(*) FROM game.matches WHERE id = $1 AND status = 'completed' AND winner_side_no = 1
		AND end_reason = 'all_rounds_played' AND started_at IS NOT NULL AND ready_at IS NOT NULL`, m.ID); got != 1 {
		t.Fatal("completed match not stored")
	}
	if got := f.count(`SELECT count(*) FROM game.match_rounds WHERE match_id = $1`, m.ID); got != 7 {
		t.Fatalf("%d rounds", got)
	}
	if got := f.count(`SELECT count(*) FROM game.answers WHERE match_id = $1`, m.ID); got != 14 {
		t.Fatalf("%d answers", got)
	}
	// Score invariant (0.3.4): side score = sum of its round points.
	if got := f.count(`SELECT count(*) FROM game.match_sides s WHERE s.match_id = $1 AND s.score = (
		SELECT coalesce(sum(points), 0) FROM game.round_results r WHERE r.match_id = s.match_id AND r.side_no = s.side_no)`, m.ID); got != 2 {
		t.Fatal("score invariant broken")
	}
	if got := f.count(`SELECT count(*) FROM game.match_sides WHERE match_id = $1 AND side_no = 1 AND result = 'win' AND score = $2 AND correct_count = 7`,
		m.ID, res.Sides[0].Score); got != 1 {
		t.Fatalf("winner side not stored: %+v", res.Sides[0])
	}
	if got := f.count(`SELECT count(*) FROM game.match_participants WHERE match_id = $1 AND status = 'finished' AND answered_count = 7`, m.ID); got != 2 {
		t.Fatal("participants not finished")
	}
	if got := f.count(`SELECT count(*) FROM game.match_transitions WHERE match_id = $1`, m.ID); got != 4 {
		t.Fatalf("%d transitions, want waiting→ready→in_progress→completed", got)
	}
	if f.lock(a) != "" || f.lock(b) != "" {
		t.Fatal("live locks not released")
	}

	// Outbox: one start, one completion; the statistics are applied once.
	if got := f.count(`SELECT count(*) FROM river_job WHERE args->'event'->>'name' = $1`, EventMatchStarted); got != 1 {
		t.Fatalf("%d MatchStarted jobs", got)
	}
	var raw []byte
	if err := f.db.QueryRow(ctx, `SELECT args->'event' FROM river_job WHERE args->>'subscriber' = $1`, content.SubscriberMatchStats).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var env events.Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := f.content.OnMatchCompleted(ctx, env); err != nil {
			t.Fatal(err)
		}
	}
	if got := f.count(`SELECT coalesce(sum(times_asked), 0) FROM content.question_stats`); got != 14 {
		t.Fatalf("times_asked %d, want 14 (counted once)", got)
	}
	if got := f.count(`SELECT coalesce(sum(times_correct), 0) FROM content.question_stats`); got != 7 {
		t.Fatalf("times_correct %d", got)
	}

	// Applying the final output again changes nothing (idempotent owner retry).
	if err := f.svc.Apply(ctx, m, engine.Output{Result: res}); err != nil {
		t.Fatal(err)
	}
	if got := f.count(`SELECT count(*) FROM river_job WHERE args->>'subscriber' = $1`, content.SubscriberMatchStats); got != 1 {
		t.Fatalf("%d completion jobs after retry", got)
	}

	// The no-repeat window: a rematch avoids the 7 questions just played.
	c2, err := f.duel(quickDuel, a, b)
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range c2.Match.Questions() {
		if asked[q.ID] {
			t.Fatal("a recently seen question was selected")
		}
	}
}

func TestCreateGuards(t *testing.T) {
	f := newFixture(t)
	a, b, c := f.user(), f.user(), f.user()

	// LC06: a user in a live match cannot join another; the other
	// participant's lock is not leaked.
	first, err := f.duel(quickDuel, a, b)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.duel(quickDuel, c, a); code(err) != apierror.AlreadyInMatch {
		t.Fatalf("second match: %v", err)
	}
	if f.lock(c) != "" {
		t.Fatal("lock of the refused match leaked")
	}

	// LC12: not enough questions → nothing stored, no lock left.
	master := quickDuel
	master.CategoryMode = domain.CategoryMaster
	master.CategoryIDs = []uuid.UUID{f.cats["tiny"]}
	d := f.user()
	if _, err := f.duel(master, c, d); code(err) != apierror.NotEnoughQuestions {
		t.Fatalf("tiny category: %v", err)
	}
	if f.lock(c) != "" || f.lock(d) != "" {
		t.Fatal("lock leaked after NOT_ENOUGH_QUESTIONS")
	}

	// Mode matrix and context rules.
	bad := quickDuel
	bad.TeamMode = domain.SharedAnswer
	if _, err := f.duel(bad, c, d); code(err) != apierror.ModeNotAllowed {
		t.Fatalf("1v1 with a team mode: %v", err)
	}
	unknown := quickDuel
	unknown.CategoryMode = domain.CategoryMaster
	unknown.CategoryIDs = []uuid.UUID{uuid.New()}
	if _, err := f.duel(unknown, c, d); err == nil {
		t.Fatal("unknown category accepted")
	}
	if got := f.count(`SELECT count(*) FROM game.matches`); got != 1 {
		t.Fatalf("%d matches stored, want only the first", got)
	}

	// A deleted (pending) account cannot start a match.
	now := f.clock.Now()
	u := f.users[d]
	u.DeletionRequestedAt = &now
	f.users[d] = u
	if _, err := f.duel(quickDuel, c, d); err == nil {
		t.Fatal("deletion-pending user accepted")
	}

	// MT05: ranked only from matchmaking.
	ranked := config.Config{MatchType: domain.Ranked, Format: domain.Format1v1, Origin: domain.Matchmaking}
	challenge := ranked
	challenge.Origin = domain.Challenge
	if _, err := f.duel(challenge, c, f.user()); code(err) != apierror.ModeNotAllowed {
		t.Fatalf("ranked challenge: %v", err)
	}

	// MT04: ranked asks multiple choice only, verified in every participant
	// locale (c5's English texts are back in review). Ranked is rated in
	// the preseason.
	if _, err := f.db.Exec(t.Context(), `UPDATE content.question_translations t SET verification_status = 'in_review'
		FROM content.questions q JOIN content.subcategories s ON s.id = q.subcategory_id
		WHERE t.question_id = q.id AND t.locale = 'en' AND s.category_id = $1`, f.cats["c5"]); err != nil {
		t.Fatal(err)
	}
	en := f.user()
	r, err := f.svc.Create(t.Context(), CreateRequest{Config: ranked, Sides: [2][]Member{{{UserID: c, Locale: "tr"}}, {{UserID: en, Locale: "en"}}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range r.Match.Questions() {
		if q.Type != domain.MultipleChoice || q.CategoryID == f.cats["c5"] {
			t.Fatalf("ranked question %+v", q)
		}
	}
	if got := f.count(`SELECT count(*) FROM game.matches WHERE id = $1 AND is_rated AND season_id = $2 AND question_count = 10`,
		r.Match.ID, PreseasonID); got != 1 {
		t.Fatal("ranked match not rated in the preseason")
	}

	// LC03: nobody ready by the deadline → cancelled, locks released.
	m := first.Match
	o := m.Fire(engine.Timer{Kind: engine.TimerReadyDeadline}, m.ReadyDeadMS)
	f.apply(m, o)
	if got := f.count(`SELECT count(*) FROM game.matches WHERE id = $1 AND status = 'cancelled' AND cancel_reason = 'not_all_ready'`, m.ID); got != 1 {
		t.Fatal("match not cancelled")
	}
	if f.lock(a) != "" || f.lock(b) != "" {
		t.Fatal("locks not released after cancel")
	}
	if got := f.count(`SELECT count(*) FROM river_job WHERE args->'event'->>'name' = $1`, EventMatchCancelled); got != 1 {
		t.Fatalf("%d MatchCancelled jobs", got)
	}
	if _, err := f.duel(quickDuel, a, b); err != nil {
		t.Fatalf("new match after cancel: %v", err)
	}
}
