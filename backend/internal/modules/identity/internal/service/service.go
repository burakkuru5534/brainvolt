// Package service holds identity's use cases: session sync (first sign-in,
// guest upgrade, email verification), principal resolution for every
// authenticated request, the link-conflict exit (D-015) and account
// deletion requests (D-022).
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/firebase"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/redis"
)

// principalTTL is the life of the id:fuid:* cache entry (redis-keys §7).
const principalTTL = 10 * time.Minute

// Verifier is the part of platform/firebase the service uses.
type Verifier interface {
	Verify(ctx context.Context, raw string) (firebase.Token, error)
	VerifyAndCheckRevoked(ctx context.Context, raw string) (firebase.Token, error)
	Account(ctx context.Context, uid string) (firebase.Account, error)
}

// Deps are the service's collaborators.
type Deps struct {
	DB       *pgxpool.Pool
	Redis    *redis.Client
	Verifier Verifier
	Profiles domain.ProfileProvisioner
	Events   *events.Bus
	Clock    clock.Clock
	Logger   *slog.Logger
}

// Service implements identity's use cases.
type Service struct{ d Deps }

// New returns a service.
func New(d Deps) *Service { return &Service{d: d} }

// User is the identity part of the session and /v1/me responses.
type User struct {
	ID                  uuid.UUID     `json:"id"`
	Tier                domain.Tier   `json:"tier"`
	Status              domain.Status `json:"status"`
	Roles               []string      `json:"roles"`
	CreatedAt           time.Time     `json:"created_at"`
	UpgradedAt          *time.Time    `json:"upgraded_at"`
	DeletionRequestedAt *time.Time    `json:"deletion_requested_at"`
}

// SessionInput is the validated body of POST /v1/session.
type SessionInput struct {
	Locale      string
	CountryHint *string
}

// SessionResult is what POST /v1/session returns.
type SessionResult struct {
	User    User
	Profile domain.ProfileSummary
	Created bool
}

// Session syncs the BrainVolt account with Firebase after a sign-in, link or
// email verification (schema-identity §5.1–§5.3). The first call creates the
// user, its mirror rows, profile and settings in one transaction; later
// calls refresh the mirror and the tier. Idempotent.
func (s *Service) Session(ctx context.Context, tok firebase.Token, in SessionInput) (SessionResult, error) {
	account, err := s.d.Verifier.Account(ctx, tok.UID)
	if err != nil {
		return SessionResult{}, err
	}
	identities := domain.IdentitiesFromAccount(account)
	tier := domain.DeriveTier(identities)
	now := s.d.Clock.Now().UTC()

	var (
		userID  uuid.UUID
		created bool
	)
	err = postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		u, err := q.GetUserByFirebaseUIDForUpdate(ctx, tok.UID)
		if errors.Is(err, pgx.ErrNoRows) {
			id := ids.New()
			n, err := q.InsertUser(ctx, sqlc.InsertUserParams{ID: id, FirebaseUid: tok.UID, AccountTier: string(tier), Now: now})
			if err != nil {
				return err
			}
			if n == 1 {
				userID, created = id, true
				return s.createTx(ctx, tx, q, id, tier, identities, in, now)
			}
			// A concurrent first session won the race: continue as an existing user.
			u, err = q.GetUserByFirebaseUIDForUpdate(ctx, tok.UID)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		userID = u.ID
		return s.syncTx(ctx, tx, q, u, identities, tier, now)
	})
	if err != nil {
		return SessionResult{}, err
	}
	s.forget(ctx, tok.UID)

	user, err := s.User(ctx, userID)
	if err != nil {
		return SessionResult{}, err
	}
	profile, err := s.d.Profiles.Summary(ctx, userID)
	if err != nil {
		return SessionResult{}, err
	}
	return SessionResult{User: user, Profile: profile, Created: created}, nil
}

