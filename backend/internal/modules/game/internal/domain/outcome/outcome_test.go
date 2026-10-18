package outcome

import "testing"

func TestEC01TieBreakers(t *testing.T) {
	cases := []struct {
		a, b   SideTotals
		winner int
		by     TieBreak
	}{
		{SideTotals{900, 5, 30_000}, SideTotals{800, 6, 10_000}, 1, ByScore},
		{SideTotals{900, 6, 30_000}, SideTotals{900, 5, 10_000}, 1, ByCorrect},
		{SideTotals{900, 5, 30_000}, SideTotals{900, 5, 29_999}, 2, ByResponseTime},
		{SideTotals{900, 5, 30_000}, SideTotals{900, 5, 30_000}, 0, NoTieBreak},
	}
	for _, tc := range cases {
		if w, by := Decide(tc.a, tc.b); w != tc.winner || by != tc.by {
			t.Errorf("%+v vs %+v = %d %s", tc.a, tc.b, w, by)
		}
	}
}

func TestEC04DisconnectLimit(t *testing.T) {
	if ExceedsDisconnectLimit(3, 40_000, 20_000) {
		t.Fatal("3 disconnects and exactly 2× window are allowed")
	}
	if !ExceedsDisconnectLimit(4, 0, 20_000) || !ExceedsDisconnectLimit(1, 40_001, 20_000) {
		t.Fatal("4th disconnect / away > 2× window is a forfeit")
	}
}
