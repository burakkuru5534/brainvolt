package clock

import (
	"testing"
	"testing/synctest"
	"time"
)

var start = time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)

func expectFire(t *testing.T, c <-chan time.Time, want time.Time) {
	t.Helper()
	select {
	case got := <-c:
		if !got.Equal(want) {
			t.Fatalf("fired at %v, want %v", got, want)
		}
	default:
		t.Fatalf("did not fire (want %v)", want)
	}
}

func expectNoFire(t *testing.T, c <-chan time.Time) {
	t.Helper()
	select {
	case got := <-c:
		t.Fatalf("unexpected fire at %v", got)
	default:
	}
}

func TestFakeNowAndSince(t *testing.T) {
	f := NewFake(start)
	f.Advance(1500 * time.Millisecond)
	if got := f.Now(); !got.Equal(start.Add(1500 * time.Millisecond)) {
		t.Fatalf("Now = %v", got)
	}
	if got := f.Since(start); got != 1500*time.Millisecond {
		t.Fatalf("Since = %v", got)
	}
	if got := UnixMilli(f.Now()); got != start.UnixMilli()+1500 {
		t.Fatalf("UnixMilli = %d", got)
	}
}

func TestFakeTimerFiresAtDeadline(t *testing.T) {
	f := NewFake(start)
	tm := f.NewTimer(15 * time.Second) // a round deadline

	f.Advance(14999 * time.Millisecond)
	expectNoFire(t, tm.C())
	f.Advance(time.Millisecond)
	expectFire(t, tm.C(), start.Add(15*time.Second))

	f.Advance(time.Hour)
	expectNoFire(t, tm.C())
	if f.Pending() != 0 {
		t.Fatalf("Pending = %d after one-shot fired", f.Pending())
	}
}

func TestFakeTimerStopAndReset(t *testing.T) {
	f := NewFake(start)
	tm := f.NewTimer(time.Second)
	if !tm.Stop() {
		t.Fatal("Stop on pending timer = false")
	}
	f.Advance(2 * time.Second)
	expectNoFire(t, tm.C())
	if tm.Stop() {
		t.Fatal("Stop on stopped timer = true")
	}

	if tm.Reset(time.Second) {
		t.Fatal("Reset on stopped timer = true")
	}
	f.Advance(time.Second)
	expectFire(t, tm.C(), start.Add(3*time.Second))
}

func TestFakeResetDropsStaleValue(t *testing.T) {
	f := NewFake(start)
	tm := f.NewTimer(time.Second)
	f.Advance(time.Second) // fired, value not received
	tm.Reset(5 * time.Second)
	expectNoFire(t, tm.C())
	f.Advance(5 * time.Second)
	expectFire(t, tm.C(), start.Add(6*time.Second))
}

func TestFakeFiresInTimeOrderAcrossOneAdvance(t *testing.T) {
	f := NewFake(start)
	var order []string
	f.AfterFunc(3*time.Second, func() { order = append(order, "reveal") })
	f.AfterFunc(time.Second, func() { order = append(order, "grace") })
	f.AfterFunc(time.Second, func() { order = append(order, "grace-2") })
	f.AfterFunc(2*time.Second, func() {
		order = append(order, "deadline")
		// Callbacks may use the clock: this sees the deadline time and schedules more.
		if !f.Now().Equal(start.Add(2 * time.Second)) {
			t.Errorf("Now inside callback = %v", f.Now())
		}
		f.AfterFunc(500*time.Millisecond, func() { order = append(order, "chained") })
	})

	f.Advance(10 * time.Second)
	want := []string{"grace", "grace-2", "deadline", "chained", "reveal"}
	if len(order) != len(want) {
		t.Fatalf("order = %v, want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order = %v, want %v", order, want)
		}
	}
	if !f.Now().Equal(start.Add(10 * time.Second)) {
		t.Fatalf("Now after Advance = %v", f.Now())
	}
}

func TestFakeAfterFuncStop(t *testing.T) {
	f := NewFake(start)
	called := false
	tm := f.AfterFunc(time.Second, func() { called = true })
	if tm.C() != nil {
		t.Fatal("AfterFunc timer must have a nil channel")
	}
	tm.Stop()
	f.Advance(time.Minute)
	if called {
		t.Fatal("stopped AfterFunc ran")
	}
}

func TestFakeTicker(t *testing.T) {
	f := NewFake(start)
	tk := f.NewTicker(2 * time.Second) // in-match ping interval

	f.Advance(2 * time.Second)
	expectFire(t, tk.C(), start.Add(2*time.Second))

	// A slow receiver loses ticks but keeps the most recent pending one.
	f.Advance(6 * time.Second)
	expectFire(t, tk.C(), start.Add(4*time.Second))
	expectNoFire(t, tk.C())

	tk.Reset(20 * time.Second) // idle ping interval
	f.Advance(19 * time.Second)
	expectNoFire(t, tk.C())
	f.Advance(time.Second)
	expectFire(t, tk.C(), start.Add(28*time.Second))

	tk.Stop()
	f.Advance(time.Minute)
	expectNoFire(t, tk.C())
}

func TestFakePanicsOnBackwardsTime(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("Set to the past did not panic")
		}
	}()
	f := NewFake(start)
	f.Set(start.Add(-time.Second))
}

func TestFakeConcurrentWaiter(t *testing.T) {
	f := NewFake(start)
	done := make(chan time.Time)
	go func() {
		tm := f.NewTimer(10 * time.Second)
		done <- <-tm.C()
	}()
	for f.Pending() == 0 {
		time.Sleep(time.Millisecond)
	}
	f.Advance(10 * time.Second)
	if got := <-done; !got.Equal(start.Add(10 * time.Second)) {
		t.Fatalf("goroutine saw %v", got)
	}
}

func TestRealClock(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var c Clock = Real{}
		t0 := c.Now()

		tm := c.NewTimer(time.Second)
		<-tm.C()
		if got := c.Since(t0); got != time.Second {
			t.Fatalf("Since = %v, want 1s", got)
		}

		fired := make(chan struct{})
		c.AfterFunc(time.Second, func() { close(fired) })
		<-fired

		tk := c.NewTicker(2 * time.Second)
		<-tk.C()
		tk.Reset(time.Second)
		<-tk.C()
		tk.Stop()
		if got := c.Since(t0); got != 5*time.Second {
			t.Fatalf("Since = %v, want 5s", got)
		}
		if tm.Stop() || tm.Reset(time.Second) {
			t.Fatal("fired timer reported pending")
		}
		tm.Stop()
	})
}
