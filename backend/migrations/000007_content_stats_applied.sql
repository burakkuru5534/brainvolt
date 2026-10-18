-- Matches whose answers are already in content.question_stats: the stats
-- event is delivered at least once, so each match is counted only once
-- (schema-content §5).
-- +goose Up
CREATE TABLE content.stats_applied (
  match_id    uuid        PRIMARY KEY,   -- game.matches id (plain id: content does not reference game)
  applied_at  timestamptz NOT NULL
);

-- +goose Down
DROP TABLE content.stats_applied;
