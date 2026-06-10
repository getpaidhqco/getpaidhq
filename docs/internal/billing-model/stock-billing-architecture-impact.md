# Stock billing — architectural impact

> Design spec: how the billing model described in [`seat-billing/`](./seat-billing/)
> and [`carry-over-meters.md`](./carry-over-meters.md) lands in this codebase.
> This doc is about *system impact*; the policy semantics live in the seat-billing docs
> and are not restated here.

---

## 1. What we are building (the model)

Seat billing is the motivating use case, but the capability is generic:

> **Bill a *stock* — a standing level that persists until changed — by collapsing
> that level over the billing period into a quantity.**

Seats held, VMs provisioned, feature flags enabled, phone lines active: all the same
shape. The model has three parts:

**1. A carry-over meter** (`BillableMetric.CarryOver = true`) measures a standing
level, not this period's events. Reading it replays history from before the period
start — a seat added last month and never removed emits no event this period, yet
must bill this period. Silence ≠ zero.

**2. The standing level is reconstructed from the event history.** Merchants
describe the level in one of two ways, both over the existing `POST /api/usage/events`:

- **add/remove events** — one event per change, carrying `metadata.operation:
  add | remove` and an identity under the meter's `FieldName` (e.g. `seat_id`).
  Replay pairs them per identity into intervals `[active_from, active_to)`; the
  level at any instant is the number of open intervals.
- **level reports** — the current total as a number under `FieldName`
  (`{"count": "5"}`). The reported level persists until the next report.

If the meter's history contains any operation events, the level is the
open-interval count; otherwise it is the last-reported value. Nothing is
configured on the meter.

**3. The aggregation collapses the standing level into a quantity** — the existing
enum, each value keeping its ordinary meaning, applied to the level:

| `aggregation` | Quantity | Needs identities? | Use case |
| --- | --- | --- | --- |
| `latest` | level at period end | no | [A — end of period](./seat-billing/use-case-a-full-period.md) |
| `max` | peak level in the period (incl. the level standing at period start) | no | A — peak |
| `unique_count` | distinct identities active in the period, incl. those standing from before | yes | A — distinct / MAU-style |
| `weighted_sum` | time-weighted level: Σ per-identity interval fractions, shaped by the price switches | for the switches | [B](./seat-billing/use-case-b-time-weighted.md) / [C](./seat-billing/use-case-c-hybrid.md) |
| `count`, `sum` | invalid on carry-over meters | — | — |

"Needs identities" rows require add/remove events to be meaningful: `unique_count`
over level reports has no identities to count, and the proration switches have no
intervals to clip (a level-report `weighted_sum` is the symmetric time-average).

**The price owns the deal.** Quantity → money is unchanged (`PriceUsage` →
`UsageLineFromPrice`; fixed/graduated/volume schemes and tiers already accept
fractional quantities). Two new price fields select between B and C on a
`weighted_sum` meter:

| Field | Meaning |
| --- | --- |
| `prorate_on_increase` | a seat added mid-period accrues from its add date |
| `credit_on_decrease` | a seat removed mid-period stops accruing at its remove date |

B = both true. C = prorate true, credit false. Both false gives every seat a full
period, which equals the `unique_count` reading — a consistency check. A seat's
fraction is its active time divided by the period length, computed from the event
timestamps as sent — nothing is truncated. Merchants control granularity through
the timestamps they send; the meter's `RoundingMode`/`RoundingScale` rounds the
final quantity.

The split is load-bearing: the **meter** owns how the data is read (carry-over,
`FieldName`, aggregation); the **price** owns how a quantity becomes money (scheme,
tiers, proration switches). Because the switches are on the price, B and C are the
**same meter**: two prices on one `weighted_sum` seat meter can differ only in
their switches — one plan prorates and credits (B), another prorates and commits
(C) — over the identical events. Whole-seat A is a different aggregation and
therefore a different meter.

Charge timing (advance vs arrears) is a separate concern: metered lines bill in
arrears at period close, and nothing here changes that.

