-- BrainVolt Plus (D-104): store events relayed by RevenueCat's webhook,
-- kept for audit and replay and made idempotent by their event id
-- (schema-platform.md §4). Entitlements are written from them.
-- +goose Up
CREATE TABLE commerce.store_events (
  id            uuid        PRIMARY KEY,
  store         text        NOT NULL,            -- 'revenuecat'
  event_id      text        NOT NULL,            -- the relay's event id
  kind          text        NOT NULL,            -- INITIAL_PURCHASE, RENEWAL, …
  user_id       uuid,                            -- resolved app user, when known
  payload       jsonb       NOT NULL,
  received_at   timestamptz NOT NULL,
  processed_at  timestamptz,
  UNIQUE (store, event_id)
);
CREATE INDEX store_events_user_idx ON commerce.store_events (user_id, received_at);

-- +goose Down
DROP TABLE commerce.store_events;
