# BrainVolt

Real-time, social, competitive quiz platform — Flutter app (Android + iOS), Go modular-monolith backend, PostgreSQL, Redis, Firebase Auth & FCM.

- Plan and status: [`docs/ROADMAP.md`](docs/ROADMAP.md)
- Architecture: [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md), [`docs/architecture/`](docs/architecture/), ADRs in [`docs/adr/`](docs/adr/)
- Game rules, data model and API contracts: [`docs/game/`](docs/game/), [`docs/domain/`](docs/domain/), [`docs/api/`](docs/api/)

## Prerequisites

| Tool | Version | Notes |
|---|---|---|
| Git | any recent | On Windows: Git for Windows (includes **Git Bash**) |
| Docker | Docker Desktop / Engine with **Compose v2** (`docker compose`) | Runs Postgres, Redis and the Firebase Auth emulator |
| Go | **1.27.x** | `backend/go.mod` pins `toolchain go1.27.2`; an older 1.27 downloads it automatically (ADR-0001) |
| Flutter | **3.47.1** (Dart 3.13) | Exact version in `mobile/.flutter-version` once the app exists |
| golangci-lint | **v2.14.0**, built with Go 1.27 | `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0` — a binary built with an older Go refuses to lint Go 1.27 code |
| sqlc | **v1.31.1** | Only to regenerate DB code (`make gen`). Linux/macOS: `go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1` (needs a C compiler); Windows: download `sqlc_1.31.1_windows_amd64.zip` from the sqlc GitHub releases and put `sqlc.exe` on PATH |
| make | GNU make | Optional — every target is also a plain command (see below) |

Exact versions and the reasons for them are in [ADR-0001](docs/adr/0001-tech-stack-and-versions.md).

## Quick start

```bash
git clone https://github.com/burakkuru5534/brainvolt.git
cd brainvolt

make up      # creates .env from .env.example if missing, starts services, waits until healthy
make ps      # all three services should show "(healthy)"
```

Without `make`:

```bash
[ -f .env ] || cp .env.example .env
docker compose up -d --build --wait
docker compose ps
```

Local services (bound to `127.0.0.1` only):

| Service | Image | Address | Data |
|---|---|---|---|
| PostgreSQL | `postgres:18.6-alpine` | `localhost:5432` | volume `brainvolt_postgres_data` |
| Redis | `redis:8.10.2-alpine` (AOF everysec + RDB, `noeviction`, password) | `localhost:6379` | volume `brainvolt_redis_data` |
| Firebase Auth emulator | built from `scripts/firebase-emulator/` (project `demo-brainvolt`) | `localhost:9099` | in memory — users reset on restart |

Credentials and ports live in `.env` (git-ignored); `.env.example` documents every variable. Never commit `.env`.

## Make targets

`make help` lists them. Each one only runs the command shown here:

| Target | Plain command |
|---|---|
| `make up` | `[ -f .env ] \|\| cp .env.example .env && docker compose up -d --build --wait` |
| `make down` | `docker compose down` (keeps data) |
| `make reset` | `docker compose down -v` (**deletes** local Postgres/Redis data) |
| `make ps` / `make logs` | `docker compose ps` / `docker compose logs -f` |
| `make psql` | `docker compose exec postgres psql -U brainvolt -d brainvolt` |
| `make redis-cli` | `docker compose exec redis redis-cli` |
| `make run` | `cd backend && go run ./cmd/api` |
| `make test` | `cd backend && go test ./...` then `cd mobile && flutter test` |
| `make lint` | `cd backend && go vet ./... && golangci-lint run` then `cd mobile && flutter analyze` |
| `make migrate` | `cd backend && go run ./cmd/migrate up` (`make migrate MIGRATE=status` for other goose commands) |
| `make gen` | `cd backend && sqlc generate` then `cd mobile && flutter gen-l10n && dart run build_runner build --delete-conflicting-outputs` |

`make` loads `.env` into the environment for you. When running the Go commands by hand, load it first:

```bash
set -a; . ./.env; set +a
```

Targets that need `mobile/` print a message until the app is created (roadmap 1.7).

## Backend in containers

`make up` starts only the infrastructure; run the API locally with `make run`. To run the same image that is deployed:

```bash
make up-app        # docker compose --profile app up -d --build --wait  → migrate, api (localhost:8080), worker
make down          # stops everything, including the app containers
```

Stop the `api` container before `make run` (both use port 8080, see `BRAINVOLT_API_PORT`).

## API contract

`docs/api/openapi.yaml` (OpenAPI 3.1) is generated from the route registry and Go types. `make gen` rewrites it; `make check-gen` (and the test suite) fails when it is stale.

## Database migrations

```bash
make migrate                    # apply pending migrations   (cd backend && go run ./cmd/migrate up)
make migrate MIGRATE=status     # list applied / pending
make migrate MIGRATE=down       # roll back the latest one
```

Migrations live in `backend/migrations/` as `NNNNNN_<module>_<change>.sql` with `-- +goose Up` / `-- +goose Down` sections; `000001` installs River's job tables. `/readyz` reports `migrations: failed` until the database is up to date. After changing SQL queries, migrations or API types run `make gen` (sqlc + OpenAPI) and commit the generated files; `make check-gen` fails if they are stale.

## When something does not start