### The use cases, concretely

Running example: $10/seat/month, billing June. alice, bob and carol have seats
since May 20; dave gets one June 16; bob loses his June 21.

As add/remove events (one event per change; the May events matter in June because
the meter is carry-over):

```jsonc
{ "metric_code": "seats", "timestamp": "2026-05-20T00:00:00Z", "metadata": { "operation": "add",    "seat_id": "alice" } }
{ "metric_code": "seats", "timestamp": "2026-05-20T00:00:00Z", "metadata": { "operation": "add",    "seat_id": "bob" } }
{ "metric_code": "seats", "timestamp": "2026-05-20T00:00:00Z", "metadata": { "operation": "add",    "seat_id": "carol" } }
{ "metric_code": "seats", "timestamp": "2026-06-16T00:00:00Z", "metadata": { "operation": "add",    "seat_id": "dave" } }
{ "metric_code": "seats", "timestamp": "2026-06-21T00:00:00Z", "metadata": { "operation": "remove", "seat_id": "bob" } }
```

The same history as level reports (one event per total change; no identities):

```jsonc
{ "metric_code": "seats", "timestamp": "2026-05-20T00:00:00Z", "metadata": { "count": "3" } }
{ "metric_code": "seats", "timestamp": "2026-06-16T00:00:00Z", "metadata": { "count": "4" } }
{ "metric_code": "seats", "timestamp": "2026-06-21T00:00:00Z", "metadata": { "count": "3" } }
```

Event fields (the top-level fields are the existing API; events also carry
`customer_id`/`external_customer_id` and optionally `subscription_id` and
`external_id`, omitted above for brevity):

| Field | Reserved | Holds |
| --- | --- | --- |
| `metric_code` | top-level, required | which meter |
| `timestamp` | top-level | when the change happened — replay sorts on it, and for B/C it is the proration input (the fraction is computed from timestamps as sent). Defaults to ingest time if omitted. |
| `metadata.operation` | reserved key, values `add` \| `remove` | makes the event an add/remove event. The only new reserved name in this design. |
| `metadata.<field_name>` | key named by the meter's `field_name` | the seat identity (add/remove events) or the numeric total (level reports) |

| Use case | The deal | Meter | Price switches | Works with | June bill |
| --- | --- | --- | --- | --- | --- |
| A — end of period | pay for the seats standing when the period closes; a seat added and removed mid-period is never billed | `carry_over: true, aggregation: latest` | none | either | 3 × $10 = $30 |
| A — peak | pay for the highest concurrent seat count, even if it held for a day | `carry_over: true, aggregation: max` | none | either | 4 × $10 = $40 |
| A — distinct | anyone who held a seat at any point pays the full period (MAU-style) | `carry_over: true, aggregation: unique_count` | none | add/remove only | 4 × $10 = $40 |
| B — time-weighted | pay per seat for exactly the time held; leavers stop accruing | `carry_over: true, aggregation: weighted_sum` | `prorate_on_increase: true`, `credit_on_decrease: true` | add/remove (level reports give the symmetric average too) | 3.17 × $10 = $31.67 |
| C — hybrid | joiners prorated from their add date; leavers keep accruing to period end | `carry_over: true, aggregation: weighted_sum` | `prorate_on_increase: true`, `credit_on_decrease: false` | add/remove only | 3.50 × $10 = $35 |

All five share `field_name: "seat_id"` (or `"count"` for level reports) and an
ordinary fixed/graduated/volume price. Worked math per use case:
[`seat-billing/README.md`](./seat-billing/README.md).

---

## 2. Hexagonal placement — impact per layer

