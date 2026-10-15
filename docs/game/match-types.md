# Match type rules — Quick Duel, Ranked, Sudden Death, Marathon

- **Roadmap task:** 0.3.5
- **Status:** Accepted (2026-10-15)
- **Owner module:** `game` (`domain/matchtype`: one rule object per type)
- **Related:** scoring.md (ruleset 1, D-037), round-lifecycle.md (`end_check`), timing.md (windows), content eligibility (schema-content §3, D-007/D-023/D-024), team modes 0.3.6, category modes 0.3.7 (may **override** question selection), ties & forfeits 0.3.9, combinations 0.5
- Values marked **(P)** were proposed and accepted (D-038, §8).

---

## 1. Summary

| | **Quick Duel** | **Ranked** | **Sudden Death** | **Marathon** |
|---|---|---|---|---|
| Purpose | short, relaxed match | competitive, changes rating & SP | tense knockout | long endurance match |
| Rated (D-004) | ✗ | ✓ | ✗ | ✗ |
| Questions | **7** (P) | **10** (P) | open-ended, **max 30** (P) | **25** (P) |
| Answer window | **15 s** | **15 s** | **10 s** (P) | **12 s** (P) |
| Question types | MC + TF (TF ≤ 30 %) | **MC only** (D-024) | MC + TF (TF ≤ 30 %) | MC + TF (TF ≤ 30 %) |
| Content | verified (D-023) | verified + verified translations (D-007) | verified | verified |
| Difficulty curve | §3.1 | §3.2 | §3.3 | §3.4 |
| Ends with | `all_rounds_played` | `all_rounds_played` | `sudden_death_decided` (or cap) | `all_rounds_played` |
| Typical length (1v1) | ~2 min | ~3 min | 1–5 min | ~7 min |
| Allowed origins | all | **matchmaking only** (P) | all | all |

Common to every type: score per answer = scoring.md ruleset 1; team formats add the team answer mode (0.3.6); category modes (0.3.7) may replace the default question selection; forfeits and ties follow 0.3.9.

**Team answer-window extension (P):** in **Shared Answer** rounds the window is **+5 s** (teammates need time to agree). Other team modes use the base window.

---

## 2. Default question selection (all types, unless a category mode overrides)

1. Pool = eligible questions for the type (§1 row "Content"), in **every** locale present in the match (schema-content §3), not seen by any participant in the last 30 days (D-032).
2. **Category mix:** questions spread across categories; no two consecutive questions from the same **sub-category**; at most **2** per category in Quick Duel, **3** in Ranked, **5** in Marathon (P).
3. **True/false share** ≤ 30 % where allowed, never two in a row (P).
4. Difficulty per round follows the type's curve (§3); if the exact difficulty is unavailable, the nearest available difficulty is used (prefer lower, then higher).
5. Fixed-length types select all questions at match creation (T1 guard: enough questions must exist); Sudden Death selects in batches of 10.

---

## 3. Per-type rules

### 3.1 Quick Duel (`quick_duel`)
- 7 questions, 15 s, difficulty curve **2, 2, 3, 3, 3, 4, 4**.
- Winner = higher total score; ties → 0.3.9.
- Casual: no rating; SP/leaderboard/XP effects per 0.3.10.

### 3.2 Ranked (`ranked`)
- 10 questions, 15 s, multiple choice only, verified question **and** verified translation for every participant locale.
- Difficulty curve **2, 3, 3, 3, 4, 4, 4, 4, 5, 5** (P) — rising, same for everyone.
- Rated: Glicko-2 update per rating bucket (`solo` 1v1, `team` 2v2–4v4, `couple`), SP & leaderboard points — formulas in 0.3.10.
- **Origins: matchmaking only** (P). Direct challenges, private rooms, rematches and revenge matches between chosen opponents cannot be ranked — this blocks rating farming between friends (win trading, D-030). Tournaments are unrated by default (D-031).
- Sanctions: `ranked_ban` blocks (D-030). Guests and unverified accounts may play (D-016).

### 3.3 Sudden Death (`sudden_death`)
One question per round, both sides answer; after each round:

