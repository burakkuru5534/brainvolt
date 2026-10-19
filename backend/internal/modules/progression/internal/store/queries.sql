-- ---------- Consumption ----------

-- name: MarkProcessed :execrows
INSERT INTO progression.processed_matches (match_id, processed_at) VALUES (@match_id, @now) ON CONFLICT DO NOTHING;

-- Locks a user's progression rows for one transaction (quest assignment races).
-- name: LockUser :exec
SELECT pg_advisory_xact_lock(hashtextextended('progression:' || @user_id::text, 0));

-- ---------- XP ----------

-- name: InsertXP :execrows
INSERT INTO progression.xp_ledger (id, user_id, amount, source, source_ref, created_at)
VALUES (@id, @user_id, @amount, @source, @source_ref, @now)
ON CONFLICT DO NOTHING;

-- name: GetProgress :one
SELECT * FROM progression.user_progress WHERE user_id = @user_id;

-- name: UpsertProgress :exec
INSERT INTO progression.user_progress (user_id, xp_total, level, updated_at) VALUES (@user_id, @xp_total, @level, @now)
ON CONFLICT (user_id) DO UPDATE SET xp_total = EXCLUDED.xp_total, level = EXCLUDED.level, updated_at = EXCLUDED.updated_at;

-- name: MatchXP :one
SELECT coalesce(sum(amount), 0)::int FROM progression.xp_ledger
WHERE user_id = @user_id AND ((source = 'match' AND source_ref = @match_ref::text) OR (source = 'badge' AND source_ref LIKE @match_ref::text || ':%'));

-- ---------- Streak & counters ----------

-- name: GetStreak :one
SELECT * FROM progression.play_streaks WHERE user_id = @user_id;

-- name: UpsertStreak :exec
INSERT INTO progression.play_streaks (user_id, current_days, longest_days, last_play_day, updated_at)
VALUES (@user_id, @current_days, @longest_days, @last_play_day, @now)
ON CONFLICT (user_id) DO UPDATE SET current_days = EXCLUDED.current_days, longest_days = EXCLUDED.longest_days,
  last_play_day = EXCLUDED.last_play_day, updated_at = EXCLUDED.updated_at;

-- name: AddStats :one
INSERT INTO progression.user_stats (user_id, matches, wins, correct, updated_at) VALUES (@user_id, 1, @wins, @correct, @now)
ON CONFLICT (user_id) DO UPDATE SET matches = progression.user_stats.matches + 1, wins = progression.user_stats.wins + EXCLUDED.wins,
  correct = progression.user_stats.correct + EXCLUDED.correct, updated_at = EXCLUDED.updated_at
RETURNING wins;

-- name: AddCategoryCorrect :one
INSERT INTO progression.category_correct (user_id, category_id, correct) VALUES (@user_id, @category_id, @correct)
ON CONFLICT (user_id, category_id) DO UPDATE SET correct = progression.category_correct.correct + EXCLUDED.correct
RETURNING correct;

-- ---------- Badges ----------

-- name: ActiveBadges :many
SELECT id, code, family, tier, criteria, icon_key FROM progression.badges WHERE is_active ORDER BY family, tier, code;

-- name: AwardBadge :execrows
INSERT INTO progression.user_badges (user_id, badge_id, earned_at, source_ref) VALUES (@user_id, @badge_id, @now, @source_ref)
ON CONFLICT DO NOTHING;

-- name: BadgeCatalog :many
SELECT b.id, b.code, b.family, b.tier, b.icon_key, b.is_hidden, t.name, t.description, ub.earned_at
FROM progression.badges b
JOIN progression.badge_translations t ON t.badge_id = b.id AND t.locale = @locale
LEFT JOIN progression.user_badges ub ON ub.badge_id = b.id AND ub.user_id = @user_id
WHERE b.is_active
ORDER BY b.family, b.tier, b.code;

-- name: BadgesBySource :many
SELECT b.code, b.tier, b.icon_key, t.name
FROM progression.user_badges ub
JOIN progression.badges b ON b.id = ub.badge_id
JOIN progression.badge_translations t ON t.badge_id = b.id AND t.locale = @locale
WHERE ub.user_id = @user_id AND ub.source_ref = @source_ref
ORDER BY b.family, b.tier;

-- ---------- Quests ----------

-- name: ActiveQuests :many
SELECT id, code, objective, target, filters, reward_xp FROM progression.quests
WHERE is_active AND cadence = @cadence AND NOT requires_full_account;

-- name: InsertAssignment :exec
INSERT INTO progression.quest_assignments (id, user_id, quest_id, period_start, period_end)
VALUES (@id, @user_id, @quest_id, @period_start, @period_end)
ON CONFLICT DO NOTHING;

-- Assignments of a period, with their quests.
-- name: PeriodAssignments :many
SELECT a.id, a.quest_id, a.period_start, a.period_end, a.progress, a.completed_at, a.claimed_at,
       q.code, q.cadence, q.objective, q.target, q.filters, q.reward_xp
FROM progression.quest_assignments a JOIN progression.quests q ON q.id = a.quest_id
WHERE a.user_id = @user_id AND q.cadence = @cadence AND a.period_start = @period_start
ORDER BY q.code;

-- name: SetAssignmentProgress :exec
UPDATE progression.quest_assignments SET progress = @progress, completed_at = sqlc.narg(completed_at) WHERE id = @id;

-- name: GetAssignmentForUpdate :one
SELECT a.*, q.cadence, q.objective, q.reward_xp FROM progression.quest_assignments a JOIN progression.quests q ON q.id = a.quest_id
WHERE a.id = @id AND a.user_id = @user_id FOR UPDATE OF a;

-- name: ClaimAssignment :exec
UPDATE progression.quest_assignments SET claimed_at = @now WHERE id = @id;

-- name: DeleteAssignment :exec
DELETE FROM progression.quest_assignments WHERE id = @id;

-- name: QuestTexts :many
SELECT quest_id, title, description FROM progression.quest_translations WHERE locale = @locale;

-- name: UseReroll :one
INSERT INTO progression.quest_rerolls (user_id, day, count) VALUES (@user_id, @day, 1)
ON CONFLICT (user_id, day) DO UPDATE SET count = progression.quest_rerolls.count + 1
RETURNING count;

-- name: RerollCount :one
SELECT coalesce((SELECT count FROM progression.quest_rerolls WHERE user_id = @user_id AND day = @day), 0)::int;

-- name: GetStats :one
SELECT matches, wins FROM progression.user_stats WHERE user_id = @user_id;