| Layer | Change | Size |
| --- | --- | --- |
| `core/domain` | `Price` gains `ProrateOnIncrease bool` + `CreditOnDecrease bool`. `BillableMetric` unchanged. One new file with the pure math: `UsageInterval`, `ReconstructIntervals`, the level collapses of §3, and the `operation` constants. | The heart of the change |
| `core/port` | `EventStore` gains `ListHistory(ctx, q)` (§3). `UsageQuery` gains the two switch fields (copied from the price at query-build time — §5). `CreatePriceInput` gains the two switches. | Small |
| `core/service` | `UsageService.AggregateForPeriod` gains the `metric.CarryOver` branch (§5); `usageQueryFor` copies the price switches onto the query; `buildEvent` gains the carry-over event validation (§4). **No new service.** | Moderate |
| `adapter/postgres` (+ clickhouse) | Implement `ListHistory`: the existing scoped event query, ordered by timestamp. `price_row` gains the 2 columns. | Small per backend |
| `adapter/http` | Price request/response DTOs gain `prorate_on_increase` / `credit_on_decrease`. Meter DTOs unchanged. | Mechanical |
| `adapter/hatchet`, `adapter/temporal` | **Zero changes.** Both engines reach billing through `InvoiceService.BuildForBillingPeriod` → `MeteredUsageForSubscription`, which is where the new logic plugs in — parity holds by construction. | None |
| `schemas/app/schema.prisma` | `Price` gains `prorateOnIncrease` / `creditOnDecrease` (`Boolean @default(false)`). No new tables, no new indexes. | Small |

The change is concentrated in one place: how `AggregateForPeriod` computes
a number when the meter is carry-over.

---

## 3. Design: flow meters stay in SQL; carry-over meters compute in core

1. **Flow meters are untouched** — each aggregation stays a SQL query, as today.
2. **Carry-over meters compute in core.** Reconstruction, clipping, and
   peak-overlap are too intricate and too high-stakes to write twice in two SQL
   dialects. The store's only new job is fetching the events; pure Go in
   `core/domain` computes the quantity — one implementation, shared by both
   workflow engines, unit-tested against the worked examples in the use-case
   docs.

The carry-over read has three steps:

**1. Fetch the events.** One new `EventStore` method:

```go
// ListHistory returns the events matching q, ordered by timestamp.
ListHistory(ctx context.Context, q UsageQuery) ([]domain.MeterEvent, error)
```

Billing queries from zero to the period end: events before the period determine
who is standing when it starts. The period itself is applied in step 3 — the
quantity covers exactly `[periodStart, periodEnd)`.

**2. Rebuild the level.** Operation events become per-identity intervals. If
there are none, the reported values are the level (§1).

```go
type UsageInterval struct {
    Identity string    // value of the meter's FieldName key
    From     time.Time
    To       time.Time // zero = still open at read time
}

// ReconstructIntervals replays ordered add/remove events into per-identity
// intervals: an "add" opens an interval for that identity, a "remove" closes it.
func ReconstructIntervals(events []MeterEvent, fieldName string) []UsageInterval
```

**3. Apply the aggregation over the period** — one pure function per row of the
§1 table:

```go
// From intervals (the level is the open-interval count):
func CountStandingAtEnd(intervals []UsageInterval, to time.Time) int64
func CountPeakConcurrent(intervals []UsageInterval, from, to time.Time) int64
func CountDistinctActive(intervals []UsageInterval, from, to time.Time) int64

// Time-weighting (use cases B and C; switches from the price):
func WeightIntervals(intervals []UsageInterval, from, to time.Time,
    prorateOnIncrease, creditOnDecrease bool) decimal.Decimal
```

With level reports the same aggregations read the reported values directly:
`latest` is the last value, `max` is the highest value in the period (including
the one in force at period start), `weighted_sum` is the average level over the
period, and `unique_count` is zero — there are no identities.

Pulling events into memory is safe because stocks are low-cardinality by nature (few changes a month).

---

## 4. Domain model changes (types only)

```go
// BillableMetric — unchanged (CarryOver and FieldName already exist)

// Price — two new fields, meaningful only for prices on weighted_sum carry-over meters
ProrateOnIncrease bool // clip an interval's start to its add date
CreditOnDecrease  bool // clip an interval's end to its remove date
```

`MeterEvent` is **unchanged**: identity and operation ride the existing `Metadata`
(`FieldName` already names the key; `operation` is a reserved metadata key with
values `add` | `remove`). No schema change to the events table.

