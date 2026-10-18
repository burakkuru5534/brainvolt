// Package engine is the match and round state machine of docs/game/
// match-lifecycle.md and round-lifecycle.md as a pure, single-writer engine:
// the match owner (Phase 4) feeds it commands one at a time — client
// intents, timer expiries, admin actions — each carrying the server time,
// and carries out what it returns: client events (with gap-free sequence
// numbers), timers to arm, rounds and results to persist. No I/O, no
// goroutines, no clock reads: tests drive it with plain numbers.
package engine

import (
	"errors"
	"fmt"
	"slices"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain/config"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain/matchtype"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain/scoring"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain/teammode"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain/timing"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
)

// Durations of the lifecycle (D-034, D-035, D-036).
const (
	ReadyDeadlineMS = 20_000
	CountdownMS     = 3_000
	RevealMS        = 3_000
	PickMS          = 10_000
	// MaxSameCategoryInARow caps Mixed Battle picks (D-043).
	MaxSameCategoryInARow = 3
)

// Random is the engine's randomness: option shuffles, the Mixed Battle
// coin toss and default picks (crypto-random in production).
type Random interface {
	Intn(n int) int
}

// Question is one selected question as the engine needs it.
type Question struct {
	ID            uuid.UUID
	Revision      int32
	Type          domain.QuestionType
	Difficulty    int // difficulty_used, fixed at selection
	CategoryID    uuid.UUID
	SubcategoryID uuid.UUID
	// Options in canonical order (positions 1…n).
	Options []uuid.UUID
	Correct uuid.UUID
}

// Seat is one participant at creation.
type Seat struct {
	UserID uuid.UUID
	Side   int // 1 or 2
	Seat   int // 1…4
	Locale string
}

// Setup creates a match.
type Setup struct {
	MatchID uuid.UUID
	Config  config.Config
	Seats   []Seat
	// Questions: the ordered selection; Mixed Battle: a pool covering its
	// three categories; Sudden Death: the first batch.
	Questions []Question
	Rules     scoring.Ruleset
	Random    Random
	NowMS     int64
}

// Participant is a participant's live state.
type Participant struct {
	Seat
	Status         domain.ParticipantStatus
	RTTMS          int
	Disconnects    int
	AwayMS         int
	DisconnectedAt int64
	Streak         int
	Contribution   int
	Correct        int
	Answered       int
}

// SideState is one side's running totals.
type SideState struct {
	Score             int
	Correct           int
	CorrectResponseMS int
	Streak            int
	LastTurnSeat      int
	LastPickSeat      int
	ForfeitedAt       int64
}

// Phase is the round sub-state (round-lifecycle §1) plus the gaps
// between rounds.
type Phase string

// Phases.
const (
	PhaseLobby     Phase = "lobby"
	PhasePreRound  Phase = "pre_round"
	PhaseQuestion  Phase = "question"
	PhaseReveal    Phase = "reveal"
	PhaseHalftime  Phase = "halftime"
	PhaseNeedsMore Phase = "needs_questions"
	PhaseOver      Phase = "over"
)

// answer is an accepted submission.
type answer struct {
	option     uuid.UUID
	receivedMS int64
	responseMS int
}

// Round is the open round.
type Round struct {
	No          int
	Question    Question
	OptionOrder []int // canonical positions in shown order
	DispatchMS  int64
	ShowAtMS    int64
	DeadlineMS  int64
	// Turn holders per side (Individual Turns), 0-indexed by side-1.
	Turn       [2]uuid.UUID
	answers    map[uuid.UUID]answer
	proposals  [2][]teammode.Proposal
	locks      [2]*teammode.Lock
	PickerSide int
	Picker     uuid.UUID
	PickDeadMS int64
}

// Match is the engine state of one match.
type Match struct {
	ID           uuid.UUID
	Config       config.Config
	Derived      config.Derived
	Type         matchtype.Type
	Rules        scoring.Ruleset
	Status       domain.Status
	Phase        Phase
	CreatedMS    int64
	ReadyDeadMS  int64
	StartsAtMS   int64
	Participants []*Participant
	Sides        [2]SideState
	RoundNo      int
	Round        *Round

	questions     []Question
	used          map[uuid.UUID]bool
	pickHistory   []uuid.UUID
	firstPicker   int
	seq           int
	transitionSeq int
	lastRound     *RoundRecord
	random        Random
	result        *Result
}

// ---------- Outputs ----------

