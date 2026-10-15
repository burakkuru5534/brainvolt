# Schema — moderation & anti-cheat (reports, cases, actions, sanctions, appeals, cheat signals)

- **Roadmap task:** 0.2.7
- **Status:** Accepted (2026-10-15)
- **Module / Postgres schema:** `moderation`
- **Decisions applied:** D-006 / D-007 / D-023 (content verification), D-016 (restricted accounts), D-026 (answer detail kept 12 months)
- Full moderation console & automated detection rules ship in **Phase 8**; the minimal question-review endpoints needed earlier are in Phase 2 (`schema-content.md` §4.2). Values marked **(P)** were proposed and accepted (D-030, §8).
- Conventions: as in `schema-identity.md`.

---

## 1. Concepts

```
report (from a player) ──┐
                         ├──▶ case (one open case per target) ──▶ action(s) by moderator/system ──▶ sanction / content change / void
cheat signal (automatic) ┘                                                     │
                                                                               └──▶ appeal (by sanctioned user) ──▶ decision
```

- **Report:** a player's complaint. Many reports on the same target are grouped into one **case**.
- **Cheat signal:** an automatic detection; also attached to a case for the affected user.
- **Action:** every decision or change, by a moderator, admin or the system — immutable audit log.
- **Sanction:** a restriction on a user, with start/end.
- **Appeal:** one request per sanction for a second look.

Who acts: `moderator` and `admin` roles (`identity.user_roles`). Admin-only actions are marked in §5.

---

## 2. Tables

```sql
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
```

---

## 3. How other modules use moderation

`moderation` is a downstream module (layer 5). Lower modules **never import it**; instead they declare small interfaces that `moderation` implements and `internal/app` wires (same pattern as `RatingReader` / `ActivityReader`):

| Interface (declared in) | Purpose |
|---|---|
| `SanctionChecker` (matchmaking, lobby, chat, social, team) | "may this user queue / play ranked / chat / befriend now?" — backed by a Redis cache of active sanctions, refreshed on every sanction change |
| `TextFilter` (profile, team, chat, creator) | check a name or message against `blocked_terms` |
| `QuestionReviewer` (content) | content asks for review / receives approve-reject decisions (moderation → content calls are normal downward calls) |

Effects of sanction kinds:
| Kind | Effect |
|---|---|
| `warning` | notice only |
| `chat_mute` | cannot send chat messages |
| `ranked_ban` | cannot queue or be challenged into `ranked` |
| `matchmaking_ban` | cannot use any matchmaking queue (private rooms still allowed) |
| `social_ban` | cannot send friend requests, challenges, team/couple invites |
| `suspension` | `identity.users.status = 'suspended'` (schema-identity §5.5): can only view profile & sanction notice |

---

## 4. Anti-cheat pipeline

| Stage | Where | What |
|---|---|---|
| **Prevent** | game (live) | Server authority: correct option never sent before reveal; server timestamps; one accepted answer per round; turn checks; deadline + grace. Most cheats are impossible by design. |
| **Detect live** | game owner | Cheap per-answer checks (response time below the reading-time floor (P), rejected-submission counters, protocol violations) → signal written via an event, match continues. |
| **Detect after match** | worker, on `game.MatchCompleted` | Statistical checks across the match and the user's recent history (consistent superhuman speed, win trading between the same accounts, reconnect patterns, rating jumps, multi-account via `installation_links`). |
| **Decide** | moderator | Signals open/raise a case; a human confirms before any sanction for cheating (P). |
| **Correct** | moderation → game / ranking | `match_void` (admin) → `game` sets the match as voided and emits `game.MatchVoided` → ranking removes that match's SP / leaderboard points (compensating ledger rows); **ratings are not recomputed retroactively** — the cheater gets a `ranked_ban` and their season standing is reset; affected honest opponents keep their results (P). |

Signals never change a match result automatically.

`game` schema addition (applied to `schema-game.md` in this task): `game.matches.voided_at timestamptz`, `void_reason text`, `voided_by uuid`; event `game.MatchVoided`.

---

## 5. Rules
- A moderator cannot act on cases about themselves, and an appeal must be decided by someone **other** than the sanction's creator.
- Admin-only actions: `match_void`, `question_retire`, permanent sanctions, `team_disband`, promoting creator questions (content §4.2).
- Every action writes `moderation.actions` in the same transaction as its effect.
- Reports: max **20 per user per day** (P); restricted accounts may report, with lower priority weight (P).
- Question cases from reports with reason `wrong_answer` reaching **3 reports** (P) automatically move the question to `in_review` (removed from ranked until re-approved) and log a `system` action.
- Reporters get a notification when their report's case is resolved as upheld (Phase 9), without details about sanctions on other users.

---

## 6. Events
| Event | Producer | Consumers |
|---|---|---|
| `moderation.ReportFiled` | moderation | notification (staff alert when priority high) |
| `moderation.SanctionApplied` / `SanctionLifted` | moderation | identity (suspension), matchmaking, lobby, chat, social, team, notification |
| `moderation.CaseResolved` | moderation | notification (reporter feedback) |
| `game.MatchVoided` (new, produced by game on admin request) | game | ranking, progression, tournament |
| `identity.SessionStarted` (new) | identity | moderation (installation links) |

## 7. Retention (D-030)
| Data | Keep |
|---|---|
| Cases, actions, sanctions, appeals | **3 years** after resolution / sanction end (P); permanent bans kept while the ban applies |
| Reports' free-text `comment` | **12 months** (P), then blanked |
| Cheat signals | **12 months** (P), matching answer-detail retention (D-026) |
| Installation links | **12 months** after `last_seen_at` (P) |
| After account deletion | records stay linked to the anonymised `user_id` (no personal data), so bans and audit remain valid |

## 8. Accepted proposals (D-030)
1. Reports: max 20 per user per day; guests/unverified may report with lower weight.
2. 3 `wrong_answer` reports → question automatically pulled back to review (out of ranked).
3. Cheating sanctions always require a human decision; signals alone never sanction.
4. Voided matches: SP & leaderboard points removed, ratings **not** recomputed retroactively, cheater gets ranked ban + season standing reset, honest opponents keep their results.
5. Reading-time floor for "superhuman" signals: computed per prompt length (formula in Phase 8 with real data; initial floor 300 ms + 15 ms per character).
6. Retention durations in §7.
