package domain

import (
	"github.com/google/uuid"
)

// Candidate is an eligible question as selection needs it.
type Candidate struct {
	ID            uuid.UUID
	Type          QuestionType
	Difficulty    int // difficulty used: calibrated (rounded) or author's
	CategoryID    uuid.UUID
	SubcategoryID uuid.UUID
}

// SelectRules are the mix rules of docs/game/match-types.md §2 (D-038).
type SelectRules struct {
	// Difficulties is the target difficulty per round; its length is the
	// number of questions.
	Difficulties []int
	// MaxPerCategory caps questions per top-level category (0 = none).
	MaxPerCategory int
	// MaxTrueFalsePercent caps the true/false share; true/false questions
	// are never consecutive.
	MaxTrueFalsePercent int
	// HomeCategory (Revenge Match): ⌊n/2⌋ questions from it, interleaved
	// so no category appears more than MaxRun times in a row.
	HomeCategory uuid.UUID
	MaxRun       int
}

// Select picks questions in round order from pool (already in random
// order): no two consecutive questions from the same sub-category, the
// category cap, the true/false share, and per round the nearest available
// difficulty (exact, then lower, then higher). It returns false when the
// pool cannot satisfy the rules.
func Select(pool []Candidate, r SelectRules) ([]Candidate, bool) {
	n := len(r.Difficulties)
	maxTF := n * r.MaxTrueFalsePercent / 100
	home := map[int]bool{}
	if r.HomeCategory != uuid.Nil {
		// Home questions on alternating rounds keep runs short.
		for i, k := 0, 0; i < n && k < n/2; i, k = i+2, k+1 {
			home[i] = true
		}
	}
	used := make([]bool, len(pool))
	perCategory := map[uuid.UUID]int{}
	out := make([]Candidate, 0, n)
	tf := 0
	for round := range n {
		target := r.Difficulties[round]
		best, bestDistance := -1, 0
		for i, c := range pool {
			if used[i] || !fits(c, round, out, perCategory, tf, maxTF, r, home) {
				continue
			}
			if d := distance(c.Difficulty, target); best < 0 || d < bestDistance {
				best, bestDistance = i, d
			}
		}
		if best < 0 {
			return nil, false
		}
		c := pool[best]
		used[best] = true
		out = append(out, c)
		perCategory[c.CategoryID]++
		if c.Type == TrueFalse {
			tf++
		}
	}
	return out, true
}

func fits(c Candidate, round int, out []Candidate, perCategory map[uuid.UUID]int, tf, maxTF int, r SelectRules, home map[int]bool) bool {
	if r.HomeCategory != uuid.Nil {
		if home[round] != (c.CategoryID == r.HomeCategory) {
			return false
		}
	} else if r.MaxPerCategory > 0 && perCategory[c.CategoryID] >= r.MaxPerCategory {
		return false
	}
	if len(out) > 0 && out[len(out)-1].SubcategoryID == c.SubcategoryID {
		return false
	}
	if c.Type == TrueFalse && (tf >= maxTF || (len(out) > 0 && out[len(out)-1].Type == TrueFalse)) {
		return false
	}
	if r.MaxRun > 0 && len(out) >= r.MaxRun {
		run := true
		for _, prev := range out[len(out)-r.MaxRun:] {
			if prev.CategoryID != c.CategoryID {
				run = false
				break
			}
		}
		if run {
			return false
		}
	}
	return true
}

// distance orders difficulties: exact first, then lower, then higher.
func distance(d, target int) int {
	if d <= target {
		return 2 * (target - d)
	}
	return 2*(d-target) + 1
}
