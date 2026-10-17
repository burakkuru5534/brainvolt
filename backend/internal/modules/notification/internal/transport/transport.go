// Package transport is the devices part of the notification module
// (docs/api/rest.md §1, schema-identity §4): POST /v1/me/devices and
// DELETE /v1/me/devices/{installation_id}. Push delivery itself is Phase 9.
package transport

import (
	"net/http"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/notification/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
)

// Transport wires the device handlers.
type Transport struct {
	db    *pgxpool.Pool
	clock clock.Clock
	ident *identity.Module
}

// New returns the transport.
func New(db *pgxpool.Pool, c clock.Clock, ident *identity.Module) *Transport {
	return &Transport{db: db, clock: c, ident: ident}
}

// DeviceRequest is the body of POST /v1/me/devices.
type DeviceRequest struct {
	InstallationID string  `json:"installation_id"`
	FcmToken       *string `json:"fcm_token,omitempty"`
	Platform       string  `json:"platform"`
	AppVersion     string  `json:"app_version"`
	Locale         string  `json:"locale"`
}

// Device is the registered device.
type Device struct {
	InstallationID string    `json:"installation_id"`
	Platform       string    `json:"platform"`
	PushEnabled    bool      `json:"push_enabled"`
	AppVersion     string    `json:"app_version"`
	Locale         string    `json:"locale"`
	LastSeenAt     time.Time `json:"last_seen_at"`
}

// Register adds the device routes.
func (t *Transport) Register(s *httpserver.Server) {
	// Suspended and deletion-pending users still sign in on devices.
	opts := []identity.AuthOption{identity.AllowSuspended(), identity.AllowDeletionPending()}
	s.Handle("POST /v1/me/devices", t.ident.Auth(t.register, opts...), httpserver.WithDoc(httpserver.Doc{
		Tag: "notification", Auth: "any",
		Summary: "Register or refresh this device and its FCM token; the device moves to the caller if another account used it",
		Request: DeviceRequest{}, Response: Device{}, Errors: []apierror.Code{apierror.ValidationFailed},
	}))
	s.Handle("DELETE /v1/me/devices/{installation_id}", t.ident.Auth(t.revoke, opts...), httpserver.WithDoc(httpserver.Doc{
		Tag: "notification", Auth: "any", Summary: "Sign out on this device (stops push); repeating it is harmless",
	}))
}

var (
	appVersion = regexp.MustCompile(`^[0-9A-Za-z.+_-]{1,32}$`)
	fcmToken   = regexp.MustCompile(`^[0-9A-Za-z_:.-]{20,}$`)
)

func (r DeviceRequest) validate() error {
	var f apierror.Fields
	if _, err := ids.Parse(r.InstallationID); err != nil {
		f.Add("installation_id", "uuid")
	}
	if r.FcmToken != nil && (len(*r.FcmToken) > 4096 || !fcmToken.MatchString(*r.FcmToken)) {
		f.Add("fcm_token", "format")
	}
	if r.Platform != "android" && r.Platform != "ios" {
		f.Add("platform", "enum", "allowed", []string{"android", "ios"})
	}
	if !appVersion.MatchString(r.AppVersion) {
		f.Add("app_version", "format")
	}
	if r.Locale != "tr" && r.Locale != "en" {
		f.Add("locale", "enum", "allowed", []string{"tr", "en"})
	}
	return f.Err()
}

func (t *Transport) register(w http.ResponseWriter, r *http.Request) error {
	var req DeviceRequest
	if err := httpserver.DecodeJSON(r, &req); err != nil {
		return err
	}
	if err := req.validate(); err != nil {
		return err
	}
	ctx := r.Context()
	p, _ := identity.PrincipalFrom(ctx)
	now := t.clock.Now().UTC()
	err := postgres.InTx(ctx, t.db, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		if req.FcmToken != nil {
			if err := q.ClearFcmTokenElsewhere(ctx, sqlc.ClearFcmTokenElsewhereParams{FcmToken: req.FcmToken, InstallationID: req.InstallationID}); err != nil {
				return err
			}
		}
		return q.UpsertDevice(ctx, sqlc.UpsertDeviceParams{
			ID: ids.New(), UserID: p.UserID, InstallationID: req.InstallationID, Platform: req.Platform,
			FcmToken: req.FcmToken, AppVersion: req.AppVersion, Locale: req.Locale, Now: now,
		})
	})
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, Device{
		InstallationID: req.InstallationID, Platform: req.Platform, PushEnabled: req.FcmToken != nil,
		AppVersion: req.AppVersion, Locale: req.Locale, LastSeenAt: now,
	})
}

func (t *Transport) revoke(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	p, _ := identity.PrincipalFrom(ctx)
	// Only the caller's own device is touched; anything else is a no-op, so
	// the answer never reveals other users' installations.
	if _, err := sqlc.New(t.db).RevokeDevice(ctx, sqlc.RevokeDeviceParams{
		UserID: p.UserID, InstallationID: r.PathValue("installation_id"), Now: t.clock.Now().UTC(),
	}); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
