# Schema — ranking & progression (rating, seasons, SP, leagues, leaderboards, XP, streaks, badges, quests)

- **Roadmap task:** 0.2.6
- **Status:** Accepted (2026-10-15)
- **Modules / Postgres schemas:** `ranking`, `progression`
- **Decisions applied:** D-003 (Glicko-2), D-004 (casual unrated), D-016 / D-017 (restricted accounts off public boards), D-023 (unverified creator quizzes earn nothing), D-026 (rating changes kept forever), D-028 (temporary sides off team boards)
- **Formulas are NOT fixed here** (Glicko-2 parameters, SP per result, league thresholds, XP curve, reset policy): they belong to 0.3.10. This document fixes *what is stored and where*.
- Conventions: as in `schema-identity.md`.

---

## 1. The four numbers (glossary §7)

| Number | Owner | Changed by | Reset | Visible as |
|---|---|---|---|---|
| **Rating** (Glicko-2 μ, RD, σ) | ranking | rated (ranked) matches only | soft reset at season end (RD widened) | matchmaking; shown to player as league/rank, raw value optional |
| **Season Points (SP)** | ranking | matches per 0.3.10 rules (ranked for sure; casual effect decided there) | every season | season leaderboards, leagues |
| **Leaderboard points** | ranking | same events as SP, bucketed by period | per period (daily/weekly/monthly); all-time never | period leaderboards |
| **XP** | progression | every completed match, quests, badges | never | level |

