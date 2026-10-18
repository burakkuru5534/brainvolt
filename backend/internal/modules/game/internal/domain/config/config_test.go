package config

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain"
)

func ruleIDs(vs []Violation) []string {
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = v.RuleID
	}
	return out
}

// configRules evaluates R01–R11 only (the raw-axis property of §5.3).
func configRules(c Config) []string {
	var out []string
	for _, r := range Rules {
		if r.Violated(c) {
			out = append(out, r.ID)
		}
	}
	return out
}

// TestMatrixProperty: of the 11 520 raw combinations exactly 1 178 pass
// R01–R11 and exactly 5 of them are Ranked (mode-matrix §5.3); every valid
// configuration has a question count, window, bucket and queue key.
func TestMatrixProperty(t *testing.T) {
	total, valid, ranked := 0, 0, 0
	for _, mt := range domain.MatchTypes {
		for _, f := range domain.Formats {
			for _, tm := range domain.TeamModes {
				for _, ag := range domain.Aggregations {
					for _, cm := range domain.CategoryModes {
						for _, o := range domain.Origins {
							total++
							c := Config{MatchType: mt, Format: f, TeamMode: tm, Aggregation: ag, CategoryMode: cm, Origin: o}
							if len(configRules(c)) > 0 {
								continue
							}
							valid++
							d := Derive(c)
							if d.IsRated {
								ranked++
								if d.RatingBucket == domain.BucketNone || mt != domain.Ranked {
									t.Errorf("rated config without bucket: %+v", c)
								}
							}
							if d.AnswerWindowMS == 0 && cm != domain.CreatorQuiz {
								t.Errorf("no answer window: %+v", c)
							}
							if d.QuestionCount == 0 && mt != domain.SuddenDeath && cm != domain.CreatorQuiz {
								t.Errorf("no question count: %+v", c)
							}
							if o == domain.Matchmaking && QueueKey(c, "tr") == "" {
								t.Errorf("no queue key: %+v", c)
							}
						}
					}
				}
			}
		}
	}
	if total != 11_520 || valid != 1_178 || ranked != 5 {
		t.Fatalf("total %d valid %d ranked %d, want 11520 / 1178 / 5", total, valid, ranked)
	}
}

func cats(n int) []uuid.UUID {
	out := make([]uuid.UUID, n)
	for i := range out {
		out[i] = uuid.New()
	}
	return out
}

func set() *uuid.UUID { id := uuid.New(); return &id }

// TestValidConfigurations: MX-V01…V12 (context parts such as friendships or
// published quizzes are checked by the service).
func TestValidConfigurations(t *testing.T) {
	ranked1v1 := Config{MatchType: domain.Ranked, Format: domain.Format1v1, Origin: domain.Matchmaking}
	cases := map[string]Config{
		"MX-V01": ranked1v1,
		"MX-V02": {MatchType: domain.Ranked, Format: domain.Format3v3, TeamMode: domain.FastTeamRound, Aggregation: domain.Average, Origin: domain.Matchmaking},
		"MX-V03": {MatchType: domain.Ranked, Format: domain.FormatCouple, TeamMode: domain.FastTeamRound, Aggregation: domain.Average, Origin: domain.Matchmaking},
		"MX-V04": {MatchType: domain.QuickDuel, Format: domain.Format2v2, TeamMode: domain.SharedAnswer, CategoryMode: domain.BlindChallenge, Origin: domain.PrivateRoom},
		"MX-V05": {MatchType: domain.Marathon, Format: domain.Format4v4, TeamMode: domain.IndividualTurns, CategoryMode: domain.MixedBattle, Origin: domain.Challenge, CategoryIDs: cats(3)},
		"MX-V06": {MatchType: domain.SuddenDeath, Format: domain.Format1v1, CategoryMode: domain.CategoryMaster, Origin: domain.Matchmaking, CategoryIDs: cats(1)},
		"MX-V07": {MatchType: domain.QuickDuel, Format: domain.Format3v3, TeamMode: domain.FastTeamRound, Aggregation: domain.Majority, Origin: domain.Tournament},
		"MX-V08": {MatchType: domain.QuickDuel, Format: domain.Format1v1, CategoryMode: domain.CreatorQuiz, Origin: domain.Challenge, QuestionSetID: set(), QuizLength: 12, QuizWindowMS: 20_000},
		"MX-V09": {MatchType: domain.QuickDuel, Format: domain.FormatCouple, TeamMode: domain.SharedAnswer, CategoryMode: domain.CreatorQuiz, Origin: domain.PrivateRoom, QuestionSetID: set(), QuizLength: 5, QuizWindowMS: 10_000},
		"MX-V10": {MatchType: domain.SuddenDeath, Format: domain.Format2v2, TeamMode: domain.FastTeamRound, Aggregation: domain.FastestCorrect, CategoryMode: domain.RevengeMatch, Origin: domain.Revenge, CategoryIDs: cats(1)},
		"MX-V11": Transform(Config{Origin: domain.Rematch}, &ranked1v1),
		"MX-V12": {MatchType: domain.QuickDuel, Format: domain.Format4v4, TeamMode: domain.FastTeamRound, Aggregation: domain.Average, CategoryMode: domain.MixedBattle, Origin: domain.Matchmaking},
	}
	for id, c := range cases {
		if vs := Validate(c); len(vs) > 0 {
			t.Errorf("%s: %v", id, vs)
		}
	}
	v11 := cases["MX-V11"]
	d := Derive(v11)
	if v11.MatchType != domain.QuickDuel || !v11.RankedLength || d.IsRated || d.QuestionCount != 10 || !d.QuestionRules.MultipleChoiceOnly {
		t.Fatalf("T01 rematch of ranked = %+v / %+v", v11, d)
	}
	if d := Derive(cases["MX-V04"]); d.AnswerWindowMS != 20_000 {
		t.Fatalf("Shared Answer window = %d, want 15 s + 5 s (T05, MT08)", d.AnswerWindowMS)
	}
	if d := Derive(cases["MX-V08"]); d.AnswerWindowMS != 20_000 || d.QuestionCount != 12 {
		t.Fatalf("creator quiz derived = %+v", d)
	}
}

