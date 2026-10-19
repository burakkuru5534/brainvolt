-- team schema (docs/domain/schema-social.md §4): persistent teams with
-- memberships and invites, and couple profiles (D-018, D-047).
-- +goose Up
CREATE SCHEMA team;

CREATE TABLE team.teams (
  id               uuid        PRIMARY KEY,
  name             citext      NOT NULL CHECK (char_length(name) BETWEEN 3 AND 24),
  tag              citext      NOT NULL CHECK (tag ~ '^[A-Za-z0-9]{2,4}$'),
  emblem_id        text        NOT NULL,
  captain_user_id  uuid        NOT NULL REFERENCES identity.users(id),
  status           text        NOT NULL DEFAULT 'active' CHECK (status IN ('active','disbanded')),
  created_at       timestamptz NOT NULL,
  updated_at       timestamptz NOT NULL,
  disbanded_at     timestamptz
);
CREATE UNIQUE INDEX teams_name_active_uq ON team.teams (name) WHERE status = 'active';
CREATE UNIQUE INDEX teams_tag_active_uq  ON team.teams (tag)  WHERE status = 'active';

CREATE TABLE team.memberships (
  team_id      uuid        NOT NULL REFERENCES team.teams(id),
  user_id      uuid        NOT NULL REFERENCES identity.users(id),
  role         text        NOT NULL CHECK (role IN ('captain','member')),
  joined_at    timestamptz NOT NULL,
  left_at      timestamptz,
  leave_reason text        CHECK (leave_reason IN ('left','kicked','team_disbanded','account_deleted')),
  PRIMARY KEY (team_id, user_id, joined_at)
);
CREATE UNIQUE INDEX memberships_current_uq     ON team.memberships (team_id, user_id) WHERE left_at IS NULL;
CREATE UNIQUE INDEX memberships_one_captain_uq ON team.memberships (team_id) WHERE left_at IS NULL AND role = 'captain';
CREATE INDEX memberships_user_idx ON team.memberships (user_id) WHERE left_at IS NULL;

CREATE TABLE team.invites (
  id               uuid        PRIMARY KEY,
  team_id          uuid        NOT NULL REFERENCES team.teams(id),
  invited_user_id  uuid        NOT NULL REFERENCES identity.users(id),
  invited_by       uuid        NOT NULL REFERENCES identity.users(id),
  status           text        NOT NULL DEFAULT 'pending'
                   CHECK (status IN ('pending','accepted','declined','cancelled','expired')),
  created_at       timestamptz NOT NULL,
  expires_at       timestamptz NOT NULL,
  responded_at     timestamptz
);
CREATE UNIQUE INDEX team_invites_pending_uq ON team.invites (team_id, invited_user_id) WHERE status = 'pending';
CREATE INDEX team_invites_user_idx ON team.invites (invited_user_id) WHERE status = 'pending';

-- One row per pair: forming the same couple again reactivates it (D-047).
CREATE TABLE team.couples (
  id                uuid        PRIMARY KEY,
  user_low_id       uuid        NOT NULL REFERENCES identity.users(id),
  user_high_id      uuid        NOT NULL REFERENCES identity.users(id),
  requested_by      uuid        NOT NULL REFERENCES identity.users(id),
  name              citext      CHECK (char_length(name) BETWEEN 3 AND 24),
  status            text        NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending','active','declined','dissolved','expired')),
  created_at        timestamptz NOT NULL,
  requested_at      timestamptz NOT NULL,                  -- the current request (expires after 7 days)
  activated_at      timestamptz,
  dissolved_at      timestamptz,
  last_dissolved_at timestamptz,
  CHECK (user_low_id < user_high_id),
  CHECK (requested_by IN (user_low_id, user_high_id))
);
CREATE UNIQUE INDEX couples_pair_uq      ON team.couples (user_low_id, user_high_id);
CREATE UNIQUE INDEX couples_low_live_uq  ON team.couples (user_low_id)  WHERE status IN ('pending','active');
CREATE UNIQUE INDEX couples_high_live_uq ON team.couples (user_high_id) WHERE status IN ('pending','active');

-- +goose Down
DROP SCHEMA team CASCADE;
