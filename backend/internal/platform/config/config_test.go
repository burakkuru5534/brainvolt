package config

import (
	"bytes"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/burakkuru5534/brainvolt/backend/internal/platform/logging"
)

// devVars mirrors .env.example after ${VAR} expansion.
func devVars() map[string]string {
	return map[string]string{
		"BRAINVOLT_ENV":                 "dev",
		"BRAINVOLT_HTTP_ADDR":           ":8080",
		"BRAINVOLT_LOG_LEVEL":           "debug",
		"BRAINVOLT_LOG_FORMAT":          "text",
		"BRAINVOLT_POSTGRES_URL":        "postgres://brainvolt:pg_secret@localhost:5432/brainvolt?sslmode=disable",
		"BRAINVOLT_REDIS_URL":           "redis://:redis_secret@localhost:6379/0",
		"BRAINVOLT_FIREBASE_PROJECT_ID": "demo-brainvolt",
		"FIREBASE_AUTH_EMULATOR_HOST":   "localhost:9099",
	}
}

func with(base map[string]string, kv ...string) map[string]string {
	m := maps.Clone(base)
	for i := 0; i < len(kv); i += 2 {
		if kv[i+1] == "<unset>" {
			delete(m, kv[i])
		} else {
			m[kv[i]] = kv[i+1]
		}
	}
	return m
}

func TestLoadDevDefaults(t *testing.T) {
	got, err := LoadFrom(devVars())
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	want := Config{
		Env: EnvDev,
		HTTP: HTTP{
			Addr: ":8080", ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second,
			IdleTimeout: 60 * time.Second, HandlerTimeout: 10 * time.Second, ShutdownTimeout: 20 * time.Second,
		},
		Log: Log{Level: slog.LevelDebug, Format: logging.FormatText},
		Postgres: Postgres{
			URL:              "postgres://brainvolt:pg_secret@localhost:5432/brainvolt?sslmode=disable",
			StatementTimeout: 3 * time.Second, WorkerStatementTimeout: 60 * time.Second,
		},
		Redis:    Redis{URL: "redis://:redis_secret@localhost:6379/0"},
		Firebase: Firebase{ProjectID: "demo-brainvolt", AuthEmulatorHost: "localhost:9099"},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("config mismatch (-want +got):\n%s", diff)
	}
}

func TestLoadProd(t *testing.T) {
	vars := with(devVars(),
		"BRAINVOLT_ENV", "prod",
		"BRAINVOLT_LOG_LEVEL", "<unset>",
		"BRAINVOLT_LOG_FORMAT", "<unset>",
		"BRAINVOLT_FIREBASE_PROJECT_ID", "brainvolt",
		"FIREBASE_AUTH_EMULATOR_HOST", "<unset>",
		"BRAINVOLT_HTTP_WRITE_TIMEOUT", "30s",
	)
	got, err := LoadFrom(vars)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if got.Log.Level != slog.LevelInfo || got.Log.Format != logging.FormatJSON {
		t.Errorf("prod log defaults = %v/%s, want INFO/json", got.Log.Level, got.Log.Format)
	}
	if got.HTTP.WriteTimeout != 30*time.Second {
		t.Errorf("WriteTimeout = %v, want 30s override", got.HTTP.WriteTimeout)
	}
}

