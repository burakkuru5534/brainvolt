-- social schema (docs/domain/schema-social.md §2): friend requests,
-- friendships (two directed rows per pair) and blocks.
-- +goose Up
CREATE SCHEMA social;

CREATE TABLE social.friend_requests (
  id            uuid        PRIMARY KEY,
  from_user_id  uuid        NOT NULL REFERENCES identity.users(id),
  to_user_id    uuid        NOT NULL REFERENCES identity.users(id),
  via           text        NOT NULL CHECK (via IN ('search','invite_code','post_match','profile')),
  status        text        NOT NULL DEFAULT 'pending'
                CHECK (status IN ('pending','accepted','declined','cancelled','expired')),
  created_at    timestamptz NOT NULL,
  responded_at  timestamptz,
  expires_at    timestamptz NOT NULL,                       -- created_at + 30 days (D-027)
  CHECK (from_user_id <> to_user_id)
);
-- at most one pending request per unordered pair
CREATE UNIQUE INDEX friend_requests_pending_pair_uq
  ON social.friend_requests (LEAST(from_user_id,to_user_id), GREATEST(from_user_id,to_user_id))
  WHERE status = 'pending';
CREATE INDEX friend_requests_inbox_idx ON social.friend_requests (to_user_id) WHERE status = 'pending';
CREATE INDEX friend_requests_outbox_idx ON social.friend_requests (from_user_id) WHERE status = 'pending';
-- daily outgoing cap (50/day) counts by sender and time
CREATE INDEX friend_requests_sent_idx ON social.friend_requests (from_user_id, created_at);

-- Two directed rows per friendship (A→B and B→A), written in one transaction.
CREATE TABLE social.friendships (
  user_id     uuid        NOT NULL REFERENCES identity.users(id),
  friend_id   uuid        NOT NULL REFERENCES identity.users(id),
  created_at  timestamptz NOT NULL,
  PRIMARY KEY (user_id, friend_id),
  CHECK (user_id <> friend_id)
);

CREATE TABLE social.blocks (
  blocker_id  uuid        NOT NULL REFERENCES identity.users(id),
  blocked_id  uuid        NOT NULL REFERENCES identity.users(id),
  created_at  timestamptz NOT NULL,
  PRIMARY KEY (blocker_id, blocked_id),
  CHECK (blocker_id <> blocked_id)
);
CREATE INDEX blocks_blocked_idx ON social.blocks (blocked_id);

-- +goose Down
DROP SCHEMA social CASCADE;
