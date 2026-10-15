# Schema — content (categories, questions, translations, sources, verification)

- **Roadmap task:** 0.2.3
- **Status:** Accepted (2026-10-15)
- **Module / Postgres schema:** `content` (creator quizzes reuse these tables; quiz metadata lives in `creator`, 0.2.8)
- **Decisions applied:** D-001 (TR first), D-002 (TR + EN), D-005 (OpenTDB + own pool), D-006 (source & license mandatory), D-007 (ranked = verified only), D-023 (no unverified content in public play), D-024 (true/false not in ranked), D-025 (text-only)
- Conventions: same as `schema-identity.md` (UUIDv7 PKs, `timestamptz` UTC, text + CHECK enums).

---

## 1. Key design points

1. **Two-level taxonomy:** category → sub-category. Every question belongs to exactly one sub-category.
2. **Language-neutral core + per-locale text.** A question's structure (category, options, which option is correct, difficulty, source) is stored once; its wording is stored per locale (`tr`, `en`). The correct answer is a property of the *option*, not of the text, so translations can never disagree on it.
3. **Verification on two levels** (D-007):
   - **Question verification** — facts are right, exactly one option is correct, not ambiguous, not offensive.
   - **Translation verification** — the wording in that locale is accurate and natural.
   - A question can be used in **ranked** in locale *L* only if **the question is `verified` AND its *L* translation is `verified`**. This covers the OpenTDB case: an English original may be verified while its Turkish translation is still in review.
4. **Source and license are mandatory** (D-006): `source_id NOT NULL`; the import tool rejects anything without them.
5. **Questions are never hard-deleted** once used in a match (match history references them). They are *retired* instead.
6. **The correct option never leaves the server before the reveal.** Option order is shuffled per match by the server.

---

## 2. Tables

