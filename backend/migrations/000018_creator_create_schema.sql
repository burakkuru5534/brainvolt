-- creator schema (schema-platform §5): player-made quizzes around a
-- content question set, their review state and likes. answer_window_ms
-- holds the author's 10 / 15 / 20 s choice (category-modes §6).
-- +goose Up
CREATE SCHEMA creator;

CREATE TABLE creator.quizzes (
  id                uuid        PRIMARY KEY,
  author_user_id    uuid        NOT NULL REFERENCES identity.users(id),
  locale            text        NOT NULL CHECK (locale IN ('tr','en')),
  title             text        NOT NULL CHECK (char_length(title) BETWEEN 3 AND 60),
  description       text        CHECK (char_length(description) <= 300),
  category_id       uuid        REFERENCES content.categories(id),
  question_set_id   uuid        NOT NULL UNIQUE REFERENCES content.question_sets(id),
  answer_window_ms  int         NOT NULL DEFAULT 15000 CHECK (answer_window_ms IN (10000, 15000, 20000)),
  question_count    smallint    NOT NULL CHECK (question_count BETWEEN 5 AND 30),
  status            text        NOT NULL DEFAULT 'draft'
                    CHECK (status IN ('draft','submitted','published','rejected','unpublished')),
  submitted_at      timestamptz,
  published_at      timestamptz,
  reviewed_by       uuid        REFERENCES identity.users(id),
  rejection_reason  text,
  play_count        int         NOT NULL DEFAULT 0,
  like_count        int         NOT NULL DEFAULT 0,
  version           int         NOT NULL DEFAULT 1,
  created_at        timestamptz NOT NULL,
  updated_at        timestamptz NOT NULL,
  deleted_at        timestamptz
);
CREATE INDEX quizzes_author_idx ON creator.quizzes (author_user_id) WHERE deleted_at IS NULL;
CREATE INDEX quizzes_published_idx ON creator.quizzes (locale, category_id, published_at DESC) WHERE status = 'published' AND deleted_at IS NULL;
CREATE INDEX quizzes_submitted_idx ON creator.quizzes (submitted_at) WHERE status = 'submitted' AND deleted_at IS NULL;

CREATE TABLE creator.quiz_likes (
  quiz_id   uuid NOT NULL REFERENCES creator.quizzes(id) ON DELETE CASCADE,
  user_id   uuid NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
  liked_at  timestamptz NOT NULL,
  PRIMARY KEY (quiz_id, user_id)
);

-- +goose Down
DROP SCHEMA creator CASCADE;
