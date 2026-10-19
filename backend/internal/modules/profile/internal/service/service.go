// Package service holds profile's use cases: creating the profile of a new
// user (inside identity's transaction), reading and editing the own profile
// and settings, and the public profile.
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
)

// maxUsernameAttempts bounds retries when a generated username is taken.
const maxUsernameAttempts = 10

// Service implements profile's use cases.
type Service struct {
	db    *pgxpool.Pool
	clock clock.Clock
}

// New returns a service.
func New(db *pgxpool.Pool, c clock.Clock) *Service { return &Service{db: db, clock: c} }

var _ identity.ProfileProvisioner = (*Service)(nil)

// CreateTx creates the profile and settings of a new user inside identity's
// first-session transaction (schema-identity §3 rules).
func (s *Service) CreateTx(ctx context.Context, tx pgx.Tx, p identity.NewProfile) error {
	q := sqlc.New(tx)
	country := p.CountryCode
	if country != nil && !domain.ValidCountry(*country) {
		country = nil
	}
	name := p.DisplayName
	if name == "" {
		name = domain.GeneratedDisplayName(p.Locale)
	}
	for attempt := range maxUsernameAttempts {
		n, err := q.InsertProfile(ctx, sqlc.InsertProfileParams{
			UserID: p.UserID, Username: domain.GeneratedUsername(attempt),
			DisplayName: name, AvatarID: domain.RandomAvatar(),
			CountryCode: country, Locale: p.Locale, Now: p.Now,
		})
		if err != nil {
			return err
		}
		if n == 1 {
			return q.InsertSettings(ctx, sqlc.InsertSettingsParams{UserID: p.UserID, Now: p.Now})
		}
	}
	return fmt.Errorf("profile: no free generated username after %d attempts", maxUsernameAttempts)
}

// Summary implements identity.ProfileProvisioner.
func (s *Service) Summary(ctx context.Context, userID uuid.UUID) (identity.ProfileSummary, error) {
	p, err := s.get(ctx, userID)
	if err != nil {
		return identity.ProfileSummary{}, err
	}
	return identity.ProfileSummary{Username: p.Username, DisplayName: p.DisplayName, AvatarID: p.AvatarID, CountryCode: p.CountryCode, Locale: p.Locale}, nil
}

func (s *Service) get(ctx context.Context, userID uuid.UUID) (sqlc.ProfileProfile, error) {
	p, err := sqlc.New(s.db).GetProfile(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, apierror.New(apierror.NotFound)
	}
	return p, err
}

// OwnProfile is the caller's profile with the change limits.
type OwnProfile struct {
	Username                  string     `json:"username"`
	DisplayName               string     `json:"display_name"`
	AvatarID                  string     `json:"avatar_id"`
	CountryCode               *string    `json:"country_code"`
	Locale                    string     `json:"locale"`
	UsernameChangeAvailableAt *time.Time `json:"username_change_available_at"`
	CountryChangeAvailableAt  *time.Time `json:"country_change_available_at"`
}

func ownProfile(p sqlc.ProfileProfile) OwnProfile {
	return OwnProfile{
		Username: p.Username, DisplayName: p.DisplayName, AvatarID: p.AvatarID,
		CountryCode: p.CountryCode, Locale: p.Locale,
		UsernameChangeAvailableAt: domain.NextChange(p.UsernameChangedAt),
		CountryChangeAvailableAt:  domain.NextChange(p.CountryChangedAt),
	}
}

// Own returns the caller's profile.
func (s *Service) Own(ctx context.Context, userID uuid.UUID) (OwnProfile, error) {
	p, err := s.get(ctx, userID)
	if err != nil {
		return OwnProfile{}, err
	}
	return ownProfile(p), nil
}

// ProfilePatch is a validated PATCH /v1/me/profile; nil fields stay.
type ProfilePatch struct {
	Username    *string
	DisplayName *string
	AvatarID    *string
	CountryCode *string
	Locale      *string
}

