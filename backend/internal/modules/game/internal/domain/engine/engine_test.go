package engine

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain/config"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain/scoring"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
)

// firstRandom always picks 0: deterministic shuffles, coin tosses, picks.
type firstRandom struct{}

func (firstRandom) Intn(int) int { return 0 }

type harness struct {
	t     *testing.T
	m     *Match
	now   int64
	users [2][]uuid.UUID
	out   []Output
	cats  []uuid.UUID
}

func question(cat uuid.UUID, qt domain.QuestionType, difficulty int) Question {
	n := 4
	if qt == domain.TrueFalse {
		n = 2
	}
	opts := make([]uuid.UUID, n)
	for i := range opts {
		opts[i] = uuid.New()
	}
	return Question{ID: uuid.New(), Revision: 1, Type: qt, Difficulty: difficulty, CategoryID: cat, SubcategoryID: uuid.New(), Options: opts, Correct: opts[0]}
}

func newHarness(t *testing.T, c config.Config, questions int) *harness {
	t.Helper()
	h := &harness{t: t, now: 1_000_000, cats: []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}}
	if c.CategoryMode == domain.MixedBattle {
		c.CategoryIDs = h.cats
	}
	var seats []Seat
	for side := 1; side <= 2; side++ {
		for seat := 1; seat <= c.Format.SideSize(); seat++ {
			u := uuid.New()
			h.users[side-1] = append(h.users[side-1], u)
			seats = append(seats, Seat{UserID: u, Side: side, Seat: seat, Locale: "tr"})
		}
	}
	var qs []Question
	for i := range questions {
		cat := h.cats[i%3]
		qs = append(qs, question(cat, domain.MultipleChoice, 2+i%3))
	}
	m, o, err := New(Setup{MatchID: uuid.New(), Config: c, Seats: seats, Questions: qs, Rules: scoring.Ruleset1, Random: firstRandom{}, NowMS: h.now})
	if err != nil {
		t.Fatal(err)
	}
	h.m = m
	h.out = append(h.out, o)
	return h
}

func duel(t domain.MatchType) config.Config {
	origin := domain.Challenge
	if t == domain.Ranked {
		origin = domain.Matchmaking // R03
	}
	return config.Config{MatchType: t, Format: domain.Format1v1, Origin: origin}
}

func (h *harness) a() uuid.UUID { return h.users[0][0] }
func (h *harness) b() uuid.UUID { return h.users[1][0] }

func (h *harness) record(o Output) Output { h.out = append(h.out, o); return o }

func (h *harness) must(o Output, err error) Output {
	h.t.Helper()
	if err != nil {
		h.t.Fatal(err)
	}
	return h.record(o)
}

func (h *harness) at(ms int64) { h.now = ms }

// start readies everyone and runs the countdown.
func (h *harness) start() {
	h.t.Helper()
	for _, side := range h.users {
		for _, u := range side {
			h.must(h.m.Ready(u, h.now))
		}
	}
	h.at(h.m.StartsAtMS)
	h.record(h.m.Fire(Timer{Kind: TimerStart}, h.now))
	if h.m.Status != domain.StatusInProgress {
		h.t.Fatalf("not started: %s", h.m.Status)
	}
}

func (h *harness) correct() uuid.UUID { return h.m.Round.Question.Correct }
func (h *harness) wrong() uuid.UUID   { return h.m.Round.Question.Options[1] }

// tick fires the deadline then the reveal timer of the current round.
func (h *harness) deadline() Output {
	h.at(h.m.Round.DeadlineMS + 250)
	return h.record(h.m.Fire(Timer{Kind: TimerRoundDeadline, Round: h.m.Round.No}, h.now))
}

func (h *harness) reveal() Output {
	h.at(h.now + RevealMS)
	return h.record(h.m.Fire(Timer{Kind: TimerReveal, Round: h.m.Round.No}, h.now))
}

