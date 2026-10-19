package service

import (
	"context"
	"errors"
	"math/rand/v2"
	"slices"
	"time"
	_ "time/tzdata" // Europe/Istanbul on hosts without zoneinfo

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/content"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain/scoring"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
)

// Daily run timing (category-modes §5.1): the Ranked 15 s window, a fixed
// answer grace, a short intro before the first question and the reveal
// pause between questions.
const (
	DailyQuestions  = 10
	DailyWindowMS   = 15_000
	dailyGraceMS    = 250
	dailyIntroMS    = 2_000
	dailyRevealMS   = 3_000
	dailyStaleAfter = 10 * time.Minute
)

// Daily run kinds and statuses (schema-platform §6).
const (
	DailyOfficial      = "official"
	DailyPractice      = "practice"
	DailyInProgress    = "in_progress"
	DailyCompleted     = "completed"
	DailyVoidNotStared = "void_not_started"
)

// EventDailyRunCompleted is published for completed official runs only.
const EventDailyRunCompleted = "game.DailyRunCompleted"

// DailyRunCompleted is the payload of game.DailyRunCompleted.
type DailyRunCompleted struct {
	RunID        uuid.UUID `json:"run_id"`
	UserID       uuid.UUID `json:"user_id"`
	Day          string    `json:"day"`
	CorrectCount int       `json:"correct_count"`
	TotalTimeMS  int       `json:"total_time_ms"`
	Score        int       `json:"score"`
	CompletedAt  time.Time `json:"completed_at"`
	TierAtStart  string    `json:"tier_at_start"`
}

var istanbul = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Istanbul")
	if err != nil {
		panic(err)
	}
	return loc
}()

// DailyDay is the Europe/Istanbul calendar day of t (D-029), at 00:00 UTC
// of that date for date columns.
func DailyDay(t time.Time) time.Time {
	l := t.In(istanbul)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, time.UTC)
}

// DailyRunView is a run with its result once finished.
type DailyRunView struct {
	ID          uuid.UUID         `json:"id"`
	Day         string            `json:"day"`
	Kind        string            `json:"kind"`
	Status      string            `json:"status"`
	StartedAt   time.Time         `json:"started_at"`
	CompletedAt *time.Time        `json:"completed_at"`
	Correct     *int16            `json:"correct_count"`
	TotalTimeMS *int32            `json:"total_time_ms"`
	Score       *int32            `json:"score"`
	Answers     []DailyAnswerView `json:"answers"`
}

// DailyAnswerView is one answered or timed-out question of a run.
type DailyAnswerView struct {
	Position   int16  `json:"position"`
	Correct    bool   `json:"correct"`
	ResponseMS *int32 `json:"response_ms"`
	Points     int32  `json:"points"`
}

// DailyToday is GET /v1/daily/today.
type DailyToday struct {
	Day       string        `json:"day"`
	Available bool          `json:"available"`
	Official  *DailyRunView `json:"official"`
	// PracticeAllowed: practice runs open once the official run is done.
	PracticeAllowed bool `json:"practice_allowed"`
}

func dailyView(r *sqlc.GameDailyRun, answers []sqlc.GameDailyRunAnswer) DailyRunView {
	v := DailyRunView{ID: r.ID, Day: r.Day.Format(time.DateOnly), Kind: r.Kind, Status: r.Status, StartedAt: r.StartedAt,
		CompletedAt: r.CompletedAt, Correct: r.CorrectCount, TotalTimeMS: r.TotalTimeMs, Score: r.Score, Answers: []DailyAnswerView{}}
	if r.Status == DailyCompleted {
		for i := range answers {
			a := &answers[i]
			v.Answers = append(v.Answers, DailyAnswerView{Position: a.Position, Correct: a.IsCorrect, ResponseMS: a.ResponseMs, Points: a.Points})
		}
	}
	return v
}

