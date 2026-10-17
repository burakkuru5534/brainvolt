package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ProfileProvisioner is implemented by the profile module and wired in
// internal/app (identity may not import profile, backend-modules.md §2).
//
// CreateTx runs inside identity's first-session transaction so a user never
// exists without a profile and settings (schema-identity §5.1, D-059: the
// one sanctioned exception to "transactions never span modules").
type ProfileProvisioner interface {
	CreateTx(ctx context.Context, tx pgx.Tx, p NewProfile) error
	Summary(ctx context.Context, userID uuid.UUID) (ProfileSummary, error)
}

// NewProfile is what identity knows when a user is created.
type NewProfile struct {
	UserID      uuid.UUID
	Locale      string  // "tr" | "en"
	CountryCode *string // device region hint, validated ISO 3166-1 alpha-2
	Now         time.Time
}

// ProfileSummary is the profile part of the session response.
type ProfileSummary struct {
	Username    string  `json:"username"`
	DisplayName string  `json:"display_name"`
	AvatarID    string  `json:"avatar_id"`
	CountryCode *string `json:"country_code"`
	Locale      string  `json:"locale"`
}

// Event names produced by identity (schema-identity §7).
const (
	EventUserRegistered     = "identity.UserRegistered"
	EventAccountTierChanged = "identity.AccountTierChanged"
	EventGuestAbandoned     = "identity.GuestAbandoned"
)

// UserRegistered is published when a first session creates a user.
type UserRegistered struct {
	UserID uuid.UUID `json:"user_id"`
	Tier   Tier      `json:"tier"`
	Locale string    `json:"locale"`
}

// AccountTierChanged is published when a session sync changes the tier.
type AccountTierChanged struct {
	UserID uuid.UUID `json:"user_id"`
	From   Tier      `json:"from"`
	To     Tier      `json:"to"`
}

// GuestAbandoned is published when a guest confirms a link conflict (D-015).
type GuestAbandoned struct {
	UserID uuid.UUID `json:"user_id"`
}
