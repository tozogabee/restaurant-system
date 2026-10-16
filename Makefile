# Monorepo entry points. Everything below works from the repository root.
#
#   make up        start Postgres and Redis
#   make check     fmt + vet + test across every Go module
#   make proto     regenerate Go code from api/proto
#
# Go modules are discovered rather than listed, so adding a service under
# services/ needs no change here.

SHELL := /bin/bash
.DEFAULT_GOAL := help

COMPOSE := docker compose -f deploy/local/docker-compose.yml
GO_MODULES := $(patsubst %/go.mod,%,$(shell find . -name go.mod -not -path "*/node_modules/*" | sort))

# A module with no buildable packages yet (a freshly created service, or api/
# before `make proto`) would otherwise fail the loops below.
HAS_PACKAGES = go list ./... >/dev/null 2>&1

# Local dev credentials, matching deploy/local/docker-compose.yml.
PG_URL ?= postgres://restaurant:restaurant@localhost:5432/restaurant?sslmode=disable

# buf and migrate run from containers so nothing has to be installed locally.
# Set BUF=buf or MIGRATE=migrate to use a binary on your PATH instead.
BUF ?= docker run --rm -v "$(CURDIR)":/workspace -w /workspace bufbuild/buf:latest
MIGRATE ?= docker run --rm -v "$(CURDIR)":/workspace -w /workspace --network host migrate/migrate:latest

##@ General

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nTargets:\n"} \
		/^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2 } \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) }' $(MAKEFILE_LIST)

.PHONY: modules
modules: ## List the Go modules in the workspace
	@printf '%s\n' $(GO_MODULES)

##@ Local stack

.PHONY: up
up: ## Start Postgres and Redis and wait until they are healthy
	$(COMPOSE) up -d --wait

.PHONY: down
down: ## Stop the local stack, keeping data
	$(COMPOSE) down

.PHONY: reset
reset: ## Stop the local stack and delete its volumes
	$(COMPOSE) down -v

.PHONY: logs
logs: ## Tail the local stack logs
	$(COMPOSE) logs -f

.PHONY: ps
ps: ## Show the local stack status
	$(COMPOSE) ps

##@ Go

.PHONY: tidy
tidy: ## Run go mod tidy in every module
	@for mod in $(GO_MODULES); do echo "==> $$mod"; (cd $$mod && go mod tidy) || exit 1; done
	go work sync

.PHONY: fmt
fmt: ## Format every module
	@for mod in $(GO_MODULES); do (cd $$mod && gofmt -l -w .) || exit 1; done

.PHONY: fmt-check
fmt-check: ## Fail if any file is unformatted (for CI)
	@unformatted=$$(for mod in $(GO_MODULES); do (cd $$mod && gofmt -l .); done); \
	if [ -n "$$unformatted" ]; then echo "unformatted files:"; echo "$$unformatted"; exit 1; fi

.PHONY: vet
vet: ## Run go vet in every module
	@for mod in $(GO_MODULES); do \
		(cd $$mod && $(HAS_PACKAGES)) || { echo "==> $$mod (no packages, skipped)"; continue; }; \
		echo "==> $$mod"; (cd $$mod && go vet ./...) || exit 1; \
	done

.PHONY: build
build: ## Build every module
	@for mod in $(GO_MODULES); do \
		(cd $$mod && $(HAS_PACKAGES)) || { echo "==> $$mod (no packages, skipped)"; continue; }; \
		echo "==> $$mod"; (cd $$mod && go build ./...) || exit 1; \
	done

.PHONY: test
test: ## Run the unit tests in every module
	@for mod in $(GO_MODULES); do \
		(cd $$mod && $(HAS_PACKAGES)) || { echo "==> $$mod (no packages, skipped)"; continue; }; \
		echo "==> $$mod"; (cd $$mod && go test -race ./...) || exit 1; \
	done

.PHONY: test-integration
test-integration: ## Run the tests that need the local stack (make up first)
	@for mod in $(GO_MODULES); do \
		(cd $$mod && $(HAS_PACKAGES)) || { echo "==> $$mod (no packages, skipped)"; continue; }; \
		echo "==> $$mod"; (cd $$mod && go test -race -tags integration -count=1 ./...) || exit 1; \
	done

.PHONY: cover
cover: ## Run tests with a coverage summary
	@for mod in $(GO_MODULES); do \
		(cd $$mod && $(HAS_PACKAGES)) || { echo "==> $$mod (no packages, skipped)"; continue; }; \
		echo "==> $$mod"; \
		(cd $$mod && go test -race -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | tail -1) || exit 1; \
	done

.PHONY: check
check: fmt-check vet test ## Everything CI runs

##@ Contracts

.PHONY: proto
proto: ## Generate Go code from api/proto into api/gen
	$(BUF) generate --template api/buf.gen.yaml --output api api
	cd api && go mod tidy

.PHONY: proto-lint
proto-lint: ## Lint the proto definitions
	$(BUF) lint api

.PHONY: proto-breaking
proto-breaking: ## Check the protos for breaking changes against the main branch
	$(BUF) breaking api --against '.git#branch=develop,subdir=api'

##@ Migrations

# Each service owns its migrations under services/<name>/migrations.
# Usage: make migrate-new SERVICE=menu NAME=create_menu_items
.PHONY: migrate-new
migrate-new: ## Create an empty migration pair (SERVICE=, NAME=)
	@test -n "$(SERVICE)" || { echo "SERVICE is required"; exit 1; }
	@test -n "$(NAME)" || { echo "NAME is required"; exit 1; }
	@mkdir -p services/$(SERVICE)/migrations
	$(MIGRATE) create -ext sql -dir services/$(SERVICE)/migrations -seq $(NAME)

.PHONY: migrate-up
migrate-up: ## Apply a service's migrations (SERVICE=)
	@test -n "$(SERVICE)" || { echo "SERVICE is required"; exit 1; }
	$(MIGRATE) -path services/$(SERVICE)/migrations -database "$(PG_URL)&x-migrations-table=schema_migrations_$(SERVICE)" up

.PHONY: migrate-down
migrate-down: ## Roll back a service's last migration (SERVICE=)
	@test -n "$(SERVICE)" || { echo "SERVICE is required"; exit 1; }
	$(MIGRATE) -path services/$(SERVICE)/migrations -database "$(PG_URL)&x-migrations-table=schema_migrations_$(SERVICE)" down 1
