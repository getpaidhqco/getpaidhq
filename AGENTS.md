# AGENTS.md

Guidance for Claude Code and other agents working in this repo. For the user-facing overview see `README.md`; for deep mechanics see `docs/` (index: `docs/README.md`). 

## Project

GetPaidHQ is a subscription-billing backend with REST API, cli, Postgres datastore and pluggable adapters for everything.

## Commands

Everything runs through the **Makefile** — `make help` lists all targets. Essentials:

- `make run` — start the API server (`go run .`; port from `SERVER_PORT`, default `8080`)
- `make build` — build the binary the Dockerfile produces
- `make test` — unit tests; `make test-integration` — incl. Postgres/Testcontainers e2e
- `make ci` — `go vet` + race tests (mirrors GitHub Actions)
- `make up` / `make down` — local stack (Postgres, Redis, Hatchet, NATS)
- `make db-migrate-all` — apply Goose migrations to all three databases

Local stack details and the Hatchet token bootstrap: `docs/internal/local-dev-hatchet.md`. Workflow engine selection: `WORKFLOW_ENGINE=hatchet|temporal` (see parity rule below).

### Test database isolation

**Tests MUST NEVER touch the developer's local docker-compose database** — it carries hand-seeded data. Enforced by construction:

- Integration tests gate on `//go:build integration`. The shared harness in `internal/adapter/storage/storagetest` spawns a **fresh `postgres:17-alpine` testcontainer per run** + applies the Goose baseline, and exposes `RunConformance(t, factory)` — the same suite both `postgresgorm` and `postgrespgx` run against their own `RepoSet`. The dev DB at `localhost:10432` is never touched.
- No test code reads `DATABASE_URL`, calls `lib.NewEnv()`, or `config.NewApp()` — the only paths to the dev DSN.

Adding a DB-touching test: put it in `storagetest` so both drivers exercise it; scope rows with `uniqueOrg(t)` + `cleanupOrg(t, ...)`. Seed through repo `Create` methods (the `RepoSet`), never a raw `*gorm.DB`, so the test is storage-agnostic.

## Architecture

Ports-and-adapters (hexagonal): `internal/core/{domain,port,service}` at the center (pure Go + interfaces + orchestration), `internal/adapter/*` implementing the ports, `internal/lib/` for cross-cutting helpers. Full map: `docs/architecture/system-hexagonal.md` and `docs/internal/hexagonal-mapping-pattern.md`.

**Wiring is manual DI** in `internal/config/app.go` (`NewApp()`) — every repo/service/handler constructed by hand. Add a service by editing `app.go`.

**Storage adapters are grouped by category** under `internal/adapter/storage/<impl>`: `postgresgorm` (GORM, default) and `postgrespgx` (hand-written `jackc/pgx/v5`) both implement the repository ports. `DB_DRIVER=gorm|pgx` (default `gorm`) selects which set `app.go` wires; only one runs at a time. They must stay at **100% behavioural parity** — same rows, domain values, errors (`port.ErrNotFound`, unique/FK conflicts) and tx semantics. Parity is enforced by one shared conformance suite, `internal/adapter/storage/storagetest` (`RunConformance(t, factory)`), which each adapter's `//go:build integration` test runs against its own `RepoSet`. Other multi-impl adapter categories follow the same `internal/adapter/<category>/<impl>` + `<category>test` conformance shape.

### Narrow-vs-orchestration service pattern 

There is a deliberate construction-order cycle: workflow steps call services, but the engine dispatches those steps — so a service can't depend on the engine. The fix (documented in `internal/core/service/subscription_orchestration.go`):

1. Build "narrow" services that don't hold the engine (`SubscriptionService`, `PaymentService`, `OrderWorkflowService`); steps receive these.
2. Build step bundles, then the engine, registering the steps.
3. Build engine-aware wrappers last (`SubscriptionOrchestrationService` *embeds* `*SubscriptionService` and adds signaling; `OrderService` takes the engine). HTTP handlers use the engine-aware variants.

The cycle is broken at the type level by embedding — preserve that. Don't give a narrow service a back-pointer to the engine.

### Workflow engine — parity rule 

Two interchangeable engines: **Hatchet** (`internal/adapter/hatchet/`) and **Temporal** (`internal/adapter/temporal/`), selected by `WORKFLOW_ENGINE` in `app.go`. Only one runs at a time.

**Every change to workflow / billing / dunning / reminder behavior MUST produce the same observable outcome on both engines.** Parity means same *behaviour*, not identical *code* — the two use deliberately opposite execution models (Hatchet = cron + per-org fan-out; Temporal = one long-lived `SubscriptionWorkflow` + `ContinueAsNew`). Keep shared *logic* in `core/` so both literally share it; only *orchestration* is per-adapter (Temporal reaches services through **activities** — workflow code stays deterministic). A change landing on one engine silently breaks the other.

Mental model, lifecycle, and the workflow/signal/keys inventory: `docs/internal/engine-parity-and-subscription-lifecycle.md`, `docs/workflows/workflow-engine-abstraction.md`, `docs/workflows/`. Engine ports `port.Engine` / `port.DunningEngine` are satisfied by both adapters; `Start*Workflow` is idempotent via deterministic ids + reuse policies.

