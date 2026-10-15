# ADR-0001 — Tech stack and pinned versions

- **Status:** Accepted
- **Date:** 2026-10-15
- **Roadmap task:** 0.1.2
- **Related decisions:** D-002 (TR + EN), D-003 (Glicko-2)

## Context
BrainVolt is a new project (empty `backend/` and `mobile/`). Before any code is written, every runtime and toolchain version must be pinned so local dev, CI and production behave identically. The existing `docker-compose.yml` pins `postgres:16-alpine` and `redis:7-alpine`; those predate this decision.

Versions below were checked against official release pages on 2026-10-15.

## Decision

### Runtimes & toolchains
| Component | Pinned version | Pin location (set in Phase 1) | Why this line |
|---|---|---|---|
| Go | **1.27** (toolchain `go1.27.2`) | `backend/go.mod` → `go 1.27` + `toolchain go1.27.2` | Newest major (released 2026-08-19); longest remaining support window (Go supports the two newest majors). |
| Flutter | **3.47.x stable** | `mobile/pubspec.yaml` → `environment.flutter: ">=3.47.0 <4.0.0"`; exact version recorded in `mobile/.flutter-version` | Current stable channel. |
| Dart | **3.13** (bundled with Flutter 3.47) | `mobile/pubspec.yaml` → `environment.sdk: ">=3.13.0 <4.0.0"` | Comes with Flutter; never installed separately. |
| PostgreSQL | **18** (image `postgres:18.6-alpine`) | `docker-compose.yml`, deployment docs | Current stable major (18.6, 2026-08-11), supported until 2030-11. PG 19 is not adopted until it has a few minor releases and managed-host support. |
| Redis | **8.10** (image `redis:8.10.2-alpine`) | `docker-compose.yml`, deployment docs | Current stable with active support. Needed features (Streams, Sorted Sets, Lua/Functions, keyspace TTLs) all present. |
| Docker Compose | **v2** (Compose Specification, no `version:` key) | `docker-compose.yml` | v1 is retired; the `version` key is obsolete and only produces warnings. |

### Firebase products
| Product | Used for | Side |
|---|---|---|
| Firebase Authentication | User sign-in; issues ID tokens | Mobile (FlutterFire) |
| Firebase Admin SDK (Go) | Verifying ID tokens on every REST call and on WebSocket handshake | Backend |
| Firebase Cloud Messaging (FCM) | Push notifications (challenges, match found, season events) — Phase 9, token storage designed in 0.2.2 | Mobile + Backend |

Firebase is used **only** for identity and push. All game data, profiles, ratings and leaderboards live in PostgreSQL/Redis — not in Firestore.

### Version policy
1. **Exact pins in lockfiles**: `go.sum`, `pubspec.lock` and image tags with a full minor/patch (never `latest`, never a bare major).
2. **Patch updates** (e.g. Go 1.27.2 → 1.27.3, PG 18.6 → 18.7): allowed any time after tests pass; recorded in the roadmap change log.
3. **Minor/major updates** (e.g. Flutter 3.47 → 3.50, PG 18 → 19, Redis 8.10 → 8.12): need a new ADR that supersedes the affected row.
4. Library-level choices (router, WebSocket, DB driver, migrations, state management…) are **not** in this ADR — see ADR-0002 (backend, task 0.1.5) and ADR-0003 (mobile, task 0.1.6).

## Consequences
- `docker-compose.yml` must move from `postgres:16-alpine` / `redis:7-alpine` to the pins above and drop the `version:` key. **This change is done in Phase 1** (Docker Compose task), not now — Phase 0 changes no runtime files.
- PostgreSQL 18 official images store data under `/var/lib/postgresql/18/docker`; the Phase 1 compose volume must mount `/var/lib/postgresql` (not `/var/lib/postgresql/data` as with ≤17).
- Redis 8.x is offered under RSALv2 / SSPLv1 / **AGPLv3** (our choice of license). BrainVolt runs Redis unmodified as a separate server, so no source-disclosure obligation applies to BrainVolt code. Managed Redis/Valkey providers remain an option for production; that choice belongs to Phase 9 deployment docs.
- Developers (Windows host) need: Go 1.27.x, Flutter 3.47.x, Docker Desktop with Compose v2. Exact setup steps go into Phase 1 docs.
- Owner machine verified 2026-10-15: Go **1.27.0**, Flutter **3.47.1**. Flutter matches. For Go, `go.mod`'s `toolchain go1.27.2` line makes the Go command fetch 1.27.2 automatically (default `GOTOOLCHAIN=auto`); installing 1.27.2 locally avoids that download. `mobile/.flutter-version` will record `3.47.1`.

## Alternatives considered
- **Go 1.26** — still supported, but drops out of support ~6 months earlier than 1.27. Rejected for a new project.
- **PostgreSQL 17** — fine, but 18 is the stable line with the longest support. Rejected.
- **PostgreSQL 19** — not yet released/mature at decision time. Revisit via new ADR.
- **Valkey instead of Redis** — viable drop-in; kept Redis per the product architecture statement. Can be revisited for production hosting in Phase 9.
- **Firestore for game state** — rejected: conflicts with "backend is sole authority" and with the relational model needed for ranking/seasons.

## Sources
- Go release history — https://go.dev/doc/devel/release
- Flutter release notes — https://docs.flutter.dev/release/release-notes
- Code With Andrea newsletter, Aug 2026 (Flutter 3.47 / Dart 3.13) — https://codewithandrea.com/newsletter/august-2026
- PostgreSQL versions — https://versionlog.com/postgresql/
- Redis release cycles & licenses — https://endoflife.date/redis
