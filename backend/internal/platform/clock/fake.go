package clock

import (
	"slices"
	"sync"
	"time"
)

// Fake is a manually driven Clock for tests. Time only moves on Advance or
// Set; timers, tickers and AfterFunc callbacks due by then fire in time
// order (ties in creation order). AfterFunc callbacks run synchronously
// inside Advance, after the clock's lock is released, so they may use the
// clock themselves. Safe for concurrent use.
type Fake struct {
	mu      sync.Mutex
	now     time.Time
	seq     uint64
	waiters []*fakeWaiter
}

var _ Clock = (*Fake)(nil)

// NewFake returns a fake clock set to start.
func NewFake(start time.Time) *Fake { return &Fake{now: start} }

type fakeWaiter struct {
	clock  *Fake
	when   time.Time
	seq    uint64 // creation order, breaks ties
	period time.Duration
	ch     chan time.Time
	fn     func()
	active bool
}

// Now returns the fake time.
func (f *Fake) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now
}

// Since returns Now().Sub(t).
func (f *Fake) Since(t time.Time) time.Duration { return f.Now().Sub(t) }

// NewTimer returns a timer that fires once the fake time reaches now+d.
func (f *Fake) NewTimer(d time.Duration) Timer {
	return f.add(d, 0, make(chan time.Time, 1), nil)
}

// AfterFunc calls fn when the fake time reaches now+d.
func (f *Fake) AfterFunc(d time.Duration, fn func()) Timer {
	return f.add(d, 0, nil, fn)
}

// NewTicker returns a ticker firing every d of fake time.
func (f *Fake) NewTicker(d time.Duration) Ticker {
	if d <= 0 {
		panic("clock: non-positive interval for NewTicker")
	}
	return fakeTicker{f.add(d, d, make(chan time.Time, 1), nil)}
}

func (f *Fake) add(d, period time.Duration, ch chan time.Time, fn func()) *fakeWaiter {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	w := &fakeWaiter{clock: f, when: f.now.Add(d), seq: f.seq, period: period, ch: ch, fn: fn, active: true}
	f.waiters = append(f.waiters, w)
	return w
}

// Pending reports how many timers and tickers are armed. Tests use it to
// wait until the code under test has created its timers before advancing.
func (f *Fake) Pending() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.waiters)
}

// Advance moves the fake time forward by d, firing everything due.
func (f *Fake) Advance(d time.Duration) {
	if d < 0 {
		panic("clock: Advance with negative duration")
	}
	f.Set(f.Now().Add(d))
}

// Set moves the fake time to t (never backwards), firing everything due.
func (f *Fake) Set(t time.Time) {
	for {
		f.mu.Lock()
		if t.Before(f.now) {
			f.mu.Unlock()
			panic("clock: Set would move time backwards")
		}
		w := f.nextDue(t)
		if w == nil {
			f.now = t
			f.mu.Unlock()
			return
		}
		f.now = w.when
		fireAt := w.when
		if w.period > 0 {
			w.when = w.when.Add(w.period)
		} else {
			w.active = false
			f.remove(w)
		}
		ch, fn := w.ch, w.fn
		f.mu.Unlock()

		if ch != nil {
			select {
			case ch <- fireAt:
			default: // receiver is behind; drop, like package time
			}
		}
		if fn != nil {
			fn()
		}
	}
}

// nextDue returns the earliest waiter due at or before t. Caller holds f.mu.
func (f *Fake) nextDue(t time.Time) *fakeWaiter {
	var next *fakeWaiter
	for _, w := range f.waiters {
		if w.when.After(t) {
			continue
		}
		if next == nil || w.when.Before(next.when) || (w.when.Equal(next.when) && w.seq < next.seq) {
			next = w
		}
	}
	return next
}

// remove drops w from the waiter list. Caller holds f.mu.
func (f *Fake) remove(w *fakeWaiter) {
	f.waiters = slices.DeleteFunc(f.waiters, func(x *fakeWaiter) bool { return x == w })
}

func (w *fakeWaiter) C() <-chan time.Time { return w.ch }

func (w *fakeWaiter) Stop() bool {
	f := w.clock
	f.mu.Lock()
	defer f.mu.Unlock()
	was := w.active
	w.active = false
	f.remove(w)
	w.drain()
	return was
}

func (w *fakeWaiter) Reset(d time.Duration) bool {
	f := w.clock
	f.mu.Lock()
	defer f.mu.Unlock()
	was := w.active
	if !was {
		f.waiters = append(f.waiters, w)
	}
	w.active = true
	w.when = f.now.Add(d)
	f.seq++
	w.seq = f.seq
	w.drain()
	return was
}

// drain discards a fired-but-unreceived value so no stale tick is seen.
func (w *fakeWaiter) drain() {
	if w.ch == nil {
		return
	}
	select {
	case <-w.ch:
	default:
	}
}

type fakeTicker struct{ w *fakeWaiter }

func (t fakeTicker) C() <-chan time.Time { return t.w.ch }
func (t fakeTicker) Stop()               { t.w.Stop() }

func (t fakeTicker) Reset(d time.Duration) {
	if d <= 0 {
		panic("clock: non-positive interval for Ticker.Reset")
	}
	f := t.w.clock
	f.mu.Lock()
	t.w.period = d
	f.mu.Unlock()
	t.w.Reset(d)
}
