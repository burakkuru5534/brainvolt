-- The daily quest for finishing the Daily Challenge (objective
-- daily_challenge, progression-formulas §8).
-- +goose Up
INSERT INTO progression.quests (id, code, cadence, objective, target, filters, reward_xp, created_at)
VALUES (gen_random_uuid(), 'daily_daily_challenge', 'daily', 'daily_challenge', 1, '{}', 75, now());
INSERT INTO progression.quest_translations (quest_id, locale, title, description)
SELECT id, 'tr', 'Günün meydan okumasını tamamla', 'Günün meydan okumasını tamamla' FROM progression.quests WHERE code = 'daily_daily_challenge'
UNION ALL
SELECT id, 'en', 'Finish the Daily Challenge', 'Finish the Daily Challenge' FROM progression.quests WHERE code = 'daily_daily_challenge';

-- +goose Down
DELETE FROM progression.quest_assignments WHERE quest_id IN (SELECT id FROM progression.quests WHERE code = 'daily_daily_challenge');
DELETE FROM progression.quests WHERE code = 'daily_daily_challenge';