// Audience says who receives an event.
type Audience struct {
	All  bool
	Side int       // 1 or 2: one side only
	User uuid.UUID // one participant only
}

// Event is a client-visible fact (websocket.md §4 names), sequenced.
type Event struct {
	Seq     int
	Type    string
	To      Audience
	Payload map[string]any
}

// TimerKind names a server timer.
type TimerKind string

// Timers (timing.md §7).
const (
	TimerReadyDeadline TimerKind = "ready_deadline"
	TimerStart         TimerKind = "start"
	TimerPick          TimerKind = "pick"
	TimerRoundDeadline TimerKind = "round_deadline"
	TimerReveal        TimerKind = "reveal"
	TimerHalftime      TimerKind = "halftime"
	TimerReconnect     TimerKind = "reconnect"
)

// Timer identifies one armed timer; an expiry for a timer the match has
// moved past is ignored (compare-and-set).
type Timer struct {
	Kind  TimerKind
	Round int
	User  uuid.UUID
	AtMS  int64
}

// Transition is one lifecycle step for game.match_transitions.
type Transition struct {
	Seq     int
	From    domain.Status
	To      domain.Status
	Reason  string
	Actor   string // system | participant | admin
	ActorID uuid.UUID
	AtMS    int64
}

// Output is what one command produced; the owner applies it in order:
// persist (round, transition, result), then send events, then arm timers.
type Output struct {
	Events      []Event
	Timers      []Timer
	Transitions []Transition
	Round       *RoundRecord
	Result      *Result
	// NeedQuestions asks the owner for more questions (Sudden Death),
	// answered with AddQuestions.
	NeedQuestions int
}

func (m *Match) emit(o *Output, typ string, to Audience, payload map[string]any) {
	m.seq++
	o.Events = append(o.Events, Event{Seq: m.seq, Type: typ, To: to, Payload: payload})
}

var everyone = Audience{All: true}

// ---------- Creation (T1) ----------

// New creates a match in `waiting` (T1). Configuration validity, side
// sizes, live locks and question availability are checked by the service
// before; New re-checks what it relies on and fails rather than starting
// an impossible match.
func New(s Setup) (*Match, Output, error) {
	var o Output
	if vs := config.Validate(s.Config); len(vs) > 0 {
		return nil, o, fmt.Errorf("engine: invalid configuration: %s", config.Error(vs))
	}
	d := config.Derive(s.Config)
	t := matchtype.For(s.Config.MatchType)
	if s.Config.RankedLength {
		t = matchtype.RankedLengthQuickDuel()
	}
	size := s.Config.Format.SideSize()
	count := [3]int{}
	for _, seat := range s.Seats {
		if seat.Side != 1 && seat.Side != 2 {
			return nil, o, fmt.Errorf("engine: side %d", seat.Side)
		}
		count[seat.Side]++
	}
	if count[1] != size || count[2] != size {
		return nil, o, fmt.Errorf("engine: sides of %d and %d for format %s", count[1], count[2], s.Config.Format)
	}
	if s.Random == nil {
		return nil, o, errors.New("engine: no random source")
	}
	m := &Match{
		ID: s.MatchID, Config: s.Config, Derived: d, Type: t, Rules: s.Rules,
		Status: domain.StatusWaiting, Phase: PhaseLobby, CreatedMS: s.NowMS,
		ReadyDeadMS: s.NowMS + ReadyDeadlineMS, questions: slices.Clone(s.Questions),
		used: map[uuid.UUID]bool{}, random: s.Random,
	}
	seats := slices.Clone(s.Seats)
	slices.SortFunc(seats, func(a, b Seat) int { return (a.Side*10 + a.Seat) - (b.Side*10 + b.Seat) })
	for _, seat := range seats {
		m.Participants = append(m.Participants, &Participant{Seat: seat, Status: domain.PJoined})
	}
	if s.Config.CategoryMode == domain.MixedBattle {
		m.firstPicker = 1 + s.Random.Intn(2) // coin toss, shown in the lobby
	}
	m.transition(&o, "", domain.StatusWaiting, "created", "system", uuid.Nil, s.NowMS)
	m.emit(&o, "match.created", everyone, map[string]any{
		"ready_deadline_at": m.ReadyDeadMS, "first_picker_side_no": m.firstPicker,
	})
	o.Timers = append(o.Timers, Timer{Kind: TimerReadyDeadline, AtMS: m.ReadyDeadMS})
	return m, o, nil
}

