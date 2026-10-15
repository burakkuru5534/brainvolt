# Server timer authority & latency model

- **Roadmap task:** 0.3.3
- **Status:** Draft for review (2026-10-15)
- **Owner module:** `game` (uses `platform/clock`; RTT measured by `realtime`)
- **Related:** round-lifecycle.md (phases), match-lifecycle.md (ready deadline, countdown), scoring 0.3.4 (speed bonus uses `response_ms`), anti-cheat D-030 ("too fast" threshold uses `response_ms`), WS heartbeat contract 0.4.3
- Values marked **(P)** are proposals (§9).

---

## 1. Principles

1. **Only server time counts.** Every deadline, every "who was faster", every timeout is decided from timestamps taken **by the server when it sends or receives**. Client-reported times are never used for decisions (at most logged for anti-cheat analysis).
2. **Clients only display time.** The countdown on the phone is a picture of a server deadline; if it disagrees with the server, the server wins.
3. **Latency must not decide matches.** A player with a 150 ms connection should not lose to a player with 20 ms just because of the network. We compensate — but only up to a **cap**, so nobody can gain time by faking a slow connection.
4. **Deterministic in tests.** All timing logic reads time through `platform/clock` (real clock in production, controllable fake in tests; `testing/synctest` for goroutine timers — ADR-0002).

---

## 2. Server clock

| Need | Source |
|---|---|
| Durations, timer expiry inside one process | Go monotonic clock (`time.Now()` carries it; never affected by wall-clock jumps) |
| Timestamps stored or sent (`dispatched_at`, `show_at`, `deadline_at`, `received_at`) | UTC wall clock, Unix **milliseconds** |
| Agreement between server instances (takeover after owner loss) | All hosts run NTP (chrony); deployment check alerts if offset > **50 ms** (P). Deadlines are stored as absolute UTC ms in Redis, so a new owner re-arms timers from them. |

---

## 3. Measuring each connection (RTT)

During a match the server sends a WebSocket **ping** every **2 s** (P) (outside matches: presence heartbeat every 20 s, redis-keys §4). The client answers immediately with a **pong**.

- One sample = server receive time of pong − server send time of ping (both server clock).
- `rtt_est` per connection = **median of the last 10 samples**, recomputed on every pong.
- `uplink_est` = `min(rtt_est / 2, 150 ms)` (P) — the estimated time an answer spends travelling to the server, **capped**.
- Stored in the match owner's memory and in `:participants` (Redis) so a takeover keeps it.

**Why median + cap:** a cheater could delay pongs to inflate their RTT and earn extra compensation. The median ignores spikes, and the 150 ms cap bounds the maximum benefit. Players whose `rtt_est` stays above **400 ms** (P) get a "weak connection" warning; ranked matchmaking may prefer opponents with similar latency (Phase 5).

---

## 4. Showing questions at the same moment (`show_at`)

Players far from the server receive a question later than players near it. To make everyone **see** it at the same moment:

1. The owner prepares the round and computes
   `show_at = now + buffer`, where `buffer = min(max(downlink_est of all participants) + 30 ms, 250 ms)` (P) and `downlink_est = rtt_est / 2`.
2. It sends `round.started {question, options, show_at, deadline_at}` to everyone immediately.
3. Each app hides the question until **its estimate of server time** reaches `show_at` (clock offset, §6), then shows it and starts the countdown.
4. `deadline_at = show_at + answer_window_ms` (answer window per match type, 0.3.5).

The buffer is capped at 250 ms so a modified client that reads the payload early gains at most a quarter of a second of reading time; the "too fast" anti-cheat signal (D-030) also uses `show_at` as its start.

---

## 5. Accepting answers & measuring response time

For a submission from participant *p*:

