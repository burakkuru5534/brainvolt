# ADR-0002 — Backend libraries

- **Status:** Accepted
- **Date:** 2026-10-15
- **Roadmap task:** 0.1.5
- **Related:** ADR-0001 (Go 1.27, PG 18, Redis 8.10), `backend-modules.md` (outbox events, live match authority, schema-per-module)

## Context
`backend-modules.md` requires: REST + WebSocket on one server; a single-writer live match loop with authoritative timers; transactional outbox events with at-least-once, idempotent delivery; scheduled jobs (season reset, daily challenge rotation, leaderboard snapshots, quest resets); one Postgres database with a schema per module; Redis for live state, queues, presence, pub/sub, leaderboards. Guiding principle: **standard library first**, add a dependency only when it removes real, risky work.

## Decision

| Concern | Choice | Version at decision* | Why |
|---|---|---|---|
| HTTP routing & middleware | **`net/http` standard library** (`http.ServeMux` with method + path patterns) | Go 1.27 | Method/path-parameter routing is built in since Go 1.22. Middleware is plain `func(http.Handler) http.Handler`. Zero dependencies. |
| WebSocket | **`github.com/coder/websocket`** | v1.8.x | Context-aware API (fits per-connection cancellation and timeouts), concurrent-write safe, full permessage-deflate, actively maintained by Coder. ISC license. |
| Postgres driver | **`github.com/jackc/pgx/v5`** (native API + `pgxpool`) | v5.11.x | De-facto Go Postgres driver; native interface is faster than `database/sql`, supports `LISTEN/NOTIFY`, `COPY` (bulk question import), rich types. |
| Query layer | **`sqlc`** (code generation, `pgx/v5` output) | v1.31.x (dev tool) | SQL is written by hand and checked against the schema at generation time; generates type-safe Go. No ORM magic, every query reviewable. One `sqlc` config block per module → each module only sees its own schema's queries. |
| Migrations | **`github.com/pressly/goose/v3`** | v3.29.x | Plain SQL up/down files, migrations embedded into the binary (`embed`), usable as a library inside `cmd/migrate`, works with pgx via its `stdlib` adapter. |
| Outbox, background & scheduled jobs | **`github.com/riverqueue/river`** (+ `riverpgxv5` driver) | v0.49.x | Postgres-backed job queue with **transactional enqueue** — a job is inserted in the *same transaction* as the state change and becomes visible only on commit. This *is* the outbox from `backend-modules.md` §3.2, without writing our own dispatcher. Also gives retries with backoff, unique jobs (idempotency) and periodic/cron jobs for `cmd/worker`. MPL-2.0 (open-source core; the paid "Pro" add-on is **not** used). |
| Redis client | **`github.com/redis/go-redis/v9`** | v9.23.x | Official client; supports Streams, Sorted Sets, Lua/Functions, pub/sub, pipelines and Redis 8 commands. |
| Firebase | **`firebase.google.com/go/v4`** (Admin SDK) — `auth` and `messaging` packages | v4.x | Official SDK for ID-token verification and FCM sending. |
| Logging | **`log/slog`** (standard library), JSON handler in prod, text handler in dev | Go 1.27 | Structured, levelled, zero dependencies. Request/match/user IDs attached via context. |
| Configuration | **`github.com/caarlos0/env/v11`** + our own validation in `platform/config` | v11.x | Parses `BRAINVOLT_*` env vars into a typed struct (defaults, required, durations). Tiny, no transitive deps. Validation errors stop start-up with a clear message. |
| IDs | **`github.com/google/uuid`** — **UUIDv7** | v1.6.x | Time-ordered UUIDs: index-friendly in Postgres (`uuid` column), safe to expose, sortable by creation time. |
| Rating math | **In-house Glicko-2** (`ranking/internal/domain`) | — | ~200 lines, must match D-003 exactly and be fully unit-tested against the published Glickman worked example. No well-maintained Go library justifies the dependency. |
| Metrics | **`github.com/prometheus/client_golang`** | v1.x | Exposed at `/metrics`. Introduced when needed for load testing (Phase 9) but the `platform` hook is reserved now. |
| Unit tests | Standard `testing` + **`github.com/google/go-cmp`** for deep comparisons | — | No assertion framework; table-driven tests; `platform/clock` fake for all timer logic. |
| Integration tests | Real Postgres 18 + Redis 8.10: **Docker Compose locally, GitHub Actions `services:` in CI** (D-010) | — | Same images as ADR-0001. Each test package gets an isolated database created from a migrated template DB, so tests run in parallel without interfering. |
| Concurrency tests | `go test -race` always on in CI; `testing/synctest` for timer/goroutine tests | Go 1.27 | Live match loop is concurrent; races must fail CI. `synctest` runs time-based tests instantly and deterministically. |
| Lint & format | **`gofmt`**, **`go vet`**, **`golangci-lint` v2** (config in `backend/.golangci.yml`) | v2.x (dev tool) | Standard toolchain checks + curated linters (errcheck, staticcheck, gosec, revive …). |
| Architecture test | Go test using `golang.org/x/tools/go/packages` | — | Enforces the allowed-import table in `backend-modules.md`; fails CI on any unlisted edge. |

