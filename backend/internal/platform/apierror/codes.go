package apierror

import "net/http"

// Code is a stable error code from docs/api/errors.md. Codes are never
// renamed or reused; clients switch on them, never on the message.
type Code string

// General.
const (
	ValidationFailed      Code = "VALIDATION_FAILED"
	MalformedRequest      Code = "MALFORMED_REQUEST"
	PayloadTooLarge       Code = "PAYLOAD_TOO_LARGE"
	NotFound              Code = "NOT_FOUND"
	MethodNotAllowed      Code = "METHOD_NOT_ALLOWED"
	Forbidden             Code = "FORBIDDEN"
	RateLimited           Code = "RATE_LIMITED"
	IdempotencyKeyReused  Code = "IDEMPOTENCY_KEY_REUSED"
	IdempotencyInProgress Code = "IDEMPOTENT_REQUEST_IN_PROGRESS"
	VersionConflict       Code = "VERSION_CONFLICT"
	AppVersionUnsupported Code = "APP_VERSION_UNSUPPORTED"
	Internal              Code = "INTERNAL"
	Unavailable           Code = "UNAVAILABLE"
)

// Auth & account.
const (
	Unauthenticated         Code = "UNAUTHENTICATED"
	TokenExpired            Code = "TOKEN_EXPIRED"
	AgeConfirmationRequired Code = "AGE_CONFIRMATION_REQUIRED"
	AccountUpgradeRequired  Code = "ACCOUNT_UPGRADE_REQUIRED"
	AccountSuspended        Code = "ACCOUNT_SUSPENDED"
	AccountDeletionPending  Code = "ACCOUNT_DELETION_PENDING"
	Sanctioned              Code = "SANCTIONED"
	UsernameTaken           Code = "USERNAME_TAKEN"
	ChangeTooSoon           Code = "CHANGE_TOO_SOON"
	TextRejected            Code = "TEXT_REJECTED"
)

// Social & lobby.
const (
	Blocked            Code = "BLOCKED"
	FriendLimitReached Code = "FRIEND_LIMIT_REACHED"
	AlreadyFriends     Code = "ALREADY_FRIENDS"
	RequestPending     Code = "REQUEST_PENDING"
	ChallengesDisabled Code = "CHALLENGES_DISABLED"
	InviteCodeInvalid  Code = "INVITE_CODE_INVALID"
	InviteCodeExpired  Code = "INVITE_CODE_EXPIRED"
	RoomFull           Code = "ROOM_FULL"
	RoomClosed         Code = "ROOM_CLOSED"
	RoomFriendsOnly    Code = "ROOM_FRIENDS_ONLY"
	ChallengeExpired   Code = "CHALLENGE_EXPIRED"
	RevengeNotAllowed  Code = "REVENGE_NOT_ALLOWED"
	TeamLimitReached   Code = "TEAM_LIMIT_REACHED"
	TeamFull           Code = "TEAM_FULL"
	NotCaptain         Code = "NOT_CAPTAIN"
	CoupleExists       Code = "COUPLE_EXISTS"
	CoupleCooldown     Code = "COUPLE_COOLDOWN"
)

// Matchmaking & match.
const (
	AlreadyInMatch      Code = "ALREADY_IN_MATCH"
	AlreadyQueued       Code = "ALREADY_QUEUED"
	QueueCooldown       Code = "QUEUE_COOLDOWN"
	ModeNotAllowed      Code = "MODE_NOT_ALLOWED"
	NotEnoughQuestions  Code = "NOT_ENOUGH_QUESTIONS"
	NoActiveSeason      Code = "NO_ACTIVE_SEASON"
	MatchNotFound       Code = "MATCH_NOT_FOUND"
	NotAParticipant     Code = "NOT_A_PARTICIPANT"
	MatchStateInvalid   Code = "MATCH_STATE_INVALID"
	AnswerLate          Code = "ANSWER_LATE"
	AnswerDuplicate     Code = "ANSWER_DUPLICATE"
	AnswerInvalidOption Code = "ANSWER_INVALID_OPTION"
	NotYourTurn         Code = "NOT_YOUR_TURN"
	PickNotAllowed      Code = "PICK_NOT_ALLOWED"
	SeatTakenElsewhere  Code = "SEAT_TAKEN_ELSEWHERE"
)

// Daily Challenge.
const (
	DailyAttemptUsed       Code = "DAILY_ATTEMPT_USED"
	DailyAttemptInProgress Code = "DAILY_ATTEMPT_IN_PROGRESS"
	DailyNotAvailable      Code = "DAILY_NOT_AVAILABLE"
)

// Content, creator & moderation.
const (
	QuestionSourceRequired Code = "QUESTION_SOURCE_REQUIRED"
	QuestionInUse          Code = "QUESTION_IN_USE"
	QuestionStateInvalid   Code = "QUESTION_STATE_INVALID"
	QuestionDuplicate      Code = "QUESTION_DUPLICATE"
	QuizLimitReached       Code = "QUIZ_LIMIT_REACHED"
	QuizSizeInvalid        Code = "QUIZ_SIZE_INVALID"
	ReportLimitReached     Code = "REPORT_LIMIT_REACHED"
	AlreadyReported        Code = "ALREADY_REPORTED"
	AppealExists           Code = "APPEAL_EXISTS"
	SelfReviewForbidden    Code = "SELF_REVIEW_FORBIDDEN"
)

