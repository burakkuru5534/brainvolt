// Package config is the match configuration and its validation against
// the mode matrix (docs/game/mode-matrix.md): rules R01–R11 as data,
// transformations T01–T06 and the derived values of §4. Context rules that
// need data beyond the configuration (R12–R21) are checked by the game
// service through the Context interface.
package config

import (
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain/matchtype"
)

// Config is one match configuration (schema-game `matches` configuration
// columns + `config_extra`).
type Config struct {
	MatchType    domain.MatchType    `json:"match_type"`
	Format       domain.Format       `json:"format"`
	TeamMode     domain.TeamMode     `json:"team_answer_mode,omitempty"`
	Aggregation  domain.Aggregation  `json:"aggregation,omitempty"`
	CategoryMode domain.CategoryMode `json:"category_mode,omitempty"`
	Origin       domain.Origin       `json:"origin"`

	// RankedLength marks a rematch/revenge of a Ranked match: Quick Duel
	// rules with Ranked's length and question rules (T01).
	RankedLength bool `json:"ranked_length,omitempty"`
	// CategoryIDs: Category Master 1, Mixed Battle 3, Revenge home 1.
	CategoryIDs []uuid.UUID `json:"category_ids,omitempty"`
	// SubcategoryID narrows Category Master in rooms and challenges.
	SubcategoryID *uuid.UUID `json:"subcategory_id,omitempty"`
	// QuestionSetID is the Creator Quiz's question set.
	QuestionSetID *uuid.UUID `json:"question_set_id,omitempty"`
	// QuizLength and QuizWindowMS are the Creator Quiz's length (5–30) and
	// the author's answer window (10/15/20 s, R21).
	QuizLength   int `json:"quiz_length,omitempty"`
	QuizWindowMS int `json:"quiz_window_ms,omitempty"`
	// CreatorQuizID is the quiz a room or challenge plays; lobby fills the
	// question set, length, window, author and published flag from it.
	CreatorQuizID *uuid.UUID `json:"creator_quiz_id,omitempty"`
	QuizAuthorID  *uuid.UUID `json:"quiz_author_id,omitempty"`
	QuizPublished bool       `json:"quiz_published,omitempty"`
	// QuizLocale is the quiz's only language: every participant plays in it.
	QuizLocale string `json:"quiz_locale,omitempty"`
}

// Rule is one declarative configuration rule (mode-matrix §3.1).
type Rule struct {
	ID          string
	Description string
	// Violated reports whether c breaks the rule.
	Violated func(c Config) bool
}

func implies(a, b bool) bool { return !a || b }

// Rules are R01–R11 in evaluation order.
var Rules = []Rule{
	{"R01", "a team answer mode is set exactly when the format has more than one player per side (D-019/D-039)",
		func(c Config) bool { return (c.Format == domain.Format1v1) != (c.TeamMode == domain.NoTeamMode) }},
	{"R02", "an aggregation is set exactly for Fast Team Round (D-039)",
		func(c Config) bool {
			return (c.Aggregation != domain.NoAggregation) != (c.TeamMode == domain.FastTeamRound)
		}},
	{"R03", "Ranked is matchmaking only (D-038)",
		func(c Config) bool { return !implies(c.MatchType == domain.Ranked, c.Origin == domain.Matchmaking) }},
	{"R04", "Ranked has no category mode (D-041)",
		func(c Config) bool {
			return !implies(c.MatchType == domain.Ranked, c.CategoryMode == domain.NoCategoryMode)
		}},
	{"R05", "matchmade team formats use Fast Team Round (D-053)",
		func(c Config) bool {
			return !implies(c.Origin == domain.Matchmaking && c.Format != domain.Format1v1, c.TeamMode == domain.FastTeamRound)
		}},
	{"R06", "matchmade Fast Team Round uses the average rule (D-039)",
		func(c Config) bool {
			return !implies(c.TeamMode == domain.FastTeamRound && c.Origin == domain.Matchmaking, c.Aggregation == domain.Average)
		}},
	{"R07", "Creator Quiz is played with Quick Duel rules (D-044)",
		func(c Config) bool {
			return !implies(c.CategoryMode == domain.CreatorQuiz, c.MatchType == domain.QuickDuel)
		}},
	{"R08", "Creator Quiz is played in private rooms and challenges only (D-031/D-044)",
		func(c Config) bool {
			return !implies(c.CategoryMode == domain.CreatorQuiz, c.Origin == domain.PrivateRoom || c.Origin == domain.Challenge)
		}},
	{"R09", "Revenge Match exactly for revenge challenges (D-045)",
		func(c Config) bool { return (c.CategoryMode == domain.RevengeMatch) != (c.Origin == domain.Revenge) }},
	{"R10", "tournament matches are never Ranked (D-031)",
		func(c Config) bool { return !implies(c.Origin == domain.Tournament, c.MatchType != domain.Ranked) }},
	{"R11", "Daily Challenge is a solo run, not a match (D-040)",
		func(c Config) bool { return c.CategoryMode == domain.DailyChallenge }},
}

