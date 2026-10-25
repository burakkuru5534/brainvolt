-- Rewarded-ad effects in progression (D-103): bonus XP after a match
-- (source 'ad_bonus', once per match), extra quest rerolls for the day, and
-- a repaired play streak. ad_rewards_applied makes the event handler
-- idempotent.
-- +goose Up
ALTER TABLE progression.xp_ledger DROP CONSTRAINT xp_ledger_source_check;
ALTER TABLE progression.xp_ledger ADD CONSTRAINT xp_ledger_source_check
  CHECK (source IN ('match','quest','badge','daily_challenge','season_reward','tournament','ad_bonus'));

ALTER TABLE progression.quest_rerolls ADD COLUMN bonus int NOT NULL DEFAULT 0;

CREATE TABLE progression.ad_rewards_applied (
  reward_id   uuid        PRIMARY KEY,
  user_id     uuid        NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
  placement   text        NOT NULL,
  applied_at  timestamptz NOT NULL
);

-- +goose Down
DROP TABLE progression.ad_rewards_applied;
ALTER TABLE progression.quest_rerolls DROP COLUMN bonus;
DELETE FROM progression.xp_ledger WHERE source = 'ad_bonus';
ALTER TABLE progression.xp_ledger DROP CONSTRAINT xp_ledger_source_check;
ALTER TABLE progression.xp_ledger ADD CONSTRAINT xp_ledger_source_check
  CHECK (source IN ('match','quest','badge','daily_challenge','season_reward','tournament'));
