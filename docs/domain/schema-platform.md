# Schema — platform (notifications, team chat, tournaments, commerce, creator quizzes, daily challenge)

- **Roadmap task:** 0.2.8
- **Status:** Accepted (2026-10-15)
- **Modules / Postgres schemas:** `notification`, `chat`, `tournament`, `commerce`, `creator`, plus the Daily Challenge attempt table in `game`
- **Decisions applied:** D-002 (TR/EN), D-016 / D-016a (restricted accounts), D-020 (13+), D-021 (preset visuals; cosmetics in Phase 9), D-023 (unverified creator quizzes: private rooms, friends of host), D-024 (no true/false in daily), D-027, D-029, D-030
- Most of these features are built in **Phase 9** (Creator Quiz and Daily Challenge in Phase 8); the schema is fixed now so earlier phases leave room for them. Values marked **(P)** were proposed and accepted (D-031, §8).
- **Guiding rule for all monetization: nothing that can be bought or watched changes a match outcome, rating, SP or leaderboard position.**
- Conventions: as in `schema-identity.md`.

---

## 1. `notification`

`notification.devices` is already defined in `schema-identity.md` §4.

```sql
-- Per-user, per-category channel preferences
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
-- Defaults (rows created lazily): all push ON except 'product_news' (marketing) = OFF until the user opts in (explicit consent, KVKK/GDPR).
-- 'moderation' notices cannot be turned off in-app.

-- Quiet hours (no push; in-app still delivered)
CREATE TABLE notification.quiet_hours (
  user_id     uuid     PRIMARY KEY REFERENCES identity.users(id) ON DELETE CASCADE,
  enabled     boolean  NOT NULL DEFAULT false,
  start_local time     NOT NULL DEFAULT '23:00',
  end_local   time     NOT NULL DEFAULT '08:00',
  utc_offset_minutes smallint NOT NULL              -- reported by the app; updated when it changes
);

-- In-app inbox (also the source for push)
CREATE TABLE notification.inbox (
  id          uuid        PRIMARY KEY,
  user_id     uuid        NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
  category    text        NOT NULL,
  kind        text        NOT NULL,                  -- template key, e.g. 'challenge.received'
  params      jsonb       NOT NULL,                  -- values for the TR/EN template (rendered on device & for push)
  deep_link   text,                                  -- app route, e.g. '/challenge/<id>'
  dedupe_key  text,                                  -- collapses repeats, e.g. one 'friend.request' per sender
  created_at  timestamptz NOT NULL,
  read_at     timestamptz,
  expires_at  timestamptz NOT NULL                   -- 30 days (P)
);
CREATE UNIQUE INDEX inbox_dedupe_uq ON notification.inbox (user_id, dedupe_key) WHERE dedupe_key IS NOT NULL AND read_at IS NULL;
CREATE INDEX inbox_user_idx ON notification.inbox (user_id, created_at DESC);

-- Push delivery attempts (short-lived, for debugging & token cleanup)
CREATE TABLE notification.push_deliveries (
  id              uuid        PRIMARY KEY,
  inbox_id        uuid        NOT NULL REFERENCES notification.inbox(id) ON DELETE CASCADE,
  device_id       uuid        NOT NULL REFERENCES notification.devices(id) ON DELETE CASCADE,
  status          text        NOT NULL CHECK (status IN ('sent','failed','token_invalid','suppressed')),
  fcm_message_id  text,
  error           text,
  attempted_at    timestamptz NOT NULL
);
CREATE INDEX push_deliveries_time_idx ON notification.push_deliveries (attempted_at);   -- purged after 30 days (P)
```
Push texts use templates in both languages, chosen by the device's `locale`. Rate limit: max **10 push per user per hour**, excess goes to inbox only (P).

---

## 2. `chat` (team chat)

