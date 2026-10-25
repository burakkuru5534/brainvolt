-- Friend tournaments (D-109): a full account sets up a single-elimination
-- 1v1 tournament for its friends (4, 8 or 16 places). Only the owner and
-- the owner's friends see and join it; it is unrated and gives no XP or
-- Volt, the winner gets a trophy on the profile. Setting one up costs a
-- fee in Volt (free with Plus), refunded when it is cancelled.
-- +goose Up
ALTER TABLE tournament.tournaments
  ADD COLUMN kind text NOT NULL DEFAULT 'official' CHECK (kind IN ('official','friends')),
  ADD COLUMN fee int NOT NULL DEFAULT 0 CHECK (fee >= 0);
CREATE INDEX tournaments_friends_owner_idx ON tournament.tournaments (created_by, starts_at) WHERE kind = 'friends';

-- +goose Down
DROP INDEX tournament.tournaments_friends_owner_idx;
ALTER TABLE tournament.tournaments DROP COLUMN fee, DROP COLUMN kind;
