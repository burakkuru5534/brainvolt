package app

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"strconv"
	"strings"
	"testing"

	"github.com/burakkuru5534/brainvolt/backend/internal/platform/testdb"
	"github.com/burakkuru5534/brainvolt/backend/migrations"
)

func TestRunMigrate(t *testing.T) {
	url := testdb.NewURL(t) // already at the latest version
	ctx := t.Context()
	run := func(cmd string) string {
		t.Helper()
		var out bytes.Buffer
		if err := runMigrate(ctx, url, cmd, &out, discard); err != nil {
			t.Fatalf("%s: %v", cmd, err)
		}
		return out.String()
	}

	// Migration 1 is River's (Go); the others are the embedded SQL files.
	sqlFiles, err := fs.Glob(migrations.SQLFiles(), "*.sql")
	if err != nil {
		t.Fatal(err)
	}
	latest := strconv.Itoa(len(sqlFiles) + 1)
	if v := strings.TrimSpace(run("version")); v != latest {
		t.Fatalf("version = %q, want %s", v, latest)
	}
	run("down")
	status := run("status")
	if !strings.Contains(status, fmt.Sprintf("%06d  pending", len(sqlFiles)+1)) || !strings.Contains(status, "river job tables") {
		t.Fatalf("status after down:\n%s", status)
	}
	run("up")
	if v := strings.TrimSpace(run("version")); v != latest {
		t.Fatalf("version after up = %q", v)
	}
	if err := runMigrate(ctx, url, "drop-everything", &bytes.Buffer{}, discard); !errors.Is(err, errUsage) {
		t.Fatalf("unknown command: %v", err)
	}
}
