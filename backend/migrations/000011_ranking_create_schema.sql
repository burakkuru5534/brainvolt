-- ranking schema (docs/domain/schema-ranking.md §3): Glicko-2 ratings and
-- their history, seasons, leagues, season standings, the points ledger
-- behind every leaderboard and archived leaderboards. Daily Challenge
-- entries come with Phase 8.
-- +goose Up
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

-- Matches already consumed (idempotent game.MatchCompleted handling).
CREATE TABLE ranking.processed_matches (
  match_id      uuid        PRIMARY KEY,
  processed_at  timestamptz NOT NULL
);

INSERT INTO ranking.leagues (code, sort_order, divisions, rules) VALUES
  ('bronze',   1, 3, '{"floor": 0,    "divisions": [0, 100, 200]}'),
  ('silver',   2, 3, '{"floor": 300,  "divisions": [300, 466, 633]}'),
  ('gold',     3, 3, '{"floor": 800,  "divisions": [800, 1066, 1333]}'),
  ('platinum', 4, 3, '{"floor": 1600, "divisions": [1600, 2000, 2400]}'),
  ('diamond',  5, 1, '{"floor": 2800, "divisions": [2800]}');
INSERT INTO ranking.league_translations (code, locale, name) VALUES
  ('bronze', 'tr', 'Bronz'), ('bronze', 'en', 'Bronze'),
  ('silver', 'tr', 'Gümüş'), ('silver', 'en', 'Silver'),
  ('gold', 'tr', 'Altın'), ('gold', 'en', 'Gold'),
  ('platinum', 'tr', 'Platin'), ('platinum', 'en', 'Platinum'),
  ('diamond', 'tr', 'Elmas'), ('diamond', 'en', 'Diamond');

-- +goose Down
DROP SCHEMA ranking CASCADE;
