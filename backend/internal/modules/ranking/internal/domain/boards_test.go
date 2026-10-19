package domain

import (
	"testing"
	"time"
)

func TestPeriodsUseIstanbulBoundaries(t *testing.T) {
	// Sunday 2026-10-18 22:30 UTC is Monday 01:30 in Istanbul (UTC+3).
	ps := Periods(time.Date(2026, 10, 18, 22, 30, 0, 0, time.UTC))
	want := []string{"d:2026-10-19", "w:2026-W43", "m:2026-10", "a"}
	for i, p := range ps {
		if p.ID != want[i] {
			t.Fatalf("period %d = %s, want %s", i, p.ID, want[i])
		}
	}
	if got := ps[1].Start.UTC(); !got.Equal(time.Date(2026, 10, 18, 21, 0, 0, 0, time.UTC)) {
		t.Fatalf("week start %v", got)
	}
	if ps[0].TTL != DailyTTL || ps[3].TTL != 0 {
		t.Fatal("TTLs")
	}
	if k := BoardKey(ps[1].ID, "team", "country:TR", "all"); k != "rank:lb:w:2026-W43:team:country:TR:all" {
		t.Fatal(k)
	}
	if s := Scopes([]string{"TR", "", "TR", "DE"}); len(s) != 3 || s[2] != "country:DE" {
		t.Fatal(s)
	}
}
