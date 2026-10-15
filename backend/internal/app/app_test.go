package app

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/burakkuru5534/brainvolt/backend/internal/platform/config"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/testdb"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/testenv"
	"github.com/burakkuru5534/brainvolt/backend/migrations"
)

var discard = slog.New(slog.NewTextHandler(io.Discard, nil))

// blocker runs until ctx is cancelled and records that it stopped.
type blocker struct {
	name    string
	stopped atomic.Bool
}

func (b *blocker) Name() string { return b.name }

func (b *blocker) Run(ctx context.Context) error {
	<-ctx.Done()
	b.stopped.Store(true)
	return ctx.Err()
}

// failer returns err after delay.
type failer struct {
	delay time.Duration
	err   error
}

func (f *failer) Name() string { return "failer" }

func (f *failer) Run(ctx context.Context) error {
	select {
	case <-time.After(f.delay):
		return f.err
	case <-ctx.Done():
		return nil
	}
}

func TestRunStopsCleanlyOnCancel(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		b1, b2 := &blocker{name: "one"}, &blocker{name: "two"}
		a := New(RoleAPI, discard, b1, b2)

		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan error, 1)
		go func() { done <- a.Run(ctx) }()

		synctest.Wait()
		select {
		case err := <-done:
			t.Fatalf("Run returned before cancel: %v", err)
		default:
		}

		cancel()
		if err := <-done; err != nil {
			t.Fatalf("Run after cancel = %v, want nil", err)
		}
		if !b1.stopped.Load() || !b2.stopped.Load() {
			t.Fatal("Run returned before every component stopped")
		}
	})
}

func TestRunWithoutComponentsBlocksUntilCancel(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), time.Hour)
		defer cancel()

		start := time.Now()
		if err := New(RoleWorker, discard).Run(ctx); err != nil {
			t.Fatalf("Run = %v, want nil", err)
		}
		if got := time.Since(start); got != time.Hour {
			t.Fatalf("Run returned after %v, want 1h (context deadline)", got)
		}
	})
}

func TestRunComponentFailureStopsOthers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		boom := errors.New("boom")
		b := &blocker{name: "other"}
		a := New(RoleAPI, discard, b, &failer{delay: time.Second, err: boom})

		err := a.Run(t.Context())
		if !errors.Is(err, boom) {
			t.Fatalf("Run = %v, want error wrapping %v", err, boom)
		}
		if want := "failer: boom"; err.Error() != want {
			t.Fatalf("error text = %q, want %q", err.Error(), want)
		}
		if !b.stopped.Load() {
			t.Fatal("healthy component was not stopped after a failure")
		}
	})
}

// devEnv is a valid local configuration (see .env.example) whose Postgres
// and Redis cannot be reached; integrationEnv points them at the test services.
var devEnv = map[string]string{
	"BRAINVOLT_LOG_FORMAT":          "text",
	"BRAINVOLT_HTTP_ADDR":           "127.0.0.1:0",
	"BRAINVOLT_POSTGRES_URL":        "postgres://brainvolt:pw@127.0.0.1:1/brainvolt?sslmode=disable",
	"BRAINVOLT_REDIS_URL":           "redis://:pw@127.0.0.1:1/0",
	"BRAINVOLT_FIREBASE_PROJECT_ID": "demo-brainvolt",
	"FIREBASE_AUTH_EMULATOR_HOST":   "localhost:9099",
}

func integrationEnv(t *testing.T) map[string]string {
	t.Helper()
	env := maps.Clone(devEnv)
	env["BRAINVOLT_POSTGRES_URL"] = testdb.NewURL(t) // migrated throw-away database
	env["BRAINVOLT_REDIS_URL"] = testenv.RedisURL(t)
	return env
}