func (h *harness) answer(u, opt uuid.UUID, afterShowMS int64) (Output, error) {
	h.at(h.m.Round.ShowAtMS + afterShowMS)
	o, err := h.m.Answer(u, h.m.Round.No, opt, h.now)
	h.record(o)
	return o, err
}

func code(err error) apierror.Code {
	if e := apierror.From(err); e != nil {
		return e.Code
	}
	return ""
}

func types(o Output) []string {
	var out []string
	for _, e := range o.Events {
		out = append(out, e.Type)
	}
	return out
}

func (h *harness) allEvents() []Event {
	var out []Event
	for _, o := range h.out {
		out = append(out, o.Events...)
	}
	return out
}

// ---------- Lifecycle ----------

func TestLC02ReadyStartAndCreation(t *testing.T) {
	h := newHarness(t, duel(domain.QuickDuel), 7)
	first := h.out[0]
	if h.m.Status != domain.StatusWaiting || len(first.Timers) != 1 || first.Timers[0].Kind != TimerReadyDeadline ||
		first.Timers[0].AtMS != h.now+ReadyDeadlineMS || first.Transitions[0].To != domain.StatusWaiting {
		t.Fatalf("created: %+v", first)
	}
	o := h.must(h.m.Ready(h.a(), h.now))
	if h.m.Status != domain.StatusWaiting || len(o.Timers) != 0 {
		t.Fatal("one ready is not enough")
	}
	o = h.must(h.m.Ready(h.b(), h.now+5_000))
	if h.m.Status != domain.StatusReady || o.Timers[0].Kind != TimerStart || o.Timers[0].AtMS != h.now+5_000+CountdownMS {
		t.Fatalf("all ready: %s %+v", h.m.Status, o)
	}
	h.at(h.m.StartsAtMS)
	o = h.record(h.m.Fire(Timer{Kind: TimerStart}, h.now))
	if h.m.Status != domain.StatusInProgress || !slices.Contains(types(o), "round.started") || h.m.RoundNo != 1 {
		t.Fatalf("started: %v", types(o))
	}
	if o.Transitions[0].From != domain.StatusReady || o.Transitions[0].To != domain.StatusInProgress {
		t.Fatalf("transition %+v", o.Transitions)
	}
}

func TestLC01StaleCommandsIgnored(t *testing.T) {
	h := newHarness(t, duel(domain.QuickDuel), 7)
	if o := h.m.Fire(Timer{Kind: TimerStart}, h.now); len(o.Events) != 0 || h.m.Status != domain.StatusWaiting {
		t.Fatal("start timer in waiting must be ignored")
	}
	h.start()
	if o := h.m.Fire(Timer{Kind: TimerReadyDeadline}, h.now); len(o.Events) != 0 {
		t.Fatal("ready deadline after start must be ignored")
	}
	if o := h.m.Fire(Timer{Kind: TimerRoundDeadline, Round: 9}, h.now); len(o.Events) != 0 {
		t.Fatal("deadline of another round must be ignored")
	}
	if _, err := h.m.Ready(h.a(), h.now); code(err) != apierror.MatchStateInvalid {
		t.Fatalf("ready in progress: %v", err)
	}
	if _, err := h.m.Ready(uuid.New(), h.now); code(err) != apierror.NotAParticipant {
		t.Fatalf("stranger: %v", err)
	}
}

func TestLC03LC04LC05ReadyDeadline(t *testing.T) {
	h := newHarness(t, duel(domain.QuickDuel), 7)
	h.must(h.m.Ready(h.a(), h.now))
	if o := h.must(h.m.Ready(h.a(), h.now+1)); len(o.Events) != 0 {
		t.Fatal("LC04 duplicate ready must be ignored")
	}
	if _, err := h.m.Ready(h.b(), h.m.ReadyDeadMS+1); code(err) != apierror.MatchStateInvalid {
		t.Fatalf("LC05 ready after the deadline: %v", err)
	}
	o := h.record(h.m.Fire(Timer{Kind: TimerReadyDeadline}, h.m.ReadyDeadMS))
	if h.m.Status != domain.StatusCancelled || o.Result.CancelReason != domain.CancelNotAllReady || o.Result.WinnerSide != 0 {
		t.Fatalf("LC03: %s %+v", h.m.Status, o.Result)
	}
}

