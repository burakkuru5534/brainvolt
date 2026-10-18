-- content schema (docs/domain/schema-content.md §2): taxonomy, sources &
-- licenses (D-006), questions with per-locale text, verification (D-007),
-- revisions, question sets and answer statistics.
-- +goose Up
CREATE SCHEMA content;

-- ---------- Taxonomy ----------
CREATE TABLE content.categories (
  id          uuid        PRIMARY KEY,
  slug        text        NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9-]{2,40}$'),   -- e.g. 'history'
  icon_key    text        NOT NULL,                                           -- built-in icon
  sort_order  int         NOT NULL DEFAULT 0,
  is_active   boolean     NOT NULL DEFAULT true,
  created_at  timestamptz NOT NULL,
  updated_at  timestamptz NOT NULL
);

CREATE TABLE content.category_translations (
  category_id uuid NOT NULL REFERENCES content.categories(id) ON DELETE CASCADE,
  locale      text NOT NULL CHECK (locale IN ('tr','en')),
  name        text NOT NULL CHECK (char_length(name) BETWEEN 2 AND 40),
  description text,
  PRIMARY KEY (category_id, locale)
);

CREATE TABLE content.subcategories (
  id           uuid        PRIMARY KEY,
  category_id  uuid        NOT NULL REFERENCES content.categories(id),
  slug         text        NOT NULL CHECK (slug ~ '^[a-z0-9-]{2,40}$'),
  sort_order   int         NOT NULL DEFAULT 0,
  is_active    boolean     NOT NULL DEFAULT true,
  created_at   timestamptz NOT NULL,
  updated_at   timestamptz NOT NULL,
  UNIQUE (category_id, slug)
);

CREATE TABLE content.subcategory_translations (
  subcategory_id uuid NOT NULL REFERENCES content.subcategories(id) ON DELETE CASCADE,
  locale         text NOT NULL CHECK (locale IN ('tr','en')),
  name           text NOT NULL CHECK (char_length(name) BETWEEN 2 AND 40),
  description    text,
  PRIMARY KEY (subcategory_id, locale)
);

-- ---------- Sources & licenses (D-006) ----------
CREATE TABLE content.sources (
  id                uuid        PRIMARY KEY,
  kind              text        NOT NULL CHECK (kind IN ('opentdb','brainvolt_team','creator','partner')),
  name              text        NOT NULL,                  -- 'Open Trivia Database'
  url               text,                                  -- 'https://opentdb.com'
  license_code      text        NOT NULL,                  -- SPDX-style: 'CC-BY-SA-4.0', 'LicenseRef-BrainVolt', 'LicenseRef-BrainVolt-Creator-Terms'
  license_url       text,
  attribution_text  text        NOT NULL,                  -- shown on the in-app Credits screen
  share_alike       boolean     NOT NULL DEFAULT false,    -- true for CC BY-SA content: derivatives keep the same license
  notes             text,
  created_at        timestamptz NOT NULL
);

-- ---------- Questions ----------
CREATE TABLE content.questions (
  id                    uuid        PRIMARY KEY,
  subcategory_id        uuid        NOT NULL REFERENCES content.subcategories(id),
  question_type         text        NOT NULL CHECK (question_type IN ('multiple_choice','true_false')),
  difficulty            smallint    NOT NULL CHECK (difficulty BETWEEN 1 AND 5),  -- author/import estimate
  difficulty_calibrated numeric(3,2),                       -- 1.00–5.00, computed from answer stats (§5); NULL until enough data
  original_locale       text        NOT NULL CHECK (original_locale IN ('tr','en')),
  source_id             uuid        NOT NULL REFERENCES content.sources(id),       -- D-006
  source_ref            text,                               -- id/URL/hash inside the source, for traceability
  author_user_id        uuid        REFERENCES identity.users(id),                  -- set for creator & team-authored questions
  pool                  text        NOT NULL DEFAULT 'general'
                        CHECK (pool IN ('general','creator_only')),                 -- creator questions stay out of general matches unless promoted
  fingerprint           text        NOT NULL UNIQUE,        -- hash of normalised original prompt + sorted option texts; blocks duplicate imports
  verification_status   text        NOT NULL DEFAULT 'draft'
                        CHECK (verification_status IN ('draft','in_review','verified','rejected')),
  verified_by           uuid        REFERENCES identity.users(id),
  verified_at           timestamptz,
  rejection_reason      text,
  retired_at            timestamptz,                        -- no longer selectable; kept for history
  revision              int         NOT NULL DEFAULT 1,     -- bumped on every content edit (§4.3)
  created_at            timestamptz NOT NULL,
  updated_at            timestamptz NOT NULL,
  CHECK (verification_status <> 'verified' OR (verified_by IS NOT NULL AND verified_at IS NOT NULL)),
  CHECK (verification_status <> 'rejected' OR rejection_reason IS NOT NULL)
);
CREATE INDEX questions_selectable_idx ON content.questions (subcategory_id, difficulty)
  WHERE retired_at IS NULL AND pool = 'general';
CREATE INDEX questions_status_idx ON content.questions (verification_status) WHERE retired_at IS NULL;

