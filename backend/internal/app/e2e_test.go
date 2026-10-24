package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/config"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/firebase"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/idempotency"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ratelimit"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/testdb"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/testenv"
)

const emulatorProject = "demo-brainvolt"

// emulator talks to the Firebase Auth emulator the way the app's SDK does.
type emulator struct {
	t    *testing.T
	host string
}

func (e emulator) call(path string, body map[string]any, admin bool) map[string]any {
	e.t.Helper()
	b, _ := json.Marshal(body)
	url := fmt.Sprintf("http://%s/identitytoolkit.googleapis.com/v1/%s?key=fake-api-key", e.host, path)
	req, _ := http.NewRequestWithContext(e.t.Context(), http.MethodPost, url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	if admin {
		req.Header.Set("Authorization", "Bearer owner") // emulator admin access
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		e.t.Fatalf("emulator %s: %d %s", path, resp.StatusCode, raw)
	}
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	return out
}

func (e emulator) signUpAnonymous() (token, uid string) {
	out := e.call("accounts:signUp", map[string]any{"returnSecureToken": true}, false)
	return out["idToken"].(string), out["localId"].(string)
}

// linkPassword links email+password to the signed-in (anonymous) user, as
// linkWithCredential does; Firebase keeps the same UID (D-013a).
func (e emulator) linkPassword(token, email string) string {
	out := e.call("accounts:update", map[string]any{"idToken": token, "email": email, "password": "secret-123", "returnSecureToken": true}, false)
	return out["idToken"].(string)
}

func (e emulator) verifyEmail(uid string) {
	e.call("projects/"+emulatorProject+"/accounts:update", map[string]any{"localId": uid, "emailVerified": true}, true)
}

func (e emulator) signIn(email string) string {
	out := e.call("accounts:signInWithPassword", map[string]any{"email": email, "password": "secret-123", "returnSecureToken": true}, false)
	return out["idToken"].(string)
}

// e2e is the API's module wiring on a throw-away database, with a test
// subscriber on every identity event so published events can be counted.
type e2e struct {
	t    *testing.T
	srv  *httpserver.Server
	db   *pgxpool.Pool
	emu  emulator
	mods modules
	// httpURL is the live test server started by url() for WebSocket tests.
	httpURL string
}

func newE2E(t *testing.T) *e2e {
	t.Helper()
	host := testenv.FirebaseEmulatorHost(t)
	db := testdb.New(t)
	verifier, err := firebase.New(t.Context(), emulatorProject, host)
	if err != nil {
		t.Fatal(err)
	}

	bus := newBus(db, discard, nil, nil)
	for _, ev := range []string{identity.EventUserRegistered, identity.EventAccountTierChanged, identity.EventGuestAbandoned} {
		bus.Subscribe(ev, "test.on_"+ev, func(context.Context, events.Envelope) error { return nil })
	}
	inserter, err := river.NewClient(riverpgxv5.New(db), &river.Config{Logger: discard})
	if err != nil {
		t.Fatal(err)
	}
	bus.Attach(inserter)

	cfg := config.HTTP{ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: time.Minute,
		HandlerTimeout: 10 * time.Second, ShutdownTimeout: time.Second}
	srv := httpserver.New(cfg, config.EnvDev, discard)
	mods := registerModules(srv, moduleDeps{db: db, verifier: verifier, bus: bus,
		limiter: ratelimit.New(clock.Real{}, discard), idem: idempotency.New(db, discard),
		clock: clock.Real{}, logger: discard, instance: "test-" + ids.NewString(), env: config.EnvDev})
	return &e2e{t: t, srv: srv, db: db, emu: emulator{t: t, host: host}, mods: mods}
}

type response struct {
	code int
	body map[string]any
	raw  string
}

func (r response) errCode() string {
	e, _ := r.body["error"].(map[string]any)
	s, _ := e["code"].(string)
	return s
}

// get walks a decoded JSON object by keys, e.g. r.get("profile", "username").
func (r response) get(keys ...string) any {
	var cur any = r.body
	for _, k := range keys {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = m[k]
	}
	return cur
}

func (x *e2e) do(method, path, token string, body any, headers ...string) response {
	x.t.Helper()
	var rd io.Reader = http.NoBody
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, rd)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept-Language", "en")
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	rec := httptest.NewRecorder()
	x.srv.Handler().ServeHTTP(rec, req)
	out := response{code: rec.Code, raw: rec.Body.String()}
	_ = json.Unmarshal(rec.Body.Bytes(), &out.body)
	return out
}

