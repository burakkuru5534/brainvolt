// Package migrations is the database schema: one global, numbered goose
// sequence for all modules (backend-modules.md §4.5, ADR-0002).
//
// SQL migrations are the embedded NNNNNN_<module>_<change>.sql files, each
// with "-- +goose Up" and "-- +goose Down" sections. River's job tables are
// applied by Go migration 000001 through River's own migrator, pinned to
// RiverSchemaVersion, so there is a single migration history and River's
// version bookkeeping stays correct.
//
// This package imports no internal package, so every package (platform,
// modules, tests) may import it (checked by internal/archtest).
package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
	"github.com/riverqueue/river/riverdriver/riverdatabasesql"
	"github.com/riverqueue/river/rivermigrate"
)

//go:embed *.sql
var sqlFiles embed.FS

// SQLFiles returns the embedded SQL migrations (read-only), e.g. for
// fingerprinting the schema in test helpers.
func SQLFiles() fs.FS { return sqlFiles }

// RiverSchemaVersion is the River schema version installed by migration 1.
// A River upgrade that adds schema versions gets a new goose migration that
// migrates River to the higher version (ADR-0001 version policy).
const RiverSchemaVersion = 8

// riverVersion is the goose version of the River migration.
const riverVersion = 1

func riverMigrator(db *sql.DB) (*rivermigrate.Migrator[*sql.Tx], error) {
	return rivermigrate.New(riverdatabasesql.New(db), nil)
}

// goMigrations returns the Go migrations. River migrations run outside a
// single transaction (River's recommendation: some steps cannot be batched).
func goMigrations() []*goose.Migration {
	return []*goose.Migration{
		goose.NewGoMigration(riverVersion,
			&goose.GoFunc{RunDB: func(ctx context.Context, db *sql.DB) error {
				m, err := riverMigrator(db)
				if err != nil {
					return err
				}
				_, err = m.Migrate(ctx, rivermigrate.DirectionUp, &rivermigrate.MigrateOpts{TargetVersion: RiverSchemaVersion})
				return err
			}},
			&goose.GoFunc{RunDB: func(ctx context.Context, db *sql.DB) error {
				m, err := riverMigrator(db)
				if err != nil {
					return err
				}
				_, err = m.Migrate(ctx, rivermigrate.DirectionDown, &rivermigrate.MigrateOpts{TargetVersion: -1})
				return err
			}},
		),
	}
}

// NewProvider returns a goose provider over db with every migration. A
// Postgres advisory lock serialises concurrent runs (several instances
// starting at once).
func NewProvider(db *sql.DB) (*goose.Provider, error) {
	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return nil, err
	}
	return goose.NewProvider(goose.DialectPostgres, db, sqlFiles,
		goose.WithGoMigrations(goMigrations()...),
		goose.WithSessionLocker(locker),
	)
}

// Check reports an error when the database is behind the embedded
// migrations (used by /readyz). A database ahead of this binary is fine:
// during a rolling deploy old instances meet the newer schema.
func Check(ctx context.Context, p *goose.Provider) error {
	current, target, err := p.GetVersions(ctx)
	if err != nil {
		return err
	}
	if current < target {
		return fmt.Errorf("database schema at version %d, binary expects %d: run migrations", current, target)
	}
	return nil
}
