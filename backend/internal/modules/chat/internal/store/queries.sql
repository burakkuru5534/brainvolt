-- chat schema queries (schema-platform §2). Only the chat schema.

-- name: TeamChannel :one
INSERT INTO chat.channels (id, kind, team_id, created_at) VALUES (@id, 'team', @team_id, @now)
ON CONFLICT (team_id) WHERE kind = 'team' DO UPDATE SET team_id = EXCLUDED.team_id
RETURNING *;

-- name: CoupleChannel :one
INSERT INTO chat.channels (id, kind, couple_id, created_at) VALUES (@id, 'couple', @couple_id, @now)
ON CONFLICT (couple_id) WHERE kind = 'couple' DO UPDATE SET couple_id = EXCLUDED.couple_id
RETURNING *;

-- name: GetChannel :one
SELECT * FROM chat.channels WHERE id = @id;

-- name: InsertMessage :exec
INSERT INTO chat.messages (id, channel_id, sender_id, kind, body, quick_id, filter_result, created_at)
VALUES (@id, @channel_id, @sender_id, @kind, sqlc.narg(body), sqlc.narg(quick_id), @filter_result, @now);

-- name: SentSince :one
SELECT count(*)::int FROM chat.messages WHERE sender_id = @sender_id AND created_at >= @since;

-- Newest first, before @before (a message id) when given.
-- name: History :many
SELECT * FROM chat.messages
WHERE channel_id = @channel_id AND (sqlc.narg(before)::uuid IS NULL OR id < sqlc.narg(before)::uuid)
ORDER BY id DESC
LIMIT @lim;

-- name: GetMessage :one
SELECT * FROM chat.messages WHERE id = @id;

-- name: RemoveMessage :execrows
UPDATE chat.messages SET removed_at = @now, removed_by = @removed_by, body = NULL
WHERE id = @id AND removed_at IS NULL;

-- name: UpsertReadCursor :exec
INSERT INTO chat.read_cursors (channel_id, user_id, last_read_id) VALUES (@channel_id, @user_id, @last_read_id)
ON CONFLICT (channel_id, user_id) DO UPDATE SET last_read_id = GREATEST(chat.read_cursors.last_read_id, EXCLUDED.last_read_id);

-- name: Unread :one
SELECT count(*)::int FROM chat.messages m
WHERE m.channel_id = @channel_id AND m.sender_id <> @user_id AND m.removed_at IS NULL
  AND m.id > coalesce((SELECT last_read_id FROM chat.read_cursors WHERE channel_id = @channel_id AND user_id = @user_id), '00000000-0000-0000-0000-000000000000');

-- name: PurgeMessages :execrows
DELETE FROM chat.messages WHERE created_at < @before;
