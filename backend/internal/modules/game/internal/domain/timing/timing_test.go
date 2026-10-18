package timing

import "testing"

// TestTM01EqualTapEqualResponse is timing.md §5's example.
func TestTM01EqualTapEqualResponse(t *testing.T) {
	const showAt, deadline = 10_000, 25_000
	if a, b := ResponseMS(14_020, showAt, 40), ResponseMS(14_120, showAt, 240); a != 4_000 || b != 4_000 {
		t.Fatalf("response %d / %d, want 4000", a, b)
	}
	if !Accepted(25_120, deadline, 40) || Accepted(25_121, deadline, 40) {
		t.Fatal("A's grace ends at 25 120")
	}
	if !Accepted(25_220, deadline, 240) || Accepted(25_221, deadline, 240) {
		t.Fatal("B's grace ends at 25 220")
	}
}

func TestTM02MedianAndCaps(t *testing.T) {
	var r RTT
	for _, s := range []int{40, 42, 38, 41, 39, 40, 43, 40, 41, 900} {
		r.Add(s)
	}
	if e := r.Estimate(); e < 38 || e > 43 {
		t.Fatalf("one spike moved the median: %d", e)
	}
	for range 10 {
		r.Add(2_000)
	}
	if r.Estimate() != 2_000 || Uplink(r.Estimate()) != UplinkCapMS || !r.Weak() {
		t.Fatalf("sustained inflation: rtt %d uplink %d", r.Estimate(), Uplink(r.Estimate()))
	}
}

func TestTM03TM04ShowAtDeadlineGrace(t *testing.T) {
	if got := ShowAt(1_000, []int{40, 240}); got != 1_000+120+30 {
		t.Fatalf("show_at = %d", got)
	}
	if got := ShowAt(1_000, []int{2_000}); got != 1_000+250 {
		t.Fatalf("buffer cap: %d", got)
	}
	if Deadline(5_000, 15_000) != 20_000 {
		t.Fatal("deadline = show_at + window")
	}
	for _, rtt := range []int{0, 40, 240, 10_000} {
		if g := Grace(rtt); g < 100 || g > 250 {
			t.Fatalf("grace(%d) = %d", rtt, g)
		}
	}
	if ResponseMS(900, 1_000, 40) != 0 {
		t.Fatal("response never negative")
	}
}