func loadConfig(t *testing.T, env map[string]string) config.Config {
	t.Helper()
	cfg, err := config.LoadFrom(env)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestMainReturnsOneWhenBuildFails(t *testing.T) {
	for k, v := range devEnv {
		t.Setenv(k, v)
	}
	built := false
	build := func(context.Context, config.Config, *slog.Logger) (*App, error) {
		built = true
		return nil, errors.New("wiring failed")
	}
	if code := Main(build); code != 1 || !built {
		t.Fatalf("Main exit code = %d (build called: %v), want 1 after build", code, built)
	}
}

func TestMainReturnsOneOnInvalidConfig(t *testing.T) {
	for k, v := range devEnv {
		t.Setenv(k, v)
	}
	t.Setenv("BRAINVOLT_POSTGRES_URL", "")
	build := func(context.Context, config.Config, *slog.Logger) (*App, error) {
		t.Fatal("build called with invalid configuration")
		return nil, nil
	}
	if code := Main(build); code != 1 {
		t.Fatalf("Main exit code = %d, want 1", code)
	}
}

func TestBuildFailsFastWhenDependenciesUnreachable(t *testing.T) {
	cfg := loadConfig(t, devEnv)
	for name, build := range map[string]func(context.Context, config.Config, *slog.Logger) (*App, error){
		"api": NewAPI, "worker": NewWorker,
	} {
		_, err := build(t.Context(), cfg, discard)
		if err == nil || !strings.Contains(err.Error(), "postgres: cannot reach 127.0.0.1:1") {
			t.Errorf("%s: error = %v, want unreachable postgres", name, err)
		}
		if err != nil && strings.Contains(err.Error(), "pw@") {
			t.Errorf("%s: error leaks the password: %v", name, err)
		}
	}

	// Postgres reachable, Redis not: the API must still fail (and close the pool).
	if pg := os.Getenv(testenv.PostgresVar); pg != "" {
		env := maps.Clone(devEnv)
		env["BRAINVOLT_POSTGRES_URL"] = pg
		_, err := NewAPI(t.Context(), loadConfig(t, env), discard)
		if err == nil || !strings.Contains(err.Error(), "redis: cannot reach 127.0.0.1:1") {
			t.Errorf("api with unreachable redis: error = %v", err)
		}
	}
}

func TestClosersRunAfterComponentsInReverseOrder(t *testing.T) {
	var order []string
	b := &blocker{name: "server"}
	a := New(RoleAPI, discard, b)
	a.closers = append(a.closers,
		func() { order = append(order, "postgres:"+strconv.FormatBool(b.stopped.Load())) },
		func() { order = append(order, "redis") },
	)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := a.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if want := []string{"redis", "postgres:true"}; !slices.Equal(order, want) {
		t.Fatalf("close order = %v, want %v", order, want)
	}
}

// TestAPIReadiness builds the real API composition against the test
// services and checks /healthz and /readyz through the HTTP handler.
func TestAPIReadiness(t *testing.T) {
	cfg := loadConfig(t, integrationEnv(t))
	a, err := NewAPI(t.Context(), cfg, discard)
	if err != nil {
		t.Fatalf("NewAPI: %v", err)
	}
	defer func() {
		for _, c := range a.closers {
			c()
		}
	}()
	if a.Role() != RoleAPI {
		t.Fatalf("role = %q", a.Role())
	}
	srv, ok := a.components[0].(*httpserver.Server)
	if !ok {
		t.Fatalf("first component is %T, want *httpserver.Server", a.components[0])
	}

	for path, want := range map[string]string{
		"/healthz": `{"status":"ok"}`,
		"/readyz":  `{"checks":{"migrations":"ok","postgres":"ok","redis":"ok"},"status":"ready"}`,
	} {
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, http.NoBody))
		if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != want {
			t.Errorf("GET %s = %d %s, want 200 %s", path, rec.Code, rec.Body.String(), want)
		}
	}

	// One migration behind: not ready, and the response names the check.
	probe, err := pgxpool.New(t.Context(), cfg.Postgres.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer probe.Close()
	sqlDB := stdlib.OpenDBFromPool(probe)
	p, err := migrations.NewProvider(sqlDB)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Down(t.Context()); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", http.NoBody))
	if rec.Code != http.StatusServiceUnavailable || !strings.Contains(rec.Body.String(), `"migrations":"failed"`) {
		t.Errorf("/readyz with a pending migration = %d %s", rec.Code, rec.Body.String())
	}
	_ = sqlDB.Close()

	w, err := NewWorker(t.Context(), cfg, discard)
	if err != nil {
		t.Fatalf("NewWorker: %v", err)
	}
	for _, c := range w.closers {
		c()
	}
}
