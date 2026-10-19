-- Daily Challenge board (schema-ranking §4.4): one official result per user
-- and day, ordered correct ↓, total time ↑, completion ↑.
-- +goose Up
CREATE TABLE ranking.daily_challenge_entries (
  day            date        NOT NULL,
  user_id        uuid        NOT NULL REFERENCES identity.users(id),
  correct_count  smallint    NOT NULL,
  total_time_ms  int         NOT NULL,
  completed_at   timestamptz NOT NULL,
  country_code   char(2),
  public         boolean     NOT NULL,
  PRIMARY KEY (day, user_id)
);
CREATE INDEX daily_challenge_entries_rank_idx
  ON ranking.daily_challenge_entries (day, correct_count DESC, total_time_ms ASC, completed_at ASC);

-- +goose Down
DROP TABLE ranking.daily_challenge_entries;
