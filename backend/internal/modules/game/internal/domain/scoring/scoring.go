// Package scoring is the per-answer formula of docs/game/scoring.md
// (ruleset 1, D-037): a pure, deterministic function with every number
// kept in the Ruleset so a later ruleset never changes old matches.
package scoring

import "github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain"

// Ruleset holds every scoring number (game.matches.ruleset_version).
type Ruleset struct {
	Version int
	// BaseMC / BaseTF are the base points before the difficulty multiplier.
	BaseMC, BaseTF int
	// MultiplierPct is the difficulty multiplier in percent for difficulty
	// 1–5 (index 0 = difficulty 1).
	MultiplierPct [5]int
	// SpeedMaxMC / SpeedMaxTF are the speed bonus at response time 0.
	SpeedMaxMC, SpeedMaxTF int
	// Streak bonus: StreakBonusLow from StreakLow correct in a row,
	// StreakBonusHigh from StreakHigh (cap).
	StreakLow, StreakBonusLow, StreakHigh, StreakBonusHigh int
}

// Ruleset1 is scoring.md as accepted (D-037).
var Ruleset1 = Ruleset{
	Version: 1,
	BaseMC:  100, BaseTF: 50,
	MultiplierPct: [5]int{80, 90, 100, 110, 120},
	SpeedMaxMC:    100, SpeedMaxTF: 50,
	StreakLow: 3, StreakBonusLow: 20, StreakHigh: 5, StreakBonusHigh: 40,
}

// Current is the ruleset new matches use.
var Current = Ruleset1

// Input is one answer to score.
type Input struct {
	Correct      bool
	QuestionType domain.QuestionType
	Difficulty   int // difficulty_used, 1–5
	ResponseMS   int
	WindowMS     int
	// StreakBefore is the number of correct answers in a row before this one.
	StreakBefore int
}

// Points is the result with its breakdown (shown at reveal, scoring §5).
type Points struct {
	Base, Speed, Streak, Total int
	StreakAfter                int
}

// divRound divides non-negative integers rounding half up.
func divRound(num, den int) int {
	if den <= 0 {
		return 0
	}
	return (2*num + den) / (2 * den)
}

// Score computes one answer's points (scoring.md §2).
func (r Ruleset) Score(in Input) Points {
	if !in.Correct {
		return Points{}
	}
	d := min(max(in.Difficulty, 1), 5)
	base, speedMax := r.BaseMC, r.SpeedMaxMC
	if in.QuestionType == domain.TrueFalse {
		base, speedMax = r.BaseTF, r.SpeedMaxTF
	}
	p := Points{Base: divRound(base*r.MultiplierPct[d-1], 100), StreakAfter: in.StreakBefore + 1}
	if in.WindowMS > 0 {
		remaining := min(max(in.WindowMS-in.ResponseMS, 0), in.WindowMS)
		p.Speed = divRound(speedMax*remaining, in.WindowMS)
	}
	switch {
	case p.StreakAfter >= r.StreakHigh:
		p.Streak = r.StreakBonusHigh
	case p.StreakAfter >= r.StreakLow:
		p.Streak = r.StreakBonusLow
	}
	p.Total = p.Base + p.Speed + p.Streak
	return p
}

// Max is the highest points one answer can earn (scoring §2.4): every
// side round is capped at this scale too (team-modes §1).
func (r Ruleset) Max(t domain.QuestionType) int {
	if t == domain.TrueFalse {
		return divRound(r.BaseTF*r.MultiplierPct[4], 100) + r.SpeedMaxTF + r.StreakBonusHigh
	}
	return divRound(r.BaseMC*r.MultiplierPct[4], 100) + r.SpeedMaxMC + r.StreakBonusHigh
}
