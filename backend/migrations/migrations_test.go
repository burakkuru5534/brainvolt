package migrations_test

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverdatabasesql"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"

	"github.com/burakkuru5534/brainvolt/backend/internal/archtest"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/testdb"
	"github.com/burakkuru5534/brainvolt/backend/migrations"
)

var fileName = regexp.MustCompile(`^(\d{6})_([a-z]+)_([a-z0-9_]+)\.sql$`)

// TestFileConventions enforces repo-layout.md naming: NNNNNN_<module>_<change>.sql,
// a catalogued module (or "platform"), a gap-free sequence after the River
// migration (version 1), and both goose sections in every file.
func TestFileConventions(t *testing.T) {
	doc, err := os.Open("../../docs/architecture/backend-modules.md")
	if err != nil {
		t.Fatal(err)
	}
	defer doc.Close()
	rules, err := archtest.ParseRules(doc)
	if err != nil {
		t.Fatal(err)
	}

	entries, err := fs.ReadDir(migrations.SQLFiles(), ".")
	if err != nil {
		t.Fatal(err)
	}
	want := 2
	for _, e := range entries {
		m := fileName.FindStringSubmatch(e.Name())
		if m == nil {
			t.Errorf("%s: want NNNNNN_<module>_<change>.sql", e.Name())
			continue
		}
		if v, _ := strconv.Atoi(m[1]); v != want {
			t.Errorf("%s: version %d, want %d (no gaps or duplicates)", e.Name(), v, want)
		}
		want++
		if m[2] != "platform" && !slices.Contains(rules.Modules, m[2]) {
			t.Errorf("%s: %q is not a module of backend-modules.md", e.Name(), m[2])
		}
		body, _ := fs.ReadFile(migrations.SQLFiles(), e.Name())
		if !strings.Contains(string(body), "-- +goose Up") || !strings.Contains(string(body), "-- +goose Down") {
			t.Errorf("%s: needs -- +goose Up and -- +goose Down sections", e.Name())
		}
	}
}

