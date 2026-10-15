//go:build unix

package app

import (
	"bufio"
	"bytes"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestBinariesStopCleanlyOnSIGTERM builds the real cmd/* binaries, waits for
// them to start, sends SIGTERM (what Docker and Kubernetes send) and expects
// a clean exit with status 0. Unix only: Windows has no SIGTERM delivery.
func TestBinariesStopCleanlyOnSIGTERM(t *testing.T) {
	if testing.Short() {
		t.Skip("builds binaries; skipped with -short")
	}
	for _, cmd := range []string{"api", "worker"} {
		t.Run(cmd, func(t *testing.T) {
			t.Parallel()
			bin := filepath.Join(t.TempDir(), cmd)
			build := exec.CommandContext(t.Context(), "go", "build", "-o", bin, "../../cmd/"+cmd)
			if out, err := build.CombinedOutput(); err != nil {
				t.Fatalf("go build: %v\n%s", err, out)
			}

			proc := exec.CommandContext(t.Context(), bin)
			stderr, err := proc.StderrPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err := proc.Start(); err != nil {
				t.Fatal(err)
			}

			var logs bytes.Buffer
			started := make(chan struct{})
			scanDone := make(chan struct{})
			go func() {
				defer close(scanDone)
				sc := bufio.NewScanner(stderr)
				signalled := false
				for sc.Scan() {
					logs.WriteString(sc.Text() + "\n")
					if !signalled && strings.Contains(sc.Text(), "msg=starting") {
						signalled = true
						close(started)
					}
				}
			}()

			select {
			case <-started:
			case <-time.After(30 * time.Second):
				_ = proc.Process.Kill()
				t.Fatal("binary did not log start-up within 30s")
			}

			if err := proc.Process.Signal(syscall.SIGTERM); err != nil {
				t.Fatal(err)
			}
			<-scanDone
			err = proc.Wait()
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				t.Fatalf("exit status %d after SIGTERM, want 0\n%s", exitErr.ExitCode(), logs.String())
			} else if err != nil {
				t.Fatal(err)
			}
			if want := "role=" + cmd; !strings.Contains(logs.String(), "msg=stopped "+want) {
				t.Fatalf("missing clean-stop log %q\n%s", "msg=stopped "+want, logs.String())
			}
		})
	}
}
