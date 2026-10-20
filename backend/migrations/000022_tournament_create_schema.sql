-- Tournaments (docs/domain/schema-platform.md §3, D-031, D-090): admin-made
-- single-elimination brackets of users, couples or teams. A bracket match
-- records who has shown up (present) until its ready window closes.
-- +goose Up
CREATE SCHEMA tournament;

CREATE TABLE tournament.tournaments (
  id                     uuid        PRIMARY KEY,
  status                 text        NOT NULL CHECK (status IN ('draft','registration','seeding','running','finished','cancelled')),
  bracket_type           text        NOT NULL CHECK (bracket_type IN ('single_elimination','double_elimination','swiss')),
  entity_kind            text        NOT NULL CHECK (entity_kind IN ('user','team','couple')),
  match_config           jsonb       NOT NULL,
  is_rated               boolean     NOT NULL DEFAULT false,
  min_entrants           int         NOT NULL CHECK (min_entrants >= 2),
  max_entrants           int         NOT NULL CHECK (max_entrants >= min_entrants),
  entry_requirements     jsonb       NOT NULL DEFAULT '{}',
  registration_opens_at  timestamptz NOT NULL,
  registration_closes_at timestamptz NOT NULL,
  starts_at              timestamptz NOT NULL,
  round_ready_window_s   int         NOT NULL DEFAULT 300 CHECK (round_ready_window_s BETWEEN 60 AND 3600),
  rewards                jsonb       NOT NULL DEFAULT '{}',
  created_by             uuid        NOT NULL REFERENCES identity.users(id),
  created_at             timestamptz NOT NULL,
  finished_at            timestamptz,
  CHECK (registration_opens_at < registration_closes_at AND registration_closes_at <= starts_at)
);
CREATE INDEX tournaments_status_idx ON tournament.tournaments (status, starts_at);

CREATE TABLE tournament.tournament_translations (
  tournament_id uuid NOT NULL REFERENCES tournament.tournaments(id) ON DELETE CASCADE,
  locale        text NOT NULL CHECK (locale IN ('tr','en')),
  name          text NOT NULL CHECK (char_length(name) BETWEEN 3 AND 60),
  description   text CHECK (char_length(description) <= 500),
  PRIMARY KEY (tournament_id, locale)
);

CREATE TABLE tournament.entries (
  id             uuid        PRIMARY KEY,
  tournament_id  uuid        NOT NULL REFERENCES tournament.tournaments(id),
  entity_kind    text        NOT NULL CHECK (entity_kind IN ('user','team','couple')),
  entity_id      uuid        NOT NULL,
  lineup         uuid[]      NOT NULL,
  seed           int,
  status         text        NOT NULL DEFAULT 'registered'
                 CHECK (status IN ('registered','checked_in','active','eliminated','withdrawn','disqualified','winner')),
  final_place    int,
  registered_at  timestamptz NOT NULL,
  UNIQUE (tournament_id, entity_kind, entity_id)
);
CREATE INDEX entries_entity_idx ON tournament.entries (entity_kind, entity_id);
CREATE INDEX entries_lineup_idx ON tournament.entries USING gin (lineup);

CREATE TABLE tournament.bracket_matches (
  tournament_id   uuid        NOT NULL REFERENCES tournament.tournaments(id),
  round_no        smallint    NOT NULL,
  slot_no         smallint    NOT NULL,
  bracket_side    text        NOT NULL DEFAULT 'main' CHECK (bracket_side IN ('main','losers','final')),
  entry_a_id      uuid        REFERENCES tournament.entries(id),
  entry_b_id      uuid        REFERENCES tournament.entries(id),
  match_id        uuid,
  winner_entry_id uuid        REFERENCES tournament.entries(id),
  status          text        NOT NULL CHECK (status IN ('pending','ready','in_progress','done','walkover')),
  scheduled_at    timestamptz,
  ready_until     timestamptz,
  present         uuid[]      NOT NULL DEFAULT '{}',
  PRIMARY KEY (tournament_id, bracket_side, round_no, slot_no)
);
CREATE UNIQUE INDEX bracket_matches_match_uq ON tournament.bracket_matches (match_id) WHERE match_id IS NOT NULL;
CREATE INDEX bracket_matches_ready_idx ON tournament.bracket_matches (ready_until) WHERE status = 'ready';

-- +goose Down
DROP SCHEMA tournament CASCADE;
