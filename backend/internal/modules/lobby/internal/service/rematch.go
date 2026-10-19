package service

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/realtime"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/redis"
)

// RematchWindow is how long after a match "play again" works (D-027).
const RematchWindow = 30 * time.Second

func rematchKey(match uuid.UUID) string { return redis.Key("lobby", "rematch", match.String()) }
func rematchStartedKey(match uuid.UUID) string {
	return redis.Key("lobby", "rematch", match.String(), "started")
}

// RespondRematch records a player's answer to "play again" on the
// results screen. Everyone gets rematch.offer with who accepted; once
// all participants accepted within 30 s of the end, game creates the
// rematch (same configuration, ranked converts, D-053) and everyone gets
// rematch.result with the new match. One decline ends the offer. Private
// room matches play again through the room.
func (s *Service) RespondRematch(ctx context.Context, user, match uuid.UUID, accept bool) error {
	sum, err := s.d.Game.Summary(ctx, match)
	if err != nil {
		return err
	}
	var users []uuid.UUID
	for _, p := range sum.Participants {
		users = append(users, p.UserID)
	}
	if !slices.Contains(users, user) {
		return apierror.New(apierror.NotAParticipant)
	}
	if sum.Status != "completed" || sum.EndedAt == nil {
		return apierror.New(apierror.MatchStateInvalid).With("reason", "match_not_completed")
	}
	if sum.Origin == game.OriginPrivateRoom {
		return apierror.New(apierror.MatchStateInvalid).With("reason", "use_room")
	}
	expires := sum.EndedAt.Add(RematchWindow)
	if !s.d.Clock.Now().Before(expires) {
		return apierror.New(apierror.ChallengeExpired)
	}
	if exists, _ := s.d.Redis.Exists(ctx, rematchStartedKey(match)).Result(); exists > 0 {
		return apierror.New(apierror.MatchStateInvalid).With("reason", "rematch_started")
	}
	key := rematchKey(match)
	if !accept {
		s.d.Redis.Del(ctx, key)
		s.tell(ctx, users, realtime.Frame{T: "rematch.result", D: map[string]any{"match_id": match, "accepted": false, "declined_by": user}})
		return nil
	}
	pipe := s.d.Redis.TxPipeline()
	pipe.HSet(ctx, key, user.String(), "1")
	pipe.PExpireAt(ctx, key, expires)
	all := pipe.HGetAll(ctx, key)
	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}
	accepted := make([]uuid.UUID, 0, len(users))
	for _, u := range users {
		if all.Val()[u.String()] == "1" {
			accepted = append(accepted, u)
		}
	}
	s.tell(ctx, users, realtime.Frame{T: "rematch.offer", D: map[string]any{
		"match_id": match, "accepted_user_ids": accepted, "expires_at": expires.UnixMilli(),
	}})
	if len(accepted) < len(users) {
		return nil
	}
	if ok, err := s.d.Redis.SetNX(ctx, rematchStartedKey(match), "1", RematchWindow).Result(); err != nil || !ok {
		return err
	}
	s.d.Redis.Del(ctx, key)
	var sides [2][]game.Member
	for _, p := range sum.Participants {
		if len(sides[p.Side-1]) < p.Seat {
			sides[p.Side-1] = append(sides[p.Side-1], make([]game.Member, p.Seat-len(sides[p.Side-1]))...)
		}
		sides[p.Side-1][p.Seat-1] = game.Member{UserID: p.UserID, Locale: p.Locale}
	}
	c, err := s.d.Game.Start(ctx, game.CreateRequest{Config: game.Config{Origin: game.OriginRematch}, Sides: sides, Parent: &match})
	if err != nil {
		reason := apierror.Internal
		var e *apierror.Error
		if errors.As(err, &e) {
			reason = e.Code
		}
		s.tell(ctx, users, realtime.Frame{T: "rematch.result", D: map[string]any{"match_id": match, "accepted": false, "reason": reason}})
		return err
	}
	s.tell(ctx, users, realtime.Frame{T: "rematch.result", D: map[string]any{"match_id": match, "accepted": true, "new_match_id": c.Match.ID}})
	return nil
}

func (s *Service) tell(ctx context.Context, users []uuid.UUID, f realtime.Frame) {
	for _, u := range users {
		s.send(ctx, u, f)
	}
}
