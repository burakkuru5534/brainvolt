package domain

import (
	"slices"
	"testing"
)

func TestBracket(t *testing.T) {
	for n, size := range map[int]int{2: 2, 3: 4, 4: 4, 5: 8, 8: 8, 9: 16, 64: 64} {
		if Size(n) != size {
			t.Errorf("Size(%d) = %d, want %d", n, Size(n), size)
		}
	}
	if Rounds(8) != 3 || Rounds(2) != 1 || Rounds(64) != 6 {
		t.Fatal("rounds")
	}
	if got := FirstRound(8); !slices.Equal(got, []Pair{{1, 8}, {4, 5}, {2, 7}, {3, 6}}) {
		t.Fatalf("8: %v", got)
	}
	// Six entrants: seeds 1 and 2 get the byes.
	if got := FirstRound(6); !slices.Equal(got, []Pair{{1, 0}, {4, 5}, {2, 0}, {3, 6}}) {
		t.Fatalf("6: %v", got)
	}
	for n := 2; n <= MaxEntrants; n++ {
		seen := map[int]bool{}
		for _, p := range FirstRound(n) {
			if p.A == 0 || p.A > n {
				t.Fatalf("%d: bad A %v", n, p)
			}
			seen[p.A], seen[p.B] = true, true
		}
		for s := 1; s <= n; s++ {
			if !seen[s] {
				t.Fatalf("%d: seed %d missing", n, s)
			}
		}
	}
	if r, s, a := Next(1, 3); r != 2 || s != 2 || !a {
		t.Fatal("next 3")
	}
	if r, s, a := Next(2, 2); r != 3 || s != 1 || a {
		t.Fatal("next 2")
	}
	if LoserPlace(3, 3) != 2 || LoserPlace(2, 3) != 3 || LoserPlace(1, 3) != 5 {
		t.Fatal("places")
	}
}
