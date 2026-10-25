-- commerce queries (schema-platform.md §4, D-103). Only the commerce schema.

-- name: HasEntitlement :one
SELECT EXISTS (
  SELECT 1 FROM commerce.entitlements
  WHERE user_id = @user_id AND code = ANY(@codes::text[]) AND starts_at <= @now AND (ends_at IS NULL OR ends_at > @now)
);

-- name: InsertAdReward :exec
INSERT INTO commerce.ad_rewards (id, user_id, placement, ref, status, created_at, expires_at)
VALUES (@id, @user_id, @placement, @ref, 'pending', @now, @expires_at);

-- name: OpenAdReward :one
-- A pending reward of the same placement and match, reused instead of a new one.
SELECT * FROM commerce.ad_rewards
WHERE user_id = @user_id AND placement = @placement AND ref = @ref AND status = 'pending' AND expires_at > @now
ORDER BY created_at DESC LIMIT 1;

-- name: CountAdRewards :one
-- Rewards that count against a limit: granted ones and pending ones not expired.
SELECT count(*)::int FROM commerce.ad_rewards
WHERE user_id = @user_id AND (@placement::text = '' OR placement = @placement) AND created_at >= @since
  AND (status = 'granted' OR expires_at > @now);

-- name: GetAdReward :one
SELECT * FROM commerce.ad_rewards WHERE id = @id AND user_id = @user_id;

-- name: GetAdRewardForUpdate :one
SELECT * FROM commerce.ad_rewards WHERE id = @id FOR UPDATE;

-- name: GrantAdReward :exec
UPDATE commerce.ad_rewards SET status = 'granted', transaction_id = @transaction_id, granted_at = @now
WHERE id = @id;