// Violation is one broken rule.
type Violation struct {
	RuleID      string `json:"rule_id"`
	Description string `json:"description"`
}

// Validate returns every violated configuration rule (R01–R11), plus the
// shape checks of the axes themselves (unknown values, missing category
// ids for category modes).
func Validate(c Config) []Violation {
	var out []Violation
	if v := checkValues(c); v != nil {
		return []Violation{*v}
	}
	for _, r := range Rules {
		if r.Violated(c) {
			out = append(out, Violation{RuleID: r.ID, Description: r.Description})
		}
	}
	out = append(out, checkModeParams(c)...)
	return out
}

func checkValues(c Config) *Violation {
	bad := func(axis string, v any) *Violation {
		return &Violation{RuleID: "AXIS", Description: fmt.Sprintf("unknown %s %q", axis, v)}
	}
	switch {
	case !slices.Contains(domain.MatchTypes, c.MatchType):
		return bad("match_type", c.MatchType)
	case !slices.Contains(domain.Formats, c.Format):
		return bad("format", c.Format)
	case !slices.Contains(domain.TeamModes, c.TeamMode):
		return bad("team_answer_mode", c.TeamMode)
	case !slices.Contains(domain.Aggregations, c.Aggregation):
		return bad("aggregation", c.Aggregation)
	case !slices.Contains(domain.CategoryModes, c.CategoryMode) && c.CategoryMode != domain.DailyChallenge:
		return bad("category_mode", c.CategoryMode)
	case !slices.Contains(domain.Origins, c.Origin):
		return bad("origin", c.Origin)
	}
	return nil
}

// Mode parameter rules (P-prefixed: shape of config_extra per mode).
func checkModeParams(c Config) []Violation {
	var out []Violation
	add := func(id, d string) { out = append(out, Violation{RuleID: id, Description: d}) }
	distinct := func(ids []uuid.UUID) bool {
		seen := map[uuid.UUID]bool{}
		for _, id := range ids {
			if id == uuid.Nil || seen[id] {
				return false
			}
			seen[id] = true
		}
		return true
	}
	switch c.CategoryMode {
	case domain.CategoryMaster:
		if len(c.CategoryIDs) != 1 || !distinct(c.CategoryIDs) {
			add("P01", "Category Master needs exactly one category")
		}
		if c.SubcategoryID != nil && c.Origin == domain.Matchmaking {
			add("R12", "matchmade Category Master uses a top-level category (D-042)")
		}
	case domain.MixedBattle:
		// Matchmaking: the server picks the 3 categories (R13), so they may
		// be absent from the request.
		serverPicks := c.Origin == domain.Matchmaking && len(c.CategoryIDs) == 0
		if !serverPicks && (len(c.CategoryIDs) != 3 || !distinct(c.CategoryIDs)) {
			add("R13", "Mixed Battle needs exactly 3 distinct categories (D-043)")
		}
	case domain.RevengeMatch:
		if len(c.CategoryIDs) != 1 || !distinct(c.CategoryIDs) {
			add("R15", "a revenge match needs the challenger's home category (D-045)")
		}
	case domain.CreatorQuiz:
		if c.QuestionSetID == nil {
			add("P02", "Creator Quiz needs the quiz's question set")
		}
		if c.QuizLength < 5 || c.QuizLength > 30 {
			add("P03", "Creator Quiz length is 5–30 questions (D-031)")
		}
		if !slices.Contains([]int{10_000, 15_000, 20_000}, c.QuizWindowMS) {
			add("R21", "Creator Quiz answer window is 10, 15 or 20 s (D-044)")
		}
	default:
		if len(c.CategoryIDs) > 0 || c.SubcategoryID != nil || c.QuestionSetID != nil || c.CreatorQuizID != nil {
			add("P04", "category ids, sub-category and question set belong to a category mode")
		}
	}
	if c.RankedLength && (c.MatchType != domain.QuickDuel || (c.Origin != domain.Rematch && c.Origin != domain.Revenge)) {
		add("P05", "Ranked length is only for rematches and revenges of Ranked matches (T01)")
	}
	return out
}

