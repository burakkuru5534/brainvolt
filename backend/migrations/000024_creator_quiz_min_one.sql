-- Creator Quiz (D-031 amended 2026-10-24): a quiz may hold 1–30 questions;
-- the Question Lab starts from one question and grows with "Add question".
-- +goose Up
ALTER TABLE creator.quizzes DROP CONSTRAINT IF EXISTS quizzes_question_count_check;
ALTER TABLE creator.quizzes ADD CONSTRAINT quizzes_question_count_check
  CHECK (question_count BETWEEN 1 AND 30);

-- +goose Down
ALTER TABLE creator.quizzes DROP CONSTRAINT IF EXISTS quizzes_question_count_check;
-- NOT VALID: quizzes saved with fewer than 5 questions stay readable.
ALTER TABLE creator.quizzes ADD CONSTRAINT quizzes_question_count_check
  CHECK (question_count BETWEEN 5 AND 30) NOT VALID;