func (m *Match) transition(o *Output, from, to domain.Status, reason, actor string, actorID uuid.UUID, now int64) {
	m.transitionSeq++
	o.Transitions = append(o.Transitions, Transition{
		Seq: m.transitionSeq, From: from, To: to, Reason: reason, Actor: actor, ActorID: actorID, AtMS: now,
	})
}

// participant returns the participant for user.
func (m *Match) participant(user uuid.UUID) *Participant {
	for _, p := range m.Participants {
		if p.UserID == user {
			return p
		}
	}
	return nil
}

func (m *Match) side(n int) []*Participant {
	var out []*Participant
	for _, p := range m.Participants {
		if p.Side == n {
			out = append(out, p)
		}
	}
	return out
}

// errors returned to the sender of an intent (errors.md codes).
func reject(code apierror.Code, reason string) error {
	return apierror.New(code).With("reason", reason)
}

var (
	errNotParticipant = apierror.New(apierror.NotAParticipant)
	errState          = func(reason string) error { return reject(apierror.MatchStateInvalid, reason) }
)

// ---------- Ready (T2) and start (T4) ----------

// Ready marks a participant ready; the last one moves the match to `ready`
// with a 3 s countdown (T2).
func (m *Match) Ready(user uuid.UUID, now int64) (Output, error) {
	var o Output
	p := m.participant(user)
	if p == nil {
		return o, errNotParticipant
	}
	if m.Status != domain.StatusWaiting {
		return o, errState("not_waiting")
	}
	if p.Status == domain.PReady {
		return o, nil // duplicate ready: ignored (LC04)
	}
	if now > m.ReadyDeadMS {
		return o, errState("ready_deadline_passed") // LC05: the deadline wins
	}
	p.Status = domain.PReady
	m.emit(&o, "match.participant_ready", everyone, map[string]any{"user_id": user})
	for _, q := range m.Participants {
		if q.Status != domain.PReady {
			return o, nil
		}
	}
	m.Status = domain.StatusReady
	m.StartsAtMS = now + CountdownMS
	m.transition(&o, domain.StatusWaiting, domain.StatusReady, "all_ready", "system", uuid.Nil, now)
	m.emit(&o, "match.starting", everyone, map[string]any{"starts_at": m.StartsAtMS})
	o.Timers = append(o.Timers, Timer{Kind: TimerStart, AtMS: m.StartsAtMS})
	return o, nil
}

// ---------- Timers ----------

// Fire handles an expired timer; stale timers are ignored.
func (m *Match) Fire(t Timer, now int64) Output {
	var o Output
	switch t.Kind {
	case TimerReadyDeadline:
		if m.Status == domain.StatusWaiting {
			m.cancel(&o, domain.CancelNotAllReady, "system", uuid.Nil, now)
		}
	case TimerStart:
		if m.Status != domain.StatusReady {
			break
		}
		for _, p := range m.Participants {
			if p.Status != domain.PReady {
				// Left or disconnected during the countdown: not started (T5).
				m.cancel(&o, domain.CancelLeftBeforeStart, "system", uuid.Nil, now)
				return o
			}
		}
		m.Status = domain.StatusInProgress
		for _, p := range m.Participants {
			p.Status = domain.PPlaying
		}
		m.transition(&o, domain.StatusReady, domain.StatusInProgress, "countdown_over", "system", uuid.Nil, now)
		m.emit(&o, "match.started", everyone, map[string]any{"started_at": now})
		m.nextRound(&o, now)
	case TimerPick:
		if m.inRound(t.Round, PhasePreRound) {
			allowed := m.allowedPicks()
			m.pick(&o, allowed[m.random.Intn(len(allowed))], true, now)
		}
	case TimerRoundDeadline:
		if m.inRound(t.Round, PhaseQuestion) {
			m.lock(&o, now)
		}
	case TimerReveal:
		if m.inRound(t.Round, PhaseReveal) {
			m.endCheck(&o, now)
		}
	case TimerHalftime:
		if m.Status == domain.StatusInProgress && m.Phase == PhaseHalftime {
			m.nextRound(&o, now)
		}
	case TimerReconnect:
		if m.Status != domain.StatusInProgress {
			break
		}
		// Every window that has run out is handled in the same check, so
		// two players gone at once cancel the match (edge-cases §3).
		var expired []*Participant
		for _, p := range m.Participants {
			if p.Status == domain.PDisconnected && now >= p.DisconnectedAt+int64(m.Type.ReconnectWindowMS) {
				p.AwayMS += int(now - p.DisconnectedAt)
				expired = append(expired, p)
			}
		}
		if len(expired) > 0 {
			m.removeParticipants(&o, expired, domain.PForfeited, now)
		}
	}
	return o
}

