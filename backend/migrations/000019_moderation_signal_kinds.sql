-- Anti-cheat (Phase 8.5): matches the server cancelled again and again
-- for one player become a signal too (D-084).
-- +goose Up
ALTER TABLE moderation.cheat_signals DROP CONSTRAINT cheat_signals_kind_check;
ALTER TABLE moderation.cheat_signals ADD CONSTRAINT cheat_signals_kind_check CHECK (kind IN (
  'superhuman_response_time', 'consistent_fast_streak', 'rejected_submission_burst', 'protocol_violation',
  'win_trading', 'team_collusion', 'reconnect_abuse', 'multi_account', 'abnormal_rating_gain',
  'repeated_server_voids'));

-- +goose Down
DELETE FROM moderation.cheat_signals WHERE kind = 'repeated_server_voids';
ALTER TABLE moderation.cheat_signals DROP CONSTRAINT cheat_signals_kind_check;
ALTER TABLE moderation.cheat_signals ADD CONSTRAINT cheat_signals_kind_check CHECK (kind IN (
  'superhuman_response_time', 'consistent_fast_streak', 'rejected_submission_burst', 'protocol_violation',
  'win_trading', 'team_collusion', 'reconnect_abuse', 'multi_account', 'abnormal_rating_gain'));
