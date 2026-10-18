// Package domain holds the game module's shared vocabulary (glossary.md,
// schema-game.md): the configuration axes of the mode matrix, lifecycle
// states and reasons, and the match/side/participant/round models. The
// rules live in sub-packages: config (mode matrix), scoring, matchtype,
// teammode, timing and engine — all pure, no I/O.
package domain

// MatchType is how a match is played and ends (match-types.md).
type MatchType string

// Match types.
const (
	QuickDuel   MatchType = "quick_duel"
	Ranked      MatchType = "ranked"
	SuddenDeath MatchType = "sudden_death"
	Marathon    MatchType = "marathon"
)

// MatchTypes in matrix order.
var MatchTypes = []MatchType{QuickDuel, Ranked, SuddenDeath, Marathon}

// Format is the side size.
type Format string

// Formats.
const (
	Format1v1    Format = "1v1"
	Format2v2    Format = "2v2"
	Format3v3    Format = "3v3"
	Format4v4    Format = "4v4"
	FormatCouple Format = "couple"
)

// Formats in matrix order.
var Formats = []Format{Format1v1, Format2v2, Format3v3, Format4v4, FormatCouple}

// SideSize is the number of participants per side (schema-game §2.1).
func (f Format) SideSize() int {
	switch f {
	case Format1v1:
		return 1
	case Format2v2, FormatCouple:
		return 2
	case Format3v3:
		return 3
	case Format4v4:
		return 4
	}
	return 0
}

// IsTeam reports whether a side has more than one participant.
func (f Format) IsTeam() bool { return f.SideSize() > 1 }

// TeamMode is the team answer mode (team-modes.md); empty for 1v1.
type TeamMode string

// Team answer modes.
const (
	NoTeamMode      TeamMode = ""
	SharedAnswer    TeamMode = "shared_answer"
	IndividualTurns TeamMode = "individual_turns"
	FastTeamRound   TeamMode = "fast_team_round"
)

// TeamModes in matrix order (including none).
var TeamModes = []TeamMode{NoTeamMode, SharedAnswer, IndividualTurns, FastTeamRound}

// Aggregation is the Fast Team Round side rule (team-modes.md §4).
type Aggregation string

// Aggregation rules.
const (
	NoAggregation  Aggregation = ""
	Average        Aggregation = "average"
	FastestCorrect Aggregation = "fastest_correct"
	Majority       Aggregation = "majority"
)

// Aggregations in matrix order (including none).
var Aggregations = []Aggregation{NoAggregation, Average, FastestCorrect, Majority}

// CategoryMode replaces the default question selection (category-modes.md).
type CategoryMode string

// Category modes. DailyChallenge is never a match (D-040, rule R11); it is
// listed so the validator can reject it by name.
const (
	NoCategoryMode CategoryMode = ""
	BlindChallenge CategoryMode = "blind_challenge"
	CategoryMaster CategoryMode = "category_master"
	MixedBattle    CategoryMode = "mixed_battle"
	CreatorQuiz    CategoryMode = "creator_quiz"
	RevengeMatch   CategoryMode = "revenge_match"
	DailyChallenge CategoryMode = "daily_challenge"
)

// CategoryModes in matrix order (including none; without DailyChallenge).
var CategoryModes = []CategoryMode{NoCategoryMode, BlindChallenge, CategoryMaster, MixedBattle, CreatorQuiz, RevengeMatch}

// Origin is how a match came about.
type Origin string

// Origins.
const (
	Matchmaking Origin = "matchmaking"
	Challenge   Origin = "challenge"
	PrivateRoom Origin = "private_room"
	Rematch     Origin = "rematch"
	Revenge     Origin = "revenge"
	Tournament  Origin = "tournament"
)

// Origins in matrix order.
var Origins = []Origin{Matchmaking, Challenge, PrivateRoom, Rematch, Revenge, Tournament}

// QuestionType mirrors content's question types.
type QuestionType string

// Question types.
const (
	MultipleChoice QuestionType = "multiple_choice"
	TrueFalse      QuestionType = "true_false"
)

// Status is the match lifecycle state (match-lifecycle.md §1).
type Status string

// Match states.
const (
	StatusWaiting    Status = "waiting"
	StatusReady      Status = "ready"
	StatusInProgress Status = "in_progress"
	StatusCompleted  Status = "completed"
	StatusCancelled  Status = "cancelled"
)

// Terminal reports whether s never changes again.
func (s Status) Terminal() bool { return s == StatusCompleted || s == StatusCancelled }

// Live reports whether the match holds its participants (one live match per
// user).
func (s Status) Live() bool { return !s.Terminal() }

// EndReason says why a match completed.
type EndReason string

// End reasons.
const (
	EndAllRoundsPlayed    EndReason = "all_rounds_played"
	EndSuddenDeathDecided EndReason = "sudden_death_decided"
	EndForfeit            EndReason = "forfeit"
	EndAllOpponentsLeft   EndReason = "all_opponents_left"
	EndTimeLimit          EndReason = "time_limit"
)

// CancelReason says why a match was cancelled.
type CancelReason string

// Cancel reasons (edge-cases.md §5).
const (
	CancelNotAllReady       CancelReason = "not_all_ready"
	CancelLeftBeforeStart   CancelReason = "participant_left_before_start"
	CancelAllSidesAbsent    CancelReason = "all_sides_absent"
	CancelServerError       CancelReason = "server_error"
	CancelAdmin             CancelReason = "admin"
	CancelIntegrityViolated CancelReason = "integrity_violation"
)

// ParticipantStatus is a participant's state inside a match
// (match-lifecycle.md §3).
type ParticipantStatus string

// Participant states.
const (
	PJoined       ParticipantStatus = "joined"
	PReady        ParticipantStatus = "ready"
	PPlaying      ParticipantStatus = "playing"
	PDisconnected ParticipantStatus = "disconnected"
	PLeft         ParticipantStatus = "left"
	PForfeited    ParticipantStatus = "forfeited"
	PFinished     ParticipantStatus = "finished"
)

// Active reports whether the participant still plays for its side
// (connected and not gone).
func (s ParticipantStatus) Active() bool { return s == PPlaying }

// Gone reports whether the participant has left the match for good.
func (s ParticipantStatus) Gone() bool { return s == PLeft || s == PForfeited }

// SideResult is a side's outcome.
type SideResult string

// Side results.
const (
	Win  SideResult = "win"
	Loss SideResult = "loss"
	Draw SideResult = "draw"
)

// RatingBucket says which rating a ranked match changes (mode-matrix §4).
type RatingBucket string

// Rating buckets.
const (
	BucketNone   RatingBucket = ""
	BucketSolo   RatingBucket = "solo"
	BucketTeam   RatingBucket = "team"
	BucketCouple RatingBucket = "couple"
)