func (m *Match) inRound(round int, phase Phase) bool {
	return m.Status == domain.StatusInProgress && m.Round != nil && m.Round.No == round && m.Phase == phase
}

// ---------- Rounds ----------

func (m *Match) nextRound(o *Output, now int64) {
	if m.Config.CategoryMode == domain.MixedBattle {
		m.RoundNo++
		m.preRound(o, now)
		return
	}
	q, ok := m.nextQuestion(uuid.Nil)
	if !ok {
		if m.Derived.QuestionCount == 0 {
			// Open-ended (Sudden Death): ask for the next batch.
			m.Phase = PhaseNeedsMore
			o.NeedQuestions = matchtype.SelectionBatch
			return
		}
		m.cancel(o, domain.CancelIntegrityViolated, "system", uuid.Nil, now)
		return
	}
	m.RoundNo++
	m.dispatch(o, q, now)
}

// AddQuestions continues an open-ended match after NeedQuestions.
func (m *Match) AddQuestions(qs []Question, now int64) Output {
	var o Output
	m.questions = append(m.questions, qs...)
	if m.Status == domain.StatusInProgress && m.Phase == PhaseNeedsMore {
		if len(qs) == 0 {
			// The pool ran dry: decide by score like the cap (match-types §3.3).
			m.finish(&o, domain.EndAllRoundsPlayed, now)
			return o
		}
		m.nextRound(&o, now)
	}
	return o
}

// nextQuestion takes the next unused question (in order), or for Mixed
// Battle the unused question of category nearest to the round's curve
// difficulty (lower first).
func (m *Match) nextQuestion(category uuid.UUID) (Question, bool) {
	if category == uuid.Nil {
		for _, q := range m.questions {
			if !m.used[q.ID] {
				m.used[q.ID] = true
				return q, true
			}
		}
		return Question{}, false
	}
	target := m.Type.Difficulty(m.RoundNo)
	best := -1
	dist := func(d int) int {
		if d <= target {
			return 2 * (target - d) // lower preferred on equal distance
		}
		return 2*(d-target) + 1
	}
	for i, q := range m.questions {
		if m.used[q.ID] || q.CategoryID != category {
			continue
		}
		if best < 0 || dist(q.Difficulty) < dist(m.questions[best].Difficulty) {
			best = i
		}
	}
	if best < 0 {
		return Question{}, false
	}
	m.used[m.questions[best].ID] = true
	return m.questions[best], true
}

// activeSeats returns the seats of a side whose members are playing.
func (m *Match) activeSeats(side int) (seats []int, active map[int]bool) {
	active = map[int]bool{}
	for _, p := range m.side(side) {
		seats = append(seats, p.Seat.Seat)
		active[p.Seat.Seat] = p.Status.Active()
	}
	return seats, active
}

func (m *Match) userAt(side, seat int) uuid.UUID {
	for _, p := range m.side(side) {
		if p.Seat.Seat == seat {
			return p.UserID
		}
	}
	return uuid.Nil
}

// ---------- Mixed Battle pre-round (D-043) ----------

func (m *Match) preRound(o *Output, now int64) {
	side := m.firstPicker
	if m.RoundNo%2 == 0 {
		side = 3 - m.firstPicker
	}
	seats, active := m.activeSeats(side)
	seat := teammode.NextTurn(m.Sides[side-1].LastPickSeat, seats, active)
	m.Sides[side-1].LastPickSeat = seat
	m.Round = &Round{No: m.RoundNo, PickerSide: side, Picker: m.userAt(side, seat), PickDeadMS: now + PickMS}
	m.Phase = PhasePreRound
	m.emit(o, "round.pre_round", everyone, map[string]any{
		"round_no": m.RoundNo, "picker_side_no": side, "picker_user_id": m.Round.Picker,
		"categories": m.Config.CategoryIDs, "pick_deadline_at": m.Round.PickDeadMS,
	})
	o.Timers = append(o.Timers, Timer{Kind: TimerPick, Round: m.RoundNo, AtMS: m.Round.PickDeadMS})
}

