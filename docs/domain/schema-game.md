# Schema — game (matches, sides, participants, rounds, answers, results)

- **Roadmap task:** 0.2.4
- **Status:** Accepted (2026-10-15)
- **Module / Postgres schema:** `game`
- **Decisions applied:** D-004 (casual unrated), D-007 / D-023 / D-024 (content eligibility), D-016a (temporary sides), D-018 (Couple Contest = format), D-019 (team answer modes)
- **Depends on:** `glossary.md`, `schema-identity.md`, `schema-content.md`. State machine & timing rules: 0.3.1–0.3.3. Scoring: 0.3.4. Redis live-state keys: 0.2.9.
- Conventions: as in `schema-identity.md`.

---

## 1. Where match data lives

| Phase of a match | Authoritative copy | Postgres copy |
|---|---|---|
| Created → waiting / ready | Redis live state (owner process) | `matches`, `match_sides`, `match_participants` rows inserted **at creation**, status kept in sync on every lifecycle transition |
| In progress | Redis live state (single writer, `backend-modules.md` §3.4) | each round persisted **at its reveal** (`match_rounds`, `answers`, `round_results`) in one transaction |
| Completed / cancelled | **Postgres** (final) | final scores/results written in the **same transaction** that enqueues `game.MatchCompleted` / `MatchCancelled` (River outbox) |

Consequences: a crash loses at most the current, not-yet-revealed round (recovery rules in 0.3.9); everything downstream (rating, XP, leaderboards, stats, anti-cheat) only ever sees committed, final data.

---

## 2. Tables

