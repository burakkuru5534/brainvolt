-- progression schema (docs/domain/schema-ranking.md §5): XP and levels, play
-- streaks, badges and quests, plus the counters badges are checked against
-- and the matches already consumed. Badge and quest catalogues are seeded
-- here (progression-formulas §7–8, D-077).
-- +goose Up
CREATE SCHEMA progression;

-- ---------- XP & level ----------
CREATE TABLE progression.user_progress (
  user_id     uuid        PRIMARY KEY REFERENCES identity.users(id) ON DELETE CASCADE,
  xp_total    bigint      NOT NULL DEFAULT 0 CHECK (xp_total >= 0),
  level       int         NOT NULL DEFAULT 1 CHECK (level >= 1),
  updated_at  timestamptz NOT NULL
);

CREATE TABLE progression.xp_ledger (
  id          uuid        PRIMARY KEY,
  user_id     uuid        NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
  amount      int         NOT NULL CHECK (amount > 0),
  source      text        NOT NULL CHECK (source IN ('match','quest','badge','daily_challenge','season_reward','tournament')),
  source_ref  text        NOT NULL,
  created_at  timestamptz NOT NULL,
  UNIQUE (user_id, source, source_ref)
);

-- ---------- Play streak ----------
CREATE TABLE progression.play_streaks (
  user_id         uuid  PRIMARY KEY REFERENCES identity.users(id) ON DELETE CASCADE,
  current_days    int   NOT NULL DEFAULT 0,
  longest_days    int   NOT NULL DEFAULT 0,
  last_play_day   date,
  updated_at      timestamptz NOT NULL
);

-- ---------- Counters for badges ----------
CREATE TABLE progression.user_stats (
  user_id      uuid        PRIMARY KEY REFERENCES identity.users(id) ON DELETE CASCADE,
  matches      int         NOT NULL DEFAULT 0,
  wins         int         NOT NULL DEFAULT 0,
  correct      int         NOT NULL DEFAULT 0,
  updated_at   timestamptz NOT NULL
);

CREATE TABLE progression.category_correct (
  user_id      uuid NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
  category_id  uuid NOT NULL,
  correct      int  NOT NULL DEFAULT 0,
  PRIMARY KEY (user_id, category_id)
);

-- ---------- Badges ----------
CREATE TABLE progression.badges (
  id          uuid        PRIMARY KEY,
  code        text        NOT NULL UNIQUE,
  family      text        NOT NULL,
  tier        smallint    NOT NULL DEFAULT 1,
  criteria    jsonb       NOT NULL,
  icon_key    text        NOT NULL,
  is_hidden   boolean     NOT NULL DEFAULT false,
  is_active   boolean     NOT NULL DEFAULT true,
  created_at  timestamptz NOT NULL
);
CREATE TABLE progression.badge_translations (
  badge_id     uuid NOT NULL REFERENCES progression.badges(id) ON DELETE CASCADE,
  locale       text NOT NULL CHECK (locale IN ('tr','en')),
  name         text NOT NULL,
  description  text NOT NULL,
  PRIMARY KEY (badge_id, locale)
);
CREATE TABLE progression.user_badges (
  user_id     uuid        NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
  badge_id    uuid        NOT NULL REFERENCES progression.badges(id),
  earned_at   timestamptz NOT NULL,
  source_ref  text,
  PRIMARY KEY (user_id, badge_id)
);
CREATE INDEX user_badges_source_idx ON progression.user_badges (source_ref);

-- ---------- Quests ----------
CREATE TABLE progression.quests (
  id              uuid        PRIMARY KEY,
  code            text        NOT NULL UNIQUE,
  cadence         text        NOT NULL CHECK (cadence IN ('daily','weekly','seasonal')),
  objective       text        NOT NULL CHECK (objective IN ('play_matches','win_matches','correct_answers',
                                                            'play_mode','play_category','win_streak',
                                                            'play_with_friend','play_team_match','daily_challenge')),
  target          int         NOT NULL CHECK (target >= 1),
  filters         jsonb       NOT NULL DEFAULT '{}',
  reward_xp       int         NOT NULL CHECK (reward_xp >= 0),
  reward_extra    jsonb       NOT NULL DEFAULT '{}',
  requires_full_account boolean NOT NULL DEFAULT false,
  season_id       uuid        REFERENCES ranking.seasons(id),
  is_active       boolean     NOT NULL DEFAULT true,
  created_at      timestamptz NOT NULL
);
CREATE TABLE progression.quest_translations (
  quest_id     uuid NOT NULL REFERENCES progression.quests(id) ON DELETE CASCADE,
  locale       text NOT NULL CHECK (locale IN ('tr','en')),
  title        text NOT NULL,
  description  text NOT NULL,
  PRIMARY KEY (quest_id, locale)
);
CREATE TABLE progression.quest_assignments (
  id            uuid        PRIMARY KEY,
  user_id       uuid        NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
  quest_id      uuid        NOT NULL REFERENCES progression.quests(id),
  period_start  timestamptz NOT NULL,
  period_end    timestamptz NOT NULL,
  progress      int         NOT NULL DEFAULT 0,
  completed_at  timestamptz,
  claimed_at    timestamptz,
  UNIQUE (user_id, quest_id, period_start)
);
CREATE INDEX quest_assignments_open_idx ON progression.quest_assignments (user_id, period_end) WHERE claimed_at IS NULL;

