// Package teammode turns the answers of one side in one round into the
// side's result (docs/game/team-modes.md, D-039): Shared Answer,
// Individual Turns and Fast Team Round with its three aggregation rules,
// plus the 1v1 case. Pure functions; every output stays on the scale of
// one player's answer (≤ 260 per round with ruleset 1).
package teammode

import (
	"slices"
	"sort"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain/scoring"
)

// Question is what scoring needs about the round's question.
type Question struct {
	Type       domain.QuestionType
	Difficulty int
	WindowMS   int
	Correct    uuid.UUID // the correct option id
}

// Member is one participant's state and answer in the round.
type Member struct {
	UserID uuid.UUID
	Seat   int
	// Active: connected and still in the match when the round started.
	Active bool
	// Answered is false for a timeout.
	Answered   bool
	OptionID   uuid.UUID
	ResponseMS int
	ReceivedMS int64
	// StreakBefore is the member's personal streak (Fast Team Round, 1v1).
	StreakBefore int
}

// SideAnswer is the one answer that counts for a side (Shared Answer after
// resolution, Individual Turns' turn holder).
type SideAnswer struct {
	// Answered is false for a side timeout.
	Answered   bool
	OptionID   uuid.UUID
	ResponseMS int
	// Submitter is the locker, the first proposer of the fallback option,
	// or the turn holder.
	Submitter uuid.UUID
	// Fallback: Shared Answer auto-lock at the deadline — no speed bonus.
	Fallback bool
}

// Input is one side's round.
type Input struct {
	Mode        domain.TeamMode
	Aggregation domain.Aggregation
	Question    Question
	Rules       scoring.Ruleset
	// SideSize is the format's full side size (Fast Team Round divisor).
	SideSize         int
	Members          []Member
	SideStreakBefore int
	// Side is set for Shared Answer and Individual Turns.
	Side SideAnswer
}

// Result is the side's outcome of the round.
type Result struct {
	Correct bool
	Points  int
	// Breakdown of the side answer (Shared Answer, Individual Turns,
	// majority, 1v1); zero for average / fastest_correct sides.
	Breakdown scoring.Points
	// ResponseMS is the response time of a correct round used by the
	// tie-breakers (edge-cases §1); -1 when the side was not correct.
	ResponseMS      int
	SideStreakAfter int
	// MemberPoints are each answer's own points (answers.points_awarded).
	MemberPoints map[uuid.UUID]scoring.Points
	// Contribution sums to Points.
	Contribution map[uuid.UUID]int
	// Detail goes to round_results.detail.
	Detail map[string]any
}

func (in Input) score(correct bool, responseMS, streakBefore int) scoring.Points {
	return in.Rules.Score(scoring.Input{
		Correct: correct, QuestionType: in.Question.Type, Difficulty: in.Question.Difficulty,
		ResponseMS: responseMS, WindowMS: in.Question.WindowMS, StreakBefore: streakBefore,
	})
}

func (in Input) correct(answered bool, option uuid.UUID) bool {
	return answered && option == in.Question.Correct
}

// Aggregate returns the side's result for the round.
func Aggregate(in Input) Result {
	switch in.Mode {
	case domain.SharedAnswer:
		return sideAnswer(in, sharedContribution)
	case domain.IndividualTurns:
		return sideAnswer(in, func(in Input, points int) map[uuid.UUID]int {
			if in.Side.Submitter == uuid.Nil {
				return map[uuid.UUID]int{}
			}
			return map[uuid.UUID]int{in.Side.Submitter: points}
		})
	case domain.FastTeamRound:
		return fastTeam(in)
	default:
		return solo(in)
	}
}

func solo(in Input) Result {
	r := Result{MemberPoints: map[uuid.UUID]scoring.Points{}, Contribution: map[uuid.UUID]int{}, Detail: map[string]any{"rule": "solo"}}
	if len(in.Members) == 0 {
		r.ResponseMS = -1
		return r
	}
	m := in.Members[0]
	ok := in.correct(m.Answered, m.OptionID)
	p := in.score(ok, m.ResponseMS, m.StreakBefore)
	r.Correct, r.Points, r.Breakdown = ok, p.Total, p
	r.MemberPoints[m.UserID] = p
	r.Contribution[m.UserID] = p.Total
	r.SideStreakAfter = p.StreakAfter
	r.ResponseMS = -1
	if ok {
		r.ResponseMS = m.ResponseMS
	}
	return r
}

