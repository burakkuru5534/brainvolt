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

// D-098: the pool is capped, keeps nothing at rest and gives idle
// connections back after MaxConnIdleTime.
func TestPoolLimitsAndIdleRelease(t *testing.T) {
	pool, err := Connect(t.Context(), Options{
		URL: testenv.PostgresURL(t), ApplicationName: "brainvolt-test-pool",
		MaxConns: 3, MaxConnIdleTime: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(pool.Close)
	c := pool.Config()
	if c.MaxConns != 3 || c.MinConns != 0 || c.MaxConnIdleTime != 2*time.Second ||
		c.MaxConnLifetime != DefaultMaxConnLifetime || c.MaxConnLifetimeJitter != DefaultMaxConnLifetime/10 ||
		c.HealthCheckPeriod != time.Second {
		t.Fatalf("pool config %+v", c)
	}

	// Use three connections at once, then let them sit idle.
	conns := make([]*pgxpool.Conn, 0, 3)
	for range 3 {
		conn, err := pool.Acquire(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		conns = append(conns, conn)
	}
	if n := pool.Stat().TotalConns(); n != 3 {
		t.Fatalf("total conns %d, want 3", n)
	}
	for _, conn := range conns {
		conn.Release()
	}
	deadline := time.Now().Add(10 * time.Second)
	for pool.Stat().TotalConns() > 0 {
		if time.Now().After(deadline) {
			t.Fatalf("%d idle connections still open", pool.Stat().TotalConns())
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Defaults when nothing is set.
	def, err := Connect(t.Context(), Options{URL: testenv.PostgresURL(t)})
	if err != nil {
		t.Fatal(err)
	}
	defer def.Close()
	if d := def.Config(); d.MaxConnIdleTime != DefaultMaxConnIdleTime || d.HealthCheckPeriod != 30*time.Second {
		t.Fatalf("default pool config idle %s health %s", d.MaxConnIdleTime, d.HealthCheckPeriod)
	}
}