// allowedPicks are the match categories that would not make a 4th
// consecutive pick of the same category.
func (m *Match) allowedPicks() []uuid.UUID {
	var out []uuid.UUID
	for _, c := range m.Config.CategoryIDs {
		if !m.wouldExceedRun(c) && m.hasQuestion(c) {
			out = append(out, c)
		}
	}
	if len(out) == 0 {
		for _, c := range m.Config.CategoryIDs {
			if m.hasQuestion(c) {
				out = append(out, c)
			}
		}
	}
	return out
}

func (m *Match) hasQuestion(category uuid.UUID) bool {
	for _, q := range m.questions {
		if q.CategoryID == category && !m.used[q.ID] {
			return true
		}
	}
	return false
}

func (m *Match) wouldExceedRun(c uuid.UUID) bool {
	n := len(m.pickHistory)
	if n < MaxSameCategoryInARow {
		return false
	}
	for _, prev := range m.pickHistory[n-MaxSameCategoryInARow:] {
		if prev != c {
			return false
		}
	}
	return true
}

// Pick is the Mixed Battle category choice of the round's picker.
func (m *Match) Pick(user uuid.UUID, round int, category uuid.UUID, now int64) (Output, error) {
	var o Output
	if m.participant(user) == nil {
		return o, errNotParticipant
	}
	if !m.inRound(round, PhasePreRound) {
		return o, errState("no_pick_now")
	}
	if user != m.Round.Picker {
		return o, reject(apierror.PickNotAllowed, "not_picker")
	}
	if !slices.Contains(m.allowedPicks(), category) {
		return o, reject(apierror.PickNotAllowed, "category_not_allowed")
	}
	m.pick(&o, category, false, now)
	return o, nil
}

func (m *Match) pick(o *Output, category uuid.UUID, byServer bool, now int64) {
	m.pickHistory = append(m.pickHistory, category)
	m.emit(o, "round.category_picked", everyone, map[string]any{"round_no": m.RoundNo, "category_id": category, "by_server": byServer})
	q, ok := m.nextQuestion(category)
	if !ok {
		m.cancel(o, domain.CancelIntegrityViolated, "system", uuid.Nil, now)
		return
	}
	m.dispatch(o, q, now)
}

// ---------- Question phase ----------

func (m *Match) dispatch(o *Output, q Question, now int64) {
	order := make([]int, len(q.Options))
	for i := range order {
		order[i] = i + 1
	}
	for i := len(order) - 1; i > 0; i-- { // Fisher–Yates with the engine's random source
		j := m.random.Intn(i + 1)
		order[i], order[j] = order[j], order[i]
	}
	var rtts []int
	for _, p := range m.Participants {
		if p.Status.Active() {
			rtts = append(rtts, p.RTTMS)
		}
	}
	showAt := timing.ShowAt(now, rtts)
	r := &Round{
		No: m.RoundNo, Question: q, OptionOrder: order, DispatchMS: now, ShowAtMS: showAt,
		DeadlineMS: timing.Deadline(showAt, m.Derived.AnswerWindowMS), answers: map[uuid.UUID]answer{},
	}
	if m.Config.TeamMode == domain.IndividualTurns {
		for side := 1; side <= 2; side++ {
			seats, active := m.activeSeats(side)
			seat := teammode.NextTurn(m.Sides[side-1].LastTurnSeat, seats, active)
			if seat != 0 {
				m.Sides[side-1].LastTurnSeat = seat
			}
			r.Turn[side-1] = m.userAt(side, seat)
		}
	}
	m.Round, m.Phase = r, PhaseQuestion
	options := make([]uuid.UUID, len(order))
	for i, pos := range order {
		options[i] = q.Options[pos-1]
	}
	payload := map[string]any{
		"round_no": r.No, "question_id": q.ID, "options": options,
		"show_at": r.ShowAtMS, "deadline_at": r.DeadlineMS, "answering_user_ids": m.expectedAnswerers(),
	}
	if m.Config.CategoryMode != domain.BlindChallenge {
		payload["category_id"], payload["subcategory_id"] = q.CategoryID, q.SubcategoryID
	}
	m.emit(o, "round.started", everyone, payload)
	o.Timers = append(o.Timers, Timer{Kind: TimerRoundDeadline, Round: r.No, AtMS: r.DeadlineMS + int64(m.maxGrace())})
}

// maxGrace is the latest grace among participants (the lock timer waits
// for it; each answer is still judged against its sender's own grace).
func (m *Match) maxGrace() int {
	g := timing.GraceBaseMS
	for _, p := range m.Participants {
		g = max(g, timing.Grace(p.RTTMS))
	}
	return g
}