Who earns what is computed **only** from `game.MatchCompleted` (and progression's own quest/badge events). Matches that are unrated private-room unverified Creator Quizzes earn **no** SP, leaderboard points or quest progress (D-023); whether they earn XP is fixed in 0.3.10.

---

## 2. Rated entities

Ratings, SP and leaderboard entries belong to an **entity**:

| `entity_kind` | Entity | Used for |
|---|---|---|
| `user` | a user | Individual boards; personal rating per bucket |
| `team` | a persistent team | Team boards; team-vs-team matchmaking |
| `couple` | a couple profile | Couples boards; Couple Contest matchmaking |

Temporary sides (`match_sides.team_id IS NULL`) are **not** entities: only their members' `user` records change (D-028).

**Rating buckets** keep skill in different formats apart. Confirmed list (D-029): `solo` (1v1), `team` (2v2/3v3/4v4 — each user's team-play rating, and each persistent team's own rating), `couple` (Couple Contest). Stored as text so the list can change without a schema change.

---

## 3. `ranking` schema

```sql
CREATE SCHEMA ranking;

-- ---------- Glicko-2 ratings ----------
CREATE TABLE ranking.ratings (
  entity_kind     text          NOT NULL CHECK (entity_kind IN ('user','team','couple')),
  entity_id       uuid          NOT NULL,
  bucket          text          NOT NULL,
  rating          numeric(7,2)  NOT NULL,          -- Glicko scale (μ shown on the 1500 scale)
  rd              numeric(6,2)  NOT NULL,          -- rating deviation
  volatility      numeric(9,7)  NOT NULL,          -- σ
  rated_matches   int           NOT NULL DEFAULT 0,
  last_rated_at   timestamptz,                     -- RD grows with inactivity from here (0.3.10)
  updated_at      timestamptz   NOT NULL,
  PRIMARY KEY (entity_kind, entity_id, bucket)
);
CREATE INDEX ratings_bucket_rating_idx ON ranking.ratings (bucket, entity_kind, rating);

-- Every change, forever (D-026). Unique key = idempotent consumption of MatchCompleted.
CREATE TABLE ranking.rating_changes (
  match_id        uuid          NOT NULL,          -- plain id (game is upstream; FK allowed but kept loose for purge independence)
  entity_kind     text          NOT NULL,
  entity_id       uuid          NOT NULL,
  bucket          text          NOT NULL,
  season_id       uuid          NOT NULL,
  rating_before   numeric(7,2)  NOT NULL,
  rating_after    numeric(7,2)  NOT NULL,
  rd_before       numeric(6,2)  NOT NULL,
  rd_after        numeric(6,2)  NOT NULL,
  vol_before      numeric(9,7)  NOT NULL,
  vol_after       numeric(9,7)  NOT NULL,
  created_at      timestamptz   NOT NULL,
  PRIMARY KEY (match_id, entity_kind, entity_id, bucket)
);
CREATE INDEX rating_changes_entity_idx ON ranking.rating_changes (entity_kind, entity_id, created_at DESC);

-- ---------- Seasons ----------
CREATE TABLE ranking.seasons (
  id            uuid        PRIMARY KEY,
  number        int         NOT NULL UNIQUE CHECK (number >= 1),
  starts_at     timestamptz NOT NULL,
  ends_at       timestamptz NOT NULL,
  status        text        NOT NULL CHECK (status IN ('scheduled','active','closing','closed')),
  reset_policy  jsonb       NOT NULL,              -- parameters of the soft reset applied at close (0.3.10)
  closed_at     timestamptz,
  CHECK (ends_at > starts_at)
);
CREATE UNIQUE INDEX seasons_one_active_uq ON ranking.seasons ((true)) WHERE status = 'active';

CREATE TABLE ranking.season_translations (
  season_id  uuid NOT NULL REFERENCES ranking.seasons(id) ON DELETE CASCADE,
  locale     text NOT NULL CHECK (locale IN ('tr','en')),
  name       text NOT NULL,
  PRIMARY KEY (season_id, locale)
);

-- ---------- Leagues (config) ----------
CREATE TABLE ranking.leagues (
  code        text      PRIMARY KEY CHECK (code IN ('bronze','silver','gold','platinum','diamond')),
  sort_order  smallint  NOT NULL UNIQUE,
  divisions   smallint  NOT NULL DEFAULT 1 CHECK (divisions BETWEEN 1 AND 5),   -- e.g. Gold III–I (0.3.10)
  rules       jsonb     NOT NULL                    -- thresholds / promotion rules (0.3.10)
);
CREATE TABLE ranking.league_translations (
  code    text NOT NULL REFERENCES ranking.leagues(code),
  locale  text NOT NULL CHECK (locale IN ('tr','en')),
  name    text NOT NULL,                            -- Bronz, Gümüş, Altın, Platin, Elmas / Bronze … Diamond
  PRIMARY KEY (code, locale)
);

-- ---------- Season standings (SP, league, record) ----------
CREATE TABLE ranking.season_standings (
  season_id       uuid        NOT NULL REFERENCES ranking.seasons(id),
  entity_kind     text        NOT NULL CHECK (entity_kind IN ('user','team','couple')),
  entity_id       uuid        NOT NULL,
  season_points   int         NOT NULL DEFAULT 0 CHECK (season_points >= 0),
  league_code     text        NOT NULL REFERENCES ranking.leagues(code),
  division        smallint    NOT NULL DEFAULT 1,
  peak_league     text        NOT NULL REFERENCES ranking.leagues(code),
  peak_division   smallint    NOT NULL DEFAULT 1,
  matches         int         NOT NULL DEFAULT 0,
  wins            int         NOT NULL DEFAULT 0,
  losses          int         NOT NULL DEFAULT 0,
  draws           int         NOT NULL DEFAULT 0,
  best_win_streak int         NOT NULL DEFAULT 0,
  current_win_streak int      NOT NULL DEFAULT 0,
  final_rank_global  int,                           -- filled when the season closes
  final_rank_country int,
  updated_at      timestamptz NOT NULL,
  PRIMARY KEY (season_id, entity_kind, entity_id)
);
CREATE INDEX season_standings_sp_idx ON ranking.season_standings (season_id, entity_kind, season_points DESC);

-- ---------- Points ledger: source for every leaderboard ----------
CREATE TABLE ranking.points_ledger (
  id            uuid        PRIMARY KEY,
  match_id      uuid        NOT NULL,
  entity_kind   text        NOT NULL CHECK (entity_kind IN ('user','team','couple')),
  entity_id     uuid        NOT NULL,
  season_id     uuid        NOT NULL,
  category_id   uuid,                               -- NULL = overall row; set = category-board row (§4.3)
  points        int         NOT NULL,               -- SP / leaderboard points delta (may be negative for SP rules in 0.3.10)
  country_code  char(2),                            -- user's country at match time (for country boards)
  public        boolean     NOT NULL,               -- false if user was restricted at match time (D-016/D-017)
  occurred_at   timestamptz NOT NULL,               -- match end time; drives day/week/month bucketing
  UNIQUE NULLS NOT DISTINCT (match_id, entity_kind, entity_id, category_id)   -- overall row has NULL category; must still be unique (F3)
);
CREATE INDEX points_ledger_entity_idx ON ranking.points_ledger (entity_kind, entity_id, occurred_at);
CREATE INDEX points_ledger_time_idx   ON ranking.points_ledger (occurred_at);

-- ---------- Archived leaderboards (period ends, season ends) ----------
CREATE TABLE ranking.leaderboard_snapshots (
  board_key     text        NOT NULL,   -- e.g. 'weekly:user:global:all' / 'season:team:country:TR:all' / 'monthly:user:global:cat:<uuid>'
  period_start  timestamptz NOT NULL,
  period_end    timestamptz NOT NULL,
  rank          int         NOT NULL,
  entity_kind   text        NOT NULL,
  entity_id     uuid        NOT NULL,
  score         bigint      NOT NULL,
  PRIMARY KEY (board_key, period_start, rank)
);
CREATE INDEX leaderboard_snapshots_entity_idx ON ranking.leaderboard_snapshots (entity_kind, entity_id);
```

---

## 4. Leaderboards

### 4.1 Filters (all combinations supported)
| Filter | Values |
|---|---|
| Period | `daily`, `weekly`, `monthly`, `all_time` (+ `season` for SP/league boards) |
| Entity | `individual` (`user`), `team`, `couple` |
| Scope | `friends`, `country`, `global` |
| Category | all categories (default) or one category |

### 4.2 How they are served
- **Live boards are Redis sorted sets**, one per (period instance, entity, scope ∈ {global, country:XX}, category or all). Updated incrementally when a ledger row is written (same River job). Exact key names & TTLs: `redis-keys.md` (0.2.9).
- **Friends scope** is computed on read: the user's friend IDs (`social`) + `ZMSCORE` on the matching global board → sorted in memory (≤ 301 entries, D-027).
- **Restricted accounts** (`public = false`) are never added to global or country boards; on `identity.AccountTierChanged → full` their ledger rows for the current periods are replayed into the boards (D-016, D-017).
- **Team / couple boards** include only persistent teams/couples (D-028); team country = country of the captain at match time (D-029).
- **Daily / weekly / monthly** periods use **Europe/Istanbul** boundaries (weeks start Monday), consistent with the Daily Challenge day (D-029).
- At each period end, a worker job writes the top **1 000** of every board to `leaderboard_snapshots` (D-029) and lets Redis keys expire. Boards can always be rebuilt from `points_ledger`.

### 4.3 Category boards
A match touching several categories writes one ledger row per category (in addition to the overall row) with that category's share of the points. How the share is computed is fixed in 0.3.10.

**Future category ratings (D-042):** Category Master is unrated at launch; if category ratings are enabled later they use buckets `category:<category_id>`, entirely separate from the global `solo` / `team` / `couple` Ranked buckets.

### 4.4 Daily Challenge board (D-040)
Separate from every other board and from SP. Fed only by `game.DailyRunCompleted` (official, completed runs).
```sql
CREATE TABLE ranking.daily_challenge_entries (
  day            date        NOT NULL,
  user_id        uuid        NOT NULL REFERENCES identity.users(id),
  correct_count  smallint    NOT NULL,
  total_time_ms  int         NOT NULL,
  completed_at   timestamptz NOT NULL,
  country_code   char(2),
  public         boolean     NOT NULL,          -- false for restricted accounts (D-016)
  PRIMARY KEY (day, user_id)                    -- idempotent consumption; one official result per day
);
CREATE INDEX daily_challenge_entries_rank_idx
  ON ranking.daily_challenge_entries (day, correct_count DESC, total_time_ms ASC, completed_at ASC);
```
Ordering: correct ↓, total time ↑, completion time ↑. Live board in Redis `rank:daily:*` (redis-keys §6.4); rebuildable from this table.

---

## 5. `progression` schema

```sql
CREATE SCHEMA progression;

-- ---------- XP & level ----------
CREATE TABLE progression.user_progress (
  user_id     uuid        PRIMARY KEY REFERENCES identity.users(id) ON DELETE CASCADE,
  xp_total    bigint      NOT NULL DEFAULT 0 CHECK (xp_total >= 0),
  level       int         NOT NULL DEFAULT 1 CHECK (level >= 1),   -- derived from xp_total via the 0.3.10 curve, stored for fast reads
  updated_at  timestamptz NOT NULL
);

CREATE TABLE progression.xp_ledger (
  id          uuid        PRIMARY KEY,
  user_id     uuid        NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
  amount      int         NOT NULL CHECK (amount > 0),
  source      text        NOT NULL CHECK (source IN ('match','quest','badge','daily_challenge','season_reward','tournament')),
  source_ref  text        NOT NULL,                  -- match id / quest assignment id / …
  created_at  timestamptz NOT NULL,
  UNIQUE (user_id, source, source_ref)               -- idempotent event consumption
);

-- ---------- Play streak ----------
CREATE TABLE progression.play_streaks (
  user_id         uuid  PRIMARY KEY REFERENCES identity.users(id) ON DELETE CASCADE,
  current_days    int   NOT NULL DEFAULT 0,
  longest_days    int   NOT NULL DEFAULT 0,
  last_play_day   date,                              -- Europe/Istanbul calendar day of last completed match
  updated_at      timestamptz NOT NULL
);

-- ---------- Badges ----------
CREATE TABLE progression.badges (
  id          uuid        PRIMARY KEY,
  code        text        NOT NULL UNIQUE,           -- 'first_win', 'streak_7', 'category_master_history_gold' …
  family      text        NOT NULL,                  -- grouping for UI: wins, streaks, categories, social, seasons …
  tier        smallint    NOT NULL DEFAULT 1,        -- bronze/silver/gold style steps of the same family
  criteria    jsonb       NOT NULL,                  -- machine-checkable rule (0.3.10 / Phase 7)
  icon_key    text        NOT NULL,
  is_hidden   boolean     NOT NULL DEFAULT false,    -- secret badges
  is_active   boolean     NOT NULL DEFAULT true,
  created_at  timestamptz NOT NULL
);
CREATE TABLE progression.badge_translations (
  badge_id     uuid NOT NULL REFERENCES progression.badges(id) ON DELETE CASCADE,
  locale       text NOT NULL CHECK (locale IN ('tr','en')),
  name         text NOT NULL,
  description  text NOT NULL,
  PRIMARY KEY (badge_id, locale)
);
CREATE TABLE progression.user_badges (
  user_id     uuid        NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
  badge_id    uuid        NOT NULL REFERENCES progression.badges(id),
  earned_at   timestamptz NOT NULL,
  source_ref  text,                                  -- match / season that triggered it
  PRIMARY KEY (user_id, badge_id)
);

-- ---------- Quests ----------
CREATE TABLE progression.quests (
  id              uuid        PRIMARY KEY,
  code            text        NOT NULL UNIQUE,
  cadence         text        NOT NULL CHECK (cadence IN ('daily','weekly','seasonal')),
  objective       text        NOT NULL CHECK (objective IN ('play_matches','win_matches','correct_answers',
                                                            'play_mode','play_category','win_streak',
                                                            'play_with_friend','play_team_match','daily_challenge')),
  target          int         NOT NULL CHECK (target >= 1),
  filters         jsonb       NOT NULL DEFAULT '{}', -- e.g. {"match_type":"ranked"}, {"category_id":"…"}
  reward_xp       int         NOT NULL CHECK (reward_xp >= 0),
  reward_extra    jsonb       NOT NULL DEFAULT '{}', -- cosmetic / badge references (Phase 9)
  requires_full_account boolean NOT NULL DEFAULT false,   -- e.g. play_with_friend (D-016)
  season_id       uuid        REFERENCES ranking.seasons(id),  -- seasonal quests only
  is_active       boolean     NOT NULL DEFAULT true,
  created_at      timestamptz NOT NULL
);
CREATE TABLE progression.quest_translations (
  quest_id     uuid NOT NULL REFERENCES progression.quests(id) ON DELETE CASCADE,
  locale       text NOT NULL CHECK (locale IN ('tr','en')),
  title        text NOT NULL,
  description  text NOT NULL,
  PRIMARY KEY (quest_id, locale)
);
CREATE TABLE progression.quest_assignments (
  id            uuid        PRIMARY KEY,
  user_id       uuid        NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
  quest_id      uuid        NOT NULL REFERENCES progression.quests(id),
  period_start  timestamptz NOT NULL,
  period_end    timestamptz NOT NULL,
  progress      int         NOT NULL DEFAULT 0,
  completed_at  timestamptz,
  claimed_at    timestamptz,                         -- reward granted (XP ledger row written in same tx)
  UNIQUE (user_id, quest_id, period_start)
);
CREATE INDEX quest_assignments_open_idx ON progression.quest_assignments (user_id, period_end) WHERE claimed_at IS NULL;
```
> `progression` may reference `ranking.seasons` for seasonal quests: `progression → ranking` must be added to the allowed-dependency table (done in this task).

Rules:
- Quest progress applies to restricted accounts too, except quests with `requires_full_account`.
- Daily quests: **3** assigned per user per day, weekly: **3** per week (D-029); assignment job runs at period start or lazily on first app open.
- Rewards are **claimed** (button) rather than auto-granted, so the player sees them (D-029).

---

## 6. Season close (process, details in 0.3.10)
1. Status `active → closing` at `ends_at`; matches already started finish under the old season; new ranked matches attach to the next season.
2. Final ranks written to `season_standings.final_rank_*`; season boards snapshotted.
3. `ranking.SeasonEnded` emitted with final standings → progression (season badges/XP), commerce (season cosmetics), notification.
4. Soft reset of ratings per `reset_policy` (each change recorded in `rating_changes` with `match_id` = a synthetic season-reset id).
5. New `season_standings` rows created lazily on an entity's first match of the new season, starting at the bottom league (or placement rules from 0.3.10).

---

## 7. Events
| Event | Producer | Consumers |
|---|---|---|
| `ranking.RatingChanged` | ranking | matchmaking (cache refresh) |
| `ranking.LeagueChanged` | ranking | progression, notification |
| `ranking.SeasonEnded` / `SeasonStarted` | ranking | progression, commerce, notification |
| `progression.LevelUp` / `BadgeEarned` / `QuestCompleted` | progression | notification |

## 8. Confirmed proposals (D-029)
1. Rating buckets `solo`, `team`, `couple` (final in 0.3.10).
2. Leaderboard periods on **Europe/Istanbul** time, week starts Monday.
3. Team country = captain's country at match time.
4. Top **1 000** per board archived at each period end.
5. **3** daily + **3** weekly quests per user; rewards claimed manually.
