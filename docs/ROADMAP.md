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

## Phase 0 — Planning & Architecture  `[~]`
Output: design documents under `docs/` only. No production code.
Verification for each task: document reviewed against the requirement list + cross-references consistent.

### 0.1 Repository & codebase analysis
- [x] 0.1.1 Inventory repository, record baseline (section above)
- [ ] 0.1.2 Write `docs/adr/0001-tech-stack-and-versions.md` — pin Go, Flutter/Dart, Postgres, Redis versions; Firebase products used
- [ ] 0.1.3 Write `docs/architecture/repo-layout.md` — monorepo folder layout (backend/, mobile/, docs/, deploy/, scripts/)
- [ ] 0.1.4 Write `docs/architecture/backend-modules.md` — Go module boundaries & allowed dependencies (auth, user, content, game, realtime, matchmaking, social, team, ranking, progression, moderation, notification, chat, tournament, monetization)
- [ ] 0.1.5 Write `docs/adr/0002-backend-libraries.md` — HTTP router, WebSocket lib, DB driver/query layer, migration tool, logging, config, test tooling
- [ ] 0.1.6 Write `docs/adr/0003-mobile-architecture.md` — state management, routing, networking/WS client, DI, folder structure

### 0.2 Data models & database schema
- [ ] 0.2.1 `docs/domain/glossary.md` — canonical terms (match, round, question slot, participant, side, team, couple, MMR, SP, XP, league…)
- [ ] 0.2.2 Schema: identity — users, firebase identity link, profiles, devices/FCM tokens, settings, country
- [ ] 0.2.3 Schema: content — categories, sub-categories, questions, answer options, difficulty, locale, source & license
- [ ] 0.2.4 Schema: game — matches, match_configs, sides/teams in match, participants, rounds, match_questions, answers, results
- [ ] 0.2.5 Schema: social — friendships, friend requests, direct challenges, invite codes, private rooms, team profiles, couple profiles, rematch links
- [ ] 0.2.6 Schema: ranking & progression — ratings (per bucket), seasons, season points, XP/levels, leagues, streaks, badges, quests, leaderboard snapshots
- [ ] 0.2.7 Schema: moderation & anti-cheat — question reports, moderation actions, user sanctions, cheat signals
- [ ] 0.2.8 Schema: platform — notifications log, team chat, tournaments, subscriptions, cosmetics/inventory, ad events, creator quizzes, daily challenges
- [ ] 0.2.9 `docs/domain/redis-keys.md` — key naming, data types, TTLs (queues, live match state, presence, locks, rate limits, leaderboard ZSETs)
- [ ] 0.2.10 Consolidated ERD (`docs/domain/erd.md`, Mermaid) + index/constraint/retention review

### 0.3 Game state machine & scoring rules
- [ ] 0.3.1 Match lifecycle: states `waiting → ready → in_progress → completed | cancelled`, transitions, guards, triggers, side effects
- [ ] 0.3.2 Round / question sub-state machine (dispatched → answering → locked → revealed)
- [ ] 0.3.3 Server timer authority model (server timestamps, latency grace window, late-answer rejection)
- [ ] 0.3.4 Base scoring formula (correctness, speed bonus, streak bonus, caps)
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
- [ ] 0.6.1 Open-decisions list resolved with owner
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
