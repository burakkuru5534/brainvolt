// Package clock is the injectable server clock (docs/game/timing.md §1–2).
//
// All server timing — round deadlines, grace windows, reveal pauses, pings —
// reads time through a Clock, so tests can drive it deterministically with
// Fake. Real wraps package time. Timestamps that are stored or sent use
// UnixMilli (UTC wall clock, Unix milliseconds); durations inside one process
// use the monotonic reading that time.Now carries.
package clock

import "time"

// Clock is the source of time for all server logic.
type Clock interface {
	Now() time.Time
	Since(t time.Time) time.Duration
	// NewTimer fires once after d on Timer.C.
	NewTimer(d time.Duration) Timer
	// AfterFunc calls f once after d.
	AfterFunc(d time.Duration, f func()) Timer
	// NewTicker fires every d on Ticker.C; d must be > 0.
	NewTicker(d time.Duration) Ticker
}

// Timer is a one-shot timer. After Stop or Reset returns, no stale value
// from before the call is received on C (Go ≥ 1.23 timer semantics).
type Timer interface {
	// C is nil for timers created by AfterFunc.
	C() <-chan time.Time
	// Stop prevents the timer from firing; it reports whether it was pending.
	Stop() bool
	// Reset re-arms the timer to fire after d; it reports whether it was pending.
	Reset(d time.Duration) bool
}

// Ticker fires periodically. Like time.Ticker it drops ticks for slow receivers.
type Ticker interface {
	C() <-chan time.Time
	Stop()
	Reset(d time.Duration)
}

// UnixMilli is the wire/storage form of a server timestamp.
func UnixMilli(t time.Time) int64 { return t.UnixMilli() }

// Real is the production clock backed by package time.
type Real struct{}

var _ Clock = Real{}

// Now returns time.Now().
func (Real) Now() time.Time { return time.Now() }

// Since returns time.Since(t).
func (Real) Since(t time.Time) time.Duration { return time.Since(t) }

// NewTimer wraps time.NewTimer.
func (Real) NewTimer(d time.Duration) Timer { return realTimer{time.NewTimer(d)} }

// AfterFunc wraps time.AfterFunc; f runs in its own goroutine.
func (Real) AfterFunc(d time.Duration, f func()) Timer {
	return realTimer{time.AfterFunc(d, f)}
}

// NewTicker wraps time.NewTicker.
func (Real) NewTicker(d time.Duration) Ticker { return realTicker{time.NewTicker(d)} }

type realTimer struct{ t *time.Timer }

func (r realTimer) C() <-chan time.Time        { return r.t.C }
func (r realTimer) Stop() bool                 { return r.t.Stop() }
func (r realTimer) Reset(d time.Duration) bool { return r.t.Reset(d) }

type realTicker struct{ t *time.Ticker }

func (r realTicker) C() <-chan time.Time   { return r.t.C }
func (r realTicker) Stop()                 { r.t.Stop() }
func (r realTicker) Reset(d time.Duration) { r.t.Reset(d) }
