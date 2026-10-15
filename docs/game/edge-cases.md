# Ties, disconnects, forfeits, takeover & cancellation

- **Roadmap task:** 0.3.9
- **Status:** Draft for review (2026-10-15)
- **Owner module:** `game` (lifecycle T3/T5/T6/T7, round engine, sweeper); `matchmaking` applies queue penalties (Phase 5)
- **Builds on:** D-032 (resume token 2 min), D-034 (ready flow, no-show cooldown), D-035/D-039 (team modes, turn passing), D-036 (timing, owner lease), D-037/D-038 (scoring, match types), D-040 (Daily Challenge has its own disconnect rules — not repeated here), D-030 (signals, sanctions)
- **Related:** match-lifecycle.md, round-lifecycle.md, timing.md, team-modes.md, schema-game.md, redis-keys.md §8
- Values marked **(P)** are proposals (§8).

---

## 1. Tie-breakers (head-to-head matches)

Applied when both sides finish with the **same total score** (and, for Sudden Death, when the 30-round cap is reached with equal scores):

| Step | Rule |
|---|---|
| 1 | **More correct rounds** wins (`match_sides.correct_count`). |
| 2 | **Lower total response time on correct rounds** wins — sum of the response time that was used to score each correct round for that side (1v1: own `response_ms`; Shared Answer: lock time; Individual Turns: turn holder; Fast Team Round: `average` → mean of correct members, `fastest_correct` → fastest, `majority` → median of majority choosers). |
| 3 | Still equal → **draw** (`is_draw = true`). |

- **Draws are allowed** in Quick Duel, Ranked, Sudden Death (at cap) and Marathon. Ranked draws count as 0.5 in Glicko-2 (0.3.10).
- **Tournaments and Revenge chains cannot end in a draw** (P): after step 2, play **golden rounds** — Sudden Death rule (first right-vs-wrong split wins), up to **5** extra questions at difficulty 4; still undecided → tournament: the **higher seed** advances; revenge: counted as a draw in the series score.

---

## 2. Disconnects during `in_progress`

| Rule | Value |
|---|---|
| Rounds pause for a disconnected player? | **No.** The match continues; their missing answers are timeouts (round-lifecycle §2). |
| Reconnect window per disconnect (P) | Quick Duel **20 s**, Ranked **30 s**, Sudden Death **20 s**, Marathon **60 s** — always ≤ the 2 min resume token (D-032). |
| What they get on reconnect | full snapshot + missed events (0.4.6), then continue at the current round with its **original** deadline. |
| Cumulative limit (anti reconnect-abuse) (P) | more than **3 disconnects** in one match, **or** total disconnected time > **2 × window** → treated as a forfeit of that participant at the moment the limit is crossed. |
| Individual Turns | a disconnected turn holder passes the turn to the next active seat (D-039). |
| Fast Team Round `average` | a disconnected member counts as 0 but the divisor stays the full side size (D-039). |
| Signal | every disconnect is counted; patterns go to `reconnect_abuse` signals (D-030). |

---

## 3. Leaving & forfeits

| Event | Participant result | Side / match result |
|---|---|---|
| Player taps **Leave** (confirm dialog) during `in_progress` | `left` immediately (no window, no rejoin) | — |
| Reconnect window expires, or cumulative limit crossed | `forfeited` | — |
| A side has **no active participant left** (all `left` / `forfeited`) | — | that side **forfeits** → T6 `completed`, opponent wins: `end_reason = all_opponents_left` if every absent member left voluntarily, otherwise `forfeit` |
| **Both** sides have no active participant at the same check | — | T7 **`cancelled`**, `cancel_reason = all_sides_absent` (new value, schema-game updated) — no winner, nothing awarded |
| Team side loses members but keeps ≥ 1 active | remaining members play on short-handed | match continues normally |

**Awards after a forfeit:**
| Who | Rated (Ranked) | SP / leaderboard (0.3.10) | XP / quests |
|---|---|---|---|
| Winning side | normal win | normal win | normal |
| Forfeiting / leaving participant | **loss** | loss | XP only for rounds actually played (P); no "win/complete match" quests |
| Remaining teammates who stayed on a losing side (P) | loss (rating) | **no SP loss** ("leaver protection") | normal |

