package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/contentfiles"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/content"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/config"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/logging"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
)

const seedUsage = `usage: seed <command> [flags]

commands:
  validate        check the files under content/ (no database)
  import          load taxonomy, sources and questions into the database;
                  new questions start in review (schema-content §6),
                  existing ones are skipped
  opentdb-fetch   download the OpenTDB question bank into
                  content/opentdb/raw/ (needs internet; takes minutes)
  opentdb-list    print the raw OpenTDB questions not yet in
                  opentdb/selected.yaml, with the ref to copy

flags:
  -dir DIR            content directory (default ../content)
  -approve-as USER    import only, BRAINVOLT_ENV=dev only: verify every
                      imported question as this user id, so local matches
                      have questions before anyone reviewed them`

// SeedMain is the body of cmd/seed.
func SeedMain(args []string, stdout io.Writer) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	logger := logging.New(os.Stderr, logging.Options{Format: logging.FormatText})
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, seedUsage)
		return 2
	}
	fs := flag.NewFlagSet("seed", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	dir := fs.String("dir", "../content", "content directory")
	approveAs := fs.String("approve-as", "", "dev only: verify imported questions as this user id")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	var err error
	switch args[0] {
	case "validate":
		err = seedValidate(*dir, stdout)
	case "import":
		err = seedImport(ctx, *dir, *approveAs, stdout, logger)
	case "opentdb-fetch":
		err = seedFetch(ctx, *dir, stdout)
	case "opentdb-list":
		err = contentfiles.ListOpenTDB(*dir, stdout)
	default:
		fmt.Fprintln(os.Stderr, seedUsage)
		return 2
	}
	if err != nil {
		var probs contentfiles.ValidationErrors
		if errors.As(err, &probs) {
			fmt.Fprintf(os.Stderr, "%d problem(s) in %s:\n%s\n", len(probs), *dir, probs)
			return 1
		}
		logger.Error("seed failed", "command", args[0], "error", err)
		return 1
	}
	return 0
}

func seedValidate(dir string, stdout io.Writer) error {
	b, err := contentfiles.Load(dir)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "ok: %d categories, %d sources, %d questions\n", len(b.Categories), len(b.Sources), len(b.Questions))
	return nil
}

func seedImport(ctx context.Context, dir, approveAs string, stdout io.Writer, logger *slog.Logger) error {
	var approver uuid.UUID
	if approveAs != "" {
		if config.Environment(os.Getenv("BRAINVOLT_ENV")) != config.EnvDev {
			return errors.New("-approve-as is only allowed with BRAINVOLT_ENV=dev")
		}
		id, err := uuid.Parse(approveAs)
		if err != nil {
			return fmt.Errorf("-approve-as: want a BrainVolt user id: %w", err)
		}
		approver = id
	}
	b, err := contentfiles.Load(dir)
	if err != nil {
		return err
	}
	pg, err := config.LoadPostgres()
	if err != nil {
		return err
	}
	db, err := postgres.Connect(ctx, postgres.Options{URL: pg.URL, ApplicationName: "brainvolt-seed"})
	if err != nil {
		return err
	}
	defer db.Close()
	// The bus has no subscribers yet; once review events have consumers the
	// seed gets an insert-only River client like the API.
	m := content.New(db, clock.Real{}, events.NewBus())
	return importBundle(ctx, m, b, approver, stdout, logger)
}

// importBundle loads b; approver != Nil verifies every question (dev).
func importBundle(ctx context.Context, m *content.Module, b *contentfiles.Bundle, approver uuid.UUID, stdout io.Writer, logger *slog.Logger) error {
	if err := m.UpsertSources(ctx, b.SourceSpecs()); err != nil {
		return err
	}
	if err := m.UpsertTaxonomy(ctx, b.CategorySpecs()); err != nil {
		return err
	}
	subIDs := map[string]uuid.UUID{}
	var created, skipped, approved int
	for i := range b.Questions {
		q := &b.Questions[i]
		key := q.Category + "/" + q.Subcategory
		sub, ok := subIDs[key]
		if !ok {
			id, err := m.SubcategoryID(ctx, q.Category, q.Subcategory)
			if err != nil {
				return err
			}
			subIDs[key], sub = id, id
		}
		d := q.Draft
		d.SubcategoryID = sub
		res, err := m.Import(ctx, d)
		if err != nil {
			return fmt.Errorf("%s: %s: %w", q.File, q.Ref, err)
		}
		if res.Created {
			created++
		} else {
			skipped++
		}
		if approver != uuid.Nil {
			if err := m.ApproveAll(ctx, approver, res.ID); err != nil {
				return fmt.Errorf("%s: %s: approve: %w", q.File, q.Ref, err)
			}
			approved++
		}
	}
	logger.InfoContext(ctx, "import finished", "created", created, "skipped", skipped, "approved", approved)
	fmt.Fprintf(stdout, "imported: %d new, %d already present, %d approved (dev)\n", created, skipped, approved)
	return nil
}

func seedFetch(ctx context.Context, dir string, stdout io.Writer) error {
	out := filepath.Join(dir, "opentdb", "raw", "opentdb-"+time.Now().UTC().Format("20060102")+".json")
	n, err := contentfiles.FetchOpenTDB(ctx, &http.Client{Timeout: 30 * time.Second}, "https://opentdb.com", out,
		func(format string, args ...any) { fmt.Fprintf(stdout, format+"\n", args...) })
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "wrote %d questions to %s\n", n, out)
	return nil
}
