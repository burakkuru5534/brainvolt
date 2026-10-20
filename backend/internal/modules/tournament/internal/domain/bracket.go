// Package domain holds the single-elimination bracket rules (D-090):
// size, first-round pairings with byes for the top seeds, where a winner
// goes next and the place an entry finishes on.
package domain

// MaxEntrants caps a tournament (a 64 bracket, six rounds).
const MaxEntrants = 64

// Size is the bracket size for n entrants: the next power of two (≥ 2).
func Size(n int) int {
	s := 2
	for s < n {
		s *= 2
	}
	return s
}

// Rounds is the number of rounds of a bracket of size.
func Rounds(size int) int {
	r := 0
	for s := size; s > 1; s /= 2 {
		r++
	}
	return r
}

// Pair is a first-round slot: two seeds (1-based); 0 is a bye.
type Pair struct{ A, B int }

// FirstRound pairs n seeds in a bracket of Size(n): 1 meets the lowest
// seed, the top two can only meet in the final, and byes go to the top
// seeds. With n > size/2 no slot has two byes.
func FirstRound(n int) []Pair {
	size := Size(n)
	order := []int{1, 2}
	for len(order) < size {
		next := make([]int, 0, 2*len(order))
		for _, s := range order {
			next = append(next, s, 2*len(order)+1-s)
		}
		order = next
	}
	pairs := make([]Pair, 0, size/2)
	for i := 0; i < size; i += 2 {
		p := Pair{A: order[i], B: order[i+1]}
		if p.B > n {
			p.B = 0
		}
		pairs = append(pairs, p)
	}
	return pairs
}

// Next is where the winner of (round, slot) plays: the next round's slot
// and whether it takes the A side.
func Next(round, slot int) (nextRound, nextSlot int, sideA bool) {
	return round + 1, (slot + 1) / 2, slot%2 == 1
}

// LoserPlace is the place of an entry knocked out in round of rounds:
// the final's loser is 2nd, semi-final losers share 3rd, and so on.
func LoserPlace(round, rounds int) int {
	return 1<<(rounds-round) + 1
}
