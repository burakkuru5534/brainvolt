-- profile queries (schema-identity.md §3). Only the profile schema.

-- name: InsertProfile :exec
INSERT INTO profile.profiles (user_id, username, display_name, avatar_id, country_code, locale, created_at, updated_at)
VALUES (@user_id, @username, @display_name, @avatar_id, @country_code, @locale, @now, @now);

-- name: InsertSettings :exec
INSERT INTO profile.settings (user_id, updated_at) VALUES (@user_id, @now);

-- name: GetProfile :one
SELECT * FROM profile.profiles WHERE user_id = $1;

-- name: GetSettings :one
SELECT * FROM profile.settings WHERE user_id = $1;

-- name: UsernameTaken :one
SELECT EXISTS (SELECT 1 FROM profile.profiles WHERE username = $1);
