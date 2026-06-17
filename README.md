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

**Checkouts & Payment links** — hosted checkouts and shareable links with pre-populated customer details and carts.

## Why it exists

I built the first version because I needed to process subscription payments using a local payment processor. 
The processor's native subscription processing was bare bones and lacking modern features like 
smart retries, dunning, easy details updates etc., and I couldn't find a solution that fit my needs.
Especially it needed to be easy to deploy and support, and relatively cost-effective to host.

This is the evolution of that first release (which is still running in production) and expanded to 
include more features like metering and usage based billling, and flexibility to support different technologies and payment processors.

## How it's built

**Ports and adapters** - Hexagonal architecture makes it easy to add/swap underlying technologies and vendors.

**Pluggable auth** makes user authenticators easy to change (Cognito, Clerk)

**Durable workflows** ensures scalable and fault-tolerant processing.

**Self-hostable** - Easy to deploy, AGPLv3


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

## License

GetPaidHQ is licensed under the GNU Affero General Public License v3.0 (AGPLv3).