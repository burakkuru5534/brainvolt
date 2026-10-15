package testdb

import (
	"strings"
	"testing"
)

// TestDatabasesAreIsolated runs in parallel: each test gets its own
// migrated database, and writes in one are invisible in the other.
func TestDatabasesAreIsolated(t *testing.T) {
	for _, name := range []string{"a", "b", "c"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			pool := New(t)
			ctx := t.Context()

			var db string
			if err := pool.QueryRow(ctx, "SELECT current_database()").Scan(&db); err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(db, clonePrefix) || strings.HasPrefix(db, templatePrefix) {
				t.Fatalf("connected to %q, want a clone", db)
			}
			var users int
			if err := pool.QueryRow(ctx, "SELECT count(*) FROM identity.users").Scan(&users); err != nil {
				t.Fatalf("schema not migrated: %v", err)
			}
			if users != 0 {
				t.Fatalf("clone not empty: %d users", users)
			}
			if _, err := pool.Exec(ctx, `INSERT INTO identity.users
				(id, firebase_uid, age_verified_at, last_seen_at, created_at, updated_at)
				VALUES (gen_random_uuid(), $1, now(), now(), now(), now())`, "fb-"+name); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestMigrationsHashIsStable(t *testing.T) {
	a, err := migrationsHash()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := migrationsHash()
	if a != b || len(a) != 16 {
		t.Fatalf("hash %q / %q", a, b)
	}
}
