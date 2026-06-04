# Usage-based metering

**Date:** 2026-06-04
**Goal:** Add usage metering to gphq-server — define meters, ingest usage events, add them up over a billing period, and attach the result as usage line items on the per-cycle invoice. Usage events are stored behind a swappable interface so we can run and compare a Postgres backend and a ClickHouse backend that produce the same numbers.

**Depends on:** `docs/superpowers/specs/2026-06-04-invoice-centric-billing.md` (Spec A) — the per-cycle `Invoice` + `InvoiceLineItem` model this builds on. **Read it first.**

**Also:** `docs/internal/clickhouse-primer.md` (storage/backends), ADRs `0001` (decimal line item), `0002` (invoice-centric billing), `0003` (trials bill usage), and `CONTEXT.md` (glossary — terms used here exactly).

## Problem

Spec A made every billing run produce an itemized `Invoice` settled by a `Payment`, but only for fixed prices (one base line per cycle). There is still **no metering**: no way to define a thing-to-measure, record how much a customer used, add it up, and turn it into an invoice line. There is also no store suited to a high-volume append-only event stream.

We want usage pricing (e.g. "$0.10 per API call", per-seat, tiered rates over total usage), and we want to try both Postgres and ClickHouse for the event store, on our own data, before committing.

## Decision

Pricing usage is two steps, modelled separately:

1. **Measure** — turn raw usage events into one number (the quantity used). A new `BillableMetric` defines *what* to measure and *how to add it up* (count, sum, max, unique-count, latest, weighted-sum).
2. **Price** — turn that quantity into money via the `Price`'s scheme (`Fixed` / `Graduated` / `Volume`). This math is **net-new** — see below.

At billing time, for a metered subscription, the period's usage is aggregated and priced into **`kind = usage` `InvoiceLineItem`s on the same per-cycle `Invoice`** Spec A already builds. There is no separate ledger and no per-event charge — usage is billed in arrears on the cycle invoice.

**Measuring is separate from billing.** A usage event names a **Customer** and a **Metric**, and may *optionally* name a **Subscription**; it never names a price. You can send usage before any subscription (or even any customer) exists. Events are customer-scoped, stored behind a swappable **`EventStore`** (Postgres or ClickHouse) in a **separate database** so they scale independently.