CREATE TABLE content.question_translations (
  question_id          uuid        NOT NULL REFERENCES content.questions(id) ON DELETE CASCADE,
  locale               text        NOT NULL CHECK (locale IN ('tr','en')),
  prompt               text        NOT NULL CHECK (char_length(prompt) BETWEEN 5 AND 300),
  explanation          text        CHECK (char_length(explanation) <= 500),  -- optional, shown after reveal
  origin               text        NOT NULL CHECK (origin IN ('original','human_translation','machine_translation')),
  verification_status  text        NOT NULL DEFAULT 'draft'
                       CHECK (verification_status IN ('draft','in_review','verified','rejected')),
  verified_by          uuid        REFERENCES identity.users(id),
  verified_at          timestamptz,
  rejection_reason     text,
  created_at           timestamptz NOT NULL,
  updated_at           timestamptz NOT NULL,
  PRIMARY KEY (question_id, locale),
  CHECK (verification_status <> 'verified' OR (verified_by IS NOT NULL AND verified_at IS NOT NULL)),
  CHECK (origin <> 'machine_translation' OR verification_status <> 'verified' OR verified_by IS NOT NULL)
);

CREATE TABLE content.options (
  id           uuid     PRIMARY KEY,
  question_id  uuid     NOT NULL REFERENCES content.questions(id) ON DELETE CASCADE,
  position     smallint NOT NULL CHECK (position BETWEEN 1 AND 4),   -- canonical order; matches shuffle
  is_correct   boolean  NOT NULL,
  UNIQUE (question_id, position)
);
CREATE UNIQUE INDEX options_one_correct_uq ON content.options (question_id) WHERE is_correct;

CREATE TABLE content.option_translations (
  option_id  uuid NOT NULL REFERENCES content.options(id) ON DELETE CASCADE,
  locale     text NOT NULL CHECK (locale IN ('tr','en')),
  text       text NOT NULL CHECK (char_length(text) BETWEEN 1 AND 120),
  PRIMARY KEY (option_id, locale)
);

-- ---------- Revision snapshots (what players saw; review finding F1, 0.2.10) ----------
CREATE TABLE content.question_revisions (
  question_id  uuid        NOT NULL REFERENCES content.questions(id),
  revision     int         NOT NULL,
  snapshot     jsonb       NOT NULL,      -- all locales: prompt, explanation, option texts by option id, correct option id
  created_by   uuid        REFERENCES identity.users(id),
  created_at   timestamptz NOT NULL,
  PRIMARY KEY (question_id, revision)
);

-- ---------- Audit ----------
CREATE TABLE content.verification_events (
  id           uuid        PRIMARY KEY,
  question_id  uuid        NOT NULL REFERENCES content.questions(id),
  locale       text        CHECK (locale IN ('tr','en')),     -- NULL = question-level; set = translation-level
  from_status  text        NOT NULL,
  to_status    text        NOT NULL,
  actor_id     uuid        REFERENCES identity.users(id),     -- NULL = system (import, auto-retire)
  reason       text,
  revision     int         NOT NULL,
  created_at   timestamptz NOT NULL
);
CREATE INDEX verification_events_question_idx ON content.verification_events (question_id, created_at);

-- ---------- Question sets (Daily Challenge, Creator Quiz, tournaments, curated) ----------
CREATE TABLE content.question_sets (
  id          uuid        PRIMARY KEY,
  kind        text        NOT NULL CHECK (kind IN ('daily','creator','tournament','curated')),
  created_by  uuid        REFERENCES identity.users(id),
  created_at  timestamptz NOT NULL
);

CREATE TABLE content.question_set_items (
  set_id       uuid     NOT NULL REFERENCES content.question_sets(id) ON DELETE CASCADE,
  position     smallint NOT NULL CHECK (position >= 1),
  question_id  uuid     NOT NULL REFERENCES content.questions(id),
  PRIMARY KEY (set_id, position),
  UNIQUE (set_id, question_id)
);

CREATE TABLE content.daily_sets (
  day          date        PRIMARY KEY,                     -- calendar day in Europe/Istanbul
  set_id       uuid        NOT NULL UNIQUE REFERENCES content.question_sets(id),
  published_at timestamptz NOT NULL
);

-- ---------- Aggregated answer statistics (§5) ----------
CREATE TABLE content.question_stats (
  question_id      uuid        NOT NULL REFERENCES content.questions(id),
  locale           text        NOT NULL CHECK (locale IN ('tr','en')),
  times_asked      bigint      NOT NULL DEFAULT 0,
  times_correct    bigint      NOT NULL DEFAULT 0,
  times_timed_out  bigint      NOT NULL DEFAULT 0,
  sum_response_ms  bigint      NOT NULL DEFAULT 0,          -- correct answers only; avg = sum / times_correct
  last_asked_at    timestamptz,
  PRIMARY KEY (question_id, locale)
);

-- Moderator listing: newest first per status (rest.md §2).
CREATE INDEX questions_list_idx ON content.questions (verification_status, id DESC);

-- +goose Down
DROP SCHEMA content CASCADE;
