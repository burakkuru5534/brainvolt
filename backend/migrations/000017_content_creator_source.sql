-- The source of player-written Creator Quiz questions (D-006: every
-- question has a source; creator questions stay in the creator_only pool).
-- +goose Up
INSERT INTO content.sources (id, kind, name, license_code, attribution_text, created_at)
VALUES ('01a1c4e0-0000-7000-8000-000000000001', 'creator', 'BrainVolt players', 'LicenseRef-BrainVolt-Creator-Terms',
        'Creator quizzes are written by BrainVolt players.', now())
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM content.sources WHERE id = '01a1c4e0-0000-7000-8000-000000000001'
  AND NOT EXISTS (SELECT 1 FROM content.questions WHERE source_id = '01a1c4e0-0000-7000-8000-000000000001');