func TestLC07LeaveDuringCountdown(t *testing.T) {
	h := newHarness(t, duel(domain.Ranked), 10)
	h.must(h.m.Ready(h.a(), h.now))
	h.must(h.m.Ready(h.b(), h.now))
	o := h.must(h.m.Leave(h.b(), h.now+1_000))
	if h.m.Status != domain.StatusCancelled || o.Result.CancelReason != domain.CancelLeftBeforeStart {
		t.Fatalf("leave in countdown: %+v", o.Result)
	}
	if o := h.m.Fire(Timer{Kind: TimerStart}, h.m.StartsAtMS); len(o.Events) != 0 {
		t.Fatal("a cancelled match never starts")
	}
}

func TestDisconnectDuringCountdownCancels(t *testing.T) {
	h := newHarness(t, duel(domain.QuickDuel), 7)
	h.must(h.m.Ready(h.a(), h.now))
	h.must(h.m.Ready(h.b(), h.now))
	h.record(h.m.Disconnect(h.b(), h.now+500))
	o := h.record(h.m.Fire(Timer{Kind: TimerStart}, h.m.StartsAtMS))
	if o.Result == nil || o.Result.CancelReason != domain.CancelLeftBeforeStart {
		t.Fatalf("T5: %+v", o.Result)
	}
}

// ---------- Rounds ----------

func TestRD01EarlyLockAndRD09Reveal(t *testing.T) {
	h := newHarness(t, duel(domain.QuickDuel), 7)
	h.start()
	if _, err := h.answer(h.a(), h.correct(), 3_000); err != nil {
		t.Fatal(err)
	}
	o, err := h.answer(h.b(), h.wrong(), 5_000)
	if err != nil {
		t.Fatal(err)
	}
	if got := types(o); !slices.Equal(got, []string{"round.answer_received", "round.locked", "round.revealed"}) {
		t.Fatalf("early lock: %v", got)
	}
	rec := o.Round
	if rec == nil || rec.No != 1 || len(rec.Answers) != 2 || rec.Sides[0].Points != 170 || rec.Sides[1].Points != 0 { // difficulty 2: 90 + 80
		t.Fatalf("round record %+v", rec)
	}
	if o.Timers[0].Kind != TimerReveal || o.Timers[0].AtMS != h.now+RevealMS {
		t.Fatalf("reveal timer %+v", o.Timers)
	}
	o = h.reveal()
	if h.m.RoundNo != 2 || !slices.Contains(types(o), "round.started") {
		t.Fatalf("next round after 3 s: %v", types(o))
	}
}

func TestRD02TimeoutRow(t *testing.T) {
	h := newHarness(t, duel(domain.QuickDuel), 7)
	h.start()
	if _, err := h.answer(h.a(), h.correct(), 1_000); err != nil {
		t.Fatal(err)
	}
	o := h.deadline()
	if o.Round == nil {
		t.Fatal("deadline locks the round")
	}
	var timeout *AnswerRecord
	for i := range o.Round.Answers {
		if o.Round.Answers[i].UserID == h.b() {
			timeout = &o.Round.Answers[i]
		}
	}
	if timeout == nil || timeout.OptionID != nil || timeout.Correct || timeout.ResponseMS != nil {
		t.Fatalf("timeout row %+v", timeout)
	}
}