// TestRiverVersionIsLatest fails after a River upgrade that brings new
// schema versions: add a migration that moves River to the new version.
func TestRiverVersionIsLatest(t *testing.T) {
	m, err := rivermigrate.New(riverdatabasesql.New(nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	all := m.AllVersions()
	if latest := all[len(all)-1].Version; latest != migrations.RiverSchemaVersion {
		t.Fatalf("River ships schema version %d, migrations install %d", latest, migrations.RiverSchemaVersion)
	}
}

func provider(t *testing.T, pool *pgxpool.Pool) *sql.DB {
	t.Helper()
	db := stdlib.OpenDBFromPool(pool)
	t.Cleanup(func() { db.Close() })
	return db
}

func schemaExists(t *testing.T, pool *pgxpool.Pool, name string) bool {
	t.Helper()
	var ok bool
	if err := pool.QueryRow(t.Context(), "SELECT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = $1)", name).Scan(&ok); err != nil {
		t.Fatal(err)
	}
	return ok
}

func tableExists(t *testing.T, pool *pgxpool.Pool, name string) bool {
	t.Helper()
	var ok bool
	if err := pool.QueryRow(t.Context(), "SELECT to_regclass($1) IS NOT NULL", name).Scan(&ok); err != nil {
		t.Fatal(err)
	}
	return ok
}

// TestDownAndUpAgain rolls every migration back and re-applies it, so every
// Down section is exercised, and checks the readiness check on the way.
func TestDownAndUpAgain(t *testing.T) {
	pool := testdb.New(t)
	ctx := t.Context()
	p, err := migrations.NewProvider(provider(t, pool))
	if err != nil {
		t.Fatal(err)
	}
	if err := migrations.Check(ctx, p); err != nil {
		t.Fatalf("fresh test database not up to date: %v", err)
	}

	if _, err := p.Down(ctx); err != nil {
		t.Fatal(err)
	}
	if err := migrations.Check(ctx, p); err == nil || !strings.Contains(err.Error(), "run migrations") {
		t.Fatalf("Check after one Down = %v, want behind error", err)
	}

	if _, err := p.DownTo(ctx, 0); err != nil {
		t.Fatalf("DownTo(0): %v", err)
	}
	for _, s := range []string{"identity", "profile", "notification"} {
		if schemaExists(t, pool, s) {
			t.Errorf("schema %s still exists after rolling back", s)
		}
	}
	if tableExists(t, pool, "public.river_job") {
		t.Error("river_job still exists after rolling back")
	}

	if _, err := p.Up(ctx); err != nil {
		t.Fatalf("Up again: %v", err)
	}
	if err := migrations.Check(ctx, p); err != nil {
		t.Fatal(err)
	}
	for _, tbl := range []string{"identity.users", "identity.auth_identities", "identity.user_roles",
		"profile.profiles", "profile.settings", "notification.devices", "public.river_job"} {
		if !tableExists(t, pool, tbl) {
			t.Errorf("table %s missing", tbl)
		}
	}
}

func pgCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

// TestSchemaConstraints spot-checks constraints from schema-identity.md.
func TestSchemaConstraints(t *testing.T) {
	pool := testdb.New(t)
	ctx := t.Context()
	now := time.Now()
	insertUser := func(tier, firebaseUID string) (string, error) {
		id := ids.NewString()
		_, err := pool.Exec(ctx, `INSERT INTO identity.users
			(id, firebase_uid, account_tier, age_verified_at, last_seen_at, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $4, $4, $4)`, id, firebaseUID, tier, now)
		return id, err
	}

	uid, err := insertUser("guest", "fb-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := insertUser("guest", "fb-1"); pgCode(err) != "23505" {
		t.Errorf("duplicate firebase_uid: %v, want unique violation", err)
	}
	if _, err := insertUser("vip", "fb-2"); pgCode(err) != "23514" {
		t.Errorf("unknown tier: %v, want check violation", err)
	}

	profile := `INSERT INTO profile.profiles (user_id, username, display_name, avatar_id, created_at, updated_at)
		VALUES ($1, $2, $3, 'fox', now(), now())`
	if _, err := pool.Exec(ctx, profile, uid, "Has Space", "Ali"); pgCode(err) != "23514" {
		t.Errorf("invalid username: %v, want check violation", err)
	}
	if _, err := pool.Exec(ctx, profile, uid, "oyuncu48213", "A"); pgCode(err) != "23514" {
		t.Errorf("1-char display name: %v, want check violation", err)
	}
	if _, err := pool.Exec(ctx, profile, uid, "oyuncu48213", "Misafir 4821"); err != nil {
		t.Fatalf("valid profile: %v", err)
	}

	// citext: password emails are unique regardless of case.
	ident := `INSERT INTO identity.auth_identities (user_id, provider, email, linked_at) VALUES ($1, 'password', $2, now())`
	if _, err := pool.Exec(ctx, ident, uid, "Ali@Example.com"); err != nil {
		t.Fatal(err)
	}
	other, _ := insertUser("unverified", "fb-3")
	if _, err := pool.Exec(ctx, ident, other, "ali@example.COM"); pgCode(err) != "23505" {
		t.Errorf("same email different case: %v, want unique violation", err)
	}

	// Deleting a user cascades to profile and identities.
	if _, err := pool.Exec(ctx, "DELETE FROM identity.users WHERE id = $1", uid); err != nil {
		t.Fatal(err)
	}
	var n int
	_ = pool.QueryRow(ctx, "SELECT (SELECT count(*) FROM profile.profiles) + (SELECT count(*) FROM identity.auth_identities WHERE user_id = $1)", uid).Scan(&n)
	if n != 0 {
		t.Errorf("%d dependent rows left after deleting the user", n)
	}
}

type pingArgs struct {
	Note string `json:"note"`
}

func (pingArgs) Kind() string { return "test_ping" }

// TestRiverTablesUsable inserts a job transactionally, as modules will
// (ADR-0002 outbox via river.InsertTx).
func TestRiverTablesUsable(t *testing.T) {
	pool := testdb.New(t)
	ctx := t.Context()
	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{}) // insert-only
	if err != nil {
		t.Fatal(err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.InsertTx(ctx, tx, pingArgs{Note: "hello"}, nil); err != nil {
		t.Fatalf("InsertTx: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	var kind string
	if err := pool.QueryRow(context.WithoutCancel(ctx), "SELECT kind FROM river_job").Scan(&kind); err != nil || kind != "test_ping" {
		t.Fatalf("river_job kind = %q, %v", kind, err)
	}
}