func (x *e2e) session(token string, extra map[string]any) response {
	x.t.Helper()
	body := map[string]any{"age_confirmed": true, "installation_id": ids.NewString(), "platform": "android", "app_version": "1.0.0+1", "locale": "tr"}
	for k, v := range extra {
		body[k] = v
	}
	return x.do(http.MethodPost, "/v1/session", token, body)
}

// jobs counts the published events of a name (one delivery job per event
// for the test subscriber).
func (x *e2e) jobs(event string) int {
	x.t.Helper()
	var n int
	if err := x.db.QueryRow(context.Background(), `SELECT count(*) FROM river_job WHERE args->'event'->>'name' = $1 AND args->>'subscriber' = $2`,
		event, "test.on_"+event).Scan(&n); err != nil {
		x.t.Fatal(err)
	}
	return n
}

func (x *e2e) forgetPrincipal(firebaseUID string) {
	x.mods.identity.Forget(firebaseUID)
}

func want(t *testing.T, what string, r response, code int, errCode string) {
	t.Helper()
	if r.code != code || r.errCode() != errCode {
		t.Fatalf("%s: got %d %s, want %d %s\n%s", what, r.code, r.errCode(), code, errCode, r.raw)
	}
}

func TestSessionAndProfileFlows(t *testing.T) {
	x := newE2E(t)
	token, uid := x.emu.signUpAnonymous()

	// --- 1.6.2 first session (guest) -------------------------------------
	want(t, "session without token", x.do(http.MethodPost, "/v1/session", "", map[string]any{}), 401, "UNAUTHENTICATED")
	want(t, "/v1/me before any session", x.do(http.MethodGet, "/v1/me", token, nil), 401, "UNAUTHENTICATED")
	want(t, "missing age confirmation", x.session(token, map[string]any{"age_confirmed": false}), 400, "AGE_CONFIRMATION_REQUIRED")
	r := x.session(token, map[string]any{"platform": "windows", "locale": "de", "installation_id": "x"})
	want(t, "invalid session body", r, 400, "VALIDATION_FAILED")
	if n := len(r.get("error", "details", "fields").([]any)); n != 3 {
		t.Fatalf("want 3 field errors, got %s", r.raw)
	}

	r = x.session(token, map[string]any{"country_hint": "tr"})
	want(t, "first session", r, 200, "")
	userID := r.get("user", "id").(string)
	if r.get("flags", "created") != true || r.get("tier") != "guest" || r.get("flags", "restricted") != true {
		t.Fatalf("first session = %s", r.raw)
	}
	if u := r.get("profile", "username").(string); !strings.HasPrefix(u, "oyuncu") {
		t.Fatalf("generated username %q", u)
	}
	if d := r.get("profile", "display_name").(string); !strings.HasPrefix(d, "Misafir ") || r.get("profile", "country_code") != "TR" {
		t.Fatalf("generated profile = %v", r.get("profile"))
	}
	if x.jobs(identity.EventUserRegistered) != 1 {
		t.Fatal("UserRegistered not queued in the session transaction")
	}

	r = x.session(token, nil)
	want(t, "repeated session", r, 200, "")
	if r.get("flags", "created") != false || r.get("user", "id") != userID {
		t.Fatalf("session is not idempotent: %s", r.raw)
	}
	if x.jobs(identity.EventUserRegistered) != 1 {
		t.Fatal("repeated session published UserRegistered again")
	}

	// --- 1.6.4 /v1/me, settings, profile rules ----------------------------
	r = x.do(http.MethodGet, "/v1/me", token, nil)
	want(t, "GET /v1/me", r, 200, "")
	if r.get("user", "id") != userID || r.get("settings", "challenges_from") != "everyone" || r.get("profile", "username_change_available_at") != nil {
		t.Fatalf("/v1/me = %s", r.raw)
	}
	want(t, "guest username change", x.do(http.MethodPatch, "/v1/me/profile", token, map[string]any{"username": "ali.veli"}), 403, "ACCOUNT_UPGRADE_REQUIRED")
	want(t, "profanity", x.do(http.MethodPatch, "/v1/me/profile", token, map[string]any{"display_name": "siktir git"}), 422, "TEXT_REJECTED")
	want(t, "bad avatar", x.do(http.MethodPatch, "/v1/me/profile", token, map[string]any{"avatar_id": "selfie.jpg"}), 400, "VALIDATION_FAILED")
	r = x.do(http.MethodPatch, "/v1/me/profile", token, map[string]any{"display_name": "  Kocaman   Kedi ", "avatar_id": "owl", "country_code": "de", "locale": "en"})
	want(t, "profile edit", r, 200, "")
	if r.get("display_name") != "Kocaman Kedi" || r.get("country_code") != "DE" || r.get("country_change_available_at") == nil {
		t.Fatalf("after edit: %s", r.raw)
	}
	r = x.do(http.MethodPatch, "/v1/me/profile", token, map[string]any{"country_code": "FR"})
	want(t, "second country change", r, 409, "CHANGE_TOO_SOON")
	if r.get("error", "details", "field") != "country_code" || r.get("error", "details", "available_at") == nil {
		t.Fatalf("CHANGE_TOO_SOON details: %s", r.raw)
	}
	r = x.do(http.MethodPatch, "/v1/me/settings", token, map[string]any{"challenges_from": "friends", "sound_enabled": false})
	if r.code != 200 || r.get("challenges_from") != "friends" || r.get("sound_enabled") != false || r.get("music_enabled") != true {
		t.Fatalf("settings patch: %s", r.raw)
	}
	want(t, "bad settings", x.do(http.MethodPatch, "/v1/me/settings", token, map[string]any{"challenges_from": "aliens"}), 400, "VALIDATION_FAILED")

	// --- 1.6.1/1.6.2 upgrade: link email (unverified), then verify (full) -
	email := "u" + strings.ReplaceAll(ids.NewString(), "-", "") + "@example.com"
	token = x.emu.linkPassword(token, email)
	// Any authenticated request notices the tier changed and re-syncs.
	r = x.do(http.MethodGet, "/v1/me", token, nil)
	if r.code != 200 || r.get("user", "tier") != "unverified" || r.get("user", "id") != userID {
		t.Fatalf("after linking email: %s", r.raw)
	}
	x.emu.verifyEmail(uid)
	token = x.emu.signIn(email)
	r = x.session(token, nil)
	if r.code != 200 || r.get("tier") != "full" || r.get("user", "upgraded_at") == nil || r.get("flags", "restricted") != false || r.get("user", "id") != userID {
		t.Fatalf("after verifying email: %s", r.raw)
	}
	if n := x.jobs(identity.EventAccountTierChanged); n != 2 {
		t.Fatalf("AccountTierChanged jobs = %d, want 2 (guest→unverified→full)", n)
	}
	var providers string
	_ = x.db.QueryRow(context.Background(), `SELECT string_agg(provider, ',' ORDER BY provider) FROM identity.auth_identities WHERE user_id = $1`, userID).Scan(&providers)
	if providers != "password" {
		t.Fatalf("mirrored providers = %q, want password (anonymous dropped after linking)", providers)
	}

	r = x.do(http.MethodPatch, "/v1/me/profile", token, map[string]any{"username": "Ali.Veli" + userID[len(userID)-4:]})
	want(t, "full account username change", r, 200, "")
	want(t, "second username change", x.do(http.MethodPatch, "/v1/me/profile", token, map[string]any{"username": "another_name"}), 409, "CHANGE_TOO_SOON")

	// --- public profile & username uniqueness ------------------------------
	other, _ := x.emu.signUpAnonymous()
	r = x.session(other, map[string]any{"locale": "en"})
	otherID := r.get("user", "id").(string)
	if !strings.HasPrefix(r.get("profile", "display_name").(string), "Guest ") {
		t.Fatalf("en display name: %s", r.raw)
	}
	r = x.do(http.MethodGet, "/v1/profiles/"+userID, other, nil)
	if r.code != 200 || r.get("display_name") != "Kocaman Kedi" || strings.Contains(r.raw, email) {
		t.Fatalf("public profile: %s", r.raw)
	}
	want(t, "unknown profile", x.do(http.MethodGet, "/v1/profiles/"+ids.NewString(), other, nil), 404, "NOT_FOUND")
	want(t, "malformed profile id", x.do(http.MethodGet, "/v1/profiles/abc", other, nil), 404, "NOT_FOUND")

	// --- 1.6.5 devices -----------------------------------------------------
	inst := ids.NewString()
	device := map[string]any{"installation_id": inst, "fcm_token": "fcm-token-0123456789abcdef", "platform": "android", "app_version": "1.0.0+1", "locale": "tr"}
	r = x.do(http.MethodPost, "/v1/me/devices", token, device)
	if r.code != 200 || r.get("push_enabled") != true {
		t.Fatalf("register device: %s", r.raw)
	}
	r = x.do(http.MethodPost, "/v1/me/devices", other, device) // same phone, other account (D-015)
	want(t, "device re-assignment", r, 200, "")
	var owner string
	_ = x.db.QueryRow(context.Background(), `SELECT user_id::text FROM notification.devices WHERE installation_id = $1`, inst).Scan(&owner)
	if owner != otherID {
		t.Fatalf("device owner = %s, want %s", owner, otherID)
	}
	want(t, "revoke foreign device is a no-op", x.do(http.MethodDelete, "/v1/me/devices/"+inst, token, nil), 204, "")
	want(t, "revoke own device", x.do(http.MethodDelete, "/v1/me/devices/"+inst, other, nil), 204, "")
	want(t, "bad device", x.do(http.MethodPost, "/v1/me/devices", other, map[string]any{"installation_id": inst}), 400, "VALIDATION_FAILED")

	// --- 1.6.3 abandon guest (D-015) ---------------------------------------
	key := ids.NewString()
	want(t, "abandon without key", x.do(http.MethodPost, "/v1/session/abandon-guest", other, nil), 400, "VALIDATION_FAILED")
	want(t, "abandon guest", x.do(http.MethodPost, "/v1/session/abandon-guest", other, nil, idempotency.Header, key), 204, "")
	r = x.do(http.MethodPost, "/v1/session/abandon-guest", other, nil, idempotency.Header, key)
	if r.code != 204 || x.jobs(identity.EventGuestAbandoned) != 1 {
		t.Fatalf("repeat abandon = %d, jobs %d", r.code, x.jobs(identity.EventGuestAbandoned))
	}
	want(t, "abandon a full account", x.do(http.MethodPost, "/v1/session/abandon-guest", token, nil, idempotency.Header, ids.NewString()), 403, "FORBIDDEN")

	// --- 1.6.6 deletion request --------------------------------------------
	r = x.do(http.MethodPost, "/v1/me/deletion", token, nil, idempotency.Header, ids.NewString())
	want(t, "request deletion", r, 200, "")
	req, _ := time.Parse(time.RFC3339Nano, r.get("requested_at").(string))
	purge, _ := time.Parse(time.RFC3339Nano, r.get("purge_after").(string))
	if purge.Sub(req) != 14*24*time.Hour {
		t.Fatalf("cooling-off = %v", purge.Sub(req))
	}
	want(t, "edit during cooling-off", x.do(http.MethodPatch, "/v1/me/settings", token, map[string]any{"sound_enabled": true}), 403, "ACCOUNT_DELETION_PENDING")
	if r = x.do(http.MethodGet, "/v1/me", token, nil); r.code != 200 || r.get("user", "deletion_requested_at") == nil {
		t.Fatalf("GET /v1/me during cooling-off: %s", r.raw)
	}
	if r = x.session(token, nil); r.get("flags", "deletion_pending") != true {
		t.Fatalf("session flags during cooling-off: %s", r.raw)
	}
	want(t, "cancel deletion", x.do(http.MethodDelete, "/v1/me/deletion", token, nil), 204, "")
	want(t, "edit after cancelling", x.do(http.MethodPatch, "/v1/me/settings", token, map[string]any{"sound_enabled": true}), 200, "")

	// --- suspension & deleted accounts --------------------------------------
	if _, err := x.db.Exec(context.Background(), `UPDATE identity.users SET status = 'suspended' WHERE id = $1`, userID); err != nil {
		t.Fatal(err)
	}
	x.forgetPrincipal(uid)
	want(t, "suspended: own profile", x.do(http.MethodGet, "/v1/me", token, nil), 200, "")
	want(t, "suspended: edit", x.do(http.MethodPatch, "/v1/me/settings", token, map[string]any{"sound_enabled": false}), 403, "ACCOUNT_SUSPENDED")
	if r = x.session(token, nil); r.code != 200 || r.get("flags", "suspended") != true {
		t.Fatalf("suspended session: %s", r.raw)
	}

	if _, err := x.db.Exec(context.Background(), `UPDATE identity.users SET status = 'deleted', deleted_at = now() WHERE id = $1`, userID); err != nil {
		t.Fatal(err)
	}
	x.forgetPrincipal(uid)
	want(t, "deleted: request", x.do(http.MethodGet, "/v1/me", token, nil), 401, "UNAUTHENTICATED")
	want(t, "deleted: session", x.session(token, nil), 401, "UNAUTHENTICATED")
	want(t, "deleted: public profile", x.do(http.MethodGet, "/v1/profiles/"+userID, other, nil), 404, "NOT_FOUND")
}

