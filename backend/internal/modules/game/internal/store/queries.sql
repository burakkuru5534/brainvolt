-- game queries (docs/domain/schema-game.md). Only the game schema.

-- ---------- Creation (T1) ----------

-- name: InsertMatch :exec
INSERT INTO game.matches (id, match_type, format, team_answer_mode, category_mode, is_rated, rating_bucket,
  season_id, question_count, answer_window_ms, locales, question_set_id, config_extra, ruleset_version,
  origin, origin_ref, parent_match_id, status, created_at, updated_at)
VALUES (@id, @match_type, @format, @team_answer_mode, @category_mode, @is_rated, @rating_bucket,
  @season_id, @question_count, @answer_window_ms, @locales, @question_set_id, @config_extra, @ruleset_version,
  @origin, @origin_ref, @parent_match_id, 'waiting', @now, @now);

-- name: InsertSide :exec
INSERT INTO game.match_sides (match_id, side_no, team_id, couple_id) VALUES ($1, $2, $3, $4);

-- name: InsertParticipant :exec
INSERT INTO game.match_participants (match_id, user_id, side_no, seat_no, locale, account_tier_at_start, joined_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: InsertTransition :exec
INSERT INTO game.match_transitions (match_id, seq, from_status, to_status, reason, actor, actor_id, at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (match_id, seq) DO NOTHING;

-- name: GetMatch :one
SELECT * FROM game.matches WHERE id = $1;

-- name: ListParticipants :many
SELECT * FROM game.match_participants WHERE match_id = $1 ORDER BY side_no, seat_no;

-- Questions any of these users saw since @since (no-repeat window, D-032).
-- name: RecentQuestionIDs :many
SELECT DISTINCT r.question_id
FROM game.match_rounds r
JOIN game.match_participants p ON p.match_id = r.match_id
WHERE p.user_id = ANY(@user_ids::uuid[]) AND r.revealed_at >= @since;

-- ---------- Lifecycle ----------

-- name: SetMatchStatus :exec
UPDATE game.matches
SET status = @status,
    ready_at = CASE WHEN @status = 'ready' THEN @at ELSE ready_at END,
    started_at = CASE WHEN @status = 'in_progress' THEN @at ELSE started_at END,
    updated_at = @at
WHERE id = @id AND status NOT IN ('completed', 'cancelled');

-- name: InsertRound :exec
INSERT INTO game.match_rounds (match_id, round_no, question_id, question_revision, difficulty_used, option_order,
  answering_side_no, answering_user_id, dispatched_at, deadline_at, revealed_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (match_id, round_no) DO NOTHING;

-- name: InsertAnswer :exec
INSERT INTO game.answers (match_id, round_no, user_id, side_no, option_id, received_at, response_ms,
  is_correct, points_awarded, is_side_answer, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (match_id, round_no, user_id) DO NOTHING;

-- name: InsertRoundResult :exec
INSERT INTO game.round_results (match_id, round_no, side_no, is_correct, points, detail)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (match_id, round_no, side_no) DO NOTHING;

-- ---------- End (T3, T5, T6, T7) ----------

-- name: CompleteMatch :execrows
UPDATE game.matches
SET status = 'completed', ended_at = @at, end_reason = @end_reason, winner_side_no = @winner_side_no,
    is_draw = @is_draw, updated_at = @at
WHERE id = @id AND status NOT IN ('completed', 'cancelled');

-- name: CancelMatch :execrows
UPDATE game.matches
SET status = 'cancelled', ended_at = @at, cancel_reason = @cancel_reason, updated_at = @at
WHERE id = @id AND status NOT IN ('completed', 'cancelled');

-- name: SetSideResult :exec
UPDATE game.match_sides
SET score = @score, correct_count = @correct_count, result = @result, forfeited_at = @forfeited_at
WHERE match_id = @match_id AND side_no = @side_no;

-- name: SetParticipantResult :exec
UPDATE game.match_participants
SET status = @status, score_contribution = @score_contribution, correct_count = @correct_count,
    answered_count = @answered_count, disconnect_count = @disconnect_count,
    disconnected_ms_total = @disconnected_ms_total,
    left_at = CASE WHEN @status IN ('left', 'forfeited') THEN @at ELSE left_at END
WHERE match_id = @match_id AND user_id = @user_id;

-- name: SideScoreFromRounds :many
SELECT side_no, coalesce(sum(points), 0)::bigint AS points
FROM game.round_results WHERE match_id = $1 GROUP BY side_no ORDER BY side_no;

-- name: RoundStats :many
SELECT r.question_id, p.locale,
       count(*)::bigint AS asked,
       count(*) FILTER (WHERE a.is_correct)::bigint AS correct,
       count(*) FILTER (WHERE a.option_id IS NULL)::bigint AS timed_out,
       coalesce(sum(a.response_ms) FILTER (WHERE a.is_correct), 0)::bigint AS sum_response_ms
FROM game.answers a
JOIN game.match_rounds r ON r.match_id = a.match_id AND r.round_no = a.round_no
JOIN game.match_participants p ON p.match_id = a.match_id AND p.user_id = a.user_id
WHERE a.match_id = $1
GROUP BY r.question_id, p.locale
ORDER BY r.question_id, p.locale;
