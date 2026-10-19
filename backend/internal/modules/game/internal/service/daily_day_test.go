package service

import (
	"testing"
	"time"
)

// The day boundary is 00:00 Europe/Istanbul (UTC+3): a run started at
// 23:59:50 belongs to that day, the next day starts at midnight.
func TestDailyDayIstanbulBoundary(t *testing.T) {
	for _, c := range []struct {
		at   string
		want string
	}{
		{"2026-10-19T20:59:50Z", "2026-10-19"}, // 23:59:50 in Istanbul
		{"2026-10-19T21:00:00Z", "2026-10-20"}, // 00:00
		{"2026-10-19T02:00:00Z", "2026-10-19"},
	} {
		at, err := time.Parse(time.RFC3339, c.at)
		if err != nil {
			t.Fatal(err)
		}
		if got := DailyDay(at).Format(time.DateOnly); got != c.want {
			t.Fatalf("%s → %s, want %s", c.at, got, c.want)
		}
	}
}
