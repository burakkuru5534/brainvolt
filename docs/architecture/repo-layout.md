# Repository layout

- **Roadmap task:** 0.1.3
- **Status:** Accepted (2026-10-15)
- **Related:** ADR-0001 (versions), `backend-modules.md` (0.1.4, module list & dependency rules), ADR-0002 / ADR-0003 (libraries)

BrainVolt is a **single Git monorepo** (https://github.com/burakkuru5534/brainvolt, D-008) holding the Go backend, the Flutter app, question content, deployment files and all documentation. One repo keeps API/WebSocket contract changes, backend and mobile changes in the same commit.

This document fixes **where things go**. It creates nothing by itself: folders appear when the phase that needs them starts. Paths marked *(reserved)* are agreed now but filled later.

## Top level

```
brainvolt/
├── .github/workflows/    CI — ci.yml: Go + Flutter tests (D-010, Phase 1)
├── backend/              Go modular monolith (one Go module)
├── mobile/               Flutter app (Android + iOS)
├── content/              Question seed data + source/license records (D-005, D-006)
├── deploy/               (reserved) container & production deployment files — Phase 9
├── scripts/              Dev helper scripts (cross-platform)
├── docs/                 All design docs, ADRs, contracts, runbooks
├── docker-compose.yml    Local dev services (Postgres, Redis, Firebase Auth emulator; backend added in Phase 1)
├── .env.example          Documented env vars for Compose, Makefile and backend — no secrets (D-054)
├── Makefile              Dev task shortcuts (run, test, lint, migrate, seed)
├── .gitignore
├── .gitattributes        Line-ending rules (Phase 1 — see "Windows notes")
├── .editorconfig         Shared indentation/charset rules (Phase 1)
└── README.md             Quick start (Phase 1)
```

## `backend/`

```
backend/
├── go.mod / go.sum                     module path: github.com/burakkuru5534/brainvolt/backend (D-009)
├── cmd/                                one folder per executable; each main.go only wires & starts
│   ├── api/                            HTTP REST + WebSocket server (the game server)
│   ├── worker/                         scheduled/background jobs: season reset, daily challenge rotation,
│   │                                   leaderboard snapshots, notification fan-out
│   ├── migrate/                        applies database migrations
│   └── seed/                           imports questions from content/ (Phase 2)
├── internal/                           nothing here is importable from outside backend/
│   ├── app/                            composition root: builds config, infra and modules, registers routes/jobs
│   ├── platform/                       technical building blocks shared by all modules (no business rules)
│   │   ├── config/                     env loading + validation
│   │   ├── logging/                    structured logger
│   │   ├── postgres/                   connection pool, transactions, health check
│   │   ├── redis/                      client, health check, key helpers
│   │   ├── httpserver/                 router setup, middleware, error envelope
│   │   ├── apierror/                   error-code registry + TR/EN messages, shared by REST and WebSocket
│   │   ├── health/                     /healthz and /readyz
│   │   ├── ratelimit/                  fixed-window limits on Redis (limits.md §1)
│   │   ├── idempotency/                Idempotency-Key handling (conventions.md §9)
│   │   ├── testenv/                    integration-test service addresses (BRAINVOLT_TEST_*)
│   │   ├── testdb/                     per-test migrated Postgres databases cloned from a template
│   │   ├── events/                     transactional outbox on River (domain events between modules)
│   │   ├── wsserver/                   WebSocket hub, connection lifecycle, envelope codec
│   │   ├── firebase/                   ID-token verification, FCM client
│   │   ├── clock/                      injectable server clock (all timers use this; tests use a fake)
│   │   ├── ids/                        ID generation
│   │   └── i18n/                       locale negotiation (tr, en) — D-002
│   └── modules/                        one folder per business module (list fixed in 0.1.4)
│       └── <module>/
│           ├── <module>.go             the module's PUBLIC API: facade type, exported interfaces & DTOs
│           ├── internal/               hidden from other modules by Go's internal-package rule
│           │   ├── domain/             entities, value objects, rules, state machines — pure Go, no I/O
│           │   ├── service/            use cases orchestrating domain + stores
│           │   ├── store/              Postgres / Redis implementations
│           │   └── transport/          REST handlers, WS message handlers
│           └── *_test.go               tests live next to the code they test
├── migrations/                         numbered goose migrations (embedded), one global sequence for all modules;
│                                       000001 = River job tables (Go migration); importable by every package
├── sqlc.yaml                           one sqlc block per module → internal/modules/<m>/internal/store/sqlc
├── testdata/                           fixtures shared by integration tests
└── Dockerfile                          (Phase 1)
```

**Why `modules/<m>/internal/`:** Go refuses to compile an import of `modules/game/internal/...` from anywhere outside `modules/game/`. Module boundaries are therefore enforced by the compiler, not by convention — other modules can only use what `<module>.go` exports.

## `mobile/`

Created with `flutter create` in Phase 1; Flutter's own generated files (`android/`, `ios/`, `mobile/.gitignore`) are kept as generated.

```
mobile/
├── pubspec.yaml / pubspec.lock
├── .flutter-version                    3.47.1 (ADR-0001)
├── env/                                dev.json / prod.json for --dart-define-from-file (D-060)
├── tool/gen_error_l10n.dart            error_<CODE> texts from the backend error registry
├── l10n.yaml                           localization config
├── lib/
│   ├── main.dart                       entry point only
│   ├── app/                            app widget, routing, theme, dependency setup
│   ├── core/                           shared, feature-agnostic code
│   │   ├── network/                    REST client, error mapping
│   │   ├── realtime/                   WebSocket client, reconnect/resume (0.4.6)
│   │   ├── auth/                       Firebase Auth wrapper, token refresh
│   │   ├── l10n/                       generated localizations; source ARB files: app_tr.arb, app_en.arb
│   │   └── widgets/                    design-system widgets
│   └── features/                       one folder per user-facing feature, mirroring backend modules
│       └── <feature>/                  e.g. auth, home, lobby, match, results, matchmaking, friends,
│           ├── data/                   teams, leaderboard, profile, season, store, moderation, chat, tournament
│           ├── domain/
│           └── presentation/
├── test/                               unit & widget tests, mirroring lib/
├── integration_test/                   on-device E2E tests (Phase 9)
├── android/  ios/                      platform projects (generated)
└── assets/                             images, fonts, sounds
```

Internal structure inside `features/<feature>/` is confirmed by ADR-0003 (0.1.6).

## `content/`

Question content is kept as reviewable files, separate from code, so every question's origin is traceable (D-006).

```
content/
├── README.md                           import format, review/verification workflow
├── SOURCES.md                          every source: name, URL, license, attribution text, date imported
├── opentdb/                            questions taken from OpenTDB after license/suitability check (D-005)
│   ├── raw/                            untouched source snapshot (English)
│   └── tr/                             Turkish translations, each linked to its raw question ID
└── brainvolt/                          our own question pool (tr, optional en)
```

The import tool (`backend/cmd/seed`) refuses any question with no source or license. Verified status (D-007) is set in the database by review, not by the files.

## `docs/`

```
docs/
├── ROADMAP.md                          phase/task tracking + decision log
├── ARCHITECTURE.md                     one-page overview (expanded at Phase 0 sign-off)
├── adr/                                numbered Architecture Decision Records (0001-…)
├── architecture/                       repo-layout.md, backend-modules.md
├── domain/                             glossary, schema per area, redis-keys, ERD        (0.2)
├── game/                               state machines, scoring, mode rules, mode matrix  (0.3, 0.5)
├── api/                                REST catalog, WebSocket protocol, error codes      (0.4)
└── runbooks/                           (reserved) operations & deployment — Phase 9
```

## `scripts/` and `Makefile`

- `Makefile` holds short targets (`make up`, `make test`, `make migrate`, …) that only call `docker compose`, `go` and `flutter`.
- `scripts/` holds anything longer, plus dev-only container builds (`scripts/firebase-emulator/` — Firebase Auth emulator image used by `docker-compose.yml`, D-054). Scripts are written so they run in **Git Bash on Windows** as well as Linux/macOS (or are written in Go under `backend/cmd/` when logic is non-trivial).

## Windows notes (owner develops on Windows)
- `make` is not built into Windows: use it from Git Bash, or install via `winget`/`scoop`. Every Makefile target is also documented as the plain command it runs, so `make` is optional.
- Line endings: Phase 1 adds `.gitattributes` with `* text=auto eol=lf` (plus `*.bat`/`*.ps1` as `crlf`) so SQL, Go, shell and YAML files don't break inside Linux containers.

## Naming conventions
| Area | Rule | Example |
|---|---|---|
| Go packages | short, lowercase, no underscores | `matchmaking`, `wsserver` |
| Go files | lowercase, underscores allowed | `match_state.go`, `match_state_test.go` |
| Dart files & folders | `snake_case` | `match_screen.dart` |
| Migrations | `NNNNNN_<module>_<change>.sql`, one goose file with `-- +goose Up` / `-- +goose Down` sections (D-058); `<module>` is a catalogued module or `platform` | `000002_identity_create_schema.sql` |
| Docs | `kebab-case.md`; ADRs `NNNN-title.md` | `redis-keys.md`, `0002-backend-libraries.md` |
| Env vars | `BRAINVOLT_` prefix, upper snake case | `BRAINVOLT_POSTGRES_URL` |

## What is deliberately *not* decided here
- The list of business modules and who may call whom → `backend-modules.md` (0.1.4).
- Router, WebSocket library, DB access style, migration tool → ADR-0002 (0.1.5).
- State management, routing, DI on mobile → ADR-0003 (0.1.6).
