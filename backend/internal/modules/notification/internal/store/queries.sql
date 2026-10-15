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
