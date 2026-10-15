# Backend modules & dependency rules

- **Roadmap task:** 0.1.4
- **Status:** Accepted (2026-10-15)
- **Related:** `repo-layout.md` (folder structure), ADR-0002 (libraries, 0.1.5), `docs/domain/*` (schemas, 0.2), `docs/game/*` (rules, 0.3/0.5)

The backend is **one Go binary family** (`cmd/api`, `cmd/worker`, `cmd/migrate`, `cmd/seed`) built from **one Go module**, split internally into business modules under `backend/internal/modules/`. Each module owns its data, its rules and its endpoints. This document fixes **which modules exist, what each owns, and who may depend on whom**.

---

## 1. Module catalogue

| # | Module | Owns (responsibilities) | Owns data | First phase |
|---|--------|-------------------------|-----------|-------------|
| 1 | `identity` | Firebase ID-token verification → internal user ID; request principal; roles (player, moderator, admin); account status (active / suspended / deleted) | users, firebase links, roles | 1 |
| 2 | `profile` | Display name, avatar, country, preferred locale (D-002), settings, public profile read model | profiles, settings | 1 |
| 3 | `content` | Categories & sub-categories; questions & answer options; TR/EN translations (D-001/002); source & license (D-006); verification status (D-007); **question selection** for matches (filters, no-repeat, verified-only for ranked); Daily Challenge question-set scheduling | categories, questions, translations, sources, daily sets | 2 |
| 4 | `realtime` | WebSocket gateway: handshake auth, connection registry, presence (online/in-match), message routing to registered handlers, delivery to users across API instances, resume support | presence, connection routing (Redis) | 4 |
| 5 | `game` | Match configuration + validation (mode matrix, 0.5); match lifecycle state machine (waiting → ready → in_progress → completed / cancelled); rounds, question dispatch, **authoritative timers**, answer ingestion, scoring; match types (Quick Duel, Ranked, Sudden Death, Marathon); team answer modes (Shared Answer, Individual Turns, Fast Team Round); category-mode **rules** (Blind Challenge, Category Master, Mixed Battle, Daily Challenge, Creator Quiz, Revenge Match); Couple Contest rules; disconnect/forfeit; results & result read model (also feeds shareable result cards) | live match state (Redis), matches, participants, answers, results (Postgres) | 3 |
| 6 | `lobby` | Pre-match gathering: direct challenges, invite codes, private rooms, room membership & ready flow up to match creation, rematch & Revenge Match creation flow | rooms, invite codes, challenges | 4–5 |
| 7 | `matchmaking` | MMR-based queues (per mode/format), expanding search ranges, party/team queueing, match proposal → creation | queues & tickets (Redis), matchmaking audit | 5 |
| 8 | `social` | Friend requests, friendships, blocks; friend-list queries (used by Friends leaderboards & challenges) | friendships, requests, blocks | 5 |
| 9 | `team` | Persistent team profiles (2–4 members) and couple profiles; membership, invites, roles, leaving/disbanding | teams, couples, memberships | 5–6 |
| 10 | `ranking` | Glicko-2 ratings (D-003) per rating bucket; Season Points; seasons & seasonal resets; leagues Bronze → Diamond; leaderboards with all filters (Daily/Weekly/Monthly/All-time × Individual/Team/Couples × Friends/Country/Global × Category) | ratings, seasons, season points, leagues, leaderboard ZSETs & snapshots | 7 |
| 11 | `progression` | XP & levels; streaks; badges; quests (daily/weekly/seasonal) | xp, streaks, badges, quests, quest progress | 7 |
| 12 | `creator` | Creator Quiz authoring: user-made quizzes, drafts, submission, publishing state; hands questions to `content` / `moderation` for review | creator quizzes, drafts | 8 |
| 13 | `moderation` | Question reports; review queues; verify/reject actions (drives D-007 status in `content`); user sanctions; anti-cheat signal collection & evaluation; moderator tools API | reports, review actions, sanctions, cheat signals | 8 |
| 14 | `notification` | FCM device tokens; notification preferences; TR/EN templates; send & delivery log; in-app notification inbox | devices, preferences, notification log | 9 (tokens stored from 1) |
| 15 | `chat` | Team chat channels & messages, delivery via `realtime`, moderation hooks | channels, messages | 9 |
| 16 | `tournament` | Tournaments: registration, brackets/rounds, scheduling, match creation through `game`, prizes | tournaments, entries, brackets | 9 |
| 17 | `commerce` | Premium subscription (store receipt validation), entitlements, cosmetics catalogue & inventory, equipped cosmetics, ad reward events | subscriptions, entitlements, cosmetics, inventory, ad events | 9 |

