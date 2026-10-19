package service

import (
	"context"
	crand "crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/lobby/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
)

// Invite code format (D-027): 6 characters without look-alikes (0/O/1/I).
const (
	codeAlphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
	codeLength   = 6
	codeTries    = 5
	// FriendCodeTTL is how long a friend invite code works.
	FriendCodeTTL = 7 * 24 * time.Hour
	// codeRetention keeps expired codes before they may be drawn again.
	codeRetention = 30 * 24 * time.Hour
)

// Code target kinds.
const (
	KindRoom   = "room"
	KindFriend = "friend"
)

func cryptoIntn(n int) int {
	v, err := crand.Int(crand.Reader, big.NewInt(int64(n)))
	if err != nil {
		panic(fmt.Sprintf("crypto/rand: %v", err))
	}
	return int(v.Int64())
}

func (s *Service) drawCode() string {
	var b strings.Builder
	for range codeLength {
		b.WriteByte(codeAlphabet[s.d.Random(len(codeAlphabet))])
	}
	return b.String()
}

// newCode stores a fresh code for a target, drawing again on a collision.
func (s *Service) newCode(ctx context.Context, q *sqlc.Queries, kind string, target, by uuid.UUID, expires, now time.Time) (string, error) {
	for range codeTries {
		code := s.drawCode()
		err := q.InsertCode(ctx, sqlc.InsertCodeParams{Code: code, TargetKind: kind, TargetID: target, CreatedBy: by, ExpiresAt: expires, Now: now})
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pe.Code == "23505" {
			continue
		}
		return code, err
	}
	return "", errors.New("lobby: no free invite code")
}

// InviteCode is a code as its creator sees it.
type InviteCode struct {
	Code      string    `json:"code"`
	Kind      string    `json:"kind"`
	ExpiresAt time.Time `json:"expires_at"`
}

// FriendCode returns the caller's friend invite code, creating one when
// none is usable (one code per user, valid for 7 days).
func (s *Service) FriendCode(ctx context.Context, user uuid.UUID) (InviteCode, error) {
	now := s.d.Clock.Now().UTC()
	q := sqlc.New(s.d.DB)
	c, err := q.ActiveCode(ctx, sqlc.ActiveCodeParams{TargetKind: KindFriend, TargetID: user, Now: now})
	if err == nil {
		return InviteCode{Code: c.Code, Kind: KindFriend, ExpiresAt: c.ExpiresAt}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return InviteCode{}, err
	}
	expires := now.Add(FriendCodeTTL)
	code, err := s.newCode(ctx, q, KindFriend, user, user, expires, now)
	if err != nil {
		return InviteCode{}, err
	}
	return InviteCode{Code: code, Kind: KindFriend, ExpiresAt: expires}, nil
}

// ResolvedCode is what a code opens (GET /v1/invite-codes/{code}).
type ResolvedCode struct {
	Code      string    `json:"code"`
	Kind      string    `json:"kind"`
	ExpiresAt time.Time `json:"expires_at"`
	// User is the inviting user of a friend code.
	User *profile.Card `json:"user,omitempty"`
	// Room is the room of a room code.
	Room *RoomPreview `json:"room,omitempty"`
}

// normalizeCode accepts lower case and surrounding spaces.
func normalizeCode(code string) (string, error) {
	c := strings.ToUpper(strings.TrimSpace(code))
	if len(c) != codeLength {
		return "", apierror.New(apierror.InviteCodeInvalid)
	}
	for _, r := range c {
		if !strings.ContainsRune(codeAlphabet, r) {
			return "", apierror.New(apierror.InviteCodeInvalid)
		}
	}
	return c, nil
}

// lookup returns a usable code (INVITE_CODE_INVALID / _EXPIRED).
func (s *Service) lookup(ctx context.Context, code string) (sqlc.LobbyInviteCode, error) {
	c, err := normalizeCode(code)
	if err != nil {
		return sqlc.LobbyInviteCode{}, err
	}
	row, err := sqlc.New(s.d.DB).GetCode(ctx, c)
	if errors.Is(err, pgx.ErrNoRows) {
		return row, apierror.New(apierror.InviteCodeInvalid)
	}
	if err != nil {
		return row, err
	}
	if row.RevokedAt != nil || !row.ExpiresAt.After(s.d.Clock.Now()) || (row.MaxUses != nil && row.UseCount >= *row.MaxUses) {
		return row, apierror.New(apierror.InviteCodeExpired)
	}
	return row, nil
}

// Resolve says what a code opens before the caller uses it.
func (s *Service) Resolve(ctx context.Context, caller uuid.UUID, code string) (ResolvedCode, error) {
	row, err := s.lookup(ctx, code)
	if err != nil {
		return ResolvedCode{}, err
	}
	out := ResolvedCode{Code: row.Code, Kind: row.TargetKind, ExpiresAt: row.ExpiresAt}
	switch row.TargetKind {
	case KindFriend:
		cards, err := s.cards(ctx, row.TargetID)
		if err != nil {
			return ResolvedCode{}, err
		}
		c, ok := cards[row.TargetID]
		if !ok {
			return ResolvedCode{}, apierror.New(apierror.InviteCodeExpired)
		}
		out.User = &c
	case KindRoom:
		p, err := s.preview(ctx, row.TargetID)
		if err != nil {
			return ResolvedCode{}, err
		}
		out.Room = &p
	}
	if blocked, err := s.d.Social.IsBlocked(ctx, caller, row.CreatedBy); err != nil || blocked {
		if err != nil {
			return ResolvedCode{}, err
		}
		return ResolvedCode{}, apierror.New(apierror.Blocked)
	}
	return out, nil
}

// SweepCodes deletes codes expired for 30 days so they can be drawn again.
func (s *Service) SweepCodes(ctx context.Context) error {
	return sqlc.New(s.d.DB).DeleteOldCodes(ctx, s.d.Clock.Now().Add(-codeRetention))
}
