package engine

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain/config"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain/matchtype"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain/scoring"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain/teammode"
)

// state is the complete engine state as JSON: the live owner saves it in
// Redis after every command, so another instance can take the match over
// and any instance can build a reconnect snapshot (match-lifecycle §5,
// edge-cases §4).
type state struct {
	ID            uuid.UUID
	Config        config.Config
	Rules         scoring.Ruleset
	Status        domain.Status
	Phase         Phase
	CreatedMS     int64
	ReadyDeadMS   int64
	StartsAtMS    int64
	Participants  []*Participant
	Sides         [2]SideState
	RoundNo       int
	Round         *roundState
	Questions     []Question
	Used          []uuid.UUID
	PickHistory   []uuid.UUID
	FirstPicker   int
	Seq           int
	TransitionSeq int
	LastRound     *RoundRecord
	Result        *Result
}

type roundState struct {
	Round
	Answers   map[uuid.UUID]answerState
	Proposals [2][]teammode.Proposal
	Locks     [2]*teammode.Lock
}

type answerState struct {
	Option     uuid.UUID
	ReceivedMS int64
	ResponseMS int
}

// Snapshot returns the engine state as JSON.
func (m *Match) Snapshot() ([]byte, error) {
	s := state{
		ID: m.ID, Config: m.Config, Rules: m.Rules, Status: m.Status, Phase: m.Phase, CreatedMS: m.CreatedMS,
		ReadyDeadMS: m.ReadyDeadMS, StartsAtMS: m.StartsAtMS, Participants: m.Participants, Sides: m.Sides,
		RoundNo: m.RoundNo, Questions: m.questions, PickHistory: m.pickHistory, FirstPicker: m.firstPicker,
		Seq: m.seq, TransitionSeq: m.transitionSeq, LastRound: m.lastRound, Result: m.result,
	}
	for id := range m.used {
		s.Used = append(s.Used, id)
	}
	if r := m.Round; r != nil {
		rs := &roundState{Round: *r, Answers: map[uuid.UUID]answerState{}, Proposals: r.proposals, Locks: r.locks}
		for u, a := range r.answers {
			rs.Answers[u] = answerState{Option: a.option, ReceivedMS: a.receivedMS, ResponseMS: a.responseMS}
		}
		s.Round = rs
	}
	return json.Marshal(s)
}

// Restore rebuilds a match from Snapshot output.
func Restore(data []byte, random Random) (*Match, error) {
	var s state
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("engine: restoring state: %w", err)
	}
	t := matchtype.For(s.Config.MatchType)
	if s.Config.RankedLength {
		t = matchtype.RankedLengthQuickDuel()
	}
	m := &Match{
		ID: s.ID, Config: s.Config, Derived: config.Derive(s.Config), Type: t, Rules: s.Rules,
		Status: s.Status, Phase: s.Phase, CreatedMS: s.CreatedMS, ReadyDeadMS: s.ReadyDeadMS, StartsAtMS: s.StartsAtMS,
		Participants: s.Participants, Sides: s.Sides, RoundNo: s.RoundNo, questions: s.Questions,
		used: map[uuid.UUID]bool{}, pickHistory: s.PickHistory, firstPicker: s.FirstPicker, seq: s.Seq,
		transitionSeq: s.TransitionSeq, lastRound: s.LastRound, random: random, result: s.Result,
	}
	for _, id := range s.Used {
		m.used[id] = true
	}
	if rs := s.Round; rs != nil {
		r := rs.Round
		r.answers = map[uuid.UUID]answer{}
		for u, a := range rs.Answers {
			r.answers[u] = answer{option: a.Option, receivedMS: a.ReceivedMS, responseMS: a.ResponseMS}
		}
		r.proposals, r.locks = rs.Proposals, rs.Locks
		m.Round = &r
	}
	return m, nil
}

// Seq is the sequence number of the last event (the match stream's head).
func (m *Match) Seq() int { return m.seq }

