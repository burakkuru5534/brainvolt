// Package transport is profile's HTTP layer (docs/api/rest.md §1–§2):
// GET /v1/me, PATCH /v1/me/profile, GET/PATCH /v1/me/settings and
// GET /v1/profiles/{user_id}.
package transport

import (
	"net/http"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile/internal/service"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ratelimit"
)

// ProfileEdits limits PATCH /v1/me/profile (limits.md §1, D-052).
var ProfileEdits = ratelimit.Limit{Name: "profile_edit", Max: 10, Window: time.Hour}

// Transport wires profile's handlers.
type Transport struct {
	svc     *service.Service
	ident   *identity.Module
	limiter *ratelimit.Limiter
}

// New returns the transport.
func New(svc *service.Service, ident *identity.Module, limiter *ratelimit.Limiter) *Transport {
	return &Transport{svc: svc, ident: ident, limiter: limiter}
}

// Me is the body of GET /v1/me. Sanctions (Phase 8) and entitlements
// (Phase 9) are added by their modules.
type Me struct {
	User     identity.User      `json:"user"`
	Profile  service.OwnProfile `json:"profile"`
	Settings service.Settings   `json:"settings"`
}

// ProfileRequest is the body of PATCH /v1/me/profile.
type ProfileRequest struct {
	Username    *string `json:"username,omitempty"`
	DisplayName *string `json:"display_name,omitempty"`
	AvatarID    *string `json:"avatar_id,omitempty"`
	CountryCode *string `json:"country_code,omitempty"`
	Locale      *string `json:"locale,omitempty"`
}

// SettingsRequest is the body of PATCH /v1/me/settings.
type SettingsRequest struct {
	ChallengesFrom      *string `json:"challenges_from,omitempty"`
	ShowOnlineStatus    *bool   `json:"show_online_status,omitempty"`
	AllowFriendRequests *bool   `json:"allow_friend_requests,omitempty"`
	SoundEnabled        *bool   `json:"sound_enabled,omitempty"`
	MusicEnabled        *bool   `json:"music_enabled,omitempty"`
	HapticsEnabled      *bool   `json:"haptics_enabled,omitempty"`
	ProfilePrivate      *bool   `json:"profile_private,omitempty"`
}

// Register adds profile's routes.
func (t *Transport) Register(s *httpserver.Server) {
	a := t.ident.Auth
	s.Handle("GET /v1/me", a(t.me, identity.AllowSuspended(), identity.AllowDeletionPending()), httpserver.WithDoc(httpserver.Doc{
		Tag: "profile", Auth: "any", Summary: "Own user, profile and settings", Response: Me{},
	}))
	s.Handle("PATCH /v1/me/profile", a(t.patchProfile), httpserver.WithDoc(httpserver.Doc{
		Tag: "profile", Auth: "any", Summary: "Edit own profile (username: full accounts, once per 30 days; country: once per 30 days)",
		Request: ProfileRequest{}, Response: service.OwnProfile{},
		Errors: []apierror.Code{apierror.ValidationFailed, apierror.TextRejected, apierror.AccountUpgradeRequired, apierror.ChangeTooSoon,
			apierror.UsernameTaken, apierror.AccountSuspended, apierror.AccountDeletionPending},
	}))
	s.Handle("GET /v1/me/settings", a(t.getSettings, identity.AllowSuspended()), httpserver.WithDoc(httpserver.Doc{
		Tag: "profile", Auth: "any", Summary: "Own privacy and app settings", Response: service.Settings{},
		Errors: []apierror.Code{apierror.AccountDeletionPending},
	}))
	s.Handle("PATCH /v1/me/settings", a(t.patchSettings), httpserver.WithDoc(httpserver.Doc{
		Tag: "profile", Auth: "any", Summary: "Edit own privacy and app settings",
		Request: SettingsRequest{}, Response: service.Settings{},
		Errors: []apierror.Code{apierror.ValidationFailed, apierror.AccountSuspended, apierror.AccountDeletionPending},
	}))
	s.Handle("GET /v1/profiles/{user_id}", a(t.publicProfile), httpserver.WithDoc(httpserver.Doc{
		Tag: "profile", Auth: "any", Summary: "Player profile (no email): card, relation, stats unless private (D-101)", Response: service.PublicProfile{},
		Errors: []apierror.Code{apierror.NotFound, apierror.AccountSuspended, apierror.AccountDeletionPending},
	}))
	s.Handle("GET /v1/profiles/by-username/{username}", a(t.byUsername), httpserver.WithDoc(httpserver.Doc{
		Tag: "profile", Auth: "any", Summary: "Public profile by username (friend search)", Response: service.PublicProfile{},
		Errors: []apierror.Code{apierror.NotFound, apierror.AccountSuspended, apierror.AccountDeletionPending},
	}))
}

func principal(r *http.Request) identity.Principal {
	p, _ := identity.PrincipalFrom(r.Context())
	return p
}

