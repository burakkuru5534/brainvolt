# Category mode rules — Blind Challenge, Category Master, Mixed Battle, Daily Challenge, Creator Quiz, Revenge Match

- **Roadmap task:** 0.3.7 (engine rules; full features in Phase 8)
- **Status:** Accepted (2026-10-15) — decisions D-040 … D-046
- **Owner module:** `game` (`domain/categorymode`: one selector + optional pre-round hook per mode; `domain/dailyrun` for the Daily Challenge solo engine); question sets from `content`, quizzes from `creator`, revenge offers from `lobby`
- **Builds on:** D-023 / D-031 (creator quizzes), D-024 (no TF in daily), D-027 (revenge by losing side, 24 h), D-029 (Istanbul days), D-031 (one counted daily attempt), D-032 (2 min resume), D-036 (10 s pick), D-038 (match types, selection defaults), D-039 (team modes)
- **Related:** match-types.md §2 (default selection that a category mode replaces), round-lifecycle.md (`pre_round`), schema-content (question sets, daily sets), schema-platform §6 (daily run tables), schema-ranking §4.4 (daily board), redis-keys §2a / §6.4, combinations 0.5

---

## 1. How a category mode plugs in

A head-to-head match has **one match type** (how it is played and ends) and **at most one category mode** (which questions are asked and how they are presented). No category mode = default mixed selection (match-types.md §2).

| Hook | What a category mode may change |
|---|---|
| `Select` | which questions, in what order, how many (overrides match-types §2) |
| `PreRound` | an optional `pre_round` phase (round-lifecycle §1), e.g. a category pick |
| `Present` | what is shown with the question (e.g. hide the category) |
| `Eligibility` | who may play / which origins and match types are allowed (checked at creation, matrix in 0.5) |
| `Awards` | whether rating / SP / leaderboard / quests / XP apply (on top of 0.3.10) |

Scoring (ruleset 1), team modes and end rules of the match type are **never** changed by a category mode, except where §6 / §7 say otherwise for question count.

**Daily Challenge is the exception:** it is **not a head-to-head match** but a **solo run** (§5) with its own engine entry point, tables and leaderboard (D-040). It reuses the round phases, timing model and per-answer scoring, but nothing is written to `game.matches` / `match_sides` / `match_participants`.

**Ranked has no category mode** (D-041): it always uses the global mixed pool so ratings stay comparable.

---

## 2. Blind Challenge (`blind_challenge`)
- **Select:** default mixed selection of the match type.
- **Present:** the category and sub-category of each question are **hidden** during `question` and shown at `reveal` (D-035 visibility rules). The match lobby shows "Kategori: ?" instead of categories.
- **Eligibility:** any match type except Ranked (D-041).
- **Awards:** normal for its match type.

## 3. Category Master (`category_master`)  (D-042)
- **Select:** every question from **one chosen top-level category** (in rooms/challenges optionally narrowed to one sub-category). Category caps from D-038 do not apply; "no consecutive sub-category" applies when the category has ≥ 2 sub-categories with enough questions; difficulty curve of the match type still applies.
- **Who chooses:** room host / challenger; in matchmaking the player picks the category when queueing, and **only players searching the exact same category are paired** (queue key includes the category id). Matchmaking is offered **only for top-level categories**.
- **Eligibility:** any match type except Ranked. Creation fails if the category has fewer eligible questions than needed in all participant locales.
- **Rating:** **unrated at launch** (casual). The rating schema already supports separate per-category buckets (`rating_bucket = 'category:<id>'`, schema-ranking §2), kept **apart from global Ranked rating**, so category ratings can be switched on later without schema changes.
- **Awards:** normal casual awards; points also feed the **category leaderboard** for that category (0.3.10).