> **Correction (was wrong in an earlier draft):** the pricing scheme math does **not** already exist. `PriceScheme` is only an enum (`internal/core/domain/price_types.go:25`) with no implementation, and `Price` has no tier table. `domain.PriceUsage` and the tier ladder are net-new here (and shared back with Spec A's base-line builder). Also: `Tiered` is collapsed into `Graduated` — they're the same thing; the schemes are **Fixed / Graduated / Volume**.

## 1. What we reuse vs add

| Capability | In gphq | Status |
|---|---|---|
| Per-cycle invoice + line items | `Invoice` / `InvoiceLineItem` (Spec A) | reuse |
| Pricing scheme math | `PriceScheme` enum exists; **no implementation** | **net-new** (`domain.PriceUsage`) |
| Meter definition | `BillableMetric` | **new** (operational DB) |
| Usage event | `MeterEvent` | **new** (separate usage DB) |
| Event ingestion API | `POST /api/usage/events` | **new** |
| Adding usage up | `EventStore` (Postgres + ClickHouse) | **new** |
| Usage charge | `kind = usage` `InvoiceLineItem` (Spec A) | reuse — **no new ledger** |
| Customer external id | `Customer.external_id` | **new field** (immutable once set) |
| Event transport | NATS | reuse |
| Billing trigger | `billing-cycle` → `InvoiceService` (Spec A) | reuse |

## 2. Domain model

Two persistence conventions, per the policy in Spec A §1:
- **Operational, single-store** types (`BillableMetric`, the new `Price` fields, `Customer.external_id`) live in the operational DB and follow the existing operational convention — structs **with** gorm tags, `(OrgId, Id)` PK, `TableName()`, like `Order`/`Subscription`.
- **`MeterEvent`** goes through the `EventStore`, which has **two** backends (Postgres + ClickHouse), so it is a **pure** struct — **no gorm tags**; each adapter maps its own table.

Numeric quantities use `decimal.Decimal` (`github.com/shopspring/decimal`, added per ADR 0004 — it's not yet a dependency; same type Spec A's line items use), stored as Postgres `numeric` / ClickHouse `Decimal(38,9)`; money is `int64` cents.

```go
// internal/core/domain/meter_types.go
type AggregationType string

const (
	AggregationCount       AggregationType = "count"        // how many events (no field)
	AggregationSum         AggregationType = "sum"          // add up a numeric field
	AggregationMax         AggregationType = "max"          // largest value of a numeric field
	AggregationLatest      AggregationType = "latest"       // last reported numeric value
	AggregationWeightedSum AggregationType = "weighted_sum" // numeric value averaged over time
	AggregationUniqueCount AggregationType = "unique_count" // distinct values of a field (usually a string id)
)
```

```go
// internal/core/domain/meter.go — operational, gorm-tagged.
type BillableMetric struct {
	OrgId         string            `gorm:"column:org_id;primaryKey" json:"org_id"`
	Id            string            `gorm:"column:id;primaryKey" json:"id"`
	Code          string            `gorm:"column:code" json:"code"`              // events reference this; unique per org
	Name          string            `gorm:"column:name" json:"name"`
	Aggregation   AggregationType   `gorm:"column:aggregation" json:"aggregation"`
	FieldName     string            `gorm:"column:field_name" json:"field_name"`  // which Metadata key to read; empty for count
	Recurring     bool              `gorm:"column:recurring" json:"recurring"`    // carry running total across periods (weighted_sum)
	RoundingMode  string            `gorm:"column:rounding_mode" json:"rounding_mode"`   // round | ceil | floor | "" (none)
	RoundingScale int               `gorm:"column:rounding_scale" json:"rounding_scale"`
	Metadata      map[string]string `gorm:"column:metadata;serializer:json" json:"metadata"`
	CreatedAt     time.Time         `gorm:"column:created_at" json:"created_at"`
	UpdatedAt     time.Time         `gorm:"column:updated_at" json:"updated_at"`
}

func (BillableMetric) TableName() string { return "billable_metrics" }
```

**`FieldName` per aggregation (important):**
- `count` — no field; counts events.
- `sum` / `max` / `latest` / `weighted_sum` — `FieldName` names a **numeric** Metadata key; its value is pulled out into `MeterEvent.Value` at ingest.
- `unique_count` — `FieldName` names the **identifier** Metadata key, which is usually a **string** (e.g. `seat_id`, `user_id`). It is *not* reduced to a number; the aggregation counts `distinct(metadata[field])`. `Value` is unused for unique-count.

**Rounding.** Adding usage up can land on a fraction (`weighted_sum` → 41.6667; a sum of decimals). `RoundingMode` + `RoundingScale` round the **quantity** before pricing — `ceil`/scale 0 bills 1.2 GB as 2; `round`/scale 2 gives 41.67; empty leaves it raw. (Rounding *money* to cents is separate, in `PriceUsage`.)

```go
// internal/core/domain/meter_event.go — PURE (two backends, separate store): no gorm tags.
type MeterEvent struct {
	OrgId              string            `json:"org_id"`
	Id                 string            `json:"id"`
	CustomerId         string            `json:"customer_id"`          // our customer id …
	ExternalCustomerId string            `json:"external_customer_id"` // … or the merchant's own id — exactly one required
	MetricCode         string            `json:"metric_code"`          // which BillableMetric
	SubscriptionId     string            `json:"subscription_id"`      // optional: bill this subscription; blank = unattributed
	ExternalId         string            `json:"external_id"`          // optional: caller's event id; the dedup key
	Metadata           map[string]string `json:"metadata"`             // event payload; FieldName names the value inside
	Value              decimal.Decimal   `json:"value"`                // numeric field pulled from Metadata at ingest (0 for count/unique_count)
	Timestamp          time.Time         `json:"timestamp"`
	CreatedAt          time.Time         `json:"created_at"`
}
```

- **Customer:** exactly one of `CustomerId` (ours) or `ExternalCustomerId` (the merchant's). We resolve to our customer when we can and store both; an unknown `ExternalCustomerId` is accepted and resolved later by matching `Customer.external_id` (§4, §8). No backfill — every event carries an id and reads filter on both.
- **`ExternalId`** is optional and is the dedup key; a resend with the same id is ignored, omit it and every event counts.
- **`SubscriptionId`** is optional; blank means *unattributed* (§10).

**`Customer.external_id` (new field).** Add `ExternalId string` to `Customer` (operational) — the merchant's own id, **immutable once set** so it's a stable join key. This is what `external_customer_id` events resolve against.

**`Price` extension (metered).** Add a `metered` category and the metering fields (operational, gorm-tagged like the rest of `Price`):

```go
// price_types.go: const PriceCategoryMetered PriceCategory = "metered"
// price.go new fields:
//   BillableMetricId string          // which meter (when category=metered)
//   Tiers            []PriceTier     // rate bands for Graduated / Volume; serializer:json
// PriceTier{ FromValue decimal.Decimal; ToValue decimal.Decimal; PerUnitAmount decimal.Decimal; FlatAmount int64 }
// (PerUnitAmount is decimal cents — sub-cent rates; FlatAmount is int64 cents.)
```

`Scheme` already exists on `Price` (Fixed/Graduated/Volume). Deferred metered fields — `pay_in_advance`, `prorated`, `min_amount` — are **not** added in v1 (see §12).

## 3. Interfaces (ports)

`internal/core/port/usage.go`. Style as `repository.go` (ctx first; `(ctx, orgId, …)`; `(value, error)`).

```go
type MeterRepository interface { // operational DB
	FindByCode(ctx context.Context, orgId, code string) (domain.BillableMetric, error)
	Create(ctx context.Context, m domain.BillableMetric) (domain.BillableMetric, error)
	Find(ctx context.Context, orgId string, p domain.Pagination) ([]domain.BillableMetric, int, error)
}

type EventStore interface { // usage DB; Postgres + ClickHouse implementations
	Ingest(ctx context.Context, e domain.MeterEvent) (IngestResult, error) // ignores a resend whose external_id was already seen
	Count(ctx context.Context, q UsageQuery) (int64, error)                  // count(*) — whole number
	UniqueCount(ctx context.Context, q UsageQuery) (int64, error)            // count(distinct metadata[FieldName]) — whole number
	Sum(ctx context.Context, q UsageQuery) (decimal.Decimal, error)
	Max(ctx context.Context, q UsageQuery) (decimal.Decimal, error)
	Latest(ctx context.Context, q UsageQuery) (decimal.Decimal, error)
	WeightedSum(ctx context.Context, q UsageQuery, initial decimal.Decimal) (decimal.Decimal, error)
}

// `Count`/`UniqueCount` are integers; the rest are `decimal.Decimal` (event values are
// summed, so they must be exact, not float). `AggregateForPeriod` widens the two
// integer results to `decimal.Decimal` to produce the one unified `units`.

type IngestResult struct{ Id string; Duplicate bool }

type UsageQuery struct {
	OrgId, MetricCode, FieldName string
	From, To time.Time // includes From, excludes To
	// A row matches if customer_id = CustomerId OR external_customer_id = ExternalCustomerId
	// (so usage recorded before the customer existed is still found). Service fills both.
	CustomerId, ExternalCustomerId string
	// SubscriptionId set → only events attributed to it; blank → all the customer's events.
	// IncludeUnattributed folds in events with no subscription_id (set when this is the
	// customer's earliest metered subscription for the meter — §10).
	SubscriptionId      string
	IncludeUnattributed bool
}
```

Picking the right `EventStore` method per metric, and the tier math, are pure functions in `internal/core/domain/pricing.go` — no DB:

```go
func PriceUsage(p Price, units decimal.Decimal) (amountCents int64, unitAmountCents decimal.Decimal)
// switches on p.Scheme: Fixed | Graduated | Volume, using p.Tiers; rounds money to whole cents once.
// Shared with Spec A's base-line builder (the only shared pricing code).
```

(No `UsageAggregationRepository` in v1 — running-total state is only for charge-as-you-go, deferred §12.)

## 4. Databases & schema

**(a) Operational DB `getpaidhq` — `DATABASE_URL`, `schemas/app/schema.prisma`** (push: `pnpm prisma:push`). Add `BillableMetric`, the `Price` metered fields + `metered` enum value, and `Customer.external_id`.

```prisma
// add to schemas/app/schema.prisma
model BillableMetric {
  orgId         String   @map("org_id")
  id            String   @default(cuid())
  code          String
  name          String
  aggregation   String                                  // AggregationType
  fieldName     String?  @map("field_name")
  recurring     Boolean  @default(false)
  roundingMode  String?  @map("rounding_mode")
  roundingScale Int      @default(0) @map("rounding_scale")
  metadata      Json?
  createdAt     DateTime @default(now()) @map("created_at")
  updatedAt     DateTime @updatedAt @map("updated_at")
  @@id([orgId, id])
  @@unique([orgId, code])
  @@map("billable_metrics")
}

// model Customer — add:  externalId String? @map("external_id")  + @@unique([orgId, externalId])
//   (immutable once set — enforced in the customer service, not the schema)
// model Price — add:  billableMetricId String? @map("billable_metric_id")
//                     tiers Json?            // [{from_value,to_value,per_unit_amount,flat_amount}]
//   and add `metered` to the PriceCategory enum.
```

**(b) Usage DB `getpaidhq_usage` — `USAGE_DATABASE_URL`, NEW `schemas/usage/schema.prisma`** (new `pnpm prisma:usage:push`, mirroring `prisma:reporting:push`; add `getpaidhq_usage` to `docker/init/01-create-databases.sql`; falls back to `DATABASE_URL` if unset so local dev runs on one Postgres). Only `meter_events` in v1.

```prisma
// schemas/usage/schema.prisma  (Postgres usage adapter)
model MeterEvent {
  orgId              String   @map("org_id")
  id                 String   @default(cuid())
  customerId         String   @map("customer_id")          // resolved to our customer at ingest when possible
  externalCustomerId String?  @map("external_customer_id") // as supplied
  metricCode         String   @map("metric_code")
  subscriptionId     String?  @map("subscription_id")      // attributed subscription (null = unattributed)
  externalId         String?  @map("external_id")          // dedup key (when supplied)
  metadata           Json?
  value              Decimal  @default(0) @db.Decimal(38, 9)
  timestamp          DateTime
  createdAt          DateTime @default(now()) @map("created_at")
  @@id([orgId, id])
  @@index([orgId, customerId, metricCode, timestamp])         // common aggregation path
  @@index([orgId, externalCustomerId, metricCode, timestamp]) // rows recorded before the customer existed
  @@map("meter_events")
}
```

Dedup uses a partial unique index Prisma can't express, added as raw SQL:

```sql
CREATE UNIQUE INDEX meter_events_external_id ON meter_events (org_id, external_id)
  WHERE external_id IS NOT NULL;
```

**(c) ClickHouse (optional backend) — own connection string.** Prisma doesn't target it; DDL ships with the adapter (`internal/adapter/usage/clickhouse/migrations/0001_meter_events.sql`).

```sql
CREATE TABLE meter_events (
  org_id               String,
  customer_id          String,
  external_customer_id String,
  metric_code          String,
  subscription_id      String,
  external_id          String,
  timestamp            DateTime64(3, 'UTC'),
  value                Decimal(38, 9),
  metadata             Map(String, String),
  id                   String,
  ingested_at          DateTime64(3, 'UTC') DEFAULT now64()
)
ENGINE = ReplacingMergeTree(ingested_at)
PARTITION BY toYYYYMM(timestamp)
ORDER BY (org_id, customer_id, metric_code, timestamp, id);  -- ends in id so distinct events never collapse
```

`id` last in `ORDER BY` keeps distinct events from merging; resends sharing an `external_id` are dropped at read time (§9). A `clickhouse` service joins `docker/docker-compose.yml` when we exercise this backend (phase 6).

**Retention — keep events forever (default).** Re-billing, audits, and disputes need the raw events; nothing expires them. The only deletion is explicit (e.g. deleting a meter could cascade — TBD). Month-partitioning makes a future retention window a cheap `DROP PARTITION` / time-bounded delete, not a redesign.

## 5. Code layout

Everything touching the **usage DB** sits under `internal/adapter/usage/`.

```
internal/core/domain/
  meter_types.go          AggregationType
  meter.go                BillableMetric
  meter_event.go          MeterEvent
  pricing.go              PriceUsage + aggregation-method dispatch (pure; shared with Spec A)
internal/core/port/
  usage.go                MeterRepository, EventStore, UsageQuery
internal/core/service/
  usage.go                UsageService (narrow — no engine)
internal/adapter/postgres/
  meter_repo.go           MeterRepository — operational DB
internal/adapter/usage/                       <-- the usage DB
  db.go                   opens USAGE_DATABASE_URL (falls back to DATABASE_URL)
  postgres/event_store.go     EventStore (Postgres)
  clickhouse/event_store.go   EventStore (ClickHouse) + migrations/0001_meter_events.sql
  compare/event_store.go      wraps both; serves Postgres, checks ClickHouse in background
internal/adapter/http/
  usage_handler.go        POST /api/usage/events (+ /batch), GET .../customers/:id/usage
internal/config/
  app.go                  open usage DB; build chosen EventStore; wire UsageService
internal/config/server.go register routes
```

## 6. Event ingestion

**Endpoints** (in `server.go`, behind `AuthnWrapperMiddleware`, org-scoped — `orgId` from auth, never the body):
- `POST /api/usage/events` — one event.
- `POST /api/usage/events/batch` — `{ "events": [ … ] }`, max 100.

**Request body:**
```json
{
  "customer_id": "cust_…",              // or "external_customer_id": "your-user-42" — exactly one
  "metric_code": "api_calls",
  "subscription_id": "sub_…",           // optional; omit to leave usage unattributed
  "external_id": "evt-123",             // optional; dedup key
  "timestamp": "2026-06-04T10:00:00Z",  // optional; defaults to now
  "metadata": { "seat_id": "u_42", "calls": "150" }
}
```

**`UsageService.RecordEvent` flow:**
1. **Resolve the metric** — `MeterRepository.FindByCode(orgId, metric_code)`; `400` if unknown.
2. **Identify the customer** — exactly one of `customer_id` / `external_customer_id` (`400` if neither). A `customer_id` must exist (`404` if not). An unknown `external_customer_id` is **accepted** (stored as-is; resolved later — §8, by design §below). When it resolves now, store the `customer_id` too.
3. **Attribute (optional)** — if `subscription_id` is given, check it belongs to the customer and carries a metered price for this metric (`422` if not); else store null (unattributed).
4. **Extract the value** — for `sum`/`max`/`latest`/`weighted_sum`, read `metadata[FieldName]`, parse to `decimal` → `Value` (`400` if missing/non-numeric). For `count` and `unique_count`, `Value` stays 0 (unique-count distincts the raw metadata field at query time).
5. **Store** — build `MeterEvent` (`Id = lib.GenerateId("mev")`), `EventStore.Ingest`. A resend with a seen `external_id` is ignored; `IngestResult.Duplicate = true` (not an error).
6. **Publish** — `usage.recorded` on NATS.

*(No charge-as-you-go step — v1 is arrears-only, §12.)*

**Dedup, same result both backends:** Postgres uses the partial unique index on `(org_id, external_id)` + `ON CONFLICT DO NOTHING`; ClickHouse drops `external_id` duplicates at read time (§9). The ClickHouse adapter uses `async_insert` so single-event calls batch server-side.

**Orphan events are intentional.** A client may record usage against an `external_customer_id` with no gphq customer yet. We store it; it counts toward billing only if and when a `Customer` with that `external_id` is created (matched by either id, §8). If one never is, the events sit unused. This is deliberate — ingestion never blocks on provisioning, and the order of "send usage" vs "create customer" doesn't matter.

**Responses:** `202 { "id": "mev_…", "status": "recorded" | "duplicate" }`; batch returns `{ "inserted": n, "duplicates": m }` plus a per-input entry with its index.

## 7. Adding usage up (Postgres backend)

Plain SQL over `meter_events`:

```sql
-- Sum: SUM(value)  |  Count: count(*)  |  Max: max(value)  |  Latest: value ORDER BY timestamp DESC LIMIT 1
-- UniqueCount: count(distinct metadata->>:field)   ← the raw (often string) field, NOT value
SELECT COALESCE(SUM(value),0) FROM meter_events
 WHERE org_id = :org
   AND (customer_id = :customer_id OR external_customer_id = :external_customer_id)   -- match either id
   AND metric_code = :metric
   AND timestamp >= :from AND timestamp < :to
   AND (:sub = '' OR subscription_id = :sub OR (:incl_unattributed AND subscription_id IS NULL));  -- §10
```

A column filter, not a join — so it runs the same in Postgres and ClickHouse (no cross-DB join to the customer table; resolution happens in the service). `unique_count` reads the raw `metadata` field, never the numeric `value`. `weighted_sum` (value averaged over time) needs a window query — built when we use that metric type (§11). The meter's rounding is applied to the result; money is rounded to cents in `PriceUsage`.

## 8. Billing integration (extends Spec A)

Usage is billed **in arrears on the per-cycle invoice**. Spec A's `InvoiceService.BuildForBillingPeriod` is extended: after the base line, for each metered `Price` on the subscription it appends usage line(s).

`UsageService` (narrow — no engine):
- `RecordEvent` — §6.
- `AggregateForPeriod(ctx, sub, price, from, to) (units decimal.Decimal, err)` — resolve the subscription's customer (fills `CustomerId` + `ExternalCustomerId`), decide whether this is the customer's earliest metered subscription for the meter (sets `SubscriptionId` + `IncludeUnattributed`, §10), call the metric's `EventStore` method, apply the meter's rounding.
- `CurrentUsage(ctx, orgId, customerId)` — aggregate period-start→now and price in memory, persisting nothing. This is the **invoice preview / pro forma** (`GET .../customers/:id/usage`).

In the build, for a metered price: `units = AggregateForPeriod(...)`, `(amount, unitAmount) = domain.PriceUsage(price, units)`, then append an `InvoiceLineItem{ Kind: usage, Quantity: units, UnitAmount: unitAmount, Total: amount }`. The invoice total (base + usage) is what the `Payment` settles — unchanged from Spec A.

**Trials (ADR 0003).** A trial waives the **base** line only — **usage is still billed.** The **trial is the first period**: the invoice that runs at trial end covers the trial window `[subscriptionStart, trialEnd)` with **no base line** and **usage lines** for trial-window usage; subsequent periods invoice base + usage normally.

The usage window must match how the base is billed, to avoid a first-invoice mismatch:
- **Base in arrears** (invoice = the just-completed period): base and usage share the same window every cycle; the trial is simply the first completed period with the base waived. No asymmetry — preferred.
- **Base in advance** (base for the upcoming period): the trial-end invoice carries trial-window usage (arrears) *plus* the first paid period's base (advance) — an intentional, documented asymmetry the build must handle explicitly.

`BuildForBillingPeriod` therefore takes the usage window from the just-closed period and omits the base line while `sub.Status == trial`. (A `Price` may opt into a fully-free trial later; bounded free allowances are the deferred credits feature.)

## 9. The two backends and keeping them equal

Chosen by `USAGE_EVENT_STORE`. Detail and per-method SQL in `docs/internal/clickhouse-primer.md` §7. Same approach as the codebase's Hatchet/Temporal engine parity (`docs/internal/engine-parity-and-subscription-lifecycle.md`): same result, different implementation.

- **Postgres** drops duplicates on **write**; reads are immediately exact. Row storage.
- **ClickHouse** drops `external_id` duplicates on **read** (keep latest per id). Column storage, far faster at summing one field over millions of rows.

Both take the same `UsageQuery` and return the same value; time ranges are half-open `[from, to)` via one shared helper.

```
USAGE_EVENT_STORE=postgres    # default
USAGE_EVENT_STORE=clickhouse
USAGE_EVENT_STORE=compare     # write both; serve Postgres; check ClickHouse in background; log diffs + timings
```

**Equality test:** one table-driven test feeds the same events (resent duplicates, out-of-window, boundary timestamps, string-id unique-count) into both implementations, runs every method over several queries, and asserts equality within a tiny tolerance. Same test, two backends.

## 10. Attributing usage to a subscription

A customer may hold several subscriptions metered on the same meter, each billed separately. An event attributes itself with `SubscriptionId`; blank = unattributed. When billing subscription `S` (customer `C`, meter `M`, period `P`), `AggregateForPeriod` sums:

1. events attributed to `S`, plus
2. **only if `S` is `C`'s earliest active subscription with a metered price on `M`**, the unattributed events for `(C, M, P)`.

So attributed usage bills its named subscription, and anything sent without a subscription falls to the **earliest** one — exactly one catch-all per `(customer, meter)`, so unattributed usage is never double-billed. "Earliest" = earliest-started, ties by creation time.

`AggregateForPeriod` finds the catch-all via a new operational query —
`SubscriptionRepository.FindActiveMeteredForMeter(ctx, orgId, customerId, billableMetricId) ([]domain.Subscription, error)` — returning the customer's active subscriptions whose metered `Price` targets that meter, **ordered by `StartDate` then `CreatedAt`**. The current subscription is the catch-all iff it's the first in that list; the service sets `UsageQuery.IncludeUnattributed` accordingly.

## 11. Phasing

1. **Schema + read path.** `getpaidhq_usage` DB + `schemas/usage/`; `billable_metrics`, `Customer.external_id`, `Price` metered fields in `schemas/app/`; domain types; `MeterRepository`; Postgres `EventStore` (count/sum/max/latest/unique_count); `domain.PriceUsage` (Fixed/Graduated/Volume) with thorough unit tests.
2. **Ingestion + billing.** §6 endpoints → `RecordEvent`; `metered` price category; extend `InvoiceService.BuildForBillingPeriod` to append usage lines. End-to-end metered subscription billed in arrears on the invoice.
3. **Invoice preview.** `GET .../customers/:id/usage` via `CurrentUsage`.
4. **weighted_sum** (window query) + recurring metrics.
5. **ClickHouse backend.** `clickhouse` `EventStore` + migration + docker service; verify with the equality test and `compare` mode.

## 12. Deferred (not in v1)

- **Charge-as-you-go (pay-in-advance)** and with it `UsageAggregation` + the highest-billed watermark and `pay_in_advance` price field — v1 bills usage only in arrears on the cycle invoice.
- **`unique_count` add/remove** ("currently active seats") — v1 is distinct-seen-in-period.
- **Proration** of usage for partial periods; **minimum-commitment** true-up (`min_amount`).
- **Credits / prepaid wallet** (incl. bounded free-trial usage allowances).
- **Additional charge models** beyond Fixed/Graduated/Volume (percentage-of-amount, package, dynamic-per-event); a formula language for event values; per-dimension (grouped/filtered) pricing.
