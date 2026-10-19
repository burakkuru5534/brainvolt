// Package service implements the lobby module (schema-social.md §3):
// invite codes, direct challenges, private rooms and rematch, up to the
// creation of the match by game.
package service

import (
	"context"
	"log/slog"
	"slices"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/realtime"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/team"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/redis"
)

// Event names (schema-social.md §5).
const (
	EventChallengeSent     = "lobby.ChallengeSent"
	EventChallengeAccepted = "lobby.ChallengeAccepted"
	EventRoomClosed        = "lobby.RoomClosed"
)

// Users is what lobby needs from identity.
type Users interface {
	User(ctx context.Context, id uuid.UUID) (identity.User, error)
}

// Profiles is what lobby needs from profile.
type Profiles interface {
	Cards(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]profile.Card, error)
	PrivacyOf(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]profile.Privacy, error)
	LocalesOf(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error)
}

// Social is what lobby needs from social.
type Social interface {
	AreFriends(ctx context.Context, a, b uuid.UUID) (bool, error)
	IsBlocked(ctx context.Context, a, b uuid.UUID) (bool, error)
}

// Teams is what lobby needs from team (couples).
type Teams interface {
	ActiveCoupleOf(ctx context.Context, user uuid.UUID) (team.CoupleInfo, bool, error)
	CoupleInfo(ctx context.Context, id uuid.UUID) (team.CoupleInfo, error)
}

// Game is what lobby needs from game.
type Game interface {
	Start(ctx context.Context, r game.CreateRequest) (*game.Created, error)
	Summary(ctx context.Context, id uuid.UUID) (game.Summary, error)
	CategoryExists(ctx context.Context, id uuid.UUID) (bool, error)
}

// Quiz is a Creator Quiz as a room or challenge plays it.
type Quiz struct {
	SetID     uuid.UUID
	Length    int
	WindowMS  int
	Locale    string
	Author    uuid.UUID
	Published bool
}

// Quizzes is what lobby needs from creator (wired in app; lobby does not
// import creator).
type Quizzes interface {
	// Playable returns a quiz the user may play: their own or a published
	// one; NOT_FOUND otherwise.
	Playable(ctx context.Context, user, id uuid.UUID) (Quiz, error)
	// CountPlay counts a started match of the quiz.
	CountPlay(ctx context.Context, id uuid.UUID) error
}

// Realtime delivers lobby events and answers deferred intents.
type Realtime interface {
	Send(ctx context.Context, t realtime.Target, f realtime.Frame) error
}

// Deps are the service's collaborators.
type Deps struct {
	DB       *pgxpool.Pool
	Redis    *redis.Client
	Users    Users
	Profiles Profiles
	Social   Social
	Teams    Teams
	Game     Game
	Quizzes  Quizzes
	RT       Realtime
	Events   *events.Bus
	Clock    clock.Clock
	Logger   *slog.Logger
	// Random draws invite codes; nil uses crypto/rand.
	Random func(n int) int
}

// Service implements lobby's use cases.
type Service struct{ d Deps }

// New returns the service.
func New(d Deps) *Service {
	if d.Random == nil {
		d.Random = cryptoIntn
	}
	return &Service{d: d}
}

// send delivers a frame to a user's sessions; delivery problems are logged
// (the REST state stays the source of truth).
func (s *Service) send(ctx context.Context, user uuid.UUID, f realtime.Frame) {
	if err := s.d.RT.Send(ctx, realtime.Target{User: user}, f); err != nil {
		s.d.Logger.WarnContext(ctx, "lobby: delivering an event failed", "type", f.T, "error", err)
	}
}

// cards returns the public cards of users.
func (s *Service) cards(ctx context.Context, ids ...uuid.UUID) (map[uuid.UUID]profile.Card, error) {
	return s.d.Profiles.Cards(ctx, slices.Compact(ids))
}

// activeUser fails with NOT_FOUND for unknown, inactive or leaving users.
func (s *Service) activeUser(ctx context.Context, id uuid.UUID) error {
	u, err := s.d.Users.User(ctx, id)
	if err != nil {
		return err
	}
	if u.Status != identity.StatusActive || u.DeletionRequestedAt != nil {
		return apierror.New(apierror.NotFound)
	}
	return nil
}

// withQuiz fills a Creator Quiz configuration from its quiz (D-044): the
// client names the quiz, never its question set. It returns the quiz, or
// nil for other category modes.
func (s *Service) withQuiz(ctx context.Context, user uuid.UUID, cfg *game.Config) (*Quiz, error) {
	if cfg.CategoryMode != game.CategoryCreatorQuiz {
		return nil, nil
	}
	if cfg.CreatorQuizID == nil {
		return nil, apierror.Validation(apierror.FieldError{Field: "match_config.creator_quiz_id", Rule: "required"})
	}
	if s.d.Quizzes == nil {
		return nil, apierror.New(apierror.NotFound)
	}
	z, err := s.d.Quizzes.Playable(ctx, user, *cfg.CreatorQuizID)
	if err != nil {
		return nil, err
	}
	author := z.Author
	cfg.QuestionSetID, cfg.QuizLength, cfg.QuizWindowMS = &z.SetID, z.Length, z.WindowMS
	cfg.QuizAuthorID, cfg.QuizPublished, cfg.QuizLocale = &author, z.Published, z.Locale
	return &z, nil
}

// quizLocale is the language members play a configuration in: the quiz's
// for a Creator Quiz, their own otherwise.
func quizLocale(cfg *game.Config, own string) string {
	if cfg.QuizLocale != "" {
		return cfg.QuizLocale
	}
	return own
}

// checkConfig validates a configuration for an origin of lobby.
func checkConfig(c game.Config) error {
	if vs := game.ValidateConfig(c); len(vs) > 0 {
		return apierror.New(apierror.ModeNotAllowed).With("rule_id", vs[0].RuleID).With("violations", vs)
	}
	return nil
}
