# BrainVolt — Architecture overview

Proje Adı: BrainVolt

Mimari: Modular Monolith (Go backend) + Flutter App

Veritabanları: PostgreSQL (kalıcı veri), Redis (cache, matchmaking, geçici maç durumu)

Auth & FCM: Firebase

---

Real-time, social, competitive quiz platform. This page is the **entry point** to the design: what the system is, the rules that never change, and where every detail lives. Status of all work: [`ROADMAP.md`](ROADMAP.md) (phases, tasks, decision log D-001 … D-053).

## 1. System at a glance

```
 Flutter app (Android + iOS)                      Go backend (one module, modular monolith)
 ┌───────────────────────────┐   HTTPS /v1      ┌──────────────────────────────────────────────┐
 │ Riverpod · go_router · dio├─────────────────▶│ cmd/api    REST + WebSocket, live match owners │
 │ RealtimeClient (WS)       │◀════ WSS /v1/ws ═▶│ cmd/worker River jobs: outbox events, seasons, │
 │ gen-l10n TR / EN          │                  │            daily set, leaderboards, purges     │
 └────────────┬──────────────┘                  │ cmd/migrate · cmd/seed                         │
              │ sign-in / push                  └───────┬─────────────────────────┬──────────────┘
        ┌─────▼──────┐                                  │                         │
        │  Firebase  │◀── ID-token verification, FCM ───┘                         │
        │ Auth · FCM │                          ┌───────▼────────┐        ┌───────▼────────┐
        └────────────┘                          │ PostgreSQL 18  │        │ Redis 8.10     │
                                                │ system of record│        │ live state,    │
                                                │ 15 schemas      │        │ queues, boards │
                                                └────────────────┘        └────────────────┘
```

## 2. Non-negotiable rules
1. **Server authority:** only the backend decides game state, timers, correctness, scores, winners, ratings and rewards. Clients send intents; the correct answer never reaches a client before the reveal (round-lifecycle, timing).
2. **Zero scope reduction:** every feature of the master roadmap has an owning module and a phase (backend-modules §1, ROADMAP).
3. **Postgres is the system of record; Redis is live/derived/cache** and every Redis key has a TTL (redis-keys).
4. **Module boundaries are compiler-enforced** (`modules/<m>/internal/`) and checked by an architecture test in CI (backend-modules §2).
5. **Cross-module side effects go through transactional outbox events** (River), consumed idempotently (ADR-0002).
6. **No pay-to-win:** nothing bought or watched changes outcomes, rating, SP or leaderboard position (schema-platform).
7. **Rules are versioned:** scoring/timing/progression numbers form a ruleset; matches store the ruleset version (scoring).
8. **Owner commits; decisions are logged** in the ROADMAP decision log (critical ones by the owner, the rest delegated).

## 3. Document map

| Area | Documents |
|---|---|
| Decisions & tracking | [`ROADMAP.md`](ROADMAP.md) |
| Stack & libraries | [`adr/0001-tech-stack-and-versions.md`](adr/0001-tech-stack-and-versions.md) · [`adr/0002-backend-libraries.md`](adr/0002-backend-libraries.md) · [`adr/0003-mobile-architecture.md`](adr/0003-mobile-architecture.md) |
| Structure | [`architecture/repo-layout.md`](architecture/repo-layout.md) · [`architecture/backend-modules.md`](architecture/backend-modules.md) |
| Domain language | [`domain/glossary.md`](domain/glossary.md) |
| Data model | [`domain/erd.md`](domain/erd.md) (overview, diagrams, retention, personal data) · [`schema-identity`](domain/schema-identity.md) · [`schema-content`](domain/schema-content.md) · [`schema-game`](domain/schema-game.md) · [`schema-social`](domain/schema-social.md) · [`schema-ranking`](domain/schema-ranking.md) · [`schema-moderation`](domain/schema-moderation.md) · [`schema-platform`](domain/schema-platform.md) · [`redis-keys`](domain/redis-keys.md) |
| Game rules | [`match-lifecycle`](game/match-lifecycle.md) · [`round-lifecycle`](game/round-lifecycle.md) · [`timing`](game/timing.md) · [`scoring`](game/scoring.md) · [`match-types`](game/match-types.md) · [`team-modes`](game/team-modes.md) · [`category-modes`](game/category-modes.md) · [`couple-contest`](game/couple-contest.md) · [`edge-cases`](game/edge-cases.md) · [`progression-formulas`](game/progression-formulas.md) · [`mode-matrix`](game/mode-matrix.md) · [`test-catalog`](game/test-catalog.md) |
| API | [`api/conventions.md`](api/conventions.md) · [`api/rest.md`](api/rest.md) · [`api/websocket.md`](api/websocket.md) · [`api/errors.md`](api/errors.md) · [`api/limits.md`](api/limits.md) |

## 4. Key numbers (ruleset 1)
| | |
|---|---|
| Match types | Quick Duel 7 q · Ranked 10 q (MC, matchmaking only) · Sudden Death ≤ 30 · Marathon 25 |
| Formats | 1v1 · 2v2 · 3v3 · 4v4 · Couple Contest (couples vs couples) |
| Valid match configurations | 1 178 of 11 520 (5 ranked) |
| Scoring | 100 × difficulty (0.8–1.2) + speed ≤ 100 + streak 20/40 · max 260 per answer/side-round |
| Timing | server time only · RTT median of 10 · compensation ≤ 150 ms · grace 100–250 ms |
| Rating | Glicko-2 (1500/350/0.06, τ 0.5) per bucket solo / team / couple |
| Seasons | 8 weeks · leagues Bronze → Diamond · SP win +30 / draw +10 / loss −15 |
| Data | 15 Postgres schemas, 89 tables · Redis keys all with TTL |
