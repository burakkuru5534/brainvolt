-- ---------- Invite codes ----------

-- name: InsertCode :exec
INSERT INTO lobby.invite_codes (code, target_kind, target_id, created_by, max_uses, expires_at, created_at)
VALUES (@code, @target_kind, @target_id, @created_by, sqlc.narg(max_uses), @expires_at, @now);

-- name: GetCode :one
SELECT * FROM lobby.invite_codes WHERE code = @code;

-- The newest usable code of a target.
-- name: ActiveCode :one
SELECT * FROM lobby.invite_codes
WHERE target_kind = @target_kind AND target_id = @target_id AND revoked_at IS NULL AND expires_at > @now
  AND (max_uses IS NULL OR use_count < max_uses)
ORDER BY created_at DESC
LIMIT 1;

-- name: RevokeCodes :exec
UPDATE lobby.invite_codes SET revoked_at = @now::timestamptz
WHERE target_kind = @target_kind AND target_id = @target_id AND revoked_at IS NULL;

-- name: UseCode :exec
UPDATE lobby.invite_codes SET use_count = use_count + 1 WHERE code = @code;

-- Codes expired for 30 days are deleted so they can be reused.
-- name: DeleteOldCodes :exec
DELETE FROM lobby.invite_codes WHERE expires_at < @before;

-- ---------- Rooms ----------

-- name: InsertRoom :exec
INSERT INTO lobby.rooms (id, host_user_id, match_config, created_at, last_activity_at)
VALUES (@id, @host_user_id, @match_config, @now, @now);

-- name: GetRoom :one
SELECT * FROM lobby.rooms WHERE id = @id;

-- name: UpdateRoom :exec
UPDATE lobby.rooms
SET host_user_id = @host_user_id, status = @status, match_config = @match_config,
    current_match_id = sqlc.narg(current_match_id), matches_played = @matches_played, last_activity_at = @now
WHERE id = @id AND status <> 'closed';

-- name: CloseRoom :execrows
UPDATE lobby.rooms SET status = 'closed', closed_at = @now::timestamptz, close_reason = @reason::text, last_activity_at = @now::timestamptz
WHERE id = @id AND status <> 'closed';

-- Rooms without activity since @before, for the idle sweep.
-- name: IdleRooms :many
SELECT id, status, current_match_id FROM lobby.rooms
WHERE status <> 'closed' AND last_activity_at < @before
ORDER BY last_activity_at
LIMIT 100;

-- ---------- Challenges ----------

-- name: InsertChallenge :exec
INSERT INTO lobby.challenges (id, kind, challenger_user_id, target_user_id, match_config, parent_match_id, created_at, expires_at)
VALUES (@id, @kind, @challenger_user_id, @target_user_id, @match_config, sqlc.narg(parent_match_id), @now, @expires_at);

-- name: GetChallenge :one
SELECT * FROM lobby.challenges WHERE id = @id;

-- name: GetChallengeForUpdate :one
SELECT * FROM lobby.challenges WHERE id = @id FOR UPDATE;

-- name: SetChallengeStatus :exec
UPDATE lobby.challenges
SET status = @status, responded_at = COALESCE(responded_at, sqlc.narg(responded_at)), match_id = COALESCE(sqlc.narg(match_id), match_id)
WHERE id = @id;

-- A pending, unexpired direct challenge between two users (either way).
-- name: PendingChallengeBetween :one
SELECT EXISTS (
  SELECT 1 FROM lobby.challenges
  WHERE status = 'pending' AND kind = 'direct' AND expires_at > @now
    AND ((challenger_user_id = @a::uuid AND target_user_id = @b::uuid) OR (challenger_user_id = @b::uuid AND target_user_id = @a::uuid))
);

-- name: ChallengesSentSince :one
SELECT count(*) FROM lobby.challenges WHERE challenger_user_id = @user_id AND created_at >= @since;

-- name: ListIncomingChallenges :many
SELECT * FROM lobby.challenges
WHERE target_user_id = @user_id::uuid AND status = 'pending' AND expires_at > @now
ORDER BY created_at DESC
LIMIT 100;

-- name: ListOutgoingChallenges :many
SELECT * FROM lobby.challenges
WHERE challenger_user_id = @user_id AND status = 'pending' AND expires_at > @now
ORDER BY created_at DESC
LIMIT 100;

-- A started challenge whose match was cancelled before it began.
-- name: FailChallengeOfMatch :execrows
UPDATE lobby.challenges SET status = 'failed' WHERE match_id = @match_id::uuid AND status = 'started';
