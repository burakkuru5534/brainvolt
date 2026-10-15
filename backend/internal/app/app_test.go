package app

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

var discard = slog.New(slog.NewTextHandler(io.Discard, nil))

// blocker runs until ctx is cancelled and records that it stopped.
type blocker struct {
	name    string
	stopped atomic.Bool
}

func (b *blocker) Name() string { return b.name }

func (b *blocker) Run(ctx context.Context) error {
	<-ctx.Done()
	b.stopped.Store(true)
	return ctx.Err()
}

// failer returns err after delay.
type failer struct {
	delay time.Duration
	err   error
}

func (f *failer) Name() string { return "failer" }

func (f *failer) Run(ctx context.Context) error {
	select {
	case <-time.After(f.delay):
		return f.err
	case <-ctx.Done():
		return nil
	}
}

func TestRunStopsCleanlyOnCancel(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		b1, b2 := &blocker{name: "one"}, &blocker{name: "two"}
		a := New(RoleAPI, discard, b1, b2)

		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan error, 1)
		go func() { done <- a.Run(ctx) }()

		synctest.Wait()
		select {
		case err := <-done:
			t.Fatalf("Run returned before cancel: %v", err)
		default:
		}

		cancel()
		if err := <-done; err != nil {
			t.Fatalf("Run after cancel = %v, want nil", err)
		}
		if !b1.stopped.Load() || !b2.stopped.Load() {
			t.Fatal("Run returned before every component stopped")
		}
	})
}

func TestRunWithoutComponentsBlocksUntilCancel(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), time.Hour)
		defer cancel()

		start := time.Now()
		if err := New(RoleWorker, discard).Run(ctx); err != nil {
			t.Fatalf("Run = %v, want nil", err)
		}
		if got := time.Since(start); got != time.Hour {
			t.Fatalf("Run returned after %v, want 1h (context deadline)", got)
		}
	})
}

func TestRunComponentFailureStopsOthers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		boom := errors.New("boom")
		b := &blocker{name: "other"}
		a := New(RoleAPI, discard, b, &failer{delay: time.Second, err: boom})

		err := a.Run(t.Context())
		if !errors.Is(err, boom) {
			t.Fatalf("Run = %v, want error wrapping %v", err, boom)
		}
		if want := "failer: boom"; err.Error() != want {
			t.Fatalf("error text = %q, want %q", err.Error(), want)
		}
		if !b.stopped.Load() {
			t.Fatal("healthy component was not stopped after a failure")
		}
	})
}

func TestMainReturnsOneWhenBuildFails(t *testing.T) {
	build := func(*slog.Logger) (*App, error) { return nil, errors.New("bad config") }
	if code := Main(build); code != 1 {
		t.Fatalf("Main exit code = %d, want 1", code)
	}
}

func TestConstructorsSetRole(t *testing.T) {
	api, err := NewAPI(discard)
	if err != nil {
		t.Fatalf("NewAPI: %v", err)
	}
	if api.Role() != RoleAPI {
		t.Fatalf("NewAPI role = %q, want %q", api.Role(), RoleAPI)
	}

	worker, err := NewWorker(discard)
	if err != nil {
		t.Fatalf("NewWorker: %v", err)
	}
	if worker.Role() != RoleWorker {
		t.Fatalf("NewWorker role = %q, want %q", worker.Role(), RoleWorker)
	}
}