```sql
CREATE SCHEMA game;

-- ---------- Match ----------
CREATE TABLE game.matches (
  id                 uuid        PRIMARY KEY,
  -- configuration (immutable after creation; validated against the mode matrix, 0.5)
  match_type         text        NOT NULL CHECK (match_type IN ('quick_duel','ranked','sudden_death','marathon')),
  format             text        NOT NULL CHECK (format IN ('1v1','2v2','3v3','4v4','couple')),
  team_answer_mode   text        CHECK (team_answer_mode IN ('shared_answer','individual_turns','fast_team_round')),
  category_mode      text        CHECK (category_mode IN ('blind_challenge','category_master','mixed_battle',
                                                          'daily_challenge','creator_quiz','revenge_match')),
  is_rated           boolean     NOT NULL,
  rating_bucket      text,                                  -- which rating changes (0.3.10); NULL when unrated
  season_id          uuid,                                  -- ranking season at creation; NULL when unrated (plain id, no FK: ranking is downstream)
  question_count     smallint    CHECK (question_count BETWEEN 1 AND 100),  -- NULL for open-ended types (e.g. sudden death)
  answer_window_ms   int         NOT NULL CHECK (answer_window_ms BETWEEN 3000 AND 60000),
  locales            text[]      NOT NULL,                  -- distinct participant locales, e.g. {tr} or {tr,en}
  question_set_id    uuid        REFERENCES content.question_sets(id),       -- daily / creator / tournament sets
  config_extra       jsonb       NOT NULL DEFAULT '{}',     -- mode-specific params (aggregation rule, chosen category, …) — schema per mode in 0.3
  ruleset_version    int         NOT NULL,                  -- version of scoring/timing rules applied; history stays reproducible
  -- origin
  origin             text        NOT NULL CHECK (origin IN ('matchmaking','challenge','private_room','rematch',
                                                            'revenge','tournament','daily_challenge')),
  origin_ref         uuid,                                  -- room / challenge / ticket / tournament-match id (owned by other modules)
  parent_match_id    uuid        REFERENCES game.matches(id),               -- rematch & revenge chains
  -- lifecycle (state machine in 0.3.1)
  status             text        NOT NULL DEFAULT 'waiting'
                     CHECK (status IN ('waiting','ready','in_progress','completed','cancelled')),
  created_at         timestamptz NOT NULL,
  ready_at           timestamptz,
  started_at         timestamptz,
  ended_at           timestamptz,
  end_reason         text        CHECK (end_reason IN ('all_rounds_played','sudden_death_decided','forfeit',
                                                       'all_opponents_left','time_limit')),
  cancel_reason      text        CHECK (cancel_reason IN ('not_all_ready','participant_left_before_start',
                                                          'server_error','admin','integrity_violation')),
  -- outcome
  winner_side_no     smallint    CHECK (winner_side_no IN (1,2)),
  is_draw            boolean     NOT NULL DEFAULT false,
  -- integrity (schema-moderation §4): result voided by admin after cheating confirmed
  voided_at          timestamptz,
  void_reason        text,
  voided_by          uuid        REFERENCES identity.users(id),
  updated_at         timestamptz NOT NULL,

  CHECK ((format = '1v1') = (team_answer_mode IS NULL)),                       -- team modes only for >1 per side
  CHECK (is_rated = (match_type = 'ranked')),                                  -- D-004
  CHECK (NOT is_rated OR (rating_bucket IS NOT NULL AND season_id IS NOT NULL)),
  CHECK (status <> 'completed' OR (ended_at IS NOT NULL AND end_reason IS NOT NULL)),
  CHECK (status <> 'cancelled' OR cancel_reason IS NOT NULL),
  CHECK (NOT (is_draw AND winner_side_no IS NOT NULL))
);
CREATE INDEX matches_status_idx   ON game.matches (status) WHERE status IN ('waiting','ready','in_progress');
CREATE INDEX matches_ended_idx    ON game.matches (ended_at) WHERE status = 'completed';
CREATE INDEX matches_parent_idx   ON game.matches (parent_match_id) WHERE parent_match_id IS NOT NULL;

-- ---------- Sides (always exactly two per match) ----------
CREATE TABLE game.match_sides (
  match_id        uuid        NOT NULL REFERENCES game.matches(id) ON DELETE CASCADE,
  side_no         smallint    NOT NULL CHECK (side_no IN (1,2)),
  team_id         uuid,                                   -- persistent team; NULL = temporary side (D-016a). Plain id: team is a lower layer but FK kept out to allow team deletion
  couple_id       uuid,                                   -- set when format = 'couple' (D-018)
  score           int         NOT NULL DEFAULT 0,         -- server-computed total (0.3.4)
  correct_count   smallint    NOT NULL DEFAULT 0,
  result          text        CHECK (result IN ('win','loss','draw')),   -- set at completion
  forfeited_at    timestamptz,
  PRIMARY KEY (match_id, side_no),
  CHECK (NOT (team_id IS NOT NULL AND couple_id IS NOT NULL))
);
CREATE INDEX match_sides_team_idx   ON game.match_sides (team_id)   WHERE team_id IS NOT NULL;
CREATE INDEX match_sides_couple_idx ON game.match_sides (couple_id) WHERE couple_id IS NOT NULL;

-- ---------- Participants ----------
CREATE TABLE game.match_participants (
  match_id               uuid        NOT NULL REFERENCES game.matches(id) ON DELETE CASCADE,
  user_id                uuid        NOT NULL REFERENCES identity.users(id),
  side_no                smallint    NOT NULL,
  seat_no                smallint    NOT NULL CHECK (seat_no BETWEEN 1 AND 4),  -- order within side (Individual Turns rotation)
  locale                 text        NOT NULL CHECK (locale IN ('tr','en')),
  account_tier_at_start  text        NOT NULL CHECK (account_tier_at_start IN ('guest','unverified','full')), -- for leaderboard visibility rules
  status                 text        NOT NULL DEFAULT 'joined'
                         CHECK (status IN ('joined','ready','playing','disconnected','left','forfeited','finished')),
  ready_at               timestamptz,
  left_at                timestamptz,
  disconnect_count       smallint    NOT NULL DEFAULT 0,
  disconnected_ms_total  int         NOT NULL DEFAULT 0,
  score_contribution     int         NOT NULL DEFAULT 0,   -- points this user earned for the side
  correct_count          smallint    NOT NULL DEFAULT 0,
  answered_count         smallint    NOT NULL DEFAULT 0,
  joined_at              timestamptz NOT NULL,
  PRIMARY KEY (match_id, user_id),
  UNIQUE (match_id, side_no, seat_no),
  FOREIGN KEY (match_id, side_no) REFERENCES game.match_sides(match_id, side_no)
);
CREATE INDEX match_participants_user_idx ON game.match_participants (user_id, joined_at DESC);  -- match history

-- ---------- Rounds ----------
CREATE TABLE game.match_rounds (
  match_id           uuid        NOT NULL REFERENCES game.matches(id) ON DELETE CASCADE,
  round_no           smallint    NOT NULL CHECK (round_no >= 1),
  question_id        uuid        NOT NULL REFERENCES content.questions(id),
  question_revision  int         NOT NULL,              -- exact version shown (schema-content §4.3)
  option_order       smallint[]  NOT NULL,              -- shuffled canonical positions as shown, e.g. {3,1,4,2}
  answering_side_no  smallint    CHECK (answering_side_no IN (1,2)),     -- set only if a mode gives a round to one side
  answering_user_id  uuid        REFERENCES identity.users(id),         -- Individual Turns: whose turn it is
  dispatched_at      timestamptz NOT NULL,              -- server time question was sent
  deadline_at        timestamptz NOT NULL,              -- server-enforced close (0.3.3)
  revealed_at        timestamptz NOT NULL,              -- rows are written at reveal, so never NULL
  PRIMARY KEY (match_id, round_no)
);
CREATE INDEX match_rounds_revealed_idx ON game.match_rounds (revealed_at);   -- purge job (D-026)
CREATE INDEX match_rounds_question_idx ON game.match_rounds (question_id);

-- ---------- Answers (accepted submissions only) ----------
CREATE TABLE game.answers (
  match_id        uuid        NOT NULL,
  round_no        smallint    NOT NULL,
  user_id         uuid        NOT NULL REFERENCES identity.users(id),
  side_no         smallint    NOT NULL,
  option_id       uuid        REFERENCES content.options(id),   -- NULL = no answer before deadline (timeout row)
  received_at     timestamptz,                                  -- server receive time; NULL for timeout rows
  response_ms     int         CHECK (response_ms >= 0),         -- server-measured, latency-adjusted (0.3.3)
  is_correct      boolean     NOT NULL,
  points_awarded  int         NOT NULL DEFAULT 0,               -- this answer's own points before side aggregation
  is_side_answer  boolean     NOT NULL DEFAULT false,           -- Shared Answer: the submission that counted for the side
  created_at      timestamptz NOT NULL,                         -- row write time (reveal); drives 12-month purge / future partitioning (D-026)
  PRIMARY KEY (match_id, round_no, user_id),
  FOREIGN KEY (match_id, round_no) REFERENCES game.match_rounds(match_id, round_no) ON DELETE CASCADE,
  FOREIGN KEY (match_id, user_id)  REFERENCES game.match_participants(match_id, user_id)
);

-- ---------- Per-side round outcome (aggregation for team modes) ----------
CREATE TABLE game.round_results (
  match_id        uuid      NOT NULL,
  round_no        smallint  NOT NULL,
  side_no         smallint  NOT NULL,
  is_correct      boolean   NOT NULL,                 -- side counted as correct for this round
  points          int       NOT NULL,                 -- points added to side score this round (incl. bonuses)
  detail          jsonb     NOT NULL DEFAULT '{}',    -- e.g. Fast Team Round: rule used, contributing answers (0.3.6)
  PRIMARY KEY (match_id, round_no, side_no),
  FOREIGN KEY (match_id, round_no) REFERENCES game.match_rounds(match_id, round_no) ON DELETE CASCADE
);

-- ---------- Lifecycle audit ----------
CREATE TABLE game.match_transitions (
  match_id     uuid        NOT NULL REFERENCES game.matches(id) ON DELETE CASCADE,
  seq          int         NOT NULL,
  from_status  text,
  to_status    text        NOT NULL,
  reason       text,
  actor        text        NOT NULL CHECK (actor IN ('system','participant','admin')),
  actor_id     uuid,
  at           timestamptz NOT NULL,
  PRIMARY KEY (match_id, seq)
);
```