func TestRD03RD04RD05Submissions(t *testing.T) {
	h := newHarness(t, duel(domain.QuickDuel), 7)
	h.start()
	deadline := h.m.Round.DeadlineMS
	// rtt 0 → grace 100 ms
	h.at(deadline + 101)
	if _, err := h.m.Answer(h.a(), 1, h.correct(), h.now); code(err) != apierror.AnswerLate {
		t.Fatalf("RD03 +1 ms: %v", err)
	}
	h.at(deadline + 100)
	if _, err := h.m.Answer(h.a(), 1, uuid.New(), h.now); code(err) != apierror.AnswerInvalidOption {
		t.Fatalf("RD05: %v", err)
	}
	if _, err := h.m.Answer(h.a(), 1, h.correct(), h.now); err != nil {
		t.Fatalf("RD03 exactly at the boundary: %v", err)
	}
	if _, err := h.m.Answer(h.a(), 1, h.wrong(), h.now); code(err) != apierror.AnswerDuplicate {
		t.Fatalf("RD04: %v", err)
	}
	if _, err := h.m.Answer(h.a(), 2, h.correct(), h.now); code(err) != apierror.MatchStateInvalid {
		t.Fatalf("another round: %v", err)
	}
}

func TestRD06CorrectOptionNeverBeforeReveal(t *testing.T) {
	h := newHarness(t, duel(domain.Ranked), 10)
	h.start()
	for range 10 {
		if h.m.Status != domain.StatusInProgress {
			break
		}
		_, _ = h.answer(h.a(), h.correct(), 2_000)
		_, _ = h.answer(h.b(), h.wrong(), 2_000)
		h.reveal()
	}
	for _, e := range h.allEvents() {
		if e.Type == "round.revealed" {
			continue
		}
		if _, leaked := e.Payload["correct_option_id"]; leaked {
			t.Fatalf("%s carries the correct option", e.Type)
		}
	}
}

// TestFullQuickDuel plays 7 rounds: LC10 completed-match invariants, the
// score invariant and a gap-free event sequence (RD08).
func TestFullQuickDuel(t *testing.T) {
	h := newHarness(t, duel(domain.QuickDuel), 7)
	h.start()
	sum := [2]int{}
	for round := 1; round <= 7; round++ {
		_, _ = h.answer(h.a(), h.correct(), 2_000)
		o, _ := h.answer(h.b(), h.wrong(), 1_000)
		for i := range 2 {
			sum[i] += o.Round.Sides[i].Points
		}
		h.reveal()
	}
	r := h.m.Result()
	if h.m.Status != domain.StatusCompleted || r == nil || r.EndReason != domain.EndAllRoundsPlayed || r.WinnerSide != 1 || r.Draw ||
		r.Sides[0].Result != domain.Win || r.Sides[1].Result != domain.Loss || r.RoundsPlayed != 7 {
		t.Fatalf("result %+v", r)
	}
	if r.Sides[0].Score != sum[0] || r.Sides[1].Score != sum[1] || r.Sides[0].Correct != 7 {
		t.Fatalf("score invariant: %+v vs %v", r.Sides, sum)
	}
	for _, p := range r.Participants {
		if p.Status != domain.PFinished || p.Answered != 7 {
			t.Fatalf("participant %+v", p)
		}
	}
	for i, e := range h.allEvents() {
		if e.Seq != i+1 {
			t.Fatalf("event %d has seq %d", i+1, e.Seq)
		}
	}
	if h.allEvents()[len(h.allEvents())-1].Type != "match.completed" {
		t.Fatal("last event")
	}
}

func TestEC01DrawAndTieBreak(t *testing.T) {
	h := newHarness(t, duel(domain.QuickDuel), 7)
	h.start()
	for range 7 {
		_, _ = h.answer(h.a(), h.correct(), 2_000)
		_, _ = h.answer(h.b(), h.correct(), 2_000)
		h.reveal()
	}
	if r := h.m.Result(); !r.Draw || r.WinnerSide != 0 || r.Sides[0].Result != domain.Draw {
		t.Fatalf("all equal → draw: %+v", r)
	}
}

