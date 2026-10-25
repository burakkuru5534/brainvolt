-- Commerce, first part (D-103): rewarded ads verified by AdMob's
-- server-side verification (SSV) and the entitlements other modules ask
-- about (Plus arrives in a later phase; schema-platform.md §4). A reward is
-- reserved before the ad is shown (pending), granted once AdMob's signed
-- callback names it, and expires unused after 30 minutes.
-- +goose Up
CREATE SCHEMA commerce;

CREATE TABLE commerce.entitlements (
  user_id      uuid        NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
  code         text        NOT NULL,               -- 'plus', 'no_ads', …
  source_kind  text        NOT NULL CHECK (source_kind IN ('subscription','purchase','reward','admin')),
  source_id    text        NOT NULL,
  starts_at    timestamptz NOT NULL,
  ends_at      timestamptz,
  PRIMARY KEY (user_id, code, source_kind, source_id)
);

CREATE TABLE commerce.ad_rewards (
  id              uuid        PRIMARY KEY,           -- sent to AdMob as custom_data
  user_id         uuid        NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
  network         text        NOT NULL DEFAULT 'admob',
  placement       text        NOT NULL CHECK (placement IN ('bonus_xp','quest_reroll','streak_repair')),
  ref             text        NOT NULL DEFAULT '',   -- the match of a bonus_xp reward
  status          text        NOT NULL CHECK (status IN ('pending','granted')),
  transaction_id  text,                              -- the network's id once verified
  created_at      timestamptz NOT NULL,
  expires_at      timestamptz NOT NULL,
  granted_at      timestamptz,
  UNIQUE (network, transaction_id)
);
CREATE INDEX ad_rewards_user_idx ON commerce.ad_rewards (user_id, created_at);

-- +goose Down
DROP SCHEMA commerce CASCADE;
