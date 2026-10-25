// Package service implements the social module (schema-social.md §1–§2):
// friend requests, friendships and blocks, with the limits of D-027.
package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/social/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
)

// Limits (D-027).
const (
	FriendCap       = 300
	RequestsPerDay  = 50
	RequestExpiry   = 30 * 24 * time.Hour
	friendsPageSize = 100
)

// Event names (schema-social.md §5).
const (
	EventFriendRequestSent = "social.FriendRequestSent"
	EventFriendshipCreated = "social.FriendshipCreated"
	EventFriendshipRemoved = "social.FriendshipRemoved"
	EventUserBlocked       = "social.UserBlocked"
)

// Request vias.
const (
	ViaSearch     = "search"
	ViaInviteCode = "invite_code"
	ViaPostMatch  = "post_match"
	ViaProfile    = "profile"
)

// Users is what social needs from identity.
type Users interface {
	User(ctx context.Context, id uuid.UUID) (identity.User, error)
}

// Profiles is what social needs from profile.
type Profiles interface {
	Cards(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]profile.Card, error)
	PrivacyOf(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]profile.Privacy, error)
}

// Presence says whether users are connected (realtime, wired in app).
type Presence interface {
	Online(ctx context.Context, user uuid.UUID) bool
}

// Deps are the service's collaborators.
type Deps struct {
	DB       *pgxpool.Pool
	Users    Users
	Profiles Profiles
	Presence Presence
	Events   *events.Bus
	Clock    clock.Clock
}

// Service implements social's use cases.
type Service struct{ d Deps }

// New returns the service.
func New(d Deps) *Service { return &Service{d: d} }

// FriendRequestSent, Friendship and UserBlocked are the event payloads.
type (
	// FriendRequestSent is the payload of social.FriendRequestSent.
	FriendRequestSent struct {
		RequestID uuid.UUID `json:"request_id"`
		FromID    uuid.UUID `json:"from_user_id"`
		ToID      uuid.UUID `json:"to_user_id"`
	}
	// Friendship is the payload of FriendshipCreated / FriendshipRemoved.
	Friendship struct {
		UserID   uuid.UUID `json:"user_id"`
		FriendID uuid.UUID `json:"friend_id"`
	}
	// UserBlocked is the payload of social.UserBlocked.
	UserBlocked struct {
		BlockerID uuid.UUID `json:"blocker_id"`
		BlockedID uuid.UUID `json:"blocked_id"`
	}
)

// FriendRequest is a friend request with the other user's card.
type FriendRequest struct {
	ID        uuid.UUID    `json:"id"`
	FromID    uuid.UUID    `json:"from_user_id"`
	ToID      uuid.UUID    `json:"to_user_id"`
	Status    string       `json:"status"`
	CreatedAt time.Time    `json:"created_at"`
	ExpiresAt time.Time    `json:"expires_at"`
	Other     profile.Card `json:"user"`
}

// SendResult is a new pending request, or a friendship when the target
// had already asked the sender (both requests accepted).
type SendResult struct {
	Request  *FriendRequest `json:"request,omitempty"`
	Accepted bool           `json:"accepted"`
}

