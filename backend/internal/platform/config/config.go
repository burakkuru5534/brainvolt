// Package config reads the BRAINVOLT_* environment variables into a typed
// Config and validates it (ADR-0002: caarlos0/env v11 + own validation).
// Every variable is documented in .env.example at the repository root.
// Invalid configuration stops start-up with one message per problem.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"

	"github.com/burakkuru5534/brainvolt/backend/internal/platform/logging"
)

// Environment names a deployment environment.
type Environment string

const (
	// EnvDev is local development (Docker Compose, Firebase Auth emulator).
	EnvDev Environment = "dev"
	// EnvProd is any deployed environment.
	EnvProd Environment = "prod"
)

// Config is the complete process configuration.
type Config struct {
	Env Environment `env:"BRAINVOLT_ENV" envDefault:"dev"`
	// EmbedWorker makes the API process also run the worker's jobs
	// (outbox deliveries, periodic jobs) so one Cloud Run service with
	// min-instances 0 does everything (D-096). cmd/worker is then not
	// deployed.
	EmbedWorker bool `env:"BRAINVOLT_EMBED_WORKER" envDefault:"false"`
	HTTP        HTTP
	Log         Log
	Postgres    Postgres
	Firebase    Firebase
}

// HTTP configures the REST/WebSocket server (timeouts: docs/api/limits.md §3).
type HTTP struct {
	Addr         string        `env:"BRAINVOLT_HTTP_ADDR" envDefault:":8080"`
	ReadTimeout  time.Duration `env:"BRAINVOLT_HTTP_READ_TIMEOUT" envDefault:"10s"`
	WriteTimeout time.Duration `env:"BRAINVOLT_HTTP_WRITE_TIMEOUT" envDefault:"15s"`
	IdleTimeout  time.Duration `env:"BRAINVOLT_HTTP_IDLE_TIMEOUT" envDefault:"60s"`
	// HandlerTimeout bounds each request's context; it is shorter than
	// WriteTimeout so the error response can still be written.
	HandlerTimeout  time.Duration `env:"BRAINVOLT_HTTP_HANDLER_TIMEOUT" envDefault:"10s"`
	ShutdownTimeout time.Duration `env:"BRAINVOLT_HTTP_SHUTDOWN_TIMEOUT" envDefault:"20s"`
}

// Log configures platform/logging.
type Log struct {
	Level  slog.Level     `env:"BRAINVOLT_LOG_LEVEL" envDefault:"info"`
	Format logging.Format `env:"BRAINVOLT_LOG_FORMAT" envDefault:"json"`
}

// Postgres configures platform/postgres (statement timeouts: limits.md §3).
type Postgres struct {
	URL                    string        `env:"BRAINVOLT_POSTGRES_URL,required,notEmpty"`
	StatementTimeout       time.Duration `env:"BRAINVOLT_POSTGRES_STATEMENT_TIMEOUT" envDefault:"3s"`
	WorkerStatementTimeout time.Duration `env:"BRAINVOLT_POSTGRES_WORKER_STATEMENT_TIMEOUT" envDefault:"60s"`
}

// Firebase configures ID-token verification (platform/firebase).
type Firebase struct {
	ProjectID string `env:"BRAINVOLT_FIREBASE_PROJECT_ID,required,notEmpty"`
	// AuthEmulatorHost is the standard variable read by the Firebase Admin
	// SDK itself; set only in local development (ADR-0003).
	AuthEmulatorHost string `env:"FIREBASE_AUTH_EMULATOR_HOST"`
	// MessagingCredentialsFile is the service-account key the worker sends
	// push with (FCM); without it pushes are only logged in dev or use ADC in prod.
	MessagingCredentialsFile string `env:"BRAINVOLT_FCM_CREDENTIALS_FILE"`
	// MessagingCredentialsJSON allows supplying the service-account key as a raw JSON string
	// (useful for GCP Secret Manager or direct environment variable injection).
	MessagingCredentialsJSON string `env:"BRAINVOLT_FCM_CREDENTIALS_JSON"`
}

// Load reads the process environment.
func Load() (Config, error) {
	return parse(env.Options{})
}

// LoadPostgres reads only the Postgres settings, for tools such as
// cmd/migrate that need nothing else.
func LoadPostgres() (Postgres, error) {
	pg, err := env.ParseAs[Postgres]()
	if err != nil {
		return Postgres{}, fmt.Errorf("config: %w", nameVariables(err))
	}
	if err := checkURL(pg.URL, "postgres", "postgresql"); err != nil {
		return Postgres{}, fmt.Errorf("config: BRAINVOLT_POSTGRES_URL: %w", err)
	}
	return pg, nil
}

// LoadFrom reads the given variables instead of the process environment.
func LoadFrom(vars map[string]string) (Config, error) {
	return parse(env.Options{Environment: vars})
}

