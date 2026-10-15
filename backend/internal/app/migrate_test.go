package app

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/burakkuru5534/brainvolt/backend/internal/platform/testdb"
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

	if v := strings.TrimSpace(run("version")); v != "4" {
		t.Fatalf("version = %q, want 4", v)
	}
	run("down")
	status := run("status")
	if !strings.Contains(status, "000004  pending") || !strings.Contains(status, "river job tables") {
		t.Fatalf("status after down:\n%s", status)
	}
	run("up")
	if v := strings.TrimSpace(run("version")); v != "4" {
		t.Fatalf("version after up = %q", v)
	}
	if err := runMigrate(ctx, url, "drop-everything", &bytes.Buffer{}, discard); !errors.Is(err, errUsage) {
		t.Fatalf("unknown command: %v", err)
	}
}
