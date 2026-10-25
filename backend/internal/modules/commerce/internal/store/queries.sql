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
-- An empty placement counts every placement but the store's Volt ad (D-105).
SELECT count(*)::int FROM commerce.ad_rewards
WHERE user_id = @user_id AND ((@placement::text = '' AND placement <> 'volt') OR placement = @placement) AND created_at >= @since
  AND (status = 'granted' OR expires_at > @now);

-- name: GetAdReward :one
SELECT * FROM commerce.ad_rewards WHERE id = @id AND user_id = @user_id;

-- name: GetAdRewardForUpdate :one
SELECT * FROM commerce.ad_rewards WHERE id = @id FOR UPDATE;

-- name: GrantAdReward :exec
UPDATE commerce.ad_rewards SET status = 'granted', transaction_id = @transaction_id, granted_at = @now
WHERE id = @id;

-- ---------- Store events and entitlements (D-104) ----------

-- name: InsertStoreEvent :execrows
INSERT INTO commerce.store_events (id, store, event_id, kind, user_id, payload, received_at)
VALUES (@id, @store, @event_id, @kind, @user_id, @payload, @now)
ON CONFLICT (store, event_id) DO NOTHING;

-- name: MarkStoreEventProcessed :exec
UPDATE commerce.store_events SET processed_at = @now WHERE store = @store AND event_id = @event_id;

-- name: ExtendEntitlement :exec
-- Grants or extends an entitlement; a late, older event never shortens it.
-- trial follows the latest event (a trial converts to a paid period).
INSERT INTO commerce.entitlements (user_id, code, source_kind, source_id, starts_at, ends_at, trial)
VALUES (@user_id, @code, @source_kind, @source_id, @starts_at, @ends_at, @trial)
ON CONFLICT (user_id, code, source_kind, source_id) DO UPDATE
SET trial = EXCLUDED.trial,
    ends_at = CASE WHEN commerce.entitlements.ends_at IS NULL OR EXCLUDED.ends_at IS NULL THEN NULL
                   ELSE greatest(commerce.entitlements.ends_at, EXCLUDED.ends_at) END;

-- name: EndEntitlement :exec
-- Ends an entitlement at ends_at (expiry, refund, transfer away).
UPDATE commerce.entitlements SET ends_at = @ends_at
WHERE user_id = @user_id AND code = @code AND source_kind = @source_kind AND source_id = @source_id
  AND (ends_at IS NULL OR ends_at > @ends_at);

-- name: SubscriptionEntitlements :many
SELECT * FROM commerce.entitlements WHERE user_id = @user_id AND source_kind = 'subscription';

-- name: ActiveEntitlements :many
SELECT code, coalesce(max(ends_at), '9999-12-31T00:00:00Z'::timestamptz)::timestamptz AS ends_at, bool_or(ends_at IS NULL) AS forever
FROM commerce.entitlements
WHERE user_id = @user_id AND starts_at <= @now AND (ends_at IS NULL OR ends_at > @now)
GROUP BY code ORDER BY code;

-- name: PaidEntitlement :one
-- An active entitlement outside a free trial (Plus's monthly Volt, D-105).
SELECT EXISTS (
  SELECT 1 FROM commerce.entitlements
  WHERE user_id = @user_id AND code = @code AND NOT trial AND starts_at <= @now AND (ends_at IS NULL OR ends_at > @now)
);

-- ---------- Volt (D-105) ----------

-- name: LockWallet :one
-- Creates the wallet when missing and locks it for the transaction.
INSERT INTO commerce.volt_wallets (user_id, balance, updated_at) VALUES (@user_id, 0, @now)
ON CONFLICT (user_id) DO UPDATE SET user_id = EXCLUDED.user_id
RETURNING balance;

-- name: InsertVolt :execrows
INSERT INTO commerce.volt_ledger (id, user_id, amount, reason, ref, key, balance_after, created_at)
VALUES (@id, @user_id, @amount, @reason, @ref, @key, @balance_after, @now)
ON CONFLICT (user_id, key) DO NOTHING;

-- name: SetBalance :exec
UPDATE commerce.volt_wallets SET balance = @balance, updated_at = @now WHERE user_id = @user_id;

-- name: GetBalance :one
SELECT balance FROM commerce.volt_wallets WHERE user_id = @user_id;

-- name: VoltHistory :many
SELECT amount, reason, ref, created_at FROM commerce.volt_ledger
WHERE user_id = @user_id ORDER BY created_at DESC, id DESC LIMIT @lim;

-- name: VoltSince :many
-- Movements of a reason since a time (a refunded Plus takes its Volt back).
SELECT key, amount FROM commerce.volt_ledger
WHERE user_id = @user_id AND reason = @reason AND created_at >= @since;

-- ---------- Cosmetics (D-106) ----------

-- name: OwnedCosmetics :many
SELECT code FROM commerce.cosmetic_inventory WHERE user_id = @user_id ORDER BY acquired_at;

-- name: OwnsCosmetic :one
SELECT EXISTS (SELECT 1 FROM commerce.cosmetic_inventory WHERE user_id = @user_id AND code = @code);

-- name: InsertCosmetic :execrows
INSERT INTO commerce.cosmetic_inventory (user_id, code, source, acquired_at)
VALUES (@user_id, @code, @source, @now)
ON CONFLICT DO NOTHING;

-- name: EquipCosmetic :exec
INSERT INTO commerce.cosmetic_equipped (user_id, kind, code, updated_at) VALUES (@user_id, @kind, @code, @now)
ON CONFLICT (user_id, kind) DO UPDATE SET code = EXCLUDED.code, updated_at = EXCLUDED.updated_at;

-- name: UnequipCosmetic :exec
DELETE FROM commerce.cosmetic_equipped WHERE user_id = @user_id AND kind = @kind;

-- name: EquippedOf :many
SELECT user_id, kind, code FROM commerce.cosmetic_equipped WHERE user_id = ANY(@user_ids::uuid[]);

-- name: UsersWithEntitlement :many
-- Which of users hold an entitlement now (Plus-only cosmetics, D-106).
SELECT DISTINCT user_id FROM commerce.entitlements
WHERE user_id = ANY(@user_ids::uuid[]) AND code = @code AND starts_at <= @now AND (ends_at IS NULL OR ends_at > @now);

-- ---------- Volt packs (D-107) ----------

-- name: VoltKeyExists :one
SELECT EXISTS (SELECT 1 FROM commerce.volt_ledger WHERE user_id = @user_id AND key = @key);

-- name: RemoveCosmetic :exec
DELETE FROM commerce.cosmetic_inventory WHERE user_id = @user_id AND code = @code;

-- name: UnequipCode :exec
DELETE FROM commerce.cosmetic_equipped WHERE user_id = @user_id AND code = @code;
