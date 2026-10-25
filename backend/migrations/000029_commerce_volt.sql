-- Volt, BrainVolt's one currency (D-105): a balance per user and a ledger
-- holding every movement. The balance lives only on the server; each
-- movement carries a key that is unique per user, so a retried reward or
-- purchase is written once. Earned by playing (quests, the Daily
-- Challenge, levels, tournaments, seasons, a rewarded ad in the store)
-- and with Plus (400 a month, not during a free trial); spent on store
-- items. Entitlements remember whether a subscription period is a trial.
-- +goose Up
CREATE TABLE commerce.volt_wallets (
  user_id     uuid        PRIMARY KEY REFERENCES identity.users(id) ON DELETE CASCADE,
  balance     bigint      NOT NULL DEFAULT 0,
  updated_at  timestamptz NOT NULL
);

CREATE TABLE commerce.volt_ledger (
  id             uuid        PRIMARY KEY,
  user_id        uuid        NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
  amount         int         NOT NULL CHECK (amount <> 0),
  reason         text        NOT NULL CHECK (reason IN ('quest','daily_challenge','level_up','tournament','season',
                                                         'ad','plus_monthly','plus_refund','item','admin')),
  ref            text        NOT NULL DEFAULT '',     -- the quest, run, item, … it came from
  key            text        NOT NULL,                -- idempotency key, unique per user
  balance_after  bigint      NOT NULL,
  created_at     timestamptz NOT NULL,
  UNIQUE (user_id, key)
);
CREATE INDEX volt_ledger_user_idx ON commerce.volt_ledger (user_id, created_at DESC);

ALTER TABLE commerce.entitlements ADD COLUMN trial boolean NOT NULL DEFAULT false;

ALTER TABLE commerce.ad_rewards DROP CONSTRAINT ad_rewards_placement_check;
ALTER TABLE commerce.ad_rewards ADD CONSTRAINT ad_rewards_placement_check
  CHECK (placement IN ('bonus_xp','quest_reroll','streak_repair','volt'));

-- +goose Down
DELETE FROM commerce.ad_rewards WHERE placement = 'volt';
ALTER TABLE commerce.ad_rewards DROP CONSTRAINT ad_rewards_placement_check;
ALTER TABLE commerce.ad_rewards ADD CONSTRAINT ad_rewards_placement_check
  CHECK (placement IN ('bonus_xp','quest_reroll','streak_repair'));
ALTER TABLE commerce.entitlements DROP COLUMN trial;
DROP TABLE commerce.volt_ledger;
DROP TABLE commerce.volt_wallets;