func (s *Service) createTx(ctx context.Context, tx pgx.Tx, q *sqlc.Queries, id uuid.UUID, tier domain.Tier,
	identities []domain.LinkedIdentity, in SessionInput, now time.Time,
) error {
	if err := s.mirrorTx(ctx, q, id, identities, now); err != nil {
		return err
	}
	if err := s.d.Profiles.CreateTx(ctx, tx, domain.NewProfile{UserID: id, Locale: in.Locale, CountryCode: in.CountryHint, Now: now}); err != nil {
		return fmt.Errorf("creating profile: %w", err)
	}
	return s.d.Events.PublishTx(ctx, tx, domain.EventUserRegistered, now,
		domain.UserRegistered{UserID: id, Tier: tier, Locale: in.Locale})
}

// syncTx refreshes an existing user's mirror and tier.
func (s *Service) syncTx(ctx context.Context, tx pgx.Tx, q *sqlc.Queries, u sqlc.IdentityUser,
	identities []domain.LinkedIdentity, tier domain.Tier, now time.Time,
) error {
	if domain.Status(u.Status) == domain.StatusDeleted {
		return apierror.New(apierror.Unauthenticated).With("reason", "account_deleted")
	}
	if err := s.mirrorTx(ctx, q, u.ID, identities, now); err != nil {
		return err
	}
	if old := domain.Tier(u.AccountTier); old != tier {
		if err := q.UpdateUserTier(ctx, sqlc.UpdateUserTierParams{ID: u.ID, AccountTier: string(tier), Now: now}); err != nil {
			return err
		}
		if err := s.d.Events.PublishTx(ctx, tx, domain.EventAccountTierChanged, now,
			domain.AccountTierChanged{UserID: u.ID, From: old, To: tier}); err != nil {
			return err
		}
	}
	return q.TouchUser(ctx, sqlc.TouchUserParams{ID: u.ID, Now: now})
}

// mirrorTx stores the user's sign-in methods as the Firebase account record
// lists them. Firebase lets one account own a Google/Apple identity or a
// password email; when another BrainVolt user still holds it here, that
// user's Firebase account no longer exists (deleted, or a restarted Auth
// emulator in development), so the stale row moves to this user.
func (s *Service) mirrorTx(ctx context.Context, q *sqlc.Queries, userID uuid.UUID, identities []domain.LinkedIdentity, now time.Time) error {
	providers := make([]string, 0, len(identities))
	for _, id := range identities {
		providers = append(providers, string(id.Provider))
		var released []uuid.UUID
		var err error
		switch {
		case id.ProviderUID != nil:
			released, err = q.ReleaseProviderUID(ctx, sqlc.ReleaseProviderUIDParams{Provider: string(id.Provider), ProviderUid: id.ProviderUID, UserID: userID})
		case id.Provider == domain.ProviderPassword && id.Email != nil:
			released, err = q.ReleasePasswordEmail(ctx, sqlc.ReleasePasswordEmailParams{Email: id.Email, UserID: userID})
		}
		if err != nil {
			return fmt.Errorf("mirroring %s: %w", id.Provider, err)
		}
		for _, other := range released {
			s.d.Logger.WarnContext(ctx, "sign-in method moved from a stale account", "provider", id.Provider, "from_user", other, "to_user", userID)
		}
		if err := q.UpsertAuthIdentity(ctx, sqlc.UpsertAuthIdentityParams{
			UserID: userID, Provider: string(id.Provider), ProviderUid: id.ProviderUID,
			Email: id.Email, EmailVerified: id.EmailVerified, Now: now,
		}); err != nil {
			return fmt.Errorf("mirroring %s: %w", id.Provider, err)
		}
	}
	return q.DeleteAuthIdentitiesExcept(ctx, sqlc.DeleteAuthIdentitiesExceptParams{UserID: userID, Providers: providers})
}

// User returns the identity view of a user.
func (s *Service) User(ctx context.Context, id uuid.UUID) (User, error) {
	q := sqlc.New(s.d.DB)
	u, err := q.GetUserByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, apierror.New(apierror.NotFound)
	}
	if err != nil {
		return User{}, err
	}
	roles, err := q.ListUserRoles(ctx, id)
	if err != nil {
		return User{}, err
	}
	return User{
		ID: u.ID, Tier: domain.Tier(u.AccountTier), Status: domain.Status(u.Status), Roles: roles,
		CreatedAt: u.CreatedAt, UpgradedAt: u.UpgradedAt, DeletionRequestedAt: u.DeletionRequestedAt,
	}, nil
}

