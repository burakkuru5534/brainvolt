package teammode

import (
	"testing"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain/scoring"
)

var (
	correct = uuid.MustParse("00000000-0000-7000-8000-00000000000c")
	wrong   = uuid.MustParse("00000000-0000-7000-8000-00000000000a")
	a1      = uuid.MustParse("00000000-0000-7000-8000-0000000000a1")
	a2      = uuid.MustParse("00000000-0000-7000-8000-0000000000a2")
	a3      = uuid.MustParse("00000000-0000-7000-8000-0000000000a3")
)

func question(window int) Question {
	return Question{Type: domain.MultipleChoice, Difficulty: 3, WindowMS: window, Correct: correct}
}

// The 3v3 example of team-modes.md §5.
func example() []Member {
	return []Member{
		{UserID: a1, Seat: 1, Active: true, Answered: true, OptionID: correct, ResponseMS: 3_000, ReceivedMS: 13_000},
		{UserID: a2, Seat: 2, Active: true, Answered: true, OptionID: correct, ResponseMS: 9_000, ReceivedMS: 19_000},
		{UserID: a3, Seat: 3, Active: true, Answered: true, OptionID: wrong, ResponseMS: 5_000, ReceivedMS: 15_000},
	}
}

func sum(m map[uuid.UUID]int) int {
	s := 0
	for _, v := range m {
		s += v
	}
	return s
}

func TestTA02WorkedExample(t *testing.T) {
	base := Input{Mode: domain.FastTeamRound, Question: question(15_000), Rules: scoring.Ruleset1, SideSize: 3, Members: example()}
	for agg, want := range map[domain.Aggregation]int{domain.Average: 107, domain.FastestCorrect: 180, domain.Majority: 160} {
		in := base
		in.Aggregation = agg
		r := Aggregate(in)
		if !r.Correct || r.Points != want || sum(r.Contribution) != r.Points {
			t.Errorf("%s: correct %v points %d contributions %v, want %d", agg, r.Correct, r.Points, r.Contribution, want)
		}
	}

	shared := Aggregate(Input{
		Mode: domain.SharedAnswer, Question: question(20_000), Rules: scoring.Ruleset1, SideSize: 3, Members: example(),
		Side: SideAnswer{Answered: true, OptionID: correct, ResponseMS: 9_000, Submitter: a2},
	})
	if shared.Points != 155 || sum(shared.Contribution) != 155 || shared.Contribution[a2] != 53 {
		t.Fatalf("shared answer: %d %v", shared.Points, shared.Contribution)
	}

	turns := Aggregate(Input{
		Mode: domain.IndividualTurns, Question: question(15_000), Rules: scoring.Ruleset1, SideSize: 3, Members: example(),
		SideStreakBefore: 4, Side: SideAnswer{Answered: true, OptionID: wrong, ResponseMS: 5_000, Submitter: a3},
	})
	if turns.Points != 0 || turns.Correct || turns.SideStreakAfter != 0 {
		t.Fatalf("individual turns: %+v", turns)
	}
}

func TestTA01SameScale(t *testing.T) {
	fast := make([]Member, 4)
	for i := range fast {
		fast[i] = Member{UserID: uuid.New(), Seat: i + 1, Active: true, Answered: true, OptionID: correct, StreakBefore: 9}
	}
	for _, mode := range []domain.TeamMode{domain.FastTeamRound, domain.SharedAnswer, domain.IndividualTurns} {
		for _, agg := range []domain.Aggregation{domain.Average, domain.FastestCorrect, domain.Majority} {
			r := Aggregate(Input{
				Mode: mode, Aggregation: agg, Question: Question{Type: domain.MultipleChoice, Difficulty: 5, WindowMS: 15_000, Correct: correct},
				Rules: scoring.Ruleset1, SideSize: 4, Members: fast, SideStreakBefore: 9,
				Side: SideAnswer{Answered: true, OptionID: correct, Submitter: fast[0].UserID},
			})
			if r.Points > 260 {
				t.Fatalf("%s/%s: %d > 260", mode, agg, r.Points)
			}
		}
	}
}

