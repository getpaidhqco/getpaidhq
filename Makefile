# Makefile — single entry point over the three underlying runners:
#   go            (build / run / test)
#   docker compose (local infra: docker/docker-compose.yml)
#   pnpm scripts  (Prisma schema push, tunnels, deploy — see package.json)
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

## ---- Database schema (Prisma db push — no migrations) ----------------------

.PHONY: db-push
db-push: ## Push operational schema -> DATABASE_URL
	pnpm prisma:push

.PHONY: db-push-reporting
db-push-reporting: ## Push reporting schema -> REPORTING_DATABASE_URL
	pnpm prisma:reporting:push

.PHONY: db-push-usage
db-push-usage: ## Push usage-event-store schema -> USAGE_DATABASE_URL
	pnpm prisma:usage:push

.PHONY: db-push-all
db-push-all: db-push db-push-reporting db-push-usage ## Push all three schemas

.PHONY: db-constraints
db-constraints: ## Apply raw CHECK constraints + triggers (run after db-push)
	psql "$(DATABASE_URL)" -f schemas/app/constraints.sql

.PHONY: db-format
db-format: ## Format all Prisma schemas
	pnpm prisma:format && pnpm prisma:reporting:format && pnpm prisma:usage:format

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
