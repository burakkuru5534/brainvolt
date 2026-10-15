# Category mode rules — Blind Challenge, Category Master, Mixed Battle, Daily Challenge, Creator Quiz, Revenge Match

- **Roadmap task:** 0.3.7 (engine rules; full features in Phase 8)
- **Status:** Draft for review (2026-10-15) — contains **one choice for the owner** (§5, Daily Challenge format)
- **Owner module:** `game` (`domain/categorymode`: one selector + optional pre-round hook per mode); question sets from `content`, quizzes from `creator`, revenge offers from `lobby`
- **Builds on:** D-023 / D-031 (creator quizzes), D-024 (no TF in daily), D-027 (revenge by losing side, 24 h), D-029 (Istanbul days), D-031 (daily first attempt counts), D-036 (10 s pick), D-038 (match types, selection defaults), D-039 (team modes)
- **Related:** match-types.md §2 (default selection that a category mode replaces), round-lifecycle.md (`pre_round`), schema-content (question sets, daily sets), schema-platform (creator quizzes, daily attempts), combinations 0.5
- Values marked **(P)** are proposals (§9).

---

## 1. How a category mode plugs in

A match has **one match type** (how it is played and ends) and **at most one category mode** (which questions are asked and how they are presented). No category mode = default mixed selection (match-types.md §2).

| Hook | What a category mode may change |
|---|---|
| `Select` | which questions, in what order, how many (overrides match-types §2) |
| `PreRound` | an optional `pre_round` phase (round-lifecycle §1), e.g. a category pick |
| `Present` | what is shown with the question (e.g. hide the category) |
| `Eligibility` | who may play / which origins and match types are allowed (checked at creation, matrix in 0.5) |
| `Awards` | whether rating / SP / leaderboard / quests / XP apply (on top of 0.3.10) |

Scoring (ruleset 1), team modes and end rules of the match type are **never** changed by a category mode, except where §5 / §6 say otherwise for question count.

---

## 2. Blind Challenge (`blind_challenge`)
- **Select:** default mixed selection of the match type.
- **Present:** the category and sub-category of each question are **hidden** during `question` and shown at `reveal` (D-035 visibility rules). The match lobby shows "Kategori: ?" instead of categories.
- **Eligibility:** any match type except Ranked (P) — Ranked keeps one standard presentation so ratings stay comparable.
- **Awards:** normal for its match type.

## 3. Category Master (`category_master`)
- **Select:** every question from **one chosen category** (or one sub-category, if chosen). Category caps from D-038 do not apply; "no consecutive sub-category" applies when the category has ≥ 2 sub-categories with enough questions; difficulty curve of the match type still applies.
- **Who chooses:** room host / challenger; in matchmaking the player picks the category when queueing (queue key includes the category) — **matchmaking offered only for categories with enough eligible questions and players** (P: the top-level categories).
- **Eligibility:** any match type except Ranked (P). Creation fails if the category has fewer eligible questions than needed in all participant locales.
- **Awards:** normal for its match type; points also feed the **category leaderboard** for that category (0.3.10).

