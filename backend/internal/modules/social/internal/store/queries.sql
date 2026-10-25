-- social queries (schema-social.md §2). Only the social schema.

-- name: InsertRequest :exec
INSERT INTO social.friend_requests (id, from_user_id, to_user_id, via, status, created_at, expires_at)
VALUES (@id, @from_user_id, @to_user_id, @via, 'pending', @now, @expires_at);

-- name: PendingBetween :one
SELECT * FROM social.friend_requests
WHERE status = 'pending' AND expires_at > @now
  AND ((from_user_id = @a AND to_user_id = @b) OR (from_user_id = @b AND to_user_id = @a))
FOR UPDATE;

-- name: PendingDirection :one
-- Who asked whom, without a lock (player profiles, D-101).
SELECT from_user_id FROM social.friend_requests
WHERE status = 'pending' AND expires_at > @now
  AND ((from_user_id = @a AND to_user_id = @b) OR (from_user_id = @b AND to_user_id = @a))
ORDER BY created_at DESC
LIMIT 1;

-- name: GetRequestForUpdate :one
SELECT * FROM social.friend_requests WHERE id = $1 FOR UPDATE;

-- name: SetRequestStatus :exec
UPDATE social.friend_requests SET status = @status, responded_at = @now WHERE id = @id;

-- Pending requests past their expiry are marked expired before a new one
-- is inserted for the pair (the pending-pair unique index).
-- name: ExpireBetween :exec
UPDATE social.friend_requests SET status = 'expired', responded_at = @now
WHERE status = 'pending' AND expires_at <= @now
  AND ((from_user_id = @a AND to_user_id = @b) OR (from_user_id = @b AND to_user_id = @a));

-- name: CancelPendingBetween :exec
UPDATE social.friend_requests SET status = 'cancelled', responded_at = @now
WHERE status = 'pending'
  AND ((from_user_id = @a AND to_user_id = @b) OR (from_user_id = @b AND to_user_id = @a));

-- name: SentSince :one
SELECT count(*) FROM social.friend_requests WHERE from_user_id = @user_id AND created_at > @since;

-- name: ListInbox :many
SELECT * FROM social.friend_requests WHERE to_user_id = @user_id AND status = 'pending' AND expires_at > @now
ORDER BY created_at DESC LIMIT 200;

-- name: ListOutbox :many
SELECT * FROM social.friend_requests WHERE from_user_id = @user_id AND status = 'pending' AND expires_at > @now
ORDER BY created_at DESC LIMIT 200;

-- name: InsertFriendship :exec
INSERT INTO social.friendships (user_id, friend_id, created_at) VALUES (@user_id, @friend_id, @now), (@friend_id, @user_id, @now)
ON CONFLICT DO NOTHING;

-- name: DeleteFriendship :execrows
DELETE FROM social.friendships WHERE (user_id = @a AND friend_id = @b) OR (user_id = @b AND friend_id = @a);

-- name: AreFriends :one
SELECT EXISTS (SELECT 1 FROM social.friendships WHERE user_id = @a AND friend_id = @b);

-- name: FriendCount :one
SELECT count(*) FROM social.friendships WHERE user_id = $1;

-- name: FriendIDs :many
SELECT friend_id FROM social.friendships WHERE user_id = $1 ORDER BY friend_id;

-- name: ListFriends :many
SELECT friend_id, created_at FROM social.friendships WHERE user_id = @user_id AND friend_id > @after
ORDER BY friend_id LIMIT @row_limit;

-- name: InsertBlock :exec
INSERT INTO social.blocks (blocker_id, blocked_id, created_at) VALUES (@blocker_id, @blocked_id, @now) ON CONFLICT DO NOTHING;

-- name: DeleteBlock :execrows
DELETE FROM social.blocks WHERE blocker_id = @blocker_id AND blocked_id = @blocked_id;

-- name: IsBlocked :one
SELECT EXISTS (SELECT 1 FROM social.blocks WHERE (blocker_id = @a AND blocked_id = @b) OR (blocker_id = @b AND blocked_id = @a));

-- name: BlockedByOrBlocking :many
SELECT (CASE WHEN b.blocker_id = @user_id::uuid THEN b.blocked_id ELSE b.blocker_id END)::uuid AS other
FROM social.blocks b WHERE b.blocker_id = @user_id::uuid OR b.blocked_id = @user_id::uuid;

-- name: ListBlocks :many
SELECT blocked_id, created_at FROM social.blocks WHERE blocker_id = $1 ORDER BY created_at DESC;