Validation, enforced at meter/price write time and at ingest:

- A carry-over meter's aggregation must be `latest`, `max`, `unique_count`, or
  `weighted_sum`; `count` and `sum` are invalid.
- `Filters` / `GroupBy` are invalid on carry-over meters (no defined replay
  semantics per rate slice).
- Ingest: an event for a carry-over meter is either an operation event —
  `metadata.operation ∈ {add, remove}` plus a non-empty identity under
  `FieldName` — or a level report — a numeric value under `FieldName`. Anything
  else is rejected. Replay tolerance: a duplicate `add` for an open identity is
  idempotent; a `remove` without an open interval is ignored; out-of-order
  arrival is handled by sorting on timestamp before replay.
- The proration switches are meaningful only on prices attached to `weighted_sum`
  carry-over meters; elsewhere they are inert.

---

## 5. Read-path flow (what changes inside `AggregateForPeriod`)

```
AggregateForPeriod(metric, q)
 ├─ metric.CarryOver == false  → existing per-aggregation pushdown (unchanged)
 └─ metric.CarryOver == true
     ├─ events = ListHistory(q)                                  // one port call
     ├─ history has operation events:
     │   ├─ ivals = ReconstructIntervals(events, FieldName)     // pure
     │   ├─ latest       → CountStandingAtEnd(ivals, To)
     │   ├─ max          → CountPeakConcurrent(ivals, From, To)
     │   ├─ unique_count → CountDistinctActive(ivals, From, To)
     │   └─ weighted_sum → WeightIntervals(ivals, From, To, switches)
     └─ history is level reports:
         ├─ latest       → last reported value ≤ To
         ├─ max          → highest value in the period, incl. the one at From
         ├─ unique_count → 0 (no identities)
         └─ weighted_sum → average reported level over the period
```

The proration switches belong to the **price**, but `AggregateForPeriod` only
receives the metric — so `usageQueryFor`, which has the price, copies them onto
the `UsageQuery` (which already carries the price's `FilterField`/`FilterValue`).
No signature change.

`credit_on_decrease` reduces the quantity itself, so the invoice gets **one
positive usage line per period** — never a negative credit line. Rounding
(`applyRounding`) applies last, as today.

---

## 6. What explicitly does NOT change

- **Invoice flow**: `BuildForBillingPeriod` → `MeteredUsageForSubscription` →
  `UsageLineFromPrice` — quantities just may now be fractional.
- **Pricing math**: `PriceUsage` (fixed/graduated/volume + tiers) already handles
  decimals.
- **Workflow engines**: no Hatchet/Temporal code touched; parity by construction.
- **Ingestion pipeline**: same `POST /api/usage/events`, same `EventIngestor` modes
  (sync/jetstream), same dedup. Only *validation* grows (§4).
- **Flow meters**: every existing meter (`CarryOver=false`) takes the exact code
  path it does today — including the deferred flow-meter `weighted_sum`
  (value-average), which remains unimplemented in the stores.

---

## 7. Open items (deferred, not blocking)

1. **History fetch performance** — only if it gets slow at scale: promote
   `operation`/identity to indexed columns, or query from a saved checkpoint
   instead of from zero.
2. **Anchor change mid-period × time-weighted stock** (two prorations stacking) —
   needs a worked example before being supported; currently undefined.

---

## 8. Build sequence

1. Domain types + pure math (`ReconstructIntervals`, the three counts,
   `WeightIntervals`, the level-report reads) with the June timeline from the
   use-case docs as the unit-test fixture — all policies, plus the consistency
   check (both switches off = the `unique_count` reading).
2. `ListHistory` in both event stores (postgres and clickhouse).
3. The carry-over branch in `AggregateForPeriod` + ingest validation.
4. Price switches: API + schema columns + meter validation (§4 rules).
5. e2e: seat billing across a period boundary, add/remove events and level reports
   (carry-over replay across months is the behaviour nothing else exercises).