package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/testenv"
)

func TestConnectErrorsHideURL(t *testing.T) {
	_, err := Connect(t.Context(), Options{URL: "postgres://u:secret@127.0.0.1:1/db?sslmode=disable"})
	if err == nil || !strings.Contains(err.Error(), "cannot reach 127.0.0.1:1") {
		t.Fatalf("unreachable: err = %v", err)
	}
	if strings.Contains(err.Error(), "secret") {
		t.Fatalf("error leaks password: %v", err)
	}

	_, err = Connect(t.Context(), Options{URL: "postgres://u:secret@host:notaport/db"})
	if err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatalf("invalid URL: err = %v", err)
	}
}

func connect(t *testing.T, timeout time.Duration) *pgxpool.Pool {
	t.Helper()
	pool, err := Connect(t.Context(), Options{
		URL:              testenv.PostgresURL(t),
		StatementTimeout: timeout,
		ApplicationName:  "brainvolt-test",
	})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestSessionSettings(t *testing.T) {
	pool := connect(t, 3*time.Second)
	var timeout, app, tz, version string
	err := pool.QueryRow(t.Context(),
		`SELECT current_setting('statement_timeout'), current_setting('application_name'),
		        current_setting('TimeZone'), current_setting('server_version_num')`).
		Scan(&timeout, &app, &tz, &version)
	if err != nil {
		t.Fatal(err)
	}
	if timeout != "3s" || app != "brainvolt-test" || tz != "UTC" {
		t.Fatalf("statement_timeout=%q application_name=%q TimeZone=%q", timeout, app, tz)
	}
	if !strings.HasPrefix(version, "18") {
		t.Fatalf("server_version_num = %s, want PostgreSQL 18 (ADR-0001)", version)
	}
}

func TestStatementTimeoutCancelsSlowQuery(t *testing.T) {
	pool := connect(t, 200*time.Millisecond)
	_, err := pool.Exec(t.Context(), "SELECT pg_sleep(2)")
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "57014" { // query_canceled
		t.Fatalf("err = %v, want SQLSTATE 57014 from statement_timeout", err)
	}
}

func TestInTx(t *testing.T) {
	pool := connect(t, 3*time.Second)
	ctx := t.Context()
	table := pgx.Identifier{"test_intx_" + strings.ReplaceAll(ids.NewString(), "-", "")}.Sanitize()
	if _, err := pool.Exec(ctx, "CREATE TABLE "+table+" (v text PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.WithoutCancel(ctx), "DROP TABLE "+table) })

	insert := func(v string) func(pgx.Tx) error {
		return func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, "INSERT INTO "+table+" VALUES ($1)", v)
			return err
		}
	}
	count := func() int {
		var n int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	if err := InTx(ctx, pool, insert("committed")); err != nil {
		t.Fatalf("commit: %v", err)
	}

	boom := errors.New("business rule failed")
	err := InTx(ctx, pool, func(tx pgx.Tx) error {
		if err := insert("rolled-back")(tx); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the callback's error", err)
	}

	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("panic was swallowed")
			}
		}()
		_ = InTx(ctx, pool, func(tx pgx.Tx) error {
			_ = insert("panicked")(tx)
			panic("bug")
		})
	}()

	// Nested call runs as a savepoint inside the outer transaction.
	err = InTx(ctx, pool, func(tx pgx.Tx) error {
		if err := insert("outer")(tx); err != nil {
			return err
		}
		_ = InTx(ctx, tx, func(inner pgx.Tx) error {
			_ = insert("inner")(inner)
			return boom // only the savepoint is rolled back
		})
		return nil
	})
	if err != nil {
		t.Fatalf("outer tx: %v", err)
	}

	if n := count(); n != 2 {
		t.Fatalf("rows = %d, want 2 (committed + outer)", n)
	}
	if pool.Stat().AcquiredConns() != 0 {
		t.Fatal("a connection leaked from InTx")
	}
}