// Commerce.
const (
	PurchaseInvalid          Code = "PURCHASE_INVALID"
	PurchaseAlreadyProcessed Code = "PURCHASE_ALREADY_PROCESSED"
	AdRewardLimit            Code = "AD_REWARD_LIMIT"
)

// Tournament.
const (
	TournamentRegistrationClosed Code = "TOURNAMENT_REGISTRATION_CLOSED"
	TournamentFull               Code = "TOURNAMENT_FULL"
	TournamentRequirementsNotMet Code = "TOURNAMENT_REQUIREMENTS_NOT_MET"
)

// wsOnly marks codes that exist only on the WebSocket (no HTTP status).
const wsOnly = 0

// statuses is the HTTP column of the errors.md registry. A test keeps it
// identical to the document.
var statuses = map[Code]int{
	ValidationFailed:      http.StatusBadRequest,
	MalformedRequest:      http.StatusBadRequest,
	PayloadTooLarge:       http.StatusRequestEntityTooLarge,
	NotFound:              http.StatusNotFound,
	MethodNotAllowed:      http.StatusMethodNotAllowed,
	Forbidden:             http.StatusForbidden,
	RateLimited:           http.StatusTooManyRequests,
	IdempotencyKeyReused:  http.StatusConflict,
	IdempotencyInProgress: http.StatusConflict,
	VersionConflict:       http.StatusConflict,
	AppVersionUnsupported: http.StatusUpgradeRequired,
	Internal:              http.StatusInternalServerError,
	Unavailable:           http.StatusServiceUnavailable,

	Unauthenticated:         http.StatusUnauthorized,
	TokenExpired:            http.StatusUnauthorized,
	AgeConfirmationRequired: http.StatusBadRequest,
	AccountUpgradeRequired:  http.StatusForbidden,
	AccountSuspended:        http.StatusForbidden,
	AccountDeletionPending:  http.StatusForbidden,
	Sanctioned:              http.StatusForbidden,
	UsernameTaken:           http.StatusConflict,
	ChangeTooSoon:           http.StatusConflict,
	TextRejected:            http.StatusUnprocessableEntity,

	Blocked:            http.StatusForbidden,
	FriendLimitReached: http.StatusConflict,
	AlreadyFriends:     http.StatusConflict,
	RequestPending:     http.StatusConflict,
	ChallengesDisabled: http.StatusForbidden,
	InviteCodeInvalid:  http.StatusNotFound,
	InviteCodeExpired:  http.StatusGone,
	RoomFull:           http.StatusConflict,
	RoomClosed:         http.StatusGone,
	RoomFriendsOnly:    http.StatusForbidden,
	ChallengeExpired:   http.StatusGone,
	RevengeNotAllowed:  http.StatusForbidden,
	TeamLimitReached:   http.StatusConflict,
	TeamFull:           http.StatusConflict,
	NotCaptain:         http.StatusForbidden,
	CoupleExists:       http.StatusConflict,
	CoupleCooldown:     http.StatusConflict,

	AlreadyInMatch:      http.StatusConflict,
	AlreadyQueued:       http.StatusConflict,
	QueueCooldown:       http.StatusConflict,
	ModeNotAllowed:      http.StatusUnprocessableEntity,
	NotEnoughQuestions:  http.StatusUnprocessableEntity,
	NoActiveSeason:      http.StatusServiceUnavailable,
	MatchNotFound:       http.StatusNotFound,
	NotAParticipant:     http.StatusForbidden,
	MatchStateInvalid:   http.StatusConflict,
	AnswerLate:          wsOnly,
	AnswerDuplicate:     wsOnly,
	AnswerInvalidOption: wsOnly,
	NotYourTurn:         wsOnly,
	PickNotAllowed:      wsOnly,
	SeatTakenElsewhere:  wsOnly,

	DailyAttemptUsed:       http.StatusConflict,
	DailyAttemptInProgress: http.StatusConflict,
	DailyNotAvailable:      http.StatusServiceUnavailable,

	QuestionSourceRequired: http.StatusUnprocessableEntity,
	QuestionInUse:          http.StatusConflict,
	QuestionStateInvalid:   http.StatusConflict,
	QuestionDuplicate:      http.StatusConflict,
	QuizLimitReached:       http.StatusConflict,
	QuizSizeInvalid:        http.StatusUnprocessableEntity,
	ReportLimitReached:     http.StatusTooManyRequests,
	AlreadyReported:        http.StatusConflict,
	AppealExists:           http.StatusConflict,
	SelfReviewForbidden:    http.StatusForbidden,

	PurchaseInvalid:          http.StatusUnprocessableEntity,
	PurchaseAlreadyProcessed: http.StatusConflict,
	AdRewardLimit:            http.StatusTooManyRequests,

	TournamentRegistrationClosed: http.StatusConflict,
	TournamentFull:               http.StatusConflict,
	TournamentRequirementsNotMet: http.StatusForbidden,
}

// HTTPStatus returns the code's HTTP status. WebSocket-only and unknown
// codes return 500, because sending them over REST is a server bug.
func (c Code) HTTPStatus() int {
	if s := statuses[c]; s != wsOnly {
		return s
	}
	return http.StatusInternalServerError
}

// Known reports whether c is in the registry.
func (c Code) Known() bool {
	_, ok := statuses[c]
	return ok
}
