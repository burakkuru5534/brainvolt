# BrainVolt — Master Roadmap

Real-time, social, competitive quiz platform.
Stack: **Flutter** (mobile) · **Go modular monolith** (backend) · **PostgreSQL** (persistent data) · **Redis** (cache, matchmaking, live match state) · **Firebase Auth & FCM**.

## Ground rules
- **Zero scope reduction** — no feature, game mode, team mechanic or requirement may be dropped.
- **Server authority** — backend alone owns game state, timers, answers, scores and winners. Client scores are never trusted.
- **Inspect before changing** — never rewrite working code.
- **Atomic tasks** — every phase is broken into 5–10 minute tasks before it starts; each task ends with tests run (no skipped failures).
- **No TODOs / mocks as final implementations.**
- **Commits are made by the project owner**, not by the assistant.

## Decision policy (from 2026-10-15, owner instruction)
The owner reviews only **critical** decisions (anything that blocks the system, changes the architecture, or carries security / budget risk).
Standard game rules, retention periods, default limits and small details are decided by the assistant using industry-standard practice, recorded in the relevant document and in the decision log below with source **"delegated"**, and work continues without waiting for confirmation. The owner can override any delegated decision at any time.

## Status legend
`[ ]` not started · `[~]` in progress · `[x]` done · `[!]` blocked / needs decision

## Phase overview

| # | Phase | Status |
|---|-------|--------|
| 0 | Planning & Architecture | [x] |
| 1 | Core Infrastructure | [~] |
| 2 | Question & Category System | [ ] |
| 3 | Game Engine | [ ] |
| 4 | Real-time 1v1 | [ ] |
| 5 | Matchmaking & Social System | [ ] |
| 6 | Team Matches & Couple Contest | [ ] |
| 7 | Ranked Seasons & Leaderboards | [ ] |
| 8 | Special Modes & Content Moderation | [ ] |
| 9 | Polish & Production Readiness | [ ] |

---

## Baseline (repository state at start — 2026-10-15)
- Git: 1 commit (`initial commit`).
- `backend/` — empty. `mobile/` — empty. `Makefile` — empty.
- `docker-compose.yml` — Postgres 16-alpine + Redis 7-alpine (no volumes, no healthchecks, hard-coded credentials, obsolete `version` key → to be addressed in Phase 1).
- `docs/ARCHITECTURE.md` — 5-line architecture note (Turkish).
- `.gitignore` — Go / Flutter / secrets rules.
- No production code exists yet.

---

