package firebase

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/testenv"
)

// projectID is the emulator-only demo project (ADR-0003).
const projectID = "demo-brainvolt"

// signUp creates an emulator user through the Identity Toolkit REST API, the
// same call the app's Firebase SDK makes, and returns its ID token.
func signUp(t *testing.T, host string, body map[string]any) string {
	t.Helper()
	body["returnSecureToken"] = true
	b, _ := json.Marshal(body)
	url := fmt.Sprintf("http://%s/identitytoolkit.googleapis.com/v1/accounts:signUp?key=fake-api-key", host)
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out struct {
		IDToken string `json:"idToken"`
	}
	if err := json.Unmarshal(raw, &out); err != nil || out.IDToken == "" {
		t.Fatalf("signUp: status %d: %s", resp.StatusCode, raw)
	}
	return out.IDToken
}

// unsignedToken builds an emulator-style token (alg "none") with claims.
func unsignedToken(claims map[string]any) string {
	enc := func(v any) string {
		b, _ := json.Marshal(v)
		return base64.RawURLEncoding.EncodeToString(b)
	}
	return enc(map[string]string{"alg": "none", "typ": "JWT"}) + "." + enc(claims) + "."
}

func newVerifier(t *testing.T) (*Verifier, string) {
	t.Helper()
	host := testenv.FirebaseEmulatorHost(t)
	t.Setenv(emulatorEnv, "") // New must set it from its argument
	v, err := New(t.Context(), projectID, host)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return v, host
}

func code(err error) apierror.Code {
	if e := apierror.From(err); e != nil {
		return e.Code
	}
	return ""
}

func TestVerifyAnonymousAndPasswordUsers(t *testing.T) {
	v, host := newVerifier(t)
	ctx := t.Context()

	tok, err := v.Verify(ctx, signUp(t, host, map[string]any{}))
	if err != nil {
		t.Fatalf("anonymous: %v", err)
	}
	if tok.UID == "" || tok.Provider != "anonymous" || tok.Email != "" {
		t.Fatalf("anonymous token = %+v", tok)
	}
	if d := tok.Expires.Sub(tok.IssuedAt); d != time.Hour {
		t.Fatalf("token lifetime = %v, want 1h", d)
	}

	email := "u" + strings.ReplaceAll(ids.NewString(), "-", "") + "@example.com"
	tok, err = v.VerifyAndCheckRevoked(ctx, signUp(t, host, map[string]any{"email": email, "password": "secret-123"}))
	if err != nil {
		t.Fatalf("password: %v", err)
	}
	if tok.Provider != "password" || tok.Email != email || tok.EmailVerified {
		t.Fatalf("password token = %+v", tok)
	}
	if got := tok.Identities["email"]; len(got) != 1 || got[0] != email {
		t.Fatalf("identities = %v", tok.Identities)
	}

	acc, err := v.Account(ctx, tok.UID)
	if err != nil {
		t.Fatalf("Account: %v", err)
	}
	if acc.Email != email || acc.EmailVerified || len(acc.Providers) != 1 || acc.Providers[0].ProviderID != "password" {
		t.Fatalf("account = %+v", acc)
	}
	if _, err := v.Account(ctx, "no-such-user"); code(err) != apierror.Unauthenticated {
		t.Fatalf("missing account: %v", err)
	}
}

func TestVerifyRejects(t *testing.T) {
	v, host := newVerifier(t)
	// In emulator mode the SDK also requires the user to exist.
	existing, err := v.Verify(t.Context(), signUp(t, host, map[string]any{}))
	if err != nil {
		t.Fatal(err)
	}
	uid := existing.UID
	now := time.Now()
	base := func() map[string]any {
		return map[string]any{
			"iss": "https://securetoken.google.com/" + projectID, "aud": projectID,
			"sub": uid, "user_id": uid,
			"iat": now.Add(-time.Minute).Unix(), "auth_time": now.Add(-time.Minute).Unix(), "exp": now.Add(time.Hour).Unix(),
			"firebase": map[string]any{"sign_in_provider": "anonymous", "identities": map[string]any{}},
		}
	}

	if _, err := v.Verify(t.Context(), unsignedToken(base())); err != nil {
		t.Fatalf("hand-made valid emulator token rejected: %v", err)
	}

	expired := base()
	expired["iat"], expired["exp"] = now.Add(-2*time.Hour).Unix(), now.Add(-time.Hour).Unix()
	otherProject := base()
	otherProject["aud"], otherProject["iss"] = "someone-else", "https://securetoken.google.com/someone-else"
	noSubject := base()
	delete(noSubject, "sub")
	unknownUser := base()
	unknownUser["sub"], unknownUser["user_id"] = "deleted-user", "deleted-user"

	tests := map[string]struct {
		token string
		want  apierror.Code
	}{
		"expired":       {unsignedToken(expired), apierror.TokenExpired},
		"other project": {unsignedToken(otherProject), apierror.Unauthenticated},
		"no subject":    {unsignedToken(noSubject), apierror.Unauthenticated},
		"unknown user":  {unsignedToken(unknownUser), apierror.Unauthenticated},
		"garbage":       {"not.a.token", apierror.Unauthenticated},
		"empty":         {"", apierror.Unauthenticated},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := v.Verify(t.Context(), tc.token); code(err) != tc.want {
				t.Fatalf("err = %v, want %s", err, tc.want)
			}
		})
	}
}

func TestBearerToken(t *testing.T) {
	for header, want := range map[string]string{"Bearer abc.def": "abc.def", "bearer   xyz ": "xyz"} {
		r := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
		r.Header.Set("Authorization", header)
		if got, err := BearerToken(r); err != nil || got != want {
			t.Errorf("BearerToken(%q) = %q, %v", header, got, err)
		}
	}
	for _, header := range []string{"", "Bearer", "Bearer  ", "Basic abc", "Token abc"} {
		r := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
		r.Header.Set("Authorization", header)
		if _, err := BearerToken(r); code(err) != apierror.Unauthenticated {
			t.Errorf("BearerToken(%q) err = %v, want UNAUTHENTICATED", header, err)
		}
	}
}
