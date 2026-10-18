package engine

import (
	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain/outcome"
)

// Result is a terminal match state for the final Postgres transaction
// (schema-game §1) and game.MatchCompleted / MatchCancelled.
type Result struct {
	Status       domain.Status
	EndReason    domain.EndReason
	CancelReason domain.CancelReason
	WinnerSide   int // 0 = none (draw or cancelled)
	Draw         bool
	TieBreak     outcome.TieBreak
	EndedMS      int64
	RoundsPlayed int
	Sides        [2]SideResult
	Participants []ParticipantResult
}

// SideResult is one side's final row.
type SideResult struct {
	Score       int
	Correct     int
	Result      domain.SideResult // empty when cancelled
	ForfeitedMS int64
}

// ParticipantResult is one participant's final row.
type ParticipantResult struct {
	UserID       uuid.UUID
	Side         int
	Status       domain.ParticipantStatus
	Contribution int
	Correct      int
	Answered     int
	Disconnects  int
	AwayMS       int
}

// Result returns the terminal result, or nil while the match is live.
func (m *Match) Result() *Result { return m.result }

// Questions returns the selected questions not yet asked, in order.
func (m *Match) Questions() []Question {
	var out []Question
	for _, q := range m.questions {
		if !m.used[q.ID] {
			out = append(out, q)
		}
	}
	return out
}

func (m *Match) snapshotResult(now int64) *Result {
	// An open, unrevealed round is aborted and never scored (round-lifecycle §1).
	r := &Result{Status: m.Status, EndedMS: now, RoundsPlayed: m.revealedRounds()}
	for i := range 2 {
		r.Sides[i] = SideResult{Score: m.Sides[i].Score, Correct: m.Sides[i].Correct, ForfeitedMS: m.Sides[i].ForfeitedAt}
	}
	for _, p := range m.Participants {
		r.Participants = append(r.Participants, ParticipantResult{
			UserID: p.UserID, Side: p.Side, Status: p.Status, Contribution: p.Contribution,
			Correct: p.Correct, Answered: p.Answered, Disconnects: p.Disconnects, AwayMS: p.AwayMS,
		})
	}
	return r
}

func (m *Match) revealedRounds() int {
	if m.lastRound == nil {
		return 0
	}
	return m.lastRound.No
}

// finish completes the match (T6) with its winner.
func (m *Match) finish(o *Output, reason domain.EndReason, now int64) {
	if m.Status.Terminal() {
		return
	}
	from := m.Status
	m.Status, m.Phase = domain.StatusCompleted, PhaseOver
	r := m.snapshotResult(now)
	r.EndReason = reason
	switch reason {
	case domain.EndSuddenDeathDecided:
		if c := m.lastSideCorrect(); c != nil && c[0] {
			r.WinnerSide = 1
		} else {
			r.WinnerSide = 2
		}
	case domain.EndForfeit, domain.EndAllOpponentsLeft:
		for i := range 2 {
			if m.Sides[i].ForfeitedAt != 0 {
				r.WinnerSide = 2 - i
			}
		}
	default:
		r.WinnerSide, r.TieBreak = outcome.Decide(m.totals(0), m.totals(1))
	}
	r.Draw = r.WinnerSide == 0
	for i := range 2 {
		switch {
		case r.Draw:
			r.Sides[i].Result = domain.Draw
		case r.WinnerSide == i+1:
			r.Sides[i].Result = domain.Win
		default:
			r.Sides[i].Result = domain.Loss
		}
	}
	for i, p := range m.Participants {
		if p.Status == domain.PPlaying || p.Status == domain.PDisconnected {
			p.Status = domain.PFinished
			r.Participants[i].Status = domain.PFinished
		}
	}
	m.result = r
	o.Result = r
	m.transition(o, from, domain.StatusCompleted, string(reason), "system", uuid.Nil, now)
	m.emit(o, "match.completed", everyone, map[string]any{
		"end_reason": reason, "winner_side_no": r.WinnerSide, "is_draw": r.Draw, "tie_break": r.TieBreak,
		"sides": [2]map[string]any{
			{"side_no": 1, "score": r.Sides[0].Score, "result": r.Sides[0].Result},
			{"side_no": 2, "score": r.Sides[1].Score, "result": r.Sides[1].Result},
		},
		"awards_pending": true,
	})
}

func (m *Match) totals(i int) outcome.SideTotals {
	return outcome.SideTotals{Score: m.Sides[i].Score, Correct: m.Sides[i].Correct, CorrectResponseMS: m.Sides[i].CorrectResponseMS}
}

// cancel ends the match without a result (T3, T5, T7).
func (m *Match) cancel(o *Output, reason domain.CancelReason, actor string, actorID uuid.UUID, now int64) {
	if m.Status.Terminal() {
		return
	}
	from := m.Status
	m.Status, m.Phase = domain.StatusCancelled, PhaseOver
	r := m.snapshotResult(now)
	r.CancelReason = reason
	m.result = r
	o.Result = r
	m.transition(o, from, domain.StatusCancelled, string(reason), actor, actorID, now)
	m.emit(o, "match.cancelled", everyone, map[string]any{"reason": reason})
}

// Cancel is an admin or owner-side cancellation (server_error, admin,
// integrity_violation).
func (m *Match) Cancel(reason domain.CancelReason, admin uuid.UUID, now int64) Output {
	var o Output
	actor := "system"
	if reason == domain.CancelAdmin {
		actor = "admin"
	}
	m.cancel(&o, reason, actor, admin, now)
	return o
}

