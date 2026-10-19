-- name: CountAuthorQuizzes :one
SELECT count(*)::int FROM creator.quizzes WHERE author_user_id = @author AND deleted_at IS NULL;

-- name: InsertQuiz :exec
INSERT INTO creator.quizzes (id, author_user_id, locale, title, description, category_id, question_set_id, answer_window_ms,
  question_count, created_at, updated_at)
VALUES (@id, @author, @locale, @title, sqlc.narg(description), @category_id, @set_id, @window_ms, @question_count, @now, @now);

-- name: GetQuiz :one
SELECT * FROM creator.quizzes WHERE id = @id AND deleted_at IS NULL;

-- name: GetQuizForUpdate :one
SELECT * FROM creator.quizzes WHERE id = @id AND deleted_at IS NULL FOR UPDATE;

-- An edit: a published or submitted quiz goes back to submitted / draft (schema-platform §5).
-- name: UpdateQuiz :exec
UPDATE creator.quizzes SET title = @title, description = sqlc.narg(description), category_id = @category_id,
  answer_window_ms = @window_ms, question_count = @question_count, status = @status,
  submitted_at = sqlc.narg(submitted_at), published_at = NULL, version = version + 1, updated_at = @now
WHERE id = @id;

-- name: SetQuizStatus :exec
UPDATE creator.quizzes SET status = @status, submitted_at = sqlc.narg(submitted_at), published_at = sqlc.narg(published_at),
  reviewed_by = sqlc.narg(reviewed_by), rejection_reason = sqlc.narg(rejection_reason), updated_at = @now
WHERE id = @id;

-- name: DeleteQuiz :exec
UPDATE creator.quizzes SET deleted_at = @now, status = 'unpublished' WHERE id = @id;

-- name: AuthorQuizzes :many
SELECT * FROM creator.quizzes WHERE author_user_id = @author AND deleted_at IS NULL ORDER BY updated_at DESC;

-- name: PublishedQuizzes :many
SELECT * FROM creator.quizzes
WHERE status = 'published' AND deleted_at IS NULL
  AND (sqlc.narg(locale)::text IS NULL OR locale = sqlc.narg(locale)::text)
  AND (sqlc.narg(category_id)::uuid IS NULL OR category_id = sqlc.narg(category_id)::uuid)
ORDER BY like_count DESC, published_at DESC
LIMIT @lim OFFSET @off;

-- name: SubmittedQuizzes :many
SELECT * FROM creator.quizzes WHERE status = 'submitted' AND deleted_at IS NULL ORDER BY submitted_at LIMIT 100;

-- name: Like :execrows
INSERT INTO creator.quiz_likes (quiz_id, user_id, liked_at) VALUES (@quiz_id, @user_id, @now) ON CONFLICT DO NOTHING;

-- name: Unlike :execrows
DELETE FROM creator.quiz_likes WHERE quiz_id = @quiz_id AND user_id = @user_id;

-- name: AddLikes :exec
UPDATE creator.quizzes SET like_count = like_count + @delta WHERE id = @id;

-- name: Liked :one
SELECT EXISTS (SELECT 1 FROM creator.quiz_likes WHERE quiz_id = @quiz_id AND user_id = @user_id);

-- name: CountPlay :exec
UPDATE creator.quizzes SET play_count = play_count + 1 WHERE id = @id;