// DailyToday returns today's set availability and the caller's official
// run (settled first, so an abandoned run shows its result).
func (s *Service) DailyToday(ctx context.Context, user uuid.UUID) (DailyToday, error) {
	now := s.d.Clock.Now()
	day := DailyDay(now)
	out := DailyToday{Day: day.Format(time.DateOnly)}
	if _, err := s.d.Questions.DailySet(ctx, day); err == nil {
		out.Available = true
	} else if apierror.From(err).Code != apierror.NotEnoughQuestions {
		return DailyToday{}, err
	}
	run, err := sqlc.New(s.d.DB).OfficialRunOf(ctx, sqlc.OfficialRunOfParams{Day: day, UserID: user})
	if errors.Is(err, pgx.ErrNoRows) {
		return out, nil
	}
	if err != nil {
		return DailyToday{}, err
	}
	if _, err := s.settleDaily(ctx, run.ID); err != nil {
		return DailyToday{}, err
	}
	v, err := s.DailyRun(ctx, user, run.ID)
	if err != nil {
		return DailyToday{}, err
	}
	out.Official, out.PracticeAllowed = &v, v.Status == DailyCompleted
	return out, nil
}

// StartDaily starts today's official run (once a day; a voided run frees
// the slot) or a practice run after the official one (category-modes
// §5.1, §5.4).
func (s *Service) StartDaily(ctx context.Context, user uuid.UUID, kind, locale string) (DailyRunView, error) {
	if kind != DailyOfficial && kind != DailyPractice {
		return DailyRunView{}, apierror.Validation(apierror.FieldError{Field: "kind", Rule: "one_of"})
	}
	if locale != "tr" && locale != "en" {
		locale = "tr"
	}
	now := s.d.Clock.Now()
	day := DailyDay(now)
	set, err := s.d.Questions.DailySet(ctx, day)
	if err != nil {
		if apierror.From(err).Code == apierror.NotEnoughQuestions {
			return DailyRunView{}, apierror.New(apierror.DailyNotAvailable)
		}
		return DailyRunView{}, err
	}
	u, err := s.d.Users.User(ctx, user)
	if err != nil {
		return DailyRunView{}, err
	}
	q := sqlc.New(s.d.DB)
	official, err := q.OfficialRunOf(ctx, sqlc.OfficialRunOfParams{Day: day, UserID: user})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		if kind == DailyPractice {
			return DailyRunView{}, apierror.New(apierror.DailyAttemptInProgress).With("reason", "official_first")
		}
	case err != nil:
		return DailyRunView{}, err
	case kind == DailyOfficial && official.Status == DailyCompleted:
		return DailyRunView{}, apierror.New(apierror.DailyAttemptUsed)
	case official.Status == DailyInProgress:
		return DailyRunView{}, apierror.New(apierror.DailyAttemptInProgress).With("run_id", official.ID)
	}
	id := ids.New()
	if err := q.InsertDailyRun(ctx, sqlc.InsertDailyRunParams{
		ID: id, Day: day, UserID: user, SetID: set.ID, Locale: locale, Kind: kind, Tier: string(u.Tier),
		RulesetVersion: int32(scoring.Current.Version), Now: now, //nolint:gosec // G115: small
	}); err != nil {
		var pg *pgconn.PgError
		if errors.As(err, &pg) && pg.Code == "23505" {
			return DailyRunView{}, apierror.New(apierror.DailyAttemptInProgress)
		}
		return DailyRunView{}, err
	}
	return s.DailyRun(ctx, user, id)
}

// DailyRun returns the caller's run.
func (s *Service) DailyRun(ctx context.Context, user, id uuid.UUID) (DailyRunView, error) {
	q := sqlc.New(s.d.DB)
	r, err := q.GetDailyRun(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && r.UserID != user) {
		return DailyRunView{}, apierror.New(apierror.NotFound)
	}
	if err != nil {
		return DailyRunView{}, err
	}
	answers, err := q.DailyAnswers(ctx, id)
	if err != nil {
		return DailyRunView{}, err
	}
	return dailyView(&r, answers), nil
}

// DailyFrame is one message of a run to its player.
type DailyFrame struct {
	Type string
	Data map[string]any
}

// settled is a run after its clock caught up with now.
type settled struct {
	tx      pgx.Tx
	run     sqlc.GameDailyRun
	answers []sqlc.GameDailyRunAnswer
	set     content.DailySet
	// newly are the positions resolved (timed out) or shown by this settle.
	timedOut []int16
	shown    []int16
	done     bool
}

