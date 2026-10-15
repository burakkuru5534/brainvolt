# BrainVolt developer shortcuts (roadmap 1.1.3).
#
# `make` is optional: every target below only runs the plain command shown in
# its comment, so the same thing works by typing it in Git Bash / any shell.
# On Windows run `make` from Git Bash (see README.md).

SHELL := bash
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := help

# Load .env (if present) and export it, so `go run` sees BRAINVOLT_* vars.
# Plain command: set -a; . ./.env; set +a
-include .env
export

COMPOSE ?= docker compose
BACKEND := backend
MOBILE  := mobile
# Goose sub-command for `make migrate`: up | down | status | ...
MIGRATE ?= up

# Fails with a clear message when a project part does not exist yet
# (backend arrives in roadmap 1.3, mobile in 1.7).
require_backend = @test -f $(BACKEND)/go.mod || { echo "backend/ is not set up yet (roadmap 1.3)"; exit 1; }
has_mobile = test -f $(MOBILE)/pubspec.yaml

.PHONY: help env up down reset ps logs psql redis-cli run test lint migrate gen check-gen

help: ## List targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-10s %s\n", $$1, $$2}'

env: ## Create .env from .env.example if missing   ([ -f .env ] || cp .env.example .env)
	@if [ ! -f .env ]; then cp .env.example .env && echo "created .env from .env.example"; fi

up: env ## Start Postgres, Redis, Firebase Auth emulator and wait until healthy   (docker compose up -d --build --wait)
	$(COMPOSE) up -d --build --wait

down: ## Stop services, keep data volumes   (docker compose down)
	$(COMPOSE) down

reset: ## Stop services and DELETE local data volumes   (docker compose down -v)
	$(COMPOSE) down -v

ps: ## Show service status   (docker compose ps)
	$(COMPOSE) ps

logs: ## Follow service logs   (docker compose logs -f)
	$(COMPOSE) logs -f

psql: ## Open psql in the postgres container   (docker compose exec postgres psql -U $BRAINVOLT_POSTGRES_USER -d $BRAINVOLT_POSTGRES_DB)
	$(COMPOSE) exec postgres psql -U "$(BRAINVOLT_POSTGRES_USER)" -d "$(BRAINVOLT_POSTGRES_DB)"

redis-cli: ## Open redis-cli in the redis container   (docker compose exec redis redis-cli)
	$(COMPOSE) exec redis redis-cli

run: ## Run the API server   (cd backend && go run ./cmd/api)
	$(require_backend)
	cd $(BACKEND) && go run ./cmd/api

test: ## Run backend and mobile tests   (cd backend && go test ./... ; cd mobile && flutter test)
	$(require_backend)
	cd $(BACKEND) && go test ./...
	@if $(has_mobile); then cd $(MOBILE) && flutter test; else echo "mobile/ not set up yet (roadmap 1.7), skipping flutter test"; fi

lint: ## Run linters   (cd backend && go vet ./... && golangci-lint run ; cd mobile && flutter analyze)
	$(require_backend)
	cd $(BACKEND) && go vet ./... && golangci-lint run
	@if $(has_mobile); then cd $(MOBILE) && flutter analyze; else echo "mobile/ not set up yet (roadmap 1.7), skipping flutter analyze"; fi

migrate: ## Apply DB migrations; MIGRATE=down|status|... for others   (cd backend && go run ./cmd/migrate up)
	$(require_backend)
	cd $(BACKEND) && go run ./cmd/migrate $(MIGRATE)

gen: ## Regenerate code   (cd backend && sqlc generate ; cd mobile && flutter gen-l10n && dart run build_runner build -d)
	$(require_backend)
	cd $(BACKEND) && sqlc generate
	@if $(has_mobile); then cd $(MOBILE) && flutter gen-l10n && dart run build_runner build --delete-conflicting-outputs; else echo "mobile/ not set up yet (roadmap 1.7), skipping mobile codegen"; fi

check-gen: ## Fail if committed sqlc code is stale   (cd backend && sqlc diff)
	$(require_backend)
	cd $(BACKEND) && sqlc diff