-- One free reroll per user and Europe/Istanbul day (progression-formulas §8).
CREATE TABLE progression.quest_rerolls (
  user_id  uuid NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
  day      date NOT NULL,
  count    int  NOT NULL,
  PRIMARY KEY (user_id, day)
);

-- Matches already consumed (idempotent game.MatchCompleted handling).
CREATE TABLE progression.processed_matches (
  match_id      uuid        PRIMARY KEY,
  processed_at  timestamptz NOT NULL
);

-- ---------- Catalogue seeds ----------
CREATE TEMPORARY TABLE seed_badges (code text, family text, tier smallint, criteria jsonb, tr_name text, tr_desc text, en_name text, en_desc text) ON COMMIT DROP;
INSERT INTO seed_badges VALUES
  ('first_win',          'wins',       1, '{"type":"wins","count":1}',    'İlk Galibiyet', 'İlk maçını kazan.', 'First Win', 'Win your first match.'),
  ('wins_10',            'wins',       2, '{"type":"wins","count":10}',   '10 Galibiyet', '10 maç kazan.', '10 Wins', 'Win 10 matches.'),
  ('wins_100',           'wins',       3, '{"type":"wins","count":100}',  '100 Galibiyet', '100 maç kazan.', '100 Wins', 'Win 100 matches.'),
  ('wins_1000',          'wins',       4, '{"type":"wins","count":1000}', '1000 Galibiyet', '1000 maç kazan.', '1,000 Wins', 'Win 1,000 matches.'),
  ('perfect_quick_duel', 'accuracy',   1, '{"type":"perfect","match_type":"quick_duel","rounds":7}', 'Kusursuz Düello', 'Hızlı Düello''da 7/7 yap.', 'Flawless Duel', 'Answer 7/7 in a Quick Duel.'),
  ('perfect_ranked',     'accuracy',   2, '{"type":"perfect","match_type":"ranked","rounds":10}',    'Kusursuz Dereceli', 'Dereceli maçta 10/10 yap.', 'Flawless Ranked', 'Answer 10/10 in a Ranked match.'),
  ('perfect_marathon',   'accuracy',   3, '{"type":"perfect","match_type":"marathon","rounds":25}',  'Kusursuz Maraton', 'Maratonda 25/25 yap.', 'Flawless Marathon', 'Answer 25/25 in a Marathon.'),
  ('answer_streak_10',   'streaks',    2, '{"type":"answer_streak","count":10}', 'Seri Bilgin', 'Bir maçta üst üste 10 doğru cevap ver.', 'On a Roll', 'Answer 10 in a row correctly in one match.'),
  ('play_streak_3',      'streaks',    1, '{"type":"play_streak","days":3}',   '3 Günlük Seri', '3 gün üst üste oyna.', '3-Day Streak', 'Play 3 days in a row.'),
  ('play_streak_7',      'streaks',    2, '{"type":"play_streak","days":7}',   '7 Günlük Seri', '7 gün üst üste oyna.', '7-Day Streak', 'Play 7 days in a row.'),
  ('play_streak_30',     'streaks',    3, '{"type":"play_streak","days":30}',  '30 Günlük Seri', '30 gün üst üste oyna.', '30-Day Streak', 'Play 30 days in a row.'),
  ('play_streak_100',    'streaks',    4, '{"type":"play_streak","days":100}', '100 Günlük Seri', '100 gün üst üste oyna.', '100-Day Streak', 'Play 100 days in a row.'),
  ('play_streak_365',    'streaks',    5, '{"type":"play_streak","days":365}', '365 Günlük Seri', 'Bir yıl boyunca her gün oyna.', '365-Day Streak', 'Play every day for a year.'),
  ('category_50',        'categories', 1, '{"type":"category_correct","count":50}',   'Kategori Meraklısı', 'Bir kategoride 50 doğru cevap ver.', 'Category Fan', 'Answer 50 correctly in one category.'),
  ('category_250',       'categories', 2, '{"type":"category_correct","count":250}',  'Kategori Uzmanı', 'Bir kategoride 250 doğru cevap ver.', 'Category Expert', 'Answer 250 correctly in one category.'),
  ('category_1000',      'categories', 3, '{"type":"category_correct","count":1000}', 'Kategori Ustası', 'Bir kategoride 1000 doğru cevap ver.', 'Category Master', 'Answer 1,000 correctly in one category.'),
  ('first_team_match',   'social',     1, '{"type":"team_match"}', 'Takım Oyuncusu', 'İlk takım maçını oyna.', 'Team Player', 'Play your first team match.'),
  ('first_couple_win',   'social',     1, '{"type":"couple_win"}', 'Uyumlu Çift', 'İlk çift maçını kazan.', 'Perfect Match', 'Win your first Couple Contest.'),
  ('season_bronze',      'seasons',    1, '{"type":"season_peak","league":"bronze"}',   'Sezon: Bronz', 'Bir sezonu Bronz ligde bitir.', 'Season: Bronze', 'Peak in Bronze in a season.'),
  ('season_silver',      'seasons',    2, '{"type":"season_peak","league":"silver"}',   'Sezon: Gümüş', 'Bir sezonda Gümüş lige ulaş.', 'Season: Silver', 'Reach Silver in a season.'),
  ('season_gold',        'seasons',    3, '{"type":"season_peak","league":"gold"}',     'Sezon: Altın', 'Bir sezonda Altın lige ulaş.', 'Season: Gold', 'Reach Gold in a season.'),
  ('season_platinum',    'seasons',    4, '{"type":"season_peak","league":"platinum"}', 'Sezon: Platin', 'Bir sezonda Platin lige ulaş.', 'Season: Platinum', 'Reach Platinum in a season.'),
  ('season_diamond',     'seasons',    5, '{"type":"season_peak","league":"diamond"}',  'Sezon: Elmas', 'Bir sezonda Elmas lige ulaş.', 'Season: Diamond', 'Reach Diamond in a season.'),
  ('season_top_100',     'seasons',    5, '{"type":"season_top","rank":100}',           'Sezonun İlk 100''ü', 'Bir sezonu ilk 100''de bitir.', 'Season Top 100', 'Finish a season in the global top 100.');
