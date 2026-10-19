-- moderation schema (docs/domain/schema-moderation.md §2): reports grouped
-- into cases, the immutable action log, sanctions, appeals, cheat signals,
-- installation links and the blocked-terms filter.
-- +goose Up
CREATE SCHEMA moderation;

-- ---------- Cases (grouping) ----------
CREATE TABLE moderation.cases (
  id             uuid        PRIMARY KEY,
  target_kind    text        NOT NULL CHECK (target_kind IN ('question','question_translation','user','team','couple',
                                                             'chat_message','creator_quiz')),
  target_id      uuid        NOT NULL,
  target_locale  text        CHECK (target_locale IN ('tr','en')),          -- for question_translation
  status         text        NOT NULL DEFAULT 'open' CHECK (status IN ('open','in_review','resolved')),
  priority       int         NOT NULL DEFAULT 0,       -- recomputed: report count × reporter trust + signal severity (P)
  report_count   int         NOT NULL DEFAULT 0,
  signal_count   int         NOT NULL DEFAULT 0,
  assigned_to    uuid        REFERENCES identity.users(id),
  resolution     text        CHECK (resolution IN ('upheld','dismissed','duplicate')),
  resolution_note text,
  opened_at      timestamptz NOT NULL,
  resolved_at    timestamptz,
  CHECK (status <> 'resolved' OR resolution IS NOT NULL)
);
CREATE UNIQUE INDEX cases_one_open_per_target_uq
  ON moderation.cases (target_kind, target_id, COALESCE(target_locale,'')) WHERE status <> 'resolved';
CREATE INDEX cases_queue_idx ON moderation.cases (status, priority DESC, opened_at) WHERE status <> 'resolved';

-- ---------- Reports ----------
CREATE TABLE moderation.reports (
  id             uuid        PRIMARY KEY,
  case_id        uuid        NOT NULL REFERENCES moderation.cases(id),
  reporter_id    uuid        NOT NULL REFERENCES identity.users(id),
  reporter_tier  text        NOT NULL CHECK (reporter_tier IN ('guest','unverified','full')),  -- lower weight for restricted (P)
  target_kind    text        NOT NULL,
  target_id      uuid        NOT NULL,
  target_locale  text,
  match_id       uuid,                               -- where it happened, if in a match
  reason         text        NOT NULL CHECK (reason IN (
                   -- questions
                   'wrong_answer','ambiguous','outdated','typo_or_translation','offensive_content',
                   -- users / teams / names / chat
                   'cheating','offensive_name','harassment','spam','impersonation',
                   'other')),
  comment        text        CHECK (char_length(comment) <= 500),
  created_at     timestamptz NOT NULL
);
-- one report per reporter per target (re-reporting the same thing has no extra effect)
CREATE UNIQUE INDEX reports_once_uq ON moderation.reports (reporter_id, target_kind, target_id, COALESCE(target_locale,''));
CREATE INDEX reports_case_idx ON moderation.reports (case_id);

-- ---------- Actions (immutable audit log) ----------
CREATE TABLE moderation.actions (
  id            uuid        PRIMARY KEY,
  case_id       uuid        REFERENCES moderation.cases(id),
  actor_kind    text        NOT NULL CHECK (actor_kind IN ('moderator','admin','system')),
  actor_id      uuid        REFERENCES identity.users(id),               -- NULL for system
  action        text        NOT NULL CHECK (action IN (
                  'question_send_to_review','question_approve','question_reject','question_edit','question_retire',
                  'translation_approve','translation_reject',
                  'user_warn','user_sanction','sanction_lift','user_rename','team_rename','team_disband',
                  'couple_dissolve','chat_message_remove','creator_quiz_unpublish',
                  'match_void','case_dismiss','case_reopen','appeal_decide')),
  target_kind   text        NOT NULL,
  target_id     uuid        NOT NULL,
  payload       jsonb       NOT NULL DEFAULT '{}',   -- before/after values, reason codes
  created_at    timestamptz NOT NULL
);
CREATE INDEX actions_target_idx ON moderation.actions (target_kind, target_id, created_at);
CREATE INDEX actions_actor_idx  ON moderation.actions (actor_id, created_at);

