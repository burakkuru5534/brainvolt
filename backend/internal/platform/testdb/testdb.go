// Package testdb gives each integration test its own, fully migrated
// Postgres database (roadmap 1.5.5).
//
// A template database named after a hash of the migrations is built once
// (under an advisory lock, so parallel test packages don't race) and reused
// across runs until a migration changes. Every New call then clones it with
// CREATE DATABASE … TEMPLATE, which takes milliseconds, and drops the clone
// when the test ends. Tests therefore run in parallel, start from a clean
// schema and never touch the development database.
//
// It connects with BRAINVOLT_TEST_POSTGRES_URL (platform/testenv); the role
// needs CREATEDB (the Compose and CI superuser has it).
package testdb

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"net/url"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/testenv"
	"github.com/burakkuru5534/brainvolt/backend/migrations"
)

const (
	templatePrefix = "bvtest_tmpl_"
	clonePrefix    = "bvtest_"
	// lockKey is an arbitrary constant for pg_advisory_lock.
	lockKey = 0x62766462 // "bvdb"
)

var (
	once        sync.Once
	templateDB  string
	errTemplate error
)

// New returns a pool connected to a fresh, migrated database that is dropped
// when the test ends. The pool uses the API statement timeout (3 s).
func New(t testing.TB) *pgxpool.Pool {
	t.Helper()
	pool, err := postgres.Connect(context.Background(), postgres.Options{
		URL:              NewURL(t),
		StatementTimeout: 3 * time.Second,
		ApplicationName:  "brainvolt-test",
	})
	if err != nil {
		t.Fatalf("testdb: %v", err)
	}
	t.Cleanup(pool.Close) // runs before the database is dropped (cleanups are LIFO)
	return pool
}

// NewURL creates a fresh, migrated database and returns its connection URL,
// for code under test that opens its own connections (e.g. app wiring).
// The database is dropped when the test ends.
func NewURL(t testing.TB) string {
	t.Helper()
	adminURL := testenv.PostgresURL(t)
	ctx := context.Background()

	once.Do(func() { templateDB, errTemplate = ensureTemplate(ctx, adminURL) })
	if errTemplate != nil {
		t.Fatalf("testdb: building template: %v", errTemplate)
	}

	admin, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		t.Fatalf("testdb: %v", err)
	}
	defer admin.Close(ctx)

	name := clonePrefix + strings.ReplaceAll(ids.NewString(), "-", "")
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize()+" TEMPLATE "+pgx.Identifier{templateDB}.Sanitize()); err != nil {
		t.Fatalf("testdb: cloning template: %v", err)
	}
	t.Cleanup(func() { drop(t, adminURL, name) })
	return withDatabaseURL(adminURL, name)
}

// ensureTemplate builds the template for the current migrations if missing.
func ensureTemplate(ctx context.Context, adminURL string) (string, error) {
	hash, err := migrationsHash()
	if err != nil {
		return "", err
	}
	name := templatePrefix + hash

	admin, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		return "", err
	}
	defer admin.Close(ctx)

	if _, err := admin.Exec(ctx, "SELECT pg_advisory_lock($1)", lockKey); err != nil {
		return "", err
	}
	defer admin.Exec(context.WithoutCancel(ctx), "SELECT pg_advisory_unlock($1)", lockKey) //nolint:errcheck // best effort; the lock dies with the session

	var exists bool
	if err := admin.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", name).Scan(&exists); err != nil {
		return "", err
	}
	if exists {
		return name, nil
	}

	// Build under a temporary name and rename on success, so a crashed run
	// never leaves a half-migrated template behind.
	building := name + "_building"
	for _, stmt := range []string{
		"DROP DATABASE IF EXISTS " + pgx.Identifier{building}.Sanitize() + " WITH (FORCE)",
		"CREATE DATABASE " + pgx.Identifier{building}.Sanitize(),
	} {
		if _, err := admin.Exec(ctx, stmt); err != nil {
			return "", err
		}
	}
	if err := migrate(ctx, withDatabaseURL(adminURL, building)); err != nil {
		return "", fmt.Errorf("migrating template: %w", err)
	}
	for _, stmt := range []string{
		"ALTER DATABASE " + pgx.Identifier{building}.Sanitize() + " RENAME TO " + pgx.Identifier{name}.Sanitize(),
		"ALTER DATABASE " + pgx.Identifier{name}.Sanitize() + " IS_TEMPLATE true",
	} {
		if _, err := admin.Exec(ctx, stmt); err != nil {
			return "", err
		}
	}
	dropOldTemplates(ctx, admin, name)
	return name, nil
}

func migrate(ctx context.Context, dbURL string) error {
	pool, err := postgres.Connect(ctx, postgres.Options{URL: dbURL, ApplicationName: "brainvolt-testdb"})
	if err != nil {
		return err
	}
	defer pool.Close()
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()
	p, err := migrations.NewProvider(db)
	if err != nil {
		return err
	}
	_, err = p.Up(ctx)
	return err
}

// dropOldTemplates removes templates of earlier migration sets.
func dropOldTemplates(ctx context.Context, admin *pgx.Conn, keep string) {
	rows, err := admin.Query(ctx, "SELECT datname FROM pg_database WHERE datname LIKE $1 AND datname <> $2", templatePrefix+"%", keep)
	if err != nil {
		return
	}
	names, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return
	}
	for _, n := range names {
		_, _ = admin.Exec(ctx, "ALTER DATABASE "+pgx.Identifier{n}.Sanitize()+" IS_TEMPLATE false")
		_, _ = admin.Exec(ctx, "DROP DATABASE IF EXISTS "+pgx.Identifier{n}.Sanitize()+" WITH (FORCE)")
	}
}

func drop(t testing.TB, adminURL, name string) {
	ctx := context.Background()
	admin, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		t.Errorf("testdb: dropping %s: %v", name, err)
		return
	}
	defer admin.Close(ctx)
	if _, err := admin.Exec(ctx, "DROP DATABASE IF EXISTS "+pgx.Identifier{name}.Sanitize()+" WITH (FORCE)"); err != nil {
		t.Errorf("testdb: dropping %s: %v", name, err)
	}
}

// migrationsHash fingerprints the SQL migrations plus the River version.
func migrationsHash() (string, error) {
	fsys := migrations.SQLFiles()
	var names []string
	if err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			names = append(names, path)
		}
		return err
	}); err != nil {
		return "", err
	}
	sort.Strings(names)
	h := sha256.New()
	fmt.Fprintf(h, "river=%d\n", migrations.RiverSchemaVersion)
	for _, n := range names {
		b, err := fs.ReadFile(fsys, n)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(h, "%s %d\n", n, len(b))
		h.Write(b)
	}
	return hex.EncodeToString(h.Sum(nil))[:16], nil
}

func withDatabaseURL(raw, db string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	u.Path = "/" + db
	return u.String()
}
