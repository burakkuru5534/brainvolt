// Package events delivers domain events between modules through a
// transactional outbox on River (backend-modules.md §3.2, ADR-0002).
//
// A producer calls PublishTx inside the transaction that changes its state;
// the bus inserts one River job per subscriber of that event in the same
// transaction, so the event exists if and only if the change commits. The
// worker process (cmd/worker) runs the jobs and calls each subscriber's
// handler. Delivery is at-least-once: handlers must be idempotent, keyed by
// Envelope.ID. An event without subscribers inserts nothing.
//
// Subscriptions are registered at start-up in internal/app, identically in
// the API (to know whom to address) and the worker (to know what to run).
package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
)

// Envelope is what a subscriber receives.
type Envelope struct {
	ID         string          `json:"id"`          // UUIDv7, the idempotency key for handlers
	Name       string          `json:"name"`        // e.g. "identity.UserRegistered"
	OccurredAt time.Time       `json:"occurred_at"` // producer's clock
	Payload    json.RawMessage `json:"payload"`     // the producer's versioned event struct as JSON
}

// Decode unmarshals the payload into v.
func (e Envelope) Decode(v any) error { return json.Unmarshal(e.Payload, v) }

// Handler processes one delivery; returning an error retries it with backoff.
type Handler func(ctx context.Context, e Envelope) error

// deliveryArgs is the River job: one event for one subscriber.
type deliveryArgs struct {
	Subscriber string   `json:"subscriber"`
	Event      Envelope `json:"event"`
}

// deliveryKind is the River job kind of every event delivery.
const deliveryKind = "event_delivery"

func (deliveryArgs) Kind() string { return deliveryKind }

type subscription struct {
	name    string
	handler Handler
}

// Bus holds the subscriptions and publishes events.
type Bus struct {
	subs   map[string][]subscription // event name → subscribers
	byName map[string]Handler        // subscriber name → handler
	client *river.Client[pgx.Tx]     // insert-only in the API, running in the worker
}

// NewBus returns an empty bus.
func NewBus() *Bus {
	return &Bus{subs: map[string][]subscription{}, byName: map[string]Handler{}}
}

// Subscribe registers handler under a unique subscriber name (e.g.
// "progression.on_user_registered") for an event name. Call it at start-up,
// before Attach.
func (b *Bus) Subscribe(event, subscriber string, handler Handler) {
	if _, dup := b.byName[subscriber]; dup {
		panic(fmt.Sprintf("events: subscriber %q registered twice", subscriber))
	}
	b.byName[subscriber] = handler
	b.subs[event] = append(b.subs[event], subscription{name: subscriber, handler: handler})
}

// Subscribers lists the subscriber names of an event, sorted.
func (b *Bus) Subscribers(event string) []string {
	names := make([]string, 0, len(b.subs[event]))
	for _, s := range b.subs[event] {
		names = append(names, s.name)
	}
	sort.Strings(names)
	return names
}

// Attach sets the River client used by PublishTx.
func (b *Bus) Attach(c *river.Client[pgx.Tx]) { b.client = c }

// Worker returns the River worker that dispatches deliveries to handlers;
// register it in the worker process's river.Workers.
func (b *Bus) Worker() river.Worker[deliveryArgs] {
	return river.WorkFunc(func(ctx context.Context, job *river.Job[deliveryArgs]) error {
		h, ok := b.byName[job.Args.Subscriber]
		if !ok {
			// Subscriber removed since the job was queued: nothing left to do.
			return river.JobCancel(fmt.Errorf("events: unknown subscriber %q", job.Args.Subscriber))
		}
		return h(ctx, job.Args.Event)
	})
}

// AddWorker registers the delivery worker in workers.
func (b *Bus) AddWorker(workers *river.Workers) {
	river.AddWorker(workers, b.Worker())
}

// PublishTx queues event for every subscriber inside tx. payload is the
// producer's event struct, marshalled to JSON.
func (b *Bus) PublishTx(ctx context.Context, tx pgx.Tx, name string, occurredAt time.Time, payload any) error {
	subs := b.subs[name]
	if len(subs) == 0 {
		return nil
	}
	if b.client == nil {
		return errors.New("events: bus has subscribers but no River client attached")
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("events: encoding %s: %w", name, err)
	}
	env := Envelope{ID: ids.NewString(), Name: name, OccurredAt: occurredAt.UTC(), Payload: raw}
	params := make([]river.InsertManyParams, 0, len(subs))
	for _, s := range subs {
		params = append(params, river.InsertManyParams{
			Args:       deliveryArgs{Subscriber: s.name, Event: env},
			InsertOpts: &river.InsertOpts{MaxAttempts: 25},
		})
	}
	if _, err := b.client.InsertManyTx(ctx, tx, params); err != nil {
		return fmt.Errorf("events: queueing %s: %w", name, err)
	}
	return nil
}

// Runner runs a River client as an app.Component in the worker process.
type Runner struct {
	Client *river.Client[pgx.Tx]
	// StopTimeout bounds the wait for running jobs on shutdown.
	StopTimeout time.Duration
}

// Name implements app.Component.
func (r Runner) Name() string { return "river" }

// Run starts job processing and stops it gracefully when ctx is cancelled.
func (r Runner) Run(ctx context.Context) error {
	if err := r.Client.Start(ctx); err != nil {
		return err
	}
	<-ctx.Done()
	stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), r.StopTimeout)
	defer cancel()
	return r.Client.Stop(stopCtx)
}
