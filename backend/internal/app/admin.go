package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/config"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/logging"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/redis"
)

const adminUsage = `usage: admin <command> <user> <role>

commands:
  grant-role    give a staff role (moderator or admin)
  revoke-role   take it away

<user> is a BrainVolt user id or a Firebase UID; the user must have signed
in with the app once. Needs BRAINVOLT_POSTGRES_URL and BRAINVOLT_REDIS_URL
(the user's cached permissions are cleared).`

// AdminMain is the body of cmd/admin: operator tasks that have no API.
func AdminMain(args []string, stdout io.Writer) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	logger := logging.New(os.Stderr, logging.Options{Format: logging.FormatText})
	if len(args) != 3 || (args[0] != "grant-role" && args[0] != "revoke-role") {
		fmt.Fprintln(os.Stderr, adminUsage)
		return 2
	}
	if err := runAdmin(ctx, args[0] == "grant-role", args[1], args[2], stdout); err != nil {
		logger.Error("admin failed", "command", args[0], "error", err)
		return 1
	}
	return 0
}

func runAdmin(ctx context.Context, grant bool, user, role string, stdout io.Writer) error {
	pg, err := config.LoadPostgres()
	if err != nil {
		return err
	}
	redisURL := os.Getenv("BRAINVOLT_REDIS_URL")
	if redisURL == "" {
		return fmt.Errorf("config: BRAINVOLT_REDIS_URL is not set")
	}
	db, err := postgres.Connect(ctx, postgres.Options{URL: pg.URL, ApplicationName: "brainvolt-admin"})
	if err != nil {
		return err
	}
	defer db.Close()
	rdb, err := redis.Connect(ctx, redisURL, "brainvolt-admin")
	if err != nil {
		return err
	}
	defer rdb.Close()
	logger := logging.New(os.Stderr, logging.Options{Format: logging.FormatText})
	ident := identity.New(identity.Deps{DB: db, Redis: rdb, Events: events.NewBus(), Clock: clock.Real{}, Logger: logger})
	id, err := ident.SetRole(ctx, user, role, grant)
	if err != nil {
		return err
	}
	verb := "granted"
	if !grant {
		verb = "revoked"
	}
	fmt.Fprintf(stdout, "%s %s for user %s\n", role, verb, id)
	return nil
}