func TestMT02SuddenDeathAndMoreQuestions(t *testing.T) {
	h := newHarness(t, duel(domain.SuddenDeath), 2)
	h.start()
	_, _ = h.answer(h.a(), h.correct(), 1_000)
	_, _ = h.answer(h.b(), h.correct(), 1_000)
	h.reveal()
	_, _ = h.answer(h.a(), h.wrong(), 1_000)
	_, _ = h.answer(h.b(), h.wrong(), 1_000)
	o := h.reveal()
	if h.m.Phase != PhaseNeedsMore || o.NeedQuestions != 10 {
		t.Fatalf("pool exhausted: phase %s need %d", h.m.Phase, o.NeedQuestions)
	}
	o = h.record(h.m.AddQuestions([]Question{question(uuid.New(), domain.MultipleChoice, 2)}, h.now))
	if !slices.Contains(types(o), "round.started") || h.m.RoundNo != 3 {
		t.Fatalf("continues: %v", types(o))
	}
	_, _ = h.answer(h.a(), h.wrong(), 1_000)
	_, _ = h.answer(h.b(), h.correct(), 9_000)
	h.reveal()
	if r := h.m.Result(); r.EndReason != domain.EndSuddenDeathDecided || r.WinnerSide != 2 {
		t.Fatalf("decided: %+v", r)
	}
}

func TestMT09MarathonHalftime(t *testing.T) {
	h := newHarness(t, duel(domain.Marathon), 25)
	h.start()
	for range 12 {
		_, _ = h.answer(h.a(), h.correct(), 1_000)
		_, _ = h.answer(h.b(), h.correct(), 1_000)
		if h.m.RoundNo < 12 {
			h.reveal()
		}
	}
	o := h.reveal()
	if h.m.Phase != PhaseHalftime || o.Timers[0].Kind != TimerHalftime || o.Timers[0].AtMS != h.now+15_000 {
		t.Fatalf("halftime: %s %+v", h.m.Phase, o.Timers)
	}
	o = h.record(h.m.Fire(Timer{Kind: TimerHalftime}, h.now+15_000))
	if h.m.RoundNo != 13 || !slices.Contains(types(o), "round.started") {
		t.Fatal("round 13 after the break")
	}
}

// ---------- Edge cases ----------

func TestEC03Reconnect(t *testing.T) {
	h := newHarness(t, duel(domain.QuickDuel), 7)
	h.start()
	start := h.now
	h.record(h.m.Disconnect(h.b(), start))
	h.record(h.m.Reconnect(h.b(), start+19_000))
	if p := h.m.participant(h.b()); p.Status != domain.PPlaying || p.AwayMS != 19_000 {
		t.Fatalf("19 s: %+v", p)
	}
	h.record(h.m.Disconnect(h.b(), start+20_000))
	o := h.record(h.m.Fire(Timer{Kind: TimerReconnect, User: h.b()}, start+40_000))
	if o.Result == nil || o.Result.EndReason != domain.EndForfeit || o.Result.WinnerSide != 1 {
		t.Fatalf("window over → forfeit: %+v", o.Result)
	}
}

func TestEC04FourthDisconnectForfeits(t *testing.T) {
	h := newHarness(t, duel(domain.QuickDuel), 7)
	h.start()
	for i := range 3 {
		h.record(h.m.Disconnect(h.b(), h.now+int64(i)*100))
		h.record(h.m.Reconnect(h.b(), h.now+int64(i)*100+10))
	}
	o := h.record(h.m.Disconnect(h.b(), h.now+1_000))
	if o.Result == nil || o.Result.EndReason != domain.EndForfeit {
		t.Fatalf("4th disconnect: %+v", o.Result)
	}
}

func TestEC05LeaveGivesWin(t *testing.T) {
	h := newHarness(t, duel(domain.Ranked), 10)
	h.start()
	o := h.must(h.m.Leave(h.a(), h.now))
	if o.Result.EndReason != domain.EndAllOpponentsLeft || o.Result.WinnerSide != 2 || o.Result.Sides[0].Result != domain.Loss {
		t.Fatalf("leave: %+v", o.Result)
	}
	if p := o.Result.Participants[0]; p.Status != domain.PLeft {
		t.Fatalf("leaver status %s", p.Status)
	}
}