| Round outcome | Result |
|---|---|
| One side correct, the other wrong / timeout | **The correct side wins** → `sudden_death_decided` |
| Both correct | continue |
| Both wrong / timeout | continue |

- Answer window **10 s**; difficulty **starts at 2 and rises by 1 every 3 rounds**, up to 5 (rounds 1–3: 2, 4–6: 3, 7–9: 4, 10+: 5).
- Scores are still computed (ruleset 1) and shown, but **do not decide the winner** — only the elimination rule does — except at the cap.
- **Safety cap 30 rounds** (P): if still undecided, the higher total score wins; then 0.3.9 tie rules. `end_reason = all_rounds_played`.
- Team formats: a side is "correct" in a round according to its team answer mode aggregation (0.3.6).

### 3.4 Marathon (`marathon`)
- 25 questions, **12 s** window (faster pace), difficulty in five blocks of 5: **2, 3, 3, 4, 5** per block average (block 1 all 2s, block 2 all 3s, block 3 all 3s, block 4 all 4s, block 5 all 5s) (P).
- **Half-time break of 15 s** after question 12 (P): scores shown, team members can chat (persistent teams/couples, D-031) or use quick chat.
- Winner = higher total score; ties → 0.3.9.
- More tolerant disconnect handling (longer reconnect window) — value in 0.3.9.
- `time_limit` end reason is reserved as a technical safety stop (total expected duration × 1.5), not a gameplay rule.

---

## 4. End-check contract

```go
// called by the round state machine after every reveal (round-lifecycle §1 end_check)
type EndDecision struct { Finished bool; Reason EndReason }
func (r QuickDuelRules)   EndCheck(s MatchState) EndDecision // round_no == 7
func (r RankedRules)      EndCheck(s MatchState) EndDecision // round_no == 10
func (r SuddenDeathRules) EndCheck(s MatchState) EndDecision // exactly one side correct in last round, or round_no == 30
func (r MarathonRules)    EndCheck(s MatchState) EndDecision // round_no == 25
```
Forfeit and "all opponents left" endings are checked by the lifecycle (0.3.9), before `EndCheck`.
Every number above lives in the `Ruleset` (ruleset 1) so tests pin them.

---

## 5. Invariants & tests (→ 0.3.11)
1. Ranked never contains a true/false or unverified question, or a translation that is not verified, for any participant locale.
2. Ranked match never created with an origin other than `matchmaking`.
3. Fixed-length types end exactly after their last question; Sudden Death ends on the first round with exactly one correct side, or at round 30.
4. No two consecutive questions share a sub-category; category caps respected; TF ≤ 30 % and never consecutive.
5. Difficulty per round equals the curve unless unavailable (fallback logged).
6. Shared Answer rounds have window + 5 s.

---

## 6. Open points for later tasks
- How category modes (0.3.7) replace §2 selection for each type, and which type × mode pairs are allowed — 0.5.
- Tie-breakers (equal scores) and forfeit details — 0.3.9.
- SP/XP amounts per type — 0.3.10.

---

## 7. Notes on the working definitions (glossary §4)
- Sudden Death's working definition mentioned "wrong (or slower)". Proposed rule uses **correct vs. wrong only**; speed matters only at the 30-round cap. This keeps the knockout readable ("one mistake and you're out").

## 8. Accepted proposals (D-038)
1. **Quick Duel:** 7 questions, 15 s, curve 2-2-3-3-3-4-4.
2. **Ranked:** 10 questions, 15 s, MC only, curve 2-3-3-3-4-4-4-4-5-5; **matchmaking only** (no ranked via challenges/rooms/rematch/revenge).
3. **Sudden Death:** 10 s window; correct-vs-wrong elimination (speed does not eliminate); difficulty rises every 3 rounds; cap 30 rounds then highest score.
4. **Marathon:** 25 questions, 12 s window, difficulty blocks 2-3-3-4-5, 15 s half-time break after question 12.
5. **Selection:** category caps 2 / 3 / 5 (Quick Duel / Ranked / Marathon), no consecutive same sub-category, true/false ≤ 30 % and never consecutive.
6. **Shared Answer** rounds get **+5 s** answer time.