```sql
CREATE SCHEMA content;

-- ---------- Taxonomy ----------
CREATE TABLE content.categories (
  id          uuid        PRIMARY KEY,
  slug        text        NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9-]{2,40}$'),   -- e.g. 'history'
  icon_key    text        NOT NULL,                                           -- built-in icon
  sort_order  int         NOT NULL DEFAULT 0,
  is_active   boolean     NOT NULL DEFAULT true,
  created_at  timestamptz NOT NULL,
  updated_at  timestamptz NOT NULL
);

CREATE TABLE content.category_translations (
  category_id uuid NOT NULL REFERENCES content.categories(id) ON DELETE CASCADE,
  locale      text NOT NULL CHECK (locale IN ('tr','en')),
  name        text NOT NULL CHECK (char_length(name) BETWEEN 2 AND 40),
  description text,
  PRIMARY KEY (category_id, locale)
);

CREATE TABLE content.subcategories (
  id           uuid        PRIMARY KEY,
  category_id  uuid        NOT NULL REFERENCES content.categories(id),
  slug         text        NOT NULL CHECK (slug ~ '^[a-z0-9-]{2,40}$'),
  sort_order   int         NOT NULL DEFAULT 0,
  is_active    boolean     NOT NULL DEFAULT true,
  created_at   timestamptz NOT NULL,
  updated_at   timestamptz NOT NULL,
  UNIQUE (category_id, slug)
);

CREATE TABLE content.subcategory_translations (
  subcategory_id uuid NOT NULL REFERENCES content.subcategories(id) ON DELETE CASCADE,
  locale         text NOT NULL CHECK (locale IN ('tr','en')),
  name           text NOT NULL CHECK (char_length(name) BETWEEN 2 AND 40),
  description    text,
  PRIMARY KEY (subcategory_id, locale)
);

-- ---------- Sources & licenses (D-006) ----------
CREATE TABLE content.sources (
  id                uuid        PRIMARY KEY,
  kind              text        NOT NULL CHECK (kind IN ('opentdb','brainvolt_team','creator','partner')),
  name              text        NOT NULL,                  -- 'Open Trivia Database'
  url               text,                                  -- 'https://opentdb.com'
  license_code      text        NOT NULL,                  -- SPDX-style: 'CC-BY-SA-4.0', 'LicenseRef-BrainVolt', 'LicenseRef-BrainVolt-Creator-Terms'
  license_url       text,
  attribution_text  text        NOT NULL,                  -- shown on the in-app Credits screen
  share_alike       boolean     NOT NULL DEFAULT false,    -- true for CC BY-SA content: derivatives keep the same license
  notes             text,
  created_at        timestamptz NOT NULL
);

-- ---------- Questions ----------
CREATE TABLE content.questions (
  id                    uuid        PRIMARY KEY,
  subcategory_id        uuid        NOT NULL REFERENCES content.subcategories(id),
  question_type         text        NOT NULL CHECK (question_type IN ('multiple_choice','true_false')),
  difficulty            smallint    NOT NULL CHECK (difficulty BETWEEN 1 AND 5),  -- author/import estimate
  difficulty_calibrated numeric(3,2),                       -- 1.00–5.00, computed from answer stats (§5); NULL until enough data
  original_locale       text        NOT NULL CHECK (original_locale IN ('tr','en')),
  source_id             uuid        NOT NULL REFERENCES content.sources(id),       -- D-006
  source_ref            text,                               -- id/URL/hash inside the source, for traceability
  author_user_id        uuid        REFERENCES identity.users(id),                  -- set for creator & team-authored questions
  pool                  text        NOT NULL DEFAULT 'general'
                        CHECK (pool IN ('general','creator_only')),                 -- creator questions stay out of general matches unless promoted
  fingerprint           text        NOT NULL UNIQUE,        -- hash of normalised original prompt + sorted option texts; blocks duplicate imports
  verification_status   text        NOT NULL DEFAULT 'draft'
                        CHECK (verification_status IN ('draft','in_review','verified','rejected')),
  verified_by           uuid        REFERENCES identity.users(id),
  verified_at           timestamptz,
  rejection_reason      text,
  retired_at            timestamptz,                        -- no longer selectable; kept for history
  revision              int         NOT NULL DEFAULT 1,     -- bumped on every content edit (§4.3)
  created_at            timestamptz NOT NULL,
  updated_at            timestamptz NOT NULL,
  CHECK (verification_status <> 'verified' OR (verified_by IS NOT NULL AND verified_at IS NOT NULL)),
  CHECK (verification_status <> 'rejected' OR rejection_reason IS NOT NULL)
);
CREATE INDEX questions_selectable_idx ON content.questions (subcategory_id, difficulty)
  WHERE retired_at IS NULL AND pool = 'general';
CREATE INDEX questions_status_idx ON content.questions (verification_status) WHERE retired_at IS NULL;

CREATE TABLE content.question_translations (
  question_id          uuid        NOT NULL REFERENCES content.questions(id) ON DELETE CASCADE,
  locale               text        NOT NULL CHECK (locale IN ('tr','en')),
  prompt               text        NOT NULL CHECK (char_length(prompt) BETWEEN 5 AND 300),
  explanation          text        CHECK (char_length(explanation) <= 500),  -- optional, shown after reveal
  origin               text        NOT NULL CHECK (origin IN ('original','human_translation','machine_translation')),
  verification_status  text        NOT NULL DEFAULT 'draft'
                       CHECK (verification_status IN ('draft','in_review','verified','rejected')),
  verified_by          uuid        REFERENCES identity.users(id),
  verified_at          timestamptz,
  rejection_reason     text,
  created_at           timestamptz NOT NULL,
  updated_at           timestamptz NOT NULL,
  PRIMARY KEY (question_id, locale),
  CHECK (verification_status <> 'verified' OR (verified_by IS NOT NULL AND verified_at IS NOT NULL)),
  CHECK (origin <> 'machine_translation' OR verification_status <> 'verified' OR verified_by IS NOT NULL)
);

CREATE TABLE content.options (
  id           uuid     PRIMARY KEY,
  question_id  uuid     NOT NULL REFERENCES content.questions(id) ON DELETE CASCADE,
  position     smallint NOT NULL CHECK (position BETWEEN 1 AND 4),   -- canonical order; matches shuffle
  is_correct   boolean  NOT NULL,
  UNIQUE (question_id, position)
);
CREATE UNIQUE INDEX options_one_correct_uq ON content.options (question_id) WHERE is_correct;

CREATE TABLE content.option_translations (
  option_id  uuid NOT NULL REFERENCES content.options(id) ON DELETE CASCADE,
  locale     text NOT NULL CHECK (locale IN ('tr','en')),
  text       text NOT NULL CHECK (char_length(text) BETWEEN 1 AND 120),
  PRIMARY KEY (option_id, locale)
);

-- ---------- Revision snapshots (what players saw; review finding F1, 0.2.10) ----------
CREATE TABLE content.question_revisions (
  question_id  uuid        NOT NULL REFERENCES content.questions(id),
  revision     int         NOT NULL,
  snapshot     jsonb       NOT NULL,      -- all locales: prompt, explanation, option texts by option id, correct option id
  created_by   uuid        REFERENCES identity.users(id),
  created_at   timestamptz NOT NULL,
  PRIMARY KEY (question_id, revision)
);

-- ---------- Audit ----------
CREATE TABLE content.verification_events (
  id           uuid        PRIMARY KEY,
  question_id  uuid        NOT NULL REFERENCES content.questions(id),
  locale       text        CHECK (locale IN ('tr','en')),     -- NULL = question-level; set = translation-level
  from_status  text        NOT NULL,
  to_status    text        NOT NULL,
  actor_id     uuid        REFERENCES identity.users(id),     -- NULL = system (import, auto-retire)
  reason       text,
  revision     int         NOT NULL,
  created_at   timestamptz NOT NULL
);
CREATE INDEX verification_events_question_idx ON content.verification_events (question_id, created_at);

-- ---------- Question sets (Daily Challenge, Creator Quiz, tournaments, curated) ----------
CREATE TABLE content.question_sets (
  id          uuid        PRIMARY KEY,
  kind        text        NOT NULL CHECK (kind IN ('daily','creator','tournament','curated')),
  created_by  uuid        REFERENCES identity.users(id),
  created_at  timestamptz NOT NULL
);

CREATE TABLE content.question_set_items (
  set_id       uuid     NOT NULL REFERENCES content.question_sets(id) ON DELETE CASCADE,
  position     smallint NOT NULL CHECK (position >= 1),
  question_id  uuid     NOT NULL REFERENCES content.questions(id),
  PRIMARY KEY (set_id, position),
  UNIQUE (set_id, question_id)
);

CREATE TABLE content.daily_sets (
  day          date        PRIMARY KEY,                     -- calendar day in Europe/Istanbul
  set_id       uuid        NOT NULL UNIQUE REFERENCES content.question_sets(id),
  published_at timestamptz NOT NULL
);

-- ---------- Aggregated answer statistics (§5) ----------
CREATE TABLE content.question_stats (
  question_id      uuid        NOT NULL REFERENCES content.questions(id),
  locale           text        NOT NULL CHECK (locale IN ('tr','en')),
  times_asked      bigint      NOT NULL DEFAULT 0,
  times_correct    bigint      NOT NULL DEFAULT 0,
  times_timed_out  bigint      NOT NULL DEFAULT 0,
  sum_response_ms  bigint      NOT NULL DEFAULT 0,          -- correct answers only; avg = sum / times_correct
  last_asked_at    timestamptz,
  PRIMARY KEY (question_id, locale)
);
```

