-- ---------- Cases & reports ----------

-- name: OpenCaseForUpdate :one
SELECT * FROM moderation.cases
WHERE target_kind = @target_kind AND target_id = @target_id AND coalesce(target_locale, '') = coalesce(sqlc.narg(target_locale)::text, '')
  AND status <> 'resolved'
FOR UPDATE;

-- name: InsertCase :exec
INSERT INTO moderation.cases (id, target_kind, target_id, target_locale, opened_at)
VALUES (@id, @target_kind, @target_id, sqlc.narg(target_locale), @now);

-- name: InsertReport :execrows
INSERT INTO moderation.reports (id, case_id, reporter_id, reporter_tier, target_kind, target_id, target_locale, match_id, reason, comment, created_at)
VALUES (@id, @case_id, @reporter_id, @reporter_tier, @target_kind, @target_id, sqlc.narg(target_locale), sqlc.narg(match_id), @reason, sqlc.narg(comment), @now)
ON CONFLICT DO NOTHING;

-- name: BumpCase :exec
UPDATE moderation.cases SET report_count = report_count + 1, priority = priority + @weight WHERE id = @id;

-- name: ReportsSince :one
SELECT count(*)::int FROM moderation.reports WHERE reporter_id = @reporter_id AND created_at >= @since;

-- name: CaseReasonCount :one
SELECT count(*)::int FROM moderation.reports WHERE case_id = @case_id AND reason = @reason;

-- Review queue: highest priority first.
-- name: Cases :many
SELECT * FROM moderation.cases
WHERE (@resolved::boolean AND status = 'resolved') OR (NOT @resolved::boolean AND status <> 'resolved')
ORDER BY priority DESC, opened_at
LIMIT @lim OFFSET @off;

-- name: GetCase :one
SELECT * FROM moderation.cases WHERE id = @id;

-- name: GetCaseForUpdate :one
SELECT * FROM moderation.cases WHERE id = @id FOR UPDATE;

-- name: CaseReports :many
SELECT * FROM moderation.reports WHERE case_id = @case_id ORDER BY created_at;

-- name: CaseSignals :many
SELECT * FROM moderation.cheat_signals WHERE case_id = @case_id ORDER BY detected_at;

-- name: TargetActions :many
SELECT * FROM moderation.actions WHERE target_kind = @target_kind AND target_id = @target_id ORDER BY created_at;

-- name: SetCaseReview :exec
UPDATE moderation.cases SET status = 'in_review', assigned_to = @assigned_to WHERE id = @id AND status = 'open';

-- name: ResolveCase :exec
UPDATE moderation.cases SET status = 'resolved', resolution = @resolution, resolution_note = sqlc.narg(note), resolved_at = @now
WHERE id = @id;

-- name: InsertAction :exec
INSERT INTO moderation.actions (id, case_id, actor_kind, actor_id, action, target_kind, target_id, payload, created_at)
VALUES (@id, sqlc.narg(case_id), @actor_kind, sqlc.narg(actor_id), @action, @target_kind, @target_id, @payload, @now);

-- ---------- Sanctions ----------

-- name: InsertSanction :exec
INSERT INTO moderation.sanctions (id, user_id, kind, reason_code, case_id, starts_at, ends_at, created_by, created_at)
VALUES (@id, @user_id, @kind, @reason_code, sqlc.narg(case_id), @now, sqlc.narg(ends_at), sqlc.narg(created_by), @now);

-- name: UserSanctions :many
SELECT s.*, a.id AS appeal_id, a.status AS appeal_status
FROM moderation.sanctions s LEFT JOIN moderation.appeals a ON a.sanction_id = s.id
WHERE s.user_id = @user_id ORDER BY s.created_at DESC LIMIT 50;

-- name: ActiveSanctions :many
SELECT * FROM moderation.sanctions
WHERE user_id = @user_id AND lifted_at IS NULL AND kind <> 'warning' AND starts_at <= @now AND (ends_at IS NULL OR ends_at > @now);

-- name: GetSanctionForUpdate :one
SELECT * FROM moderation.sanctions WHERE id = @id FOR UPDATE;

-- name: LiftSanction :exec
UPDATE moderation.sanctions SET lifted_at = @now, lifted_by = sqlc.narg(lifted_by) WHERE id = @id AND lifted_at IS NULL;

-- name: OtherActiveSuspension :one
SELECT EXISTS (
  SELECT 1 FROM moderation.sanctions
  WHERE user_id = @user_id AND id <> @id AND kind = 'suspension' AND lifted_at IS NULL AND (ends_at IS NULL OR ends_at > @now)
);

-- Ended timed suspensions still holding an account suspended.
-- name: EndedSuspensions :many
SELECT * FROM moderation.sanctions
WHERE kind = 'suspension' AND lifted_at IS NULL AND ends_at IS NOT NULL AND ends_at <= @now
LIMIT 200;

-- ---------- Appeals ----------

-- name: InsertAppeal :execrows
INSERT INTO moderation.appeals (id, sanction_id, user_id, text, created_at)
VALUES (@id, @sanction_id, @user_id, @text, @now)
ON CONFLICT DO NOTHING;

-- name: PendingAppeals :many
SELECT a.*, s.kind AS sanction_kind, s.reason_code, s.created_by AS sanction_created_by
FROM moderation.appeals a JOIN moderation.sanctions s ON s.id = a.sanction_id
WHERE a.status = 'pending' ORDER BY a.created_at LIMIT 100;

-- name: GetAppealForUpdate :one
SELECT a.*, s.created_by AS sanction_created_by FROM moderation.appeals a JOIN moderation.sanctions s ON s.id = a.sanction_id
WHERE a.id = @id FOR UPDATE OF a;

-- name: DecideAppeal :exec
UPDATE moderation.appeals SET status = @status, decided_by = @decided_by, decided_at = @now, decision_note = sqlc.narg(note) WHERE id = @id;
