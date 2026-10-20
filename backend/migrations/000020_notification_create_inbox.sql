-- notification preferences, quiet hours, the in-app inbox and push
-- deliveries (docs/domain/schema-platform.md §1, D-031).
-- +goose Up
CREATE TABLE notification.preferences (
  user_id     uuid        NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
  category    text        NOT NULL CHECK (category IN (
                'challenge','friend','match_found','team','chat','season','quest',
                'tournament','moderation','product_news')),
  push        boolean     NOT NULL,
  in_app      boolean     NOT NULL DEFAULT true,
  updated_at  timestamptz NOT NULL,
  PRIMARY KEY (user_id, category)
);

CREATE TABLE notification.quiet_hours (
  user_id            uuid     PRIMARY KEY REFERENCES identity.users(id) ON DELETE CASCADE,
  enabled            boolean  NOT NULL DEFAULT false,
  start_local        time     NOT NULL DEFAULT '23:00',
  end_local          time     NOT NULL DEFAULT '08:00',
  utc_offset_minutes smallint NOT NULL
);

CREATE TABLE notification.inbox (
  id          uuid        PRIMARY KEY,
  user_id     uuid        NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
  category    text        NOT NULL,
  kind        text        NOT NULL,
  params      jsonb       NOT NULL,
  deep_link   text,
  dedupe_key  text,
  created_at  timestamptz NOT NULL,
  read_at     timestamptz,
  expires_at  timestamptz NOT NULL
);
CREATE UNIQUE INDEX inbox_dedupe_uq ON notification.inbox (user_id, dedupe_key) WHERE dedupe_key IS NOT NULL AND read_at IS NULL;
CREATE INDEX inbox_user_idx ON notification.inbox (user_id, created_at DESC);

CREATE TABLE notification.push_deliveries (
  id              uuid        PRIMARY KEY,
  inbox_id        uuid        NOT NULL REFERENCES notification.inbox(id) ON DELETE CASCADE,
  device_id       uuid        NOT NULL REFERENCES notification.devices(id) ON DELETE CASCADE,
  status          text        NOT NULL CHECK (status IN ('sent','failed','token_invalid','suppressed')),
  fcm_message_id  text,
  error           text,
  attempted_at    timestamptz NOT NULL
);
CREATE INDEX push_deliveries_time_idx ON notification.push_deliveries (attempted_at);

-- +goose Down
DROP TABLE notification.push_deliveries;
DROP TABLE notification.inbox;
DROP TABLE notification.quiet_hours;
DROP TABLE notification.preferences;