// expectedAnswerers are who may submit this round (round-lifecycle §2):
// everyone still in the match, or the turn holders. Shared Answer sides
// answer through propose/lock by any member.
func (m *Match) expectedAnswerers() []uuid.UUID {
	var out []uuid.UUID
	if m.Config.TeamMode == domain.IndividualTurns && m.Round != nil {
		for _, u := range m.Round.Turn {
			if u != uuid.Nil {
				out = append(out, u)
			}
		}
		return out
	}
	for _, p := range m.Participants {
		if !p.Status.Gone() {
			out = append(out, p.UserID)
		}
	}
	return out
}

func (m *Match) checkSubmission(user uuid.UUID, round int, option uuid.UUID, now int64) (*Participant, error) {
	p := m.participant(user)
	if p == nil {
		return nil, errNotParticipant
	}
	if !m.inRound(round, PhaseQuestion) {
		return nil, errState("round_not_open")
	}
	if p.Status.Gone() {
		return nil, errState("left_match")
	}
	if m.Config.TeamMode == domain.IndividualTurns && user != m.Round.Turn[p.Side-1] {
		return nil, apierror.New(apierror.NotYourTurn)
	}
	if !slices.Contains(m.Round.Question.Options, option) {
		return nil, apierror.New(apierror.AnswerInvalidOption)
	}
	if !timing.Accepted(now, m.Round.DeadlineMS, p.RTTMS) {
		return nil, apierror.New(apierror.AnswerLate)
	}
	return p, nil
}

// Answer is a submission (1v1, Individual Turns, Fast Team Round). Checks
// in order: round open → expected answerer → option of this question →
// first answer → in time (round-lifecycle §5).
func (m *Match) Answer(user uuid.UUID, round int, option uuid.UUID, now int64) (Output, error) {
	var o Output
	if m.Config.TeamMode == domain.SharedAnswer {
		return o, errState("use_propose_or_lock")
	}
	p, err := m.checkSubmission(user, round, option, now)
	if err != nil {
		if e := apierror.From(err); e != nil && e.Code == apierror.AnswerLate {
			if _, dup := m.Round.answers[user]; dup {
				return o, apierror.New(apierror.AnswerDuplicate)
			}
		}
		return o, err
	}
	if _, dup := m.Round.answers[user]; dup {
		return o, apierror.New(apierror.AnswerDuplicate)
	}
	m.Round.answers[user] = answer{option: option, receivedMS: now, responseMS: timing.ResponseMS(now, m.Round.ShowAtMS, p.RTTMS)}
	// option_id is shown to the sender only; the delivery layer removes it
	// for everyone else (round-lifecycle §4).
	m.emit(&o, "round.answer_received", everyone, map[string]any{"round_no": round, "user_id": user, "option_id": option})
	if m.allAnswered() {
		m.lock(&o, now) // early lock (D-035)
	}
	return o, nil
}

func (m *Match) allAnswered() bool {
	if m.Config.TeamMode == domain.SharedAnswer {
		for side := range 2 {
			if m.Round.locks[side] == nil && m.sideHasMembers(side+1) {
				return false
			}
		}
		return true
	}
	for _, u := range m.expectedAnswerers() {
		if _, ok := m.Round.answers[u]; !ok {
			return false
		}
	}
	return true
}

func (m *Match) sideHasMembers(side int) bool {
	for _, p := range m.side(side) {
		if !p.Status.Gone() {
			return true
		}
	}
	return false
}

// Propose is a Shared Answer proposal; teammates see it live.
func (m *Match) Propose(user uuid.UUID, round int, option uuid.UUID, now int64) (Output, error) {
	var o Output
	if m.Config.TeamMode != domain.SharedAnswer {
		return o, errState("not_shared_answer")
	}
	p, err := m.checkSubmission(user, round, option, now)
	if err != nil {
		return o, err
	}
	if m.Round.locks[p.Side-1] != nil {
		return o, apierror.New(apierror.AnswerDuplicate)
	}
	m.Round.proposals[p.Side-1] = append(m.Round.proposals[p.Side-1], teammode.Proposal{UserID: user, OptionID: option, ReceivedMS: now})
	latest := map[uuid.UUID]uuid.UUID{}
	for _, pr := range m.Round.proposals[p.Side-1] {
		latest[pr.UserID] = pr.OptionID
	}
	var list []map[string]any
	for _, member := range m.side(p.Side) {
		if opt, ok := latest[member.UserID]; ok {
			list = append(list, map[string]any{"user_id": member.UserID, "option_id": opt})
		}
	}
	m.emit(&o, "round.proposals", Audience{Side: p.Side}, map[string]any{"round_no": round, "proposals": list})
	return o, nil
}

