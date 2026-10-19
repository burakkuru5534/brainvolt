package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/team/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
)

// Couple is a couple profile with both partners' cards.
type Couple struct {
	ID          uuid.UUID      `json:"id"`
	Name        *string        `json:"name"`
	Status      string         `json:"status"`
	RequestedBy uuid.UUID      `json:"requested_by"`
	Partners    []profile.Card `json:"partners"`
	CreatedAt   time.Time      `json:"created_at"`
	RequestedAt time.Time      `json:"requested_at"`
	ExpiresAt   *time.Time     `json:"expires_at,omitempty"`
	ActivatedAt *time.Time     `json:"activated_at"`
}

// CoupleRequest is the body of POST /v1/couples.
type CoupleRequest struct {
	PartnerUserID uuid.UUID `json:"partner_user_id"`
	Name          *string   `json:"name,omitempty"`
}

func pair(a, b uuid.UUID) (low, high uuid.UUID) {
	if a.String() < b.String() {
		return a, b
	}
	return b, a
}

func (s *Service) couple(ctx context.Context, c *sqlc.TeamCouple) (Couple, error) {
	cards, err := s.d.Profiles.Cards(ctx, []uuid.UUID{c.UserLowID, c.UserHighID})
	if err != nil {
		return Couple{}, err
	}
	out := Couple{ID: c.ID, Name: c.Name, Status: c.Status, RequestedBy: c.RequestedBy, CreatedAt: c.CreatedAt,
		RequestedAt: c.RequestedAt, ActivatedAt: c.ActivatedAt, Partners: []profile.Card{cards[c.UserLowID], cards[c.UserHighID]}}
	if c.Status == "pending" {
		e := c.RequestedAt.Add(CoupleTTL)
		out.ExpiresAt = &e
	}
	return out, nil
}

// liveCouple returns a user's pending or active couple, expiring a stale
// request first.
func (s *Service) liveCouple(ctx context.Context, q *sqlc.Queries, user uuid.UUID) (*sqlc.TeamCouple, error) {
	c, err := q.LiveCoupleOf(ctx, user)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if c.Status == "pending" && !c.RequestedAt.Add(CoupleTTL).After(s.now()) {
		if err := q.SetCoupleStatus(ctx, sqlc.SetCoupleStatusParams{ID: c.ID, Status: "expired", Now: s.now()}); err != nil {
			return nil, err
		}
		return nil, nil
	}
	return &c, nil
}

// RequestCouple asks a friend to form a couple (both full accounts, one
// live couple each). The same pair reactivates its old profile (same id,
// history and rating); a different partner needs 7 days after the last
// dissolved couple (D-047).
func (s *Service) RequestCouple(ctx context.Context, user uuid.UUID, r CoupleRequest) (Couple, error) {
	partner := r.PartnerUserID
	if partner == user || partner == uuid.Nil {
		return Couple{}, apierror.Validation(apierror.FieldError{Field: "partner_user_id", Rule: "not_self"})
	}
	if r.Name != nil {
		n := strings.Join(strings.Fields(*r.Name), " ")
		if l := len([]rune(n)); l < 3 || l > 24 {
			return Couple{}, apierror.Validation(apierror.FieldError{Field: "name", Rule: "length"})
		}
		if profile.TextRejected(n) {
			return Couple{}, apierror.New(apierror.TextRejected)
		}
		r.Name = &n
	}
	for _, u := range []uuid.UUID{user, partner} {
		if err := s.fullUser(ctx, u, "couples"); err != nil {
			return Couple{}, err
		}
	}
	blocked, err := s.d.Social.IsBlocked(ctx, user, partner)
	if err != nil {
		return Couple{}, err
	}
	if blocked {
		return Couple{}, apierror.New(apierror.Blocked)
	}
	friends, err := s.d.Social.AreFriends(ctx, user, partner)
	if err != nil {
		return Couple{}, err
	}
	if !friends {
		return Couple{}, apierror.New(apierror.Forbidden).With("reason", "not_friends")
	}
	now := s.now()
	low, high := pair(user, partner)
	var id uuid.UUID
	err = postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		for _, u := range []uuid.UUID{user, partner} {
			live, err := s.liveCouple(ctx, q, u)
			if err != nil {
				return err
			}
			if live != nil {
				return apierror.New(apierror.CoupleExists).With("user_id", u)
			}
			last, err := q.LastDissolvedWithOther(ctx, sqlc.LastDissolvedWithOtherParams{UserID: u, Partner: map[uuid.UUID]uuid.UUID{user: partner, partner: user}[u]})
			if err != nil {
				return err
			}
			if at := last.Add(CoupleCooldown); at.After(now) {
				return apierror.New(apierror.CoupleCooldown).With("user_id", u).With("available_at", at)
			}
		}
		old, err := q.CoupleOfPair(ctx, sqlc.CoupleOfPairParams{Low: low, High: high})
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			id = ids.New()
			if err := q.InsertCouple(ctx, sqlc.InsertCoupleParams{ID: id, UserLowID: low, UserHighID: high, RequestedBy: user, Name: r.Name, Now: now}); err != nil {
				return err
			}
		case err != nil:
			return err
		default:
			id = old.ID // the same pair again: reactivate (couple-contest §6)
			if err := q.RequestAgain(ctx, sqlc.RequestAgainParams{ID: id, RequestedBy: user, Name: r.Name, Now: now}); err != nil {
				return err
			}
		}
		return s.d.Events.PublishTx(ctx, tx, EventCoupleRequested, now, CoupleEvent{CoupleID: id, UserIDs: [2]uuid.UUID{user, partner}})
	})
	if err != nil {
		return Couple{}, err
	}
	return s.Couple(ctx, id)
}