// settleDaily brings a run up to now (the clock never stops, §5.4):
// open questions past their deadline time out, the next question is shown
// a reveal pause after the previous one ended, the run completes after
// the 10th. A run whose first question was never shown and is stale is
// voided as not started.
func (s *Service) settleDaily(ctx context.Context, id uuid.UUID) (*settled, error) {
	var out *settled
	err := postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		r, err := q.GetDailyRunForUpdate(ctx, id)
		if err != nil {
			return err
		}
		out = &settled{tx: tx, run: r}
		if r.Status != DailyInProgress {
			out.done = true
			out.answers, err = q.DailyAnswers(ctx, id)
			return err
		}
		if out.answers, err = q.DailyAnswers(ctx, id); err != nil {
			return err
		}
		now := s.d.Clock.Now()
		if len(out.answers) == 0 {
			if now.Sub(r.StartedAt) > dailyStaleAfter {
				out.done, out.run.Status = true, DailyVoidNotStared
				return q.VoidDailyRun(ctx, sqlc.VoidDailyRunParams{Status: DailyVoidNotStared, Now: &now, ID: id})
			}
			return nil
		}
		if out.set, err = s.d.Questions.DailySet(ctx, r.Day); err != nil {
			return err
		}
		return s.advance(ctx, q, out, now)
	})
	return out, err
}

// advance resolves overdue questions and shows the following ones.
func (s *Service) advance(ctx context.Context, q *sqlc.Queries, st *settled, now time.Time) error {
	for {
		last := st.answers[len(st.answers)-1]
		if last.OptionID == nil {
			if !now.After(last.DeadlineAt.Add(dailyGraceMS * time.Millisecond)) {
				return nil // still open
			}
			st.timedOut = append(st.timedOut, last.Position)
		}
		if int(last.Position) >= len(st.set.Questions) {
			return s.completeDaily(ctx, q, st)
		}
		end := last.DeadlineAt
		if last.ReceivedAt != nil {
			end = *last.ReceivedAt
		}
		next, err := s.showDaily(ctx, q, st, last.Position+1, end.Add(dailyRevealMS*time.Millisecond))
		if err != nil {
			return err
		}
		st.answers = append(st.answers, next)
		st.shown = append(st.shown, next.Position)
	}
}

// showDaily stores question pos shown at showAt with a shuffled option
// order.
func (s *Service) showDaily(ctx context.Context, q *sqlc.Queries, st *settled, pos int16, showAt time.Time) (sqlc.GameDailyRunAnswer, error) {
	qu := st.set.Questions[pos-1]
	order := make([]int16, len(qu.Options))
	for i := range order {
		order[i] = int16(i + 1)
	}
	rand.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] }) //nolint:gosec // G404: option order, not security relevant
	a := sqlc.GameDailyRunAnswer{
		RunID: st.run.ID, Position: pos, QuestionID: qu.ID, QuestionRevision: qu.Revision, OptionOrder: order,
		ShowAt: showAt, DeadlineAt: showAt.Add(DailyWindowMS * time.Millisecond), CreatedAt: s.d.Clock.Now(),
	}
	return a, q.ShowDailyQuestion(ctx, sqlc.ShowDailyQuestionParams{
		RunID: a.RunID, Position: a.Position, QuestionID: a.QuestionID, QuestionRevision: a.QuestionRevision,
		OptionOrder: a.OptionOrder, ShowAt: a.ShowAt, DeadlineAt: a.DeadlineAt, Now: a.CreatedAt,
	})
}

// completeDaily totals the run (timeouts count the full window, §5.2) and
// publishes game.DailyRunCompleted for an official run.
func (s *Service) completeDaily(ctx context.Context, q *sqlc.Queries, st *settled) error {
	correct, total, score := 0, 0, 0
	end := st.run.StartedAt
	for i := range st.answers {
		a := &st.answers[i]
		if a.IsCorrect {
			correct++
		}
		if a.ResponseMs != nil && a.OptionID != nil {
			total += int(*a.ResponseMs)
		} else {
			total += DailyWindowMS
		}
		score += int(a.Points)
		e := a.DeadlineAt
		if a.ReceivedAt != nil {
			e = *a.ReceivedAt
		}
		end = e
	}
	c16, t32, s32 := int16(correct), int32(total), int32(score)
	if err := q.CompleteDailyRun(ctx, sqlc.CompleteDailyRunParams{ID: st.run.ID, CompletedAt: &end, CorrectCount: &c16, TotalTimeMs: &t32, Score: &s32}); err != nil {
		return err
	}
	st.run.Status, st.run.CompletedAt, st.run.CorrectCount, st.run.TotalTimeMs, st.run.Score = DailyCompleted, &end, &c16, &t32, &s32
	st.done = true
	if st.run.Kind != DailyOfficial {
		return nil
	}
	return s.d.Events.PublishTx(ctx, st.tx, EventDailyRunCompleted, end, DailyRunCompleted{
		RunID: st.run.ID, UserID: st.run.UserID, Day: st.run.Day.Format(time.DateOnly), CorrectCount: correct,
		TotalTimeMS: total, Score: score, CompletedAt: end, TierAtStart: st.run.AccountTierAtStart,
	})
}