// cachedPrincipal is the JSON stored under id:fuid:<firebase_uid>.
type cachedPrincipal struct {
	UserID              uuid.UUID     `json:"user_id"`
	Tier                domain.Tier   `json:"tier"`
	Status              domain.Status `json:"status"`
	DeletionRequestedAt *time.Time    `json:"deletion_requested_at"`
	Roles               []string      `json:"roles"`
}

func principalKey(firebaseUID string) string { return redis.Key("id", "fuid", firebaseUID) }

// Principal resolves the caller of an authenticated request. A tier that no
// longer matches the token's claims is re-synced first (schema-identity §1),
// so a stale tier can never grant more than Firebase says.
func (s *Service) Principal(ctx context.Context, tok firebase.Token) (domain.Principal, error) {
	p, err := s.loadPrincipal(ctx, tok.UID)
	if err != nil {
		return domain.Principal{}, err
	}
	if domain.TierFromToken(tok) != p.Tier && p.Status != domain.StatusDeleted {
		if err := s.resync(ctx, tok); err != nil {
			return domain.Principal{}, err
		}
		if p, err = s.loadPrincipal(ctx, tok.UID); err != nil {
			return domain.Principal{}, err
		}
	}
	if p.Status == domain.StatusDeleted {
		return domain.Principal{}, apierror.New(apierror.Unauthenticated).With("reason", "account_deleted")
	}
	return p, nil
}

func (s *Service) resync(ctx context.Context, tok firebase.Token) error {
	account, err := s.d.Verifier.Account(ctx, tok.UID)
	if err != nil {
		return err
	}
	identities := domain.IdentitiesFromAccount(account)
	tier := domain.DeriveTier(identities)
	now := s.d.Clock.Now().UTC()
	err = postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		u, err := q.GetUserByFirebaseUIDForUpdate(ctx, tok.UID)
		if err != nil {
			return err
		}
		return s.syncTx(ctx, tx, q, u, identities, tier, now)
	})
	s.forget(ctx, tok.UID)
	return err
}

func (s *Service) loadPrincipal(ctx context.Context, firebaseUID string) (domain.Principal, error) {
	key := principalKey(firebaseUID)
	if raw, err := s.d.Redis.Get(ctx, key).Bytes(); err == nil {
		var c cachedPrincipal
		if json.Unmarshal(raw, &c) == nil {
			return c.principal(firebaseUID), nil
		}
	} else if !errors.Is(err, redis.ErrNil) {
		s.d.Logger.WarnContext(ctx, "principal cache read failed", "error", err)
	}

	q := sqlc.New(s.d.DB)
	u, err := q.GetUserByFirebaseUID(ctx, firebaseUID)
	if errors.Is(err, pgx.ErrNoRows) {
		// Signed in with Firebase but never synced: the app must call POST /v1/session first.
		return domain.Principal{}, apierror.New(apierror.Unauthenticated).With("reason", "session_required")
	}
	if err != nil {
		return domain.Principal{}, err
	}
	roles, err := q.ListUserRoles(ctx, u.ID)
	if err != nil {
		return domain.Principal{}, err
	}
	c := cachedPrincipal{UserID: u.ID, Tier: domain.Tier(u.AccountTier), Status: domain.Status(u.Status),
		DeletionRequestedAt: u.DeletionRequestedAt, Roles: roles}
	if raw, err := json.Marshal(c); err == nil {
		if err := s.d.Redis.Set(ctx, key, raw, principalTTL).Err(); err != nil {
			s.d.Logger.WarnContext(ctx, "principal cache write failed", "error", err)
		}
	}
	return c.principal(firebaseUID), nil
}

func (c cachedPrincipal) principal(firebaseUID string) domain.Principal {
	return domain.Principal{UserID: c.UserID, FirebaseUID: firebaseUID, Tier: c.Tier, Status: c.Status,
		DeletionRequestedAt: c.DeletionRequestedAt, Roles: c.Roles}
}