### 2.1 Rules enforced in the service layer (cannot be expressed as simple constraints)
| Rule | Where checked |
|---|---|
| `multiple_choice` has exactly **4** options, `true_false` exactly **2** (positions 1–2 = True/False / Doğru/Yanlış) | create/update question |
| Every option has a translation for every locale the question has | create/update translation |
| A question can only become `verified` if it has ≥1 translation and its own facts were reviewed; its **original-locale** translation is verified together with it | moderation review action |
| A machine translation can be selected for casual play only after **human** verification (never shown raw) | selection (§3) |
| `share_alike` sources: all derivatives (our TR translations of OpenTDB) inherit the source's license; Credits screen lists them | import + credits endpoint |
| Retiring a question removes it from all *future* daily/curated sets not yet published | retire action |

---

## 3. Eligibility & selection (contract for `game`)

`content` exposes one selection function to `game`; it returns question IDs plus the localized text for the participants' locales. The **eligibility** rules are fixed here; the **selection strategy** per match type / category mode (counts, difficulty curve, category mix) is fixed in 0.3.5 and 0.3.7.

| Context | Question must be | Translation (each locale in the match) must be |
|---|---|---|
| **Ranked** (D-007, D-024) | `verified`, not retired, `pool = general`, **`question_type = multiple_choice` only** | `verified` |
| Casual — every non-ranked public mode (Quick Duel, Sudden Death, Marathon, Blind Challenge, Category Master, Mixed Battle, matchmade or challenge) | `verified`, not retired, `pool = general`; `true_false` allowed (D-024) | `verified` |
| Daily Challenge | `verified`, in that day's set, `multiple_choice` only (it feeds public leaderboards) | `verified` in **both** `tr` and `en` (everyone plays the identical set) |
| Creator Quiz — **published** (verified by review) | in the quiz's set, `verified` | `verified` in the quiz's locale |
| Creator Quiz — **unverified** (D-023) | in the quiz's set, any status except `rejected` | exists in the quiz's locale |