INSERT INTO progression.badges (id, code, family, tier, criteria, icon_key, created_at)
  SELECT gen_random_uuid(), code, family, tier, criteria, code, now() FROM seed_badges;
INSERT INTO progression.badge_translations (badge_id, locale, name, description)
  SELECT b.id, 'tr', s.tr_name, s.tr_desc FROM seed_badges s JOIN progression.badges b USING (code)
  UNION ALL
  SELECT b.id, 'en', s.en_name, s.en_desc FROM seed_badges s JOIN progression.badges b USING (code);

CREATE TEMPORARY TABLE seed_quests (code text, cadence text, objective text, target int, filters jsonb, reward_xp int, tr_title text, en_title text) ON COMMIT DROP;
INSERT INTO seed_quests VALUES
  ('daily_play_3',        'daily',  'play_matches',    3,   '{}', 50,  '3 maç oyna', 'Play 3 matches'),
  ('daily_win_2',         'daily',  'win_matches',     2,   '{}', 75,  '2 maç kazan', 'Win 2 matches'),
  ('daily_correct_20',    'daily',  'correct_answers', 20,  '{}', 60,  '20 doğru cevap ver', 'Answer 20 correctly'),
  ('daily_ranked_2',      'daily',  'play_mode',       2,   '{"match_type":"ranked"}', 75, '2 dereceli maç oyna', 'Play 2 Ranked matches'),
  ('daily_team_1',        'daily',  'play_team_match', 1,   '{}', 75,  'Bir takım maçı oyna', 'Play a team match'),
  ('daily_win_streak_2',  'daily',  'win_streak',      2,   '{}', 100, 'Üst üste 2 maç kazan', 'Win 2 matches in a row'),
  ('weekly_play_15',      'weekly', 'play_matches',    15,  '{}', 200, '15 maç oyna', 'Play 15 matches'),
  ('weekly_win_10',       'weekly', 'win_matches',     10,  '{}', 300, '10 maç kazan', 'Win 10 matches'),
  ('weekly_correct_120',  'weekly', 'correct_answers', 120, '{}', 250, '120 doğru cevap ver', 'Answer 120 correctly'),
  ('weekly_ranked_10',    'weekly', 'play_mode',       10,  '{"match_type":"ranked"}', 300, '10 dereceli maç oyna', 'Play 10 Ranked matches'),
  ('weekly_marathon_2',   'weekly', 'play_mode',       2,   '{"match_type":"marathon"}', 250, '2 maraton oyna', 'Play 2 Marathons'),
  ('weekly_team_5',       'weekly', 'play_team_match', 5,   '{}', 300, '5 takım maçı oyna', 'Play 5 team matches'),
  ('weekly_win_streak_5', 'weekly', 'win_streak',      5,   '{}', 400, 'Üst üste 5 maç kazan', 'Win 5 matches in a row');
INSERT INTO progression.quests (id, code, cadence, objective, target, filters, reward_xp, created_at)
  SELECT gen_random_uuid(), code, cadence, objective, target, filters, reward_xp, now() FROM seed_quests;
INSERT INTO progression.quest_translations (quest_id, locale, title, description)
  SELECT q.id, 'tr', s.tr_title, s.tr_title FROM seed_quests s JOIN progression.quests q USING (code)
  UNION ALL
  SELECT q.id, 'en', s.en_title, s.en_title FROM seed_quests s JOIN progression.quests q USING (code);

-- +goose Down
DROP SCHEMA progression CASCADE;
