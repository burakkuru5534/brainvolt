# Game rules — test catalogue

- **Roadmap task:** 0.3.11
- **Status:** Accepted (2026-10-15)
- **Purpose:** every rule fixed in 0.3.1–0.3.10 has at least one test here, with a stable ID. Phase 3+ implementations reference these IDs in test names (e.g. `TestLifecycle_LC03_ReadyDeadlineCancels`). A rule change must update the matching tests in the same commit.
- **Types:** **U** = pure unit test (domain functions, fake clock), **I** = integration (Postgres + Redis, real match owner), **E** = end-to-end over WebSocket.
- **Clock:** every timing test uses the `platform/clock` fake or `testing/synctest` (ADR-0002); no sleeps.

| Prefix | Source document |
|---|---|
| LC | match-lifecycle.md (0.3.1) |
| RD | round-lifecycle.md (0.3.2) |
| TM | timing.md (0.3.3) |
| SC | scoring.md (0.3.4) |
| MT | match-types.md (0.3.5) |
| TA | team-modes.md (0.3.6) |
| CM | category-modes.md (0.3.7) |
| DC | category-modes.md §5 Daily Challenge (0.3.7) |
| CP | couple-contest.md (0.3.8) |
| EC | edge-cases.md (0.3.9) |
| PF | progression-formulas.md (0.3.10) |

---

## LC — match lifecycle (Phase 3 U, Phase 4 I/E)
| ID | Case | Type |
|---|---|---|
| LC01 | Only the 7 transitions of match-lifecycle §1 are possible; every other (state, event) pair is ignored and logged | U |
| LC02 | All participants ready before 20 s → `ready`, 3 s countdown → `in_progress` | U |
| LC03 | Ready deadline passes with one participant not ready → `cancelled/not_all_ready`; ready players re-queued at the front | U+I |
| LC04 | Duplicate `ready` from the same user → second ignored | U |
| LC05 | `ready` received 1 ms after the deadline → rejected (deadline wins, compare-and-set) | U |
| LC06 | Participant already in a live match → creation rejected, no rows, no locks leaked | I |
| LC07 | Leave during countdown → `cancelled/participant_left_before_start`, nothing awarded | U |
| LC08 | Crash after Postgres commit of `completed`, before Redis cleanup → sweeper cleans; event emitted exactly once | I |
| LC09 | Duplicate delivery of `MatchCompleted` to ranking/progression → applied once | I |
| LC10 | Completed match invariants (ended_at, end_reason, results, winner xor draw) hold for every end reason | U |
| LC11 | 3 missed ready checks in 30 min (matchmade) → 5 min queue cooldown; not a sanction | I |
| LC12 | Creation guard: ranked without active season / sanction `ranked_ban` / not enough eligible questions → rejected | U+I |

## RD — round lifecycle (Phase 3 U, Phase 4 E)
| ID | Case | Type |
|---|---|---|
| RD01 | All expected answerers answer early → lock before deadline (early close) | U |
| RD02 | One answers, other doesn't → lock at deadline + grace, other gets timeout row | U |
| RD03 | Submission at exactly `deadline_at + grace_p` accepted; +1 ms rejected `late` | U |
| RD04 | Duplicate submission → `duplicate`, first answer stands | U |
| RD05 | Option of another question → `invalid_option` | U |
| RD06 | `correct_option_id` never appears in any outgoing message before `round.revealed` (message-capture test) | E |
| RD07 | Revealed round persisted before next round starts; persist failure retried then `cancelled/server_error` | I |
| RD08 | Event stream contains every phase change in order | I |
| RD09 | Reveal lasts 3 s before next round | U |
| RD10 | Opponents see only "answered ✓" during `question` | E |