// Send asks to for friendship (schema-social §2 rules).
func (s *Service) Send(ctx context.Context, from, to uuid.UUID, via string) (SendResult, error) {
	if from == to {
		return SendResult{}, apierror.Validation(apierror.FieldError{Field: "to_user_id", Rule: "self"})
	}
	target, err := s.d.Users.User(ctx, to)
	if err != nil || target.Status == identity.StatusDeleted {
		return SendResult{}, apierror.New(apierror.NotFound)
	}
	if target.Tier.Restricted() {
		// Guests and unverified accounts have no friends (D-016).
		return SendResult{}, apierror.New(apierror.Forbidden).With("reason", "target_restricted")
	}
	privacy, err := s.d.Profiles.PrivacyOf(ctx, []uuid.UUID{to})
	if err != nil {
		return SendResult{}, err
	}
	if !privacy[to].AllowFriendRequests {
		return SendResult{}, apierror.New(apierror.Forbidden).With("reason", "friend_requests_off")
	}
	now := s.d.Clock.Now().UTC()
	var res SendResult
	err = postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		if blocked, err := q.IsBlocked(ctx, sqlc.IsBlockedParams{A: from, B: to}); err != nil || blocked {
			if err != nil {
				return err
			}
			return apierror.New(apierror.Blocked)
		}
		if friends, err := q.AreFriends(ctx, sqlc.AreFriendsParams{A: from, B: to}); err != nil || friends {
			if err != nil {
				return err
			}
			return apierror.New(apierror.AlreadyFriends)
		}
		if err := q.ExpireBetween(ctx, sqlc.ExpireBetweenParams{A: from, B: to, Now: &now}); err != nil {
			return err
		}
		pending, err := q.PendingBetween(ctx, sqlc.PendingBetweenParams{A: from, B: to, Now: now})
		switch {
		case err == nil && pending.FromUserID == from:
			return apierror.New(apierror.RequestPending)
		case err == nil:
			// The target asked first: sending back accepts it.
			res.Accepted = true
			return s.acceptTx(ctx, tx, q, pending, now)
		case !errors.Is(err, pgx.ErrNoRows):
			return err
		}
		if err := s.checkCaps(ctx, q, from, to, now); err != nil {
			return err
		}
		id := ids.New()
		if err := q.InsertRequest(ctx, sqlc.InsertRequestParams{ID: id, FromUserID: from, ToUserID: to, Via: via, Now: now, ExpiresAt: now.Add(RequestExpiry)}); err != nil {
			return err
		}
		res.Request = &FriendRequest{ID: id, FromID: from, ToID: to, Status: "pending", CreatedAt: now, ExpiresAt: now.Add(RequestExpiry)}
		return s.d.Events.PublishTx(ctx, tx, EventFriendRequestSent, now, FriendRequestSent{RequestID: id, FromID: from, ToID: to})
	})
	if err != nil {
		return SendResult{}, err
	}
	if res.Request != nil {
		cards, err := s.d.Profiles.Cards(ctx, []uuid.UUID{to})
		if err != nil {
			return SendResult{}, err
		}
		res.Request.Other = cards[to]
	}
	return res, nil
}

func (s *Service) checkCaps(ctx context.Context, q *sqlc.Queries, from, to uuid.UUID, now time.Time) error {
	sent, err := q.SentSince(ctx, sqlc.SentSinceParams{UserID: from, Since: now.Add(-24 * time.Hour)})
	if err != nil {
		return err
	}
	if sent >= RequestsPerDay {
		return apierror.New(apierror.RateLimited).With("limit", RequestsPerDay).WithRetryAfter(time.Hour)
	}
	for _, u := range []uuid.UUID{from, to} {
		n, err := q.FriendCount(ctx, u)
		if err != nil {
			return err
		}
		if n >= FriendCap {
			return apierror.New(apierror.FriendLimitReached).With("limit", FriendCap)
		}
	}
	return nil
}

func (s *Service) acceptTx(ctx context.Context, tx pgx.Tx, q *sqlc.Queries, r sqlc.SocialFriendRequest, now time.Time) error {
	for _, u := range []uuid.UUID{r.FromUserID, r.ToUserID} {
		n, err := q.FriendCount(ctx, u)
		if err != nil {
			return err
		}
		if n >= FriendCap {
			return apierror.New(apierror.FriendLimitReached).With("limit", FriendCap)
		}
	}
	if err := q.SetRequestStatus(ctx, sqlc.SetRequestStatusParams{ID: r.ID, Status: "accepted", Now: &now}); err != nil {
		return err
	}
	if err := q.InsertFriendship(ctx, sqlc.InsertFriendshipParams{UserID: r.FromUserID, FriendID: r.ToUserID, Now: now}); err != nil {
		return err
	}
	return s.d.Events.PublishTx(ctx, tx, EventFriendshipCreated, now, Friendship{UserID: r.FromUserID, FriendID: r.ToUserID})
}

// Respond accepts or declines a request addressed to user.
func (s *Service) Respond(ctx context.Context, user, id uuid.UUID, accept bool) error {
	now := s.d.Clock.Now().UTC()
	return postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		r, err := s.pending(ctx, q, id, now)
		if err != nil {
			return err
		}
		if r.ToUserID != user {
			return apierror.New(apierror.NotFound)
		}
		if !accept {
			return q.SetRequestStatus(ctx, sqlc.SetRequestStatusParams{ID: id, Status: "declined", Now: &now})
		}
		if blocked, err := q.IsBlocked(ctx, sqlc.IsBlockedParams{A: r.FromUserID, B: r.ToUserID}); err != nil || blocked {
			if err != nil {
				return err
			}
			return apierror.New(apierror.Blocked)
		}
		return s.acceptTx(ctx, tx, q, r, now)
	})
}

// Cancel withdraws a request the user sent.
func (s *Service) Cancel(ctx context.Context, user, id uuid.UUID) error {
	now := s.d.Clock.Now().UTC()
	return postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		r, err := s.pending(ctx, q, id, now)
		if err != nil {
			return err
		}
		if r.FromUserID != user {
			return apierror.New(apierror.NotFound)
		}
		return q.SetRequestStatus(ctx, sqlc.SetRequestStatusParams{ID: id, Status: "cancelled", Now: &now})
	})
}

