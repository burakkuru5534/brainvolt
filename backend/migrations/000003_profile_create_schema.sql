-- profile schema (docs/domain/schema-identity.md §3).
-- +goose Up
CREATE SCHEMA profile;

CREATE TABLE profile.profiles (
  user_id              uuid        PRIMARY KEY REFERENCES identity.users(id) ON DELETE CASCADE,
  username             citext      NOT NULL UNIQUE
                       CHECK (username ~ '^[a-z0-9_.]{3,20}$'),  -- handle used for friend search
  display_name         text        NOT NULL CHECK (char_length(display_name) BETWEEN 2 AND 24),
  avatar_id            text        NOT NULL,                       -- built-in preset key only (D-021)
  country_code         char(2)     CHECK (country_code ~ '^[A-Z]{2}$'),
  locale               text        NOT NULL DEFAULT 'tr' CHECK (locale IN ('tr','en')),
  username_changed_at  timestamptz,
  country_changed_at   timestamptz,
  created_at           timestamptz NOT NULL,
  updated_at           timestamptz NOT NULL
);
CREATE INDEX profiles_country_idx ON profile.profiles (country_code);

CREATE TABLE profile.settings (
  user_id                 uuid    PRIMARY KEY REFERENCES identity.users(id) ON DELETE CASCADE,
  challenges_from         text    NOT NULL DEFAULT 'everyone'
                          CHECK (challenges_from IN ('everyone','friends','nobody')),
  show_online_status      boolean NOT NULL DEFAULT true,
  allow_friend_requests   boolean NOT NULL DEFAULT true,
  sound_enabled           boolean NOT NULL DEFAULT true,
  music_enabled           boolean NOT NULL DEFAULT true,
  haptics_enabled         boolean NOT NULL DEFAULT true,
  updated_at              timestamptz NOT NULL
);

-- +goose Down
DROP SCHEMA profile CASCADE;
