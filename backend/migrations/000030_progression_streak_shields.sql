-- Streak shields (D-105): bought with Volt (100) or one a week with Plus,
-- at most 2 held. A shield is used up for each missed day when the player
-- plays again, so the streak goes on. plus_shield_week is the week whose
-- Plus shield was given.
-- +goose Up
ALTER TABLE progression.play_streaks
  ADD COLUMN shields int NOT NULL DEFAULT 0 CHECK (shields BETWEEN 0 AND 2),
  ADD COLUMN plus_shield_week date;

-- +goose Down
ALTER TABLE progression.play_streaks DROP COLUMN plus_shield_week, DROP COLUMN shields;