// pending loads a still-pending request (NOT_FOUND otherwise; an expired
// one is marked so).
func (s *Service) pending(ctx context.Context, q *sqlc.Queries, id uuid.UUID, now time.Time) (sqlc.SocialFriendRequest, error) {
	r, err := q.GetRequestForUpdate(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, apierror.New(apierror.NotFound)
	}
	if err != nil {
		return r, err
	}
	if r.Status == "pending" && !r.ExpiresAt.After(now) {
		if err := q.SetRequestStatus(ctx, sqlc.SetRequestStatusParams{ID: id, Status: "expired", Now: &now}); err != nil {
			return r, err
		}
		return r, apierror.New(apierror.NotFound).With("reason", "expired")
	}
	if r.Status != "pending" {
		return r, apierror.New(apierror.NotFound)
	}
	return r, nil
}

// Unfriend removes a friendship (both rows); unknown pairs are fine.
func (s *Service) Unfriend(ctx context.Context, user, friend uuid.UUID) error {
	now := s.d.Clock.Now().UTC()
	return postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		n, err := sqlc.New(tx).DeleteFriendship(ctx, sqlc.DeleteFriendshipParams{A: user, B: friend})
		if err != nil || n == 0 {
			return err
		}
		return s.d.Events.PublishTx(ctx, tx, EventFriendshipRemoved, now, Friendship{UserID: user, FriendID: friend})
	})
}

// Block blocks other: removes the friendship and pending requests in the
// same transaction (schema-social §2).
func (s *Service) Block(ctx context.Context, user, other uuid.UUID) error {
	if user == other {
		return apierror.Validation(apierror.FieldError{Field: "user_id", Rule: "self"})
	}
	if u, err := s.d.Users.User(ctx, other); err != nil || u.Status == identity.StatusDeleted {
		return apierror.New(apierror.NotFound)
	}
	now := s.d.Clock.Now().UTC()
	return postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		removed, err := q.DeleteFriendship(ctx, sqlc.DeleteFriendshipParams{A: user, B: other})
		if err != nil {
			return err
		}
		if err := q.CancelPendingBetween(ctx, sqlc.CancelPendingBetweenParams{A: user, B: other, Now: &now}); err != nil {
			return err
		}
		if err := q.InsertBlock(ctx, sqlc.InsertBlockParams{BlockerID: user, BlockedID: other, Now: now}); err != nil {
			return err
		}
		if removed > 0 {
			if err := s.d.Events.PublishTx(ctx, tx, EventFriendshipRemoved, now, Friendship{UserID: user, FriendID: other}); err != nil {
				return err
			}
		}
		return s.d.Events.PublishTx(ctx, tx, EventUserBlocked, now, UserBlocked{BlockerID: user, BlockedID: other})
	})
}

// Unblock removes the user's block of other.
func (s *Service) Unblock(ctx context.Context, user, other uuid.UUID) error {
	_, err := sqlc.New(s.d.DB).DeleteBlock(ctx, sqlc.DeleteBlockParams{BlockerID: user, BlockedID: other})
	return err
}

// AreFriends reports a friendship.
func (s *Service) AreFriends(ctx context.Context, a, b uuid.UUID) (bool, error) {
	return sqlc.New(s.d.DB).AreFriends(ctx, sqlc.AreFriendsParams{A: a, B: b})
}

// FriendIDs lists a user's friends.
func (s *Service) FriendIDs(ctx context.Context, user uuid.UUID) ([]uuid.UUID, error) {
	return sqlc.New(s.d.DB).FriendIDs(ctx, user)
}

// IsBlocked reports a block in either direction.
func (s *Service) IsBlocked(ctx context.Context, a, b uuid.UUID) (bool, error) {
	return sqlc.New(s.d.DB).IsBlocked(ctx, sqlc.IsBlockedParams{A: a, B: b})
}

// Relation is viewer's tie to other for player profiles (D-101): blocked,
// friend, request_sent, request_received or none.
func (s *Service) Relation(ctx context.Context, viewer, other uuid.UUID) (string, error) {
	q := sqlc.New(s.d.DB)
	if blocked, err := q.IsBlocked(ctx, sqlc.IsBlockedParams{A: viewer, B: other}); err != nil || blocked {
		return profile.RelationBlocked, err
	}
	if friends, err := q.AreFriends(ctx, sqlc.AreFriendsParams{A: viewer, B: other}); err != nil || friends {
		return profile.RelationFriend, err
	}
	from, err := q.PendingDirection(ctx, sqlc.PendingDirectionParams{Now: s.d.Clock.Now().UTC(), A: viewer, B: other})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return profile.RelationNone, nil
	case err != nil:
		return "", err
	case from == viewer:
		return profile.RelationSent, nil
	default:
		return profile.RelationReceived, nil
	}
}