## Decision log
| # | Topic | Decision | Date | Affects |
|---|-------|----------|------|---------|
| D-001 | Initial content language | Turkish | 2026-10-15 | 0.2.3, Phase 2 seed |
| D-002 | Multi-language infrastructure | Turkish + English supported (locale-aware content & UI from day one) | 2026-10-15 | 0.2.2, 0.2.3, 0.4.1, mobile i18n |
| D-003 | Ranked rating algorithm | Glicko-2 | 2026-10-15 | 0.3.10, Phase 7 |
| D-004 | Casual (unranked) matches | Do not affect rating | 2026-10-15 | 0.3.5, 0.3.10, 0.5.1 |
| D-005 | Seed questions | OpenTDB after license/suitability check + our own question pool | 2026-10-15 | 0.2.3, Phase 2 |
| D-006 | Question source tracking | Mandatory for every question (source + license) | 2026-10-15 | 0.2.3, Phase 2, Phase 8 |
| D-007 | Content in ranked matches | Verified questions only | 2026-10-15 | 0.2.3, 0.3.5, 0.5.3, Phase 8 moderation |
| D-008 | Repository | Single monorepo at https://github.com/burakkuru5534/brainvolt | 2026-10-15 | 0.1.3, Phase 1 |
| D-009 | Go module path | `github.com/burakkuru5534/brainvolt/backend` | 2026-10-15 | 0.1.3, Phase 1 |
| D-010 | CI | GitHub Actions in Phase 1 — `.github/workflows/ci.yml` runs Go and Flutter tests | 2026-10-15 | 0.1.3, Phase 1 |
| D-011 | Launch platforms | Android + iOS together | 2026-10-15 | ADR-0003, Phase 1, Phase 9 |
| D-012 | Application / bundle ID | `com.brainvolt.app` (Android applicationId and iOS bundle ID) | 2026-10-15 | Phase 1 (`flutter create`, Firebase apps) |
| D-013 | Sign-in methods | Firebase Auth: Anonymous (guest), Google, Apple, Email/Password. Sign in with Apple offered alongside Google (App Store rule) | 2026-10-15 | 0.2.2, 0.4.1, Phase 1 |
| D-013a | Guest upgrade | Guests play immediately; later link Google / Apple / Email to the same account with no loss of progress (Firebase account linking keeps the same UID) | 2026-10-15 | 0.2.2, Phase 1 |
| D-014 | Firebase projects | Single Firebase project `brainvolt` | 2026-10-15 | ADR-0003, Phase 1 |
| D-015 | Guest link conflict | If the credential a guest tries to link already belongs to another BrainVolt user: warn that guest progress will be lost, then switch to the existing account. No automatic merge | 2026-10-15 | 0.2.2, 0.4.2, Phase 1 |
| D-016 | Guest limits | Guests may play casual and ranked; they do **not** appear on public leaderboards and cannot add friends or create teams until they upgrade | 2026-10-15 | 0.2.2, 0.2.5, 0.2.6, Phase 5, Phase 7 |
| D-017 | Email verification | Email/password users must verify their email; unverified accounts have the same limits as guests (D-016) | 2026-10-15 | 0.2.2, Phase 1 |
| D-016a | Guests in team formats | Guests cannot create or lead persistent teams, but **can** join random matchmaking for 2v2/3v3/4v4 and be placed on temporary sides with strangers. Post-match "add friend" / "form team" prompts an account upgrade | 2026-10-15 | 0.2.5, 0.5, Phase 5, Phase 6 |
| D-018 | Couple Contest classification | Couple Contest is a **player format** (specialised 2v2 of couple profiles), not a category mode | 2026-10-15 | glossary, 0.3.8, 0.5.1 |
| D-019 | Team answer modes | Shared Answer = one collective decision per question; Individual Turns = questions cycle through members in order; Fast Team Round = all answer simultaneously, side score by predefined aggregation rule (e.g. fastest correct, majority, total correct) | 2026-10-15 | glossary, 0.3.6, Phase 6 |
| D-020 | Minimum age | 13+; confirmation checkbox at first launch; store only `age_verified_at` | 2026-10-15 | 0.2.2, Phase 1 |
| D-021 | Avatars | Built-in presets only at launch (no photo uploads); cosmetic frames & custom badges in Phase 9 | 2026-10-15 | 0.2.2, Phase 9 |
| D-022 | Retention | Abandoned guest (0 matches, no activity) deleted after 7 days; inactive guest (≥1 match, not upgraded) after 90 days inactive; user deletion has a 14-day cooling-off window before purge/anonymisation | 2026-10-15 | 0.2.2, 0.2.10 |
| D-023 | Unverified content | No unverified questions in public/casual/ranked matches. Unverified Creator Quiz sets allowed only in private/custom rooms among friends | 2026-10-15 | 0.2.3, 0.3.7, 0.5, Phase 8 |
| D-024 | True/false questions | Excluded from Ranked; allowed in casual and quick modes | 2026-10-15 | 0.2.3, 0.3.5, 0.5 |
| D-025 | Question media | Text-only at launch; image/audio deferred to a future media pipeline | 2026-10-15 | 0.2.3 |
| D-026 | Match data retention | Match results, scores, rating changes and history kept forever; round-by-round and per-answer detail kept 12 months, then purged/archived | 2026-10-15 | 0.2.4, 0.2.10 |
| D-027 | Social limits | Accepted as proposed in `schema-social.md` §6 (friends 300, 50 requests/day, 6-char codes, rematch 30 s, revenge & direct challenge 24 h, 20 challenges/h, idle room 15 min, teams 2–4 members & max 3 per user, couple one per user). Revenge only by the losing side | 2026-10-15 | 0.2.5, 0.3.7, Phase 5 |
| D-028 | Temporary sides & team boards | Results of temporary sides do not count toward Team leaderboards; members' individual rating/leaderboards update normally | 2026-10-15 | 0.2.6, 0.3.10 |
| D-029 | Ranking & progression parameters | Rating buckets solo/team/couple; leaderboard periods on Europe/Istanbul time, week starts Monday; team country = captain's country; top 1 000 per board archived each period; 3 daily + 3 weekly quests, rewards claimed manually | 2026-10-15 | 0.2.6, 0.3.10, Phase 7 |
| D-030 | Moderation & anti-cheat policy | Max 20 reports/user/day, restricted-account reports weigh less; 3 `wrong_answer` reports pull a question from Ranked into review; cheat signals only flag — every sanction needs a human; voided match revokes SP/leaderboard points, no retroactive Glicko-2 recompute, cheater gets ranked ban + SP reset, honest opponents keep results; "too fast" signal < 300 ms + 15 ms/char (tune in Phase 8); retention: cases/sanctions/appeals 3 years, report comments/cheat signals/install links 12 months, records stay on anonymised id after deletion | 2026-10-15 | 0.2.7, 0.2.10, Phase 8 |
| D-031 | Platform policies | Marketing push opt-in; quiet hours; ≤10 push/user/h; inbox 30 days. Free-text match chat only for persistent teams/couples, temporary sides quick-chat only; 5 msgs/10 s; chat kept 6 months. Tournaments admin-created, full accounts, unrated by default, rewards XP/badges/cosmetics. Premium: no interstitials, exclusive cosmetics, +1 daily quest, extended stats — zero gameplay advantage. Ads: ≤1 interstitial per 3 matches (never mid-match / premium), ≤5 rewarded/day (quest reroll, bonus XP, cosmetic trial). No soft/hard currency at launch. Creator Quiz: full accounts, 5–30 questions, ≤20 quizzes/author, never rated / no SP. Daily Challenge: first completed attempt per day counts | 2026-10-15 | 0.2.8, 0.3.7, Phase 8, Phase 9 |
| D-032 | Redis-level limits | Matchmaking ticket 10 min; resume token 2 min after disconnect (game reconnect window ≤ this); no-repeat window 30 days / 2 000 questions per user; API 60 req/10 s per user, 300 req/min per IP, 20 WS msg/s per connection, 20 session calls/h per installation | 2026-10-15 | 0.2.9, 0.3.9, 0.4.8, Phase 4, Phase 5 |
| D-033 | Remaining retention | Closed friend requests, challenges, rooms, team invites purged after 90 days; quest assignments 12 months after period end; commerce transactions/subscriptions/store notifications 10 years (TTK / VUK); match transitions & round event logs = match detail, 12 months | 2026-10-15 | 0.2.10 |
| D-034 | Ready flow & voiding | Ready deadline 20 s after creation (tournaments own window); 3 s start countdown; 3 missed matchmade ready checks in 30 min → 5 min queue cooldown (queue health, not a sanction); ready players return to the front of the queue; voiding revokes SP & leaderboard points only — non-cheaters keep XP, quest progress, streaks; cheaters handled by sanctions | 2026-10-15 | 0.3.1, Phase 4, Phase 5 |
| D-035 | Round flow | 3 s reveal; early close when all expected active answerers answered; Shared Answer: live teammate picks, any member locks, fallback most-picked (tie → earliest), none → timeout; Fast Team Round: teammates see only "answered" until reveal; Individual Turns: seat-order rotation, symmetric for both sides | 2026-10-15 | 0.3.2, 0.3.6, Phase 4, Phase 6 |
| D-036 | Timing & latency | In-match ping every 2 s, RTT = median of last 10; uplink compensation capped 150 ms; grace = uplink estimate + 100 ms (100–250 ms); synchronized show_at buffer ≤ 250 ms; "weak connection" warning above 400 ms sustained; operator alert on host clock drift > 50 ms; pre-round category pick window 10 s | 2026-10-15 | 0.3.3, 0.4.3, Phase 4 |
| D-037 | Scoring ruleset 1 | No negative points (Sudden Death elimination separate); base 100 × difficulty multiplier 0.8/0.9/1.0/1.1/1.2; speed bonus up to 100, linear to 0 at deadline; streak +20 at 3–4, +40 from 5; true/false worth half (base 50, speed ≤ 50) | 2026-10-15 | 0.3.4, Phase 3 |
| D-038 | Match types | Quick Duel 7 q / 15 s / curve 2-2-3-3-3-4-4; Ranked 10 q / 15 s / MC only / rising curve / **matchmaking only**; Sudden Death 10 s, first right-vs-wrong split decides, difficulty +1 every 3 rounds, cap 30 then highest score; Marathon 25 q / 12 s / 5 difficulty blocks / 15 s half-time after q12; category caps 2/3/5, no consecutive sub-category, TF ≤ 30 % never consecutive; Shared Answer +5 s | 2026-10-15 | 0.3.5, 0.5, Phase 3 |
| D-039 | Team answer modes | Same scale (side ≤ one player's max per round); Shared Answer: auto-lock = no speed bonus, side streak, equal split (remainder to locker); Individual Turns: side streak across turns, points to turn holder, disconnected/left turn holder → next active seat; Fast Team Round: personal streaks, rules Average (default) / Fastest Correct / Majority, Average correct if ≥ 50 % of active members correct; Ranked & matchmade casual fixed to Average, hosts/organisers choose in rooms, challenges, tournaments | 2026-10-15 | 0.3.6, 0.5, Phase 6 |
| D-040 | Daily Challenge | **Solo score race** (engine format `solo`, no opponent/winner): same 10 MC questions per day for every player (one set verified in TR and EN), Ranked curve, 15 s; ONE official attempt per day, practice runs allowed with zero leaderboard impact; board ranked by correct answers, then lowest total time (timeout = 15 s), then earliest completion; share card + "beat my score" to friends; runs stored in `game.daily_runs` / `daily_run_answers` and board in `ranking.daily_challenge_entries` + `rank:daily:*`, isolated from match tables, SP and general boards; server errors and drops before Q1 do not consume the attempt; drops after Q1: clock keeps running, resume ≤ 2 min, else remaining = timeouts and attempt consumed | 2026-10-15 | 0.3.7, 0.3.11, Phase 8 |
| D-041 | Ranked integrity | No category mode in Ranked (global mixed pool); matchmaking driven by Glicko-2 rating **and** RD | 2026-10-15 | 0.3.7, 0.3.10, 0.5, Phase 5 |
| D-042 | Category Master | Matchmaking only for top-level categories, pairs only identical category searches; unrated at launch; future category ratings use separate `category:<id>` buckets, never mixed with global Ranked rating | 2026-10-15 | 0.3.7, Phase 5, Phase 8 |
| D-043 | Mixed Battle | 3 categories pre-selected before the match (host/challenger, or server in matchmaking); picking side alternates per round, 10 s pick from those 3; max 3 consecutive same category; in teams the designated turn holder (seat rotation) picks, and also answers in Individual Turns | 2026-10-15 | 0.3.7, Phase 8 |
| D-044 | Creator Quiz | 10/15/20 s timers; author defines order, content, options, category; rooms & direct challenges; author earns zero XP on own quiz; validation required before public discovery; never in Ranked pool, never affects rating | 2026-10-15 | 0.3.7, Phase 8 |
| D-045 | Revenge Match | Offered instantly on results screen (valid 24 h) against the preceding opponent, copies the configuration; 50 % home category (rounded down); max 3 consecutive same-category questions; never rated (Ranked only via fresh matchmade game); max 3 revenges per chain | 2026-10-15 | 0.3.7, Phase 8 |
| D-046 | 0.3.7 confirmations | One daily set identical in TR and EN → single unified Daily leaderboard (no per-language split); Mixed Battle: turn holder picks the category, answering follows the configured team answer mode (all three compatible); practice daily runs give zero XP and zero quest progress; max 3 consecutive revenge matches per chain | 2026-10-15 | 0.3.7, Phase 8 |
| D-047 | Couple Contest | Couples only play couples; Sync Bonus +20 when both partners correct in Fast Team Round (casual & Ranked, within 260 cap); dual rating: couple rating + each partner's personal couple rating, new couple starts at partners' average; couple appears on both partners' country boards; re-forming the same pair restores the old profile/history/rating; 7-day cooldown before forming a couple with a different partner | 2026-10-15 | 0.3.8, 0.3.10, Phase 5, Phase 6 |
| D-048 | Edge cases | Tie-breakers: more correct rounds → lower total correct response time → draw (tournaments & revenge: up to 5 golden rounds, then higher seed / series draw); reconnect windows QD 20 s, Ranked 30 s, SD 20 s, Marathon 60 s, no pausing, >3 disconnects or >2× window away = forfeit; leaver gets loss + XP only for rounds played, staying teammates keep SP; leaver cooldown 2nd leave/24 h 5 min, 3rd+ 15 min; takeover voids open round and replaces it, no takeover in 30 s → cancel; both sides absent → cancel `all_sides_absent` | 2026-10-15 | 0.3.9, Phase 4, Phase 5 |
| D-049 | Rating / SP / XP formulas (delegated) | Glicko-2 1500/350/0.06, τ 0.5, RD 50–350, per-match periods, daily inactivity inflation, composite opponents for team play, new team/couple RD 200; raw rating hidden (league/SP shown); SP win +30 / draw +10 / loss −15, streak +5…+15, league floor protection; leagues Bronze 0 / Silver 300 / Gold 800 / Platinum 1600 / Diamond 2800 with 3 divisions (Diamond none); leaderboard points win 20 / draw 10 + 2 per correct round, **matchmade matches only**; XP 20 + 5/correct + win 15 / draw 8, level step 100 + 25(L−1), cap 100, no premium XP boost; seasons 8 weeks, SP reset, RD raised to ≥ 150 — see `docs/game/progression-formulas.md` | 2026-10-15 (delegated) | 0.3.10, Phase 7 |
| D-050 | API conventions (delegated) | `/v1` + forced update via 426; JSON snake_case; UUIDv7; RFC 3339 on REST, Unix ms for WS timing; Firebase Bearer; WS token in first message; `Accept-Language`; error envelope with stable codes; cursor pagination; `Idempotency-Key` (24 h) on creating/spending POSTs; `If-Match` for shared resources — `docs/api/conventions.md` | 2026-10-15 (delegated) | 0.4, Phase 1 |
| D-051 | WebSocket protocol (delegated) | One socket per installation; one seat-holding device per match; per-stream gap-free `seq`; ack/error per intent; ping 2 s in play / 20 s idle, dead after 3 missed; replay-or-snapshot resume (2 min); awards sent asynchronously; reconnect backoff 0.5 s → 10 s — `docs/api/websocket.md` | 2026-10-15 (delegated) | 0.4, Phase 4 |
| D-052 | Extra limits & timeouts (delegated) | Rooms 10/h, wrong invite codes 10/10 min, codes 30/day, tickets 30/10 min, quick chat 3/5 s, profile edits 10/h, creator writes 60/h; REST body 64 KB (quiz 256 KB), WS 4 KB in / 256 KB out; HTTP timeouts 10/15/60 s, DB statement 3 s (worker 60 s) — `docs/api/limits.md` | 2026-10-15 (delegated) | 0.4.8, Phase 1 |
| D-053 | Mode matrix choices (delegated) | Matchmade team/couple queues (ranked & casual) = Fast Team Round + average at launch (other team modes via rooms, challenges, rematch/revenge, tournaments; future config flag); tournament matches never ranked; rematch of ranked converts like revenge (quick_duel, 10 MC, unrated) — `docs/game/mode-matrix.md` | 2026-10-15 (delegated) | 0.5, Phase 3, Phase 5 |
| D-054 | Local dev environment (delegated) | One `.env.example` at the repo root, shared by Docker Compose, the Makefile and the backend (repo-layout updated; was `backend/.env.example`); Compose fails fast on missing credentials (`${VAR:?}`); dev ports bound to `127.0.0.1`; local Redis password-protected (`BRAINVOLT_REDIS_PASSWORD`); Firebase Auth emulator built from `scripts/firebase-emulator/` (`node:24.22.0-alpine` + `firebase-tools` 15.34.0) with offline project `demo-brainvolt`, emulator users in memory | 2026-10-15 (delegated) | 1.1, 1.2, Phase 1 |

---

## Phase 0 — Planning & Architecture  `[x]`
Output: design documents under `docs/` only. No production code.
Verification for each task: document reviewed against the requirement list + cross-references consistent.

### 0.1 Repository & codebase analysis
- [x] 0.1.1 Inventory repository, record baseline (section above)
- [x] 0.1.2 Write `docs/adr/0001-tech-stack-and-versions.md` — pin Go, Flutter/Dart, Postgres, Redis versions; Firebase products used
- [x] 0.1.3 Write `docs/architecture/repo-layout.md` — monorepo folder layout (backend/, mobile/, docs/, deploy/, scripts/)
- [x] 0.1.4 Write `docs/architecture/backend-modules.md` — Go module boundaries & allowed dependencies (auth, user, content, game, realtime, matchmaking, social, team, ranking, progression, moderation, notification, chat, tournament, monetization)
- [x] 0.1.5 Write `docs/adr/0002-backend-libraries.md` — HTTP router, WebSocket lib, DB driver/query layer, migration tool, logging, config, test tooling
- [x] 0.1.6 Write `docs/adr/0003-mobile-architecture.md` — state management, routing, networking/WS client, DI, folder structure

### 0.2 Data models & database schema
- [x] 0.2.1 `docs/domain/glossary.md` — canonical terms (match, round, question slot, participant, side, team, couple, MMR, SP, XP, league…)
- [x] 0.2.2 Schema: identity — users, firebase identity link, profiles, devices/FCM tokens, settings, country
- [x] 0.2.3 Schema: content — categories, sub-categories, questions, answer options, difficulty, locale, source & license
- [x] 0.2.4 Schema: game — matches, match_configs, sides/teams in match, participants, rounds, match_questions, answers, results
- [x] 0.2.5 Schema: social — friendships, friend requests, direct challenges, invite codes, private rooms, team profiles, couple profiles, rematch links
- [x] 0.2.6 Schema: ranking & progression — ratings (per bucket), seasons, season points, XP/levels, leagues, streaks, badges, quests, leaderboard snapshots
- [x] 0.2.7 Schema: moderation & anti-cheat — question reports, moderation actions, user sanctions, cheat signals
- [x] 0.2.8 Schema: platform — notifications log, team chat, tournaments, subscriptions, cosmetics/inventory, ad events, creator quizzes, daily challenges
- [x] 0.2.9 `docs/domain/redis-keys.md` — key naming, data types, TTLs (queues, live match state, presence, locks, rate limits, leaderboard ZSETs)
- [x] 0.2.10 Consolidated ERD (`docs/domain/erd.md`, Mermaid) + index/constraint/retention review

### 0.3 Game state machine & scoring rules
- [x] 0.3.1 Match lifecycle: states `waiting → ready → in_progress → completed | cancelled`, transitions, guards, triggers, side effects
- [x] 0.3.2 Round / question sub-state machine (dispatched → answering → locked → revealed)
- [x] 0.3.3 Server timer authority model (server timestamps, latency grace window, late-answer rejection)
- [x] 0.3.4 Base scoring formula (correctness, speed bonus, streak bonus, caps)
- [x] 0.3.5 Match type rules: Quick Duel, Ranked, Sudden Death, Marathon
- [x] 0.3.6 Team answer mode rules: Shared Answer, Individual Turns, Fast Team Round
- [x] 0.3.7 Category mode rules: Blind Challenge, Category Master, Mixed Battle, Daily Challenge, Creator Quiz, Revenge Match
- [x] 0.3.8 Couple Contest rules
- [x] 0.3.9 Ties, tie-breakers, forfeit, abandonment, disconnect/reconnect windows, cancellation conditions
- [x] 0.3.10 MMR, Season Points and XP formulas; league thresholds; seasonal reset rule
- [x] 0.3.11 State-machine & scoring test-case catalog (drives Phase 3 tests)

### 0.4 API & WebSocket contracts
- [x] 0.4.1 Conventions: versioning, Firebase ID-token auth, error envelope, pagination, idempotency keys, time format
- [x] 0.4.2 REST endpoint catalog per module (`docs/api/rest.md`)
- [x] 0.4.3 WS handshake, auth, envelope (type, seq, ack, ts), heartbeat
- [x] 0.4.4 WS client → server messages
- [x] 0.4.5 WS server → client events
- [x] 0.4.6 Reconnect / resume protocol (state snapshot, missed-event replay)
- [x] 0.4.7 Error-code registry (REST + WS)
- [x] 0.4.8 Rate limits & payload size limits

### 0.5 Game mode combination matrix & validation
- [x] 0.5.1 Define axes: match type × format (1v1, 2v2, 3v3, 4v4, couple) × team answer mode × category mode × ranked/unranked
- [x] 0.5.2 Fill matrix: allowed / forbidden + reason for every combination
- [x] 0.5.3 Declarative validation rules for match configuration (input to Phase 3 validator)
- [x] 0.5.4 Matrix test-case table (valid + invalid combos)

### 0.6 Review & sign-off
- [x] 0.6.1 Open-decisions list resolved with owner (D-001 … D-053; no open decisions block Phase 1)
- [x] 0.6.2 Cross-document consistency check; Phase 0 marked done

---

## Phase 1 — Core Infrastructure  `[~]`
Goal: running skeleton — backend (config, logging, health, Postgres, Redis, migrations, Firebase auth, session/profile endpoints), Flutter app (age gate, guest + Google/Apple/email sign-in, upgrade, TR/EN), Docker Compose, CI. Every task ends with its tests green.

**Owner prerequisites (external accounts — cannot be done from code):**
- [ ] P1.A Create Firebase project `brainvolt`; add Android + iOS apps `com.brainvolt.app`; enable Anonymous, Google, Apple, Email/Password providers (D-013, D-014)
- [ ] P1.B Apple Developer Program membership (required for iOS builds and Sign in with Apple; annual fee) and Google Play Console account
- [ ] P1.C GitHub repository `burakkuru5534/brainvolt` with Actions enabled (D-008, D-010)

### 1.1 Repository & tooling  `[x]`
- [x] 1.1.1 `.gitattributes` (LF; `*.bat`/`*.ps1` CRLF) and `.editorconfig`
- [x] 1.1.2 `README.md` quick start (Windows + Git Bash notes)
- [x] 1.1.3 Makefile targets (`up`, `down`, `run`, `test`, `lint`, `migrate`, `gen`) — each also documented as the plain command

### 1.2 Local environment (Docker Compose)  `[x]`
- [x] 1.2.1 Compose v2: `postgres:18.6-alpine`, `redis:8.10.2-alpine`, named volumes (PG 18 mounts `/var/lib/postgresql`), healthchecks, no `version:` key (ADR-0001)
- [x] 1.2.2 Redis config: AOF everysec + RDB, `maxmemory-policy noeviction` (redis-keys §1)
- [x] 1.2.3 `.env.example` with all `BRAINVOLT_*` vars; credentials only from env
- [x] 1.2.4 Firebase Auth Emulator service for local dev (ADR-0003)

### 1.3 Go project setup  `[x]`
- [x] 1.3.1 `backend/go.mod` (`github.com/burakkuru5534/brainvolt/backend`, `go 1.27`, `toolchain go1.27.2`) + `cmd/api/main.go` that starts and stops cleanly
- [x] 1.3.2 `internal/app` composition root skeleton (api / worker wiring)
- [x] 1.3.3 `.golangci.yml` (v2) + `go vet` clean
- [x] 1.3.4 Architecture test enforcing `backend-modules.md` §2 import rules

### 1.4 Platform packages (each with unit tests)  `[~]`
- [x] 1.4.1 `platform/config` — env v11 parsing + validation
- [x] 1.4.2 `platform/logging` — slog JSON/text, request/user/match ids from context
- [x] 1.4.3 `platform/clock` — real + fake clock
- [x] 1.4.4 `platform/ids` — UUIDv7
- [ ] 1.4.5 `platform/httpserver` — ServeMux, middleware (request id, recover, logging, timeouts, body limits), error envelope + codes (errors.md, TR/EN messages)
- [ ] 1.4.6 Health checks: `/healthz`, `/readyz` (Postgres, Redis, migrations)
- [ ] 1.4.7 `platform/postgres` — pgxpool, tx helper, statement timeout, health (integration test)
- [ ] 1.4.8 `platform/redis` — go-redis client, health (integration test)
- [ ] 1.4.9 `platform` rate limiter (fixed windows, limits.md §1 general + session limits)
- [ ] 1.4.10 Idempotency middleware (`Idempotency-Key`, 24 h)
- [ ] 1.4.11 `platform/firebase` — ID-token verification incl. emulator (unit + emulator test)

### 1.5 PostgreSQL migration setup
- [ ] 1.5.1 goose embedded migrations + `cmd/migrate` (up/down/status)
- [ ] 1.5.2 Migration: River tables
- [ ] 1.5.3 Migrations: `identity`, `profile`, `notification.devices` schemas (schema-identity.md)
- [ ] 1.5.4 sqlc config per module + generated code committed + `sqlc diff` check
- [ ] 1.5.5 Integration-test harness: template DB per package, parallel-safe

### 1.6 Identity & profile (first business code)
- [ ] 1.6.1 Tier derivation function (schema-identity §2.1) + table tests
- [ ] 1.6.2 `POST /v1/session` provisioning (age gate, idempotent, events) + integration tests
- [ ] 1.6.3 `POST /v1/session/abandon-guest` (D-015)
- [ ] 1.6.4 `GET /v1/me`, `PATCH /v1/me/profile` (30-day rules), `GET/PATCH /v1/me/settings`
- [ ] 1.6.5 Devices endpoints (`POST/DELETE /v1/me/devices`)
- [ ] 1.6.6 Account deletion request/cancel (cooling-off only; purge job in a later phase with River)
- [ ] 1.6.7 OpenAPI 3.1 generation for implemented endpoints (`docs/api/openapi.yaml`) + CI drift check
- [ ] 1.6.8 Backend Dockerfile + `backend` service in Compose

### 1.7 Flutter app skeleton
- [ ] 1.7.1 `flutter create` in `mobile/` (org `com.brainvolt`, app id `com.brainvolt.app`), `.flutter-version` 3.47.1, pinned `pubspec.lock`
- [ ] 1.7.2 `analysis_options.yaml` (flutter_lints + riverpod_lint, zero warnings)
- [ ] 1.7.3 Folder structure `app/ core/ features/` (ADR-0003) + Riverpod + go_router skeleton
- [ ] 1.7.4 Flavors dev/prod via `--dart-define-from-file`
- [ ] 1.7.5 gen-l10n with `app_tr.arb` (template) + `app_en.arb`; locale override
- [ ] 1.7.6 dio client: auth interceptor, `Accept-Language`, `X-App-Version`, error-code mapping + tests
- [ ] 1.7.7 build_runner (freezed/json_serializable) with generated files committed

### 1.8 Firebase Auth integration (mobile)
- [ ] 1.8.1 FlutterFire configuration (needs P1.A)
- [ ] 1.8.2 Age gate screen (D-020) + test
- [ ] 1.8.3 Anonymous sign-in + `POST /v1/session` bootstrap
- [ ] 1.8.4 Google, Apple, Email/Password sign-in & email verification flow
- [ ] 1.8.5 Guest upgrade via account linking (D-013a) + link-conflict flow (D-015) + tests with mocked auth
- [ ] 1.8.6 Profile & settings screens wired to the API

### 1.9 CI (D-010)
- [ ] 1.9.1 `.github/workflows/ci.yml`: Go vet, golangci-lint, unit + integration tests (`-race`) with Postgres 18 & Redis 8.10 services, sqlc diff, OpenAPI drift
- [ ] 1.9.2 Flutter analyze, test, codegen-up-to-date check
- [ ] 1.9.3 Branch protection guidance in README (owner applies on GitHub)

## Phase 2 — Question & Category System  `[ ]`
- [ ] Category & sub-category models
- [ ] Migrations
- [ ] Repository layer
- [ ] Seed data (source & license documented)
- [ ] Question listing API
- [ ] Validation
- [ ] Tests

## Phase 3 — Game Engine  `[ ]`
- [ ] Match & player models
- [ ] Match configuration model + validator (per 0.5)
- [ ] Match types: Quick Duel, Ranked, Sudden Death, Marathon
- [ ] Team answer modes: Shared Answer, Individual Turns, Fast Team Round
- [ ] Category modes (engine rules): Blind Challenge, Category Master, Mixed Battle, Daily Challenge, Creator Quiz, Revenge Match
- [ ] Server-side time management
- [ ] Server-side score management
- [ ] State machine (waiting, ready, in_progress, completed, cancelled) + tests

## Phase 4 — Real-time 1v1  `[ ]`
- [ ] WebSocket connection
- [ ] WebSocket auth
- [ ] Lobby / ready flow
- [ ] Question dispatch
- [ ] Answer ingestion
- [ ] Live score updates
- [ ] Live timer updates
- [ ] Match completion
- [ ] Results screen
- [ ] Disconnect / reconnect handling
- [ ] Tests

## Phase 5 — Matchmaking & Social System  `[ ]`
- [ ] MMR-based matchmaking queue
- [ ] Expanding search ranges
- [ ] Friends
- [ ] Direct challenges
- [ ] Invite codes
- [ ] Private rooms
- [ ] Team profiles
- [ ] Couple profiles
- [ ] Rematch flows

## Phase 6 — Team Matches & Couple Contest  `[ ]`
- [ ] 2v2 team matches
- [ ] 3v3 team matches
- [ ] 4v4 team matches
- [ ] Team membership & management
- [ ] Team answer mechanics (all three modes, live)
- [ ] Aggregated team scoring
- [ ] Team result screens
- [ ] Couple Contest rules & flow

## Phase 7 — Ranked Seasons & Leaderboards  `[ ]`
- [ ] MMR (skill)
- [ ] Season Points (leaderboard)
- [ ] XP (progression)
- [ ] Leaderboard time filters: Daily / Weekly / Monthly / All-time
- [ ] Leaderboard entity filters: Individual / Team / Couples
- [ ] Leaderboard scope filters: Friends / Country / Global
- [ ] Category-based leaderboards
- [ ] Leagues: Bronze → Diamond
- [ ] Streaks
- [ ] Badges
- [ ] Quests
- [ ] Seasonal resets

## Phase 8 — Special Modes & Content Moderation  `[ ]`
- [ ] Blind Challenge (full feature)
- [ ] Category Master (full feature)
- [ ] Mixed Battle (full feature)
- [ ] Daily Challenge (full feature) — solo score race per D-040:
  - [ ] Daily set publishing job (10 MC, verified TR+EN, Ranked curve)
  - [ ] Solo run engine (`format = solo`) reusing round phases, timing and scoring
  - [ ] Isolated persistence: `game.daily_runs`, `game.daily_run_answers`
  - [ ] Official-attempt limit (one per day) + practice runs
  - [ ] Failure grace: server-error / not-started voids; strict resume (≤ 2 min, clock keeps running)
  - [ ] Daily board (`ranking.daily_challenge_entries`, `rank:daily:*`) with tie-breakers correct → total time → completion time
  - [ ] Share card + "beat my score" notification & deep link
  - [ ] Tests: attempt limits, tie-breakers & score encoding, disconnect/error handling (category-modes.md §9)
- [ ] Creator Quiz (full feature incl. authoring)
- [ ] Revenge Match (full feature)
- [ ] Question reporting
- [ ] Moderation tools
- [ ] Anti-cheat checks

## Phase 9 — Polish & Production Readiness  `[ ]`
- [ ] FCM notifications
- [ ] Shareable result cards
- [ ] Team chat
- [ ] Tournaments
- [ ] Premium subscription
- [ ] Cosmetic monetization
- [ ] Ad integration
- [ ] Security hardening
- [ ] Load testing
- [ ] E2E device testing
- [ ] Deployment docs

---

## Change log
- 2026-10-15 — Roadmap created; Phase 0 broken into atomic tasks; task 0.1.1 completed.
- 2026-10-15 — Decisions D-001…D-007 recorded (language, i18n, Glicko-2, casual unrated, seed sources, source tracking, verified-only ranked content).
- 2026-10-15 — 0.1.2 done: ADR-0001 pins Go 1.27, Flutter 3.47 / Dart 3.13, PostgreSQL 18, Redis 8.10, Compose v2; Firebase scoped to Auth + FCM.
- 2026-10-15 — 0.1.3 done: repo layout fixed (backend cmd/internal/modules with compiler-enforced boundaries, mobile feature folders, content/ for traceable seed data, docs tree). ADR-0001 notes owner machine versions (Go 1.27.0, Flutter 3.47.1).
- 2026-10-15 — Decisions D-008…D-010 recorded (monorepo, Go module path, GitHub Actions CI); Phase 1 gains CI, .gitattributes/.editorconfig and architecture-test tasks.
- 2026-10-15 — 0.1.4 done: 17 backend modules, allowed dependency table, outbox-based domain events, realtime push/route pattern, schema-per-module data ownership.
- 2026-10-15 — 0.1.5 done: ADR-0002 — stdlib net/http + slog, coder/websocket, pgx v5 + sqlc, goose, River (transactional outbox + scheduled jobs), go-redis v9, Firebase Admin v4, env v11, UUIDv7, in-house Glicko-2, real PG/Redis integration tests, golangci-lint v2.
- 2026-10-15 — 0.1.6 done: ADR-0003 — Riverpod 3 (state + DI), go_router, dio, web_socket_channel + own RealtimeClient, freezed/json_serializable, gen-l10n (TR template + EN), feature-first data/domain/presentation, client-side server-authority rules, dev/prod flavors. Pending decisions D-011…D-014 logged. Section 0.1 complete.
- 2026-10-15 — Decisions D-011…D-014 recorded (Android + iOS, com.brainvolt.app, guest/Google/Apple/email sign-in with guest upgrade, single Firebase project).
- 2026-10-15 — Decisions D-015…D-017 recorded (link conflict, guest limits, email verification).
- 2026-10-15 — 0.2.1 done: glossary with code names and Turkish UI labels; side vs team, score vs SP vs XP vs rating, answer vs play streak disambiguated; working definitions for modes flagged for 0.3.x.
- 2026-10-15 — Decisions D-016a, D-018, D-019 recorded; glossary revised (player formats section with Couple Contest, confirmed team answer modes, temporary side term).
- 2026-10-15 — 0.2.2 drafted: `docs/domain/schema-identity.md` (users, auth identity mirror, roles, profiles, settings, devices; tier rules; guest upgrade, email verification, link conflict, suspension, deletion flows). Waiting on owner for min age, avatars, retention durations.
- 2026-10-15 — Decisions D-020…D-022 recorded; 0.2.2 done (age gate, preset avatars, retention rules, ActivityReader for guest cleanup).
- 2026-10-15 — 0.2.3 drafted: `docs/domain/schema-content.md` (taxonomy, sources/licenses, language-neutral questions + per-locale text, two-level verification, question sets & daily sets, stats, OpenTDB import & CC BY-SA obligations). Waiting on owner for D-023…D-025.
- 2026-10-15 — Decisions D-023…D-025 recorded; 0.2.3 done (eligibility table updated: ranked & daily multiple-choice only; unverified creator quizzes private rooms only).
- 2026-10-15 — 0.2.4 done (draft for review): `docs/domain/schema-game.md` — Redis-live / Postgres-final split, matches with immutable config & origin, two sides (team / couple / temporary), participants with seats & tier at start, rounds with question revision & shuffled order, accepted answers only, per-side round results, lifecycle audit, MatchCompleted payload.
- 2026-10-15 — Decision D-026 recorded (match detail retention).
- 2026-10-15 — 0.2.5 drafted: `docs/domain/schema-social.md` (friends as directed pairs, blocks, invite codes, private rooms, direct/rematch/revenge challenges, teams with captain & lineups, couples; D-023 room rule = friends of host). Waiting on owner to confirm proposed limits.
- 2026-10-15 — D-027 (social limits accepted) and D-028 (temporary sides off team boards, no objection raised) recorded; 0.2.5 done.
- 2026-10-15 — 0.2.6 drafted: `docs/domain/schema-ranking.md` (entity-based Glicko-2 ratings per bucket + full change history, seasons, leagues, standings, points ledger feeding Redis leaderboards for every filter combo, snapshots, XP ledger, play streaks, badges, quests). Allowed dependency progression → ranking added. Waiting on owner for 5 proposals.
- 2026-10-15 — D-029 recorded; 0.2.6 done.
- 2026-10-15 — 0.2.7 drafted: `docs/domain/schema-moderation.md` (cases grouping reports & signals, immutable action log, sanctions & appeals, cheat signals, installation links, blocked terms, interface-based enforcement, anti-cheat pipeline, match void). game schema gains voided_* + MatchVoided. Waiting on owner for 6 proposals.
- 2026-10-15 — D-030 recorded; 0.2.7 done.
- 2026-10-15 — 0.2.8 drafted: `docs/domain/schema-platform.md` (notification preferences/inbox/push log, team & side chat, tournaments with brackets, commerce: server-verified subscriptions, entitlements, cosmetics, inventory, rewarded ads; creator quizzes; daily challenge attempts). Waiting on owner for 8 proposals.
- 2026-10-15 — D-031 recorded; 0.2.8 done.
- 2026-10-15 — 0.2.9 drafted: `docs/domain/redis-keys.md` (module prefixes, hash tags, TTL on every key, live match keys with owner lease & event stream, matchmaking, presence & routing, rooms, leaderboard ZSET naming for all filters, caches, rate limits, Redis-loss recovery). Waiting on owner for 4 proposals.
- 2026-10-15 — D-032 recorded; 0.2.9 done.
- 2026-10-15 — 0.2.10 drafted: `docs/domain/erd.md` (87 tables / 15 schemas inventory, FK-vs-dependency check passed, Mermaid diagrams, review findings F1–F5 fixed in schema docs, consolidated retention, personal-data inventory). Waiting on owner for 4 retention proposals.
- 2026-10-15 — D-033 recorded; 0.2.10 done. **Section 0.2 (data model) complete.**
- 2026-10-15 — 0.3.1 drafted: `docs/game/match-lifecycle.md` (5 states, 7 transitions with triggers/guards/effects, participant sub-states, ready flow per origin, single-writer execution model, award matrix, invariants, key tests). Waiting on owner for 4 proposals.
- 2026-10-15 — D-034 recorded; 0.3.1 done.
- 2026-10-15 — 0.3.2 drafted: `docs/game/round-lifecycle.md` (phases pre_round → question → locked → reveal → end_check, expected answerers per team mode, Shared Answer propose/lock, visibility rules, server steps, invariants, tests). Waiting on owner for 5 proposals.
- 2026-10-15 — D-035 recorded; 0.3.2 done.
- 2026-10-15 — 0.3.3 drafted: `docs/game/timing.md` (server-time-only decisions, RTT median, capped latency compensation, synchronized show_at, per-player grace, response_ms formula with worked example, client offset display-only, timer table). Waiting on owner for 5 proposals.
- 2026-10-15 — D-036 recorded; 0.3.3 done.
- 2026-10-15 — 0.3.4 drafted: `docs/game/scoring.md` (ruleset 1: base × difficulty, linear speed bonus, streak bonus, true/false half value, no negatives, worked examples, pure-function contract); `match_rounds.difficulty_used` added. Waiting on owner for 5 proposals.
- 2026-10-15 — D-037 recorded; 0.3.4 done.
- 2026-10-15 — 0.3.5 drafted: `docs/game/match-types.md` (Quick Duel, Ranked, Sudden Death, Marathon: counts, windows, curves, question types, end checks, ranked = matchmaking only, default selection rules). Waiting on owner for 6 proposals.
- 2026-10-15 — D-038 recorded; 0.3.5 done; glossary §4 definitions updated from working to final.
- 2026-10-15 — 0.3.6 drafted: `docs/game/team-modes.md` (per-mode side_correct / side_points / contribution / streak, same-scale rule, Fast Team aggregation rules average / fastest_correct / majority with worked 3v3 example). Waiting on owner for 5 proposals.
- 2026-10-15 — D-039 recorded; 0.3.6 done (turn passes on disconnect too; Average correctness over active members, points still divided by full side size).
- 2026-10-15 — 0.3.7 drafted: `docs/game/category-modes.md` (plug-in hooks, rules for all six category modes, summary matrix; owner choice needed: Daily Challenge solo score race vs matchmade 1v1). round-lifecycle §2 updated for D-039.
- 2026-10-15 — D-040…D-045 recorded; 0.3.7 done. Daily Challenge redesigned as isolated solo score race (category-modes.md rewritten; schema-game, schema-platform §6, schema-ranking §4.4, redis-keys §2a/§6.4, match-lifecycle, glossary, backend-modules updated). Phase 8 Daily Challenge broken into sub-tasks incl. required tests.
- 2026-10-15 — D-046 recorded (0.3.7 confirmations).
- 2026-10-15 — 0.3.8 drafted: `docs/game/couple-contest.md` (couple-vs-couple 2v2, allowed combinations, Sync Bonus, couple + personal couple ratings, boards, lifecycle & anti-abuse); `team.couples.last_dissolved_at` added. Waiting on owner for 5 proposals.
- 2026-10-15 — D-047 recorded; 0.3.8 done.
- 2026-10-15 — 0.3.9 drafted: `docs/game/edge-cases.md` (tie-breakers & golden rounds, reconnect windows & abuse limit, leave/forfeit matrix & awards, leaver cooldowns, owner takeover with round voiding, complete cancellation list); cancel_reason `all_sides_absent` added. Waiting on owner for 6 proposals.
- 2026-10-15 — D-048 recorded; 0.3.9 done. Owner delegated non-critical decisions (decision policy added at top).
- 2026-10-15 — 0.3.10 done (delegated D-049): `docs/game/progression-formulas.md`.
- 2026-10-15 — 0.3.11 done: `docs/game/test-catalog.md` (≈100 test IDs across 11 rule areas, mapped to unit/integration/E2E and phases). **Section 0.3 complete.**
- 2026-10-15 — 0.4.1 done (delegated D-050): `docs/api/conventions.md`. 0.4.2 done: `docs/api/rest.md` (~110 endpoints, 16 groups, auth tier, idempotency and phase per endpoint; OpenAPI generated in Phase 1).
- 2026-10-15 — 0.4.3–0.4.6 done (delegated D-051): `docs/api/websocket.md` (auth handshake, close codes, envelope & streams, heartbeat, full client/server message catalogue, replay-or-snapshot resume, fan-out).
- 2026-10-15 — 0.4.7 done: `docs/api/errors.md` (~65 stable codes, REST status + WS mapping). 0.4.8 done (delegated D-052): `docs/api/limits.md`. **Section 0.4 complete.**
- 2026-10-15 — 0.5.1–0.5.4 done (delegated D-053): `docs/game/mode-matrix.md` — 7 axes, 11 520 raw combinations → 1 178 valid (5 ranked), 21 rules (R01–R21), 6 transformations, 36 test cases + property tests. **Section 0.5 complete.**
- 2026-10-15 — 0.6 done: consistency check (all decision refs logged, no broken doc links, enum values aligned; glossary gained `origin` and `aggregation`), `docs/ARCHITECTURE.md` rewritten as overview & document map. **Phase 0 complete.** Phase 1 broken into 51 atomic tasks + 3 owner prerequisites (all original Phase 1 items kept).
- 2026-10-15 — 1.1 done: `.gitattributes` (LF, CRLF for `*.bat`/`*.cmd`/`*.ps1`), `.editorconfig`, `README.md` quick start with Windows/Git Bash notes, Makefile (`up`, `down`, `reset`, `ps`, `logs`, `psql`, `redis-cli`, `run`, `test`, `lint`, `migrate`, `gen`; each documented as its plain command). 1.2 done (delegated D-054): Compose v2 with `postgres:18.6-alpine` / `redis:8.10.2-alpine`, named volumes (PG 18 at `/var/lib/postgresql`), healthchecks, Redis AOF everysec + RDB + `noeviction`, `.env.example`, Firebase Auth emulator service. Verified: all services healthy, PG 18.6 data dir `/var/lib/postgresql/18/docker`, Redis config applied and auth enforced, emulator anonymous sign-up works, data survives `down`/`up`.
- 2026-10-15 — 1.3 done: `backend/go.mod` (`go 1.27`, `toolchain go1.27.2`, deps `golang.org/x/tools` v0.52.0, `go-cmp` v0.7.0); `cmd/api` + `cmd/worker` call `internal/app` (composition root: `Component` interface, `App.Run` stops all components together, `Main` handles SIGINT/SIGTERM and exit codes); `backend/.golangci.yml` (v2, golangci-lint v2.14.0 built with Go 1.27; misspell without locale because docs use British spelling); architecture test `internal/archtest` reads the §1 catalogue and §2 import table straight from `backend-modules.md`. Verified: `go vet`, `go test -race` (incl. real binaries stopping cleanly on SIGTERM), `golangci-lint run` 0 issues; the architecture test was shown to fail on a forbidden edge (identity → profile) and on an uncatalogued module folder.
- 2026-10-15 — 1.4.1–1.4.4 done: `platform/config` (env v11 → typed `Config`; defaults from limits.md §3; validation of env, addresses, URLs and durations, prod forbids the Firebase emulator and `demo-` projects; every error names its variable and never contains a password; `LogValue` redacts URL passwords); `platform/logging` (slog JSON/text, `request_id` / `user_id` / `match_id` from context); `platform/clock` (`Real` + deterministic `Fake` with timers, tickers, AfterFunc, Go 1.23 stop/reset semantics); `platform/ids` (UUIDv7 via google/uuid v1.6.0, canonical-form parsing, embedded timestamp). `app.Main` now loads config and builds the logger before wiring; invalid config exits 1 with one message per problem. Optional timeout overrides added to `.env.example`.