// Error renders violations for logs and error details.
func Error(vs []Violation) string {
	ids := make([]string, len(vs))
	for i, v := range vs {
		ids[i] = v.RuleID
	}
	return strings.Join(ids, ",")
}

// Transform applies T01–T06 (mode-matrix §3.3) before validation. parent
// is the parent match's configuration for rematch and revenge; nil
// otherwise.
func Transform(c Config, parent *Config) Config {
	if parent != nil && (c.Origin == domain.Rematch || c.Origin == domain.Revenge) {
		origin, cats := c.Origin, c.CategoryIDs
		if c.Origin == domain.Rematch {
			// T03: a rematch copies the parent configuration exactly.
			c = *parent
			c.Origin = origin
		} else {
			// T02: a revenge keeps format, team mode, aggregation, type and
			// window; its category mode is always Revenge Match.
			c.MatchType, c.Format, c.TeamMode, c.Aggregation = parent.MatchType, parent.Format, parent.TeamMode, parent.Aggregation
			c.RankedLength = parent.RankedLength
			c.CategoryMode, c.CategoryIDs = domain.RevengeMatch, cats
			c.SubcategoryID, c.QuestionSetID, c.QuizLength, c.QuizWindowMS = nil, nil, 0, 0
			c.CreatorQuizID, c.QuizAuthorID, c.QuizPublished, c.QuizLocale = nil, nil, false, ""
		}
		// T01: never rated outside matchmaking.
		if c.MatchType == domain.Ranked {
			c.MatchType, c.RankedLength = domain.QuickDuel, true
		}
	}
	if c.Origin == domain.Revenge && c.CategoryMode == domain.NoCategoryMode {
		c.CategoryMode = domain.RevengeMatch // T02 without a parent at hand
	}
	if c.Origin == domain.Matchmaking && c.Format.IsTeam() {
		c.TeamMode, c.Aggregation = domain.FastTeamRound, domain.Average // T04 (D-053)
	}
	if c.TeamMode == domain.FastTeamRound && c.Aggregation == domain.NoAggregation {
		c.Aggregation = domain.Average // T06 (D-039)
	}
	return c
}

// Derived are the values fixed by a valid configuration (mode-matrix §4).
type Derived struct {
	IsRated      bool
	RatingBucket domain.RatingBucket
	// QuestionCount is 0 for open-ended types (Sudden Death, cap 30).
	QuestionCount int
	// AnswerWindowMS includes the Shared Answer extension (T05).
	AnswerWindowMS int
	// QuestionRules are the eligibility rules of the selected questions.
	QuestionRules matchtype.QuestionRules
	// LeaderboardPoints: only matchmade matches feed leaderboards (D-049).
	LeaderboardPoints bool
	// SP: only Ranked gives Season Points (D-049).
	SP bool
}

// Derive returns the derived values of a valid configuration.
func Derive(c Config) Derived {
	t := matchtype.For(c.MatchType)
	if c.RankedLength {
		t = matchtype.RankedLengthQuickDuel()
	}
	d := Derived{
		IsRated:           c.MatchType == domain.Ranked,
		QuestionCount:     t.QuestionCount,
		AnswerWindowMS:    t.AnswerWindowMS,
		QuestionRules:     t.Questions,
		LeaderboardPoints: c.Origin == domain.Matchmaking,
		SP:                c.MatchType == domain.Ranked,
	}
	if c.CategoryMode == domain.CreatorQuiz {
		d.QuestionCount, d.AnswerWindowMS = c.QuizLength, c.QuizWindowMS
	}
	if c.TeamMode == domain.SharedAnswer {
		d.AnswerWindowMS += matchtype.SharedAnswerExtraMS // T05 (D-038)
	}
	if d.IsRated {
		switch {
		case c.Format == domain.FormatCouple:
			d.RatingBucket = domain.BucketCouple
		case c.Format.IsTeam():
			d.RatingBucket = domain.BucketTeam
		default:
			d.RatingBucket = domain.BucketSolo
		}
	}
	return d
}

// QueueKey is the matchmaking queue of a matchmade configuration
// (mode-matrix §3.4): type:format:bucket-or-casual:category-mode:category:locale-group.
func QueueKey(c Config, localeGroup string) string {
	bucket := string(Derive(c).RatingBucket)
	if bucket == "" {
		bucket = "casual"
	}
	mode := string(c.CategoryMode)
	if mode == "" {
		mode = "default"
	}
	cat := "-"
	if c.CategoryMode == domain.CategoryMaster && len(c.CategoryIDs) == 1 {
		cat = c.CategoryIDs[0].String()
	}
	return strings.Join([]string{string(c.MatchType), string(c.Format), bucket, mode, cat, localeGroup}, ":")
}
