# Makefile — single entry point over the three underlying runners:
#   go            (build / run / test)
#   docker compose (local infra: docker/docker-compose.yml)
#   pnpm scripts  (tunnels, deploy — see package.json)
#
# Run `make` or `make help` to list targets.

COMPOSE := docker compose -f docker/docker-compose.yml
BIN     := main

.DEFAULT_GOAL := help

## ---- Build / run -----------------------------------------------------------

.PHONY: run
run: ## Start the API server (go run .)
	go run .

.PHONY: build
build: ## Build the server binary (same as the Dockerfile)
	go build -o $(BIN) .

.PHONY: build-cli
build-cli: ## Build the gphq CLI binary to bin/gphq
	go build -o bin/gphq ./cmd/gphq

.PHONY: install-cli
install-cli: ## go install the gphq CLI
	go install ./cmd/gphq

.PHONY: docs-cli
docs-cli: ## Regenerate the CLI markdown reference (docs/cli/reference)
	rm -rf docs/cli/reference && mkdir -p docs/cli/reference
	go run ./cmd/gphq docs --dir docs/cli/reference

.PHONY: clean
clean: ## Remove build artifacts
	rm -f $(BIN)

## ---- Test / lint -----------------------------------------------------------

.PHONY: test
test: ## Run unit tests
	go test ./...

.PHONY: test-integration
test-integration: ## Run all tests incl. Postgres/Testcontainers integration tests
	go test -tags=integration ./...

.PHONY: test-race
test-race: ## Run unit tests with the race detector (CGO required)
	CGO_ENABLED=1 go test -race ./...

.PHONY: test-integration-race
test-integration-race: ## Race-detect the full suite incl. Testcontainers integration tests
	CGO_ENABLED=1 go test -race -tags=integration ./...

.PHONY: vet
vet: ## go vet
	go vet ./...

.PHONY: ci
ci: vet test-race ## Mirror the GitHub Actions job (vet + race tests)

.PHONY: fmt
fmt: ## Format Go sources
	go fmt ./...

.PHONY: tidy
tidy: ## Tidy go.mod / go.sum
	go mod tidy

## ---- Local infra (Postgres, Redis, Hatchet, NATS) --------------------------

.PHONY: up
up: ## Start the local stack (detached)
	$(COMPOSE) up -d

.PHONY: down
down: ## Stop the local stack
	$(COMPOSE) down

.PHONY: logs
logs: ## Tail local stack logs
	$(COMPOSE) logs -f

## ---- Database schema (Goose migrations) ------------------------------------

GOOSE        := go tool goose
GOOSE_DRIVER := postgres

# $(call goose,<migration-dir>,<dbstring>,<args...>)
define goose
	GOOSE_DRIVER=$(GOOSE_DRIVER) GOOSE_DBSTRING="$(2)" GOOSE_MIGRATION_DIR=$(1) $(GOOSE) $(3)
endef

.PHONY: db-migrate
db-migrate: ## Apply operational migrations -> DATABASE_URL
	$(call goose,schemas/app/migrations,$(DATABASE_URL),up)

.PHONY: db-migrate-reporting
db-migrate-reporting: ## Apply reporting migrations -> REPORTING_DATABASE_URL
	$(call goose,schemas/reporting/migrations,$(REPORTING_DATABASE_URL),up)

.PHONY: db-migrate-usage
db-migrate-usage: ## Apply usage-store migrations -> USAGE_DATABASE_URL
	$(call goose,schemas/usage/migrations,$(USAGE_DATABASE_URL),up)

.PHONY: db-migrate-all
db-migrate-all: db-migrate db-migrate-reporting db-migrate-usage ## Apply all three schemas

.PHONY: db-migrate-status
db-migrate-status: ## Show operational migration status
	$(call goose,schemas/app/migrations,$(DATABASE_URL),status)

.PHONY: db-migrate-status-reporting
db-migrate-status-reporting: ## Show reporting migration status
	$(call goose,schemas/reporting/migrations,$(REPORTING_DATABASE_URL),status)

.PHONY: db-migrate-status-usage
db-migrate-status-usage: ## Show usage-store migration status
	$(call goose,schemas/usage/migrations,$(USAGE_DATABASE_URL),status)

.PHONY: db-migrate-down
db-migrate-down: ## Roll back the last operational migration
	$(call goose,schemas/app/migrations,$(DATABASE_URL),down)

.PHONY: db-migrate-down-reporting
db-migrate-down-reporting: ## Roll back the last reporting migration
	$(call goose,schemas/reporting/migrations,$(REPORTING_DATABASE_URL),down)

.PHONY: db-migrate-down-usage
db-migrate-down-usage: ## Roll back the last usage-store migration
	$(call goose,schemas/usage/migrations,$(USAGE_DATABASE_URL),down)

.PHONY: db-migrate-create
db-migrate-create: ## Scaffold a new operational migration: make db-migrate-create name=add_foo
	$(call goose,schemas/app/migrations,$(DATABASE_URL),-s create $(name) sql)

.PHONY: db-seed
db-seed: ## Seed the operational DB from schemas/app/seed.sql
	psql -v ON_ERROR_STOP=1 "$(DATABASE_URL)" -f schemas/app/seed.sql

## ---- Tunnels / deploy (AWS profiles + bastion PEM required) ----------------

.PHONY: tunnel-test
tunnel-test: ## SSH tunnel to the test environment
	pnpm tunnel:test

.PHONY: tunnel-prod
tunnel-prod: ## SSH tunnel to the prod environment
	pnpm tunnel:prod

.PHONY: deploy-test
deploy-test: ## Kick off the test CodeBuild pipeline
	pnpm deploy:test

.PHONY: deploy-prod
deploy-prod: ## Kick off the prod CodeBuild pipeline
	pnpm deploy:prod

## ---- Help ------------------------------------------------------------------

.PHONY: help
help: ## List available targets
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| sort \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