func team(f domain.Format, mode domain.TeamMode, agg domain.Aggregation) config.Config {
	return config.Config{MatchType: domain.QuickDuel, Format: f, TeamMode: mode, Aggregation: agg, Origin: domain.PrivateRoom}
}

func TestEC06ShortHandedThenSideForfeits(t *testing.T) {
	h := newHarness(t, team(domain.Format3v3, domain.FastTeamRound, domain.Average), 7)
	h.start()
	h.must(h.m.Leave(h.users[0][0], h.now))
	h.record(h.m.Disconnect(h.users[0][1], h.now))
	if h.m.Status != domain.StatusInProgress {
		t.Fatal("one member gone: continue")
	}
	h.must(h.m.Leave(h.users[0][2], h.now+10))
	o := h.record(h.m.Fire(Timer{Kind: TimerReconnect, User: h.users[0][1]}, h.now+30_000))
	if o.Result == nil || o.Result.EndReason != domain.EndForfeit || o.Result.WinnerSide != 2 {
		t.Fatalf("all gone: %+v", o.Result)
	}
}

func TestEC07BothGoneSameCheckCancels(t *testing.T) {
	h := newHarness(t, duel(domain.QuickDuel), 7)
	h.start()
	h.record(h.m.Disconnect(h.a(), h.now))
	h.record(h.m.Disconnect(h.b(), h.now+500))
	o := h.record(h.m.Fire(Timer{Kind: TimerReconnect, User: h.a()}, h.now+25_000))
	if o.Result == nil || o.Result.Status != domain.StatusCancelled || o.Result.CancelReason != domain.CancelAllSidesAbsent {
		t.Fatalf("both windows over: %+v", o.Result)
	}
}

// ---------- Team modes in the engine ----------

func TestTA05IndividualTurnsInEngine(t *testing.T) {
	h := newHarness(t, team(domain.Format2v2, domain.IndividualTurns, domain.NoAggregation), 7)
	h.start()
	if h.m.Round.Turn != [2]uuid.UUID{h.users[0][0], h.users[1][0]} {
		t.Fatalf("round 1 turn holders %v", h.m.Round.Turn)
	}
	if _, err := h.answer(h.users[0][1], h.correct(), 1_000); code(err) != apierror.NotYourTurn {
		t.Fatalf("not your turn: %v", err)
	}
	_, _ = h.answer(h.users[0][0], h.correct(), 1_000)
	o, _ := h.answer(h.users[1][0], h.correct(), 1_000)
	if len(o.Round.Answers) != 2 {
		t.Fatalf("only turn holders have rows: %+v", o.Round.Answers)
	}
	h.record(h.m.Disconnect(h.users[1][1], h.now))
	h.reveal()
	if h.m.Round.Turn != [2]uuid.UUID{h.users[0][1], h.users[1][0]} {
		t.Fatalf("TA06: disconnected seat 2 skipped on side 2: %v", h.m.Round.Turn)
	}
}

