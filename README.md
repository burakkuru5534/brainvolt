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

## Database migrations

```bash
make migrate                    # apply pending migrations   (cd backend && go run ./cmd/migrate up)
make migrate MIGRATE=status     # list applied / pending
make migrate MIGRATE=down       # roll back the latest one
```

Migrations live in `backend/migrations/` as `NNNNNN_<module>_<change>.sql` with `-- +goose Up` / `-- +goose Down` sections; `000001` installs River's job tables. `/readyz` reports `migrations: failed` until the database is up to date. After changing SQL queries or migrations run `make gen` (sqlc) and commit the generated code; `make check-gen` fails if it is stale.

## Tests

`make test` runs unit **and** integration tests. Integration tests use the Compose services through `BRAINVOLT_TEST_POSTGRES_URL`, `BRAINVOLT_TEST_REDIS_URL` and `BRAINVOLT_TEST_FIREBASE_AUTH_EMULATOR_HOST` (defined in `.env.example`; Redis tests use logical database 15). Every database test gets its own freshly migrated database (cloned from a cached template, dropped afterwards), so tests never touch your development data. Without those variables, e.g. a bare `go test ./...` with no Docker, integration tests are skipped with a hint and unit tests still run. If your `.env` predates these variables, copy the two lines from `.env.example`.

## Backend layout rules

Module boundaries from [`backend-modules.md`](docs/architecture/backend-modules.md) are checked by a test: `go test ./internal/archtest/` reads the allowed-import table from that document and fails on any edge not listed there. To add a dependency between modules, edit the table first.

## Windows notes (Git Bash)

- Run all commands from **Git Bash**, not `cmd.exe` / PowerShell — the Makefile and scripts use bash.
- `make` is not bundled with Windows. Install it with `winget install ezwinports.make` or `scoop install make`, or simply type the plain commands from the table above.
- Line endings: the repository forces **LF** through `.gitattributes` (only `*.bat` / `*.cmd` / `*.ps1` use CRLF), so SQL, shell and YAML files keep working inside Linux containers. Recommended once per machine:
  ```bash
  git config --global core.autocrlf false
  ```
  If you cloned before `.gitattributes` existed, refresh the working tree with `git add --renormalize .`.
- Editors: `.editorconfig` sets UTF-8, LF, final newline and indentation (tabs for Go and Makefiles). VS Code needs the *EditorConfig for VS Code* extension; JetBrains IDEs support it out of the box.
- Docker Desktop must be running with the WSL 2 backend. If a port is already taken (for example a locally installed PostgreSQL on 5432), change `BRAINVOLT_POSTGRES_PORT` in `.env` — the connection URL picks it up automatically.
- `go test -race` needs a C compiler (cgo); on Windows plain `go test` is the default and CI runs the race detector.

## Repository layout

See [`docs/architecture/repo-layout.md`](docs/architecture/repo-layout.md). In short: `backend/` (Go), `mobile/` (Flutter), `content/` (question data), `scripts/` (dev helpers), `docs/` (design and contracts).
