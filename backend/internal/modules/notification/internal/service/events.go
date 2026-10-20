package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
)

// Cards reads display names for the texts (profile).
type Cards interface {
	Cards(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]profile.Card, error)
}

// Handlers turns other modules' events into notices (backend-modules
// §3.2). Each handler is idempotent through the inbox de-duplication key.
type Handlers struct {
	Svc   *Service
	Cards Cards
}

func (h Handlers) name(ctx context.Context, user uuid.UUID) (string, error) {
	cards, err := h.Cards.Cards(ctx, []uuid.UUID{user})
	if err != nil {
		return "", err
	}
	return cards[user].DisplayName, nil
}

// notify tells to about something from from.
func (h Handlers) notify(ctx context.Context, to, from uuid.UUID, kind, link, dedupe string) error {
	name, err := h.name(ctx, from)
	if err != nil {
		return err
	}
	return h.Svc.Notify(ctx, Notice{User: to, Kind: kind, Params: map[string]string{"name": name}, DeepLink: link, Dedupe: dedupe})
}

// OnFriendRequestSent handles social.FriendRequestSent → friend.request.
func (h Handlers) OnFriendRequestSent(ctx context.Context, e events.Envelope) error {
	var p struct {
		RequestID uuid.UUID `json:"request_id"`
		From      uuid.UUID `json:"from_user_id"`
		To        uuid.UUID `json:"to_user_id"`
	}
	if err := e.Decode(&p); err != nil {
		return err
	}
	return h.notify(ctx, p.To, p.From, "friend.request", "/friends", "friend.request:"+p.From.String())
}

// OnFriendshipCreated handles social.FriendshipCreated → friend.accepted to the
// requester.
func (h Handlers) OnFriendshipCreated(ctx context.Context, e events.Envelope) error {
	var p struct {
		Requester uuid.UUID `json:"user_id"`
		Friend    uuid.UUID `json:"friend_id"`
	}
	if err := e.Decode(&p); err != nil {
		return err
	}
	return h.notify(ctx, p.Requester, p.Friend, "friend.accepted", "/friends", "friend.accepted:"+p.Friend.String())
}

// OnChallengeSent handles lobby.ChallengeSent → challenge.received.
func (h Handlers) OnChallengeSent(ctx context.Context, e events.Envelope) error {
	var p struct {
		ChallengeID uuid.UUID   `json:"challenge_id"`
		Kind        string      `json:"kind"`
		From        uuid.UUID   `json:"challenger_user_id"`
		To          uuid.UUID   `json:"target_user_id"`
		Targets     []uuid.UUID `json:"target_user_ids"`
	}
	if err := e.Decode(&p); err != nil {
		return err
	}
	kind := "challenge.received"
	if p.Kind == "revenge" {
		kind = "challenge.revenge"
	}
	if len(p.Targets) == 0 {
		p.Targets = []uuid.UUID{p.To} // published before D-089 named every target
	}
	for _, to := range p.Targets {
		if err := h.notify(ctx, to, p.From, kind, "/challenges", "challenge:"+p.ChallengeID.String()); err != nil {
			return err
		}
	}
	return nil
}

// OnChallengeAccepted handles lobby.ChallengeAccepted → challenge.accepted to
// the challenger, opening the match.
func (h Handlers) OnChallengeAccepted(ctx context.Context, e events.Envelope) error {
	var p struct {
		ChallengeID uuid.UUID `json:"challenge_id"`
		MatchID     uuid.UUID `json:"match_id"`
		Challenger  uuid.UUID `json:"challenger_user_id"`
		AcceptedBy  uuid.UUID `json:"accepted_by_user_id"`
	}
	if err := e.Decode(&p); err != nil {
		return err
	}
	if p.Challenger == uuid.Nil || p.AcceptedBy == uuid.Nil {
		return nil // published before Phase 9 carried the users
	}
	return h.notify(ctx, p.Challenger, p.AcceptedBy, "challenge.accepted", "/match/"+p.MatchID.String(), "challenge.accepted:"+p.ChallengeID.String())
}

// OnCoupleRequested handles team.CoupleRequested → couple.request to the partner.
func (h Handlers) OnCoupleRequested(ctx context.Context, e events.Envelope) error {
	var p struct {
		CoupleID uuid.UUID    `json:"couple_id"`
		Users    [2]uuid.UUID `json:"user_ids"`
	}
	if err := e.Decode(&p); err != nil {
		return err
	}
	return h.notify(ctx, p.Users[1], p.Users[0], "couple.request", "/teams", "couple:"+p.CoupleID.String())
}

// OnSanctionApplied handles moderation.SanctionApplied → moderation.sanction
// (warnings included; the sanctions screen explains).
func (h Handlers) OnSanctionApplied(ctx context.Context, e events.Envelope) error {
	var p struct {
		SanctionID uuid.UUID `json:"sanction_id"`
		User       uuid.UUID `json:"user_id"`
	}
	if err := e.Decode(&p); err != nil {
		return err
	}
	return h.Svc.Notify(ctx, Notice{User: p.User, Kind: "moderation.sanction", Params: map[string]string{},
		DeepLink: "/sanctions", Dedupe: "sanction:" + p.SanctionID.String()})
}

// ChatNoticeEvery is how often one chat channel may notify a member: a
// burst of messages becomes one notice (D-089).
const ChatNoticeEvery = 10 * time.Minute

// OnChatMessage handles chat.MessageSent → chat.team / chat.couple to the
// other members, opening the chat.
func (h Handlers) OnChatMessage(ctx context.Context, e events.Envelope) error {
	var p struct {
		ChannelID  uuid.UUID   `json:"channel_id"`
		Kind       string      `json:"channel_kind"`
		TeamID     *uuid.UUID  `json:"team_id"`
		CoupleID   *uuid.UUID  `json:"couple_id"`
		Sender     uuid.UUID   `json:"sender_user_id"`
		Recipients []uuid.UUID `json:"recipient_user_ids"`
	}
	if err := e.Decode(&p); err != nil {
		return err
	}
	var kind, link string
	switch {
	case p.Kind == "team" && p.TeamID != nil:
		kind, link = "chat.team", "/chat/team/"+p.TeamID.String()
	case p.Kind == "couple" && p.CoupleID != nil:
		kind, link = "chat.couple", "/chat/couple/"+p.CoupleID.String()
	default:
		return nil
	}
	bucket := e.OccurredAt.Unix() / int64(ChatNoticeEvery/time.Second)
	for _, to := range p.Recipients {
		dedupe := fmt.Sprintf("chat:%s:%d", p.ChannelID, bucket)
		if err := h.notify(ctx, to, p.Sender, kind, link, dedupe); err != nil {
			return err
		}
	}
	return nil
}
