# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Payloop is a Go subscription-billing backend. HTTP API on Fuego, Hatchet for durable subscription/webhook workflows, PostgreSQL via GORM, NATS pub/sub, Redis cache, Cedar for authorization, Paystack + Checkout.com as payment gateways, Clerk for auth.

The README's "Architecture" section is partly outdated — see "Architecture" below for what the code actually does.

## Commands

Run / build (Go 1.24):
- `go run .` — start the API server (entrypoint `main.go` → `internal/config.NewApp().Run()`). Port from `SERVER_PORT` env, default `8080`.
- `go build -o main .` — same build the Dockerfile produces.
- `go test ./...` — run all tests. Most tests live under `internal/core/domain` and `internal/adapter/{redis,sqs,nats}`.
- `go test ./internal/core/domain -run TestNextBillingDate` — run a single test.

Local stack (single Postgres shared by app + Hatchet, plus Hatchet Lite):
- `docker compose -f docker/docker-compose.yml up -d` — required services. Hatchet UI at `localhost:8888`, Hatchet gRPC at `localhost:7077`, Postgres at `localhost:6432`.
- The shared Postgres exposes three databases (auto-created by `docker/init/01-create-databases.sql`): `getpaidhq` (app), `getpaidhq_reports` (reporting), `hatchet` (Hatchet's own internal store).

Workflow engine bootstrap:
- `WORKFLOW_ENGINE=hatchet` (the only supported engine; default in `lib.NewEnv()`).
- First-time bootstrap (after the stack is up):
  1. The default tenant + admin user are seeded automatically. Tenant id is in the `Tenant` table of the `hatchet` DB (slug `default`).
  2. Mint a token: `docker exec hatchet-lite /hatchet-admin --config /config token create --name local-dev --tenant-id <tenant-id>`. The `--config /config` flag is required to load the auto-generated encryption keyset.
  3. Set in `.env`: `HATCHET_CLIENT_TOKEN=<token>`. The other client vars (`HATCHET_CLIENT_HOST_PORT=localhost:7077`, `HATCHET_CLIENT_NAMESPACE=getpaidhq`, `HATCHET_CLIENT_TLS_STRATEGY=none`) are already there.

Database schema (Prisma is the source of truth, no migrations — clean-slate `db push` only):
- `pnpm prisma:push` — push operational schema (`schemas/getpaidhq/schema.prisma`) to `DATABASE_URL`.
- `pnpm prisma:reporting:push` — push reporting schema (`schemas/reporting/schema.prisma`) to `REPORTING_DATABASE_URL`.
- `pnpm prisma:format` / `pnpm prisma:reporting:format` — format the schemas.
- There are no Prisma migrations checked in. `db push` syncs the schema directly. The previous migrations were thrown away as part of the local-only reset; once we deploy again, migrations will be regenerated from this base.

Tunnels & deploy (require AWS profiles + bastion PEM):
- `pnpm tunnel:test` / `pnpm tunnel:prod` — SSH tunnel to test/prod (DB 7777, API 8888, Redis 6379 on test).
- `pnpm deploy:test` / `pnpm deploy:prod` — kicks off the CodeBuild pipeline.

## Architecture

Ports-and-adapters (hexagonal), not the four-layer DDD split the README describes. The directory layout:

- `internal/core/domain/` — entities, value objects, domain logic. Pure Go, no infra imports.
- `internal/core/port/` — interfaces the core depends on (`Repository`, `PubSub`, `Cache`, `Engine`, `GatewayAdapter`, `Authenticator`, `Scheduler`, etc.). Adapters implement these.
- `internal/core/service/` — application services. Take ports in their constructors. This is where business orchestration lives.
- `internal/adapter/{postgres,redis,nats,sqs,hatchet,cedar,clerk,cognito,apikey,checkout_com,paystack,cron,http,memory}/` — concrete implementations of ports.
- `internal/lib/` — cross-cutting helpers (`Env`, `Logger`, `ErrorReporter`, `RequestHandler`, validator).
- `internal/config/app.go` — wiring.

### Wiring — manual DI, not FX

The README says "Uses Uber's FX library for dependency injection." That is **no longer true.** `internal/config/app.go` wires every repo/service/handler by hand in `NewApp()`. There are no `fx.Module` definitions, no `modules.go`. When adding a new service, edit `app.go`.

### The narrow-vs-orchestration service pattern (load-bearing — read before touching subscription/order code)

There is a deliberate construction-order cycle: Hatchet **workflow steps** call into services, but the Hatchet **engine** is what dispatches those steps. If a service depended on the engine, it could not be passed into the step that is then registered with the engine.

The pattern in `internal/config/app.go` (and documented in `internal/core/service/subscription_orchestration.go`):

1. Build "narrow" services first that **do not** hold the engine: `SubscriptionService`, `PaymentService`, `OrderWorkflowService`. Steps receive these.
2. Build the step bundles (`OutgoingWebhookSteps`, `DunningSteps`) holding refs to narrow services.
3. Build the Hatchet engine, registering those steps.
4. Build "engine-aware" wrappers / services last: `SubscriptionOrchestrationService` embeds `*SubscriptionService` and adds engine signaling; `OrderService` takes the engine directly. HTTP handlers depend on the engine-aware variants.

The wrapping happens at the type level (the orchestration service embeds the narrow one), so the cycle is broken statically, not papered over with setters. Preserve this — don't shortcut by giving the narrow service a back-pointer to the engine.

### Workflow engine

Hatchet is the only engine. The wiring lives in `internal/config/app.go` (right after the narrow services are built).

- `internal/adapter/hatchet/hatchet.go` boots a single worker named `getpaidhq-events`. Workflows: `payment-success` (DAG), `payment-refunded`, `outgoing-webhook`, `billing-cycle` (DAG), `subscription-charge-reminder` (durable), `subscription-runner` (durable; long-running, one per subscription), `dunning-runner` (durable; long-running, one per failed charge), `dunning-attempt` (DAG; one per retry inside a dunning campaign).
- Engine ports: `port.Engine` exposes `StartWorkflow`, `StartSubscriptionWorkflow`, `UpdateSubscriptionWorkflow`, `CancelSubscriptionWorkflow`, `SignalSubscriptionWorkflow`. `port.DunningEngine` exposes `StartDunningWorkflow`, `SignalDunningWorkflow`, `CancelDunningWorkflow`. The Hatchet engine type satisfies both.

**Update semantics — fire-and-forget.** `UpdateSubscriptionWorkflow` pushes a user event (`update:<updateName>:<orgId>:<subId>`) and returns immediately; the durable runner observes the event in its select loop, usually within seconds. Callers in `subscription_orchestration.go` `pubsub.Publish` after, so downstream observers are unaffected; do **not** assume synchronous acknowledgment when reading post-call state.

### Dunning

`internal/core/domain/dunning*.go`, `internal/core/service/dunning*.go`, `internal/adapter/postgres/dunning_repo.go`, `internal/adapter/hatchet/{steps,workflows}/dunning_*.go`, `internal/adapter/http/dunning_handler.go`.

Failed subscription charges automatically open a `DunningCampaign` (the `DunningOrchestrationService` subscribes to `subscription.payment.charge.failed`) and a `dunning-runner` durable task is started per campaign. The runner walks two phases against a resolved `DunningConfig`:

1. **Immediate retries** — short waits, only when `InitialFailureReason` matches `ImmediateRetries.FailureTypes` (transient / network / rate-limit).
2. **Progressive retries** — long waits with customer communications dispatched before each attempt.

Each retry inside the runner spawns the `dunning-attempt` DAG (one task: `execute-attempt`), reads back the resulting `DunningAttempt`, and hands it to `DunningService.UpdateCampaignWithAttemptResult` which owns the escalation policy (recover / suspend / cancel). Terminal exits: campaign status ∈ {recovered, failed, cancelled, expired}.

Control signals respected at every wait:
- `dunning_signal:dunning.pause` / `.resume` / `.cancel` — driven by the HTTP API
- `dunning_pm_updated:<orgId>:<campaignId>` — driven by payment method update flows; triggers an immediate retry

Configurations are scoped (`organization`, `customer_segment`, `subscription_tier`, `customer`, `ab_test`) and priority-ordered; the active highest-priority config wins. A snapshot is stored on the campaign at start so mid-flight policy changes don't break a running campaign.

Payment-update tokens (`PaymentUpdateToken`) are magic-links delivered to customers as part of dunning communications. Lifecycle endpoints under `/api/payment-tokens/*` (verify / activate) and `/api/admin/subscriptions/:id/payment-tokens` (admin create).

### Payment gateways

Adapter registry in `app.go`: `map[domain.Gateway]port.GatewayAdapter` with `domain.Paystack` and `domain.CheckoutDotCom`. The `GatewayFactory` (`internal/core/service/factory.go`) looks up the gateway for an org's PSP config and returns an adapter — this avoids importing adapter packages from the service layer. Add a gateway by implementing `port.GatewayAdapter` under `internal/adapter/<name>/` and registering it in `app.go`.

### Authentication & authorization

- Authentication: `port.Authenticator` implementations are pluggable. Currently only Clerk is constructed in `app.go`; the `authenticators` slice is the FX-tag substitute referenced in the README. (Cognito and apikey adapters exist but are not wired.)
- Authorization: Cedar via `internal/adapter/cedar/`. Policies live in `policy.cedar` at repo root (copied into the Docker image). Handler signatures take `authzEngine` and call it before mutating actions — see `OrderHandler`, `CustomerHandler`, etc.

### Two databases

- `DATABASE_URL` → `getpaidhq` (operational), `REPORTING_DATABASE_URL` → `getpaidhq_reports` (reports). If the reporting URL fails to connect, the code falls back to the operational DB (see `app.go:48`).
- The trigger that populates the reporting DB has been removed; `ReportService.ProcessDataChange` is still in place and will be hooked up to a replacement mechanism later.

## Conventions and gotchas

- `internal/config/app.go` ignores some constructed services with `_ = ...` (e.g., `metadataService`, `userService`, `cache`). They are constructed for side-effects or because their dependencies are needed elsewhere; don't delete them without checking.
- HTTP layer runs on [`go-fuego/fuego`](https://github.com/go-fuego/fuego) — Gin was removed. The OpenAPI spec is generated from typed handler signatures (`fuego.ContextWithBody[T]` / `fuego.ContextNoBody`) and committed as `openapi.yml` via `go run ./cmd/openapi-export`. Swagger UI is served at `/swagger/` by the running app.
- Server wiring lives in `internal/config/server.go` (`BuildServer`) and is shared by `NewApp()` and the exporter. Global middleware order in `BuildServer`: CORS (rs/cors, configured via `ALLOWED_ORIGINS` — comma-separated allowlist, `*` enables wildcard for dev only) then `AuthnWrapperMiddleware`, which stores the resolved `port.AuthUser` on `r.Context()` under the typed `middleware.AuthUserKey`. Handlers read it via `handler.AuthUserFrom(c)`. Onboarding bypass for `POST /api/organizations` on `ErrOnboardingRequired` is preserved inside that middleware.
- The HTTP layer no longer wraps each request in a DB transaction. The previous `DatabaseTrx` middleware was already FX-era dead code that didn't compile against the current `*gorm.DB`. Transactional consistency is the repository/service layer's responsibility — adapters that need atomic multi-statement work should open their own `tx := db.Begin()` inside the service method, or use `lib.Database.WithTx` once that lands.
- Request validation runs against a single `*validator.Validate` built by `lib.NewValidator(logger)`; the `iso4217` rule is registered there. DTOs use `validate:"..."` struct tags (renamed from gin's `binding:"..."`).
- Error envelope: handlers return `ApiError` (`{code,message,details}`); Fuego is wired with `fuego.WithErrorSerializer(handler.ApiErrorSerializer)` so its own `BadRequestError`/`UnauthorizedError`/etc. also marshal in the same shape.
- Raw-body endpoints (PSP webhooks signed against the unparsed body) opt out via `fuego.PostStd` — see `internal/adapter/http/webhook_handler.go`.
- Logger is `port.Logger` (a thin facade); zap is the backing impl via `internal/lib/logger.go`. Use the injected logger, not `log` or `fmt.Println`.
- Env loading: `lib.NewEnv()` loads `.env` via godotenv then binds known keys via viper. Add new env vars by extending the `Env` struct **and** calling `viper.BindEnv` in `NewEnv()`. Local-only setup uses `WORKFLOW_ENGINE=hatchet`. Hatchet vars (`HATCHET_CLIENT_TOKEN`, `HATCHET_CLIENT_HOST_PORT`, `HATCHET_CLIENT_NAMESPACE`, `HATCHET_CLIENT_TLS_STRATEGY`) are the canonical names the Hatchet SDK auto-reads.
- `.env.example` lists every var the app reads. The active `.env` is gitignored; previous environment-specific copies (`.env.local`, `.env.prod`, `.env.test`, `docker/.env`) live under `.temp/` while we run local-only, and will be restored once we redo the deployment.
- Tests are sparse and live next to the code under test (`*_test.go`). The richest test surface today is `internal/core/domain/subscription*_test.go` — model new domain-logic tests on those.