// Lock fixes a side's Shared Answer; the first lock wins.
func (m *Match) Lock(user uuid.UUID, round int, option uuid.UUID, now int64) (Output, error) {
	var o Output
	if m.Config.TeamMode != domain.SharedAnswer {
		return o, errState("not_shared_answer")
	}
	p, err := m.checkSubmission(user, round, option, now)
	if err != nil {
		return o, err
	}
	if m.Round.locks[p.Side-1] != nil {
		return o, apierror.New(apierror.AnswerDuplicate)
	}
	m.Round.locks[p.Side-1] = &teammode.Lock{UserID: user, OptionID: option, ResponseMS: timing.ResponseMS(now, m.Round.ShowAtMS, p.RTTMS)}
	m.Round.answers[user] = answer{option: option, receivedMS: now, responseMS: m.Round.locks[p.Side-1].ResponseMS}
	m.emit(&o, "round.answer_received", everyone, map[string]any{"round_no": round, "user_id": user, "side_no": p.Side})
	if m.allAnswered() {
		m.lock(&o, now)
	}
	return o, nil
}

// ---------- Lock, evaluate, reveal ----------

// RoundRecord is one revealed round for game.match_rounds / answers /
// round_results (persisted at reveal, before the next round).
type RoundRecord struct {
	No           int
	Question     Question
	OptionOrder  []int
	AnsweringIDs [2]uuid.UUID
	DispatchMS   int64
	DeadlineMS   int64
	RevealedMS   int64
	Answers      []AnswerRecord
	Sides        [2]SideRecord
}

// AnswerRecord is one row of game.answers (timeouts have no option).
type AnswerRecord struct {
	UserID       uuid.UUID
	Side         int
	OptionID     *uuid.UUID
	ReceivedMS   *int64
	ResponseMS   *int
	Correct      bool
	Points       scoring.Points
	IsSideAnswer bool
}

// SideRecord is one row of game.round_results.
type SideRecord struct {
	Correct    bool
	Points     int
	ResponseMS int
	Detail     map[string]any
}

