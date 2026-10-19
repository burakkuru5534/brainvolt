-- profile queries (schema-identity.md §3). Only the profile schema.

-- name: InsertProfile :execrows
-- 0 rows: the generated username is taken; the caller retries with another.
INSERT INTO profile.profiles (user_id, username, display_name, avatar_id, country_code, locale, created_at, updated_at)
VALUES (@user_id, @username, @display_name, @avatar_id, @country_code, @locale, @now, @now)
ON CONFLICT (username) DO NOTHING;

-- name: InsertSettings :exec
INSERT INTO profile.settings (user_id, updated_at) VALUES (@user_id, @now);

-- name: GetProfile :one
SELECT * FROM profile.profiles WHERE user_id = $1;

-- name: GetSettings :one
SELECT * FROM profile.settings WHERE user_id = $1;

-- name: UsernameTaken :one
SELECT EXISTS (SELECT 1 FROM profile.profiles WHERE username = $1);

-- name: GetProfileForUpdate :one
SELECT * FROM profile.profiles WHERE user_id = $1 FOR UPDATE;

-- name: UpdateProfile :exec
UPDATE profile.profiles
SET username = @username, display_name = @display_name, avatar_id = @avatar_id,
    country_code = @country_code, locale = @locale,
    username_changed_at = @username_changed_at, country_changed_at = @country_changed_at,
    updated_at = @now
WHERE user_id = @user_id;

-- name: UpdateSettings :exec
UPDATE profile.settings
SET challenges_from = @challenges_from, show_online_status = @show_online_status,
    allow_friend_requests = @allow_friend_requests, sound_enabled = @sound_enabled,
    music_enabled = @music_enabled, haptics_enabled = @haptics_enabled, updated_at = @now
WHERE user_id = @user_id;

-- name: PublicCards :many
SELECT user_id, username, display_name, avatar_id, country_code FROM profile.profiles WHERE user_id = ANY(@ids::uuid[]);

-- name: UserIDByUsername :one
SELECT user_id FROM profile.profiles WHERE username = $1;

-- name: PrivacyOf :many
SELECT user_id, challenges_from, show_online_status, allow_friend_requests FROM profile.settings WHERE user_id = ANY(@ids::uuid[]);

-- name: LocalesOf :many
SELECT user_id, locale FROM profile.profiles WHERE user_id = ANY(@ids::uuid[]);
