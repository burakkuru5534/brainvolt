-- identity queries (schema-identity.md §2, §5). Only the identity schema.

-- name: GetUserByFirebaseUID :one
SELECT * FROM identity.users WHERE firebase_uid = $1;

-- name: GetUserByID :one
SELECT * FROM identity.users WHERE id = $1;

-- name: GetUserByFirebaseUIDForUpdate :one
-- Serialises concurrent session syncs of the same user.
SELECT * FROM identity.users WHERE firebase_uid = $1 FOR UPDATE;

-- name: InsertUser :execrows
-- 0 rows: a concurrent first session created the user already.
INSERT INTO identity.users (id, firebase_uid, account_tier, age_verified_at, last_seen_at, created_at, updated_at)
VALUES (@id, @firebase_uid, @account_tier, @now, @now, @now, @now)
ON CONFLICT (firebase_uid) DO NOTHING;

-- name: UpdateUserTier :exec
-- upgraded_at is set the first time the tier becomes 'full' and never cleared.
UPDATE identity.users
SET account_tier = @account_tier,
    upgraded_at  = CASE WHEN @account_tier = 'full' THEN COALESCE(upgraded_at, @now) ELSE upgraded_at END,
    updated_at   = @now
WHERE id = @id;

-- name: TouchUser :exec
UPDATE identity.users SET last_seen_at = @now WHERE id = @id;

-- name: MarkGuestAbandoned :execrows
UPDATE identity.users SET abandoned_at = @now::timestamptz, updated_at = @now::timestamptz
WHERE id = @id AND account_tier = 'guest' AND abandoned_at IS NULL;

-- name: SetDeletionRequested :exec
-- NULL cancels a pending deletion (14-day cooling-off, D-022).
UPDATE identity.users SET deletion_requested_at = @deletion_requested_at, updated_at = @now WHERE id = @id;

-- name: ListAuthIdentities :many
SELECT * FROM identity.auth_identities WHERE user_id = $1 ORDER BY provider;

-- name: UpsertAuthIdentity :exec
INSERT INTO identity.auth_identities (user_id, provider, provider_uid, email, email_verified, linked_at, last_sign_in_at)
VALUES (@user_id, @provider, @provider_uid, @email, @email_verified, @now, @now)
ON CONFLICT (user_id, provider) DO UPDATE
SET provider_uid    = EXCLUDED.provider_uid,
    email           = EXCLUDED.email,
    email_verified  = EXCLUDED.email_verified,
    last_sign_in_at = EXCLUDED.last_sign_in_at;

-- Firebase allows one owner per provider identity; a row held by another
-- user is stale (that Firebase account was deleted) and is removed.
-- name: ReleaseProviderUID :many
DELETE FROM identity.auth_identities
WHERE provider = @provider AND provider_uid = @provider_uid AND user_id <> @user_id
RETURNING user_id;

-- name: ReleasePasswordEmail :many
DELETE FROM identity.auth_identities
WHERE provider = 'password' AND email = @email AND user_id <> @user_id
RETURNING user_id;

-- name: DeleteAuthIdentitiesExcept :exec
-- Drops sign-in methods no longer linked in Firebase (the mirror follows Firebase).
DELETE FROM identity.auth_identities WHERE user_id = @user_id AND NOT (provider = ANY(@providers::text[]));

-- name: ListUserRoles :many
SELECT role FROM identity.user_roles WHERE user_id = $1 ORDER BY role;

-- name: GrantRole :exec
INSERT INTO identity.user_roles (user_id, role, granted_by, granted_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, role) DO NOTHING;

-- name: RevokeRole :exec
DELETE FROM identity.user_roles WHERE user_id = $1 AND role = $2;
