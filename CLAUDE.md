# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Payloop is a Go subscription-billing backend. HTTP API on Gin, Temporal for subscription/webhook workflows, PostgreSQL via GORM, NATS pub/sub, Redis cache, Cedar for authorization, Paystack + Checkout.com as payment gateways, Clerk for auth.

The README's "Architecture" section is partly outdated — see "Architecture" below for what the code actually does.

## Commands

Run / build (Go 1.24):
- `go run .` — start the API server (entrypoint `main.go` → `internal/config.NewApp().Run()`). Port from `SERVER_PORT` env, default `8080`.
- `go build -o main .` — same build the Dockerfile produces.
- `go test ./...` — run all tests. Most tests live under `internal/core/domain` and `internal/adapter/{redis,sqs,nats}`.
- `go test ./internal/core/domain -run TestNextBillingDate` — run a single test.

Local stack (Postgres + Temporal + Temporal UI):
- `docker-compose -f docker/docker-compose.yml up -d` — required services. Temporal UI at `localhost:8080`, Temporal gRPC at `localhost:7233`, Postgres at `localhost:5432`.
- `temporal operator namespace create -n subscriptions` — one-time after first Temporal start. The Go client hard-codes namespace `subscriptions` in `internal/adapter/temporal/temporal.go`.

Database schema (Prisma is the source of truth, not gorm migrations):
- `pnpm prisma:push` — push operational schema (`prisma/schema.prisma`) to `DATABASE_URL`.
- `pnpm prisma:reporting:push` — push reporting schema (`schemas/reporting/schema.prisma`) to the reporting DB.
- `pnpm prisma:format` / `pnpm prisma:reporting:format` — format the schemas.
- CI runs `npx prisma migrate deploy` before building the image (see `buildspec.yml`).

Tunnels & deploy (require AWS profiles + bastion PEM):
- `pnpm tunnel:test` / `pnpm tunnel:prod` — SSH tunnel to test/prod (DB 7777, API 8888, Temporal UI 9999, Redis 6379 on test).
- `pnpm deploy:test` / `pnpm deploy:prod` — kicks off the CodeBuild pipeline.

## Architecture

Ports-and-adapters (hexagonal), not the four-layer DDD split the README describes. The directory layout:

- `internal/core/domain/` — entities, value objects, domain logic. Pure Go, no infra imports.
- `internal/core/port/` — interfaces the core depends on (`Repository`, `PubSub`, `Cache`, `Engine`, `GatewayAdapter`, `Authenticator`, `Scheduler`, etc.). Adapters implement these.
- `internal/core/service/` — application services. Take ports in their constructors. This is where business orchestration lives.
- `internal/adapter/{postgres,redis,nats,sqs,temporal,cedar,clerk,cognito,apikey,checkout_com,paystack,cron,http,memory}/` — concrete implementations of ports.
- `internal/lib/` — cross-cutting helpers (`Env`, `Logger`, `ErrorReporter`, `RequestHandler`, validator).
- `internal/config/app.go` — wiring.

### Wiring — manual DI, not FX

The README says "Uses Uber's FX library for dependency injection." That is **no longer true.** `internal/config/app.go` wires every repo/service/handler by hand in `NewApp()`. There are no `fx.Module` definitions, no `modules.go`. When adding a new service, edit `app.go`.

### The narrow-vs-orchestration service pattern (load-bearing — read before touching subscription/order code)

There is a deliberate construction-order cycle: Temporal **activities** call into services, but the Temporal **engine** is what dispatches activities. If a service depended on the engine, it could not be passed into the activity that is then registered with the engine.

The pattern in `internal/config/app.go` (and documented in `internal/core/service/subscription_orchestration.go`):

1. Build "narrow" services first that **do not** hold the engine: `SubscriptionService`, `PaymentService`, `OrderWorkflowService`. Activities receive these.
2. Build the activities (`OrderActivities`, `OutgoingWebhookActivities`) holding refs to narrow services.
3. Build the Temporal engine, registering those activities.
4. Build "engine-aware" wrappers / services last: `SubscriptionOrchestrationService` embeds `*SubscriptionService` and adds engine signaling; `OrderService` takes the engine directly. HTTP handlers depend on the engine-aware variants.

The wrapping happens at the type level (the orchestration service embeds the narrow one), so the cycle is broken statically, not papered over with setters. Preserve this — don't shortcut by giving the narrow service a back-pointer to the engine.

### Workflow engine

- `internal/adapter/temporal/temporal.go` boots a single Temporal worker on task queue `events` in namespace `subscriptions`. Workflows registered: `PaymentSuccessWorkflow`, `SubscriptionChargeReminder`, `SubscriptionWorkflow`, `OutgoingWebhookWorkflow`, `PaymentRefunded` (see `internal/adapter/temporal/workflows/`).
- Engine port: `port.Engine` exposes `StartWorkflow`, `StartSubscriptionWorkflow`, `UpdateSubscriptionWorkflow`, `CancelSubscriptionWorkflow`, `SignalSubscriptionWorkflow`. Adapter logic is engine-specific; the rest of the codebase only sees `port.Engine`.

### Payment gateways

Adapter registry in `app.go`: `map[domain.Gateway]port.GatewayAdapter` with `domain.Paystack` and `domain.CheckoutDotCom`. The `GatewayFactory` (`internal/core/service/factory.go`) looks up the gateway for an org's PSP config and returns an adapter — this avoids importing adapter packages from the service layer. Add a gateway by implementing `port.GatewayAdapter` under `internal/adapter/<name>/` and registering it in `app.go`.

### Authentication & authorization

- Authentication: `port.Authenticator` implementations are pluggable. Currently only Clerk is constructed in `app.go`; the `authenticators` slice is the FX-tag substitute referenced in the README. (Cognito and apikey adapters exist but are not wired.)
- Authorization: Cedar via `internal/adapter/cedar/`. Policies live in `policy.cedar` at repo root (copied into the Docker image). Handler signatures take `authzEngine` and call it before mutating actions — see `OrderHandler`, `CustomerHandler`, etc.

### Two databases + CDC

- `DATABASE_URL` → `payloop` (operational), `REPORTING_DATABASE_URL` → `payloop_reporting` (reports). If the reporting URL fails to connect, the code falls back to the operational DB (see `app.go:48`).
- CDC keeps reporting in sync via Postgres logical replication. The README has the manual recovery steps (drop `cdc_pub` publication, terminate replication backend, drop `cdc_slot2`) for when the CDC stream service redeploys.

## Conventions and gotchas

- `internal/config/app.go` ignores some constructed services with `_ = ...` (e.g., `metadataService`, `userService`, `cache`, `authenticators`). They are constructed for side-effects or because their dependencies are needed elsewhere; don't delete them without checking.
- `go.mod.docker` is a separate module file copied during the Docker build (see `Dockerfile`); when changing dependencies remember to update it as well or the image build breaks.
- Logger is `port.Logger` (a thin facade); zap is the backing impl via `internal/lib/logger.go`. Use the injected logger, not `log` or `fmt.Println`.
- Env loading: `lib.NewEnv()` loads `.env` via godotenv then binds known keys via viper. Add new env vars by extending the `Env` struct **and** calling `viper.BindEnv` in `NewEnv()`.
- Tests are sparse and live next to the code under test (`*_test.go`). The richest test surface today is `internal/core/domain/subscription*_test.go` — model new domain-logic tests on those.
