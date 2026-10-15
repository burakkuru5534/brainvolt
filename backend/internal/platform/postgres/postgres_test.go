package postgres

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

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