// TestInvalidConfigurations: MX-I01…I17, I21, I23 (rules checkable from the
// configuration; the rest need context and are tested with the service).
func TestInvalidConfigurations(t *testing.T) {
	cases := []struct {
		id   string
		c    Config
		want string
	}{
		{"MX-I01", Config{MatchType: domain.QuickDuel, Format: domain.Format1v1, TeamMode: domain.SharedAnswer, Origin: domain.PrivateRoom}, "R01"},
		{"MX-I02", Config{MatchType: domain.QuickDuel, Format: domain.Format2v2, Origin: domain.PrivateRoom}, "R01"},
		{"MX-I03", Config{MatchType: domain.QuickDuel, Format: domain.Format2v2, TeamMode: domain.SharedAnswer, Aggregation: domain.Average, Origin: domain.PrivateRoom}, "R02"},
		{"MX-I04", Config{MatchType: domain.QuickDuel, Format: domain.Format3v3, TeamMode: domain.FastTeamRound, Origin: domain.PrivateRoom}, "R02"},
		{"MX-I05", Config{MatchType: domain.Ranked, Format: domain.Format1v1, Origin: domain.Challenge}, "R03"},
		{"MX-I06", Config{MatchType: domain.Ranked, Format: domain.Format1v1, Origin: domain.PrivateRoom}, "R03"},
		{"MX-I07", Config{MatchType: domain.Ranked, Format: domain.Format1v1, CategoryMode: domain.BlindChallenge, Origin: domain.Matchmaking}, "R04"},
		{"MX-I08", Config{MatchType: domain.QuickDuel, Format: domain.Format2v2, TeamMode: domain.IndividualTurns, Origin: domain.Matchmaking}, "R05"},
		{"MX-I09", Config{MatchType: domain.QuickDuel, Format: domain.Format3v3, TeamMode: domain.FastTeamRound, Aggregation: domain.FastestCorrect, Origin: domain.Matchmaking}, "R06"},
		{"MX-I10", Config{MatchType: domain.Marathon, Format: domain.Format1v1, CategoryMode: domain.CreatorQuiz, Origin: domain.PrivateRoom, QuestionSetID: set(), QuizLength: 10, QuizWindowMS: 15_000}, "R07"},
		{"MX-I11", Config{MatchType: domain.QuickDuel, Format: domain.Format1v1, CategoryMode: domain.CreatorQuiz, Origin: domain.Matchmaking, QuestionSetID: set(), QuizLength: 10, QuizWindowMS: 15_000}, "R08"},
		{"MX-I12", Config{MatchType: domain.QuickDuel, Format: domain.Format1v1, CategoryMode: domain.RevengeMatch, Origin: domain.Challenge, CategoryIDs: cats(1)}, "R09"},
		{"MX-I13", Config{MatchType: domain.QuickDuel, Format: domain.Format1v1, Origin: domain.Revenge}, "R09"},
		{"MX-I14", Config{MatchType: domain.Ranked, Format: domain.Format1v1, Origin: domain.Tournament}, "R10"},
		{"MX-I15", Config{MatchType: domain.QuickDuel, Format: domain.Format1v1, CategoryMode: domain.DailyChallenge, Origin: domain.Challenge}, "R11"},
		{"MX-I16", Config{MatchType: domain.QuickDuel, Format: domain.Format1v1, CategoryMode: domain.CategoryMaster, Origin: domain.Matchmaking, CategoryIDs: cats(1), SubcategoryID: set()}, "R12"},
		{"MX-I17", Config{MatchType: domain.QuickDuel, Format: domain.Format1v1, CategoryMode: domain.MixedBattle, Origin: domain.Challenge, CategoryIDs: cats(2)}, "R13"},
		{"MX-I23", Config{MatchType: domain.QuickDuel, Format: domain.Format1v1, CategoryMode: domain.CreatorQuiz, Origin: domain.Challenge, QuestionSetID: set(), QuizLength: 10, QuizWindowMS: 12_000}, "R21"},
	}
	for _, tc := range cases {
		got := ruleIDs(Validate(tc.c))
		if !slices.Contains(got, tc.want) {
			t.Errorf("%s: violations %v, want %s", tc.id, got, tc.want)
		}
	}
	// MX-I14 breaks R03 too.
	if got := ruleIDs(Validate(cases[13].c)); !slices.Contains(got, "R03") {
		t.Errorf("MX-I14: %v, want R03 and R10", got)
	}
	if got := ruleIDs(Validate(Config{MatchType: "chess", Format: domain.Format1v1, Origin: domain.Challenge})); len(got) != 1 || got[0] != "AXIS" {
		t.Errorf("unknown axis value: %v", got)
	}
}

