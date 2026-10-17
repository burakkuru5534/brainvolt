# BrainVolt developer shortcuts (roadmap 1.1.3).
#
# `make` is optional: every target below only runs the plain command shown in
# its comment, so the same thing works by typing it in any shell.
#
# Works from PowerShell / cmd.exe on Windows as well as from Git Bash, Linux
# and macOS: checks use make's own functions ($(wildcard), $(error)) and the
# recipes only use commands every shell understands (`cd dir && command`).

ifeq ($(OS),Windows_NT)
  # cmd.exe has no grep/awk; findstr prints the "target: ## help" lines as-is.
  HELP = @findstr /R /C:"^[a-z-]*:.*\#\# " Makefile
else
  SHELL := bash
  .SHELLFLAGS := -eu -o pipefail -c
  HELP = @grep -hE '^[a-zA-Z_-]+:.*?\#\# ' Makefile | awk 'BEGIN {FS = ":.*?\#\# "}; {printf "  %-10s %s\n", $$1, $$2}'
endif
.DEFAULT_GOAL := help

# Load .env (if present) and export it, so `go run` sees BRAINVOLT_* vars.
# Plain command (Git Bash): set -a; . ./.env; set +a
-include .env
export

COMPOSE ?= docker compose
BACKEND := backend
MOBILE  := mobile
# Goose sub-command for `make migrate`: up | down | status | version
MIGRATE ?= up

# Stop with a clear message when a project part does not exist yet
# (backend arrives in roadmap 1.3, mobile in 1.7).
require_backend = $(if $(wildcard $(BACKEND)/go.mod),,$(error backend/ is not set up yet (roadmap 1.3)))
# $(call mobile,<command>,<what is skipped>) runs <command> in mobile/ if it exists.
mobile = $(if $(wildcard $(MOBILE)/pubspec.yaml),cd $(MOBILE) && $(1),@echo mobile/ not set up yet - roadmap 1.7 - skipping $(2))

.PHONY: help env up up-app down reset ps logs psql redis-cli run test lint migrate gen check-gen

help: ## List targets
	$(HELP)

env: ## Create .env from .env.example if missing   (copy .env.example .env)
ifeq ($(wildcard .env),)
	$(file > .env,$(file < .env.example))
	@echo created .env from .env.example
endif

up: env ## Start Postgres, Redis, Firebase Auth emulator and wait until healthy   (docker compose up -d --build --wait)
	$(COMPOSE) up -d --build --wait

up-app: env ## Also build and run migrate, api and worker containers   (docker compose --profile app up -d --build --wait)
	$(COMPOSE) --profile app up -d --build --wait

down: ## Stop all services, keep data volumes   (docker compose --profile app down)
	$(COMPOSE) --profile app down

reset: ## Stop all services and DELETE local data volumes   (docker compose --profile app down -v)
	$(COMPOSE) --profile app down -v

ps: ## Show service status   (docker compose --profile app ps)
	$(COMPOSE) --profile app ps

logs: ## Follow service logs   (docker compose --profile app logs -f)
	$(COMPOSE) --profile app logs -f

psql: ## Open psql in the postgres container   (docker compose exec postgres psql -U brainvolt -d brainvolt)
	$(COMPOSE) exec postgres psql -U "$(BRAINVOLT_POSTGRES_USER)" -d "$(BRAINVOLT_POSTGRES_DB)"

redis-cli: ## Open redis-cli in the redis container   (docker compose exec redis redis-cli)
	$(COMPOSE) exec redis redis-cli

run: ## Run the API server   (cd backend && go run ./cmd/api)
	$(require_backend)
	cd $(BACKEND) && go run ./cmd/api

test: ## Run backend and mobile tests   (cd backend && go test ./... ; cd mobile && flutter test)
	$(require_backend)
	cd $(BACKEND) && go test ./...
	$(call mobile,flutter test,flutter test)

lint: ## Run linters   (cd backend && go vet ./... && golangci-lint run ; cd mobile && flutter analyze)
	$(require_backend)
	cd $(BACKEND) && go vet ./... && golangci-lint run
	$(call mobile,flutter analyze,flutter analyze)

migrate: ## Apply DB migrations; MIGRATE=down|status|version for others   (cd backend && go run ./cmd/migrate up)
	$(require_backend)
	cd $(BACKEND) && go run ./cmd/migrate $(MIGRATE)

gen: ## Regenerate code and docs/api/openapi.yaml   (cd backend && sqlc generate && go test ./internal/app -run TestOpenAPIUpToDate -update ; mobile codegen)
	$(require_backend)
	cd $(BACKEND) && sqlc generate
	cd $(BACKEND) && go test ./internal/app -run TestOpenAPIUpToDate -update
	$(call mobile,flutter gen-l10n && dart run build_runner build --delete-conflicting-outputs,mobile codegen)

check-gen: ## Fail if committed sqlc code or openapi.yaml is stale   (cd backend && sqlc diff && go test ./internal/app -run TestOpenAPIUpToDate)
	$(require_backend)
	cd $(BACKEND) && sqlc diff
	cd $(BACKEND) && go test -count=1 ./internal/app -run TestOpenAPIUpToDate