| Quantity | Definition |
|---|---|
| `received_at` | server UTC ms when the owner's connection handler read the message |
| `grace_p` | `uplink_est_p + 100 ms` (P) → between **100 ms** and **250 ms** |
| **Accepted** if | phase is `question` **and** `received_at ≤ deadline_at + grace_p` (plus the other checks in round-lifecycle §5) |
| `response_ms` | `max(0, received_at − show_at − uplink_est_p)` |
| Timeout row | no accepted answer when the round locks → `option_id NULL`, `response_ms NULL`, `is_correct false` |

`response_ms` is what scoring (speed bonus, 0.3.4), tie-breakers (0.3.9), answer statistics (content) and the superhuman-speed signal (D-030) use. It is stored in `game.answers.response_ms`.

**Example:** window 15 s, `show_at` = 10 000. A has `rtt_est` 40 ms (uplink 20), B has 240 ms (uplink capped at 120).
A's answer arrives at 14 020 → `response_ms` = 14 020 − 10 000 − 20 = **4 000**.
B's answer arrives at 14 120 → `response_ms` = 14 120 − 10 000 − 120 = **4 000** → equally fast, as both tapped after 4 s on screen.
Deadline 25 000: A accepted until 25 120, B until 25 220.

---

## 6. Client clock offset (display only)

The client estimates `server_now ≈ local_now + offset`:
- The server includes its time `t_s` in each ping; the client stamps local send/receive of its pong cycle and keeps the sample with the **smallest round trip** among the last 10 (NTP-style). `offset = t_s + rtt/2 − local_receive`.
- Used **only** to decide when to show the question (`show_at`) and to draw countdowns for `deadline_at`, ready deadline, start countdown, reveal timer.
- The client **does not block** an answer at its own countdown zero; it keeps the buttons active until it receives `round.locked` or local time passes `deadline_at + 250 ms`, and lets the server decide.

---

## 7. All server timers in a match

| Timer | Duration | Defined in |
|---|---|---|
| Ready deadline | 20 s after creation | D-034 |
| Start countdown | 3 s | D-034 |
| Pre-round pick (modes that use it) | 10 s (P) | 0.3.7 may adjust |
| `show_at` buffer | ≤ 250 ms | §4 |
| Answer window | per match type | 0.3.5 |
| Grace | 100–250 ms per participant | §5 |
| Reveal | 3 s | D-035 |
| Reconnect window | ≤ 2 min resume token (D-032); exact game value | 0.3.9 |
| Marathon time limit | per config | 0.3.5 |

Every timer is an **absolute server timestamp** stored in Redis (`:state` / `:round`), armed in the owner, re-armed after takeover.

---

## 8. Invariants & tests
**Invariants**
1. No decision uses a client-supplied timestamp.
2. `deadline_at = show_at + answer_window_ms` for every round.
3. An answer is accepted iff `received_at ≤ deadline_at + grace_p` (and other checks pass); `grace_p ≤ 250 ms`.
4. `0 ≤ response_ms ≤ answer_window_ms + 250`.
5. Compensation per participant never exceeds the caps (uplink 150 ms, show buffer 250 ms).

**Tests (→ 0.3.11), all with the fake clock**
- Exact-boundary acceptance: at `deadline_at + grace_p` accepted, +1 ms rejected.
- Equal tap time with different latencies → equal `response_ms` (example in §5).
- RTT spike in one sample → median unchanged; sustained inflation → compensation stops at cap.
- Takeover: new owner re-arms deadline from Redis; no round gets extra time beyond the original deadline + grace.
- Client clock 5 s wrong → still sees question at the correct moment after offset estimation (simulated).

---

## 9. Proposals to confirm
1. In-match ping every **2 s**; RTT estimate = median of last 10 samples.
2. Latency compensation capped at **150 ms** uplink; grace = uplink estimate + **100 ms** (range 100–250 ms).
3. Synchronized question display with `show_at` buffer capped at **250 ms**.
4. "Weak connection" warning when RTT stays above **400 ms**; server hosts alert if clock offset > **50 ms**.
5. Pre-round pick timer **10 s** (for modes with a pick phase).