### 2.1 Rules enforced in the service layer
| Rule | Check |
|---|---|
| Side size matches format: 1v1 → 1, 2v2/couple → 2, 3v3 → 3, 4v4 → 4 participants per side | at creation |
| `couple` format: each side has `couple_id`, its 2 participants are that couple's members | at creation |
| A user is in at most **one** live match (`waiting`/`ready`/`in_progress`) | at creation (Redis lock, 0.2.9) |
| Ranked: every round's question is `multiple_choice`, verified + verified translations for all `locales` (D-007, D-024) | at question selection |
| `question_set_id` required for `daily_challenge`, `creator_quiz` and tournament matches | at creation |
| Answers only accepted while round open and before `deadline_at + grace` (0.3.3); first accepted answer per user per round is final | live, in owner |
| Individual Turns: only `answering_user_id` may answer; others' submissions are rejected | live |
| Shared Answer: exactly one `is_side_answer = true` per side per round | at reveal |
| Final `match_sides.score` = sum of its `round_results.points` | at completion (asserted) |

Rejected submissions (late, duplicate, not-your-turn, malformed) are **not** rows in `answers`; they are counted in Redis and sent to `moderation` as cheat signals when thresholds are exceeded (Phase 8).

---

## 3. Read models
| Need | Source |
|---|---|
| A user's match history | `match_participants (user_id, joined_at DESC)` → join `matches`, `match_sides` |
| Result screen & shareable result card | one query over the match's sides, participants, rounds and answers (contract in 0.4.2) |
| Revenge / rematch chains | `matches.parent_match_id` |
| Team / couple history | `match_sides.team_id` / `couple_id` |