**D-023:** no unverified question ever appears in public, matchmade or ranked play. The **only** exception is an *unverified* Creator Quiz, which may be played **solely in a private/custom room**, never via matchmaking, never rated, never counted for public leaderboards, Season Points or quests. Exact room-entry rule (friends only vs. anyone with the host's invite code): 0.3.7.

**Mixed-locale matches:** a Turkish-speaking and an English-speaking player can meet. Each participant sees their own locale's text; therefore every selected question must satisfy the translation rule **for every locale present in the match**. If the pool is too small, matchmaking prefers same-locale opponents (Phase 5).

**No-repeat rule:** recently seen questions per user are excluded (window and storage in `redis-keys.md`, 0.2.9); fallback to least-recently-seen when the pool is exhausted.

---

## 4. Content lifecycle

### 4.1 Status machine (question and translation use the same states)
```
draft ──submit──▶ in_review ──approve──▶ verified
   ▲                  │                     │
   └────── edit ◀─────┴──reject──▶ rejected │
                                            │
verified ──content edit──▶ in_review (revision+1)
verified ──report upheld──▶ in_review or rejected (moderation, Phase 8)
any ──retire──▶ (retired_at set; status kept for history)
```
Every transition writes a row to `content.verification_events`.

### 4.2 Who can do what
| Action | player | creator (Phase 8) | moderator | admin |
|---|:-:|:-:|:-:|:-:|
| Create draft | — | own creator-quiz questions | ✓ | ✓ |
| Submit for review | — | ✓ (own) | ✓ | ✓ |
| Approve / reject | — | — | ✓ | ✓ |
| Edit verified question | — | — | ✓ (back to review) | ✓ |
| Retire / promote `creator_only` → `general` | — | — | — | ✓ |

**Minimal review tool in Phase 2** (needed so ranked has content, as agreed earlier): admin-only endpoints to list `in_review` items and approve/reject them. The full moderation console comes in Phase 8.

### 4.3 Editing a used question
Text/option edits bump `revision` and write a full snapshot to `content.question_revisions`. **Once a question has been used in any match, its option rows are immutable** (no delete, no change of `is_correct`): `game.answers` points at them. Changing which option is correct means **retiring** the question and creating a new one; wording fixes are translation edits. Matches store the **revision** they used (0.2.4), so history and reports refer to exactly what the player saw. A fact change on a verified question puts it back to `in_review` and removes it from ranked until re-approved.

---

## 5. Answer statistics & calibrated difficulty
- `game` enqueues a stats job when a match completes (game → content is an allowed edge); the job increments `content.question_stats` per (question, locale).
- `difficulty_calibrated` is recomputed by a worker job once a question has ≥ 50 answers (correct rate + average response time → 1.00–5.00). Formula fixed in 0.3.4.
- Signals for moderation: correct rate < 5 % or > 98 % after ≥ 100 answers flags the question for review (possible wrong key or too easy).

---

## 6. OpenTDB import (D-005, D-006)

| Step | Detail |
|---|---|
| 1. Snapshot | Raw OpenTDB questions saved unchanged in `content/opentdb/raw/` (English, with their category/type/difficulty). |
| 2. Suitability check | Script + human pass: drop questions that are culturally irrelevant to a TR audience, time-sensitive ("current" facts), offensive, or ambiguous. Kept list recorded in `content/opentdb/selected.*`. |
| 3. Mapping | OpenTDB category → BrainVolt sub-category; `multiple` → `multiple_choice`, `boolean` → `true_false`; difficulty easy/medium/hard → 2/3/4. |
| 4. Translation | TR text added in `content/opentdb/tr/` (`origin = human_translation` or `machine_translation` + human edit). |
| 5. Import | `cmd/seed` creates questions with `source = OpenTDB`, `license_code = CC-BY-SA-4.0`, `share_alike = true`, `source_ref` = hash of the original question; EN translation `origin = original`, TR translation per step 4; all start as `in_review`. |
| 6. Review | Moderators verify question + EN, then TR. Only then eligible for matches. |

**License obligations (CC BY-SA 4.0):** credit "Open Trivia Database (opentdb.com), CC BY-SA 4.0" on an in-app **Credits** screen and in `content/SOURCES.md`; mark that questions were translated/modified; our Turkish translations of OpenTDB questions are themselves CC BY-SA 4.0. This applies only to that question content, not to app code or our own questions.

---

## 7. Resolved points
| # | Decision |
|---|----------|
| D-023 | No unverified questions in public, casual or ranked play. Unverified Creator Quiz sets only in private/custom rooms among friends |
| D-024 | True/false questions excluded from Ranked (and from the Daily Challenge, which feeds public leaderboards); allowed in casual/quick modes |
| D-025 | Text-only questions at launch; image/audio deferred to a future media pipeline |