func (m *Match) lock(o *Output, now int64) {
	r := m.Round
	m.Phase = PhaseReveal
	m.emit(o, "round.locked", everyone, map[string]any{"round_no": r.No})
	q := r.Question
	rec := &RoundRecord{
		No: r.No, Question: q, OptionOrder: r.OptionOrder, AnsweringIDs: r.Turn,
		DispatchMS: r.DispatchMS, DeadlineMS: r.DeadlineMS, RevealedMS: now,
	}
	var sidesOut []map[string]any
	var answersOut []map[string]any
	for side := 1; side <= 2; side++ {
		in := teammode.Input{
			Mode: m.Config.TeamMode, Aggregation: m.Config.Aggregation, Rules: m.Rules,
			Question: teammode.Question{Type: q.Type, Difficulty: q.Difficulty, WindowMS: m.Derived.AnswerWindowMS, Correct: q.Correct},
			SideSize: m.Config.Format.SideSize(), SideStreakBefore: m.Sides[side-1].Streak,
		}
		for _, p := range m.side(side) {
			a, ok := r.answers[p.UserID]
			in.Members = append(in.Members, teammode.Member{
				UserID: p.UserID, Seat: p.Seat.Seat, Active: p.Status.Active(), Answered: ok,
				OptionID: a.option, ResponseMS: a.responseMS, ReceivedMS: a.receivedMS, StreakBefore: p.Streak,
			})
		}
		switch m.Config.TeamMode {
		case domain.SharedAnswer:
			in.Side = teammode.ResolveShared(r.locks[side-1], r.proposals[side-1])
		case domain.IndividualTurns:
			holder := r.Turn[side-1]
			a, ok := r.answers[holder]
			in.Side = teammode.SideAnswer{Answered: ok, OptionID: a.option, ResponseMS: a.responseMS, Submitter: holder}
		case domain.NoTeamMode, domain.FastTeamRound:
			// Every member's own answer counts.
		}
		res := teammode.Aggregate(in)
		st := &m.Sides[side-1]
		st.Score += res.Points
		st.Streak = res.SideStreakAfter
		if res.Correct {
			st.Correct++
			st.CorrectResponseMS += max(res.ResponseMS, 0)
		}
		rec.Sides[side-1] = SideRecord{Correct: res.Correct, Points: res.Points, ResponseMS: res.ResponseMS, Detail: res.Detail}
		sidesOut = append(sidesOut, map[string]any{"side_no": side, "correct": res.Correct, "points": res.Points, "score": st.Score})

		for _, p := range m.side(side) {
			a, answered := r.answers[p.UserID]
			pts := res.MemberPoints[p.UserID]
			p.Contribution += res.Contribution[p.UserID]
			counted := m.answerCounts(side, p.UserID, answered)
			if !counted {
				continue
			}
			ok := answered && a.option == q.Correct
			if m.Config.TeamMode == domain.FastTeamRound || m.Config.TeamMode == domain.NoTeamMode {
				p.Streak = pts.StreakAfter
			}
			ar := AnswerRecord{UserID: p.UserID, Side: side, Correct: ok, Points: pts}
			if answered {
				p.Answered++
				opt, recv, resp := a.option, a.receivedMS, a.responseMS
				ar.OptionID, ar.ReceivedMS, ar.ResponseMS = &opt, &recv, &resp
			}
			if ok {
				p.Correct++
			}
			if m.Config.TeamMode == domain.SharedAnswer {
				ar.IsSideAnswer = true
				// The side answer may be a fallback chosen by proposals.
				opt := in.Side.OptionID
				ar.OptionID, ar.Correct = &opt, res.Correct
				ar.Points = res.Breakdown
			}
			rec.Answers = append(rec.Answers, ar)
			answersOut = append(answersOut, map[string]any{
				"user_id": p.UserID, "option_id": ar.OptionID, "response_ms": ar.ResponseMS, "correct": ar.Correct,
				"points": map[string]int{"base": ar.Points.Base, "speed": ar.Points.Speed, "streak": ar.Points.Streak, "total": ar.Points.Total},
			})
		}
	}
	o.Round, m.lastRound = rec, rec
	m.emit(o, "round.revealed", everyone, map[string]any{
		"round_no": r.No, "question_id": q.ID, "correct_option_id": q.Correct, "category_id": q.CategoryID,
		"answers": answersOut, "sides": sidesOut,
	})
	o.Timers = append(o.Timers, Timer{Kind: TimerReveal, Round: r.No, AtMS: now + RevealMS})
}

// answerCounts reports whether a participant gets an answers row this
// round: everyone in 1v1 and Fast Team Round (timeouts included), the turn
// holder in Individual Turns, the side answer's submitter in Shared Answer.
func (m *Match) answerCounts(side int, user uuid.UUID, answered bool) bool {
	p := m.participant(user)
	switch m.Config.TeamMode {
	case domain.IndividualTurns:
		return m.Round.Turn[side-1] == user
	case domain.SharedAnswer:
		ans := teammode.ResolveShared(m.Round.locks[side-1], m.Round.proposals[side-1])
		return ans.Answered && ans.Submitter == user
	default:
		return answered || !p.Status.Gone()
	}
}

func (m *Match) endCheck(o *Output, now int64) {
	r := m.Round
	last := matchtype.RoundOutcome{Round: r.No}
	if c := m.lastSideCorrect(); c != nil {
		last.SideCorrect = *c
	}
	end := m.Type.EndCheck(last, m.Derived.QuestionCount)
	if end.Finished {
		m.finish(o, end.Reason, now)
		return
	}
	if m.Type.HalfTimeAfter > 0 && r.No == m.Type.HalfTimeAfter {
		m.Phase = PhaseHalftime
		ends := now + int64(m.Type.HalfTimeMS)
		m.emit(o, "match.halftime", everyone, map[string]any{"ends_at": ends, "scores": [2]int{m.Sides[0].Score, m.Sides[1].Score}})
		o.Timers = append(o.Timers, Timer{Kind: TimerHalftime, AtMS: ends})
		return
	}
	m.nextRound(o, now)
}

// lastSideCorrect is the side correctness of the last revealed round.
func (m *Match) lastSideCorrect() *[2]bool {
	if m.lastRound == nil {
		return nil
	}
	return &[2]bool{m.lastRound.Sides[0].Correct, m.lastRound.Sides[1].Correct}
}