## 4. Mixed Battle (`mixed_battle`)  (D-043)
- **Before the match:** **3 categories are pre-selected** and shown in the lobby: chosen by the host / challenger in rooms and challenges; chosen randomly by the server (3 different top-level categories with enough eligible questions) in matchmaking.
- **PreRound (every round):** the **picking side alternates** each round (first picker by coin toss shown in the lobby). The picking side chooses **one of the 3 match categories** for that round; pick time **10 s** (D-036); no pick → server picks randomly among the 3. The same category may be picked again, but **never more than 3 rounds in a row** (diversity cap, same as D-045).
- **Teams:** the picking side's **designated turn holder** (seat rotation per question, D-035/D-039) makes the pick. The **answer phase follows the configured team answer mode** (D-046): in Individual Turns the same turn holder answers; in Shared Answer and Fast Team Round the side answers per that mode. All three team modes are compatible with Mixed Battle.
- **Select:** one question from the picked category at the round's curve difficulty.
- **Eligibility:** any match type except Ranked; works with Sudden Death (each round still decided by right vs. wrong).
- **Awards:** normal casual awards; category leaderboard rows per picked category.

## 5. Daily Challenge — solo score race  (D-040)

### 5.1 Content & rules
| Item | Rule |
|---|---|
| Questions | The day's `content.daily_sets` entry (Europe/Istanbul day, D-029): **10 multiple-choice questions**, same 10 and same order for every player; verified in **both** TR and EN (D-024), so TR and EN players face the identical set. Option order shuffled per run. |
| Difficulty | Ranked curve 2-3-3-3-4-4-4-4-5-5 |
| Answer window | 15 s per question; timing model unchanged (show_at, grace, latency compensation — timing.md) |
| Format | **`solo`**: one player, no opponent, no winner; engine-level format used **only** by Daily Challenge |
| Per-answer scoring | ruleset 1 (D-037) — shown to the player for fun, **not** used for ranking |
| Official attempt | **exactly one per player per day** (D-031). Later runs the same day are **practice**: same questions, zero leaderboard impact |

### 5.2 Daily leaderboard ranking
1. **More correct answers** ranks higher.
2. Tie → **lower total answer time** ranks higher. Total time = sum over the 10 questions of `response_ms` (timing.md §5); an unanswered question (timeout) counts as the **full 15 000 ms** (P).
3. Still tied → **earlier completion time** ranks higher (P).

Boards: global, country, friends (friends computed at read time); restricted accounts are excluded from public boards (D-016). The daily board is **separate** from the general daily/weekly/monthly leaderboards and from SP (D-040): a Daily Challenge run gives no SP and no general leaderboard points.

### 5.3 What an official run gives
| | Official completed | Practice | Not consumed (§5.4) |
|---|:-:|:-:|:-:|
| Daily Challenge leaderboard entry | ✓ | ✗ | ✗ |
| XP (amount in 0.3.10) | ✓ | ✗ (D-046) | ✗ |
| Quest progress (`daily_challenge` objective) & play streak | ✓ | ✗ | ✗ |
| Shareable result card & "beat my score" link | ✓ | ✗ | ✗ |

### 5.4 Failure grace — when the official attempt is **not** consumed
| Situation | Result |
|---|---|
| **Server-side error** (our fault: run owner lost and run state unrecoverable, persistence failure, Redis loss) | Run marked `void_server_error`; **attempt not consumed**; the player may start a new official run (same questions — logged, accepted). |
| Player's connection drops **before the first question is shown** (`show_at` of Q1 not reached) | Run marked `void_not_started`; **attempt not consumed**. |
| Player's connection drops **after** Q1 was shown — *strict fallback validation* | The server **keeps the clock running** (questions are never paused). The player may **resume within 2 min** (D-032) at the current question. If not resumed, the run is **completed** with every remaining question as a timeout; **attempt consumed**. This prevents "disconnect when doing badly, retry tomorrow-style" abuse. |
| App closed / killed by the player | Same as a connection drop after Q1. |
| More than **one** `void_server_error` for the same player on the same day | Allowed; every void is logged; > 3 per day raises a moderation signal (`protocol_violation`, D-030) for review (P). |