What is **not** stored in `game`: rating changes, SP, XP, badges (owned by `ranking` / `progression`, written when they consume `game.MatchCompleted`).

---

## 4. Events produced
| Event | Payload essentials |
|---|---|
| `game.MatchStarted` | match id, config summary, participant user ids |
| `game.MatchCompleted` | match id, config (type, format, modes, rated, rating_bucket, season_id, category info), per side: score, result, team/couple id; per participant: user id, side, tier at start, correct/answered counts, contribution, disconnect stats; duration; question ids + locales (for stats) |
| `game.MatchCancelled` | match id, reason, participant ids, origin |
| `game.MatchVoided` | match id, reason, admin id, affected participant & side ids (schema-moderation §4) |

Payload is final and immutable; consumers never query `game` tables to reconstruct it.

---

## 5. Volume & retention (detailed in 0.2.10)
- `answers` is the largest table (≈ rounds × participants). Partitioning by month on match creation time is planned for when volume requires it; PKs above allow adding a time column without changing app logic.
- **D-026:** `match_rounds`, `answers`, `round_results` kept **12 months**, then purged/archived by a worker job. `matches`, `match_sides`, `match_participants` (results, scores, history) and rating changes (in `ranking`) kept **forever**. Aggregated `content.question_stats` are unaffected by the purge.

## 6. Open points
1. **Draw handling & tie-breakers** (e.g. equal score → faster total correct response time?) — 0.3.9; `is_draw` exists for when no tie-breaker applies.
2. **Rating bucket list** (e.g. `solo`, `team`, `couple`) — 0.3.10.