-- ---------- Sanctions ----------
CREATE TABLE moderation.sanctions (
  id            uuid        PRIMARY KEY,
  user_id       uuid        NOT NULL REFERENCES identity.users(id),
  kind          text        NOT NULL CHECK (kind IN ('warning','chat_mute','ranked_ban','matchmaking_ban',
                                                     'social_ban','suspension')),
  reason_code   text        NOT NULL,                -- same vocabulary as report reasons + 'cheating_confirmed'
  case_id       uuid        REFERENCES moderation.cases(id),
  starts_at     timestamptz NOT NULL,
  ends_at       timestamptz,                         -- NULL = permanent
  lifted_at     timestamptz,
  lifted_by     uuid        REFERENCES identity.users(id),
  created_by    uuid        REFERENCES identity.users(id),    -- NULL = system
  created_at    timestamptz NOT NULL,
  CHECK (kind <> 'warning' OR ends_at IS NULL)
);
CREATE INDEX sanctions_active_idx ON moderation.sanctions (user_id) WHERE lifted_at IS NULL;

-- ---------- Appeals ----------
CREATE TABLE moderation.appeals (
  id            uuid        PRIMARY KEY,
  sanction_id   uuid        NOT NULL UNIQUE REFERENCES moderation.sanctions(id),   -- one appeal per sanction
  user_id       uuid        NOT NULL REFERENCES identity.users(id),
  text          text        NOT NULL CHECK (char_length(text) BETWEEN 10 AND 1000),
  status        text        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','accepted','rejected')),
  decided_by    uuid        REFERENCES identity.users(id),           -- must differ from sanction's created_by (§5)
  decided_at    timestamptz,
  decision_note text,
  created_at    timestamptz NOT NULL
);

-- ---------- Cheat signals ----------
CREATE TABLE moderation.cheat_signals (
  id           uuid        PRIMARY KEY,
  user_id      uuid        NOT NULL REFERENCES identity.users(id),
  match_id     uuid,
  kind         text        NOT NULL CHECK (kind IN (
                 'superhuman_response_time',   -- correct answers faster than a reading-time floor for the prompt
                 'consistent_fast_streak',      -- statistically improbable speed+accuracy over many rounds
                 'rejected_submission_burst',   -- many late/duplicate/not-your-turn submissions (game, schema-game §2.1)
                 'protocol_violation',          -- malformed or out-of-sequence WS messages
                 'win_trading',                 -- same opponents repeatedly, alternating forfeits/losses
                 'team_collusion',              -- side members feeding results / intentional losses
                 'reconnect_abuse',             -- disconnect patterns used to dodge losses
                 'multi_account',               -- several accounts on one installation trading wins
                 'abnormal_rating_gain')),
  severity     smallint    NOT NULL CHECK (severity BETWEEN 1 AND 5),
  details      jsonb       NOT NULL,             -- measured values + threshold used, for reviewer
  case_id      uuid        REFERENCES moderation.cases(id),
  detected_at  timestamptz NOT NULL
);
CREATE INDEX cheat_signals_user_idx ON moderation.cheat_signals (user_id, detected_at DESC);

-- ---------- Installation ↔ account links (multi-account detection) ----------
CREATE TABLE moderation.installation_links (
  installation_id  text        NOT NULL,          -- random app-generated id (schema-identity §4); no hardware ids
  user_id          uuid        NOT NULL REFERENCES identity.users(id),
  first_seen_at    timestamptz NOT NULL,
  last_seen_at     timestamptz NOT NULL,
  PRIMARY KEY (installation_id, user_id)
);
CREATE INDEX installation_links_user_idx ON moderation.installation_links (user_id);
-- Filled from identity.SessionStarted (emitted by POST /v1/session with the app's installation_id).

-- ---------- Name / text filter terms ----------
CREATE TABLE moderation.blocked_terms (
  id          uuid        PRIMARY KEY,
  term        text        NOT NULL,               -- normalised (lowercase, Turkish-aware folding: İ/ı, ş, ğ …)
  locale      text        CHECK (locale IN ('tr','en')),   -- NULL = all
  match_kind  text        NOT NULL CHECK (match_kind IN ('exact','contains')),
  applies_to  text[]      NOT NULL,               -- {username, display_name, team_name, couple_name, chat, creator_quiz}
  severity    smallint    NOT NULL CHECK (severity BETWEEN 1 AND 3),   -- 1 flag, 2 block, 3 block + signal
  created_by  uuid        REFERENCES identity.users(id),
  created_at  timestamptz NOT NULL,
  UNIQUE NULLS NOT DISTINCT (term, locale, match_kind)   -- locale NULL = all; must still be unique (F3)
);

-- +goose Down
DROP SCHEMA moderation CASCADE;
