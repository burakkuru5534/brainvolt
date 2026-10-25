-- Volt packs (D-107): Volt bought with money through RevenueCat (Google
-- Play one-time products volt_100 … volt_2000 and the once-only
-- volt_starter, which also gives the Starter frame). A refund takes the
-- pack's Volt back; the balance may go below zero, which blocks spending.
-- +goose Up
ALTER TABLE commerce.volt_ledger DROP CONSTRAINT volt_ledger_reason_check;
ALTER TABLE commerce.volt_ledger ADD CONSTRAINT volt_ledger_reason_check
  CHECK (reason IN ('quest','daily_challenge','level_up','tournament','season',
                    'ad','plus_monthly','plus_refund','item','cosmetic',
                    'purchase','purchase_refund','admin'));

ALTER TABLE commerce.cosmetic_inventory DROP CONSTRAINT cosmetic_inventory_source_check;
ALTER TABLE commerce.cosmetic_inventory ADD CONSTRAINT cosmetic_inventory_source_check
  CHECK (source IN ('volt','admin','reward','purchase'));

-- +goose Down
DELETE FROM commerce.cosmetic_inventory WHERE source = 'purchase';
ALTER TABLE commerce.cosmetic_inventory DROP CONSTRAINT cosmetic_inventory_source_check;
ALTER TABLE commerce.cosmetic_inventory ADD CONSTRAINT cosmetic_inventory_source_check
  CHECK (source IN ('volt','admin','reward'));
DELETE FROM commerce.volt_ledger WHERE reason IN ('purchase','purchase_refund');
ALTER TABLE commerce.volt_ledger DROP CONSTRAINT volt_ledger_reason_check;
ALTER TABLE commerce.volt_ledger ADD CONSTRAINT volt_ledger_reason_check
  CHECK (reason IN ('quest','daily_challenge','level_up','tournament','season',
                    'ad','plus_monthly','plus_refund','item','cosmetic','admin'));
