// Package domain holds identity's pure rules: account tiers, statuses and
// how a verified Firebase token maps onto linked sign-in methods
// (docs/domain/schema-identity.md §1–§2). No I/O.
package domain

import (
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/platform/firebase"
)

// Tier is the account tier (schema-identity §2.1).
type Tier string

const (
	// TierGuest is an anonymous-only account.
	TierGuest Tier = "guest"
	// TierUnverified is an email/password account whose email is not verified (D-017).
	TierUnverified Tier = "unverified"
	// TierFull is a Google/Apple account or a verified email account.
	TierFull Tier = "full"
)

// Restricted reports whether t is limited by D-016 / D-017.
func (t Tier) Restricted() bool { return t != TierFull }

// Status is the account status.
type Status string

const (
	// StatusActive is a normal account.
	StatusActive Status = "active"
	// StatusSuspended can sign in and read its profile but not play (§5.5).
	StatusSuspended Status = "suspended"
	// StatusDeleted is purged; its tokens are no longer accepted.
	StatusDeleted Status = "deleted"
)

// Provider is a sign-in method as stored in identity.auth_identities.
type Provider string

// Providers as stored in identity.auth_identities.provider.
const (
	ProviderAnonymous Provider = "anonymous"
	ProviderGoogle    Provider = "google"
	ProviderApple     Provider = "apple"
	ProviderPassword  Provider = "password"
)

// LinkedIdentity is one sign-in method linked to the Firebase user.
type LinkedIdentity struct {
	Provider      Provider
	ProviderUID   *string // Google/Apple subject
	Email         *string
	EmailVerified bool
}

// DeriveTier implements the table of schema-identity §2.1:
// any Google or Apple, or password with a verified email → full;
// password with an unverified email (and no Google/Apple) → unverified;
// otherwise (anonymous only) → guest.
func DeriveTier(identities []LinkedIdentity) Tier {
	tier := TierGuest
	for _, id := range identities {
		switch id.Provider {
		case ProviderGoogle, ProviderApple:
			return TierFull
		case ProviderPassword:
			if id.EmailVerified {
				return TierFull
			}
			tier = TierUnverified
		case ProviderAnonymous:
		}
	}
	return tier
}

// TierFromToken derives the tier from a verified ID token alone; used on
// every request to detect a stale cached tier cheaply. It equals the tier
// of IdentitiesFromAccount: Google/Apple identities → full; otherwise an
// "email" identity (password account) → full if verified, else unverified;
// no identities → guest.
func TierFromToken(t firebase.Token) Tier {
	if _, ok := t.Identities["google.com"]; ok {
		return TierFull
	}
	if _, ok := t.Identities["apple.com"]; ok {
		return TierFull
	}
	if _, ok := t.Identities["email"]; ok {
		if t.EmailVerified {
			return TierFull
		}
		return TierUnverified
	}
	return TierGuest
}

// IdentitiesFromAccount maps the Firebase user record onto the mirror rows
// (schema-identity §1). No linked provider means an anonymous account.
func IdentitiesFromAccount(a firebase.Account) []LinkedIdentity {
	var out []LinkedIdentity
	for _, p := range a.Providers {
		switch p.ProviderID {
		case "google.com":
			out = append(out, LinkedIdentity{Provider: ProviderGoogle, ProviderUID: optional(p.UID), Email: optional(p.Email), EmailVerified: a.EmailVerified})
		case "apple.com":
			out = append(out, LinkedIdentity{Provider: ProviderApple, ProviderUID: optional(p.UID), Email: optional(p.Email), EmailVerified: a.EmailVerified})
		case "password":
			email := p.Email
			if email == "" {
				email = a.Email
			}
			out = append(out, LinkedIdentity{Provider: ProviderPassword, Email: optional(email), EmailVerified: a.EmailVerified})
		}
	}
	if len(out) == 0 {
		out = append(out, LinkedIdentity{Provider: ProviderAnonymous})
	}
	slices.SortFunc(out, func(x, y LinkedIdentity) int { return strings.Compare(string(x.Provider), string(y.Provider)) })
	return out
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// Providers lists the providers of identities.
func Providers(identities []LinkedIdentity) []string {
	out := make([]string, 0, len(identities))
	for _, id := range identities {
		out = append(out, string(id.Provider))
	}
	slices.Sort(out)
	return out
}

// Principal is the authenticated caller of a request.
type Principal struct {
	UserID              uuid.UUID
	FirebaseUID         string
	Tier                Tier
	Status              Status
	DeletionRequestedAt *time.Time
	Roles               []string
}

// HasRole reports whether the principal has a staff role.
func (p Principal) HasRole(role string) bool { return slices.Contains(p.Roles, role) }

// DeletionGracePeriod is the cooling-off window before a requested deletion
// is executed (D-022).
const DeletionGracePeriod = 14 * 24 * time.Hour