func (t *Transport) me(w http.ResponseWriter, r *http.Request) error {
	ctx, p := r.Context(), principal(r)
	user, err := t.ident.User(ctx, p.UserID)
	if err != nil {
		return err
	}
	prof, err := t.svc.Own(ctx, p.UserID)
	if err != nil {
		return err
	}
	settings, err := t.svc.Settings(ctx, p.UserID)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, Me{User: user, Profile: prof, Settings: settings})
}

func (t *Transport) patchProfile(w http.ResponseWriter, r *http.Request) error {
	ctx, p := r.Context(), principal(r)
	var req ProfileRequest
	if err := httpserver.DecodeJSON(r, &req); err != nil {
		return err
	}
	patch, err := validateProfile(req, p)
	if err != nil {
		return err
	}
	if err := t.limiter.Check(ctx, ProfileEdits, p.UserID.String()); err != nil {
		return err
	}
	out, err := t.svc.UpdateOwn(ctx, p.UserID, patch)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, out)
}

func validateProfile(req ProfileRequest, p identity.Principal) (service.ProfilePatch, error) {
	var (
		f     apierror.Fields
		patch service.ProfilePatch
	)
	rejected := func(field string) error {
		return apierror.New(apierror.TextRejected).With("field", field)
	}
	if req.DisplayName != nil {
		v := domain.NormalizeDisplayName(*req.DisplayName)
		switch {
		case !domain.ValidDisplayName(v):
			f.Add("display_name", "length", "min", 2, "max", 24)
		case domain.Rejected(v):
			return patch, rejected("display_name")
		}
		patch.DisplayName = &v
	}
	if req.Username != nil {
		if p.Tier.Restricted() {
			return patch, apierror.New(apierror.AccountUpgradeRequired).With("action", "profile.username")
		}
		v := domain.NormalizeUsername(*req.Username)
		switch {
		case !domain.ValidUsername(v):
			f.Add("username", "pattern", "pattern", "^[a-z0-9_.]{3,20}$")
		case domain.Rejected(v):
			return patch, rejected("username")
		}
		patch.Username = &v
	}
	if req.AvatarID != nil {
		if !domain.ValidAvatar(*req.AvatarID) {
			f.Add("avatar_id", "enum", "allowed", domain.Avatars)
		}
		patch.AvatarID = req.AvatarID
	}
	if req.CountryCode != nil {
		v := domain.NormalizeCountry(*req.CountryCode)
		if !domain.ValidCountry(v) {
			f.Add("country_code", "iso3166_alpha2")
		}
		patch.CountryCode = &v
	}
	if req.Locale != nil {
		if !slices.Contains(domain.Locales, *req.Locale) {
			f.Add("locale", "enum", "allowed", domain.Locales)
		}
		patch.Locale = req.Locale
	}
	return patch, f.Err()
}

func (t *Transport) getSettings(w http.ResponseWriter, r *http.Request) error {
	st, err := t.svc.Settings(r.Context(), principal(r).UserID)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, st)
}

func (t *Transport) patchSettings(w http.ResponseWriter, r *http.Request) error {
	var req SettingsRequest
	if err := httpserver.DecodeJSON(r, &req); err != nil {
		return err
	}
	if req.ChallengesFrom != nil && !slices.Contains(domain.ChallengesFrom, *req.ChallengesFrom) {
		var f apierror.Fields
		f.Add("challenges_from", "enum", "allowed", domain.ChallengesFrom)
		return f.Err()
	}
	st, err := t.svc.UpdateSettings(r.Context(), principal(r).UserID, service.SettingsPatch{
		ChallengesFrom: req.ChallengesFrom, ShowOnlineStatus: req.ShowOnlineStatus,
		AllowFriendRequests: req.AllowFriendRequests, SoundEnabled: req.SoundEnabled,
		MusicEnabled: req.MusicEnabled, HapticsEnabled: req.HapticsEnabled, ProfilePrivate: req.ProfilePrivate,
	})
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, st)
}

func (t *Transport) byUsername(w http.ResponseWriter, r *http.Request) error {
	id, err := t.svc.UserIDByUsername(r.Context(), r.PathValue("username"))
	if err != nil {
		return err
	}
	return t.writePublic(w, r, id)
}

func (t *Transport) publicProfile(w http.ResponseWriter, r *http.Request) error {
	id, err := ids.Parse(r.PathValue("user_id"))
	if err != nil {
		return apierror.New(apierror.NotFound)
	}
	return t.writePublic(w, r, id)
}

func (t *Transport) writePublic(w http.ResponseWriter, r *http.Request, id uuid.UUID) error {
	ctx := r.Context()
	// Deleted accounts are invisible (conventions.md §7: never leak existence).
	u, err := t.ident.User(ctx, id)
	if err != nil {
		return err
	}
	if u.Status == identity.StatusDeleted {
		return apierror.New(apierror.NotFound)
	}
	pub, err := t.svc.View(ctx, principal(r).UserID, id)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, pub)
}
