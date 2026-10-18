package scoring

import (
	"testing"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain"
)

const window = 15_000

// TestSC01WorkedExamples reproduces every row of scoring.md §3.
func TestSC01WorkedExamples(t *testing.T) {
	mc, tf := domain.MultipleChoice, domain.TrueFalse
	cases := []struct {
		name                string
		in                  Input
		base, speed, streak int
		total               int
	}{
		{"MC d3 3000ms streak1", Input{true, mc, 3, 3_000, window, 0}, 100, 80, 0, 180},
		{"MC d5 7500ms streak4", Input{true, mc, 5, 7_500, window, 3}, 120, 50, 20, 190},
		{"MC d2 14900ms streak6", Input{true, mc, 2, 14_900, window, 5}, 90, 1, 40, 131},
		{"MC wrong", Input{false, mc, 3, 2_000, window, 4}, 0, 0, 0, 0},
		{"TF d3 1500ms streak3", Input{true, tf, 3, 1_500, window, 2}, 50, 45, 20, 115},
	}
	for _, tc := range cases {
		p := Ruleset1.Score(tc.in)
		if p.Base != tc.base || p.Speed != tc.speed || p.Streak != tc.streak || p.Total != tc.total {
			t.Errorf("%s: %+v, want %d+%d+%d=%d", tc.name, p, tc.base, tc.speed, tc.streak, tc.total)
		}
	}
}

func TestSC02WrongOrTimeoutResetsStreak(t *testing.T) {
	if p := Ruleset1.Score(Input{Correct: false, StreakBefore: 7, WindowMS: window}); p != (Points{}) {
		t.Fatalf("wrong = %+v", p)
	}
}

func TestSC03SpeedBonusMonotonic(t *testing.T) {
	for _, qt := range []domain.QuestionType{domain.MultipleChoice, domain.TrueFalse} {
		prev := 1 << 30
		for ms := -500; ms <= window+500; ms += 250 {
			s := Ruleset1.Score(Input{Correct: true, QuestionType: qt, Difficulty: 3, ResponseMS: ms, WindowMS: window}).Speed
			limit := 100
			if qt == domain.TrueFalse {
				limit = 50
			}
			if s < 0 || s > limit || s > prev {
				t.Fatalf("%s at %d ms: speed %d (prev %d)", qt, ms, s, prev)
			}
			prev = s
		}
	}
}

func TestSC04Max(t *testing.T) {
	if Ruleset1.Max(domain.MultipleChoice) != 260 || Ruleset1.Max(domain.TrueFalse) != 150 {
		t.Fatal("max points")
	}
	for d := 1; d <= 5; d++ {
		for _, qt := range []domain.QuestionType{domain.MultipleChoice, domain.TrueFalse} {
			if p := Ruleset1.Score(Input{true, qt, d, 0, window, 99}); p.Total > Ruleset1.Max(qt) {
				t.Fatalf("%s d%d: %d", qt, d, p.Total)
			}
		}
	}
}

func TestSC05StreakThresholds(t *testing.T) {
	for before, want := range map[int]int{0: 0, 1: 0, 2: 20, 3: 20, 4: 40, 9: 40} {
		p := Ruleset1.Score(Input{true, domain.MultipleChoice, 3, window, window, before})
		if p.Streak != want || p.StreakAfter != before+1 {
			t.Errorf("streak before %d: bonus %d after %d, want %d", before, p.Streak, p.StreakAfter, want)
		}
	}
}