// AnswerOf returns the participant's accepted answer in the open round.
func (m *Match) AnswerOf(user uuid.UUID) (uuid.UUID, bool) {
	if m.Round == nil {
		return uuid.Nil, false
	}
	a, ok := m.Round.answers[user]
	return a.option, ok
}

// SideRound is what one side sees of the open round (Shared Answer): the
// latest proposal per member, in seat order, and the locked option.
func (m *Match) SideRound(side int) (proposals []teammode.Proposal, locked *uuid.UUID) {
	if m.Round == nil || side < 1 || side > 2 {
		return nil, nil
	}
	latest := map[uuid.UUID]teammode.Proposal{}
	for _, p := range m.Round.proposals[side-1] {
		latest[p.UserID] = p
	}
	for _, member := range m.side(side) {
		if p, ok := latest[member.UserID]; ok {
			proposals = append(proposals, p)
		}
	}
	if l := m.Round.locks[side-1]; l != nil {
		locked = &l.OptionID
	}
	return proposals, locked
}

// ExpectedAnswerers are who may submit in the open round.
func (m *Match) ExpectedAnswerers() []uuid.UUID { return m.expectedAnswerers() }

// Answered lists who submitted in the open round.
func (m *Match) Answered() []uuid.UUID {
	if m.Round == nil {
		return nil
	}
	var out []uuid.UUID
	for _, p := range m.Participants {
		if _, ok := m.Round.answers[p.UserID]; ok {
			out = append(out, p.UserID)
		}
	}
	return out
}

// Participant returns a participant's state, or nil.
func (m *Match) Participant(user uuid.UUID) *Participant { return m.participant(user) }

// Takeover resumes the match on a new owner after the previous one was
// lost (edge-cases §4). An open question round is voided — not scored, its
// question never reused — and a fresh round with new show_at and deadline
// takes its number; spare is the replacement question for fixed-length
// matches. Every pending timer is returned again from the stored
// deadlines, and participants get match.resumed.
func (m *Match) Takeover(spare []Question, now int64) Output {
	var o Output
	m.questions = append(m.questions, spare...)
	switch m.Status {
	case domain.StatusWaiting:
		o.Timers = append(o.Timers, Timer{Kind: TimerReadyDeadline, AtMS: m.ReadyDeadMS})
		return o
	case domain.StatusReady:
		o.Timers = append(o.Timers, Timer{Kind: TimerStart, AtMS: m.StartsAtMS})
		return o
	case domain.StatusInProgress:
	default:
		return o
	}
	payload := map[string]any{"reason": "takeover"}
	voided := 0
	switch m.Phase {
	case PhaseQuestion:
		voided = m.Round.No
		m.RoundNo, m.Round = voided-1, nil
		payload["voided_round_no"] = voided
		m.transition(&o, domain.StatusInProgress, domain.StatusInProgress, "round_voided_takeover", "system", uuid.Nil, now)
	case PhaseReveal:
		at := now
		if m.lastRound != nil {
			at = max(m.lastRound.RevealedMS+RevealMS, now)
		}
		o.Timers = append(o.Timers, Timer{Kind: TimerReveal, Round: m.Round.No, AtMS: at})
	case PhasePreRound:
		o.Timers = append(o.Timers, Timer{Kind: TimerPick, Round: m.Round.No, AtMS: max(m.Round.PickDeadMS, now)})
	case PhaseHalftime:
		o.Timers = append(o.Timers, Timer{Kind: TimerHalftime, AtMS: now + int64(m.Type.HalfTimeMS)})
	case PhaseLobby, PhaseNeedsMore, PhaseOver:
	}
	m.emit(&o, "match.resumed", everyone, payload)
	for _, p := range m.Participants {
		if p.Status == domain.PDisconnected {
			o.Timers = append(o.Timers, Timer{Kind: TimerReconnect, User: p.UserID, AtMS: p.DisconnectedAt + int64(m.Type.ReconnectWindowMS)})
		}
	}
	if voided > 0 {
		m.nextRound(&o, now)
	}
	return o
}
