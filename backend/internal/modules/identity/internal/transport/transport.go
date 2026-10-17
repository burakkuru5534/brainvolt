// Package transport is identity's HTTP layer: the authentication wrapper used
// by every module, POST /v1/session, the link-conflict exit and account
// deletion requests (docs/api/rest.md §1).
package transport

import (
	"context"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity/internal/service"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/firebase"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/idempotency"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/logging"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ratelimit"
)

// Transport wires identity's handlers.
type Transport struct {
	svc      *service.Service
	verifier service.Verifier
	limiter  *ratelimit.Limiter
	idem     *idempotency.Store
	logger   *slog.Logger
}

// New returns the transport.
func New(svc *service.Service, verifier service.Verifier, limiter *ratelimit.Limiter, idem *idempotency.Store, logger *slog.Logger) *Transport {
	return &Transport{svc: svc, verifier: verifier, limiter: limiter, idem: idem, logger: logger}
}

type principalKey struct{}

// PrincipalFrom returns the authenticated caller stored by Auth.
func PrincipalFrom(ctx context.Context) (domain.Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(domain.Principal)
	return p, ok
}

// AuthOption relaxes or tightens Auth for one route.
type AuthOption func(*authOptions)

type authOptions struct {
	allowSuspended       bool
	allowDeletionPending bool
	fullAction           string
}

// AllowSuspended lets suspended accounts through (own profile, sanction
// notice, appeals, deletion — schema-identity §5.5).
func AllowSuspended() AuthOption { return func(o *authOptions) { o.allowSuspended = true } }

// AllowDeletionPending lets accounts in the deletion cooling-off through
// (GET /v1/me, cancelling the deletion).
func AllowDeletionPending() AuthOption { return func(o *authOptions) { o.allowDeletionPending = true } }

// RequireFull rejects guest and unverified accounts with
// ACCOUNT_UPGRADE_REQUIRED, details.action = action (D-016, D-017).
func RequireFull(action string) AuthOption { return func(o *authOptions) { o.fullAction = action } }

// Auth verifies the Bearer token, resolves the BrainVolt account, applies
// the status/tier rules and the per-user rate limit, then calls h with the
// principal in the context.
func (t *Transport) Auth(h httpserver.HandlerFunc, opts ...AuthOption) httpserver.HandlerFunc {
	var o authOptions
	for _, opt := range opts {
		opt(&o)
	}
	return func(w http.ResponseWriter, r *http.Request) error {
		raw, err := firebase.BearerToken(r)
		if err != nil {
			return err
		}
		ctx := r.Context()
		tok, err := t.verifier.Verify(ctx, raw)
		if err != nil {
			return err
		}
		p, err := t.svc.Principal(ctx, tok)
		if err != nil {
			return err
		}
		ctx = logging.WithUserID(ctx, p.UserID.String())
		if err := checkAccess(p, o); err != nil {
			return err
		}
		if err := t.limiter.Check(ctx, ratelimit.APIPerUser, p.UserID.String()); err != nil {
			return err
		}
		return h(w, r.WithContext(context.WithValue(ctx, principalKey{}, p)))
	}
}

// checkAccess applies the status and tier rules of one route.
func checkAccess(p domain.Principal, o authOptions) error {
	if p.Status == domain.StatusSuspended && !o.allowSuspended {
		return apierror.New(apierror.AccountSuspended)
	}
	if p.DeletionRequestedAt != nil && !o.allowDeletionPending {
		return apierror.New(apierror.AccountDeletionPending)
	}
	if o.fullAction != "" && p.Tier.Restricted() {
		return apierror.New(apierror.AccountUpgradeRequired).With("action", o.fullAction)
	}
	return nil
}

// Register adds identity's routes.
func (t *Transport) Register(s *httpserver.Server) {
	s.Handle("POST /v1/session", t.session, httpserver.WithDoc(httpserver.Doc{
		Tag: "identity", Auth: "any",
		Summary: "Sync the account after sign-in, link or email verification; creates it on the first call",
		Request: SessionRequest{}, Response: SessionResponse{}, Status: http.StatusOK,
		Errors: []apierror.Code{apierror.AgeConfirmationRequired, apierror.ValidationFailed, apierror.Unauthenticated, apierror.TokenExpired, apierror.RateLimited},
	}))
	s.Handle("POST /v1/session/abandon-guest", t.Auth(t.idem.Wrap(idempotency.Required, t.abandonGuest)), httpserver.WithDoc(httpserver.Doc{
		Tag: "identity", Auth: "any", Idempotency: true,
		Summary: "Link conflict (D-015): give up this guest account before switching to the existing one",
		Errors:  []apierror.Code{apierror.Forbidden},
	}))
	s.Handle("POST /v1/me/deletion", t.Auth(t.idem.Wrap(idempotency.Required, t.requestDeletion), AllowSuspended(), AllowDeletionPending()), httpserver.WithDoc(httpserver.Doc{
		Tag: "identity", Auth: "any", Idempotency: true,
		Summary:  "Request account deletion (14-day cooling-off, D-022)",
		Response: service.Deletion{},
	}))
	s.Handle("DELETE /v1/me/deletion", t.Auth(t.cancelDeletion, AllowSuspended(), AllowDeletionPending()), httpserver.WithDoc(httpserver.Doc{
		Tag: "identity", Auth: "any",
		Summary: "Cancel a pending account deletion",
	}))
}

