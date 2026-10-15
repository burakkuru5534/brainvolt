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

## Status legend
`[ ]` not started · `[~]` in progress · `[x]` done · `[!]` blocked / needs decision

## Phase overview

| # | Phase | Status |
|---|-------|--------|
| 0 | Planning & Architecture | [~] |
| 1 | Core Infrastructure | [ ] |
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

---

## Phase 0 — Planning & Architecture  `[~]`
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
- [~] 0.3.4 Base scoring formula (correctness, speed bonus, streak bonus, caps)
- [ ] 0.3.5 Match type rules: Quick Duel, Ranked, Sudden Death, Marathon
- [ ] 0.3.6 Team answer mode rules: Shared Answer, Individual Turns, Fast Team Round
- [ ] 0.3.7 Category mode rules: Blind Challenge, Category Master, Mixed Battle, Daily Challenge, Creator Quiz, Revenge Match
- [ ] 0.3.8 Couple Contest rules
- [ ] 0.3.9 Ties, tie-breakers, forfeit, abandonment, disconnect/reconnect windows, cancellation conditions
- [ ] 0.3.10 MMR, Season Points and XP formulas; league thresholds; seasonal reset rule
- [ ] 0.3.11 State-machine & scoring test-case catalog (drives Phase 3 tests)

### 0.4 API & WebSocket contracts
- [ ] 0.4.1 Conventions: versioning, Firebase ID-token auth, error envelope, pagination, idempotency keys, time format
- [ ] 0.4.2 REST endpoint catalog per module (`docs/api/rest.md`)
- [ ] 0.4.3 WS handshake, auth, envelope (type, seq, ack, ts), heartbeat
- [ ] 0.4.4 WS client → server messages
- [ ] 0.4.5 WS server → client events
- [ ] 0.4.6 Reconnect / resume protocol (state snapshot, missed-event replay)
- [ ] 0.4.7 Error-code registry (REST + WS)
- [ ] 0.4.8 Rate limits & payload size limits

### 0.5 Game mode combination matrix & validation
- [ ] 0.5.1 Define axes: match type × format (1v1, 2v2, 3v3, 4v4, couple) × team answer mode × category mode × ranked/unranked
- [ ] 0.5.2 Fill matrix: allowed / forbidden + reason for every combination
- [ ] 0.5.3 Declarative validation rules for match configuration (input to Phase 3 validator)
- [ ] 0.5.4 Matrix test-case table (valid + invalid combos)

### 0.6 Review & sign-off
- [~] 0.6.1 Open-decisions list resolved with owner (D-001…D-007 recorded; further decisions added as they arise)
- [ ] 0.6.2 Cross-document consistency check; Phase 0 marked done

---

## Phase 1 — Core Infrastructure  `[ ]`
- [ ] Go project setup (module, layout per 0.1.3/0.1.4)
- [ ] Configuration loading (env-based, validated)
- [ ] Structured logging
- [ ] Health checks (liveness, readiness incl. Postgres & Redis)
- [ ] PostgreSQL migration setup
- [ ] Flutter app skeleton
- [ ] Firebase Auth integration (mobile sign-in + backend token verification)
- [ ] Docker Compose local dev environment (volumes, healthchecks, env file, backend service)
- [ ] Makefile targets (run, test, migrate, lint)
- [ ] `.gitattributes` (LF line endings) and `.editorconfig`
- [ ] Module-boundary architecture test (allowed dependency graph from `backend-modules.md`)
- [ ] GitHub Actions CI — `.github/workflows/ci.yml`: Go vet/lint/test (with Postgres & Redis services) + Flutter analyze/test (D-010)

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
- [ ] Daily Challenge (full feature)
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