// Couple returns a couple profile.
func (s *Service) Couple(ctx context.Context, id uuid.UUID) (Couple, error) {
	c, err := sqlc.New(s.d.DB).GetCouple(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Couple{}, apierror.New(apierror.NotFound)
	}
	if err != nil {
		return Couple{}, err
	}
	return s.couple(ctx, &c)
}

// MyCouple returns the caller's pending or active couple (nil when none).
func (s *Service) MyCouple(ctx context.Context, user uuid.UUID) (*Couple, error) {
	c, err := s.liveCouple(ctx, sqlc.New(s.d.DB), user)
	if err != nil || c == nil {
		return nil, err
	}
	out, err := s.couple(ctx, c)
	return &out, err
}

// Couple actions.
const (
	CoupleAccept   = "accept"
	CoupleDecline  = "decline"
	CoupleDissolve = "dissolve"
)

// RespondCouple accepts (the partner who was asked), declines (either
// partner while pending; the requester withdraws) or dissolves (either
// partner while active) a couple.
func (s *Service) RespondCouple(ctx context.Context, user, id uuid.UUID, action string) (Couple, error) {
	now := s.now()
	err := postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		c, err := q.GetCoupleForUpdate(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && user != c.UserLowID && user != c.UserHighID) {
			return apierror.New(apierror.NotFound)
		}
		if err != nil {
			return err
		}
		pending := c.Status == "pending" && c.RequestedAt.Add(CoupleTTL).After(now)
		var status, event string
		switch {
		case action == CoupleAccept && pending && user != c.RequestedBy:
			status, event = "active", EventCoupleFormed
		case action == CoupleDecline && pending:
			status = "declined"
		case action == CoupleDissolve && c.Status == "active":
			status, event = "dissolved", EventCoupleDissolved
		case c.Status == "pending" && !pending:
			if err := q.SetCoupleStatus(ctx, sqlc.SetCoupleStatusParams{ID: id, Status: "expired", Now: now}); err != nil {
				return err
			}
			return apierror.New(apierror.ChallengeExpired).With("reason", "couple_request_expired")
		default:
			return apierror.New(apierror.MatchStateInvalid).With("reason", "couple_"+c.Status)
		}
		if err := q.SetCoupleStatus(ctx, sqlc.SetCoupleStatusParams{ID: id, Status: status, Now: now}); err != nil {
			return err
		}
		if event == "" {
			return nil
		}
		return s.d.Events.PublishTx(ctx, tx, event, now, CoupleEvent{CoupleID: id, UserIDs: [2]uuid.UUID{c.UserLowID, c.UserHighID}})
	})
	if err != nil {
		return Couple{}, err
	}
	return s.Couple(ctx, id)
}