// UpdateOwn applies a patch. Username: full accounts only (checked by the
// caller), once per 30 days; country: once per 30 days.
func (s *Service) UpdateOwn(ctx context.Context, userID uuid.UUID, patch ProfilePatch) (OwnProfile, error) {
	now := s.clock.Now().UTC()
	var out sqlc.ProfileProfile
	err := postgres.InTx(ctx, s.db, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		p, err := q.GetProfileForUpdate(ctx, userID)
		if errors.Is(err, pgx.ErrNoRows) {
			return apierror.New(apierror.NotFound)
		}
		if err != nil {
			return err
		}
		if patch.Username != nil && *patch.Username != p.Username {
			if !domain.CanChange(p.UsernameChangedAt, now) {
				return apierror.New(apierror.ChangeTooSoon).With("field", "username").With("available_at", domain.NextChange(p.UsernameChangedAt))
			}
			p.Username, p.UsernameChangedAt = *patch.Username, &now
		}
		if patch.CountryCode != nil && (p.CountryCode == nil || *patch.CountryCode != *p.CountryCode) {
			if !domain.CanChange(p.CountryChangedAt, now) {
				return apierror.New(apierror.ChangeTooSoon).With("field", "country_code").With("available_at", domain.NextChange(p.CountryChangedAt))
			}
			p.CountryCode, p.CountryChangedAt = patch.CountryCode, &now
		}
		if patch.DisplayName != nil {
			p.DisplayName = *patch.DisplayName
		}
		if patch.AvatarID != nil {
			p.AvatarID = *patch.AvatarID
		}
		if patch.Locale != nil {
			p.Locale = *patch.Locale
		}
		err = q.UpdateProfile(ctx, sqlc.UpdateProfileParams{
			UserID: userID, Username: p.Username, DisplayName: p.DisplayName, AvatarID: p.AvatarID,
			CountryCode: p.CountryCode, Locale: p.Locale,
			UsernameChangedAt: p.UsernameChangedAt, CountryChangedAt: p.CountryChangedAt, Now: now,
		})
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return apierror.New(apierror.UsernameTaken)
		}
		out = p
		return err
	})
	if err != nil {
		return OwnProfile{}, err
	}
	return ownProfile(out), nil
}

// Settings are the caller's privacy and app preferences.
type Settings struct {
	ChallengesFrom      string `json:"challenges_from"`
	ShowOnlineStatus    bool   `json:"show_online_status"`
	AllowFriendRequests bool   `json:"allow_friend_requests"`
	SoundEnabled        bool   `json:"sound_enabled"`
	MusicEnabled        bool   `json:"music_enabled"`
	HapticsEnabled      bool   `json:"haptics_enabled"`
}

// SettingsPatch is a validated PATCH /v1/me/settings; nil fields stay.
type SettingsPatch struct {
	ChallengesFrom      *string
	ShowOnlineStatus    *bool
	AllowFriendRequests *bool
	SoundEnabled        *bool
	MusicEnabled        *bool
	HapticsEnabled      *bool
}

// Settings returns the caller's settings.
func (s *Service) Settings(ctx context.Context, userID uuid.UUID) (Settings, error) {
	st, err := sqlc.New(s.db).GetSettings(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Settings{}, apierror.New(apierror.NotFound)
	}
	if err != nil {
		return Settings{}, err
	}
	return Settings{ChallengesFrom: st.ChallengesFrom, ShowOnlineStatus: st.ShowOnlineStatus, AllowFriendRequests: st.AllowFriendRequests,
		SoundEnabled: st.SoundEnabled, MusicEnabled: st.MusicEnabled, HapticsEnabled: st.HapticsEnabled}, nil
}

