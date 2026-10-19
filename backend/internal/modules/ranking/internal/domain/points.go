package domain

import "math"

// Result is a side's outcome from one entity's view.
type Result string

// Results.
const (
	Win  Result = "win"
	Draw Result = "draw"
	Loss Result = "loss"
)

// Score is the Glicko-2 outcome score (forfeit = loss / win, D-048).
func (r Result) Score() float64 {
	switch r {
	case Win:
		return 1
	case Draw:
		return 0.5
	case Loss:
	}
	return 0
}

// Season Points (progression-formulas §2).
const (
	SPWin            = 30
	SPDraw           = 10
	SPLoss           = -15
	SPStreakStep     = 5
	SPStreakMaxBonus = 15
)

// SPDelta is a ranked match's SP change before the league floor: the
// win-streak bonus from the 3rd consecutive win (streak includes this
// win), and leaver protection (a loss while a teammate left → 0).
func SPDelta(r Result, streak int, leaverProtected bool) int {
	switch r {
	case Win:
		bonus := 0
		if streak >= 3 {
			bonus = min(SPStreakStep*(streak-2), SPStreakMaxBonus)
		}
		return SPWin + bonus
	case Draw:
		return SPDraw
	case Loss:
	}
	if leaverProtected {
		return 0
	}
	return SPLoss
}

// League is a league with its division floors (lowest division first:
// III, II, I).
type League struct {
	Code      string
	Floor     int
	Divisions []int
}

// Leagues are the leagues of ruleset 1 (progression-formulas §3).
var Leagues = []League{
	{Code: "bronze", Floor: 0, Divisions: []int{0, 100, 200}},
	{Code: "silver", Floor: 300, Divisions: []int{300, 466, 633}},
	{Code: "gold", Floor: 800, Divisions: []int{800, 1066, 1333}},
	{Code: "platinum", Floor: 1600, Divisions: []int{1600, 2000, 2400}},
	{Code: "diamond", Floor: 2800, Divisions: []int{2800}},
}

// Placement is a league and division (3 = III … 1 = I; Diamond has 1).
type Placement struct {
	League   string
	Division int
}

// Place returns the league and division of an SP total.
func Place(sp int) Placement {
	l := Leagues[0]
	for _, x := range Leagues {
		if sp >= x.Floor {
			l = x
		}
	}
	div := len(l.Divisions)
	for i, floor := range l.Divisions {
		if sp >= floor {
			div = len(l.Divisions) - i
		}
	}
	return Placement{League: l.Code, Division: div}
}

// LeagueIndex orders leagues (bronze 0 … diamond 4).
func LeagueIndex(code string) int {
	for i, l := range Leagues {
		if l.Code == code {
			return i
		}
	}
	return 0
}

// ApplySP adds delta with league floor protection: SP never drops below
// the floor of the league held before the match, nor below 0.
func ApplySP(current, delta int) int {
	floor := Leagues[LeagueIndex(Place(current).League)].Floor
	return max(current+delta, floor, 0)
}

// Leaderboard points (progression-formulas §4).
const (
	LPWin     = 20
	LPDraw    = 10
	LPCorrect = 2
)

// ResultPoints are the result part of leaderboard points.
func ResultPoints(r Result) int {
	switch r {
	case Win:
		return LPWin
	case Draw:
		return LPDraw
	case Loss:
	}
	return 0
}

// LeaderboardPoints are an entity's points for a match: result + 2 per
// correct round.
func LeaderboardPoints(r Result, correct int) int { return ResultPoints(r) + LPCorrect*correct }

// CategoryPoints is a category board's share (§4): the result points in
// proportion to the category's rounds, plus 2 per correct round in it.
func CategoryPoints(r Result, roundsInCategory, totalRounds, correctInCategory int) int {
	if totalRounds == 0 {
		return 0
	}
	share := math.Round(float64(ResultPoints(r)) * float64(roundsInCategory) / float64(totalRounds))
	return int(share) + LPCorrect*correctInCategory
}
