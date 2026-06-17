# GetPaidHQ

GetPaidHQ is a self-hostable subscription billing platform. 

It handles checkouts, subscriptions, invoicing, usage metering, dunning, and it is processor-agnostic, so you bring your own payment gateway. 

It currently supports Paystack and Checkout.com, and adding another processor means implementing a single gateway interface.

## What it does

Subscriptions are the core. You can run fixed-price plans, usage-based plans, or a hybrid of the two, with support for trials, pauses, resumes, cancellations, proration, and configurable billing anchor dates. Billing is invoice-centric, with refunds and idempotent payment handling so retries don't double-charge.

Usage metering is built in. You define meters, send usage events, and have them ingested either synchronously or through a stream, with the events stored in Postgres or ClickHouse depending on how much volume you need to handle.

When a payment fails, dunning takes over: durable recovery campaigns retry the charge on a schedule and can send customers a link to update their payment details. Outbound webhooks let you subscribe to events and react to them in your own systems.

Everything is multi-tenant — every entity is scoped to an organization — and the billing, subscription, dunning, and webhook workflows are durable, meaning they survive process restarts. Those workflows run on either Hatchet or Temporal; the two are interchangeable.

## Getting started

You'll need Docker, Go 1.26+, and make. From the repo root:

```bash
cp .env.example .env # then fill in provider secrets as needed
make up              # start Postgres, Redis, NATS, and hatchet-lite
make db-migrate-all  # apply the Goose schema migrations to all three databases
make run             # start the API
```

The database schema is managed with [Goose](https://github.com/pressly/goose) migrations under `schemas/<db>/migrations/` (operational, reporting, usage); create new ones with `make db-migrate-create name=...`. For a database that already has the schema, stamp it instead of re-running the baseline — see the migration notes via `make help`.

Hatchet needs a token minted before the first run — the full bootstrap is in [docs/internal/local-dev-hatchet.md](docs/internal/local-dev-hatchet.md). Run `make help` to see every available target.

The REST API is mounted under `/api`. The authoritative surface is `openapi.json`, regenerated at the repo root on every boot, and Swagger UI is served at `/swagger/`.

## Documentation

The `docs/` tree is the source of truth for the deeper mechanics:

- [docs/](docs/README.md) — documentation index
- [System architecture](docs/architecture/system-hexagonal.md) — the ports-and-adapters design
- [Workflows](docs/workflows/README.md) — every durable workflow, with diagrams
- [Workflow engine abstraction](docs/workflows/workflow-engine-abstraction.md) — how Hatchet and Temporal stay interchangeable
- [ADRs](docs/adr/) — accepted architectural decisions

## License

GetPaidHQ is licensed under the GNU Affero General Public License v3.0 (AGPLv3).