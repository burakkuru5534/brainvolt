-- identity schema (docs/domain/schema-identity.md §2).
-- +goose Up
CREATE EXTENSION IF NOT EXISTS citext;
CREATE SCHEMA identity;

CREATE TABLE identity.users (
  id              uuid        PRIMARY KEY,                 -- UUIDv7, the user_id used everywhere
  firebase_uid    text        NOT NULL UNIQUE,             -- never changes, also not after account upgrade (D-013a)
  account_tier    text        NOT NULL DEFAULT 'guest'
                  CHECK (account_tier IN ('guest','unverified','full')),
  status          text        NOT NULL DEFAULT 'active'
                  CHECK (status IN ('active','suspended','deleted')),
  age_verified_at timestamptz NOT NULL,                    -- 13+ self-confirmation on first launch (D-020)
  upgraded_at     timestamptz,                             -- first time tier became 'full'
  abandoned_at    timestamptz,                             -- guest left via link conflict (D-015)
  deletion_requested_at timestamptz,                       -- user asked to delete account (D-022)
  deleted_at      timestamptz,                             -- PII scrubbed
  last_seen_at    timestamptz NOT NULL,
  created_at      timestamptz NOT NULL,
  updated_at      timestamptz NOT NULL,
  CHECK (status <> 'deleted' OR deleted_at IS NOT NULL)
);
CREATE INDEX users_guest_cleanup_idx ON identity.users (last_seen_at)
  WHERE account_tier = 'guest' AND status = 'active';

-- Mirror of Firebase-linked sign-in methods
CREATE TABLE identity.auth_identities (
  user_id         uuid        NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
  provider        text        NOT NULL CHECK (provider IN ('anonymous','google','apple','password')),
  provider_uid    text,                                    -- Google/Apple subject; NULL for anonymous & password
  email           citext,                                  -- NULL for anonymous; Apple may give a relay address
  email_verified  boolean     NOT NULL DEFAULT false,
  linked_at       timestamptz NOT NULL,
  last_sign_in_at timestamptz,
  PRIMARY KEY (user_id, provider)
);
CREATE UNIQUE INDEX auth_identities_provider_uid_uq
  ON identity.auth_identities (provider, provider_uid) WHERE provider_uid IS NOT NULL;
CREATE UNIQUE INDEX auth_identities_password_email_uq
  ON identity.auth_identities (email) WHERE provider = 'password';

-- Staff roles; 'player' is implicit for everyone
CREATE TABLE identity.user_roles (
  user_id     uuid        NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
  role        text        NOT NULL CHECK (role IN ('moderator','admin')),
  granted_by  uuid        REFERENCES identity.users(id),
  granted_at  timestamptz NOT NULL,
  PRIMARY KEY (user_id, role)
);

-- +goose Down
DROP SCHEMA identity CASCADE;
-- citext stays: other schemas (social, team, …) use it too.