`make check-env` compares your `.env` with `.env.example`. It lists missing variables and `${VAR}` references that expand to nothing — e.g. a missing `BRAINVOLT_FIREBASE_AUTH_EMULATOR_PORT` turns the emulator address into `localhost:` and tests fail with `dial tcp 127.0.0.1:80: connection refused`. `make up`, `run`, `migrate` and `test` run it first. A variable must be defined **above** the lines that use it. Changing a port (e.g. `BRAINVOLT_POSTGRES_PORT=5433` when 5432 is taken) needs only that one line: the URLs are built from it.

## Tests

`make test` runs unit **and** integration tests. Integration tests use the Compose services through `BRAINVOLT_TEST_POSTGRES_URL`, `BRAINVOLT_TEST_REDIS_URL` and `BRAINVOLT_TEST_FIREBASE_AUTH_EMULATOR_HOST` (defined in `.env.example`; Redis tests use logical database 15). Every database test gets its own freshly migrated database (cloned from a cached template, dropped afterwards), so tests never touch your development data. Without those variables, e.g. a bare `go test ./...` with no Docker, integration tests are skipped with a hint and unit tests still run. If your `.env` predates these variables, copy the two lines from `.env.example`.

## Backend layout rules

Module boundaries from [`backend-modules.md`](docs/architecture/backend-modules.md) are checked by a test: `go test ./internal/archtest/` reads the allowed-import table from that document and fails on any edge not listed there. To add a dependency between modules, edit the table first.

## Windows notes

- `make` targets work from **PowerShell**, `cmd.exe` and **Git Bash** alike. Scripts under `scripts/` (when added) need Git Bash.
- `make` is not bundled with Windows. Install it with `winget install ezwinports.make` or `scoop install make`. Without `make`, the plain commands need the `.env` variables loaded first, which is easiest in Git Bash (`set -a; . ./.env; set +a`); `make` loads `.env` for you in every shell.
- Tools that `make` calls (`go`, `docker`, `golangci-lint`, `sqlc`) must be on `PATH`. Simplest for `sqlc.exe`: move it into `%USERPROFILE%\go\bin` (already on PATH after installing Go) and check with `sqlc version` in a new terminal.
- Line endings: the repository forces **LF** through `.gitattributes` (only `*.bat` / `*.cmd` / `*.ps1` use CRLF), so SQL, shell and YAML files keep working inside Linux containers. Recommended once per machine:
  ```bash
  git config --global core.autocrlf false
  ```
  If you cloned before `.gitattributes` existed, refresh the working tree with `git add --renormalize .`.
- Editors: `.editorconfig` sets UTF-8, LF, final newline and indentation (tabs for Go and Makefiles). VS Code needs the *EditorConfig for VS Code* extension; JetBrains IDEs support it out of the box.
- Docker Desktop must be running with the WSL 2 backend. If a port is already taken (for example a locally installed PostgreSQL on 5432), change `BRAINVOLT_POSTGRES_PORT` in `.env` — the connection URL picks it up automatically.
- `go test -race` needs a C compiler (cgo); on Windows plain `go test` is the default and CI runs the race detector.

## Continuous integration

[`.github/workflows/ci.yml`](.github/workflows/ci.yml) runs on every push to `main` and every pull request (D-010):

| Job | Steps |
|---|---|
| `backend` | starts Postgres, Redis and the Firebase Auth emulator with `docker compose` and the values of `.env.example` (same as local), `go vet`, `golangci-lint` v2.14.0, `sqlc diff`, `go test -race ./...` (unit + integration + OpenAPI drift test; `BRAINVOLT_TEST_REQUIRE_SERVICES=1` makes a missing service fail instead of skip) |
| `mobile` | Flutter from `mobile/.flutter-version`, `flutter pub get --enforce-lockfile`, `dart analyze --fatal-infos`, `flutter test`, generated code up to date (`make gen` + `git diff --exit-code`) |

There is no iOS job until the Apple Developer account exists (D-057). The same checks run locally with `make lint`, `make test` and `make check-gen`.

### Branch protection (repository owner, once)

GitHub → **Settings → Rules → Rulesets → New branch ruleset**, target branch `main`, enforcement *Active*. Two levels:

**A. Now (direct pushes to `main` stay possible, as the current workflow merges the working branch into `main`):**
- **Restrict deletions** and **Block force pushes**.
- CI still runs on every push to `main`; a red run shows on the commit and in the Actions tab.

**B. Later (recommended once more people work on the repository, or when you want CI to gate every change):** everything in A, plus
- **Require a pull request before merging** — required approvals 0 while you work alone (1 when someone joins), dismiss stale approvals on new commits;
- **Require status checks to pass** → add `backend` and `mobile` (they appear in the search box after the workflow has run once) and tick **Require branches to be up to date before merging**;
- optional: **Require linear history**.

With B, changes reach `main` only through pull requests whose `backend` and `mobile` checks are green; the owner can keep a bypass for emergencies under *Bypass list*.

## Mobile app

See [`mobile/README.md`](mobile/README.md): `cd mobile && flutter run --dart-define-from-file=env/dev.json` with the backend running. `make test`, `make lint` and `make gen` cover the app too (lint uses `dart analyze`, which also runs riverpod_lint).

## Repository layout

See [`docs/architecture/repo-layout.md`](docs/architecture/repo-layout.md). In short: `backend/` (Go), `mobile/` (Flutter), `content/` (question data), `scripts/` (dev helpers), `docs/` (design and contracts).