func TestLoadErrors(t *testing.T) {
	tests := []struct {
		name string
		vars map[string]string
		want []string // substrings of the error
	}{
		{"missing required", with(devVars(), "BRAINVOLT_POSTGRES_URL", "<unset>", "BRAINVOLT_REDIS_URL", "<unset>"),
			[]string{"BRAINVOLT_POSTGRES_URL", "BRAINVOLT_REDIS_URL"}},
		{"empty required", with(devVars(), "BRAINVOLT_FIREBASE_PROJECT_ID", ""),
			[]string{"BRAINVOLT_FIREBASE_PROJECT_ID"}},
		{"bad env", with(devVars(), "BRAINVOLT_ENV", "staging"), []string{"BRAINVOLT_ENV"}},
		{"bad level", with(devVars(), "BRAINVOLT_LOG_LEVEL", "loud"), []string{"BRAINVOLT_LOG_LEVEL"}},
		{"bad format", with(devVars(), "BRAINVOLT_LOG_FORMAT", "xml"), []string{"BRAINVOLT_LOG_FORMAT"}},
		{"bad duration", with(devVars(), "BRAINVOLT_HTTP_READ_TIMEOUT", "ten"), []string{"BRAINVOLT_HTTP_READ_TIMEOUT"}},
		{"zero duration", with(devVars(), "BRAINVOLT_POSTGRES_STATEMENT_TIMEOUT", "0s"),
			[]string{"BRAINVOLT_POSTGRES_STATEMENT_TIMEOUT", "must be positive"}},
		{"handler timeout not below write timeout", with(devVars(), "BRAINVOLT_HTTP_HANDLER_TIMEOUT", "15s"),
			[]string{"must be shorter than BRAINVOLT_HTTP_WRITE_TIMEOUT"}},
		{"bad addr", with(devVars(), "BRAINVOLT_HTTP_ADDR", "8080"), []string{"BRAINVOLT_HTTP_ADDR"}},
		{"wrong scheme", with(devVars(), "BRAINVOLT_REDIS_URL", "http://localhost:6379"),
			[]string{"BRAINVOLT_REDIS_URL", `scheme "http"`}},
		{"no host", with(devVars(), "BRAINVOLT_POSTGRES_URL", "postgres:///db"),
			[]string{"BRAINVOLT_POSTGRES_URL", "missing host"}},
		{"emulator in prod", with(devVars(), "BRAINVOLT_ENV", "prod", "BRAINVOLT_FIREBASE_PROJECT_ID", "brainvolt"),
			[]string{"FIREBASE_AUTH_EMULATOR_HOST must not be set in prod"}},
		{"demo project in prod", with(devVars(), "BRAINVOLT_ENV", "prod", "FIREBASE_AUTH_EMULATOR_HOST", "<unset>"),
			[]string{"demo- projects are emulator-only"}},
		{"demo project without emulator", with(devVars(), "FIREBASE_AUTH_EMULATOR_HOST", "<unset>"),
			[]string{"needs FIREBASE_AUTH_EMULATOR_HOST"}},
		{"bad emulator host", with(devVars(), "FIREBASE_AUTH_EMULATOR_HOST", "localhost"),
			[]string{"FIREBASE_AUTH_EMULATOR_HOST"}},
		{"several problems at once", with(devVars(), "BRAINVOLT_ENV", "qa", "BRAINVOLT_HTTP_ADDR", "x"),
			[]string{"BRAINVOLT_ENV", "BRAINVOLT_HTTP_ADDR"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadFrom(tc.vars)
			if err == nil {
				t.Fatal("LoadFrom succeeded, want error")
			}
			for _, w := range tc.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error %q does not mention %q", err, w)
				}
			}
			for _, secret := range []string{"pg_secret", "redis_secret"} {
				if strings.Contains(err.Error(), secret) {
					t.Errorf("error leaks a password: %q", err)
				}
			}
		})
	}
}

func TestLogValueRedactsPasswords(t *testing.T) {
	cfg, err := LoadFrom(devVars())
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	slog.New(slog.NewTextHandler(&buf, nil)).Info("config", "config", cfg)
	out := buf.String()
	for _, secret := range []string{"pg_secret", "redis_secret"} {
		if strings.Contains(out, secret) {
			t.Errorf("log output leaks %q:\n%s", secret, out)
		}
	}
	if !strings.Contains(out, `config.postgres_url="postgres://brainvolt:xxxxx@localhost:5432/brainvolt?sslmode=disable"`) {
		t.Errorf("redacted postgres URL missing:\n%s", out)
	}
}

func TestLoadReadsProcessEnvironment(t *testing.T) {
	for k, v := range devVars() {
		t.Setenv(k, v)
	}
	t.Setenv("BRAINVOLT_HTTP_ADDR", "127.0.0.1:9000")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.HTTP.Addr != "127.0.0.1:9000" {
		t.Fatalf("Addr = %q", cfg.HTTP.Addr)
	}
}

