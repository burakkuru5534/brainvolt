// Package firebase verifies Firebase ID tokens (docs/api/conventions.md §3,
// ADR-0002: Firebase Admin SDK). It only turns a token into verified claims;
// mapping firebase_uid to a BrainVolt user is the identity module's job.
//
// Locally the backend talks to the Firebase Auth emulator (ADR-0003): when
// an emulator host is configured the SDK accepts the emulator's unsigned
// tokens and never contacts Google. platform/config forbids that in prod.
package firebase

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	fb "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"

	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
)

// emulatorEnv is the variable the Admin SDK itself reads.
const emulatorEnv = "FIREBASE_AUTH_EMULATOR_HOST"

// Token is a verified ID token reduced to what BrainVolt uses
// (schema-identity §1: provider, linked identities, email verification).
type Token struct {
	UID string
	// Provider is firebase.sign_in_provider: anonymous, google.com,
	// apple.com, password, …
	Provider      string
	Email         string
	EmailVerified bool
	// Identities maps a provider to the linked account ids (firebase.identities).
	Identities map[string][]string
	AuthTime   time.Time
	IssuedAt   time.Time
	Expires    time.Time
}

// Verifier checks ID tokens for one Firebase project.
type Verifier struct {
	client *auth.Client
}

// New returns a verifier for projectID. With emulatorHost set, tokens are
// verified against the emulator; otherwise against Google's public keys,
// with Application Default Credentials used for revocation checks.
func New(ctx context.Context, projectID, emulatorHost string) (*Verifier, error) {
	var opts []option.ClientOption
	if emulatorHost != "" {
		// The SDK only looks at the environment variable; set it so the
		// configured value is the one in effect.
		if err := os.Setenv(emulatorEnv, emulatorHost); err != nil {
			return nil, err
		}
		opts = append(opts, option.WithoutAuthentication())
	}
	app, err := fb.NewApp(ctx, &fb.Config{ProjectID: projectID}, opts...)
	if err != nil {
		return nil, fmt.Errorf("firebase: %w", err)
	}
	client, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("firebase auth: %w", err)
	}
	return &Verifier{client: client}, nil
}

// Verify checks signature, audience (project), issuer and expiry. Failures
// are *apierror.Error: TOKEN_EXPIRED (refresh and retry), UNAUTHENTICATED,
// or UNAVAILABLE when Google's signing keys cannot be fetched.
func (v *Verifier) Verify(ctx context.Context, raw string) (Token, error) {
	t, err := v.client.VerifyIDToken(ctx, raw)
	if err != nil {
		return Token{}, classify(err)
	}
	return convert(t), nil
}

// VerifyAndCheckRevoked is Verify plus a revocation/disabled-user check,
// used on sensitive endpoints (session, account deletion, purchases).
func (v *Verifier) VerifyAndCheckRevoked(ctx context.Context, raw string) (Token, error) {
	t, err := v.client.VerifyIDTokenAndCheckRevoked(ctx, raw)
	if err != nil {
		return Token{}, classify(err)
	}
	return convert(t), nil
}

func classify(err error) error {
	switch {
	case auth.IsIDTokenExpired(err):
		return apierror.Wrap(apierror.TokenExpired, err)
	case auth.IsCertificateFetchFailed(err):
		return apierror.Wrap(apierror.Unavailable, err).WithRetryAfter(5 * time.Second)
	default: // invalid, revoked, disabled user, wrong project …
		return apierror.Wrap(apierror.Unauthenticated, err)
	}
}

func convert(t *auth.Token) Token {
	out := Token{
		UID:        t.UID,
		Provider:   t.Firebase.SignInProvider,
		Identities: map[string][]string{},
		AuthTime:   time.Unix(t.AuthTime, 0).UTC(),
		IssuedAt:   time.Unix(t.IssuedAt, 0).UTC(),
		Expires:    time.Unix(t.Expires, 0).UTC(),
	}
	out.Email, _ = t.Claims["email"].(string)
	out.EmailVerified, _ = t.Claims["email_verified"].(bool)
	for provider, raw := range t.Firebase.Identities {
		list, _ := raw.([]any)
		for _, id := range list {
			if s, ok := id.(string); ok {
				out.Identities[provider] = append(out.Identities[provider], s)
			}
		}
	}
	return out
}

// BearerToken extracts the token from "Authorization: Bearer <token>".
func BearerToken(r *http.Request) (string, error) {
	h := r.Header.Get("Authorization")
	scheme, token, ok := strings.Cut(h, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
		return "", apierror.Wrap(apierror.Unauthenticated, errors.New("missing bearer token"))
	}
	return strings.TrimSpace(token), nil
}
