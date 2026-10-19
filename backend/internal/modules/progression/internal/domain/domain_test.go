package domain

import (
	"testing"
	"time"
)

func TestMatchXP(t *testing.T) {
	// progression-formulas §10: 7 correct win = 20 + 35 + 15.
	if xp := MatchXP("win", 7, false); xp != 70 {
		t.Fatal(xp)
	}
	if xp := MatchXP("loss", 4, false); xp != 40 {
		t.Fatal(xp)
	}
	if xp := MatchXP("draw", 0, false); xp != 28 {
		t.Fatal(xp)
	}
	if xp := MatchXP("loss", 3, true); xp != 15 {
		t.Fatal(xp)
	}
}

func TestLevelCurve(t *testing.T) {
	for _, c := range []struct {
		level int
		xp    int64
	}{{2, 100}, {10, 1800}, {50, 34300}, {100, 131175}} {
		if got := LevelStart(c.level); got != c.xp {
			t.Fatalf("LevelStart(%d) = %d, want %d", c.level, got, c.xp)
		}
		if Level(c.xp) != c.level || Level(c.xp-1) != c.level-1 {
			t.Fatalf("Level around %d", c.xp)
		}
	}
	if Level(10_000_000) != MaxLevel || Level(0) != 1 {
		t.Fatal("bounds")
	}
}

func TestStreak(t *testing.T) {
	d := func(day int) time.Time { return time.Date(2026, 10, day, 0, 0, 0, 0, Istanbul) }
	var s Streak
	s = s.Play(d(1)).Play(d(1)).Play(d(2)).Play(d(3))
	if s.Current != 3 || s.Longest != 3 {
		t.Fatalf("%+v", s)
	}
	if s.Live(d(4)) != 3 || s.Live(d(5)) != 0 {
		t.Fatal("live streak")
	}
	s = s.Play(d(5))
	if s.Current != 1 || s.Longest != 3 {
		t.Fatalf("after a gap %+v", s)
	}
	if s.Play(d(2)) != s {
		t.Fatal("older day changed the streak")
	}
}

func TestPeriodsAndDay(t *testing.T) {
	// 22:30 UTC Sunday is Monday 01:30 in Istanbul.
	at := time.Date(2026, 10, 18, 22, 30, 0, 0, time.UTC)
	if s, e := Period("weekly", at); !s.Equal(time.Date(2026, 10, 18, 21, 0, 0, 0, time.UTC)) || e.Sub(s) != 7*24*time.Hour {
		t.Fatalf("week %v %v", s, e)
	}
	if s, _ := Period("daily", at); s.Day() != 19 {
		t.Fatalf("day %v", s)
	}
}

func TestBadgeCriteria(t *testing.T) {
	f := &MatchFacts{Wins: 1, Won: true, MatchType: "quick_duel", Format: "couple", Correct: 7, Rounds: 7, AnswerStreak: 7, PlayStreak: 3, CategoryCorrect: 49}
	for _, c := range []struct {
		c    Criteria
		want bool
	}{
		{Criteria{Type: "wins", Count: 1}, true},
		{Criteria{Type: "wins", Count: 10}, false},
		{Criteria{Type: "perfect", MatchType: "quick_duel", Rounds: 7}, true},
		{Criteria{Type: "perfect", MatchType: "ranked", Rounds: 10}, false},
		{Criteria{Type: "answer_streak", Count: 10}, false},
		{Criteria{Type: "play_streak", Days: 3}, true},
		{Criteria{Type: "category_correct", Count: 50}, false},
		{Criteria{Type: "team_match"}, false},
		{Criteria{Type: "couple_win"}, true},
		{Criteria{Type: "season_peak", League: "gold"}, false},
	} {
		if got := c.c.EarnedBy(f); got != c.want {
			t.Fatalf("%+v = %v", c.c, got)
		}
	}
	if BadgeXP(1) != 25 || BadgeXP(2) != 50 || BadgeXP(5) != 100 {
		t.Fatal("badge XP")
	}
}
