package service

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
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
	ID         uuid.UUID    `json:"id"`
	Kind       string       `json:"kind"`
	Status     string       `json:"status"`
	Challenger profile.Card `json:"challenger"`
	Target     profile.Card `json:"target"`
	// ChallengerSide / TargetSide are every player of each side (two for
	// couple challenges).
	ChallengerSide     []profile.Card `json:"challenger_side"`
	TargetSide         []profile.Card `json:"target_side"`
	ChallengerCoupleID *uuid.UUID     `json:"challenger_couple_id,omitempty"`
	TargetCoupleID     *uuid.UUID     `json:"target_couple_id,omitempty"`
	MatchConfig        game.Config    `json:"match_config"`
	MatchID            *uuid.UUID     `json:"match_id,omitempty"`
	// ParentMatchID is the match a revenge answers.
	ParentMatchID *uuid.UUID `json:"parent_match_id,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	ExpiresAt     time.Time  `json:"expires_at"`
}

// ChallengeSent / ChallengeAccepted are the event payloads.
type (
	// ChallengeSent is the payload of lobby.ChallengeSent.
	ChallengeSent struct {
		ChallengeID  uuid.UUID `json:"challenge_id"`
		ChallengerID uuid.UUID `json:"challenger_user_id"`
		TargetID     uuid.UUID `json:"target_user_id"`
		// Kind and TargetIDs (every target user) since D-089.
		Kind      string      `json:"kind"`
		TargetIDs []uuid.UUID `json:"target_user_ids"`
	}
	// ChallengeAccepted is the payload of lobby.ChallengeAccepted.
	ChallengeAccepted struct {
		ChallengeID  uuid.UUID `json:"challenge_id"`
		MatchID      uuid.UUID `json:"match_id"`
		ChallengerID uuid.UUID `json:"challenger_user_id"`
		AcceptedBy   uuid.UUID `json:"accepted_by_user_id"`
	}
)

// ChallengeTarget is who is challenged: a user (1v1) or a couple (Couple
// Contest, both partners play).
type ChallengeTarget struct {
	User   *uuid.UUID
	Couple *uuid.UUID
}

// sides are the players of a challenge: challenger side, target side.
func (s *Service) sides(ctx context.Context, r *sqlc.LobbyChallenge) (challenger, target []uuid.UUID, err error) {
	if r.ChallengerCoupleID == nil || r.TargetCoupleID == nil {
		return []uuid.UUID{r.ChallengerUserID}, []uuid.UUID{*r.TargetUserID}, nil
	}
	for _, c := range []struct {
		id  uuid.UUID
		out *[]uuid.UUID
	}{{*r.ChallengerCoupleID, &challenger}, {*r.TargetCoupleID, &target}} {
		info, err := s.d.Teams.CoupleInfo(ctx, c.id)
		if err != nil {
			return nil, nil, err
		}
		*c.out = info.Users[:]
	}
	// The challenger first on their side.
	if challenger[1] == r.ChallengerUserID {
		challenger[0], challenger[1] = challenger[1], challenger[0]
	}
	return challenger, target, nil
}

func (s *Service) view(ctx context.Context, rows ...sqlc.LobbyChallenge) ([]Challenge, error) {
	sides := make([][2][]uuid.UUID, len(rows))
	var users []uuid.UUID
	for i := range rows {
		a, b, err := s.sides(ctx, &rows[i])
		if err != nil {
			return nil, err
		}
		sides[i] = [2][]uuid.UUID{a, b}
		users = append(append(users, a...), b...)
	}
	cards, err := s.d.Profiles.Cards(ctx, users)
	if err != nil {
		return nil, err
	}
	pick := func(ids []uuid.UUID) []profile.Card {
		out := make([]profile.Card, len(ids))
		for i, id := range ids {
			out[i] = cards[id]
		}
		return out
	}
	out := make([]Challenge, len(rows))
	for i := range rows {
		r := &rows[i]
		var cfg game.Config
		if err := json.Unmarshal(r.MatchConfig, &cfg); err != nil {
			return nil, err
		}
		out[i] = Challenge{
			ID: r.ID, Kind: r.Kind, Status: r.Status, Challenger: cards[sides[i][0][0]], Target: cards[sides[i][1][0]],
			ChallengerSide: pick(sides[i][0]), TargetSide: pick(sides[i][1]),
			ChallengerCoupleID: r.ChallengerCoupleID, TargetCoupleID: r.TargetCoupleID,
			MatchConfig: cfg, MatchID: r.MatchID, ParentMatchID: r.ParentMatchID, CreatedAt: r.CreatedAt, ExpiresAt: r.ExpiresAt,
		}
	}
	return out, nil
}

// mayChallenge applies a target's challenges_from setting and blocks.
func (s *Service) mayChallenge(ctx context.Context, from uuid.UUID, targets []uuid.UUID) error {
	privacy, err := s.d.Profiles.PrivacyOf(ctx, targets)
	if err != nil {
		return err
	}
	for _, to := range targets {
		blocked, err := s.d.Social.IsBlocked(ctx, from, to)
		if err != nil {
			return err
		}
		if blocked {
			return apierror.New(apierror.Blocked)
		}
		switch privacy[to].ChallengesFrom {
		case "nobody":
			return apierror.New(apierror.ChallengesDisabled)
		case "friends":
			friends, err := s.d.Social.AreFriends(ctx, from, to)
			if err != nil {
				return err
			}
			if !friends {
				return apierror.New(apierror.ChallengesDisabled).With("reason", "friends_only")
			}
		}
	}
	return nil
}

// Challenge sends a direct challenge (schema-social §3.2): to a user in
// 1v1, or from the caller's couple to another couple in the couple format
// (team formats are played in rooms, D-071). The targets' challenges_from
// settings and blocks apply, at most one pending challenge per pair, 20
// per hour.
func (s *Service) Challenge(ctx context.Context, from uuid.UUID, to ChallengeTarget, cfg game.Config) (Challenge, error) {
	cfg.Origin = game.OriginChallenge
	quiz, err := s.withQuiz(ctx, from, &cfg)
	if err != nil {
		return Challenge{}, err
	}
	if err := checkConfig(cfg); err != nil {
		return Challenge{}, err
	}
	// Unpublished quizzes are played in friends-of-host rooms only (R14).
	if quiz != nil && !quiz.Published {
		return Challenge{}, apierror.New(apierror.ModeNotAllowed).With("rule_id", "R14")
	}
	if (to.User == nil) == (to.Couple == nil) {
		return Challenge{}, apierror.Validation(apierror.FieldError{Field: "target_user_id", Rule: "one_of"})
	}
	var targets []uuid.UUID
	var mine *uuid.UUID
	if to.User != nil {
		if cfg.Format != "1v1" {
			return Challenge{}, apierror.New(apierror.ModeNotAllowed).With("reason", "user_challenges_are_1v1")
		}
		if *to.User == from {
			return Challenge{}, apierror.Validation(apierror.FieldError{Field: "target_user_id", Rule: "not_self"})
		}
		if err := s.activeUser(ctx, *to.User); err != nil {
			return Challenge{}, err
		}
		targets = []uuid.UUID{*to.User}
	} else {
		if cfg.Format != "couple" {
			return Challenge{}, apierror.New(apierror.ModeNotAllowed).With("reason", "couple_challenges_are_couple_format")
		}
		own, ok, err := s.d.Teams.ActiveCoupleOf(ctx, from)
		if err != nil {
			return Challenge{}, err
		}
		if !ok {
			return Challenge{}, apierror.New(apierror.Forbidden).With("reason", "no_active_couple")
		}
		other, err := s.d.Teams.CoupleInfo(ctx, *to.Couple)
		if err != nil {
			return Challenge{}, err
		}
		if !other.Active || other.ID == own.ID {
			return Challenge{}, apierror.Validation(apierror.FieldError{Field: "target_couple_id", Rule: "active_other_couple"})
		}
		mine, targets = &own.ID, other.Users[:]
		partner := own.Users[0]
		if partner == from {
			partner = own.Users[1]
		}
		// The partner meets the other couple too: blocks count for them.
		for _, t := range targets {
			if blocked, err := s.d.Social.IsBlocked(ctx, partner, t); err != nil || blocked {
				if err != nil {
					return Challenge{}, err
				}
				return Challenge{}, apierror.New(apierror.Blocked)
			}
		}
	}
	if err := s.mayChallenge(ctx, from, targets); err != nil {
		return Challenge{}, err
	}
	body, err := json.Marshal(cfg)
	if err != nil {
		return Challenge{}, err
	}
	now := s.d.Clock.Now().UTC()
	id := ids.New()
	err = postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		var pending bool
		var err error
		if to.User != nil {
			pending, err = q.PendingChallengeBetween(ctx, sqlc.PendingChallengeBetweenParams{Now: now, A: from, B: *to.User})
		} else {
			pending, err = q.PendingCoupleChallengeBetween(ctx, sqlc.PendingCoupleChallengeBetweenParams{Now: now, A: *mine, B: *to.Couple})
		}
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
			ID: id, Kind: "direct", ChallengerUserID: from, ChallengerCoupleID: mine, TargetUserID: to.User, TargetCoupleID: to.Couple,
			MatchConfig: body, Now: now, ExpiresAt: now.Add(ChallengeTTL),
		}); err != nil {
			return err
		}
		return s.d.Events.PublishTx(ctx, tx, EventChallengeSent, now, ChallengeSent{ChallengeID: id, ChallengerID: from, TargetID: targets[0], Kind: "direct", TargetIDs: targets})
	})
	if err != nil {
		return Challenge{}, err
	}
	c, err := s.challenge(ctx, id)
	if err != nil {
		return Challenge{}, err
	}
	for _, t := range targets {
		s.send(ctx, t, realtime.Frame{T: "challenge.received", D: map[string]any{"challenge": c}})
	}
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
	var couple uuid.UUID
	if s.d.Teams != nil {
		c, ok, cerr := s.d.Teams.ActiveCoupleOf(ctx, user)
		if cerr != nil {
			return nil, cerr
		}
		if ok {
			couple = c.ID
		}
	}
	if box == "out" {
		rows, err = q.ListOutgoingChallenges(ctx, sqlc.ListOutgoingChallengesParams{UserID: user, CoupleID: couple, Now: now})
	} else {
		rows, err = q.ListIncomingChallenges(ctx, sqlc.ListIncomingChallengesParams{UserID: user, CoupleID: couple, Now: now})
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
		challengers, targets, err := s.sides(ctx, &row)
		if err != nil {
			return err
		}
		if (action == ActionCancel && !slices.Contains(challengers, user)) || (action != ActionCancel && !slices.Contains(targets, user)) {
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
			locales, err := s.d.Profiles.LocalesOf(ctx, append(slices.Clone(challengers), targets...))
			if err != nil {
				return err
			}
			var sides [2][]game.Member
			for i, side := range [][]uuid.UUID{challengers, targets} {
				for _, u := range side {
					sides[i] = append(sides[i], game.Member{UserID: u, Locale: quizLocale(&cfg, locales[u])})
				}
			}
			req := game.CreateRequest{Config: cfg, Sides: sides, OriginRef: &row.ID, Parent: row.ParentMatchID}
			if row.ChallengerCoupleID != nil {
				req.Couples = [2]*uuid.UUID{row.ChallengerCoupleID, row.TargetCoupleID}
			}
			c, err := s.d.Game.Start(ctx, req)
			if err != nil {
				return err
			}
			matchID = &c.Match.ID
			s.countPlay(ctx, &cfg)
			if err := s.d.Events.PublishTx(ctx, tx, EventChallengeAccepted, now, ChallengeAccepted{ChallengeID: id, MatchID: c.Match.ID, ChallengerID: row.ChallengerUserID, AcceptedBy: user}); err != nil {
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

// notifyChallenge tells every player a challenge's new status.
func (s *Service) notifyChallenge(ctx context.Context, row sqlc.LobbyChallenge, matchID *uuid.UUID) {
	d := map[string]any{"challenge_id": row.ID, "status": row.Status}
	if matchID != nil {
		d["match_id"] = *matchID
	}
	a, b, err := s.sides(ctx, &row)
	if err != nil {
		s.d.Logger.WarnContext(ctx, "lobby: reading challenge sides failed", "error", err)
		return
	}
	for _, u := range append(a, b...) {
		s.send(ctx, u, realtime.Frame{T: "challenge.updated", D: d})
	}
}

// FailChallengeOfMatch marks the challenge of a match cancelled before it
// started as failed (schema-social §3.1).
func (s *Service) FailChallengeOfMatch(ctx context.Context, match uuid.UUID) error {
	_, err := sqlc.New(s.d.DB).FailChallengeOfMatch(ctx, match)
	return err
}