// sideAnswer scores the side's single answer with the side streak.
func sideAnswer(in Input, contribution func(Input, int) map[uuid.UUID]int) Result {
	s := in.Side
	ok := in.correct(s.Answered, s.OptionID)
	resp := s.ResponseMS
	if s.Fallback {
		resp = in.Question.WindowMS // no speed bonus for an auto-locked answer (D-039)
	}
	p := in.score(ok, resp, in.SideStreakBefore)
	r := Result{
		Correct: ok, Points: p.Total, Breakdown: p, SideStreakAfter: p.StreakAfter, ResponseMS: -1,
		MemberPoints: map[uuid.UUID]scoring.Points{},
		Detail:       map[string]any{"rule": string(in.Mode), "fallback": s.Fallback, "submitter": s.Submitter},
	}
	if ok {
		r.ResponseMS = resp
	}
	if s.Submitter != uuid.Nil {
		r.MemberPoints[s.Submitter] = p
	}
	r.Contribution = contribution(in, p.Total)
	return r
}

// sharedContribution splits points equally among members connected during
// the round; the remainder goes to the locker (D-039).
func sharedContribution(in Input, points int) map[uuid.UUID]int {
	out := map[uuid.UUID]int{}
	var connected []uuid.UUID
	for _, m := range in.Members {
		if m.Active {
			connected = append(connected, m.UserID)
		}
	}
	if len(connected) == 0 {
		if in.Side.Submitter != uuid.Nil {
			out[in.Side.Submitter] = points
		}
		return out
	}
	share := points / len(connected)
	for _, u := range connected {
		out[u] = share
	}
	rest := points - share*len(connected)
	if rest > 0 {
		to := in.Side.Submitter
		if _, ok := out[to]; !ok {
			to = connected[0]
		}
		out[to] += rest
	}
	return out
}

func fastTeam(in Input) Result {
	r := Result{MemberPoints: map[uuid.UUID]scoring.Points{}, Contribution: map[uuid.UUID]int{}, ResponseMS: -1}
	type scored struct {
		m  Member
		ok bool
		p  scoring.Points
	}
	all := make([]scored, 0, len(in.Members))
	active, correctActive := 0, 0
	for _, m := range in.Members {
		ok := in.correct(m.Answered, m.OptionID)
		p := in.score(ok, m.ResponseMS, m.StreakBefore)
		r.MemberPoints[m.UserID] = p
		all = append(all, scored{m, ok, p})
		if m.Active {
			active++
			if ok {
				correctActive++
			}
		}
	}
	size := max(in.SideSize, 1)

	switch in.Aggregation {
	case domain.FastestCorrect:
		var best *scored
		for i := range all {
			s := &all[i]
			if !s.ok {
				continue
			}
			if best == nil || s.m.ResponseMS < best.m.ResponseMS ||
				(s.m.ResponseMS == best.m.ResponseMS && s.m.ReceivedMS < best.m.ReceivedMS) {
				best = s
			}
		}
		r.Detail = map[string]any{"rule": string(domain.FastestCorrect)}
		if best != nil {
			r.Correct, r.Points, r.ResponseMS = true, best.p.Total, best.m.ResponseMS
			r.Contribution[best.m.UserID] = best.p.Total
			r.Detail["member"] = best.m.UserID
		}
	case domain.Majority:
		votes := map[uuid.UUID][]Member{}
		for _, s := range all {
			if s.m.Answered {
				votes[s.m.OptionID] = append(votes[s.m.OptionID], s.m)
			}
		}
		var choice uuid.UUID
		var choosers []Member
		firstAt := func(ms []Member) int64 {
			t := ms[0].ReceivedMS
			for _, m := range ms {
				t = min(t, m.ReceivedMS)
			}
			return t
		}
		for opt, ms := range votes {
			if len(ms) > len(choosers) || (len(ms) == len(choosers) && firstAt(ms) < firstAt(choosers)) {
				choice, choosers = opt, ms
			}
		}
		r.Detail = map[string]any{"rule": string(domain.Majority)}
		if len(choosers) == 0 {
			r.SideStreakAfter = 0
			break
		}
		times := make([]int, len(choosers))
		for i, m := range choosers {
			times[i] = m.ResponseMS
		}
		slices.Sort(times)
		median := times[len(times)/2]
		if len(times)%2 == 0 {
			median = (times[len(times)/2-1] + times[len(times)/2] + 1) / 2
		}
		ok := choice == in.Question.Correct
		p := in.score(ok, median, in.SideStreakBefore)
		r.Correct, r.Points, r.Breakdown, r.SideStreakAfter = ok, p.Total, p, p.StreakAfter
		if ok {
			r.ResponseMS = median
		}
		ids := make([]uuid.UUID, len(choosers))
		fastest := choosers[0]
		for i, m := range choosers {
			ids[i] = m.UserID
			if m.ReceivedMS < fastest.ReceivedMS {
				fastest = m
			}
		}
		share := p.Total / len(ids)
		for _, u := range ids {
			r.Contribution[u] = share
		}
		r.Contribution[fastest.UserID] += p.Total - share*len(ids)
		r.Detail["option"], r.Detail["choosers"], r.Detail["median_response_ms"] = choice, ids, median
	default: // average (D-039 default)
		sum, correctTimes, nCorrect := 0, 0, 0
		for _, s := range all {
			sum += s.p.Total
			if s.ok {
				correctTimes += s.m.ResponseMS
				nCorrect++
			}
		}
		r.Correct = active > 0 && 2*correctActive >= active
		r.Points = (2*sum + size) / (2 * size) // round(sum / side size), half up
		if r.Correct && nCorrect > 0 {
			r.ResponseMS = (correctTimes + nCorrect/2) / nCorrect
		}
		r.Contribution = proportional(all, func(s scored) (uuid.UUID, int) { return s.m.UserID, s.p.Total }, size, r.Points)
		r.Detail = map[string]any{"rule": string(domain.Average), "active": active, "correct_active": correctActive, "side_size": size}
	}
	if in.Aggregation != domain.Majority {
		if r.Correct {
			r.SideStreakAfter = in.SideStreakBefore + 1
		}
	}
	return r
}

