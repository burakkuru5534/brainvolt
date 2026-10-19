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

-- name: ListSides :many
SELECT side_no, score, correct_count, result FROM game.match_sides WHERE match_id = $1 ORDER BY side_no;

-- name: ListRounds :many
SELECT round_no, question_id, option_order FROM game.match_rounds WHERE match_id = $1 ORDER BY round_no;

-- name: ListAnswers :many
SELECT round_no, user_id, side_no, option_id, response_ms, is_correct, points_awarded
FROM game.answers WHERE match_id = $1 ORDER BY round_no, side_no, user_id;

-- name: ListRoundResults :many
SELECT round_no, side_no, is_correct, points FROM game.round_results WHERE match_id = $1 ORDER BY round_no, side_no;

-- Each round's category, side results and correct players (ranking:
-- category boards, team entities).
-- name: RoundOutcomes :many
SELECT r.round_no, sc.category_id,
       coalesce(bool_or(rr.is_correct) FILTER (WHERE rr.side_no = 1), false)::boolean AS side1_correct,
       coalesce(bool_or(rr.is_correct) FILTER (WHERE rr.side_no = 2), false)::boolean AS side2_correct,
       coalesce(array_agg(DISTINCT a.user_id) FILTER (WHERE a.is_correct), '{}')::uuid[] AS correct_user_ids
FROM game.match_rounds r
JOIN content.questions q ON q.id = r.question_id
JOIN content.subcategories sc ON sc.id = q.subcategory_id
LEFT JOIN game.round_results rr ON rr.match_id = r.match_id AND rr.round_no = r.round_no
LEFT JOIN game.answers a ON a.match_id = r.match_id AND a.round_no = r.round_no
WHERE r.match_id = $1
GROUP BY r.round_no, sc.category_id
ORDER BY r.round_no;

-- name: SideEntities :many
SELECT side_no, team_id, couple_id FROM game.match_sides WHERE match_id = $1 ORDER BY side_no;

-- ---------- Daily Challenge runs (schema-platform §6) ----------

-- name: InsertDailyRun :exec
INSERT INTO game.daily_runs (id, day, user_id, set_id, locale, kind, status, account_tier_at_start, ruleset_version, started_at)
VALUES (@id, @day, @user_id, @set_id, @locale, @kind, 'in_progress', @tier, @ruleset_version, @now);

-- name: GetDailyRun :one
SELECT * FROM game.daily_runs WHERE id = @id;

-- name: GetDailyRunForUpdate :one
SELECT * FROM game.daily_runs WHERE id = @id FOR UPDATE;

-- name: OfficialRunOf :one
SELECT * FROM game.daily_runs
WHERE day = @day AND user_id = @user_id AND kind = 'official' AND status IN ('in_progress','completed');

-- name: CompleteDailyRun :exec
UPDATE game.daily_runs SET status = 'completed', completed_at = @completed_at, correct_count = @correct_count,
  total_time_ms = @total_time_ms, score = @score
WHERE id = @id;

-- name: VoidDailyRun :exec
UPDATE game.daily_runs SET status = @status, completed_at = @now WHERE id = @id;

-- name: CountDailyDisconnect :exec
UPDATE game.daily_runs SET disconnects = disconnects + 1 WHERE id = @id;

-- name: DailyAnswers :many
SELECT * FROM game.daily_run_answers WHERE run_id = @run_id ORDER BY position;

-- name: ShowDailyQuestion :exec
INSERT INTO game.daily_run_answers (run_id, position, question_id, question_revision, option_order, show_at, deadline_at, is_correct, created_at)
VALUES (@run_id, @position, @question_id, @question_revision, @option_order, @show_at, @deadline_at, false, @now);

-- name: AnswerDailyQuestion :exec
UPDATE game.daily_run_answers SET option_id = @option_id, received_at = @received_at, response_ms = @response_ms,
  is_correct = @is_correct, points = @points
WHERE run_id = @run_id AND position = @position;

-- Runs left running (abandoned apps), settled by the worker.
-- name: StaleDailyRuns :many
SELECT id FROM game.daily_runs WHERE status = 'in_progress' AND started_at < @before ORDER BY started_at LIMIT 200;