// UpdateSettings applies a patch.
func (s *Service) UpdateSettings(ctx context.Context, userID uuid.UUID, patch SettingsPatch) (Settings, error) {
	cur, err := s.Settings(ctx, userID)
	if err != nil {
		return Settings{}, err
	}
	set := func(dst *bool, v *bool) {
		if v != nil {
			*dst = *v
		}
	}
	if patch.ChallengesFrom != nil {
		cur.ChallengesFrom = *patch.ChallengesFrom
	}
	set(&cur.ShowOnlineStatus, patch.ShowOnlineStatus)
	set(&cur.AllowFriendRequests, patch.AllowFriendRequests)
	set(&cur.SoundEnabled, patch.SoundEnabled)
	set(&cur.MusicEnabled, patch.MusicEnabled)
	set(&cur.HapticsEnabled, patch.HapticsEnabled)
	err = sqlc.New(s.db).UpdateSettings(ctx, sqlc.UpdateSettingsParams{
		UserID: userID, ChallengesFrom: cur.ChallengesFrom, ShowOnlineStatus: cur.ShowOnlineStatus,
		AllowFriendRequests: cur.AllowFriendRequests, SoundEnabled: cur.SoundEnabled,
		MusicEnabled: cur.MusicEnabled, HapticsEnabled: cur.HapticsEnabled, Now: s.clock.Now().UTC(),
	})
	return cur, err
}

// PublicProfile is what any signed-in user may see (never the email).
type PublicProfile struct {
	UserID      uuid.UUID `json:"user_id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	AvatarID    string    `json:"avatar_id"`
	CountryCode *string   `json:"country_code"`
}

// Public returns another user's public profile.
func (s *Service) Public(ctx context.Context, userID uuid.UUID) (PublicProfile, error) {
	p, err := s.get(ctx, userID)
	if err != nil {
		return PublicProfile{}, err
	}
	return PublicProfile{UserID: p.UserID, Username: p.Username, DisplayName: p.DisplayName, AvatarID: p.AvatarID, CountryCode: p.CountryCode}, nil
}

// Card is the public face of a player in matches and lists.
type Card struct {
	UserID      uuid.UUID `json:"user_id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	AvatarID    string    `json:"avatar_id"`
	CountryCode *string   `json:"country_code"`
}

// Cards returns the public cards of users (missing users are left out).
func (s *Service) Cards(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]Card, error) {
	rows, err := sqlc.New(s.db).PublicCards(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]Card, len(rows))
	for _, r := range rows {
		out[r.UserID] = Card{UserID: r.UserID, Username: r.Username, DisplayName: r.DisplayName, AvatarID: r.AvatarID, CountryCode: r.CountryCode}
	}
	return out, nil
}

// UserIDByUsername finds a user by username (case-insensitive; NOT_FOUND).
func (s *Service) UserIDByUsername(ctx context.Context, username string) (uuid.UUID, error) {
	id, err := sqlc.New(s.db).UserIDByUsername(ctx, strings.ToLower(strings.TrimPrefix(strings.TrimSpace(username), "@")))
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, apierror.New(apierror.NotFound)
	}
	return id, err
}

// Privacy is the part of a user's settings other modules enforce.
type Privacy struct {
	ChallengesFrom      string // everyone | friends | nobody
	ShowOnlineStatus    bool
	AllowFriendRequests bool
}

// PrivacyOf returns the privacy settings of users (defaults for missing ones).
func (s *Service) PrivacyOf(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]Privacy, error) {
	rows, err := sqlc.New(s.db).PrivacyOf(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]Privacy, len(ids))
	for _, id := range ids {
		out[id] = Privacy{ChallengesFrom: "everyone", ShowOnlineStatus: true, AllowFriendRequests: true}
	}
	for _, r := range rows {
		out[r.UserID] = Privacy{ChallengesFrom: r.ChallengesFrom, ShowOnlineStatus: r.ShowOnlineStatus, AllowFriendRequests: r.AllowFriendRequests}
	}
	return out, nil
}