// TestConcurrentFirstSessions: two first sessions of the same new user at
// once create exactly one account.
func TestConcurrentFirstSessions(t *testing.T) {
	x := newE2E(t)
	token, _ := x.emu.signUpAnonymous()
	results := make(chan response, 4)
	for range 4 {
		go func() { results <- x.session(token, nil) }()
	}
	created, users := 0, map[any]bool{}
	for range 4 {
		r := <-results
		if r.code != 200 {
			t.Fatalf("concurrent session: %s", r.raw)
		}
		if r.get("flags", "created") == true {
			created++
		}
		users[r.get("user", "id")] = true
	}
	if created != 1 || len(users) != 1 {
		t.Fatalf("created=%d, distinct users=%d; want 1 and 1", created, len(users))
	}
	if n := x.jobs(identity.EventUserRegistered); n != 1 {
		t.Fatalf("UserRegistered jobs = %d", n)
	}
}

// linkGoogle links a Google identity (emulator: an unsigned JSON id_token)
// to the signed-in user, as linkWithCredential does.
func (e emulator) linkGoogle(token, sub, email string) string {
	idToken := fmt.Sprintf(`{"sub":%q,"email":%q,"email_verified":true}`, sub, email)
	out := e.call("accounts:signInWithIdp", map[string]any{
		"idToken": token, "postBody": "id_token=" + idToken + "&providerId=google.com",
		"requestUri": "http://localhost", "returnSecureToken": true,
	}, false)
	return out["idToken"].(string)
}