func TestTA03SharedResolution(t *testing.T) {
	lock := &Lock{UserID: a2, OptionID: wrong, ResponseMS: 4_000}
	props := []Proposal{{a1, correct, 100}, {a3, correct, 200}}
	if got := ResolveShared(lock, props); got.OptionID != wrong || got.Fallback || got.Submitter != a2 {
		t.Fatalf("lock wins: %+v", got)
	}
	if got := ResolveShared(nil, props); got.OptionID != correct || !got.Fallback || got.Submitter != a1 {
		t.Fatalf("most proposed: %+v", got)
	}
	tie := []Proposal{{a1, wrong, 300}, {a2, correct, 100}}
	if got := ResolveShared(nil, tie); got.OptionID != correct {
		t.Fatalf("tie → earliest: %+v", got)
	}
	changed := []Proposal{{a1, wrong, 100}, {a1, correct, 500}, {a2, wrong, 200}}
	if got := ResolveShared(nil, changed); got.OptionID != wrong || got.Submitter != a2 {
		t.Fatalf("only the latest proposal of a member counts: %+v", got)
	}
	if got := ResolveShared(nil, nil); got.Answered {
		t.Fatalf("no proposals → timeout: %+v", got)
	}

	fallback := Aggregate(Input{
		Mode: domain.SharedAnswer, Question: question(20_000), Rules: scoring.Ruleset1, SideSize: 2,
		Members: example()[:2], Side: ResolveShared(nil, props),
	})
	if fallback.Breakdown.Speed != 0 || fallback.Points != 100 {
		t.Fatalf("fallback gets no speed bonus: %+v", fallback.Breakdown)
	}
}

func TestTA04SharedContributionRemainderToLocker(t *testing.T) {
	r := Aggregate(Input{
		Mode: domain.SharedAnswer, Question: question(20_000), Rules: scoring.Ruleset1, SideSize: 3, Members: example(),
		Side: SideAnswer{Answered: true, OptionID: correct, ResponseMS: 9_000, Submitter: a3},
	})
	if r.Contribution[a1] != 51 || r.Contribution[a2] != 51 || r.Contribution[a3] != 53 {
		t.Fatalf("155 split 51/51/53 with the remainder to the locker: %v", r.Contribution)
	}
}

func TestTA05TA06TurnRotation(t *testing.T) {
	seats := []int{1, 2, 3}
	all := map[int]bool{1: true, 2: true, 3: true}
	last, got := 0, []int{}
	for range 7 {
		last = NextTurn(last, seats, all)
		got = append(got, last)
	}
	if want := []int{1, 2, 3, 1, 2, 3, 1}; !equal(got, want) {
		t.Fatalf("rotation %v, want %v", got, want)
	}
	if s := NextTurn(1, seats, map[int]bool{1: true, 2: false, 3: true}); s != 3 {
		t.Fatalf("seat 2 disconnected → 3, got %d", s)
	}
	if s := NextTurn(3, seats, map[int]bool{1: false, 2: true, 3: true}); s != 2 {
		t.Fatalf("wrap past inactive seat 1 → 2, got %d", s)
	}
	if s := NextTurn(2, seats, map[int]bool{}); s != 0 {
		t.Fatalf("nobody active → 0 (side timeout), got %d", s)
	}
}

func equal(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestTA08AverageActiveMembers(t *testing.T) {
	members := example()
	members[2].Active, members[2].Answered = false, false // a3 disconnected
	members[1].OptionID = wrong                           // 1 of 2 active correct
	r := Aggregate(Input{Mode: domain.FastTeamRound, Aggregation: domain.Average, Question: question(15_000), Rules: scoring.Ruleset1, SideSize: 3, Members: members})
	if !r.Correct {
		t.Fatal("half of the active members correct counts as correct")
	}
	if r.Points != 60 { // round(180 / 3): the divisor stays the full side size
		t.Fatalf("points %d, want 60", r.Points)
	}
	members[0].OptionID = wrong
	if r := Aggregate(Input{Mode: domain.FastTeamRound, Aggregation: domain.Average, Question: question(15_000), Rules: scoring.Ruleset1, SideSize: 3, Members: members}); r.Correct || r.Points != 0 {
		t.Fatalf("no correct member: %+v", r)
	}
}

func TestSoloAndMajorityTimeouts(t *testing.T) {
	solo := Aggregate(Input{Question: question(15_000), Rules: scoring.Ruleset1, SideSize: 1,
		Members: []Member{{UserID: a1, Active: true, Answered: true, OptionID: correct, ResponseMS: 3_000, StreakBefore: 2}}})
	if solo.Points != 200 || solo.SideStreakAfter != 3 || solo.ResponseMS != 3_000 {
		t.Fatalf("1v1: %+v", solo)
	}
	none := Aggregate(Input{Mode: domain.FastTeamRound, Aggregation: domain.Majority, Question: question(15_000), Rules: scoring.Ruleset1, SideSize: 2,
		Members: []Member{{UserID: a1, Active: true}, {UserID: a2, Active: true}}})
	if none.Correct || none.Points != 0 || none.ResponseMS != -1 {
		t.Fatalf("majority without votes: %+v", none)
	}
}