func parse(opts env.Options) (Config, error) {
	cfg, err := env.ParseAsWithOptions[Config](opts)
	if err != nil {
		return Config{}, fmt.Errorf("config: %w", nameVariables(err))
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	return cfg, nil
}

// nameVariables rewrites env's parse errors, which name the Go struct field
// ("ReadTimeout"), so that they name the environment variable instead.
func nameVariables(err error) error {
	var agg env.AggregateError
	if !errors.As(err, &agg) {
		return err
	}
	keys := envKeys(reflect.TypeFor[Config]())
	out := make([]error, 0, len(agg.Errors))
	for _, e := range agg.Errors {
		var pe env.ParseError
		if errors.As(e, &pe) && len(keys[pe.Name]) > 0 {
			e = fmt.Errorf("%s: %w", strings.Join(keys[pe.Name], " or "), pe.Err)
		}
		out = append(out, e)
	}
	return errors.Join(out...)
}

// envKeys maps struct field names to the env variables tagged on them.
func envKeys(t reflect.Type) map[string][]string {
	keys := map[string][]string{}
	for f := range t.Fields() {
		if f.Type.Kind() == reflect.Struct && f.Tag.Get("env") == "" && f.Type.PkgPath() == t.PkgPath() {
			for name, k := range envKeys(f.Type) {
				keys[name] = append(keys[name], k...)
			}
			continue
		}
		if key, _, _ := strings.Cut(f.Tag.Get("env"), ","); key != "" {
			keys[f.Name] = append(keys[f.Name], key)
		}
	}
	return keys
}

// Validate checks values that parsing alone cannot.
func (c Config) Validate() error {
	var errs []error
	add := func(format string, args ...any) { errs = append(errs, fmt.Errorf(format, args...)) }

	switch c.Env {
	case EnvDev, EnvProd:
	default:
		add("BRAINVOLT_ENV=%q: want dev or prod", c.Env)
	}

	if c.HTTP.HandlerTimeout >= c.HTTP.WriteTimeout {
		add("BRAINVOLT_HTTP_HANDLER_TIMEOUT=%s must be shorter than BRAINVOLT_HTTP_WRITE_TIMEOUT=%s", c.HTTP.HandlerTimeout, c.HTTP.WriteTimeout)
	}
	if _, _, err := net.SplitHostPort(c.HTTP.Addr); err != nil {
		add("BRAINVOLT_HTTP_ADDR=%q: want host:port or :port", c.HTTP.Addr)
	}
	for name, d := range map[string]time.Duration{
		"BRAINVOLT_HTTP_READ_TIMEOUT":                 c.HTTP.ReadTimeout,
		"BRAINVOLT_HTTP_WRITE_TIMEOUT":                c.HTTP.WriteTimeout,
		"BRAINVOLT_HTTP_IDLE_TIMEOUT":                 c.HTTP.IdleTimeout,
		"BRAINVOLT_HTTP_HANDLER_TIMEOUT":              c.HTTP.HandlerTimeout,
		"BRAINVOLT_HTTP_SHUTDOWN_TIMEOUT":             c.HTTP.ShutdownTimeout,
		"BRAINVOLT_POSTGRES_STATEMENT_TIMEOUT":        c.Postgres.StatementTimeout,
		"BRAINVOLT_POSTGRES_WORKER_STATEMENT_TIMEOUT": c.Postgres.WorkerStatementTimeout,
	} {
		if d <= 0 {
			add("%s=%s: must be positive", name, d)
		}
	}

	if err := checkURL(c.Postgres.URL, "postgres", "postgresql"); err != nil {
		add("BRAINVOLT_POSTGRES_URL: %v", err)
	}

	demoProject := strings.HasPrefix(c.Firebase.ProjectID, "demo-")
	if c.Env == EnvProd {
		if c.Firebase.AuthEmulatorHost != "" {
			add("FIREBASE_AUTH_EMULATOR_HOST must not be set in prod: tokens would not be verified against Firebase")
		}
		if demoProject {
			add("BRAINVOLT_FIREBASE_PROJECT_ID=%q: demo- projects are emulator-only, not allowed in prod", c.Firebase.ProjectID)
		}
	}
	if demoProject && c.Firebase.AuthEmulatorHost == "" {
		add("BRAINVOLT_FIREBASE_PROJECT_ID=%q needs FIREBASE_AUTH_EMULATOR_HOST: demo- projects only exist in the emulator", c.Firebase.ProjectID)
	}
	if h := c.Firebase.AuthEmulatorHost; h != "" {
		if _, _, err := net.SplitHostPort(h); err != nil {
			add("FIREBASE_AUTH_EMULATOR_HOST=%q: want host:port", h)
		}
	}

	return errors.Join(errs...)
}

// checkURL verifies the scheme and host; the error never contains the URL,
// because it may hold a password.
func checkURL(raw string, schemes ...string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return errors.New("not a valid URL")
	}
	ok := false
	for _, s := range schemes {
		ok = ok || u.Scheme == s
	}
	if !ok {
		return fmt.Errorf("scheme %q: want %s", u.Scheme, strings.Join(schemes, " or "))
	}
	if u.Host == "" {
		return errors.New("missing host")
	}
	return nil
}

// LogValue logs the configuration with passwords removed from URLs.
func (c Config) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("env", string(c.Env)),
		slog.String("http_addr", c.HTTP.Addr),
		slog.String("log_level", c.Log.Level.String()),
		slog.String("log_format", string(c.Log.Format)),
		slog.String("postgres_url", redact(c.Postgres.URL)),
		slog.Duration("postgres_statement_timeout", c.Postgres.StatementTimeout),
		slog.String("firebase_project_id", c.Firebase.ProjectID),
		slog.String("firebase_auth_emulator_host", c.Firebase.AuthEmulatorHost),
	)
}

func redact(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "<invalid>"
	}
	return u.Redacted()
}
