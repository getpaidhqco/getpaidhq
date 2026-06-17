# GetPaidHQ

Open-source subscription billing for any payment processor - self-hostable subscription billing that plugs into any gateway.

It handles checkouts, subscriptions, invoicing, usage metering, dunning, and it is processor-agnostic, so you bring your own payment gateway. 

It currently supports Paystack and Checkout.com, and adding another processor means implementing a single gateway interface.

## What it does

**Subscriptions** — fixed-price, usage-based, or hybrid plans. Trials, pauses,
resumes, cancellations, proration, plan changes, and configurable billing anchor
dates.

**Invoicing** — invoice-centric billing with line items, invoice history, credit
notes, and document sequencing. Idempotent payment handling means retries never
double-charge.

**Pricing & products** — a product catalog with variants and prices, supporting
multiple pricing schemes and tiered pricing, across multiple billing intervals and currencies.

**Usage metering** — define meters, send usage events, and ingest them. Drives usage-based and hybrid plans.

**Dunning** — durable recovery campaigns retry failed charges on a schedule,
with configurable scope, customer communications, payment-update
tokens so customers can fix their own details, and dunning analytics.

**Discounts** — a discount and redemption system applied across orders and
subscriptions.

**Payment links** — shareable links with pre-populated customer details and carts.

**Checkout** — carts and sessions backing a hosted checkout flow.

**Webhooks** — outbound webhook subscriptions to react to billing events in your
own systems.

**Reporting** — built-in reporting endpoints over billing and dunning data.

**Multi-tenant by construction** — every entity is scoped to an organization,
with users, roles, and API keys per org.

**Durable workflows** — billing, subscription, dunning, and webhook workflows.

## How it's built

**Ports and adapters.** easy to swop underlying technologies and add new adapters.

**Pluggable auth.** pluggable user authenticators (Cognito, Clerk)

Authorization is policy-based through Cedar (`policy.cedar`).

**Durable where it counts.** Anything that touches money — billing runs, dunning
campaigns, webhook delivery — runs as a durable workflow that resumes cleanly
after a crash or restart.

**Self-hostable, no lock-in.** AGPLv3, your infrastructure, your processor, your
data.

Built in Go. See [System architecture](docs/architecture/system-hexagonal.md)
for the full picture.

## Getting started

You'll need Docker, Go 1.26+, and make. From the repo root:

```bash
cp .env.example .env # then fill in provider secrets as needed
make up              # start Postgres, Redis, NATS, and hatchet-lite
make db-migrate-all  # apply the Goose schema migrations to all three databases
make run             # start the API
```

The database schema is managed with [Goose](https://github.com/pressly/goose) migrations under `schemas/<db>/migrations/` (operational, reporting, usage); create new ones with `make db-migrate-create name=...`.

Hatchet needs a token minted before the first run — the full bootstrap is in [docs/internal/local-dev-hatchet.md](docs/internal/local-dev-hatchet.md). Run `make help` to see every available target.

The REST API is mounted under `/api`. The spec is `openapi.json`, regenerated at the repo root on boot, and Swagger UI optionally served at `/swagger/`.

## Documentation

- [docs/](docs/README.md) — documentation index
- [System architecture](docs/architecture/system-hexagonal.md) — the ports-and-adapters design
- [Workflows](docs/workflows/README.md) — every durable workflow, with diagrams
- [Workflow engine abstraction](docs/workflows/workflow-engine-abstraction.md) — how Hatchet and Temporal stay interchangeable
- [ADRs](docs/adr/) — accepted architectural decisions

## License

GetPaidHQ is licensed under the GNU Affero General Public License v3.0 (AGPLv3).