-- Team chat (docs/domain/schema-platform.md §2, D-031, D-088): channels of
-- persistent teams and couples, messages and read cursors. match_side
-- channels are kept in the schema for in-match chat.
-- +goose Up
CREATE SCHEMA chat;

CREATE TABLE chat.channels (
  id          uuid        PRIMARY KEY,
  kind        text        NOT NULL CHECK (kind IN ('team','couple','match_side')),
  team_id     uuid,
  couple_id   uuid,
  match_id    uuid,
  side_no     smallint,
  created_at  timestamptz NOT NULL,
  closed_at   timestamptz,
  CHECK ((kind = 'team') = (team_id IS NOT NULL)),
  CHECK ((kind = 'couple') = (couple_id IS NOT NULL)),
  CHECK ((kind = 'match_side') = (match_id IS NOT NULL AND side_no IS NOT NULL))
);
CREATE UNIQUE INDEX channels_team_uq ON chat.channels (team_id) WHERE kind = 'team';
CREATE UNIQUE INDEX channels_couple_uq ON chat.channels (couple_id) WHERE kind = 'couple';
CREATE UNIQUE INDEX channels_match_side_uq ON chat.channels (match_id, side_no) WHERE kind = 'match_side';

CREATE TABLE chat.messages (
  id            uuid        PRIMARY KEY,
  channel_id    uuid        NOT NULL REFERENCES chat.channels(id),
  sender_id     uuid        NOT NULL REFERENCES identity.users(id),
  kind          text        NOT NULL CHECK (kind IN ('text','quick')),
  body          text        CHECK (char_length(body) BETWEEN 1 AND 300),
  quick_id      text,
  filter_result text        NOT NULL DEFAULT 'clean' CHECK (filter_result IN ('clean','masked','blocked')),
  created_at    timestamptz NOT NULL,
  removed_at    timestamptz,
  removed_by    uuid        REFERENCES identity.users(id),
  CHECK ((kind = 'text') = (body IS NOT NULL) OR removed_at IS NOT NULL),
  CHECK ((kind = 'quick') = (quick_id IS NOT NULL))
);
CREATE INDEX messages_channel_idx ON chat.messages (channel_id, id DESC);
CREATE INDEX messages_sender_idx ON chat.messages (sender_id, created_at DESC);

CREATE TABLE chat.read_cursors (
  channel_id    uuid NOT NULL REFERENCES chat.channels(id) ON DELETE CASCADE,
  user_id       uuid NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
  last_read_id  uuid NOT NULL,
  PRIMARY KEY (channel_id, user_id)
);

-- +goose Down
DROP SCHEMA chat CASCADE;