Every feature named in the master roadmap maps to exactly one owning module above. Shareable result cards: data from `game` results, image rendering on mobile (Phase 9).

---

## 2. Allowed dependencies (compile-time)

"A → B" means module A may import B's **public package** (`modules/B/B.go`) and call it synchronously.

```
Layer 0  platform/*                  (no module imports)
Layer 1  identity   profile   commerce
Layer 2  content   social   team   realtime
Layer 3  game
Layer 4  lobby   matchmaking   tournament   creator   chat
Layer 5  ranking   progression   moderation   notification   (event consumers)
```

| Module | May import |
|--------|-----------|
| identity | — |
| profile | identity |
| commerce | identity |
| content | identity |
| social | identity, profile |
| team | identity, profile, social |
| realtime | identity |
| game | identity, profile, content, realtime, team, commerce |
| lobby | identity, profile, social, team, game, realtime |
| matchmaking | identity, profile, team, game, realtime, ranking* |
| tournament | identity, profile, team, game, realtime |
| creator | identity, profile, content |
| chat | identity, profile, team, realtime |
| ranking | identity, profile, social, team, content |
| progression | identity, profile, content, ranking (seasons, for seasonal quests) |
| moderation | identity, profile, content, game, creator, chat |
| notification | identity, profile, social, team |

\* **matchmaking → ranking** (and **identity → game** for the guest-cleanup `ActivityReader`, see `schema-identity.md` §6) are the upward reads: matchmaking needs a player's current rating to place them in a queue. It goes through a narrow read-only interface (`ranking.RatingReader`) declared in matchmaking and **implemented** by ranking, wired in `internal/app`. matchmaking never imports ranking's package directly, so there is no import cycle.

**Rules**
1. **No cycles.** If two modules need each other, one side defines an interface and `internal/app` wires the implementation (same pattern as `RatingReader`).
2. **Public package only.** Importing `modules/X/internal/...` from outside X is impossible (Go internal rule).
3. **Not in the table = forbidden.** The architecture test `backend/internal/archtest` reads the §1 catalogue and this table **from this file** and fails CI on any import edge not listed here, on imports of a module's non-public packages, and on module folders missing from §1. Adding an edge therefore means updating this document — keep the table's two-column format (`—` for none, `*` for interface-wired reads that are not imports, `(…)` for comments).
4. **platform never imports modules.**
5. **`backend/migrations` (the schema) may be imported by any package** — `cmd/migrate`, readiness checks, test databases; it imports no internal package itself.

---

## 3. Runtime communication

### 3.1 Synchronous calls
Used when the caller needs an answer now (e.g. `game` asks `content` for N questions; `lobby` asks `game` to create a match).

### 3.2 Domain events (asynchronous)
Used when **downstream** modules must react to something that already happened, and the producer must not know about them. Example: when a match completes, `game` does not call ranking, progression, notification, tournament or moderation — it publishes `game.MatchCompleted`.

- Events are written to a **transactional outbox** table in the **same Postgres transaction** as the state change. A dispatcher in `cmd/worker` delivers them to subscribers. Result: no lost rating updates if a process crashes.
- Delivery is **at-least-once**; every handler is **idempotent** (keyed by event ID).
- Event payloads are versioned structs in the producer's public package.

Initial event catalogue (expanded in 0.2 / 0.3):