**Leaver penalties (Ranked & matchmade casual) (P):** leaving or forfeiting a matchmade match increments a leaver counter; within **24 h**: 1st → no penalty, 2nd → **5 min** matchmaking cooldown, 3rd and more → **15 min** each. Queue-health measure like D-034, not a moderation sanction; repeated patterns still produce `reconnect_abuse` signals for human review (D-030).

---

## 4. Owner loss & takeover

| Situation | Handling |
|---|---|
| Owner instance dies / lease (10 s, D-036) expires | Another instance acquires `game:{m:<id>}:owner` (`SET NX`) via the sweeper (`game:live`), loads Redis state, re-arms timers from stored absolute deadlines. |
| Outage during an **open round** (`question` phase) | That round is **voided**: not scored, not persisted, no streak change; the same question is **not** reused; a **new** round starts with fresh `show_at` / `deadline_at`. Participants receive `match.resumed` (P). |
| Outage during `reveal` / between rounds | the revealed round is already persisted (or is persisted by the new owner from Redis); play continues with the next round. |
| No takeover within **30 s** of lease expiry (P) | sweeper cancels the match: T7 `server_error`. |
| Redis state missing / corrupted | T7 `server_error` from the Postgres row (redis-keys §8). |
| Round persistence keeps failing | T7 `server_error` (round-lifecycle §5). |

Voided rounds are logged in `match_transitions` (`reason = round_voided_takeover`) so audits can see them. Fixed-length match types add a replacement round, so the question count stays as configured.

---

## 5. Cancellation conditions — complete list

| `cancel_reason` | When | Phase |
|---|---|---|
| `not_all_ready` | ready deadline (20 s) missed by ≥ 1 participant | `waiting` |
| `participant_left_before_start` | leave/decline/disconnect before T4 | `waiting`, `ready` |
| `all_sides_absent` | no active participant on either side at the same moment | `in_progress` |
| `server_error` | unrecoverable owner/Redis/persistence failure | any |
| `admin` | staff cancels | any |
| `integrity_violation` | impossible state detected (e.g. question set corrupted, invariant broken) | any |

A cancelled match awards nothing to anyone (match-lifecycle §6) and ready players of matchmade matches return to the front of the queue (D-034) when cancelled before start.

---

## 6. Schema changes made in this task
- `game.matches.cancel_reason` CHECK adds `'all_sides_absent'`.
- `game.matches.end_reason` unchanged (`forfeit`, `all_opponents_left` already exist).
- Golden rounds are ordinary rounds (`match_rounds`), flagged in `round_results.detail` (`"golden": true`).

---

## 7. Tests (→ 0.3.11)
- Equal scores, different correct counts → more correct wins; equal correct → lower total response time wins; all equal → draw.
- Tournament tie → golden rounds; still tied after 5 → higher seed advances.
- Disconnect 19 s (window 20 s) → rejoins, original deadlines kept; 21 s → forfeited.
- 4th disconnect within one match → forfeit at that moment.
- 1v1 voluntary leave → opponent wins immediately with `all_opponents_left`.
- 3v3: one member forfeits → match continues; all three gone → side forfeits.
- Both 1v1 players disconnect and both windows expire in the same check → `cancelled/all_sides_absent`.
- Owner killed during `question` → new owner voids round, issues a fresh round, total question count unchanged.
- No takeover for 30 s → `cancelled/server_error`; nothing awarded.
- Leaver counter: 2nd leave in 24 h → 5 min cooldown; 3rd → 15 min.
- Ranked loss with a leaver teammate → stayed members keep their SP.

---

## 8. Proposals to confirm
1. **Tie-breakers:** more correct rounds → lower total correct response time → draw. Draws allowed except tournaments & revenge chains, which use up to 5 golden rounds (then higher seed advances / series counts a draw).
2. **Reconnect windows:** Quick Duel 20 s, Ranked 30 s, Sudden Death 20 s, Marathon 60 s; no pausing; more than 3 disconnects or total away time over 2× window = forfeit.
3. **Forfeit awards:** leaver gets a loss and XP only for rounds played; teammates who stayed on a losing side keep their SP (leaver protection), rating still changes.
4. **Leaver cooldown:** 2nd leave within 24 h → 5 min, 3rd+ → 15 min matchmaking cooldown (queue health, not a sanction).
5. **Takeover:** an open round interrupted by an owner failure is voided and replaced by a fresh round; no takeover within 30 s → match cancelled (`server_error`).
6. **Both sides absent** → match cancelled (new reason `all_sides_absent`), nothing awarded.
