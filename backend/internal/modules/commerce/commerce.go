// Package commerce is the public API of the commerce module
// (backend-modules.md §1, schema-platform.md §4): rewarded ads verified by
// AdMob (D-103) and entitlements; Plus subscriptions and cosmetics come in
// later phases.
package commerce

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/commerce/internal/service"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/commerce/internal/transport"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
)

// EventAdRewardGranted: a rewarded ad was verified (payload AdRewardGranted).
const EventAdRewardGranted = service.EventAdRewardGranted

// Placements of rewarded ads.
const (
	PlacementBonusXP      = service.PlacementBonusXP
	PlacementQuestReroll  = service.PlacementQuestReroll
	PlacementStreakRepair = service.PlacementStreakRepair
)

// AdRewardGranted is EventAdRewardGranted's payload.
type AdRewardGranted = service.AdRewardGranted

// Offer is what a rewarded ad gives.
type Offer = service.Offer

// RewardChecker is implemented by progression and wired in internal/app
// (commerce → progression*, backend-modules.md §2).
type RewardChecker = service.RewardChecker

// KeySource returns AdMob's SSV verifying keys.
type KeySource = service.KeySource

// ParsePublicKey reads an ECDSA public key in PEM (tests).
var ParsePublicKey = service.ParsePublicKey

// Deps are the module's collaborators.
type Deps struct {
	DB       *pgxpool.Pool
	Identity *identity.Module
	Events   *events.Bus
	// Keys verifies SSV callbacks; nil uses AdMob's published keys.
	Keys   KeySource
	Clock  clock.Clock
	Logger *slog.Logger
	// DevGrant enables granting rewards without AdMob (development only).
	DevGrant bool
	// RevenueCatSecret is the webhook's Authorization value (D-104); empty
	// turns the webhook off.
	RevenueCatSecret string
}

// Module is the commerce module.
type Module struct {
	svc      *service.Service
	ident    *identity.Module
	devGrant bool
	rcSecret string
}

type users struct{ *identity.Module }

func (u users) CreatedAt(ctx context.Context, id uuid.UUID) (time.Time, error) {
	user, err := u.User(ctx, id)
	return user.CreatedAt, err
}

// New builds the module.
func New(d Deps) *Module {
	keys := d.Keys
	if keys == nil {
		keys = service.NewHTTPKeys()
	}
	return &Module{
		svc:      service.New(service.Deps{DB: d.DB, Users: users{d.Identity}, Events: d.Events, Keys: keys, Clock: d.Clock, Logger: d.Logger}),
		ident:    d.Identity,
		devGrant: d.DevGrant,
		rcSecret: d.RevenueCatSecret,
	}
}

// SetRewards wires progression's reward checker.
func (m *Module) SetRewards(r RewardChecker) { m.svc.SetRewards(r) }

// Register adds commerce's routes.
func (m *Module) Register(s *httpserver.Server) {
	transport.New(m.svc, m.ident, m.devGrant, m.rcSecret).Register(s)
}

// EntitlementPlus is BrainVolt Plus (D-104).
const EntitlementPlus = service.EntitlementPlus

// EntitlementReader answers HasEntitlement without the rest of the module
// (the worker's progression, which has no identity module).
type EntitlementReader struct{ svc *service.Service }

// NewEntitlementReader returns a reader over db.
func NewEntitlementReader(db *pgxpool.Pool, c clock.Clock) EntitlementReader {
	return EntitlementReader{svc: service.New(service.Deps{DB: db, Clock: c})}
}

// HasEntitlement reports whether user holds one of codes now.
func (r EntitlementReader) HasEntitlement(ctx context.Context, user uuid.UUID, codes ...string) (bool, error) {
	return r.svc.HasEntitlement(ctx, user, codes...)
}

// HasEntitlement reports whether user holds one of codes now ('plus',
// 'no_ads', …).
func (m *Module) HasEntitlement(ctx context.Context, user uuid.UUID, codes ...string) (bool, error) {
	return m.svc.HasEntitlement(ctx, user, codes...)
}
