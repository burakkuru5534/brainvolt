package domain

import "testing"

func TestSeasonPoints(t *testing.T) {
	for _, c := range []struct {
		r      Result
		streak int
		leaver bool
		want   int
	}{
		{Win, 1, false, 30}, {Win, 2, false, 30}, {Win, 3, false, 35}, {Win, 4, false, 40},
		{Win, 5, false, 45}, {Win, 9, false, 45}, {Draw, 0, false, 10}, {Loss, 0, false, -15}, {Loss, 0, true, 0},
	} {
		if got := SPDelta(c.r, c.streak, c.leaver); got != c.want {
			t.Errorf("SPDelta(%s, %d, %v) = %d, want %d", c.r, c.streak, c.leaver, got, c.want)
		}
	}
}

func TestLeaguesAndFloor(t *testing.T) {
	for sp, want := range map[int]Placement{
		0: {"bronze", 3}, 99: {"bronze", 3}, 100: {"bronze", 2}, 299: {"bronze", 1}, 300: {"silver", 3},
		466: {"silver", 2}, 799: {"silver", 1}, 800: {"gold", 3}, 1600: {"platinum", 3}, 2799: {"platinum", 1},
		2800: {"diamond", 1}, 9000: {"diamond", 1},
	} {
		if got := Place(sp); got != want {
			t.Errorf("Place(%d) = %+v, want %+v", sp, got, want)
		}
	}
	// No relegation within a season: the floor of the current league holds;
	// divisions inside it can go down.
	for _, c := range [][3]int{{310, -15, 300}, {300, -15, 300}, {520, -15, 505}, {10, -15, 0}, {800, -15, 800}, {290, 30, 320}} {
		if got := ApplySP(c[0], c[1]); got != c[2] {
			t.Errorf("ApplySP(%d, %d) = %d, want %d", c[0], c[1], got, c[2])
		}
	}
}

// progression-formulas §10 and the category share (§4).
func TestLeaderboardPoints(t *testing.T) {
	if got := LeaderboardPoints(Win, 7); got != 34 {
		t.Fatalf("ranked win with 7 correct: %d", got)
	}
	if got := LeaderboardPoints(Loss, 6); got != 12 {
		t.Fatalf("loss with 6 side-correct rounds: %d", got)
	}
	// 7 rounds: 3 history (2 correct), 4 science (1 correct), a win.
	h, s := CategoryPoints(Win, 3, 7, 2), CategoryPoints(Win, 4, 7, 1)
	if h != 9+4 || s != 11+2 {
		t.Fatalf("category points %d %d", h, s)
	}
}

func TestNextThreshold(t *testing.T) {
	for _, c := range []struct{ sp, next int }{{0, 100}, {299, 300}, {300, 466}, {2400, 2800}} {
		if got, ok := NextThreshold(c.sp); !ok || got != c.next {
			t.Fatalf("NextThreshold(%d) = %d", c.sp, got)
		}
	}
	if _, ok := NextThreshold(2800); ok {
		t.Fatal("diamond has no next threshold")
	}
}
