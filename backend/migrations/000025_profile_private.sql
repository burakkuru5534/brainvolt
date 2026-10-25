-- Private profiles (D-101): a player may hide their profile details (level,
-- matches, wins, badges, streak) from everyone but their friends. Their
-- name and avatar stay on leaderboards and in matches.
-- +goose Up
ALTER TABLE profile.settings
  ADD COLUMN profile_private boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE profile.settings DROP COLUMN profile_private;
