-- lobby schema (docs/domain/schema-social.md §3): invite codes, private
-- rooms and challenges. Live room state (members, seats, ready flags) is in
-- Redis (redis-keys §5).
-- +goose Up
CREATE SCHEMA lobby;

CREATE TABLE lobby.invite_codes (
  code          text        PRIMARY KEY CHECK (code ~ '^[2-9A-HJ-NP-Z]{6}$'),
  target_kind   text        NOT NULL CHECK (target_kind IN ('room','challenge','team','couple','friend')),
  target_id     uuid        NOT NULL,
  created_by    uuid        NOT NULL REFERENCES identity.users(id),
  max_uses      int         CHECK (max_uses >= 1),
  use_count     int         NOT NULL DEFAULT 0,
  expires_at    timestamptz NOT NULL,
  revoked_at    timestamptz,
  created_at    timestamptz NOT NULL
);
CREATE INDEX invite_codes_target_idx ON lobby.invite_codes (target_kind, target_id);

CREATE TABLE lobby.rooms (
  id                   uuid        PRIMARY KEY,
  host_user_id         uuid        NOT NULL REFERENCES identity.users(id),
  status               text        NOT NULL DEFAULT 'open'
                       CHECK (status IN ('open','starting','in_match','closed')),
  match_config         jsonb       NOT NULL,
  creator_quiz_id      uuid,
  friends_of_host_only boolean     NOT NULL DEFAULT false,
  current_match_id     uuid        REFERENCES game.matches(id),
  matches_played       int         NOT NULL DEFAULT 0,
  created_at           timestamptz NOT NULL,
  last_activity_at     timestamptz NOT NULL,
  closed_at            timestamptz,
  close_reason         text        CHECK (close_reason IN ('host_closed','idle','all_left','moderation'))
);
CREATE INDEX rooms_open_idx ON lobby.rooms (last_activity_at) WHERE status <> 'closed';

CREATE TABLE lobby.challenges (
  id                   uuid        PRIMARY KEY,
  kind                 text        NOT NULL CHECK (kind IN ('direct','rematch','revenge')),
  challenger_user_id   uuid        NOT NULL REFERENCES identity.users(id),
  challenger_team_id   uuid,
  challenger_couple_id uuid,
  target_user_id       uuid        REFERENCES identity.users(id),
  target_team_id       uuid,
  target_couple_id     uuid,
  match_config         jsonb       NOT NULL,
  parent_match_id      uuid        REFERENCES game.matches(id),
  status               text        NOT NULL DEFAULT 'pending'
                       CHECK (status IN ('pending','accepted','declined','cancelled','expired','started','failed')),
  match_id             uuid        REFERENCES game.matches(id),
  created_at           timestamptz NOT NULL,
  expires_at           timestamptz NOT NULL,
  responded_at         timestamptz,
  CHECK (num_nonnulls(target_user_id, target_team_id, target_couple_id) = 1),
  CHECK ((kind = 'direct') = (parent_match_id IS NULL))
);
CREATE INDEX challenges_target_user_idx ON lobby.challenges (target_user_id) WHERE status = 'pending';
CREATE INDEX challenges_parent_idx      ON lobby.challenges (parent_match_id) WHERE parent_match_id IS NOT NULL;
CREATE INDEX challenges_challenger_idx  ON lobby.challenges (challenger_user_id) WHERE status = 'pending';
-- hourly send cap (20/h) counts by challenger and time
CREATE INDEX challenges_sent_idx        ON lobby.challenges (challenger_user_id, created_at);
CREATE INDEX challenges_match_idx       ON lobby.challenges (match_id) WHERE match_id IS NOT NULL;

-- +goose Down
DROP SCHEMA lobby CASCADE;
