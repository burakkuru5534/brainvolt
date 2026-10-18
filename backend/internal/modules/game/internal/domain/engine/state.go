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

// Participant returns a participant's state, or nil.
func (m *Match) Participant(user uuid.UUID) *Participant { return m.participant(user) }
