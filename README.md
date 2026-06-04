# Payloop

Payloop is a Go subscription-billing backend. HTTP API on Gin, Hatchet for durable subscription/webhook workflows, PostgreSQL via GORM, NATS pub/sub, Redis cache, Cedar for authorization, Paystack + Checkout.com as payment gateways, Clerk for auth.

> **Status**: we're currently running local-only while we rework the deployment story. AWS pipelines and prod tunnels are paused.

## Table of contents

- [Architecture](#architecture)
- [API endpoints](#api-endpoints)
- [Data model](#data-model)
- [Integrations](#integrations)
- [Auth & authorization](#auth--authorization)
- [Installation](#installation)
- [Configuration](#configuration)
- [Database schema](#database-schema)
- [Development](#development)

## Architecture

Ports-and-adapters (hexagonal). See `CLAUDE.md` for the load-bearing details (narrow-vs-orchestration service pattern, engine-construction order, etc.).

```
internal/
├── core/
│   ├── domain/   # entities, value objects, pure Go
│   ├── port/     # interfaces the core depends on
│   └── service/  # application services (business orchestration)
├── adapter/
│   ├── postgres/      # GORM repository implementations
│   ├── hatchet/       # workflow engine adapter
│   ├── redis/         # cache adapter
│   ├── nats/          # pub/sub adapter
│   ├── sqs/           # AWS SQS adapter (FIFO event queue)
│   ├── cedar/         # authorization adapter
│   ├── clerk/         # Clerk auth provider
│   ├── cognito/       # AWS Cognito auth provider (unwired)
│   ├── apikey/        # API-key auth provider (unwired)
│   ├── checkout_com/  # Checkout.com payment gateway
│   ├── paystack/      # Paystack payment gateway
│   ├── cron/          # cron scheduler
│   ├── http/          # Gin handlers
│   └── memory/        # in-memory adapters used for tests
├── lib/                # cross-cutting helpers (Env, Logger, RequestHandler, ...)
└── config/app.go       # manual DI — every repo/service/handler is wired here
```

### Key components

- **Web framework**: Gin
- **Workflow engine**: Hatchet (durable workflows, event-driven scheduling). The runner workflow is long-lived per subscription and sleeps until next charge or until it receives an update/cancel event.
- **Database**: two PostgreSQL databases (operational + reporting) managed via Prisma — `db push` only, no migrations are checked in
- **Event system**: NATS for pub/sub
- **Cache**: Redis
- **Authorization**: Cedar policy engine (`policy.cedar`)
- **DI**: manual wiring in `internal/config/app.go` (no Uber FX)

## API endpoints

RESTful API mounted under `/api`. Main groups:

**Subscriptions**
- `GET /api/subscriptions` — list subscriptions
- `GET /api/subscriptions/:id` — get details
- `GET /api/subscriptions/:id/payments` — list payments
- `PUT /api/subscriptions/:id/pause` / `…/resume` / `…/cancel` — lifecycle
- `PUT /api/subscriptions/:id/billing-anchor` — update billing anchor
- `PATCH /api/subscriptions/:id` — update details

**Customers** / **Payment methods**
- `POST /api/customers` — create
- `POST /api/customers/:id/payment-methods` — add payment method
- `PUT /api/customers/:id/payment-methods/:pmid` — update payment method
- `GET /api/payment-methods/:id` — get details

Plus endpoints for orders, products, sessions, carts, webhooks, webhook subscriptions, organizations, reports, and PSPs. See `openapi.json` (regenerated at the repo root on every server boot) for the full surface.

## Data model

- **Org**, **User**, **ApiKey** — tenancy and access
- **Product**, **Variant**, **Price** — product catalog
- **Cart**, **Session**, **Order**, **OrderItem** — sales
- **Customer**, **PaymentMethod**, **Cohort** — customer profile
- **Subscription**, **Payment**, **Refund** — billing
- **WebhookSubscription** — outbound event notifications
- **Psp**, **Setting**, **MetadataStore**, **IdempotencyKey** — integration plumbing

The canonical schema lives in `schemas/getpaidhq/schema.prisma`; the reporting projection is at `schemas/reporting/schema.prisma`.

## Integrations

| Concern              | Adapter                                |
|----------------------|----------------------------------------|
| Workflow engine      | Hatchet (`internal/adapter/hatchet`)   |
| Pub/Sub              | NATS                                   |
| Queue                | AWS SQS (FIFO)                         |
| Cache                | Redis                                  |
| Database             | PostgreSQL via GORM                    |
| Authorization        | Cedar                                  |
| Authentication       | Clerk (active), Cognito + API key (compiled, unwired) |
| Payment gateways     | Paystack, Checkout.com                 |
| Email                | Resend, Loops                          |
| Token vault          | AES (local), AWS Secrets Manager (hosted) |

## Auth & authorization

Authentication is pluggable via `port.Authenticator`. Clerk is the only authenticator wired in `app.go` today; Cognito and the API-key adapters are compiled but not registered.

Authorization is policy-based via Cedar. Policies live in `policy.cedar` at the repo root and are loaded at startup.

## Installation

### Prerequisites

- Docker + Docker Compose v2
- Go 1.24+
- pnpm (for Prisma scripts)

### Setup

1. Clone the repo and install JS deps:
   ```
   pnpm install
   ```
2. Copy `.env.example` to `.env` and fill in any required secrets (Clerk, Paystack, etc.). Most local defaults work out of the box.
3. Start the stack:
   ```
   docker compose -f docker/docker-compose.yml up -d
   ```
   This brings up a single Postgres (host port `6432`) that hosts three databases — `getpaidhq`, `getpaidhq_reports`, `hatchet` — and a `hatchet-lite` container (UI on `8888`, gRPC on `7077`).
4. Push the Prisma schemas:
   ```
   pnpm prisma:push
   pnpm prisma:reporting:push
   ```
5. Mint a Hatchet token and put it in `.env`:
   ```
   TENANT_ID=$(psql "postgresql://postgres:postgres@localhost:6432/hatchet" -tA -c "SELECT id FROM \"Tenant\" WHERE slug='default';")
   docker exec hatchet-lite /hatchet-admin --config /config token create --name local-dev --tenant-id "$TENANT_ID"
   # paste the returned token into HATCHET_CLIENT_TOKEN in .env
   ```
6. Run the API:
   ```
   go run .
   ```

## Configuration

All runtime config is read from environment variables. `.env.example` lists every variable the app understands, with empty values. Drop a `.env` next to it for local development — `lib.NewEnv()` loads it via godotenv.

Important keys:

- `WORKFLOW_ENGINE=hatchet` — the only supported value while we're local-only
- `DATABASE_URL` / `REPORTING_DATABASE_URL` — operational + reporting Postgres URLs
- `HATCHET_CLIENT_*` — Hatchet SDK config (auto-read by the SDK)
- `GPHQ_*` — legacy prefix kept on side-vars; the Go app reads un-prefixed names

There is a `config.yml` checked in for historical reasons. It is not the source of truth — env vars override anything in it.

## Database schema

Prisma is the schema source of truth, but there are **no migrations checked in** — the local-only reset starts from a single base model. To sync:

```
pnpm prisma:push                 # → getpaidhq
pnpm prisma:reporting:push       # → getpaidhq_reports
```

Both commands use the local Prisma 6 install (`pnpm exec`), not `pnpm dlx` — Prisma 7 changed the datasource syntax in a way our schema does not yet support.

When we re-add a deployment pipeline, migrations will be regenerated from the current state.

## Development

### Common commands

- `go run .` — start the API server
- `go build -o main .` — same build the Dockerfile produces
- `go test ./...` — run unit tests.
- `go test -tags=integration ./...` — run all tests including Postgres integration tests (uses Testcontainers).

### Working with Hatchet

- UI: http://localhost:8888 (default tenant slug `default`, seeded automatically)
- gRPC: `localhost:7077`
- Hatchet's own state is in the `hatchet` database inside our shared Postgres — useful for inspecting workflow runs from `psql` when debugging

### Stack lifecycle

```
docker compose -f docker/docker-compose.yml up -d         # start
docker compose -f docker/docker-compose.yml down          # stop, keep data
docker compose -f docker/docker-compose.yml down -v       # stop + wipe volumes (clean slate)
```

The Postgres init script (`docker/init/01-create-databases.sql`) recreates `getpaidhq_reports` and `hatchet` alongside the default `getpaidhq` database whenever the volume is fresh.

### Reporting database

`getpaidhq_reports` is created but no trigger mechanism is wired up to populate it yet. Reads against the reporting schema will return empty data until a replacement is in place.

See `CLAUDE.md` for architectural conventions, the engine-construction-order pattern, and gotchas before making changes.