Pubsub→signal fan-in (`subscription.*` topics → engine signals) is owned by `service.SubscriptionEventBridge`, not the adapters — add topic→signal mappings there. `UpdateSubscriptionWorkflow` is **fire-and-forget** (pushes an event, returns immediately); don't assume synchronous acknowledgment when reading post-call state.

### Dunning

Failed subscription charges auto-open a `DunningCampaign` and a per-campaign durable runner that walks immediate then progressive retries against a resolved (and snapshotted) `DunningConfig`; escalation policy (recover/suspend/cancel) lives in `DunningService.UpdateCampaignWithAttemptResult`. Control signals: `dunning.pause/.resume/.cancel` and `dunning_pm_updated:*`. Payment-update magic-links are `PaymentUpdateToken`s under `/api/payment-tokens/*`. Full flow: `docs/workflows/dunning-recovery.md`. Code: `internal/core/{domain,service}/dunning*.go`, `internal/adapter/{postgres,hatchet}/.../dunning_*.go`, `internal/adapter/http/dunning_handler.go`.

### Payment gateways

Adapter registry in `app.go`: `map[domain.Gateway]port.GatewayAdapter` (`domain.Paystack`, `domain.CheckoutDotCom`). `GatewayFactory` (`internal/core/service/factory.go`) resolves the adapter for an org's PSP config without importing adapter packages. Add one by implementing `port.GatewayAdapter` under `internal/adapter/<name>/` and registering it in `app.go`.

### Authentication & authorization

- **Authn**: `port.Authenticator`s tried in order — `app.go` wires `{clerkAuth, apiKeyAuth}` (Clerk first, falling through to `x-api-key`). API keys are HMAC'd with `API_KEY_PEPPER`. Cognito exists but is unwired.
- **Authz**: Cedar (`internal/adapter/cedar/`), policies in `policy.cedar` at repo root. Handlers take `authzEngine` and call it before mutating actions.

### Usage metering & event ingestion

Metered billing records `meter_events` into a dedicated store, scaled/retained independently of the operational DB. Swappable via env:

- **Event store** `USAGE_EVENT_STORE`: `postgres` (default) | `clickhouse`.
- **Ingestion** `USAGE_INGEST_MODE`: `sync` (default) | `jetstream` (NATS JetStream + background batch consumer). Behind the `EventIngestor` port.
- Endpoints: `POST /api/usage/events`; meters under `/api/meters`. Meter-event ids are `NULL` when absent (never `""`); dedup index is defined in the Goose baseline (`schemas/usage/migrations/00001_baseline.sql`).

### Databases the app opens

- `DB_DRIVER` (`gorm` default | `pgx`) picks the storage adapter (`internal/adapter/storage/postgresgorm` vs `postgrespgx`); both open the same DSNs below.
- `DATABASE_URL` → `getpaidhq` (operational) — always opened.
- `USAGE_DATABASE_URL` → `getpaidhq_usage` — separate pool when set; falls back to `DATABASE_URL` when empty.
- `REPORTING_DATABASE_URL` is **not** opened — reporting is not wired. `internal/adapter/storage/postgresgorm/report_repo.go` is a stub (logs once, returns zero). To enable it: rewrite each method against `schemas/reporting/migrations/00001_baseline.sql` (the reporting schema baseline), add a service + handler, wire in `app.go`, register routes in `internal/config/server.go`.

## Conventions and gotchas

- **Wiring root**: server setup is `internal/config/server.go` (`BuildServer`), shared by `NewApp()`. Middleware order: CORS (`ALLOWED_ORIGINS`) → `AuthnWrapperMiddleware` (stores `port.AuthUser` on ctx under `middleware.AuthUserKey`; handlers read via `handler.AuthUserFrom(c)`; preserves the onboarding bypass for `POST /api/organizations`). Some constructed services are kept as `_ = ...` for side-effects/deps — don't delete without checking.
- **Transactions**: no per-request tx. Services needing multi-row atomicity use `port.TxManager` (`s.tx.RunInTx(ctx, ...)`); the tx propagates via ctx and repos use `dbFromCtx(ctx, r.db)`. Post-commit side effects (workflow starts, pubsub) go *after* `RunInTx` returns nil — never inside the closure (a rollback would orphan them). First user: `OrderService.CompleteOrder`.
- **Validation / errors**: one `*validator.Validate` from `lib.NewValidator` (registers `iso4217`); DTOs use `validate:"..."` tags. Handlers return `ApiError` (`{code,message,details}`); Fuego's own errors marshal the same shape via `handler.ApiErrorSerializer`.
- **Env**: `lib.NewEnv()` loads `.env` (godotenv) then binds via viper. Add a var by extending the `Env` struct **and** `viper.BindEnv` in `NewEnv()`. `.env.example` lists every var; the active `.env` is gitignored.
- **Logging**: use the injected `port.Logger` (`log/slog` via `internal/lib/logger.go`), not `log`/`fmt`. GORM and Hatchet logs are bridged into the same handler. See `docs/internal/logging.md`.
- **Tests** live next to code (`*_test.go`). Strongest coverage in `internal/core/domain` and `internal/adapter/http` (real httptest harness with real Cedar authz + authn middleware); `internal/core/service` is lighter; DB behaviour via `//go:build integration` tests.