// ---------- Leaving, disconnects (edge-cases §2–§3) ----------

// Leave is a participant leaving on purpose: before the start it cancels
// the match (T3/T5); during play it is final for that participant.
func (m *Match) Leave(user uuid.UUID, now int64) (Output, error) {
	var o Output
	p := m.participant(user)
	if p == nil {
		return o, errNotParticipant
	}
	switch m.Status {
	case domain.StatusWaiting, domain.StatusReady:
		p.Status = domain.PLeft
		m.emit(&o, "match.participant_status", everyone, map[string]any{"user_id": user, "status": domain.PLeft})
		m.cancel(&o, domain.CancelLeftBeforeStart, "participant", user, now)
	case domain.StatusInProgress:
		if p.Status.Gone() {
			return o, nil
		}
		if p.Status == domain.PDisconnected {
			p.AwayMS += int(now - p.DisconnectedAt)
		}
		m.forfeitParticipant(&o, p, domain.PLeft, now)
	default:
		return o, errState("match_over")
	}
	return o, nil
}

// Disconnect marks a lost socket. During play the match continues; the
// participant has the type's reconnect window (D-048).
func (m *Match) Disconnect(user uuid.UUID, now int64) Output {
	var o Output
	p := m.participant(user)
	if p == nil {
		return o
	}
	switch m.Status {
	case domain.StatusInProgress:
		if p.Status != domain.PPlaying {
			return o
		}
		p.Status, p.DisconnectedAt = domain.PDisconnected, now
		p.Disconnects++
		if outcome.ExceedsDisconnectLimit(p.Disconnects, p.AwayMS, m.Type.ReconnectWindowMS) {
			m.forfeitParticipant(&o, p, domain.PForfeited, now)
			return o
		}
		deadline := now + int64(m.Type.ReconnectWindowMS)
		m.emit(&o, "match.participant_status", everyone, map[string]any{
			"user_id": user, "status": domain.PDisconnected, "reconnect_deadline_at": deadline,
		})
		o.Timers = append(o.Timers, Timer{Kind: TimerReconnect, User: user, AtMS: deadline})
	case domain.StatusReady:
		// A disconnect in the countdown is treated as "not started": the
		// start timer cancels unless the participant is back (T5).
		p.Status = domain.PJoined
	case domain.StatusWaiting, domain.StatusCompleted, domain.StatusCancelled:
		// Waiting: the ready deadline decides; over: nothing to do.
	}
	return o
}

// Reconnect brings a participant back within the window, keeping every
// original deadline.
func (m *Match) Reconnect(user uuid.UUID, now int64) Output {
	var o Output
	p := m.participant(user)
	if p == nil {
		return o
	}
	switch {
	case m.Status == domain.StatusInProgress && p.Status == domain.PDisconnected:
		p.AwayMS += int(now - p.DisconnectedAt)
		if now > p.DisconnectedAt+int64(m.Type.ReconnectWindowMS) ||
			outcome.ExceedsDisconnectLimit(p.Disconnects, p.AwayMS, m.Type.ReconnectWindowMS) {
			m.forfeitParticipant(&o, p, domain.PForfeited, now)
			return o
		}
		p.Status = domain.PPlaying
		m.emit(&o, "match.participant_status", everyone, map[string]any{"user_id": user, "status": domain.PPlaying})
	case m.Status == domain.StatusReady && p.Status == domain.PJoined:
		p.Status = domain.PReady
	}
	return o
}

// RTT records a ping round trip of the participant's connection.
func (m *Match) RTT(user uuid.UUID, rttMS int) {
	if p := m.participant(user); p != nil {
		p.RTTMS = rttMS
	}
}

// forfeitParticipant removes one participant for good.
func (m *Match) forfeitParticipant(o *Output, p *Participant, status domain.ParticipantStatus, now int64) {
	m.removeParticipants(o, []*Participant{p}, status, now)
}

// removeParticipants removes participants for good, then checks whether a
// side (or both) has nobody left (edge-cases §3).
func (m *Match) removeParticipants(o *Output, ps []*Participant, status domain.ParticipantStatus, now int64) {
	for _, p := range ps {
		p.Status = status
		m.emit(o, "match.participant_status", everyone, map[string]any{"user_id": p.UserID, "status": status})
	}
	gone := [2]bool{}
	voluntary := [2]bool{true, true}
	for side := 1; side <= 2; side++ {
		gone[side-1] = true
		for _, q := range m.side(side) {
			if !q.Status.Gone() {
				gone[side-1] = false
			}
			if q.Status == domain.PForfeited {
				voluntary[side-1] = false
			}
		}
	}
	switch {
	case gone[0] && gone[1]:
		m.cancel(o, domain.CancelAllSidesAbsent, "system", uuid.Nil, now)
	case gone[0] || gone[1]:
		i := 0
		if gone[1] {
			i = 1
		}
		m.Sides[i].ForfeitedAt = now
		reason := domain.EndForfeit
		if voluntary[i] {
			reason = domain.EndAllOpponentsLeft
		}
		m.finish(o, reason, now)
	default:
		// Short-handed play continues; turns skip inactive seats.
		if m.Phase == PhaseQuestion && m.allAnswered() {
			m.lock(o, now)
		}
	}
}