## TM — timing (Phase 4)
| ID | Case | Type |
|---|---|---|
| TM01 | Equal tap time with RTT 40 ms vs 240 ms → equal `response_ms` (timing.md §5 example) | U |
| TM02 | Single RTT spike → median unchanged; sustained inflation → compensation stops at 150 ms | U |
| TM03 | `deadline_at = show_at + window` for every round; `show_at` buffer ≤ 250 ms | U |
| TM04 | Grace per participant within 100–250 ms | U |
| TM05 | Takeover re-arms deadlines from Redis; no extra answer time beyond original deadline + grace | I |
| TM06 | Client clock offset 5 s wrong → question shown at the correct moment (client unit test with fake channel) | U (mobile) |
| TM07 | No server decision reads a client timestamp (static check: domain functions take only server times) | U |

## SC — scoring ruleset 1 (Phase 3)
| ID | Case | Type |
|---|---|---|
| SC01 | Every row of scoring.md §3 reproduces exactly | U |
| SC02 | Wrong / timeout → 0 points, streak 0 | U |
| SC03 | Speed bonus monotonic non-increasing, 0…100 (MC) / 0…50 (TF) | U |
| SC04 | Max 260 (MC) / 150 (TF) per answer | U |
| SC05 | Streak bonus thresholds 3 → +20, 5 → +40 | U |
| SC06 | `difficulty_used` fixed at selection; recalibration later doesn't change stored scores | I |

## MT — match types (Phase 3)
| ID | Case | Type |
|---|---|---|
| MT01 | Quick Duel ends after 7, Ranked after 10, Marathon after 25 | U |
| MT02 | Sudden Death ends on first round with exactly one side correct; both right / both wrong continue | U |
| MT03 | Sudden Death round 30 tie → highest score; then edge-case tie-breakers | U |
| MT04 | Ranked: only MC, verified question + verified translation for every participant locale | I |
| MT05 | Ranked with origin ≠ matchmaking → creation rejected | U |
| MT06 | Selection: no consecutive sub-category; category caps 2/3/5; TF ≤ 30 % never consecutive | U |
| MT07 | Difficulty curve followed; unavailable difficulty → nearest lower then higher, logged | U |
| MT08 | Shared Answer rounds get window + 5 s | U |
| MT09 | Marathon 15 s half-time break after question 12 | U |

## TA — team answer modes (Phase 3 U, Phase 6 E)
| ID | Case | Type |
|---|---|---|
| TA01 | Same-scale: side round points ≤ 260 for every mode and side size | U |
| TA02 | team-modes.md §5 example (average 107, fastest 180, majority 160, shared 155, turns 0) | U |
| TA03 | Shared Answer: lock wins; no lock → most-picked; tie → earliest; none → timeout; fallback speed bonus 0 | U |
| TA04 | Shared Answer contributions sum to side points; remainder to locker | U |
| TA05 | Individual Turns: non-turn submission → `not_your_turn`; rotation 3 seats × 7 rounds wraps symmetrically | U |
| TA06 | Individual Turns: disconnected/left turn holder → next active seat; none active → side timeout | U |
| TA07 | Fast Team Round: teammates see only "answered" until reveal | E |
| TA08 | Average: correct if ≥ 50 % of active members correct; divisor = full side size | U |
| TA09 | Ranked & matchmade casual always `average`; host choice honoured in rooms/challenges/tournaments | U |

## CM — category modes (Phase 8)
| ID | Case | Type |
|---|---|---|
| CM01 | Blind Challenge: category hidden in `round.started`, shown in `round.revealed` | E |
| CM02 | Category Master matchmaking pairs only identical category ids; top-level only | I |
| CM03 | Mixed Battle: picks alternate sides; only the 3 lobby categories offered; 4th consecutive same pick rejected; no pick → random of the 3 | U |
| CM04 | Mixed Battle in teams: turn holder picks; answer follows configured team mode | U |
| CM05 | Creator Quiz: author order kept; window 10/15/20 s; author gets 0 XP; unverified quiz room rejects non-friends of host | U+I |
| CM06 | Revenge: 7 questions → 3 home; no 4 consecutive same-category; Ranked parent → unrated; 4th revenge in chain rejected | U |
| CM07 | Ranked with any category mode → rejected | U |