// proportional gives each member value/size, rounded so the parts sum to
// total (largest remainders first, then seat order via input order).
func proportional[T any](items []T, f func(T) (uuid.UUID, int), size, total int) map[uuid.UUID]int {
	type part struct {
		id   uuid.UUID
		base int
		frac int
		i    int
	}
	parts := make([]part, 0, len(items))
	sum := 0
	for i, it := range items {
		id, v := f(it)
		parts = append(parts, part{id: id, base: v / size, frac: v % size, i: i})
		sum += v / size
	}
	sort.SliceStable(parts, func(a, b int) bool { return parts[a].frac > parts[b].frac })
	out := map[uuid.UUID]int{}
	for _, p := range parts {
		out[p.id] = p.base
	}
	for i := 0; sum < total && len(parts) > 0; i = (i + 1) % len(parts) {
		out[parts[i].id]++
		sum++
	}
	return out
}

// ---------- Shared Answer resolution (round-lifecycle §3) ----------

// Proposal is a member's current proposal (the latest one counts).
type Proposal struct {
	UserID     uuid.UUID
	OptionID   uuid.UUID
	ReceivedMS int64
}

// Lock is an explicit side lock.
type Lock struct {
	UserID     uuid.UUID
	OptionID   uuid.UUID
	ResponseMS int
}

// ResolveShared returns the side's answer: the lock if any; otherwise the
// most-proposed option (tie → the option proposed earliest), answered at
// the deadline without speed bonus; no proposals → side timeout.
func ResolveShared(lock *Lock, proposals []Proposal) SideAnswer {
	if lock != nil {
		return SideAnswer{Answered: true, OptionID: lock.OptionID, ResponseMS: lock.ResponseMS, Submitter: lock.UserID}
	}
	latest := map[uuid.UUID]Proposal{}
	for _, p := range proposals {
		if cur, ok := latest[p.UserID]; !ok || p.ReceivedMS >= cur.ReceivedMS {
			latest[p.UserID] = p
		}
	}
	count := map[uuid.UUID]int{}
	first := map[uuid.UUID]Proposal{}
	for _, p := range latest {
		count[p.OptionID]++
		if f, ok := first[p.OptionID]; !ok || p.ReceivedMS < f.ReceivedMS {
			first[p.OptionID] = p
		}
	}
	var best uuid.UUID
	for opt := range count {
		if best == uuid.Nil || count[opt] > count[best] ||
			(count[opt] == count[best] && first[opt].ReceivedMS < first[best].ReceivedMS) {
			best = opt
		}
	}
	if best == uuid.Nil {
		return SideAnswer{}
	}
	return SideAnswer{Answered: true, OptionID: best, Submitter: first[best].UserID, Fallback: true}
}

// ---------- Individual Turns rotation (D-035, D-039) ----------

// NextTurn returns the seat whose turn it is after lastSeat (0 before the
// first round): the next seat in order, wrapping, skipping members who are
// not active; 0 when nobody is active (side timeout).
func NextTurn(lastSeat int, seats []int, active map[int]bool) int {
	if len(seats) == 0 {
		return 0
	}
	ordered := slices.Clone(seats)
	slices.Sort(ordered)
	start := 0
	for i, s := range ordered {
		if s > lastSeat {
			start = i
			break
		}
		if i == len(ordered)-1 {
			start = 0
		}
	}
	for k := range ordered {
		s := ordered[(start+k)%len(ordered)]
		if active[s] {
			return s
		}
	}
	return 0
}
