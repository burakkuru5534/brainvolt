// Package profile is the public API of the profile module
// (backend-modules.md §1): display name, username, avatar, country,
// locale, settings and the public profile read model.
package profile

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile/internal/service"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile/internal/transport"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ratelimit"
)

// Module is the profile module.
type Module struct {
	svc *service.Service
}

// New builds the module.
func New(db *pgxpool.Pool, c clock.Clock) *Module { return &Module{svc: service.New(db, c)} }

// Provisioner creates and summarises profiles for identity's session flow
// (wired in internal/app, schema-identity §5.1).
func (m *Module) Provisioner() identity.ProfileProvisioner { return m.svc }

// Card is a player's public face (display name, avatar, country).
type Card = service.Card

// Cards returns the public cards of users (missing users are left out).
func (m *Module) Cards(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]Card, error) {
	return m.svc.Cards(ctx, ids)
}

// Privacy is the part of a user's settings other modules enforce.
type Privacy = service.Privacy

// PrivacyOf returns users' privacy settings (challenges_from, online
// status visibility, friend requests).
func (m *Module) PrivacyOf(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]Privacy, error) {
	return m.svc.PrivacyOf(ctx, ids)
}

// TextRejected reports profanity or reserved names in a public text (team
// and couple names; the basic list until moderation, Phase 8).
func TextRejected(s string) bool { return domain.Rejected(s) }

// MaskText hides blocked words in free text (chat) with asterisks;
// severe reports a slur, which callers refuse (D-030, D-088).
func MaskText(s string) (masked string, changed, severe bool) { return domain.Mask(s) }

// LocalesOf returns users' content languages (match texts).
func (m *Module) LocalesOf(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	return m.svc.LocalesOf(ctx, ids)
}

// UserIDByUsername finds a user by username (NOT_FOUND).
func (m *Module) UserIDByUsername(ctx context.Context, username string) (uuid.UUID, error) {
	return m.svc.UserIDByUsername(ctx, username)
}

// PlayerStats is the public part of a player's progress (D-101).
type PlayerStats = service.PlayerStats

// RelationReader is implemented by social (profile → social*).
type RelationReader = service.RelationReader

// StatsReader is implemented by progression (profile → progression*).
type StatsReader = service.StatsReader

// Relation values of a player profile (D-101).
const (
	RelationSelf     = service.RelationSelf
	RelationFriend   = service.RelationFriend
	RelationSent     = service.RelationSent
	RelationReceived = service.RelationReceived
	RelationBlocked  = service.RelationBlocked
	RelationNone     = service.RelationNone
)

// SetReaders wires the upward reads of player profiles (internal/app).
func (m *Module) SetReaders(rel RelationReader, stats StatsReader) { m.svc.SetReaders(rel, stats) }

// Register adds profile's routes; ident authenticates them.
func (m *Module) Register(s *httpserver.Server, ident *identity.Module, limiter *ratelimit.Limiter) {
	transport.New(m.svc, ident, limiter).Register(s)
}

// Worn is the part of a player's cosmetics everyone sees (D-106).
type Worn = service.Worn

// CosmeticsReader is commerce's cosmetics, wired in internal/app.
type CosmeticsReader = service.CosmeticsReader

// SetCosmetics wires the cosmetics shown on cards (profile → commerce*).
func (m *Module) SetCosmetics(c CosmeticsReader) { m.svc.SetCosmetics(c) }
