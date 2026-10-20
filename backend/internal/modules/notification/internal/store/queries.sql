-- notification.devices queries (schema-identity.md §4). Only the notification schema.

-- name: UpsertDevice :exec
-- A device belongs to one user at a time: signing in as another user on the
-- same installation re-assigns the row (D-015) and un-revokes it.
INSERT INTO notification.devices (id, user_id, installation_id, platform, fcm_token, app_version, locale, last_seen_at, created_at)
VALUES (@id, @user_id, @installation_id, @platform, @fcm_token, @app_version, @locale, @now, @now)
ON CONFLICT (installation_id) DO UPDATE
SET user_id      = EXCLUDED.user_id,
    platform     = EXCLUDED.platform,
    fcm_token    = EXCLUDED.fcm_token,
    app_version  = EXCLUDED.app_version,
    locale       = EXCLUDED.locale,
    last_seen_at = EXCLUDED.last_seen_at,
    revoked_at   = NULL;

-- name: RevokeDevice :execrows
UPDATE notification.devices SET revoked_at = @now::timestamptz, fcm_token = NULL
WHERE user_id = @user_id AND installation_id = @installation_id AND revoked_at IS NULL;

-- name: ListActiveDevices :many
SELECT * FROM notification.devices WHERE user_id = $1 AND revoked_at IS NULL ORDER BY last_seen_at DESC;

-- name: ClearFcmTokenElsewhere :exec
-- An FCM token is unique: after a reinstall the same token may arrive with a
-- new installation id, so it is removed from any other row first.
UPDATE notification.devices SET fcm_token = NULL
WHERE fcm_token = @fcm_token AND installation_id <> @installation_id;

-- ---------- Inbox (schema-platform §1) ----------

-- name: InsertInbox :one
-- A repeat with the same unread dedupe key refreshes the existing item.
INSERT INTO notification.inbox (id, user_id, category, kind, params, deep_link, dedupe_key, created_at, expires_at)
VALUES (@id, @user_id, @category, @kind, @params, sqlc.narg(deep_link), sqlc.narg(dedupe_key), @now, @expires_at)
ON CONFLICT (user_id, dedupe_key) WHERE dedupe_key IS NOT NULL AND read_at IS NULL
DO UPDATE SET params = EXCLUDED.params, deep_link = EXCLUDED.deep_link, created_at = EXCLUDED.created_at, expires_at = EXCLUDED.expires_at
RETURNING id, (xmax = 0)::boolean AS inserted;

-- name: Inbox :many
SELECT * FROM notification.inbox
WHERE user_id = @user_id AND expires_at > @now
  AND (sqlc.narg(before)::timestamptz IS NULL OR created_at < sqlc.narg(before)::timestamptz)
ORDER BY created_at DESC
LIMIT @lim;

-- name: UnreadCount :one
SELECT count(*)::int FROM notification.inbox WHERE user_id = @user_id AND read_at IS NULL AND expires_at > @now;

-- name: MarkRead :exec
UPDATE notification.inbox SET read_at = @now
WHERE user_id = @user_id AND read_at IS NULL AND (@every::boolean OR id = ANY(@ids::uuid[]));

-- name: PurgeInbox :execrows
DELETE FROM notification.inbox WHERE expires_at < @now;

-- name: PurgeDeliveries :execrows
DELETE FROM notification.push_deliveries WHERE attempted_at < @before;

-- ---------- Preferences ----------

-- name: Preferences :many
SELECT category, push, in_app FROM notification.preferences WHERE user_id = @user_id;

-- name: UpsertPreference :exec
INSERT INTO notification.preferences (user_id, category, push, in_app, updated_at)
VALUES (@user_id, @category, @push, @in_app, @now)
ON CONFLICT (user_id, category) DO UPDATE SET push = EXCLUDED.push, in_app = EXCLUDED.in_app, updated_at = EXCLUDED.updated_at;

-- name: QuietHours :one
SELECT * FROM notification.quiet_hours WHERE user_id = @user_id;

-- name: UpsertQuietHours :exec
INSERT INTO notification.quiet_hours (user_id, enabled, start_local, end_local, utc_offset_minutes)
VALUES (@user_id, @enabled, @start_local, @end_local, @utc_offset_minutes)
ON CONFLICT (user_id) DO UPDATE SET enabled = EXCLUDED.enabled, start_local = EXCLUDED.start_local,
  end_local = EXCLUDED.end_local, utc_offset_minutes = EXCLUDED.utc_offset_minutes;

-- ---------- Push ----------

-- name: PushTargets :many
SELECT id, fcm_token, locale FROM notification.devices
WHERE user_id = @user_id AND revoked_at IS NULL AND fcm_token IS NOT NULL;

-- Pushes sent to a user's devices since @since (the hourly cap).
-- name: PushesSince :one
SELECT count(DISTINCT d.inbox_id)::int FROM notification.push_deliveries d
JOIN notification.devices v ON v.id = d.device_id
WHERE v.user_id = @user_id AND d.status = 'sent' AND d.attempted_at >= @since;

-- name: InsertDelivery :exec
INSERT INTO notification.push_deliveries (id, inbox_id, device_id, status, fcm_message_id, error, attempted_at)
VALUES (@id, @inbox_id, @device_id, @status, sqlc.narg(fcm_message_id), sqlc.narg(error), @now);

-- name: RevokeToken :exec
UPDATE notification.devices SET fcm_token = NULL WHERE id = @id;
