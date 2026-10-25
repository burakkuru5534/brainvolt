// Package domain holds progression's pure rules (progression-formulas.md
// §5–8): match XP, the level curve, play streaks and badge criteria.
package domain

import (
	"time"
	_ "time/tzdata" // Europe/Istanbul on hosts without zoneinfo
)

// XP per match (progression-formulas §5.1).
const (
	XPBase    = 20
	XPCorrect = 5
	XPWin     = 15
	XPDraw    = 8
)

// MatchXP is a participant's XP for a completed match; a leaver or
// forfeiter only gets the correct answers they played (D-048).
func MatchXP(result string, correct int, left bool) int {
	if left {
		return XPCorrect * correct
	}
	xp := XPBase + XPCorrect*correct
	switch result {
	case "win":
		xp += XPWin
	case "draw":
		xp += XPDraw
	}
	return xp
}

// MaxLevel is the level cap.
const MaxLevel = 100

// LevelStart is the total XP needed to reach level l (§5.2: going from
// L to L+1 costs 100 + 25(L−1)).
func LevelStart(l int) int64 {
	n := int64(l - 1)
	return 100*n + 25*n*(n-1)/2
}

// Level is the level of an XP total.
func Level(xp int64) int {
	l := 1
	for l < MaxLevel && LevelStart(l+1) <= xp {
		l++
	}
	return l
}

// BadgeXP is the XP of earning a badge of a tier (§5.1: 25 / 50 / 100).
func BadgeXP(tier int) int {
	switch tier {
	case 1:
		return 25
	case 2:
		return 50
	}
	return 100
}

// Istanbul is the time zone of play days and quest periods (D-029).
var Istanbul = mustZone("Europe/Istanbul")

func mustZone(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

// Day is the Europe/Istanbul calendar day of t at 00:00.
func Day(t time.Time) time.Time {
	l := t.In(Istanbul)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, Istanbul)
}

// WeekStart is Monday 00:00 Europe/Istanbul of t's week.
func WeekStart(t time.Time) time.Time {
	d := Day(t)
	return d.AddDate(0, 0, -((int(d.Weekday()) + 6) % 7))
}

// Period is a quest period: the day or the week holding t.
func Period(cadence string, t time.Time) (start, end time.Time) {
	if cadence == "weekly" {
		start = WeekStart(t)
		return start, start.AddDate(0, 0, 7)
	}
	start = Day(t)
	return start, start.AddDate(0, 0, 1)
}

// MaxShields is how many streak shields a player can hold (D-105).
const MaxShields = 2

// Streak is a play streak (§6).
type Streak struct {
	Current, Longest int
	// LastDay is the last play day (zero when never played).
	LastDay time.Time
	// Shields each cover one missed day (D-105).
	Shields int
}

// Play records play on day (an Istanbul day at 00:00): the same day
// changes nothing, the next day extends the streak, a gap restarts it
// unless shields cover every missed day (they are used up).
func (s Streak) Play(day time.Time) Streak {
	switch {
	case !s.LastDay.IsZero() && sameDay(s.LastDay, day):
		return s
	case !s.LastDay.IsZero() && sameDay(s.LastDay.AddDate(0, 0, 1), day):
		s.Current++
	case !s.LastDay.IsZero() && day.Before(s.LastDay):
		return s // a late event for an older day
	default:
		if missed := s.missed(day); missed > 0 && missed <= s.Shields {
			s.Shields -= missed
			s.Current++
		} else {
			s.Current = 1
		}
	}
	s.LastDay = day
	s.Longest = max(s.Longest, s.Current)
	return s
}

// missed is the number of days between the last play day and day (0 when
// more than the shields could cover).
func (s Streak) missed(day time.Time) int {
	if s.LastDay.IsZero() {
		return 0
	}
	for k := 1; k <= MaxShields; k++ {
		if sameDay(s.LastDay.AddDate(0, 0, k+1), day) {
			return k
		}
	}
	return 0
}

// Live is the streak shown on day: it is lost once more whole days passed
// without play than the shields cover.
func (s Streak) Live(day time.Time) int {
	if s.LastDay.IsZero() || day.After(s.LastDay.AddDate(0, 0, 1+s.Shields)) {
		return 0
	}
	return s.Current
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

// Criteria is a badge's machine-checkable rule (seeded in migration 12).
type Criteria struct {
	Type      string `json:"type"`
	Count     int    `json:"count,omitempty"`
	Days      int    `json:"days,omitempty"`
	MatchType string `json:"match_type,omitempty"`
	Rounds    int    `json:"rounds,omitempty"`
	League    string `json:"league,omitempty"`
	Rank      int    `json:"rank,omitempty"`
}

// MatchFacts are what match badges are checked against for one player.
type MatchFacts struct {
	Wins            int
	Won             bool
	MatchType       string
	Format          string
	Correct         int
	Rounds          int
	AnswerStreak    int
	PlayStreak      int
	CategoryCorrect int
}

// EarnedBy reports whether a match fulfils a match badge's criteria
// (season badges are awarded at season close).
func (c Criteria) EarnedBy(f *MatchFacts) bool {
	switch c.Type {
	case "wins":
		return f.Wins >= c.Count
	case "perfect":
		return f.MatchType == c.MatchType && f.Rounds == c.Rounds && f.Correct == c.Rounds
	case "answer_streak":
		return f.AnswerStreak >= c.Count
	case "play_streak":
		return f.PlayStreak >= c.Days
	case "category_correct":
		return f.CategoryCorrect >= c.Count
	case "team_match":
		return f.Format != "1v1" && f.Format != "couple"
	case "couple_win":
		return f.Format == "couple" && f.Won
	}
	return false
}