// JoinDaily starts (first question after a short intro) or resumes a run
// and returns the frames that bring the player's screen up to date.
func (s *Service) JoinDaily(ctx context.Context, user, id uuid.UUID) ([]DailyFrame, error) {
	r, err := sqlc.New(s.d.DB).GetDailyRun(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && r.UserID != user) {
		return nil, apierror.New(apierror.NotFound)
	}
	if err != nil {
		return nil, err
	}
	if r.Status == DailyInProgress {
		err := postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
			q := sqlc.New(tx)
			r, err := q.GetDailyRunForUpdate(ctx, id)
			if err != nil {
				return err
			}
			answers, err := q.DailyAnswers(ctx, id)
			if err != nil || len(answers) > 0 || r.Status != DailyInProgress {
				return err
			}
			set, err := s.d.Questions.DailySet(ctx, r.Day)
			if err != nil {
				return err
			}
			_, err = s.showDaily(ctx, q, &settled{run: r, set: set}, 1, s.d.Clock.Now().Add(dailyIntroMS*time.Millisecond))
			return err
		})
		if err != nil {
			return nil, err
		}
	}
	st, err := s.settleDaily(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.dailyFrames(ctx, st, true)
}

// AnswerDaily records the answer to the open question, then reveals it
// and shows the next one.
func (s *Service) AnswerDaily(ctx context.Context, user, id uuid.UUID, pos int16, option uuid.UUID) ([]DailyFrame, error) {
	var st *settled
	err := postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		r, err := q.GetDailyRunForUpdate(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && r.UserID != user) {
			return apierror.New(apierror.NotFound)
		}
		if err != nil {
			return err
		}
		if r.Status != DailyInProgress {
			return apierror.New(apierror.MatchStateInvalid).With("reason", "run_"+r.Status)
		}
		answers, err := q.DailyAnswers(ctx, id)
		if err != nil {
			return err
		}
		if len(answers) == 0 {
			return apierror.New(apierror.MatchStateInvalid).With("reason", "not_started")
		}
		set, err := s.d.Questions.DailySet(ctx, r.Day)
		if err != nil {
			return err
		}
		st = &settled{tx: tx, run: r, answers: answers, set: set}
		now := s.d.Clock.Now()
		last := &st.answers[len(st.answers)-1]
		switch {
		case last.Position != pos:
			return apierror.New(apierror.AnswerLate)
		case last.OptionID != nil:
			return apierror.New(apierror.AnswerDuplicate)
		case now.After(last.DeadlineAt.Add(dailyGraceMS * time.Millisecond)):
			return apierror.New(apierror.AnswerLate)
		case now.Before(last.ShowAt.Add(-dailyGraceMS * time.Millisecond)):
			// Sent before the question was shown: no blind answers.
			return apierror.New(apierror.MatchStateInvalid).With("reason", "not_shown")
		}
		qu := set.Questions[pos-1]
		if !slices.Contains(qu.Options, option) {
			return apierror.New(apierror.AnswerInvalidOption)
		}
		resp := min(max(int(now.Sub(last.ShowAt).Milliseconds()), 0), DailyWindowMS)
		streak := 0
		for i := len(st.answers) - 2; i >= 0 && st.answers[i].IsCorrect; i-- {
			streak++
		}
		correct := option == qu.Correct
		pts := scoring.Current.Score(scoring.Input{
			Correct: correct, QuestionType: domain.QuestionType(string(qu.Type)), Difficulty: qu.Difficulty,
			ResponseMS: resp, WindowMS: DailyWindowMS, StreakBefore: streak,
		})
		resp32, pts32 := int32(resp), int32(pts.Total) //nolint:gosec // G115: ≤ window / ≤ 260
		if err := q.AnswerDailyQuestion(ctx, sqlc.AnswerDailyQuestionParams{
			OptionID: &option, ReceivedAt: &now, ResponseMs: &resp32, IsCorrect: correct, Points: pts32, RunID: id, Position: pos,
		}); err != nil {
			return err
		}
		last.OptionID, last.ReceivedAt, last.ResponseMs, last.IsCorrect, last.Points = &option, &now, &resp32, correct, pts32
		st.timedOut = append(st.timedOut, pos) // revealed below like a resolved question
		return s.advance(ctx, q, st, now)
	})
	if err != nil {
		return nil, err
	}
	return s.dailyFrames(ctx, st, false)
}

