package domain

import (
	"fmt"
	"time"
	_ "time/tzdata" // Europe/Istanbul on hosts without zoneinfo
)

// Istanbul is the time zone of every period boundary (D-029).
var Istanbul = mustZone("Europe/Istanbul")

func mustZone(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

// SeasonLength is 8 weeks (progression-formulas §9).
const SeasonLength = 8 * 7 * 24 * time.Hour

// WeekStart is Monday 00:00 Europe/Istanbul of t's week.
func WeekStart(t time.Time) time.Time {
	l := t.In(Istanbul)
	d := time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, Istanbul)
	return d.AddDate(0, 0, -((int(d.Weekday()) + 6) % 7))
}

// Period is one leaderboard period instance (redis-keys §6.1).
type Period struct {
	// ID is the key part: "d:2026-10-15", "w:2026-W42", "m:2026-10", "a".
	ID string
	// Start and End bound the period; zero for all-time.
	Start, End time.Time
	// TTL keeps the board after End (0 = no expiry).
	TTL time.Duration
}

// Board TTLs after the period ends.
const (
	DailyTTL   = 3 * 24 * time.Hour
	WeeklyTTL  = 14 * 24 * time.Hour
	MonthlyTTL = 40 * 24 * time.Hour
	SeasonTTL  = 30 * 24 * time.Hour
)

// Periods are the daily, weekly, monthly and all-time periods holding t.
func Periods(t time.Time) []Period {
	l := t.In(Istanbul)
	day := time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, Istanbul)
	week := WeekStart(t)
	y, w := week.ISOWeek()
	month := time.Date(l.Year(), l.Month(), 1, 0, 0, 0, 0, Istanbul)
	return []Period{
		{ID: "d:" + day.Format(time.DateOnly), Start: day, End: day.AddDate(0, 0, 1), TTL: DailyTTL},
		{ID: fmt.Sprintf("w:%d-W%02d", y, w), Start: week, End: week.AddDate(0, 0, 7), TTL: WeeklyTTL},
		{ID: "m:" + month.Format("2006-01"), Start: month, End: month.AddDate(0, 1, 0), TTL: MonthlyTTL},
		{ID: "a"},
	}
}

// SeasonPeriodID is a season board's period part.
func SeasonPeriodID(number int32) string { return fmt.Sprintf("s:%d", number) }

// BoardKey is a leaderboard's Redis key: scope is "global" or
// "country:<CC>", category "all" or "cat:<id>".
func BoardKey(periodID, entity, scope, category string) string {
	return "rank:lb:" + periodID + ":" + entity + ":" + scope + ":" + category
}

// Scopes are the global scope plus one per country.
func Scopes(countries []string) []string {
	out := []string{"global"}
	seen := map[string]bool{}
	for _, c := range countries {
		if c != "" && !seen[c] {
			seen[c] = true
			out = append(out, "country:"+c)
		}
	}
	return out
}