// TestEnvExampleIsValid keeps .env.example and Config in step: the file,
// expanded the way Compose and the Makefile expand it, must load cleanly.
func TestEnvExampleIsValid(t *testing.T) {
	f, err := ReadEnvFile("../../../../.env.example")
	if err != nil {
		t.Fatal(err)
	}
	noEnv := func(string) (string, bool) { return "", false }
	cfg, err := LoadFrom(f.Expand(noEnv))
	if err != nil {
		t.Fatalf(".env.example does not load: %v", err)
	}
	if cfg.Postgres.URL != "postgres://brainvolt:brainvolt_dev_password@localhost:5432/brainvolt?sslmode=disable" {
		t.Errorf("expanded postgres URL = %q", cfg.Postgres.URL)
	}
	if problems := CheckEnvFile(f, f, noEnv); len(problems) != 0 {
		t.Errorf(".env.example must pass its own check:\n%s", strings.Join(problems, "\n"))
	}
}

func writeEnv(t *testing.T, content string) EnvFile {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := ReadEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestCheckEnvFile(t *testing.T) {
	example := writeEnv(t, "# comment\nPORT=9099\nHOST=localhost:${PORT}\nPG_PORT=5432\n")
	noEnv := func(string) (string, bool) { return "", false }

	// The owner's case: PORT missing, so HOST becomes "localhost:".
	actual := writeEnv(t, "HOST=localhost:${PORT}\nPG_PORT=5433\n")
	problems := CheckEnvFile(example, actual, noEnv)
	if len(problems) != 2 ||
		!strings.Contains(problems[0], "PORT is missing from .env (copy it from .env.example: PORT=9099)") ||
		!strings.Contains(problems[1], "HOST uses ${PORT}") || !strings.Contains(problems[1], "add PORT=9099") {
		t.Fatalf("problems = %q", problems)
	}
	if got := actual.Expand(noEnv)["HOST"]; got != "localhost:" {
		t.Fatalf("expanded HOST = %q (documents the failure mode)", got)
	}

	// Defined below its use: still empty when expanded in order.
	actual = writeEnv(t, "HOST=localhost:${PORT}\nPORT=9099\nPG_PORT=5433\n")
	if problems := CheckEnvFile(example, actual, noEnv); len(problems) != 1 || !strings.Contains(problems[0], "move its definition above") {
		t.Fatalf("problems = %q", problems)
	}

	// A value from the process environment counts as defined; changed values are fine.
	actual = writeEnv(t, "PORT=9100\nHOST=localhost:${PORT}\nPG_PORT=5433\nEXTRA=${HOME}\n")
	env := func(k string) (string, bool) { return "/home/x", k == "HOME" }
	if problems := CheckEnvFile(example, actual, env); len(problems) != 0 {
		t.Fatalf("problems = %q", problems)
	}
	if got := actual.Expand(env); got["HOST"] != "localhost:9100" || got["EXTRA"] != "/home/x" {
		t.Fatalf("Expand = %v", got)
	}
}

func TestReadEnvFileRejectsMalformedLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	_ = os.WriteFile(path, []byte("GOOD=1\nthis is not an assignment\n"), 0o600)
	if _, err := ReadEnvFile(path); err == nil || !strings.Contains(err.Error(), ":2:") {
		t.Fatalf("err = %v, want line number", err)
	}
}

func TestLoadPostgresNeedsOnlyPostgres(t *testing.T) {
	t.Setenv("BRAINVOLT_REDIS_URL", "")
	t.Setenv("BRAINVOLT_FIREBASE_PROJECT_ID", "")
	t.Setenv("BRAINVOLT_POSTGRES_URL", "postgres://u:secret@db:5432/x")
	pg, err := LoadPostgres()
	if err != nil || pg.URL != "postgres://u:secret@db:5432/x" || pg.StatementTimeout != 3*time.Second {
		t.Fatalf("LoadPostgres = %+v, %v", pg, err)
	}
	t.Setenv("BRAINVOLT_POSTGRES_URL", "mysql://u:secret@db/x")
	if _, err := LoadPostgres(); err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatalf("bad scheme: %v", err)
	}
}
