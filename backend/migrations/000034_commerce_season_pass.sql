-- Season Pass in commerce (D-108): buying the premium track spends Volt
-- (reason season_pass) and its rewards give Volt (pass_reward) and
-- season-only cosmetics (inventory source reward).
-- +goose Up
ALTER TABLE commerce.volt_ledger DROP CONSTRAINT volt_ledger_reason_check;
ALTER TABLE commerce.volt_ledger ADD CONSTRAINT volt_ledger_reason_check
  CHECK (reason IN ('quest','daily_challenge','level_up','tournament','season',
                    'ad','plus_monthly','plus_refund','item','cosmetic',
                    'purchase','purchase_refund','season_pass','pass_reward','admin'));

-- +goose Down
DELETE FROM commerce.volt_ledger WHERE reason IN ('season_pass','pass_reward');
ALTER TABLE commerce.volt_ledger DROP CONSTRAINT volt_ledger_reason_check;
ALTER TABLE commerce.volt_ledger ADD CONSTRAINT volt_ledger_reason_check
  CHECK (reason IN ('quest','daily_challenge','level_up','tournament','season',
                    'ad','plus_monthly','plus_refund','item','cosmetic',
                    'purchase','purchase_refund','admin'));
