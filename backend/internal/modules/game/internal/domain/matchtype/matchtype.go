// Package matchtype holds the per-type rules of docs/game/match-types.md
// (D-038) and the reconnect windows of edge-cases.md §2 (D-048): length,
// answer window, difficulty curve, question rules and the end check.
package matchtype

import (
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain"
)

// SharedAnswerExtraMS is added to the answer window in Shared Answer
// rounds (D-038, T05).
const SharedAnswerExtraMS = 5_000

// SuddenDeathCap is the safety cap of Sudden Death rounds.
const SuddenDeathCap = 30

// SelectionBatch is how many questions open-ended types select at a time.
const SelectionBatch = 10

// QuestionRules are the eligibility and mix rules of the selected
// questions (match-types §2).
type QuestionRules struct {
	// MultipleChoiceOnly: Ranked and ranked-length matches (D-024).
	MultipleChoiceOnly bool
	// MaxTrueFalsePercent is the true/false share cap (never consecutive).
	MaxTrueFalsePercent int
	// MaxPerCategory caps questions per top-level category; 0 = no cap
	// (Sudden Death has none, being open-ended).
	MaxPerCategory int
}

// Type is one match type's rule set.
type Type struct {
	Name domain.MatchType
	// QuestionCount is 0 for open-ended types.
	QuestionCount  int
	AnswerWindowMS int
	Questions      QuestionRules
	// ReconnectWindowMS per disconnect (D-048).
	ReconnectWindowMS int
	// HalfTimeAfter is the round after which a break is taken (Marathon:
	// 12); 0 = none. HalfTimeMS is its length.
	HalfTimeAfter int
	HalfTimeMS    int
	curve         func(round int) int
}

// Difficulty is the target difficulty of a round (1-based).
func (t Type) Difficulty(round int) int { return t.curve(round) }

func fixed(curve ...int) func(int) int {
	return func(round int) int {
		if round < 1 {
			round = 1
		}
		if round > len(curve) {
			return curve[len(curve)-1]
		}
		return curve[round-1]
	}
}

var (
	quickDuel = Type{
		Name: domain.QuickDuel, QuestionCount: 7, AnswerWindowMS: 15_000,
		Questions:         QuestionRules{MaxTrueFalsePercent: 30, MaxPerCategory: 2},
		ReconnectWindowMS: 20_000,
		curve:             fixed(2, 2, 3, 3, 3, 4, 4),
	}
	ranked = Type{
		Name: domain.Ranked, QuestionCount: 10, AnswerWindowMS: 15_000,
		Questions:         QuestionRules{MultipleChoiceOnly: true, MaxPerCategory: 3},
		ReconnectWindowMS: 30_000,
		curve:             fixed(2, 3, 3, 3, 4, 4, 4, 4, 5, 5),
	}
	suddenDeath = Type{
		Name: domain.SuddenDeath, AnswerWindowMS: 10_000,
		Questions:         QuestionRules{MaxTrueFalsePercent: 30},
		ReconnectWindowMS: 20_000,
		// Starts at 2, +1 every 3 rounds, up to 5.
		curve: func(round int) int { return min(5, 2+(max(round, 1)-1)/3) },
	}
	marathon = Type{
		Name: domain.Marathon, QuestionCount: 25, AnswerWindowMS: 12_000,
		Questions:         QuestionRules{MaxTrueFalsePercent: 30, MaxPerCategory: 5},
		ReconnectWindowMS: 60_000, HalfTimeAfter: 12, HalfTimeMS: 15_000,
		// Five blocks of five: 2, 3, 3, 4, 5.
		curve: func(round int) int { return []int{2, 3, 3, 4, 5}[min(4, (max(round, 1)-1)/5)] },
	}
)

// For returns the rules of t (zero Type for unknown values).
func For(t domain.MatchType) Type {
	switch t {
	case domain.QuickDuel:
		return quickDuel
	case domain.Ranked:
		return ranked
	case domain.SuddenDeath:
		return suddenDeath
	case domain.Marathon:
		return marathon
	}
	return Type{}
}

// RankedLengthQuickDuel is a rematch or revenge of a Ranked match: Quick
// Duel rules (unrated) with Ranked's length, curve and question rules (T01).
func RankedLengthQuickDuel() Type {
	t := ranked
	t.Name = domain.QuickDuel
	t.ReconnectWindowMS = quickDuel.ReconnectWindowMS
	return t
}

// RoundOutcome is what the end check needs about the last revealed round.
type RoundOutcome struct {
	Round       int
	SideCorrect [2]bool
}

// End is the end-check decision.
type End struct {
	Finished bool
	Reason   domain.EndReason
}

// EndCheck decides after every reveal whether the match is over
// (match-types §4). questionCount is the configured length (creator quiz
// lengths differ from the type's).
func (t Type) EndCheck(last RoundOutcome, questionCount int) End {
	if t.Name == domain.SuddenDeath && questionCount == 0 {
		if last.SideCorrect[0] != last.SideCorrect[1] {
			return End{Finished: true, Reason: domain.EndSuddenDeathDecided}
		}
		if last.Round >= SuddenDeathCap {
			return End{Finished: true, Reason: domain.EndAllRoundsPlayed}
		}
		return End{}
	}
	if last.Round >= questionCount {
		return End{Finished: true, Reason: domain.EndAllRoundsPlayed}
	}
	return End{}
}
