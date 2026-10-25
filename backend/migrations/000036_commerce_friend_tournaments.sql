-- Friend tournaments in commerce (D-109): the set-up fee and its refund.
-- +goose Up
ALTER TABLE commerce.volt_ledger DROP CONSTRAINT volt_ledger_reason_check;
ALTER TABLE commerce.volt_ledger ADD CONSTRAINT volt_ledger_reason_check
  CHECK (reason IN ('quest','daily_challenge','level_up','tournament','season',
                    'ad','plus_monthly','plus_refund','item','cosmetic',
                    'purchase','purchase_refund','season_pass','pass_reward',
                    'friend_tournament','friend_tournament_refund','admin'));

-- +goose Down
DELETE FROM commerce.volt_ledger WHERE reason IN ('friend_tournament','friend_tournament_refund');
ALTER TABLE commerce.volt_ledger DROP CONSTRAINT volt_ledger_reason_check;
ALTER TABLE commerce.volt_ledger ADD CONSTRAINT volt_ledger_reason_check
  CHECK (reason IN ('quest','daily_challenge','level_up','tournament','season',
                    'ad','plus_monthly','plus_refund','item','cosmetic',
                    'purchase','purchase_refund','season_pass','pass_reward','admin'));
