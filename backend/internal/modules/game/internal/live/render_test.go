package live

import (
	"testing"

	"github.com/google/uuid"
)

// D-100: a reveal shows the correct option only to players who got it.
func TestRevealHidesTheAnswerFromWrongAnswers(t *testing.T) {
	right, wrong, turn := uuid.New(), uuid.New(), uuid.New()
	payload := func() map[string]any {
		return map[string]any{
			"correct_option_id": "opt-c", "explanation": "because",
			"answers": []any{
				map[string]any{"user_id": right.String(), "option_id": "opt-c", "correct": true},
				map[string]any{"user_id": wrong.String(), "option_id": "opt-a", "correct": false},
			},
			"sides": []any{
				map[string]any{"side_no": float64(1), "correct": true},
				map[string]any{"side_no": float64(2), "correct": false},
			},
		}
	}

	d := payload()
	if !earnedAnswer(d, recipient{UserID: right, Side: 2}) {
		t.Fatal("a right answer earns the answer")
	}
	if earnedAnswer(d, recipient{UserID: wrong, Side: 1}) {
		t.Fatal("a wrong answer does not, whatever the side did")
	}
	// No own row (Individual Turns): the side's result decides.
	if !earnedAnswer(d, recipient{UserID: turn, Side: 1}) || earnedAnswer(d, recipient{UserID: turn, Side: 2}) {
		t.Fatal("side result for players without an answer row")
	}

	orig := payload()
	answers := orig["answers"].([]any)
	hideAnswer(orig)
	if orig["correct_option_id"] != nil || orig["explanation"] != nil || orig["answer_hidden"] != true {
		t.Fatalf("hidden reveal %v", orig)
	}
	got := orig["answers"].([]any)
	if got[0].(map[string]any)["option_id"] != nil || got[0].(map[string]any)["correct"] != true {
		t.Fatalf("the right answer's choice must go, its result stays: %v", got[0])
	}
	if got[1].(map[string]any)["option_id"] != "opt-a" {
		t.Fatalf("a wrong choice may stay: %v", got[1])
	}
	// The stored payload other recipients render from is untouched.
	if answers[0].(map[string]any)["option_id"] != "opt-c" {
		t.Fatal("hideAnswer changed the shared payload")
	}
}