// SessionRequest is the body of POST /v1/session.
type SessionRequest struct {
	AgeConfirmed   *bool   `json:"age_confirmed"`
	InstallationID string  `json:"installation_id"`
	Platform       string  `json:"platform"`
	AppVersion     string  `json:"app_version"`
	Locale         string  `json:"locale"`
	CountryHint    *string `json:"country_hint,omitempty"`
}

// SessionResponse is the body returned by POST /v1/session.
type SessionResponse struct {
	User    service.User          `json:"user"`
	Profile domain.ProfileSummary `json:"profile"`
	Tier    domain.Tier           `json:"tier"`
	Status  domain.Status         `json:"status"`
	Flags   SessionFlags          `json:"flags"`
}

// SessionFlags tell the app what to show next (D-059).
type SessionFlags struct {
	Created                   bool `json:"created"`
	Restricted                bool `json:"restricted"`
	EmailVerificationRequired bool `json:"email_verification_required"`
	DeletionPending           bool `json:"deletion_pending"`
	Suspended                 bool `json:"suspended"`
}

var (
	countryCode = regexp.MustCompile(`^[A-Z]{2}$`)
	appVersion  = regexp.MustCompile(`^[0-9A-Za-z.+_-]{1,32}$`)
)

func (r SessionRequest) validate() (service.SessionInput, error) {
	if r.AgeConfirmed == nil || !*r.AgeConfirmed {
		return service.SessionInput{}, apierror.New(apierror.AgeConfirmationRequired)
	}
	var f apierror.Fields
	if _, err := ids.Parse(r.InstallationID); err != nil {
		f.Add("installation_id", "uuid")
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
	var hint *string
	if r.CountryHint != nil {
		c := strings.ToUpper(strings.TrimSpace(*r.CountryHint))
		if !countryCode.MatchString(c) {
			f.Add("country_hint", "iso3166_alpha2")
		}
		hint = &c
	}
	return service.SessionInput{Locale: r.Locale, CountryHint: hint}, f.Err()
}

func (t *Transport) session(w http.ResponseWriter, r *http.Request) error {
	raw, err := firebase.BearerToken(r)
	if err != nil {
		return err
	}
	ctx := r.Context()
	// Sensitive endpoint: also reject revoked tokens and disabled users (conventions.md §3).
	tok, err := t.verifier.VerifyAndCheckRevoked(ctx, raw)
	if err != nil {
		return err
	}
	var req SessionRequest
	if err := httpserver.DecodeJSON(r, &req); err != nil {
		return err
	}
	in, err := req.validate()
	if err != nil {
		return err
	}
	if err := t.limiter.Check(ctx, ratelimit.Session, req.InstallationID); err != nil {
		return err
	}

	res, err := t.svc.Session(ctx, tok, in)
	if err != nil {
		return err
	}
	u := res.User
	return httpserver.WriteJSON(w, http.StatusOK, SessionResponse{
		User: u, Profile: res.Profile, Tier: u.Tier, Status: u.Status,
		Flags: SessionFlags{
			Created:                   res.Created,
			Restricted:                u.Tier.Restricted(),
			EmailVerificationRequired: u.Tier == domain.TierUnverified,
			DeletionPending:           u.DeletionRequestedAt != nil,
			Suspended:                 u.Status == domain.StatusSuspended,
		},
	})
}

func (t *Transport) abandonGuest(w http.ResponseWriter, r *http.Request) error {
	p, _ := PrincipalFrom(r.Context())
	if err := t.svc.AbandonGuest(r.Context(), p); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (t *Transport) requestDeletion(w http.ResponseWriter, r *http.Request) error {
	p, _ := PrincipalFrom(r.Context())
	d, err := t.svc.RequestDeletion(r.Context(), p)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, d)
}

func (t *Transport) cancelDeletion(w http.ResponseWriter, r *http.Request) error {
	p, _ := PrincipalFrom(r.Context())
	if err := t.svc.CancelDeletion(r.Context(), p); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
