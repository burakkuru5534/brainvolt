# Team answer mode rules — Shared Answer, Individual Turns, Fast Team Round

- **Roadmap task:** 0.3.6
- **Status:** Accepted (2026-10-15)
- **Owner module:** `game` (`domain/teammode`: one aggregator per mode)
- **Applies to:** formats `2v2`, `3v3`, `4v4`, `couple` (any side with > 1 participant)
- **Builds on:** D-019 (definitions), D-035 (round flow: Shared Answer propose/lock/fallback, Fast Team privacy, Individual Turns rotation), D-037 (per-answer scoring, ruleset 1), D-038 (Shared Answer +5 s)
- **Related:** round-lifecycle.md §2–§4, scoring.md, match-types.md (Sudden Death needs a per-side "correct"), disconnect rules 0.3.9, combinations 0.5
- Values marked **(P)** were proposed and accepted (D-039, §8).

---

## 1. What every mode must produce per round

For each side, the aggregator returns:

| Output | Used for |
|---|---|
| `side_correct` (bool) | Sudden Death elimination, side answer streak, stats |
| `side_points` (int ≥ 0) | side score (`round_results.points`) |
| per-participant `contribution` | `match_participants.score_contribution`, personal stats, MVP on results screen |
| `detail` (JSON) | `round_results.detail` — rule applied and inputs, for the results screen and audits |

**Scale rule (P):** `side_points` is always on the **same scale as one player's answer** (max 260 per round with ruleset 1). A 4v4 side does not score 4× a 1v1 player. This keeps scores comparable across formats (results screens, Sudden Death cap, tie-breaks, SP formulas in 0.3.10).

---

## 2. Shared Answer (`shared_answer`)

**Flow (D-035):** members propose live → any member locks → no lock by `deadline_at + grace` → most-proposed option (tie → earliest proposed) → no proposals → side timeout. Window = base + 5 s (D-038).

| Output | Rule |
|---|---|
| Side answer | the locked option (or fallback option) |
| `response_ms` | lock time (timing.md §5); **fallback** → `response_ms = answer_window_ms` → speed bonus **0** (P) |
| `side_correct` | side answer is correct |
| `side_points` | `ScoreAnswer(side answer, response_ms, side_streak)` — speed bonus measured against the **extended** window |
| Streak | **side streak** (consecutive rounds with `side_correct`) |
| `contribution` | `side_points` split equally among members who were connected during the round; remainder to the locker (P) |

---

## 3. Individual Turns (`individual_turns`)

**Flow (D-035):** turn holder per side rotates by seat order 1 → 2 → … → wrap, same on both sides. Only the turn holder may answer.

| Output | Rule |
|---|---|
| Side answer | the turn holder's answer (or timeout) |
| `side_correct` | turn holder's answer is correct |
| `side_points` | `ScoreAnswer(turn holder answer, response_ms, side_streak)` |
| Streak | **side streak** — continues across different turn holders (rewards the whole team keeping the run) |
| `contribution` | all of `side_points` to the turn holder |
| Turn holder **disconnected or left** (D-039) | the turn passes automatically to the **next active seat** in rotation order for that round (an "active" member = connected and still in the match at round start); the rotation then continues from that seat. If no member is active → side timeout. Reconnection rules in 0.3.9. |

---

## 4. Fast Team Round (`fast_team_round`)

**Flow (D-035):** every member answers independently; teammates see only "answered", not what, until reveal.
Each member's answer is first scored on its own: `member_points = ScoreAnswer(member answer, member response_ms, member_streak)` with a **personal streak** per member (P).

The side result then depends on the **aggregation rule** stored in `match_config.config_extra.aggregation`:

| Rule | `side_correct` | `side_points` | `contribution` |
|---|---|---|---|
| **`average`** — "total correct", normalised (P default) | ≥ half of **active** members correct (D-039; active = connected and in the match at round start) | `round(sum(member_points) / side_size)` — timeouts, wrong answers and inactive members count as 0 (dividing by the full side size means disconnecting never raises a side's average) | each member's own `member_points / side_size` |
| **`fastest_correct`** | ≥ 1 member correct | `member_points` of the fastest correct member (lowest `response_ms`; tie → earliest `received_at`) | all to that member |
| **`majority`** | the most-chosen option is correct (tie between options → the option whose first chooser was fastest; timeouts don't vote) | `ScoreAnswer(majority option, median response_ms of members who chose it, side_streak)` | split equally among members who chose the majority option |

Why these exist:
- `average` rewards everyone knowing the answer — every member matters (best default, hardest to carry).
- `fastest_correct` rewards having one expert who is fast — exciting, but lets one player carry.
- `majority` rewards consensus without talking — a middle ground.

**Who chooses the rule (P):**
| Context | Rule |
|---|---|
| Ranked (team / couple buckets) | **fixed `average`** (comparability of ratings) |
| Matchmade casual | `average` |
| Private rooms, challenges, tournaments | host / organiser chooses any of the three; default `average` |

---

## 5. Worked example (3v3, multiple choice, difficulty 3, window 15 s, no streak bonus yet)

| Member | Answer | `response_ms` | `member_points` |
|---|---|---|---|
| A1 | correct | 3 000 | 100 + 80 = 180 |
| A2 | correct | 9 000 | 100 + 40 = 140 |
| A3 | wrong | 5 000 | 0 |

| Rule | Side A correct? | Side A points |
|---|---|---|
| `average` | yes (2 of 3 ≥ half) | round(320 / 3) = **107** |
| `fastest_correct` | yes | **180** (A1) |
| `majority` | yes (correct option chosen by 2) | ScoreAnswer(correct, median(3 000, 9 000) = 6 000) = 100 + 60 = **160** |

Same round in **Shared Answer** (A2 locks the correct option at 9 000 ms of a 20 s window): 100 + round(100 × (1 − 9 000/20 000)) = 100 + 55 = **155**.
In **Individual Turns** (A3's turn, wrong): **0**, side streak resets.

---

## 6. Interaction with match types
| Match type | Effect |
|---|---|
| Quick Duel, Ranked, Marathon | side score = sum of `side_points`; higher wins |
| Sudden Death | uses `side_correct` per round: exactly one side correct → that side wins (D-038) |
| All | side answer streaks / member streaks reset on a wrong/timeout per the rules above |

---

## 7. Invariants & tests (→ 0.3.11)
1. `0 ≤ side_points ≤ 260` (MC) per round for every mode and side size.
2. Sum of contributions in a round = `side_points`.
3. Shared Answer: exactly one side answer (or timeout) per side per round; fallback answers get speed bonus 0.
4. Individual Turns: only the turn holder's answer counts; rotation symmetric on both sides.
5. Fast Team Round: aggregation rule recorded in `round_results.detail`; ranked always `average`.
6. Same inputs → same outputs (pure functions, table-driven tests incl. §5 example).

---

## 8. Accepted proposals (D-039)
1. **Same scale for all formats:** a side's round points never exceed one player's maximum (260), so 1v1 and 4v4 scores are comparable.
2. **Shared Answer:** auto-locked (fallback) answers get **no speed bonus**; side streak; points shared equally among members (remainder to the locker).
3. **Individual Turns:** side streak continues across turn holders; points credited to the turn holder; if the turn holder is disconnected or has left, the turn passes automatically to the next active seat.
4. **Fast Team Round:** personal streaks; three aggregation rules — `average` (default), `fastest_correct`, `majority`; side counts as "correct" when at least half of the **active** members are correct (`average`).
5. **Who picks the rule:** Ranked and matchmade casual fixed to `average`; private rooms, challenges and tournaments let the host/organiser pick.