| Event | Producer | Consumers |
|-------|----------|-----------|
| `identity.UserRegistered` | identity | profile, progression, notification |
| `identity.UserStatusChanged` | identity | matchmaking, lobby, realtime, chat |
| `identity.AccountTierChanged` | identity | ranking, social, team, notification |
| `identity.GuestAbandoned` | identity | matchmaking, lobby, realtime |
| `identity.UserDeleted` | identity | all modules holding personal data |
| `identity.SessionStarted` | identity | moderation |
| `content.QuestionVerified` / `QuestionRejected` | content | creator, moderation |
| `social.FriendshipCreated` / `Removed` | social | ranking (friends boards), notification |
| `team.MemberJoined` / `MemberLeft` / `TeamDisbanded` | team | chat, ranking, notification |
| `lobby.ChallengeSent` / `ChallengeAccepted` | lobby | notification |
| `matchmaking.MatchFound` | matchmaking | notification |
| `game.MatchStarted` | game | realtime (presence), notification |
| `game.MatchCompleted` | game | ranking, progression, tournament, moderation (anti-cheat), notification |
| `game.MatchCancelled` | game | ranking (no change, audit), tournament, matchmaking |
| `game.MatchVoided` | game | ranking, progression, tournament |
| `game.DailyRunCompleted` | game | ranking (daily board), progression, notification |
| `moderation.ReportFiled` / `SanctionApplied` | moderation | identity (status), notification |
| `ranking.SeasonEnded` / `LeagueChanged` | ranking | progression, notification, commerce (season rewards) |
| `progression.BadgeEarned` / `QuestCompleted` / `LevelUp` | progression | notification |
| `commerce.SubscriptionChanged` | commerce | notification |
| `tournament.RoundScheduled` / `TournamentFinished` | tournament | notification, progression |

### 3.3 Real-time push
`realtime` exposes a `Publisher` interface (send to user / to match / to team). Modules call it; with several API instances, `realtime` fans out through Redis pub/sub so the instance holding the socket delivers it.

Inbound WebSocket messages are routed by type prefix (`match.*`, `lobby.*`, `chat.*`, …) to handlers that each module **registers** at start-up in `internal/app`. `realtime` therefore never imports `game`, `lobby` or `chat`.

### 3.4 Live match authority
`game` keeps live match state in Redis and processes each match on **one owner** at a time (single-writer: one goroutine per live match, ownership held as a Redis lease). All timers, answer acceptance and scoring run there using `platform/clock`. Clients only send intents (`ready`, `answer`); they never send scores or results. Details: 0.3.1–0.3.3.

---

## 4. Data ownership rules
1. **One PostgreSQL database, one schema per module** (`identity.users`, `content.questions`, `game.matches`, …). A module reads and writes **only its own schema**.
2. **Foreign keys only along allowed import edges** (e.g. `game` → `content.questions`, any module → `identity.users`). Other cross-module references are stored as plain IDs.
3. **Transactions never span modules.** Cross-module consistency goes through outbox events.
4. **Redis keys are prefixed by module** (`game:`, `mm:`, `rt:`, `rank:` …); key catalogue in `docs/domain/redis-keys.md` (0.2.9).
5. Migrations live in one sequence (`backend/migrations/`) with the module name in each file name.

---

## 5. Which process runs what
| Process | Runs |
|---------|------|
| `cmd/api` | REST + WebSocket for all modules; live match owners (`game`); matchmaking loops (`matchmaking`) |
| `cmd/worker` | Outbox dispatcher; scheduled jobs: daily challenge rotation (`content`), leaderboard snapshots & season resets (`ranking`), quest resets (`progression`), tournament scheduling, notification fan-out, stale-match cleanup (`game`) |
| `cmd/migrate` | Schema migrations |
| `cmd/seed` | Question import from `content/` (`content`) |

Both `api` and `worker` are built from the same modules; only the wiring in `internal/app` differs.

---

## 6. Module → roadmap phase map
| Phase | Modules introduced or extended |
|-------|-------------------------------|
| 1 | platform/*, identity, profile (minimal) |
| 2 | content |
| 3 | game (engine, no network) |
| 4 | realtime, game (live), lobby (ready flow) |
| 5 | matchmaking, social, team, lobby (challenges, invite codes, private rooms, rematch) |
| 6 | team, game (2v2/3v3/4v4, team answer modes live, Couple Contest) |
| 7 | ranking, progression |
| 8 | content/game/lobby (special modes complete), creator, moderation |
| 9 | notification, chat, tournament, commerce; hardening across all |
