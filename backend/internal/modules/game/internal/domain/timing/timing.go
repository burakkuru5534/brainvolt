// Package timing is the server timer authority and latency model of
// docs/game/timing.md (D-036). Every function takes server times only —
// no decision ever reads a client timestamp (TM07). Times are Unix ms.
package timing

import "slices"

// Caps and constants (timing.md §3–§5, §9).
const (
	// UplinkCapMS caps latency compensation, so a faked slow connection
	// gains at most this much.
	UplinkCapMS = 150
	// GraceBaseMS is added to the uplink estimate for the answer grace.
	GraceBaseMS = 100
	// ShowBufferExtraMS is added to the slowest downlink for show_at.
	ShowBufferExtraMS = 30
	// ShowBufferCapMS caps how early a question is sent before it is shown.
	ShowBufferCapMS = 250
	// RTTSamples is the window of the RTT median.
	RTTSamples = 10
	// WeakConnectionRTTMS triggers the "weak connection" warning.
	WeakConnectionRTTMS = 400
	// PingIntervalMS is the in-match ping cadence.
	PingIntervalMS = 2_000
)

// RTT keeps the last samples of one connection; the estimate is their
// median, so a single spike changes nothing.
type RTT struct {
	samples []int
}

// Add records one ping→pong round trip measured on the server clock.
func (r *RTT) Add(ms int) {
	if ms < 0 {
		return
	}
	r.samples = append(r.samples, ms)
	if len(r.samples) > RTTSamples {
		r.samples = r.samples[len(r.samples)-RTTSamples:]
	}
}

// Estimate is the median of the last samples (0 without samples; even
// counts take the lower middle, rounding in no one's favour).
func (r *RTT) Estimate() int {
	if len(r.samples) == 0 {
		return 0
	}
	s := slices.Clone(r.samples)
	slices.Sort(s)
	return s[(len(s)-1)/2]
}

// Weak reports whether the connection gets the weak-connection warning.
func (r *RTT) Weak() bool { return r.Estimate() > WeakConnectionRTTMS }

// Uplink is the capped one-way estimate used for compensation.
func Uplink(rttMS int) int { return min(max(rttMS, 0)/2, UplinkCapMS) }

// Grace is how long after the deadline a participant's answer is still
// accepted: 100–250 ms.
func Grace(rttMS int) int { return Uplink(rttMS) + GraceBaseMS }

// ShowAt is when every participant sees the question: now + a buffer that
// covers the slowest downlink, capped at 250 ms (timing.md §4).
func ShowAt(nowMS int64, rttsMS []int) int64 {
	slowest := 0
	for _, r := range rttsMS {
		slowest = max(slowest, max(r, 0)/2)
	}
	return nowMS + int64(min(slowest+ShowBufferExtraMS, ShowBufferCapMS))
}

// Deadline is show_at + the answer window (TM03).
func Deadline(showAtMS int64, windowMS int) int64 { return showAtMS + int64(windowMS) }

// Accepted reports whether an answer received at receivedMS is in time.
func Accepted(receivedMS, deadlineMS int64, rttMS int) bool {
	return receivedMS <= deadlineMS+int64(Grace(rttMS))
}

// ResponseMS is the latency-compensated answer time (timing.md §5), used
// for the speed bonus, tie-breakers, statistics and cheat signals.
func ResponseMS(receivedMS, showAtMS int64, rttMS int) int {
	return int(max(0, receivedMS-showAtMS-int64(Uplink(rttMS))))
}
