-- Daily Challenge runs (category-modes §5, D-040): solo runs owned by game,
-- isolated from head-to-head tables (schema-platform §6).
-- +goose Up
CREATE TABLE game.daily_runs (
  id               uuid        PRIMARY KEY,
  day              date        NOT NULL,
  user_id          uuid        NOT NULL REFERENCES identity.users(id),
  set_id           uuid        NOT NULL REFERENCES content.question_sets(id),
  locale           text        NOT NULL CHECK (locale IN ('tr','en')),
  kind             text        NOT NULL CHECK (kind IN ('official','practice')),
  status           text        NOT NULL CHECK (status IN ('in_progress','completed','void_server_error','void_not_started')),
  account_tier_at_start text   NOT NULL CHECK (account_tier_at_start IN ('guest','unverified','full')),
  ruleset_version  int         NOT NULL,
  started_at       timestamptz NOT NULL,
  completed_at     timestamptz,
  correct_count    smallint    CHECK (correct_count BETWEEN 0 AND 10),
  total_time_ms    int         CHECK (total_time_ms >= 0),
  score            int,
  disconnects      smallint    NOT NULL DEFAULT 0,
  CHECK (status <> 'completed' OR (completed_at IS NOT NULL AND correct_count IS NOT NULL AND total_time_ms IS NOT NULL))
);
CREATE UNIQUE INDEX daily_runs_one_official_uq ON game.daily_runs (day, user_id)
  WHERE kind = 'official' AND status IN ('in_progress','completed');
CREATE INDEX daily_runs_user_idx ON game.daily_runs (user_id, day DESC);
CREATE INDEX daily_runs_open_idx ON game.daily_runs (started_at) WHERE status = 'in_progress';

CREATE TABLE game.daily_run_answers (
  run_id             uuid        NOT NULL REFERENCES game.daily_runs(id) ON DELETE CASCADE,
  position           smallint    NOT NULL CHECK (position BETWEEN 1 AND 10),
  question_id        uuid        NOT NULL REFERENCES content.questions(id),
  question_revision  int         NOT NULL,
  option_order       smallint[]  NOT NULL,
  show_at            timestamptz NOT NULL,
  deadline_at        timestamptz NOT NULL,
  option_id          uuid        REFERENCES content.options(id),
  received_at        timestamptz,
  response_ms        int         CHECK (response_ms >= 0),
  is_correct         boolean     NOT NULL,
  points             int         NOT NULL DEFAULT 0,
  created_at         timestamptz NOT NULL,
  PRIMARY KEY (run_id, position)
);

-- +goose Down
DROP TABLE game.daily_run_answers;
DROP TABLE game.daily_runs;