```sql
CREATE SCHEMA chat;

CREATE TABLE chat.channels (
  id          uuid        PRIMARY KEY,
  kind        text        NOT NULL CHECK (kind IN ('team','match_side')),
  team_id     uuid,                                  -- kind = team
  match_id    uuid,                                  -- kind = match_side (plain ids, owned upstream)
  side_no     smallint,
  created_at  timestamptz NOT NULL,
  closed_at   timestamptz,
  CHECK ((kind = 'team') = (team_id IS NOT NULL)),
  CHECK ((kind = 'match_side') = (match_id IS NOT NULL AND side_no IS NOT NULL))
);
CREATE UNIQUE INDEX channels_team_uq ON chat.channels (team_id) WHERE kind = 'team';
CREATE UNIQUE INDEX channels_match_side_uq ON chat.channels (match_id, side_no) WHERE kind = 'match_side';

CREATE TABLE chat.messages (
  id          uuid        PRIMARY KEY,               -- UUIDv7 → time-ordered pagination
  channel_id  uuid        NOT NULL REFERENCES chat.channels(id),
  sender_id   uuid        NOT NULL REFERENCES identity.users(id),
  kind        text        NOT NULL CHECK (kind IN ('text','quick')),     -- quick = preset phrase/emote id
  body        text        CHECK (char_length(body) BETWEEN 1 AND 300),  -- text messages
  quick_id    text,                                  -- preset key, translated on device
  filter_result text      NOT NULL DEFAULT 'clean' CHECK (filter_result IN ('clean','masked','blocked')),
  created_at  timestamptz NOT NULL,
  removed_at  timestamptz,                           -- by sender or moderation
  removed_by  uuid        REFERENCES identity.users(id),
  CHECK ((kind = 'text') = (body IS NOT NULL)),
  CHECK ((kind = 'quick') = (quick_id IS NOT NULL))
);
CREATE INDEX messages_channel_idx ON chat.messages (channel_id, id DESC);

CREATE TABLE chat.read_cursors (
  channel_id    uuid NOT NULL REFERENCES chat.channels(id) ON DELETE CASCADE,
  user_id       uuid NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
  last_read_id  uuid NOT NULL,
  PRIMARY KEY (channel_id, user_id)
);
```
Rules (P):
- **Team channel:** members of a persistent team only; history visible to current members.
- **In-match side channel:** created only when the side is a **persistent team or couple**. Temporary sides with strangers (D-016a) get **quick chat only** (preset phrases/emotes, no free text) — protects 13+ players and guests from abuse.
- Messages pass `TextFilter` (D-030 interface): severe terms blocked, mild terms masked. `chat_mute` sanction blocks sending.
- Rate limit **5 messages / 10 s** per user. Retention: **6 months** (P); removed messages blanked immediately.

---

## 3. `tournament`

