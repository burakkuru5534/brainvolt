-- Cosmetics (D-106): what a player owns and what they wear. The catalogue
-- (codes, kinds, Volt prices, Plus-only items) lives in code; a purchase
-- writes the inventory and the Volt ledger in one transaction. Plus-only
-- items are never in the inventory: they are worn while Plus lasts and
-- dropped from what others see when it ends. Special avatars are worn
-- through the profile's avatar_id.
-- +goose Up
CREATE TABLE commerce.cosmetic_inventory (
  user_id      uuid        NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
  code         text        NOT NULL,
  source       text        NOT NULL CHECK (source IN ('volt','admin','reward')),
  acquired_at  timestamptz NOT NULL,
  PRIMARY KEY (user_id, code)
);

CREATE TABLE commerce.cosmetic_equipped (
  user_id     uuid        NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
  kind        text        NOT NULL CHECK (kind IN ('frame','name_color','victory','match_theme')),
  code        text        NOT NULL,
  updated_at  timestamptz NOT NULL,
  PRIMARY KEY (user_id, kind)
);

ALTER TABLE commerce.volt_ledger DROP CONSTRAINT volt_ledger_reason_check;
ALTER TABLE commerce.volt_ledger ADD CONSTRAINT volt_ledger_reason_check
  CHECK (reason IN ('quest','daily_challenge','level_up','tournament','season',
                    'ad','plus_monthly','plus_refund','item','cosmetic','admin'));

-- +goose Down
DELETE FROM commerce.volt_ledger WHERE reason = 'cosmetic';
ALTER TABLE commerce.volt_ledger DROP CONSTRAINT volt_ledger_reason_check;
ALTER TABLE commerce.volt_ledger ADD CONSTRAINT volt_ledger_reason_check
  CHECK (reason IN ('quest','daily_challenge','level_up','tournament','season',
                    'ad','plus_monthly','plus_refund','item','admin'));
DROP TABLE commerce.cosmetic_equipped;
DROP TABLE commerce.cosmetic_inventory;