### 5.5 Sharing & "beat my score"
- After an official run: share card (score, correct count, time, rank) and **"beat my score"** to friends: an in-app notification (`notification` category `challenge`, kind `daily.beat_my_score`) and a deep link `brainvolt://daily/<yyyy-mm-dd>?from=<username>`.
- The friend plays **their own official run**; afterwards both see the comparison. No head-to-head match is created. Sends count toward the direct-challenge rate limit (20/h, D-027).

### 5.6 Isolation (D-040 instruction)
- Persistence: `game.daily_runs` and `game.daily_run_answers` (schema-platform §6) — **never** `game.matches` / `match_sides` / `match_participants` / `answers`.
- Leaderboard: `ranking.daily_challenge_entries` + Redis `rank:daily:*` (schema-ranking §4.4, redis-keys §6.4) — **never** `ranking.points_ledger` / `season_standings`.
- Event: `game.DailyRunCompleted` (official completed runs only) → ranking (daily board), progression (XP, quest, streak), notification.

## 6. Creator Quiz (`creator_quiz`)  (D-044)
- **Select:** the quiz's question set **in the author's order**; the author defines questions, options, order and the quiz category; question count = quiz length (5–30, D-031). Options shuffled per match.
- **Answer window:** the author picks **10, 15 or 20 s** per quiz (default 15 s).
- **Match types:** fixed-length play only (Quick Duel rules with the quiz's length) — never Ranked, Sudden Death or Marathon.
- **Where it can be played:**

| Quiz status | Where | Who can join |
|---|---|---|
| `draft`, `rejected`, `unpublished` (unverified) | **private rooms only** | **friends of the room host** (D-023) |
| `published` (validated by moderation) | private rooms and direct challenges; listed in public discovery; no matchmaking at launch | anyone allowed by the room/challenge rules |

- **Awards:** never rated, never enters the Ranked pool, no SP, no leaderboard points (D-023, D-031). XP for players of a **published** quiz; the **author earns zero XP** on their own quiz (shown with an "author" label). Unverified quizzes give nothing.

## 7. Revenge Match (`revenge_match`)  (D-045)
- **Created from** a `revenge` challenge issued by the **losing side** against the preceding opponent(s) — offered **instantly on the results screen** and valid up to **24 h** (D-027).
- **Config:** copies the parent match's configuration (format, team answer mode, match type, answer window). If the parent was **Ranked**, the revenge uses Quick Duel rules with Ranked's length (10 questions, MC only) and is **unrated** — rating changes only through a fresh **matchmade** Ranked game (D-038).
- **Home category:** the challenger picks one category; **50 % of the questions** come from it, **rounded down** for odd counts (e.g. 7 → 3 home questions); the rest use default mixed selection.
- **Diversity cap:** **at most 3 consecutive questions from the same category** (home questions are interleaved accordingly).
- **Chain:** the loser of a revenge may issue another revenge (new `parent_match_id`), up to **3 revenge matches per chain** (D-046); results screen shows the series score ("Rövanş serisi: 2–1").
- **Awards:** never rated; casual SP/XP per 0.3.10; revenge wins count toward revenge badges/quests.

---

## 8. Summary matrix (detailed validation in 0.5)

| Category mode | Ranked | Quick Duel | Sudden Death | Marathon | Origins |
|---|:-:|:-:|:-:|:-:|---|
| none (default) | ✓ | ✓ | ✓ | ✓ | all (Ranked: matchmaking only) |
| Blind Challenge | ✗ | ✓ | ✓ | ✓ | all |
| Category Master | ✗ | ✓ | ✓ | ✓ | all (matchmaking: top-level categories, same-category pairing) |
| Mixed Battle | ✗ | ✓ | ✓ | ✓ | all |
| Creator Quiz | ✗ | ✓ (quiz length) | ✗ | ✗ | rooms; challenges if published |
| Revenge Match | ✗ | ✓ (or parent type) | ✓ | ✓ | revenge challenge only |
| **Daily Challenge** | — solo run, not a match type combination — | | | | daily entry point only |

---

## 9. Test coverage required (→ 0.3.11; implemented in Phase 8 with the features)

**Daily attempt limits**
- First completed run of the day is `official`; any later run the same day is `practice` and never reaches the leaderboard, XP, quests or streak.
- Two concurrent "start official run" requests → exactly one succeeds (unique index), the other gets `DAILY_ATTEMPT_IN_PROGRESS`.
- Day boundary at 00:00 Europe/Istanbul: a run started 23:59:50 belongs to the day it started; the next day's official attempt is available at 00:00.
- Restricted account completes an official run → stored, excluded from public boards, appears after upgrade (D-016).

**Leaderboard tie-breakers**
- More correct beats fewer correct regardless of time.
- Equal correct → lower total time wins; timeouts add 15 000 ms each.
- Equal correct and equal total time → earlier `completed_at` wins.
- Redis score encoding round-trips exactly (no precision loss) for 0–10 correct and 0–150 000 ms.

**Disconnect & error handling**
- Server error mid-run → `void_server_error`, attempt not consumed, new official run allowed.
- Drop before Q1 `show_at` → `void_not_started`, not consumed.
- Drop after Q1 shown, resume within 2 min → continues at the current question with the original deadlines (no extra time).
- Drop after Q1 shown, no resume → run completed, remaining questions timeouts, attempt consumed.
- > 3 server-error voids for one user in one day → moderation signal emitted.

**Other modes**
- Mixed Battle: picks alternate sides; only the 3 lobby categories are offered; 4th consecutive pick of the same category rejected; no pick → random among the 3.
- Category Master matchmaking pairs only identical category ids.
- Revenge: 7-question revenge has exactly 3 home questions; no 4 consecutive same-category questions; Ranked parent → unrated revenge.
- Creator Quiz: author's run gives 0 XP; unverified quiz room rejects non-friends of the host.

**Where these are tested (Phase 8.6):**

| Item | Test |
|---|---|
| Official / practice, attempt used, void frees the attempt | `internal/app/daily_e2e_test.go` `TestDailyChallenge` |
| Concurrent official starts → one wins | `internal/app/daily_rules_e2e_test.go` |
| Day boundary 00:00 Europe/Istanbul | `game/internal/service/daily_day_test.go` |
| Restricted run off the board until upgrade | `internal/app/daily_rules_e2e_test.go` |
| Tie-breakers (correct ↓, time ↑, completed_at ↑) | `internal/app/daily_rules_e2e_test.go`; timeouts count 15 000 ms in `TestDailyChallenge` |
| Resume keeps the original deadlines; no resume → timeouts | `TestDailyChallenge` ("the clock keeps running") |
| Mixed Battle picks (alternation, 3 categories, 4th consecutive, random) | `engine_test.go` `TestCM03MixedBattlePicks`, `TestLiveMixedBattlePick` |
| Category Master pairs identical categories | `matchmaking/internal/service/service_test.go` |
| Revenge home share and runs; Ranked parent → unrated | `content/internal/domain/selection_test.go` `TestSelectRevengeHome`, `config_test.go` (T01) |
| Creator Quiz awards and friends-only rooms | `internal/app/creator_e2e_test.go` |

Two items changed with the implementation (D-081, D-084): the daily board is ordered in Postgres, so there is no Redis score encoding to round-trip; and the daily run is settled lazily from Postgres with no server-held state, so a server error fails one request without voiding the run — there is no `void_server_error` and hence no daily void signal. Repeated server cancellations of **matches** raise the `repeated_server_voids` signal instead (≥ 3 in 7 days).