## DC — Daily Challenge (Phase 8)
| ID | Case | Type |
|---|---|---|
| DC01 | First completed run = official; later runs = practice with no board, XP, quests, streak | I |
| DC02 | Two concurrent "start official" requests → exactly one succeeds, other `DAILY_ATTEMPT_IN_PROGRESS` | I |
| DC03 | Day boundary 00:00 Europe/Istanbul; run started 23:59:50 belongs to its start day | U |
| DC04 | Restricted account official run stored, hidden from public boards, visible after upgrade | I |
| DC05 | Ranking: more correct > fewer; equal correct → lower total time; equal → earlier completion | U |
| DC06 | Timeouts add 15 000 ms to total time | U |
| DC07 | Redis score encoding exact for 0–10 correct × 0–150 000 ms | U |
| DC08 | Server error mid-run → `void_server_error`, attempt not consumed, new official allowed | I |
| DC09 | Drop before Q1 `show_at` → `void_not_started`, not consumed | I |
| DC10 | Drop after Q1, resume within 2 min → continues with original deadlines | I |
| DC11 | Drop after Q1, no resume → completed, remaining timeouts, attempt consumed | I |
| DC12 | > 3 server-error voids in one day → moderation signal | I |
| DC13 | Daily run writes nothing to `game.matches` / `points_ledger` / `season_standings` | I |

## CP — Couple Contest (Phase 6)
| ID | Case | Type |
|---|---|---|
| CP01 | Couple side never faces a non-couple side | U |
| CP02 | Ranked couple ⇒ Fast Team Round + average, no category mode, matchmaking | U |
| CP03 | Sync Bonus +20 only in Fast Team Round when both correct; capped at 260 | U |
| CP04 | Ranked couple match updates couple entity + both personal couple ratings only | I |
| CP05 | Re-forming same pair reactivates old couple id; new partner within 7 days rejected | I |

## EC — edge cases (Phase 4–6)
| ID | Case | Type |
|---|---|---|
| EC01 | Equal score: more correct wins; then lower total correct response time; then draw | U |
| EC02 | Tournament / revenge tie → golden rounds (max 5) → higher seed / series draw | U |
| EC03 | Disconnect 19 s (window 20 s) rejoins with original deadlines; 21 s → forfeited | U+E |
| EC04 | 4th disconnect / away > 2× window → forfeit at that moment | U |
| EC05 | 1v1 voluntary leave → opponent wins, `all_opponents_left` | U |
| EC06 | 3v3: one forfeits → continue; all three gone → side forfeits | U |
| EC07 | Both 1v1 players gone in same check → `cancelled/all_sides_absent` | U |
| EC08 | Owner killed during `question` → round voided, fresh round, count unchanged | I |
| EC09 | No takeover in 30 s → `cancelled/server_error`, nothing awarded | I |
| EC10 | Leaver counter: 2nd in 24 h → 5 min, 3rd → 15 min cooldown | I |
| EC11 | Ranked loss with leaver teammate → stayed members SP 0 change | U |

## PF — progression formulas (Phase 7)
| ID | Case | Type |
|---|---|---|
| PF01 | Glicko-2 reproduces Glickman's worked example to 4 decimals | U |
| PF02 | Composite opponent update (3v3); draw 0.5; forfeit = loss | U |
| PF03 | RD bounded 50…350; inactivity inflation once per idle day | U |
| PF04 | SP table incl. streak bonus; league floor protection; leaver protection; void reversal | U |
| PF05 | League/division boundaries for every threshold | U |
| PF06 | Leaderboard points only for `origin = matchmaking`; category shares sum correctly | U |
| PF07 | XP table incl. leaver, daily, author-zero, ad bonus cap; no premium boost | U |
| PF08 | Level totals: L2 100, L10 1 800, L50 34 300, L100 131 175; cap 100 | U |
| PF09 | Season close: SP reset, RD ≥ 150, rewards by peak league, boards archived | I |
| PF10 | progression-formulas.md §10 worked examples | U |

---

## Coverage rule
CI (D-010) runs all **U** and **I** tests on every push; **E** tests run on the Phase 4+ pipeline. A pull request that changes a file in `docs/game/` must add or update tests with the matching prefix.
