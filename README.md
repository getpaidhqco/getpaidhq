# GetPaidHQ

GetPaidHQ is a Go subscription-billing backend. HTTP API on [Fuego](https://github.com/go-fuego/fuego), durable subscription/dunning/webhook workflows on **Hatchet or Temporal** (interchangeable), PostgreSQL via GORM, NATS pub/sub, Redis cache, Cedar for authorization, Paystack + Checkout.com as payment gateways, Clerk for auth.

> **Status**: running local-only while the deployment story is reworked. AWS pipelines and prod tunnels are paused; the CodeBuild buildspec lives at `infra/aws/buildspec.yml`.

## Table of contents

- [Architecture](#architecture)
- [Documentation](#documentation)
- [API](#api)
- [Data model](#data-model)
- [Integrations](#integrations)
- [Auth & authorization](#auth--authorization)
- [Installation](#installation)
- [Configuration](#configuration)
- [Database schema](#database-schema)
- [Development](#development)

## Architecture

Ports-and-adapters (hexagonal): the domain and application services sit at the center and depend only on `port` interfaces; adapters implement those ports and depend inward. Wiring is **manual DI** in `internal/config/app.go` — every repo/service/handler is constructed by hand.

```
internal/
├── core/
│   ├── domain/   # entities, value objects — pure Go, no infra imports
│   ├── port/     # interfaces the core depends on (Repository, Engine, GatewayAdapter, ...)
│   └── service/  # application services (business orchestration)
├── adapter/      # concrete port implementations (one dir per adapter)
│   ├── postgres/ redis/ nats/ jetstream/        # storage, cache, pub/sub, usage ingest
│   ├── hatchet/ temporal/                        # interchangeable workflow engines
│   ├── clickhouse/                               # alternative usage-event store
│   ├── cedar/ clerk/ cognito/ apikey/            # authz + authenticators
│   ├── checkout_com/ paystack/                   # payment gateways
│   ├── cron/ http/ memory/                       # scheduler, Fuego handlers, test fakes
├── lib/          # cross-cutting helpers (Env, Logger, RequestHandler, validator)
└── config/app.go # manual DI — the wiring root
```

The single most important architectural decision is the **two interchangeable workflow engines** (Hatchet default, Temporal optional) presenting the same workflow surface over the same engine-agnostic core services. See the docs below before changing any workflow/billing/dunning behavior.

## Documentation

The `docs/` tree is code-derived and is the source of truth for the deep mechanics:

- **[docs/README.md](docs/README.md)** — documentation index
- **[docs/architecture/system-hexagonal.md](docs/architecture/system-hexagonal.md)** — the hexagon, ports, dependency direction
- **[docs/workflows/](docs/workflows/README.md)** — every durable workflow (subscription runner, billing cycle, dunning, reminders, webhooks) with diagrams
- **[docs/workflows/workflow-engine-abstraction.md](docs/workflows/workflow-engine-abstraction.md)** — Hatchet ⇄ Temporal parity
- **[docs/internal/](docs/internal/README.md)** — engine-parity & subscription lifecycle, durable-runner timeouts, Hatchet architecture, local-dev setup, logging, ClickHouse primer
- **[docs/adr/](docs/adr/)** — accepted decisions (invoice-centric billing, decimal quantities, trials)
- **`CLAUDE.md`** — load-bearing conventions and guardrails to read before editing

## API

RESTful API mounted under `/api` (subscriptions, customers, payment methods, orders, products, carts/sessions, meters, usage events, webhooks, organizations, payment tokens, PSPs). The full, authoritative surface is **`openapi.json`**, regenerated at the repo root on every server boot (Fuego derives it from typed handler signatures). Swagger UI is served at `/swagger/`.

## Data model

Multi-tenant — every core entity is scoped to `orgId`.

- **Org**, **User**, **ApiKey** — tenancy and access
- **Product**, **Variant**, **Price** — product catalog
- **Cart**, **Session**, **Order**, **OrderItem** — sales
- **Customer**, **PaymentMethod**, **Cohort** — customer profile
- **Subscription**, **Payment**, **Refund**, **Invoice** — billing
- **Meter**, **MeterEvent** — usage metering
- **DunningCampaign**, **DunningAttempt**, **PaymentUpdateToken** — failed-payment recovery
- **WebhookSubscription** — outbound event notifications
- **Psp**, **Setting**, **MetadataStore**, **IdempotencyKey** — integration plumbing

Schemas are managed by Goose migrations, split by database: `schemas/app/migrations/` (operational), `schemas/usage/migrations/` (usage events), `schemas/reporting/migrations/` (reporting projection — not currently wired).

## Integrations

| Concern          | Adapter                                                      |
|------------------|-------------------------------------------------------------|
| Workflow engine  | Hatchet (default) or Temporal (`internal/adapter/{hatchet,temporal}`) |
| Pub/Sub          | NATS (`nats`), JetStream for usage ingest (`jetstream`)     |
| Database         | PostgreSQL via GORM (`postgres`)                            |
| Usage event store| Postgres (default) or ClickHouse                            |
| Cache            | Redis                                                        |
| Authorization    | Cedar                                                        |
| Authentication   | Clerk + API key (both wired); Cognito (compiled, unwired)   |
| Payment gateways  | Paystack, Checkout.com                                       |

## Auth & authorization

Authentication is pluggable via `port.Authenticator`, tried in order — `app.go` wires `{clerk, apiKey}`: Clerk first, falling through to API-key auth (`x-api-key`). API keys are HMAC'd with `API_KEY_PEPPER` before storage. The Cognito adapter is compiled but not registered.

Authorization is policy-based via Cedar. Policies live in `policy.cedar` at the repo root, loaded at startup; handlers call the authz engine before mutating actions.

## Installation

### Prerequisites

- Docker + Docker Compose v2
- Go 1.24+
- pnpm (for tunnel/deploy scripts) and `make`

### Setup

1. Install JS deps (for tunnel/deploy scripts): `pnpm install`
2. Copy `.env.example` to `.env`. Most local defaults work out of the box; fill in provider secrets (Clerk, Paystack, …) as needed.
3. Start the local stack: `make up`
   This brings up a single Postgres (host port **`10432`**) hosting four databases — `getpaidhq`, `getpaidhq_reports`, `getpaidhq_usage`, `hatchet` — plus Redis (`10379`), NATS (`10422`), and `hatchet-lite` (UI `10888`, gRPC `10707`).
4. Apply the migrations: `make db-migrate-all`
5. Mint a Hatchet token and put it in `.env` (see [docs/internal/local-dev-hatchet.md](docs/internal/local-dev-hatchet.md) for the full bootstrap):
   ```
   docker exec hatchet-lite /hatchet-admin --config /config token create \
     --name local-dev --tenant-id <tenant-id>
   # paste into HATCHET_CLIENT_TOKEN in .env
   ```
6. Run the API: `make run`

## Configuration

All runtime config is read from environment variables; `lib.NewEnv()` loads `.env` via godotenv then binds known keys via viper. **`.env.example` lists every variable the app understands** — add new vars by extending the `Env` struct *and* calling `viper.BindEnv` in `NewEnv()`.

Important keys:

- `WORKFLOW_ENGINE` — `hatchet` (default) or `temporal`
- `DATABASE_URL` — operational Postgres (always opened)
- `USAGE_DATABASE_URL` — usage-event store (falls back to `DATABASE_URL` when empty)
- `USAGE_EVENT_STORE` (`postgres` | `clickhouse`), `USAGE_INGEST_MODE` (`sync` | `jetstream`)
- `HATCHET_CLIENT_*` — Hatchet SDK config (auto-read by the SDK)
- `TEMPORAL_*` — Temporal host/namespace/task-queue (when `WORKFLOW_ENGINE=temporal`)
- `ALLOWED_ORIGINS`, `API_KEY_PEPPER`

`REPORTING_DATABASE_URL` is not opened at boot — reporting is not currently wired.

## Database schema

Schema is owned by **Goose SQL migrations** checked in under `schemas/<db>/migrations/`. The migration history is the source of truth; apply with:

```
make db-migrate            # → getpaidhq      (schemas/app/migrations/)
make db-migrate-usage      # → getpaidhq_usage (schemas/usage/migrations/)
make db-migrate-reporting  # → getpaidhq_reports (schemas/reporting/migrations/)
make db-migrate-all        # all three
```

To add a new migration: `make db-migrate-create name=add_foo`, hand-write the SQL with `-- +goose Up` / `-- +goose Down` markers, then `make db-migrate`.

### Existing databases (already schema-synced)

A fresh database (local, CI) just runs `make db-migrate-all`. A database that already has the schema (created before the Goose cutover by the old `prisma db push`) must NOT re-run the baseline — stamp it as already applied instead.

Each of the three databases has its own connection string and its own `goose_db_version` table, so run the stamp **in every database you've already synced** — connect to each of `DATABASE_URL`, `REPORTING_DATABASE_URL`, and `USAGE_DATABASE_URL` in turn and execute:

```sql
CREATE TABLE IF NOT EXISTS goose_db_version (
  id SERIAL PRIMARY KEY,
  version_id BIGINT NOT NULL,
  is_applied BOOLEAN NOT NULL,
  tstamp TIMESTAMP NULL DEFAULT now()
);
INSERT INTO goose_db_version (version_id, is_applied) VALUES (0, true), (1, true);
```

Then `make db-migrate-status` (and `db-migrate-status-reporting` / `db-migrate-status-usage`) shows the baseline as applied and future migrations run normally.

## Development

Everything runs through the **Makefile** — `make help` lists all targets. Common ones:

| Command                | What it does                                              |
|------------------------|----------------------------------------------------------|
| `make run`             | Start the API server (`go run .`)                        |
| `make build`           | Build the binary (same as the Dockerfile)                |
| `make test`            | Unit tests                                               |
| `make test-integration`| All tests incl. Postgres/Testcontainers integration tests|
| `make ci`              | `go vet` + race tests (mirrors GitHub Actions)           |
| `make up` / `make down`| Start / stop the local stack                             |
| `make db-migrate-all`  | Apply all Goose migrations (all three databases)         |
| `make db-migrate-create name=...` | Scaffold a new migration file              |
| `make db-seed`         | Seed the operational DB from `schemas/app/seed.sql`      |

CI (`.github/workflows/go-test.yml`) runs `go vet` and `go test -race`; integration tests (`//go:build integration`) are opt-in and spawn their own Postgres via Testcontainers — they never touch the local stack.

### Working with Hatchet

- UI: http://localhost:10888 (default tenant slug `default`, seeded automatically)
- gRPC: `localhost:10707`
- Hatchet's own state lives in the `hatchet` database inside the shared Postgres — handy for inspecting workflow runs from `psql` when debugging.

See `CLAUDE.md` and [docs/](docs/README.md) for architectural conventions and the engine-parity rules before making changes.
