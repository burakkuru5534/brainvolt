package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/stdlib"

	"github.com/burakkuru5534/brainvolt/backend/internal/platform/config"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/logging"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
	"github.com/burakkuru5534/brainvolt/backend/migrations"
)

const migrateUsage = `usage: migrate <command>

commands:
  up       apply all pending migrations
  down     roll back the most recent migration
  status   list migrations and whether they are applied
  version  print the current schema version`

// MigrateMain is the body of cmd/migrate. It needs only BRAINVOLT_POSTGRES_URL
// and runs without a statement timeout (DDL may take long).
func MigrateMain(args []string, stdout io.Writer) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	logger := logging.New(os.Stderr, logging.Options{Format: logging.FormatText})

	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, migrateUsage)
		return 2
	}
	pg, err := config.LoadPostgres()
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		return 1
	}
	if err := runMigrate(ctx, pg.URL, args[0], stdout, logger); err != nil {
		if errors.Is(err, errUsage) {
			fmt.Fprintln(os.Stderr, migrateUsage)
			return 2
		}
		logger.Error("migrate failed", "command", args[0], "error", err)
		return 1
	}
	return 0
}

var errUsage = errors.New("unknown command")

func runMigrate(ctx context.Context, url, command string, stdout io.Writer, logger *slog.Logger) error {
	switch command {
	case "up", "down", "status", "version":
	default:
		return errUsage
	}
	pool, err := postgres.Connect(ctx, postgres.Options{URL: url, ApplicationName: "brainvolt-migrate"})
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
	switch command {
	case "up":
		results, err := p.Up(ctx)
		for _, r := range results {
			logger.Info("applied", "version", r.Source.Version, "file", r.Source.Path, "took", r.Duration)
		}
		if err != nil {
			return err
		}
		if len(results) == 0 {
			logger.Info("schema already up to date")
		}
	case "down":
		r, err := p.Down(ctx)
		if err != nil {
			return err
		}
		logger.Info("rolled back", "version", r.Source.Version, "file", r.Source.Path, "took", r.Duration)
	case "status":
		statuses, err := p.Status(ctx)
		if err != nil {
			return err
		}
		for _, s := range statuses {
			applied := "pending"
			if !s.AppliedAt.IsZero() {
				applied = s.AppliedAt.UTC().Format("2006-01-02 15:04:05Z")
			}
			name := s.Source.Path
			if name == "" {
				name = "river job tables (Go migration)"
			}
			fmt.Fprintf(stdout, "%06d  %-22s  %s\n", s.Source.Version, applied, name)
		}
	case "version":
		v, err := p.GetDBVersion(ctx)
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, v)
	}
	return nil
}
