-- Season Pass (D-108): 40 tiers a season, one per 500 XP earned inside the
-- season. Everyone has the free track; the premium track is bought with
-- 900 Volt (season_pass). Each reward is claimed once (season_pass_claims).
-- +goose Up
CREATE TABLE progression.season_pass (
  user_id     uuid        NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
  season_id   uuid        NOT NULL,
  premium_at  timestamptz NOT NULL,
  PRIMARY KEY (user_id, season_id)
);

CREATE TABLE progression.season_pass_claims (
  user_id     uuid        NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
  season_id   uuid        NOT NULL,
  track       text        NOT NULL CHECK (track IN ('free','premium')),
  tier        int         NOT NULL CHECK (tier BETWEEN 1 AND 40),
  claimed_at  timestamptz NOT NULL,
  PRIMARY KEY (user_id, season_id, track, tier)
);

-- Season XP is summed from the ledger by time.
CREATE INDEX xp_ledger_user_time_idx ON progression.xp_ledger (user_id, created_at);

-- +goose Down
DROP INDEX progression.xp_ledger_user_time_idx;
DROP TABLE progression.season_pass_claims;
DROP TABLE progression.season_pass;
