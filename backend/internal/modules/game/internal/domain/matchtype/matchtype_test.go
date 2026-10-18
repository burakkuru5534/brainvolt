package matchtype

import (
	"slices"
	"testing"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain"
)

func TestMT01FixedLengths(t *testing.T) {
	for typ, n := range map[domain.MatchType]int{domain.QuickDuel: 7, domain.Ranked: 10, domain.Marathon: 25} {
		ty := For(typ)
		if ty.QuestionCount != n {
			t.Fatalf("%s: %d questions", typ, ty.QuestionCount)
		}
		if e := ty.EndCheck(RoundOutcome{Round: n - 1, SideCorrect: [2]bool{true, false}}, n); e.Finished {
			t.Fatalf("%s ended early", typ)
		}
		if e := ty.EndCheck(RoundOutcome{Round: n}, n); !e.Finished || e.Reason != domain.EndAllRoundsPlayed {
			t.Fatalf("%s: %+v", typ, e)
		}
	}
}

func TestMT02MT03SuddenDeath(t *testing.T) {
	sd := For(domain.SuddenDeath)
	if e := sd.EndCheck(RoundOutcome{Round: 4, SideCorrect: [2]bool{true, false}}, 0); !e.Finished || e.Reason != domain.EndSuddenDeathDecided {
		t.Fatalf("one side right: %+v", e)
	}
	for _, c := range [][2]bool{{true, true}, {false, false}} {
		if e := sd.EndCheck(RoundOutcome{Round: 4, SideCorrect: c}, 0); e.Finished {
			t.Fatalf("%v must continue", c)
		}
	}
	if e := sd.EndCheck(RoundOutcome{Round: 30, SideCorrect: [2]bool{true, true}}, 0); !e.Finished || e.Reason != domain.EndAllRoundsPlayed {
		t.Fatalf("cap: %+v", e)
	}
}

func TestMT07Curves(t *testing.T) {
	curve := func(ty Type, n int) []int {
		out := make([]int, n)
		for i := range out {
			out[i] = ty.Difficulty(i + 1)
		}
		return out
	}
	if got := curve(For(domain.QuickDuel), 7); !slices.Equal(got, []int{2, 2, 3, 3, 3, 4, 4}) {
		t.Fatalf("quick duel %v", got)
	}
	if got := curve(For(domain.Ranked), 10); !slices.Equal(got, []int{2, 3, 3, 3, 4, 4, 4, 4, 5, 5}) {
		t.Fatalf("ranked %v", got)
	}
	if got := curve(For(domain.SuddenDeath), 12); !slices.Equal(got, []int{2, 2, 2, 3, 3, 3, 4, 4, 4, 5, 5, 5}) {
		t.Fatalf("sudden death %v", got)
	}
	m := curve(For(domain.Marathon), 25)
	for i, want := range []int{2, 3, 3, 4, 5} {
		for _, d := range m[i*5 : i*5+5] {
			if d != want {
				t.Fatalf("marathon block %d: %v", i, m)
			}
		}
	}
}

func TestTypeRules(t *testing.T) {
	if r := For(domain.Ranked); !r.Questions.MultipleChoiceOnly || r.Questions.MaxPerCategory != 3 || r.AnswerWindowMS != 15_000 {
		t.Fatalf("ranked %+v", r)
	}
	if m := For(domain.Marathon); m.HalfTimeAfter != 12 || m.HalfTimeMS != 15_000 || m.AnswerWindowMS != 12_000 {
		t.Fatalf("MT09 marathon %+v", m)
	}
	if s := For(domain.SuddenDeath); s.AnswerWindowMS != 10_000 || s.QuestionCount != 0 {
		t.Fatalf("sudden death %+v", s)
	}
	for typ, w := range map[domain.MatchType]int{domain.QuickDuel: 20_000, domain.Ranked: 30_000, domain.SuddenDeath: 20_000, domain.Marathon: 60_000} {
		if For(typ).ReconnectWindowMS != w {
			t.Fatalf("%s reconnect window", typ)
		}
	}
	rl := RankedLengthQuickDuel()
	if rl.Name != domain.QuickDuel || rl.QuestionCount != 10 || !rl.Questions.MultipleChoiceOnly {
		t.Fatalf("ranked length %+v", rl)
	}
}