## 4. Mixed Battle (`mixed_battle`)
- **PreRound:** before every round, **one side picks the category** for that round from **3 random categories** offered by the server (never the previous round's category). Pick time **10 s** (D-036); no pick → server picks randomly.
- **Who picks:** sides alternate; the first picker is chosen randomly (coin toss shown in the lobby). Inside a team: Individual Turns → the turn holder picks; Shared Answer / Fast Team Round → **any member**, first pick counts (P).
- **Select:** one question from the picked category at the round's curve difficulty.
- **Eligibility:** any match type except Ranked (P); works with Sudden Death too (each round is still decided by right vs. wrong).
- **Awards:** normal; category leaderboard rows per picked category.

## 5. Daily Challenge (`daily_challenge`)
- **Question set:** the day's `content.daily_sets` entry (Europe/Istanbul day, D-029): **10 multiple-choice questions** (P), verified in **both** TR and EN (D-024, schema-content §3), difficulty curve as Ranked (2-3-3-3-4-4-4-4-5-5). Answer window 15 s. Same set, same order for everyone; option order still shuffled per player.
- **Counting:** only the **first completed attempt per day** counts for leaderboards, quests and play streak; later attempts are practice (D-031).
- **Awards:** never rated; daily leaderboard points per 0.3.10.

### 5.1 Owner choice — how is the Daily Challenge played?

| Option | How it works | Pros | Cons / changes needed |
|---|---|---|---|
| **A. Solo score race** (recommended) | Each player plays the 10 questions **alone**, any time during the day; score goes to the **Daily Challenge leaderboard** (global / country / friends). After playing, a player can send "beat my score" to friends (async). | Everyone faces identical conditions (no opponent luck); works at any hour; no waiting for matchmaking; perfect for a daily habit. | Needs a new player format **`solo`** (one side, no winner) used **only** by Daily Challenge: schema-game sides check becomes "2 sides, or 1 side when `format = solo`"; result = score + rank instead of win/loss. |
| B. Matchmade 1v1 on the daily set | Players queue and play a normal 1v1 on today's set. | No schema change. | Result depends on the opponent; late-day players see fewer opponents; "first attempt" may be wasted by an opponent's disconnect. |

The rest of this document assumes **Option A** unless the owner chooses B.

## 6. Creator Quiz (`creator_quiz`)
- **Select:** the quiz's question set (`creator.quizzes.question_set_id`) **in the author's order**; question count = quiz length (5–30, D-031). Options shuffled per match as always. Difficulty curve does not apply (author's questions).
- **Answer window:** the author picks **10, 15 or 20 s** per quiz (P, default 15 s).
- **Match types:** only fixed-length play (Quick Duel rules with the quiz's length) (P) — not Ranked, not Sudden Death, not Marathon.
- **Where it can be played:**

| Quiz status | Where | Who can join |
|---|---|---|
| `draft`, `rejected`, `unpublished` (unverified) | **private rooms only** | **friends of the room host** (D-023; schema-social §3.4) |
| `published` (verified) | private rooms and direct challenges; no matchmaking at launch (P) | anyone allowed by the room/challenge rules |

- **Awards:** never rated, no SP, no leaderboard points (D-023, D-031). XP: yes for players of a **published** quiz except the **author**, who knows the answers (P); unverified quizzes give nothing.
- **Author playing their own quiz:** allowed (e.g. hosting friends), but the author's score is shown with an "author" label and gives no XP (P).

## 7. Revenge Match (`revenge_match`)
- **Created from** a `revenge` challenge issued by the **losing side** within **24 h** of the parent match (D-027), to the same opponents (same team / couple, or the same users for temporary sides who are still available).
- **Config:** same format and team answer mode as the parent. Match type = the parent's type, **except Ranked → Quick Duel rules with Ranked's 10-question length and MC-only** (Ranked is matchmaking-only, D-038).
- **Special rule — "home advantage" (P):** the challenger (former loser) picks **one category** before sending; **half of the questions** (rounded down) come from that category, the rest are default mixed selection.
- **Chain:** the loser of a revenge match may issue a revenge again (new `parent_match_id`), up to **3 revenge matches** in one chain (P), to stop endless loops.
- **Awards:** never rated (D-038); casual SP/XP per 0.3.10; winning a revenge counts toward "revenge" badges/quests (progression).
- Results screen shows the **series score** of the chain (e.g. "Rövanş serisi: 2–1").

---

## 8. Summary matrix (detailed validation in 0.5)

| Category mode | Ranked | Quick Duel | Sudden Death | Marathon | Origins |
|---|:-:|:-:|:-:|:-:|---|
| none (default) | ✓ | ✓ | ✓ | ✓ | all (Ranked: matchmaking) |
| Blind Challenge | ✗ | ✓ | ✓ | ✓ | all |
| Category Master | ✗ | ✓ | ✓ | ✓ | all |
| Mixed Battle | ✗ | ✓ | ✓ | ✓ | all |
| Daily Challenge | ✗ | (own rules, §5) | ✗ | ✗ | daily entry point only |
| Creator Quiz | ✗ | ✓ (quiz length) | ✗ | ✗ | rooms; challenges if published |
| Revenge Match | ✗ | ✓ (or parent type) | ✓ | ✓ | revenge challenge only |

## 9. Proposals to confirm
1. **Ranked has no category mode** (only the default mixed selection), so ratings stay comparable.
2. **Category Master:** whole match from one chosen category (or sub-category); matchmaking queues only for top-level categories.
3. **Mixed Battle:** sides alternate picking one of 3 offered categories each round (10 s, random if no pick); first picker by coin toss; in teams the turn holder or the first member to pick decides.
4. **Daily Challenge:** 10 multiple-choice questions, ranked-style difficulty curve, 15 s — and **choose Option A (solo score race, recommended) or B (matchmade 1v1)**.
5. **Creator Quiz:** author's question order; author picks 10/15/20 s; fixed-length play only; published quizzes in rooms and challenges (no matchmaking at launch); players of published quizzes get XP, the author doesn't.
6. **Revenge Match:** copies the parent's setup (Ranked parent → unranked 10-question MC match); the challenger picks a "home" category for half the questions; max 3 revenges per chain; series score shown.
