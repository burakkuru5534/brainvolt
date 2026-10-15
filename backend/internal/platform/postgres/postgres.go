// Package postgres opens the pgx connection pool (ADR-0002) and provides the
// transaction helper used by every module's store.
//
// Every connection gets a server-side statement_timeout (docs/api/limits.md
// §3: 3 s for API requests, 60 s for worker jobs), so a slow query can never
// hold a connection longer than that, even if the caller forgets a deadline.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// connectTimeout bounds the start-up ping.
const connectTimeout = 5 * time.Second

// Options configures Connect.
type Options struct {
	URL              string
	StatementTimeout time.Duration
	// ApplicationName shows up in pg_stat_activity ("brainvolt-api", "brainvolt-worker").
	ApplicationName string
}

// Connect creates the pool and verifies the database is reachable. The
// returned error never contains the URL (it may hold a password).
func Connect(ctx context.Context, opts Options) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(opts.URL)
	if err != nil {
		return nil, errors.New("postgres: invalid connection URL")
	}
	params := cfg.ConnConfig.RuntimeParams
	if opts.StatementTimeout > 0 {
		params["statement_timeout"] = strconv.FormatInt(opts.StatementTimeout.Milliseconds(), 10)
	}
	if opts.ApplicationName != "" {
		params["application_name"] = opts.ApplicationName
	}
	// Times are stored and compared in UTC (conventions.md §1).
	params["timezone"] = "UTC"

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("postgres: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: cannot reach %s:%d: %w", cfg.ConnConfig.Host, cfg.ConnConfig.Port, err)
	}
	return pool, nil
}

// Beginner starts transactions; *pgxpool.Pool and pgx.Tx implement it, so
// InTx also works inside an outer transaction (as a savepoint).
type Beginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// InTx runs fn in a transaction: committed if fn returns nil, rolled back if
// it returns an error or panics (the panic is re-raised after rollback).
func InTx(ctx context.Context, db Beginner, fn func(tx pgx.Tx) error) (err error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(context.WithoutCancel(ctx))
			panic(p)
		}
	}()
	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(context.WithoutCancel(ctx)); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
			return errors.Join(err, fmt.Errorf("rollback: %w", rbErr))
		}
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
