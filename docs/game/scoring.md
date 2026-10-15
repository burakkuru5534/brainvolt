# Base scoring formula

- **Roadmap task:** 0.3.4
- **Status:** Draft for review (2026-10-15)
- **Owner module:** `game` (`game/internal/domain/scoring` — pure function, no I/O)
- **Related:** timing.md (`response_ms`), round-lifecycle.md (evaluate step), match types 0.3.5, team aggregation 0.3.6, tie-breakers 0.3.9, SP/XP 0.3.10, content calibrated difficulty (schema-content §5)
- **Ruleset version:** this document defines **ruleset 1** (`game.matches.ruleset_version = 1`). Any later change to numbers or formulas creates ruleset 2; old matches stay reproducible.
- Values marked **(P)** are proposals (§8).

---

## 1. What "score" is
- **Score** = points inside one match, computed only by the server, used to decide the winner (glossary §3).
- It is **not** rating, SP or XP. Those are derived from the match *result* (and partly from performance) in 0.3.10.
- All arithmetic in integers; any rounding is **half-up** to the nearest whole point.

---

## 2. Points for one answer

```
if answer is not correct (wrong or timeout):
    points = 0                                   — no negative points (P)
else:
    points = base + speed_bonus + streak_bonus
```

### 2.1 Base points
| Question type | Base |
|---|---|
| `multiple_choice` | `round(100 × difficulty_multiplier)` |
| `true_false` (casual only, D-024) | `round(50 × difficulty_multiplier)` — half, because a guess is right 50 % of the time (P) |

| Difficulty used | 1 | 2 | 3 | 4 | 5 |
|---|---|---|---|---|---|
| `difficulty_multiplier` (P) | 0.8 | 0.9 | 1.0 | 1.1 | 1.2 |

**Difficulty used** = the question's `difficulty_calibrated` rounded to the nearest integer if it exists, otherwise its author `difficulty` (schema-content). It is fixed at question selection and stored per round (`game.match_rounds.difficulty_used`, added to schema-game in this task), so later recalibration never changes past scores. Both sides always get the same question, so difficulty adds excitement but never favours one side.

### 2.2 Speed bonus (correct answers only)
```
speed_max   = 100 for multiple_choice, 50 for true_false (P)
fraction    = clamp(1 − response_ms / answer_window_ms, 0, 1)
speed_bonus = round(speed_max × fraction)
```
Linear: answering at the very first moment gives the full bonus, answering at the deadline gives 0. `response_ms` is the latency-compensated value from timing.md §5, so the network doesn't decide the bonus.

### 2.3 Answer-streak bonus (correct answers only)
`answer_streak` = consecutive correct answers **including this one** (wrong or timeout resets to 0).

| Streak | Bonus (P) |
|---|---|
| 1–2 | 0 |
| 3–4 | +20 |
| 5 or more | +40 (cap) |

Whose streak counts in team modes (individual or side) is fixed in 0.3.6.

### 2.4 Range
| | Min (correct) | Max |
|---|---|---|
| multiple_choice | 80 (difficulty 1, last moment, no streak) | 120 + 100 + 40 = **260** |
| true_false | 40 | 60 + 50 + 40 = **150** |

---

## 3. Worked examples (answer window 15 000 ms)

| Case | Base | Speed | Streak | Total |
|---|---|---|---|---|
| MC, difficulty 3, `response_ms` 3 000, streak 1 | 100 | round(100 × 0.8) = 80 | 0 | **180** |
| MC, difficulty 5, `response_ms` 7 500, streak 4 | 120 | 50 | 20 | **190** |
| MC, difficulty 2, `response_ms` 14 900, streak 6 | 90 | round(100 × 0.00667) = 1 | 40 | **131** |
| MC, wrong answer | — | — | — (streak → 0) | **0** |
| TF, difficulty 3, `response_ms` 1 500, streak 3 | 50 | 45 | 20 | **115** |
| Timeout | — | — | — (streak → 0) | **0** |

---

## 4. Match score
- **1v1:** a participant's match score = sum of their answer points; side score = participant score.
- **Team modes:** side round points come from the aggregation rule of the team answer mode (0.3.6), built on the per-answer points above. Side score = sum of `round_results.points` (invariant, schema-game §2.1).
- **Match-type modifiers** (e.g. Sudden Death elimination, Marathon pacing) are defined in 0.3.5 and may only **add rules on top**; they never change this per-answer formula within ruleset 1.

---

## 5. What players see
At `reveal`, each participant's points are sent **with their breakdown** (`base`, `speed_bonus`, `streak_bonus`), so the app can animate "+100 / +80 hız / +20 seri" and players understand why they won or lost.

---

## 6. Implementation contract
```go
// pure, deterministic; table-driven tests cover every row of §3 and the bounds in §2.4
func ScoreAnswer(in AnswerInput, rules Ruleset) AnswerPoints
type AnswerInput  struct { Correct bool; QuestionType QType; Difficulty int; ResponseMs, WindowMs int; StreakBefore int }
type AnswerPoints struct { Base, Speed, Streak, Total int; StreakAfter int }
```
`Ruleset` holds every number in this document, keyed by `ruleset_version`.

---

## 7. Invariants & tests (→ 0.3.11)
1. Wrong or timeout → 0 points and streak 0.
2. 0 ≤ speed bonus ≤ speed_max; monotonically non-increasing in `response_ms`.
3. Total ≤ 260 (MC) / 150 (TF).
4. Same inputs → same output (no randomness, no clock).
5. Changing any number requires a new ruleset version; tests pin ruleset 1 values.

---

## 8. Proposals to confirm
1. **No negative points** for wrong answers (Sudden Death has its own elimination rule in 0.3.5).
2. **Base 100 × difficulty multiplier** 0.8 / 0.9 / 1.0 / 1.1 / 1.2 for difficulty 1–5.
3. **Speed bonus up to 100**, linear from show time to deadline.
4. **Streak bonus** +20 at 3–4 correct in a row, +40 from 5 (cap).
5. **True/false questions** (casual only) worth half: base 50 × multiplier, speed bonus up to 50.