// dailyFrames are daily.revealed for questions resolved by this step,
// daily.question for the open one and daily.completed at the end; join
// also re-sends the open question.
func (s *Service) dailyFrames(ctx context.Context, st *settled, join bool) ([]DailyFrame, error) {
	runID := st.run.ID
	if st.run.Status == DailyVoidNotStared {
		return []DailyFrame{{Type: "daily.voided", Data: map[string]any{"run_id": runID, "reason": st.run.Status}}}, nil
	}
	var out []DailyFrame
	byPos := map[int16]*sqlc.GameDailyRunAnswer{}
	for i := range st.answers {
		byPos[st.answers[i].Position] = &st.answers[i]
	}
	for _, pos := range st.timedOut {
		a := byPos[pos]
		out = append(out, DailyFrame{Type: "daily.revealed", Data: map[string]any{
			"run_id": runID, "position": pos, "correct_option_id": st.set.Questions[pos-1].Correct,
			"option_id": a.OptionID, "correct": a.IsCorrect, "points": a.Points, "response_ms": a.ResponseMs,
		}})
	}
	if st.done || st.run.Status == DailyCompleted {
		out = append(out, DailyFrame{Type: "daily.completed", Data: map[string]any{
			"run_id": runID, "kind": st.run.Kind, "correct_count": st.run.CorrectCount, "total_time_ms": st.run.TotalTimeMs,
			"score": st.run.Score,
		}})
		return out, nil
	}
	if len(st.answers) == 0 {
		return out, nil
	}
	open := st.answers[len(st.answers)-1]
	if open.OptionID == nil && (join || slices.Contains(st.shown, open.Position)) {
		f, err := s.dailyQuestionFrame(ctx, st, &open)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, nil
}

func (s *Service) dailyQuestionFrame(ctx context.Context, st *settled, a *sqlc.GameDailyRunAnswer) (DailyFrame, error) {
	qu := st.set.Questions[a.Position-1]
	texts, err := s.d.Questions.Texts(ctx, []uuid.UUID{qu.ID}, []string{st.run.Locale})
	if err != nil {
		return DailyFrame{}, err
	}
	t := texts[qu.ID][st.run.Locale]
	options := make([]map[string]any, 0, len(a.OptionOrder))
	for _, p := range a.OptionOrder {
		id := qu.Options[p-1]
		options = append(options, map[string]any{"id": id, "text": t.Options[id]})
	}
	return DailyFrame{Type: "daily.question", Data: map[string]any{
		"run_id": st.run.ID, "kind": st.run.Kind, "position": a.Position, "total": len(st.set.Questions),
		"show_at": a.ShowAt.UnixMilli(), "deadline": a.DeadlineAt.UnixMilli(), "prompt": t.Prompt, "category": t.Category,
		"options": options,
	}}, nil
}

// SettleStaleDaily finishes runs their players left (abandoned apps): the
// remaining questions time out, a never-started run is voided.
func (s *Service) SettleStaleDaily(ctx context.Context) error {
	stale, err := sqlc.New(s.d.DB).StaleDailyRuns(ctx, s.d.Clock.Now().Add(-dailyStaleAfter))
	if err != nil {
		return err
	}
	for _, id := range stale {
		if _, err := s.settleDaily(ctx, id); err != nil {
			return err
		}
	}
	return nil
}
