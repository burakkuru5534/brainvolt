package events

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"

	"github.com/burakkuru5534/brainvolt/backend/internal/platform/testdb"
)

type happened struct {
	UserID string `json:"user_id"`
}

func countJobs(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM river_job").Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestPublishWithoutSubscribersIsNoop(t *testing.T) {
	bus := NewBus() // no client attached: must not be needed
	if err := bus.PublishTx(t.Context(), nil, "test.Nobody", time.Now(), happened{}); err != nil {
		t.Fatal(err)
	}
}

func TestSubscribeTwicePanics(t *testing.T) {
	bus := NewBus()
	bus.Subscribe("a", "x", func(context.Context, Envelope) error { return nil })
	defer func() {
		if recover() == nil {
			t.Fatal("duplicate subscriber name accepted")
		}
	}()
	bus.Subscribe("b", "x", func(context.Context, Envelope) error { return nil })
}

// TestOutboxDelivery publishes inside a transaction (as the API does), then
// runs a worker (as cmd/worker does) and checks every subscriber gets the
// event exactly as published; a rolled-back transaction delivers nothing.
func TestOutboxDelivery(t *testing.T) {
	pool := testdb.New(t)
	ctx := t.Context()

	type got struct {
		sub string
		env Envelope
	}
	received := make(chan got, 10)
	var failOnce sync.Once
	bus := NewBus()
	bus.Subscribe("test.Happened", "alpha.on_happened", func(_ context.Context, e Envelope) error {
		received <- got{"alpha", e}
		return nil
	})
	bus.Subscribe("test.Happened", "beta.on_happened", func(_ context.Context, e Envelope) error {
		var err error
		failOnce.Do(func() { err = errors.New("transient") }) // first attempt fails, retry succeeds
		if err != nil {
			return err
		}
		received <- got{"beta", e}
		return nil
	})
	if s := bus.Subscribers("test.Happened"); len(s) != 2 || s[0] != "alpha.on_happened" {
		t.Fatalf("Subscribers = %v", s)
	}

	// API side: insert-only client.
	inserter, err := river.NewClient(riverpgxv5.New(pool), &river.Config{})
	if err != nil {
		t.Fatal(err)
	}
	bus.Attach(inserter)

	publish := func(commit bool, userID string) {
		t.Helper()
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if err := bus.PublishTx(ctx, tx, "test.Happened", time.Now(), happened{UserID: userID}); err != nil {
			t.Fatal(err)
		}
		if commit {
			err = tx.Commit(ctx)
		} else {
			err = tx.Rollback(ctx)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	publish(false, "rolled-back")
	if n := countJobs(t, pool); n != 0 {
		t.Fatalf("rolled-back publish left %d jobs", n)
	}
	publish(true, "u-1")
	if n := countJobs(t, pool); n != 2 {
		t.Fatalf("jobs = %d, want one per subscriber", n)
	}

	// Worker side.
	workers := river.NewWorkers()
	bus.AddWorker(workers)
	worker, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues:            map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: 2}},
		Workers:           workers,
		FetchPollInterval: 200 * time.Millisecond,
		RetryPolicy:       immediateRetry{},
		Logger:            slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	runCtx, stop := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { done <- Runner{Client: worker, StopTimeout: 5 * time.Second}.Run(runCtx) }()

	seen := map[string]Envelope{}
	timeout := time.After(20 * time.Second)
	for len(seen) < 2 {
		select {
		case g := <-received:
			seen[g.sub] = g.env
		case <-timeout:
			t.Fatalf("deliveries after 20s: %v", seen)
		}
	}
	stop()
	if err := <-done; err != nil {
		t.Fatalf("runner: %v", err)
	}

	a, b := seen["alpha"], seen["beta"]
	if a.ID == "" || a.ID != b.ID || a.Name != "test.Happened" {
		t.Fatalf("envelopes differ or empty: %+v / %+v", a, b)
	}
	var p happened
	if err := b.Decode(&p); err != nil || p.UserID != "u-1" {
		t.Fatalf("payload = %+v, %v", p, err)
	}
}

// immediateRetry retries failed jobs right away so the test stays fast.
type immediateRetry struct{}

func (immediateRetry) NextRetry(*rivertype.JobRow) time.Time { return time.Now() }