\* "Version at decision" is informational. Exact versions are pinned by `go.mod`/`go.sum` in Phase 1 and updated per ADR-0001's version policy.

### Explicitly **not** used
| Rejected | Reason |
|---|---|
| Gin / Echo / Fiber / chi | Stdlib routing now covers our needs; frameworks add their own context types and middleware ecosystems we don't need. Fiber is not `net/http`-compatible. |
| `gorilla/websocket` | Mature, but its API is not context-based and concurrent writes need manual locking; `coder/websocket` fits our cancellation model better. |
| GORM / ent / bun (ORMs) | Hide SQL; harder to reason about locking and performance on the hot match path. sqlc keeps SQL explicit and type-safe. |
| `golang-migrate` | Viable; goose chosen for its library-first API and embedded migrations in our own `cmd/migrate`. |
| Hand-written outbox dispatcher | River already provides transactional enqueue, retries, uniqueness and periodic jobs, all on our existing Postgres. |
| `testify` | Not needed; stdlib + go-cmp keeps tests plain Go. |
| Testcontainers | Would duplicate the Compose/CI service setup; may be revisited if test isolation becomes a problem. |

## How these fit together
- **Request path:** `net/http` → middleware (request ID, slog, recover, Firebase auth via `identity`, rate limit via Redis) → module handler → service → sqlc queries over `pgxpool`.
- **WebSocket path:** `coder/websocket` connection in `platform/wsserver` → `realtime` (auth, presence, routing) → module handler; outbound fan-out via go-redis pub/sub.
- **State change + event:** inside one `pgx.Tx`: sqlc write + `river.InsertTx(...)` per subscriber → commit → River worker in `cmd/worker` runs the subscriber's idempotent handler.
- **Scheduled jobs:** River periodic jobs in `cmd/worker` (only one worker executes each run).
- **Migrations:** goose SQL files in `backend/migrations/`; River's own tables are added through a goose migration so there is one migration history.

## Consequences
- Developers need two CLI tools in addition to Go: `sqlc` and `golangci-lint` (installed via `go install` / documented in Phase 1; CI installs them too).
- Generated sqlc code is **committed** so the project builds without running the generator; CI checks it is up to date (`sqlc diff`).
- River is pre-1.0 (v0.x): upgrades are reviewed per release notes and follow ADR-0001's version policy; its Postgres schema changes are applied via our goose migrations.
- The Glicko-2 implementation is our responsibility and gets property-based and reference-example tests in Phase 7.

## Sources
- https://pkg.go.dev/github.com/coder/websocket
- https://pkg.go.dev/github.com/jackc/pgx/v5
- https://github.com/sqlc-dev/sqlc/releases
- https://pkg.go.dev/github.com/pressly/goose/v3
- https://pkg.go.dev/github.com/riverqueue/river
- https://pkg.go.dev/github.com/redis/go-redis/v9