// BlockedWith lists every user blocking or blocked by user (matchmaking).
func (s *Service) BlockedWith(ctx context.Context, user uuid.UUID) ([]uuid.UUID, error) {
	return sqlc.New(s.d.DB).BlockedByOrBlocking(ctx, user)
}

// Friend is one entry of the friends list.
type Friend struct {
	profile.Card
	Since  time.Time `json:"since"`
	Online *bool     `json:"online"` // null when the friend hides their status
}

// FriendsPage is GET /v1/friends.
type FriendsPage struct {
	Friends    []Friend   `json:"friends"`
	NextCursor *uuid.UUID `json:"next_cursor"`
}

// Friends lists friends after cursor, with presence where shown.
func (s *Service) Friends(ctx context.Context, user, after uuid.UUID) (FriendsPage, error) {
	rows, err := sqlc.New(s.d.DB).ListFriends(ctx, sqlc.ListFriendsParams{UserID: user, After: after, RowLimit: friendsPageSize + 1})
	if err != nil {
		return FriendsPage{}, err
	}
	page := FriendsPage{Friends: []Friend{}}
	if len(rows) > friendsPageSize {
		rows = rows[:friendsPageSize]
		last := rows[len(rows)-1].FriendID
		page.NextCursor = &last
	}
	idsList := make([]uuid.UUID, len(rows))
	for i, r := range rows {
		idsList[i] = r.FriendID
	}
	cards, err := s.d.Profiles.Cards(ctx, idsList)
	if err != nil {
		return FriendsPage{}, err
	}
	privacy, err := s.d.Profiles.PrivacyOf(ctx, idsList)
	if err != nil {
		return FriendsPage{}, err
	}
	for _, r := range rows {
		f := Friend{Card: cards[r.FriendID], Since: r.CreatedAt}
		if privacy[r.FriendID].ShowOnlineStatus && s.d.Presence != nil {
			on := s.d.Presence.Online(ctx, r.FriendID)
			f.Online = &on
		}
		page.Friends = append(page.Friends, f)
	}
	return page, nil
}

// Requests lists pending requests: "in" addressed to user, "out" sent.
func (s *Service) Requests(ctx context.Context, user uuid.UUID, box string) ([]FriendRequest, error) {
	q := sqlc.New(s.d.DB)
	now := s.d.Clock.Now().UTC()
	var rows []sqlc.SocialFriendRequest
	var err error
	if box == "out" {
		rows, err = q.ListOutbox(ctx, sqlc.ListOutboxParams{UserID: user, Now: now})
	} else {
		rows, err = q.ListInbox(ctx, sqlc.ListInboxParams{UserID: user, Now: now})
	}
	if err != nil {
		return nil, err
	}
	others := make([]uuid.UUID, len(rows))
	for i := range rows {
		others[i] = rows[i].FromUserID
		if box == "out" {
			others[i] = rows[i].ToUserID
		}
	}
	cards, err := s.d.Profiles.Cards(ctx, others)
	if err != nil {
		return nil, err
	}
	out := make([]FriendRequest, len(rows))
	for i := range rows {
		r := &rows[i]
		out[i] = FriendRequest{ID: r.ID, FromID: r.FromUserID, ToID: r.ToUserID, Status: r.Status, CreatedAt: r.CreatedAt, ExpiresAt: r.ExpiresAt, Other: cards[others[i]]}
	}
	return out, nil
}

// BlockedUser is one entry of the block list.
type BlockedUser struct {
	profile.Card
	Since time.Time `json:"since"`
}

// Blocks lists the users the caller blocked.
func (s *Service) Blocks(ctx context.Context, user uuid.UUID) ([]BlockedUser, error) {
	rows, err := sqlc.New(s.d.DB).ListBlocks(ctx, user)
	if err != nil {
		return nil, err
	}
	others := make([]uuid.UUID, len(rows))
	for i, r := range rows {
		others[i] = r.BlockedID
	}
	cards, err := s.d.Profiles.Cards(ctx, others)
	if err != nil {
		return nil, err
	}
	out := make([]BlockedUser, len(rows))
	for i, r := range rows {
		out[i] = BlockedUser{Card: cards[r.BlockedID], Since: r.CreatedAt}
	}
	return out, nil
}