// TestTransformations: T02, T04, T06 make matchmade and revenge configs
// valid (MX-I04 / MX-I13 after transformation, TA09).
func TestTransformations(t *testing.T) {
	mm := Transform(Config{MatchType: domain.QuickDuel, Format: domain.Format3v3, Origin: domain.Matchmaking}, nil)
	if mm.TeamMode != domain.FastTeamRound || mm.Aggregation != domain.Average || len(Validate(mm)) > 0 {
		t.Fatalf("T04 = %+v %v", mm, Validate(mm))
	}
	room := Transform(Config{MatchType: domain.QuickDuel, Format: domain.Format3v3, TeamMode: domain.FastTeamRound, Origin: domain.PrivateRoom}, nil)
	if room.Aggregation != domain.Average {
		t.Fatalf("T06 default = %+v", room)
	}
	hostChoice := Transform(Config{MatchType: domain.QuickDuel, Format: domain.Format3v3, TeamMode: domain.FastTeamRound, Aggregation: domain.Majority, Origin: domain.PrivateRoom}, nil)
	if hostChoice.Aggregation != domain.Majority {
		t.Fatalf("host's aggregation kept (TA09) = %+v", hostChoice)
	}
	parent := Config{MatchType: domain.SuddenDeath, Format: domain.Format2v2, TeamMode: domain.FastTeamRound, Aggregation: domain.FastestCorrect, Origin: domain.PrivateRoom}
	home := cats(1)
	rev := Transform(Config{Origin: domain.Revenge, CategoryIDs: home}, &parent)
	if rev.CategoryMode != domain.RevengeMatch || rev.MatchType != domain.SuddenDeath || rev.Aggregation != domain.FastestCorrect ||
		rev.CategoryIDs[0] != home[0] || len(Validate(rev)) > 0 {
		t.Fatalf("T02 = %+v %v", rev, Validate(rev))
	}
	rankedParent := Config{MatchType: domain.Ranked, Format: domain.Format1v1, Origin: domain.Matchmaking}
	rev = Transform(Config{Origin: domain.Revenge, CategoryIDs: home}, &rankedParent)
	if rev.MatchType != domain.QuickDuel || !rev.RankedLength || Derive(rev).IsRated || len(Validate(rev)) > 0 {
		t.Fatalf("T01 revenge of ranked = %+v %v", rev, Validate(rev))
	}
}

func TestDerivedAndQueueKey(t *testing.T) {
	c := Config{MatchType: domain.Ranked, Format: domain.FormatCouple, TeamMode: domain.FastTeamRound, Aggregation: domain.Average, Origin: domain.Matchmaking}
	if d := Derive(c); !d.IsRated || d.RatingBucket != domain.BucketCouple || !d.SP || !d.LeaderboardPoints || d.QuestionCount != 10 {
		t.Fatalf("ranked couple = %+v", d)
	}
	if got := QueueKey(c, "tr"); got != "ranked:couple:couple:default:-:tr" {
		t.Fatalf("queue key = %q", got)
	}
	cm := Config{MatchType: domain.QuickDuel, Format: domain.Format1v1, CategoryMode: domain.CategoryMaster, Origin: domain.Matchmaking, CategoryIDs: cats(1)}
	if got := QueueKey(cm, "mixed"); got != "quick_duel:1v1:casual:category_master:"+cm.CategoryIDs[0].String()+":mixed" {
		t.Fatalf("category master queue key = %q", got)
	}
	if d := Derive(Config{MatchType: domain.QuickDuel, Format: domain.Format1v1, Origin: domain.Challenge}); d.LeaderboardPoints || d.SP {
		t.Fatalf("challenge awards = %+v", d)
	}
}
