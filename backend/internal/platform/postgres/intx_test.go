package postgres_test

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/testdb"
)

func TestInTx(t *testing.T) {
	pool := testdb.New(t) // throw-away database, dropped after the test
	ctx := t.Context()
	const table = "intx_probe"
	if _, err := pool.Exec(ctx, "CREATE TABLE "+table+" (v text PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}

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

	if err := postgres.InTx(ctx, pool, insert("committed")); err != nil {
		t.Fatalf("commit: %v", err)
	}

	boom := errors.New("business rule failed")
	err := postgres.InTx(ctx, pool, func(tx pgx.Tx) error {
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
		_ = postgres.InTx(ctx, pool, func(tx pgx.Tx) error {
			_ = insert("panicked")(tx)
			panic("bug")
		})
	}()

	// Nested call runs as a savepoint inside the outer transaction.
	err = postgres.InTx(ctx, pool, func(tx pgx.Tx) error {
		if err := insert("outer")(tx); err != nil {
			return err
		}
		_ = postgres.InTx(ctx, tx, func(inner pgx.Tx) error {
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
