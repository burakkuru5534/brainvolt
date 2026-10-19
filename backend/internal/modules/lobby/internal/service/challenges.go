package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/lobby/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/realtime"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
)

// Direct challenge limits (D-027).
const (
	ChallengeTTL      = 24 * time.Hour
	ChallengesPerHour = 20
)

// Challenge statuses.
const (
	StatusPending   = "pending"
	StatusDeclined  = "declined"
	StatusCancelled = "cancelled"
	StatusExpired   = "expired"
	StatusStarted   = "started"
	StatusFailed    = "failed"
)

// Challenge is a challenge with both players' cards.
type Challenge struct {
	ID          uuid.UUID    `json:"id"`
	Kind        string       `json:"kind"`
	Status      string       `json:"status"`
	Challenger  profile.Card `json:"challenger"`
	Target      profile.Card `json:"target"`
	MatchConfig game.Config  `json:"match_config"`
	MatchID     *uuid.UUID   `json:"match_id,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	ExpiresAt   time.Time    `json:"expires_at"`
}

// ChallengeSent / ChallengeAccepted are the event payloads.
type (
	// ChallengeSent is the payload of lobby.ChallengeSent.
	ChallengeSent struct {
		ChallengeID  uuid.UUID `json:"challenge_id"`
		ChallengerID uuid.UUID `json:"challenger_user_id"`
		TargetID     uuid.UUID `json:"target_user_id"`
	}
	// ChallengeAccepted is the payload of lobby.ChallengeAccepted.
	ChallengeAccepted struct {
		ChallengeID uuid.UUID `json:"challenge_id"`
		MatchID     uuid.UUID `json:"match_id"`
	}
)

func (s *Service) view(ctx context.Context, rows ...sqlc.LobbyChallenge) ([]Challenge, error) {
	users := make([]uuid.UUID, 0, 2*len(rows))
	for i := range rows {
		users = append(users, rows[i].ChallengerUserID, *rows[i].TargetUserID)
	}
	cards, err := s.d.Profiles.Cards(ctx, users)
	if err != nil {
		return nil, err
	}
	out := make([]Challenge, len(rows))
	for i := range rows {
		r := &rows[i]
		var cfg game.Config
		if err := json.Unmarshal(r.MatchConfig, &cfg); err != nil {
			return nil, err
		}
		out[i] = Challenge{
			ID: r.ID, Kind: r.Kind, Status: r.Status, Challenger: cards[r.ChallengerUserID], Target: cards[*r.TargetUserID],
			MatchConfig: cfg, MatchID: r.MatchID, CreatedAt: r.CreatedAt, ExpiresAt: r.ExpiresAt,
		}
	}
	return out, nil
}

// Challenge sends a direct challenge (schema-social §3.2): the target's
// challenges_from setting and blocks apply, at most one pending challenge
// per pair, 20 per hour.
func (s *Service) Challenge(ctx context.Context, from, to uuid.UUID, cfg game.Config) (Challenge, error) {
	if from == to {
		return Challenge{}, apierror.Validation(apierror.FieldError{Field: "target_user_id", Rule: "not_self"})
	}
	cfg.Origin = game.OriginChallenge
	if err := checkConfig(cfg); err != nil {
		return Challenge{}, err
	}
	if err := s.activeUser(ctx, to); err != nil {
		return Challenge{}, err
	}
	blocked, err := s.d.Social.IsBlocked(ctx, from, to)
	if err != nil {
		return Challenge{}, err
	}
	if blocked {
		return Challenge{}, apierror.New(apierror.Blocked)
	}
	privacy, err := s.d.Profiles.PrivacyOf(ctx, []uuid.UUID{to})
	if err != nil {
		return Challenge{}, err
	}
	switch privacy[to].ChallengesFrom {
	case "nobody":
		return Challenge{}, apierror.New(apierror.ChallengesDisabled)
	case "friends":
		friends, err := s.d.Social.AreFriends(ctx, from, to)
		if err != nil {
			return Challenge{}, err
		}
		if !friends {
			return Challenge{}, apierror.New(apierror.ChallengesDisabled).With("reason", "friends_only")
		}
	}
	body, err := json.Marshal(cfg)
	if err != nil {
		return Challenge{}, err
	}
	now := s.d.Clock.Now().UTC()
	id := ids.New()
	err = postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		pending, err := q.PendingChallengeBetween(ctx, sqlc.PendingChallengeBetweenParams{Now: now, A: from, B: to})
		if err != nil {
			return err
		}
		if pending {
			return apierror.New(apierror.RequestPending)
		}
		sent, err := q.ChallengesSentSince(ctx, sqlc.ChallengesSentSinceParams{UserID: from, Since: now.Add(-time.Hour)})
		if err != nil {
			return err
		}
		if sent >= ChallengesPerHour {
			return apierror.New(apierror.RateLimited).WithRetryAfter(time.Hour)
		}
		if err := q.InsertChallenge(ctx, sqlc.InsertChallengeParams{
			ID: id, Kind: "direct", ChallengerUserID: from, TargetUserID: &to, MatchConfig: body, Now: now, ExpiresAt: now.Add(ChallengeTTL),
		}); err != nil {
			return err
		}
		return s.d.Events.PublishTx(ctx, tx, EventChallengeSent, now, ChallengeSent{ChallengeID: id, ChallengerID: from, TargetID: to})
	})
	if err != nil {
		return Challenge{}, err
	}
	c, err := s.challenge(ctx, id)
	if err != nil {
		return Challenge{}, err
	}
	s.send(ctx, to, realtime.Frame{T: "challenge.received", D: map[string]any{"challenge": c}})
	return c, nil
}

func (s *Service) challenge(ctx context.Context, id uuid.UUID) (Challenge, error) {
	row, err := sqlc.New(s.d.DB).GetChallenge(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Challenge{}, apierror.New(apierror.NotFound)
	}
	if err != nil {
		return Challenge{}, err
	}
	v, err := s.view(ctx, row)
	if err != nil {
		return Challenge{}, err
	}
	return v[0], nil
}

// Challenges lists pending challenges: box "in" (default) or "out".
func (s *Service) Challenges(ctx context.Context, user uuid.UUID, box string) ([]Challenge, error) {
	q := sqlc.New(s.d.DB)
	now := s.d.Clock.Now()
	var rows []sqlc.LobbyChallenge
	var err error
	if box == "out" {
		rows, err = q.ListOutgoingChallenges(ctx, sqlc.ListOutgoingChallengesParams{UserID: user, Now: now})
	} else {
		rows, err = q.ListIncomingChallenges(ctx, sqlc.ListIncomingChallengesParams{UserID: user, Now: now})
	}
	if err != nil {
		return nil, err
	}
	return s.view(ctx, rows...)
}

// Challenge actions.
const (
	ActionAccept  = "accept"
	ActionDecline = "decline"
	ActionCancel  = "cancel"
)

// RespondChallenge accepts or declines (target) or cancels (challenger) a
// pending challenge. Accepting asks game to create the match in
// `waiting`; both players receive match.created and challenge.updated.
func (s *Service) RespondChallenge(ctx context.Context, user, id uuid.UUID, action string) (Challenge, error) {
	now := s.d.Clock.Now().UTC()
	var row sqlc.LobbyChallenge
	var matchID *uuid.UUID
	err := postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		var err error
		row, err = q.GetChallengeForUpdate(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return apierror.New(apierror.NotFound)
		}
		if err != nil {
			return err
		}
		target := *row.TargetUserID
		if (action == ActionCancel && user != row.ChallengerUserID) || (action != ActionCancel && user != target) {
			return apierror.New(apierror.NotFound)
		}
		if row.Status != StatusPending {
			return apierror.New(apierror.MatchStateInvalid).With("reason", "challenge_"+row.Status)
		}
		if !row.ExpiresAt.After(now) {
			row.Status = StatusExpired
			if err := q.SetChallengeStatus(ctx, sqlc.SetChallengeStatusParams{ID: id, Status: StatusExpired}); err != nil {
				return err
			}
			return errExpired
		}
		status := map[string]string{ActionDecline: StatusDeclined, ActionCancel: StatusCancelled, ActionAccept: StatusStarted}[action]
		if action == ActionAccept {
			var cfg game.Config
			if err := json.Unmarshal(row.MatchConfig, &cfg); err != nil {
				return err
			}
			locales, err := s.d.Profiles.LocalesOf(ctx, []uuid.UUID{row.ChallengerUserID, target})
			if err != nil {
				return err
			}
			c, err := s.d.Game.Start(ctx, game.CreateRequest{
				Config:    cfg,
				Sides:     [2][]game.Member{{{UserID: row.ChallengerUserID, Locale: locales[row.ChallengerUserID]}}, {{UserID: target, Locale: locales[target]}}},
				OriginRef: &row.ID,
			})
			if err != nil {
				return err
			}
			matchID = &c.Match.ID
			if err := s.d.Events.PublishTx(ctx, tx, EventChallengeAccepted, now, ChallengeAccepted{ChallengeID: id, MatchID: c.Match.ID}); err != nil {
				return err
			}
		}
		row.Status = status
		return q.SetChallengeStatus(ctx, sqlc.SetChallengeStatusParams{ID: id, Status: status, RespondedAt: &now, MatchID: matchID})
	})
	if errors.Is(err, errExpired) {
		s.notifyChallenge(ctx, row, nil)
		return Challenge{}, apierror.New(apierror.ChallengeExpired)
	}
	if err != nil {
		return Challenge{}, err
	}
	s.notifyChallenge(ctx, row, matchID)
	return s.challenge(ctx, id)
}

var errExpired = errors.New("challenge expired")

// notifyChallenge tells both players a challenge's new status.
func (s *Service) notifyChallenge(ctx context.Context, row sqlc.LobbyChallenge, matchID *uuid.UUID) {
	d := map[string]any{"challenge_id": row.ID, "status": row.Status}
	if matchID != nil {
		d["match_id"] = *matchID
	}
	for _, u := range []uuid.UUID{row.ChallengerUserID, *row.TargetUserID} {
		s.send(ctx, u, realtime.Frame{T: "challenge.updated", D: d})
	}
}

// FailChallengeOfMatch marks the challenge of a match cancelled before it
// started as failed (schema-social §3.1).
func (s *Service) FailChallengeOfMatch(ctx context.Context, match uuid.UUID) error {
	_, err := sqlc.New(s.d.DB).FailChallengeOfMatch(ctx, match)
	return err
}