// forget drops the cached principal after a change; on failure the entry
// still expires within principalTTL.
func (s *Service) forget(ctx context.Context, firebaseUID string) {
	if err := s.d.Redis.Del(context.WithoutCancel(ctx), principalKey(firebaseUID)).Err(); err != nil {
		s.d.Logger.WarnContext(ctx, "principal cache invalidation failed", "error", err)
	}
}

// SetRole grants (grant = true) or revokes a staff role. user is a
// BrainVolt user id or a Firebase UID. Used by cmd/admin; the principal
// cache is cleared so the change applies to the next request.
func (s *Service) SetRole(ctx context.Context, user, role string, grant bool) (uuid.UUID, error) {
	if !domain.ValidRole(role) {
		return uuid.Nil, fmt.Errorf("unknown role %q (want %s or %s)", role, domain.RoleModerator, domain.RoleAdmin)
	}
	q := sqlc.New(s.d.DB)
	var u sqlc.IdentityUser
	var err error
	if id, perr := uuid.Parse(user); perr == nil {
		u, err = q.GetUserByID(ctx, id)
	} else {
		u, err = q.GetUserByFirebaseUID(ctx, user)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, fmt.Errorf("no BrainVolt user %q (sign in with the app once first)", user)
	}
	if err != nil {
		return uuid.Nil, err
	}
	if grant {
		err = q.GrantRole(ctx, sqlc.GrantRoleParams{UserID: u.ID, Role: role, GrantedAt: s.d.Clock.Now().UTC()})
	} else {
		err = q.RevokeRole(ctx, sqlc.RevokeRoleParams{UserID: u.ID, Role: role})
	}
	if err != nil {
		return uuid.Nil, err
	}
	s.forget(ctx, u.FirebaseUid)
	return u.ID, nil
}

// AbandonGuest records that a guest gave up its account in a link conflict
// (D-015). Repeating it is harmless.
func (s *Service) AbandonGuest(ctx context.Context, p domain.Principal) error {
	if p.Tier != domain.TierGuest {
		return apierror.New(apierror.Forbidden).With("reason", "not_a_guest")
	}
	now := s.d.Clock.Now().UTC()
	err := postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		n, err := sqlc.New(tx).MarkGuestAbandoned(ctx, sqlc.MarkGuestAbandonedParams{ID: p.UserID, Now: now})
		if err != nil || n == 0 {
			return err // already abandoned: nothing to publish
		}
		return s.d.Events.PublishTx(ctx, tx, domain.EventGuestAbandoned, now, domain.GuestAbandoned{UserID: p.UserID})
	})
	s.forget(ctx, p.FirebaseUID)
	return err
}

// Deletion is the state of an account deletion request.
type Deletion struct {
	RequestedAt time.Time `json:"requested_at"`
	PurgeAfter  time.Time `json:"purge_after"`
}

// RequestDeletion starts the 14-day cooling-off (D-022). A repeated request
// keeps the original date.
func (s *Service) RequestDeletion(ctx context.Context, p domain.Principal) (Deletion, error) {
	at := s.d.Clock.Now().UTC()
	if p.DeletionRequestedAt != nil {
		at = *p.DeletionRequestedAt
	} else if err := sqlc.New(s.d.DB).SetDeletionRequested(ctx, sqlc.SetDeletionRequestedParams{
		ID: p.UserID, DeletionRequestedAt: &at, Now: at,
	}); err != nil {
		return Deletion{}, err
	}
	s.forget(ctx, p.FirebaseUID)
	return Deletion{RequestedAt: at, PurgeAfter: at.Add(domain.DeletionGracePeriod)}, nil
}

// CancelDeletion ends a pending deletion request.
func (s *Service) CancelDeletion(ctx context.Context, p domain.Principal) error {
	if err := sqlc.New(s.d.DB).SetDeletionRequested(ctx, sqlc.SetDeletionRequestedParams{
		ID: p.UserID, Now: s.d.Clock.Now().UTC(),
	}); err != nil {
		return err
	}
	s.forget(ctx, p.FirebaseUID)
	return nil
}
