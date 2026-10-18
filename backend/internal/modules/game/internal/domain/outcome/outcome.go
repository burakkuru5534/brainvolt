// Package outcome decides results and absence rules of docs/game/
// edge-cases.md (D-048): tie-breakers, the disconnect limits and when a
// side forfeits or a match must be cancelled.
package outcome

// MaxDisconnects is how many disconnects one participant may have in a
// match; the next one is a forfeit.
const MaxDisconnects = 3

// SideTotals are what the tie-breakers compare.
type SideTotals struct {
	Score   int
	Correct int
	// CorrectResponseMS is the sum of the response times used to score the
	// side's correct rounds.
	CorrectResponseMS int
}

// TieBreak names the step that decided a tie (match.completed tie_break).
type TieBreak string

// Tie-break steps.
const (
	ByScore        TieBreak = ""
	ByCorrect      TieBreak = "more_correct"
	ByResponseTime TieBreak = "faster_correct"
	NoTieBreak     TieBreak = "draw"
)

// Decide returns the winning side (1 or 2) or 0 for a draw: higher score,
// then more correct rounds, then lower total correct response time.
func Decide(a, b SideTotals) (winner int, by TieBreak) {
	switch {
	case a.Score != b.Score:
		return pick(a.Score > b.Score), ByScore
	case a.Correct != b.Correct:
		return pick(a.Correct > b.Correct), ByCorrect
	case a.CorrectResponseMS != b.CorrectResponseMS:
		return pick(a.CorrectResponseMS < b.CorrectResponseMS), ByResponseTime
	}
	return 0, NoTieBreak
}

func pick(first bool) int {
	if first {
		return 1
	}
	return 2
}

// ExceedsDisconnectLimit reports a forfeit: more than 3 disconnects, or
// total time away over twice the reconnect window.
func ExceedsDisconnectLimit(disconnects, awayMS, windowMS int) bool {
	return disconnects > MaxDisconnects || awayMS > 2*windowMS
}