func (e emulator) deleteUser(uid string) {
	e.call("projects/"+emulatorProject+"/accounts:delete", map[string]any{"localId": uid}, true)
}

// TestStaleIdentityMovesToNewAccount: a Firebase account that no longer
// exists (deleted, or a restarted Auth emulator) can leave its Google
// identity or password email in identity.auth_identities. When a new
// Firebase user links the same sign-in method, the session must move it
// instead of failing with a unique violation (owner report 2026-10-18).
func TestStaleIdentityMovesToNewAccount(t *testing.T) {
	x := newE2E(t)
	email := "stale-" + ids.NewString()[24:] + "@example.com"
	sub := "google-" + ids.NewString()[24:]

	oldToken, oldUID := x.emu.signUpAnonymous()
	want(t, "old guest", x.session(oldToken, nil), 200, "")
	oldToken = x.emu.linkGoogle(oldToken, sub, email)
	oldToken = x.emu.linkPassword(oldToken, email)
	r := x.session(oldToken, nil)
	want(t, "old account with google + password", r, 200, "")
	oldID := r.get("user", "id").(string)
	x.emu.deleteUser(oldUID)

	newToken, _ := x.emu.signUpAnonymous()
	want(t, "new guest", x.session(newToken, nil), 200, "")
	newToken = x.emu.linkGoogle(newToken, sub, email)
	newToken = x.emu.linkPassword(newToken, email)
	r = x.session(newToken, nil)
	want(t, "new account takes over google + password", r, 200, "")
	if r.get("tier") != "full" {
		t.Fatalf("new account = %s", r.raw)
	}
	var left int
	if err := x.db.QueryRow(t.Context(), `SELECT count(*) FROM identity.auth_identities WHERE user_id = $1 AND provider IN ('google', 'password')`, oldID).Scan(&left); err != nil {
		t.Fatal(err)
	}
	if left != 0 {
		t.Fatalf("stale account still holds %d sign-in methods", left)
	}
}