func TestSharedAnswerInEngine(t *testing.T) {
	h := newHarness(t, team(domain.Format2v2, domain.SharedAnswer, domain.NoAggregation), 7)
	h.start()
	if h.m.Round.DeadlineMS-h.m.Round.ShowAtMS != 20_000 {
		t.Fatal("MT08: Shared Answer window 15 + 5 s")
	}
	if _, err := h.m.Answer(h.users[0][0], 1, h.correct(), h.now); code(err) != apierror.MatchStateInvalid {
		t.Fatalf("answer in shared mode: %v", err)
	}
	o := h.must(h.m.Propose(h.users[0][0], 1, h.wrong(), h.m.Round.ShowAtMS+1_000))
	if o.Events[0].Type != "round.proposals" || o.Events[0].To.Side != 1 {
		t.Fatalf("proposals go to own side only: %+v", o.Events)
	}
	h.must(h.m.Lock(h.users[0][1], 1, h.correct(), h.m.Round.ShowAtMS+2_000))
	if _, err := h.m.Lock(h.users[0][0], 1, h.wrong(), h.m.Round.ShowAtMS+2_500); code(err) != apierror.AnswerDuplicate {
		t.Fatalf("second lock: %v", err)
	}
	h.must(h.m.Propose(h.users[1][1], 1, h.correct(), h.m.Round.ShowAtMS+3_000))
	o = h.deadline()
	side1, side2 := o.Round.Sides[0], o.Round.Sides[1]
	// difficulty 2 (base 90), window 20 s: lock at 2 s → 90 + 90; fallback → no speed bonus.
	if !side1.Correct || side1.Points != 180 || !side2.Correct || side2.Points != 90 || side2.Detail["fallback"] != true {
		t.Fatalf("lock vs fallback: %+v / %+v", side1, side2)
	}
	if len(o.Round.Answers) != 2 || !o.Round.Answers[0].IsSideAnswer {
		t.Fatalf("one side answer per side: %+v", o.Round.Answers)
	}
}

// ---------- Category modes ----------

func TestBlindChallengeHidesCategory(t *testing.T) {
	c := duel(domain.QuickDuel)
	c.CategoryMode = domain.BlindChallenge
	h := newHarness(t, c, 7)
	h.start()
	var started, revealed map[string]any
	_, _ = h.answer(h.a(), h.correct(), 1_000)
	_, _ = h.answer(h.b(), h.correct(), 1_000)
	for _, e := range h.allEvents() {
		switch e.Type {
		case "round.started":
			started = e.Payload
		case "round.revealed":
			revealed = e.Payload
		}
	}
	if _, ok := started["category_id"]; ok {
		t.Fatal("category shown before the reveal")
	}
	if _, ok := revealed["category_id"]; !ok {
		t.Fatal("category missing at the reveal")
	}
}

func TestCM03MixedBattlePicks(t *testing.T) {
	c := duel(domain.QuickDuel)
	c.CategoryMode = domain.MixedBattle
	h := newHarness(t, c, 21)
	h.start()
	if h.m.Phase != PhasePreRound || h.m.Round.PickerSide != 1 || h.m.Round.Picker != h.a() {
		t.Fatalf("round 1 picker: %+v", h.m.Round)
	}
	if _, err := h.m.Pick(h.b(), 1, h.cats[0], h.now); code(err) != apierror.PickNotAllowed {
		t.Fatalf("non-picker: %v", err)
	}
	if _, err := h.m.Pick(h.a(), 1, uuid.New(), h.now); code(err) != apierror.PickNotAllowed {
		t.Fatalf("foreign category: %v", err)
	}
	play := func(cat uuid.UUID, picker uuid.UUID) {
		t.Helper()
		h.must(h.m.Pick(picker, h.m.RoundNo, cat, h.now))
		if h.m.Round.Question.CategoryID != cat {
			t.Fatalf("question from %v, picked %v", h.m.Round.Question.CategoryID, cat)
		}
		_, _ = h.answer(h.a(), h.correct(), 1_000)
		_, _ = h.answer(h.b(), h.correct(), 1_000)
		h.reveal()
	}
	play(h.cats[0], h.a())
	if h.m.Round.PickerSide != 2 {
		t.Fatal("picking side alternates")
	}
	play(h.cats[0], h.b())
	play(h.cats[0], h.a())
	if _, err := h.m.Pick(h.b(), h.m.RoundNo, h.cats[0], h.now); code(err) != apierror.PickNotAllowed {
		t.Fatalf("4th consecutive pick of the same category: %v", err)
	}
	o := h.record(h.m.Fire(Timer{Kind: TimerPick, Round: h.m.RoundNo}, h.m.Round.PickDeadMS))
	if !slices.Contains(types(o), "round.started") || h.m.Round.Question.CategoryID == h.cats[0] {
		t.Fatalf("no pick → server picks another category: %v", types(o))
	}
}
