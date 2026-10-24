-- +goose Up
-- Redis replacement tables: matchmaking queue, idempotency, cooldowns and
-- event counters that previously lived in Redis.
CREATE SCHEMA IF NOT EXISTS matchmaking;
CREATE SCHEMA IF NOT EXISTS platform;

-- Idempotency keys (replaces Redis idem:* keys)
CREATE TABLE IF NOT EXISTS platform.idempotency_keys (
  key         text        PRIMARY KEY,
  user_id     text        NOT NULL,
  state       text        NOT NULL CHECK (state IN ('pending','done')),
  hash        text        NOT NULL,
  status_code int,
  headers     jsonb,
  body        bytea,
  created_at  timestamptz NOT NULL DEFAULT now(),
  expires_at  timestamptz NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_idempotency_keys_expires ON platform.idempotency_keys (expires_at);

-- Matchmaking tickets (replaces Redis mm:t:*, mm:q:*, mm:user:* keys)
CREATE TABLE IF NOT EXISTS matchmaking.tickets (
  id          uuid        PRIMARY KEY,
  queue       text        NOT NULL,
  user_ids    uuid[]      NOT NULL,
  request     jsonb       NOT NULL,
  entry_json  jsonb       NOT NULL,
  rating      float8      NOT NULL,
  rd          float8      NOT NULL,
  status      text        NOT NULL DEFAULT 'searching'
              CHECK (status IN ('searching','matched','cancelled','expired')),
  match_id    uuid,
  enqueued_at timestamptz NOT NULL DEFAULT now(),
  expires_at  timestamptz NOT NULL,
  updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_mm_tickets_queue ON matchmaking.tickets (queue, rating) WHERE status = 'searching';
CREATE INDEX IF NOT EXISTS idx_mm_tickets_expires ON matchmaking.tickets (expires_at) WHERE status = 'searching';
CREATE INDEX IF NOT EXISTS idx_mm_tickets_user ON matchmaking.tickets USING GIN (user_ids) WHERE status = 'searching';

-- Matchmaking cooldowns (replaces Redis mm:cooldown:* keys)
CREATE TABLE IF NOT EXISTS matchmaking.cooldowns (
  user_id     uuid        PRIMARY KEY,
  expires_at  timestamptz NOT NULL
);

-- Matchmaking event counters (replaces Redis mm:missed:*, mm:leaves:* sorted sets)
CREATE TABLE IF NOT EXISTS matchmaking.event_counts (
  id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     uuid        NOT NULL,
  event_kind  text        NOT NULL CHECK (event_kind IN ('missed_ready','leave')),
  occurred_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_mm_event_counts_user ON matchmaking.event_counts (user_id, event_kind, occurred_at);
CREATE INDEX IF NOT EXISTS idx_mm_event_counts_cleanup ON matchmaking.event_counts (occurred_at);

-- Matchmaking match memory for re-queue (replaces Redis mm:m:* hashes)
CREATE TABLE IF NOT EXISTS matchmaking.match_memory (
  match_id    uuid        NOT NULL,
  user_id     uuid        NOT NULL,
  entry_json  jsonb       NOT NULL,
  created_at  timestamptz NOT NULL DEFAULT now(),
  expires_at  timestamptz NOT NULL,
  PRIMARY KEY (match_id, user_id)
);

-- Ranking leaderboard entries (replaces Redis rank:lb:* sorted sets)
CREATE TABLE IF NOT EXISTS ranking.leaderboard_entries (
  board_key   text        NOT NULL,
  member      text        NOT NULL,
  score       float8      NOT NULL,
  updated_at  timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (board_key, member)
);
CREATE INDEX IF NOT EXISTS idx_ranking_lb_scores ON ranking.leaderboard_entries (board_key, score DESC, member ASC);

-- +goose Down
DROP TABLE IF EXISTS ranking.leaderboard_entries;
DROP TABLE IF EXISTS matchmaking.match_memory;
DROP TABLE IF EXISTS matchmaking.event_counts;
DROP TABLE IF EXISTS matchmaking.cooldowns;
DROP TABLE IF EXISTS matchmaking.tickets;
DROP TABLE IF EXISTS platform.idempotency_keys;