```sql
CREATE SCHEMA tournament;

CREATE TABLE tournament.tournaments (
  id                  uuid        PRIMARY KEY,
  status              text        NOT NULL CHECK (status IN ('draft','registration','seeding','running','finished','cancelled')),
  bracket_type        text        NOT NULL CHECK (bracket_type IN ('single_elimination','double_elimination','swiss')),
  entity_kind         text        NOT NULL CHECK (entity_kind IN ('user','team','couple')),
  match_config        jsonb       NOT NULL,          -- format, team answer mode, category mode, question count … (validated by game, 0.5)
  is_rated            boolean     NOT NULL DEFAULT false,   -- tournament matches are unrated by default (P)
  min_entrants        int         NOT NULL CHECK (min_entrants >= 2),
  max_entrants        int         NOT NULL CHECK (max_entrants >= min_entrants),
  entry_requirements  jsonb       NOT NULL DEFAULT '{}',    -- e.g. {"min_level":5,"full_account":true,"league_min":"gold"}
  registration_opens_at  timestamptz NOT NULL,
  registration_closes_at timestamptz NOT NULL,
  starts_at           timestamptz NOT NULL,
  round_ready_window_s int        NOT NULL DEFAULT 300,     -- time to show up for each round match (P)
  rewards             jsonb       NOT NULL DEFAULT '{}',    -- per placement: XP, badge ids, cosmetic ids (never currency or rating)
  created_by          uuid        NOT NULL REFERENCES identity.users(id),   -- admin (P)
  created_at          timestamptz NOT NULL,
  finished_at         timestamptz
);
CREATE TABLE tournament.tournament_translations (
  tournament_id uuid NOT NULL REFERENCES tournament.tournaments(id) ON DELETE CASCADE,
  locale        text NOT NULL CHECK (locale IN ('tr','en')),
  name          text NOT NULL,
  description   text,
  PRIMARY KEY (tournament_id, locale)
);

CREATE TABLE tournament.entries (
  id             uuid        PRIMARY KEY,
  tournament_id  uuid        NOT NULL REFERENCES tournament.tournaments(id),
  entity_kind    text        NOT NULL,
  entity_id      uuid        NOT NULL,              -- user / team / couple
  lineup         uuid[],                            -- team entries: user ids fixed at check-in
  seed           int,
  status         text        NOT NULL DEFAULT 'registered'
                 CHECK (status IN ('registered','checked_in','active','eliminated','withdrawn','disqualified','winner')),
  final_place    int,
  registered_at  timestamptz NOT NULL,
  UNIQUE (tournament_id, entity_kind, entity_id)
);
CREATE INDEX entries_entity_idx ON tournament.entries (entity_kind, entity_id);

CREATE TABLE tournament.bracket_matches (
  tournament_id   uuid        NOT NULL REFERENCES tournament.tournaments(id),
  round_no        smallint    NOT NULL,
  slot_no         smallint    NOT NULL,
  bracket_side    text        NOT NULL DEFAULT 'main' CHECK (bracket_side IN ('main','losers','final')),
  entry_a_id      uuid        REFERENCES tournament.entries(id),
  entry_b_id      uuid        REFERENCES tournament.entries(id),    -- NULL = bye
  match_id        uuid,                              -- game match once created
  winner_entry_id uuid        REFERENCES tournament.entries(id),
  status          text        NOT NULL CHECK (status IN ('pending','ready','in_progress','done','walkover')),
  scheduled_at    timestamptz,
  ready_until     timestamptz,                       -- scheduled_at + round_ready_window_s (D-090)
  present         uuid[]      NOT NULL DEFAULT '{}', -- players who pressed Play in the window (D-090)
  PRIMARY KEY (tournament_id, bracket_side, round_no, slot_no)
);
```
Rules (P): admin-created tournaments only at launch; entrants need a **full account**; no-show within the ready window = walkover; voided matches (D-030) re-resolve the bracket slot by admin decision. **Launch scope (D-090):** `single_elimination` only (the other bracket types stay in the schema), entries `lineup` is always filled (user → the user, couple → both partners, team → the captain's lineup), random seeding of the checked-in entries with byes for the top seeds, a draw goes to the higher seed (golden rounds come later), and `entry_requirements` is not used yet.

---

## 4. `commerce` (premium, entitlements, cosmetics, ads)

Store purchases (Apple App Store, Google Play) are **verified server-side**; the client's claim of a purchase is never trusted. The integration library (e.g. RevenueCat vs. direct store APIs) is chosen in a Phase 9 ADR — the schema works with either.

```sql
CREATE SCHEMA commerce;

CREATE TABLE commerce.products (
  id            text        PRIMARY KEY,            -- store product id, same on both stores where possible: 'premium_monthly'
  kind          text        NOT NULL CHECK (kind IN ('subscription','non_consumable')),
  grants        jsonb       NOT NULL,               -- entitlement codes and/or cosmetic ids
  is_active     boolean     NOT NULL DEFAULT true
);

CREATE TABLE commerce.subscriptions (
  id                      uuid        PRIMARY KEY,
  user_id                 uuid        NOT NULL REFERENCES identity.users(id),
  store                   text        NOT NULL CHECK (store IN ('app_store','play_store')),
  product_id              text        NOT NULL REFERENCES commerce.products(id),
  original_transaction_id text        NOT NULL,     -- stable id across renewals
  status                  text        NOT NULL CHECK (status IN ('active','in_grace_period','in_billing_retry',
                                                                'cancelled_active','expired','refunded','revoked')),
  current_period_end      timestamptz NOT NULL,
  auto_renew              boolean     NOT NULL,
  environment             text        NOT NULL CHECK (environment IN ('sandbox','production')),
  updated_at              timestamptz NOT NULL,
  UNIQUE (store, original_transaction_id)
);
CREATE INDEX subscriptions_user_idx ON commerce.subscriptions (user_id);

-- Raw, verified store events (App Store Server Notifications v2, Google RTDN), for audit & replay
CREATE TABLE commerce.store_events (
  id            uuid        PRIMARY KEY,
  store         text        NOT NULL,
  event_id      text        NOT NULL,              -- store's notification id → idempotency
  kind          text        NOT NULL,
  payload       jsonb       NOT NULL,              -- verified, decoded payload
  received_at   timestamptz NOT NULL,
  processed_at  timestamptz,
  UNIQUE (store, event_id)
);

-- What a user may currently use; derived from subscriptions, purchases, rewards
CREATE TABLE commerce.entitlements (
  user_id      uuid        NOT NULL REFERENCES identity.users(id),
  code         text        NOT NULL,               -- 'premium', 'no_ads', …
  source_kind  text        NOT NULL CHECK (source_kind IN ('subscription','purchase','reward','admin')),
  source_id    text        NOT NULL,
  starts_at    timestamptz NOT NULL,
  ends_at      timestamptz,
  PRIMARY KEY (user_id, code, source_kind, source_id)
);

-- Cosmetics catalogue & ownership
CREATE TABLE commerce.cosmetics (
  id           text        PRIMARY KEY,            -- 'frame_neon_blue'
  slot         text        NOT NULL CHECK (slot IN ('avatar_frame','profile_badge','name_color','victory_emote',
                                                    'match_theme','team_emblem_frame')),
  rarity       text        NOT NULL CHECK (rarity IN ('common','rare','epic','legendary','seasonal')),
  obtain       text        NOT NULL CHECK (obtain IN ('purchase','premium','season_reward','tournament_reward',
                                                      'badge_reward','quest_reward','event')),
  is_active    boolean     NOT NULL DEFAULT true
);
CREATE TABLE commerce.cosmetic_translations (
  cosmetic_id  text NOT NULL REFERENCES commerce.cosmetics(id),
  locale       text NOT NULL CHECK (locale IN ('tr','en')),
  name         text NOT NULL,
  PRIMARY KEY (cosmetic_id, locale)
);
CREATE TABLE commerce.inventory (
  user_id      uuid        NOT NULL REFERENCES identity.users(id),
  cosmetic_id  text        NOT NULL REFERENCES commerce.cosmetics(id),
  source_kind  text        NOT NULL,
  source_ref   text        NOT NULL,
  acquired_at  timestamptz NOT NULL,
  expires_at   timestamptz,                          -- premium-only items stop being usable when premium ends
  PRIMARY KEY (user_id, cosmetic_id)
);
CREATE TABLE commerce.equipped (
  user_id      uuid NOT NULL REFERENCES identity.users(id),
  slot         text NOT NULL,
  cosmetic_id  text NOT NULL REFERENCES commerce.cosmetics(id),
  PRIMARY KEY (user_id, slot)
);

-- Rewarded ads (server-side verified callbacks, e.g. AdMob SSV)
CREATE TABLE commerce.ad_rewards (
  id              uuid        PRIMARY KEY,
  user_id         uuid        NOT NULL REFERENCES identity.users(id),
  network         text        NOT NULL,             -- 'admob'
  transaction_id  text        NOT NULL,             -- network's id → idempotency
  placement       text        NOT NULL CHECK (placement IN ('quest_reroll','bonus_xp_after_match','cosmetic_trial')),
  reward          jsonb       NOT NULL,
  verified_at     timestamptz NOT NULL,
  UNIQUE (network, transaction_id)
);
CREATE INDEX ad_rewards_user_day_idx ON commerce.ad_rewards (user_id, verified_at);
```
Rules (P):
- **Premium perks:** no interstitial ads, exclusive cosmetics, 1 extra daily quest, extended stats/match history. **No** gameplay advantage.
- **Ads:** interstitial at most once every **3 matches**, never during a match, never for premium users; rewarded ads limited to **5 per day** per user.
- **No soft currency** at launch: cosmetics come from direct purchase, premium, rewards.
- Restricted accounts may buy; on link conflict (D-015) purchases are restored by the store "restore purchases" flow to whichever account signs in, so nothing paid is lost.
- `commerce` exposes `HasEntitlement(user, code)` to other modules (layer 1, so anyone may call it).

---

## 5. `creator` (Creator Quiz)

Questions themselves live in `content` (`pool = 'creator_only'`, `source.kind = 'creator'`); `creator` holds the quiz around them.

```sql
CREATE SCHEMA creator;

CREATE TABLE creator.quizzes (
  id                uuid        PRIMARY KEY,
  author_user_id    uuid        NOT NULL REFERENCES identity.users(id),
  locale            text        NOT NULL CHECK (locale IN ('tr','en')),
  title             text        NOT NULL CHECK (char_length(title) BETWEEN 3 AND 60),
  description       text        CHECK (char_length(description) <= 300),
  category_id       uuid        REFERENCES content.categories(id),
  question_set_id   uuid        NOT NULL UNIQUE REFERENCES content.question_sets(id),
  status            text        NOT NULL DEFAULT 'draft'
                    CHECK (status IN ('draft','submitted','published','rejected','unpublished')),
  -- draft/rejected/unpublished: playable only in private rooms among the author's friends (D-023)
  -- published: verified by review; playable in any private room and challenges (still never ranked)
  submitted_at      timestamptz,
  published_at      timestamptz,
  reviewed_by       uuid        REFERENCES identity.users(id),
  rejection_reason  text,
  play_count        int         NOT NULL DEFAULT 0,
  like_count        int         NOT NULL DEFAULT 0,
  created_at        timestamptz NOT NULL,
  updated_at        timestamptz NOT NULL
);
CREATE INDEX quizzes_author_idx ON creator.quizzes (author_user_id);
CREATE INDEX quizzes_published_idx ON creator.quizzes (locale, category_id, published_at DESC) WHERE status = 'published';

CREATE TABLE creator.quiz_likes (
  quiz_id   uuid NOT NULL REFERENCES creator.quizzes(id) ON DELETE CASCADE,
  user_id   uuid NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
  liked_at  timestamptz NOT NULL,
  PRIMARY KEY (quiz_id, user_id)
);
```
Rules (P): only **full accounts** create quizzes; **5–30 questions** per quiz; max **20 quizzes** per author; author's own questions go through `TextFilter`; submitting sends every question + translation to moderation review (D-007); a published quiz edited goes back to `submitted`. Creator Quiz matches are **never rated** and never give SP or leaderboard points (D-023 for unverified; proposal extends to published quizzes too, since authors know the answers).

---

## 6. Daily Challenge runs (owned by `game`, isolated from head-to-head tables — D-040)

The daily question set is `content.daily_sets` (`schema-content.md`). Runs are **solo** and are stored **only** here, never in `game.matches` / `match_sides` / `match_participants` / `answers`. Rules: `category-modes.md` §5.

```sql
CREATE TABLE game.daily_runs (
  id               uuid        PRIMARY KEY,
  day              date        NOT NULL,                    -- Europe/Istanbul day (D-029)
  user_id          uuid        NOT NULL REFERENCES identity.users(id),
  set_id           uuid        NOT NULL REFERENCES content.question_sets(id),
  locale           text        NOT NULL CHECK (locale IN ('tr','en')),
  kind             text        NOT NULL CHECK (kind IN ('official','practice')),
  status           text        NOT NULL CHECK (status IN ('in_progress','completed',
                                                          'void_server_error','void_not_started')),
  account_tier_at_start text   NOT NULL CHECK (account_tier_at_start IN ('guest','unverified','full')),
  ruleset_version  int         NOT NULL,
  started_at       timestamptz NOT NULL,
  completed_at     timestamptz,
  correct_count    smallint    CHECK (correct_count BETWEEN 0 AND 10),
  total_time_ms    int         CHECK (total_time_ms >= 0),  -- sum of response_ms; timeout = full window (category-modes §5.2)
  score            int,                                     -- ruleset-1 points, display only
  disconnects      smallint    NOT NULL DEFAULT 0,
  CHECK (status <> 'completed' OR (completed_at IS NOT NULL AND correct_count IS NOT NULL AND total_time_ms IS NOT NULL))
);
-- exactly one official run per user per day that is running or completed; voided runs don't count (failure grace)
CREATE UNIQUE INDEX daily_runs_one_official_uq ON game.daily_runs (day, user_id)
  WHERE kind = 'official' AND status IN ('in_progress','completed');
CREATE INDEX daily_runs_user_idx ON game.daily_runs (user_id, day DESC);

CREATE TABLE game.daily_run_answers (
  run_id             uuid        NOT NULL REFERENCES game.daily_runs(id) ON DELETE CASCADE,
  position           smallint    NOT NULL CHECK (position BETWEEN 1 AND 10),
  question_id        uuid        NOT NULL REFERENCES content.questions(id),
  question_revision  int         NOT NULL,
  option_order       smallint[]  NOT NULL,
  show_at            timestamptz NOT NULL,
  deadline_at        timestamptz NOT NULL,
  option_id          uuid        REFERENCES content.options(id),   -- NULL = timeout
  received_at        timestamptz,
  response_ms        int         CHECK (response_ms >= 0),
  is_correct         boolean     NOT NULL,
  points             int         NOT NULL DEFAULT 0,
  created_at         timestamptz NOT NULL,                         -- 12-month purge like match detail (D-026)
  PRIMARY KEY (run_id, position)
);
```
Rules: starting an official run when one is `in_progress`/`completed` for that day fails with `DAILY_ATTEMPT_USED` / `DAILY_ATTEMPT_IN_PROGRESS`; such a player may start a `practice` run. Runs in `void_*` states free the official slot (failure grace, category-modes §5.4). Live run state is in Redis (redis-keys §2a).

## 7. Events
| Event | Producer | Consumers |
|---|---|---|
| `chat.MessageSent` | chat | realtime (delivery), notification (offline members) |
| `chat.MessageReported` → via moderation reports | — | — |
| `tournament.RegistrationOpened` / `RoundScheduled` / `TournamentFinished` | tournament | notification, progression |
| `commerce.SubscriptionChanged` / `EntitlementChanged` | commerce | notification, progression (extra quest slot) |
| `commerce.CosmeticGranted` | commerce | notification |
| `creator.QuizSubmitted` / `QuizPublished` / `QuizRejected` | creator | moderation, notification |
| `game.DailyRunCompleted` (official, completed only) | game | ranking (daily board), progression (XP, quest, streak), notification |

## 8. Accepted proposals (D-031)
1. **Notifications:** marketing push off until opt-in; quiet hours available; max 10 push/user/hour; inbox items expire after 30 days.
2. **Chat:** free-text side chat only for persistent teams/couples; temporary sides (strangers, guests) get quick-chat presets only; 5 messages per 10 s; messages kept 6 months.
3. **Tournaments:** admin-created only at launch; full accounts only; unrated by default; rewards are XP/badges/cosmetics only.
4. **Premium perks:** no interstitials, exclusive cosmetics, +1 daily quest, extended stats — no gameplay advantage.
5. **Ads:** interstitial at most every 3 matches (never mid-match, never for premium); max 5 rewarded ads/day; rewarded placements = quest reroll, bonus XP after match, cosmetic trial.
6. **No soft currency** at launch.
7. **Creator Quiz:** full accounts only; 5–30 questions; max 20 quizzes per author; published quizzes also never rated / no SP.
8. **Daily Challenge:** first completed attempt per day counts, later ones are practice. *(Refined by D-040: solo score race, isolated tables above.)*
