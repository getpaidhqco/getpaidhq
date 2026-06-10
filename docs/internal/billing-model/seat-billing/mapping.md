# Seat billing — how we model and bill it

> **[← Index](./README.md)** · Use cases: [A — Full-period](./use-case-a-full-period.md) · [B — Time-weighted](./use-case-b-time-weighted.md) · [C — Hybrid](./use-case-c-hybrid.md)

This is *our* answer to the seat-billing problem. The [use-case docs](./README.md#the-three-answers)
define the policies in product-neutral terms; this doc maps them onto GetPaidHQ's
meters, events, prices, proration, and billing cycle — then states honestly what
exists today versus what we still have to build.

It builds on [`../carry-over-meters.md`](../carry-over-meters.md): seats are the
canonical **stock** metric (a quantity that persists until changed), so a seat
meter is a **carry-over meter**, and the aggregation must replay history to know
"who is active right now" rather than counting only this period's events.

---

## 1. One model, three policies

All three use cases reduce to a single idea:

> **A seat is an interval `[active_from, active_to]` inside the billing period.**
> A meter holds a set of these intervals; billing turns that set into a quantity.

Two **orthogonal axes** decide how the set of intervals becomes a number.

### Axis 1: whole-seat counting

When we are **not** time-weighting, we bill a **whole-seat count** (an integer).
Which moment defines the count is the meter's **aggregation**, each keeping its
ordinary meaning, applied to the standing seat level:

| `aggregation` | Quantity | June timeline |
| --- | --- | --- |
| `latest` | seats standing at period end | 3 |
| `max` | peak concurrent seats at any instant | 4 |
| `unique_count` | distinct seats active at any point in the period | 4 |

### Axis 2: time-weighting

When **on**, the meter bills a **fractional quantity** via `weighted_sum`:
quantity = `Σ (effective_interval ÷ period_length)`. Two independent
**proration switches** reshape each seat's *effective* interval before summing:

| Switch | `true` | `false` |
| --- | --- | --- |
| `prorate_on_increase` | a mid-period joiner accrues from its **join** date | a joiner is treated as present from **period start** (billed full) |
| `credit_on_decrease` | a mid-period leaver accrues only to its **leave** date | a leaver is treated as staying to **period end** (committed, no credit) |

### The switch table generates every use case

| Use case | Mode | `prorate_on_increase` | `credit_on_decrease` | June qty |
| --- | --- | --- | --- | --- |
| **[A](#use-case-a)** full-period | whole-seat (`latest` / `max` / `unique_count`) | — | — | 3 or 4 |
| **[B](#use-case-b)** time-weighted | time-weighted | `true` | `true` | 3.17 |
| **[C](#use-case-c)** hybrid | time-weighted | `true` | `false` | 3.50 |
| _(degenerate)_ | time-weighted | `false` | `false` | 4 |

The corner case is the consistency check: a time-weighted meter with **both
switches off** gives every overlapping seat an effective interval of the full
period → `1.0` each → a total equal to the `distinct_active` whole-seat count.
The two axes meet, so the model isn't ad hoc.

---

## 2. Mapping onto our model

### 2.1 Meter (`domain.BillableMetric`)

A seat meter is a **carry-over meter** with seat-specific settings:

| Field | Value for seats |
| --- | --- |
| `CarryOver` | `true` — stock semantics; reads replay history across periods |
| `Aggregation` | `latest` \| `max` \| `unique_count` \| `weighted_sum`, read over the standing level (see §2.4) |
| `FieldName` | metadata key holding the **seat identity** in add/remove events (e.g. `"seat_id"`), or the numeric **count** key in level reports |

No other meter settings exist — behaviour is fully determined by these plus the
price (§2.3).

`CarryOver` is the switch described in [`carry-over-meters.md`](../carry-over-meters.md):
non-carry-over meters apply the period's start boundary (`timestamp >= from`);
a carry-over meter **drops the start boundary** and replays the full add/remove
history to reconstruct the currently-active set.

### 2.2 Events — two ways to describe the level

Merchants describe the standing level in one of two ways. Both ride the existing
`POST /api/usage/events` endpoint and the `domain.MeterEvent` record
(`{metric_code, metadata, value, timestamp, …}`); the difference is what they
carry. Nothing is configured on the meter — if the history contains operation
events, the level is the open-interval count; otherwise it is the last-reported
value.

**Add/remove events** — for merchants who can emit one event per seat change.
Each event carries an **operation** and the **seat identity**:

```jsonc
// seat added
{ "metric_code": "seats",
  "timestamp": "2026-06-16T09:00:00Z",
  "metadata": { "operation": "add", "seat_id": "user_123" } }

// seat removed
{ "metric_code": "seats",
  "timestamp": "2026-06-21T17:00:00Z",
  "metadata": { "operation": "remove", "seat_id": "user_456" } }
```

Reconstruction replays these per `seat_id`: an `add` opens an interval, a `remove`
closes it. Because the meter is carry-over, replay reaches back before the period
start, so a seat added in May and never removed is still active in June even though
it emitted no June event. Add/remove events support every reading: all three
whole-seat counts, and per-seat time-weighting with asymmetric switches (B/C).

**Level reports** — for merchants who can only report the current total:

```jsonc
{ "metric_code": "seats", "timestamp": "2026-06-30T00:00:00Z",
  "metadata": { "count": "3" } }
```

The reported level persists until the next report. Level reports carry no per-seat
identity, so they support the level readings (`latest`, `max`, and the symmetric
time-average via `weighted_sum`) but not `unique_count` or the asymmetric C — there
are no identities to count or to clip.

> Operation and seat identity live in `metadata` (no schema change: `metadata`
> is already `Json`, and `FieldName` already names a metadata key). Promoting
> them to first-class columns is a possible later optimisation — see §6.

### 2.3 Price (`domain.Price`) and proration

The seat **rate** is an ordinary metered price keyed to the seat meter
(`BillableMetricId` set, `IsMetered() == true`). The seat **quantity** (whole or
fractional) flows through the existing `PriceUsage` path:

- **Fixed** scheme → `quantity × UnitPrice` (flat per-seat price).
- **Graduated** / **Volume** schemes → seat **tiers** already supported by
  `PriceTier` (e.g. first 10 seats at $10, next 40 at $8).

Two **new proration fields** on the price carry Axis 2:

| Field | Meaning |
| --- | --- |
| `prorate_on_increase bool` | clip a seat's interval start to its join date |
| `credit_on_decrease bool` | clip a seat's interval end to its leave date |

A seat's fraction is its active time divided by the period length, computed from
the event timestamps as sent — nothing is truncated. Merchants control
granularity through the timestamps they send (midnight timestamps give whole-day
fractions, as in the examples below); the meter's rounding settings round the
final quantity.

### 2.4 Aggregation — what the engine must compute

For a carry-over meter, `usage.AggregateForPeriod` fetches the full event history
(no period-start bound) and folds it: operation events are **reconstructed into
intervals** (replay keyed by `FieldName`) and collapsed; with no operation events,
the same readings come off the reported step function:

| `aggregation` | add/remove events | level reports |
| --- | --- | --- |
| `latest` | count of intervals open at `periodEnd` | last reported value up to `periodEnd` |
| `max` | max overlap depth across the period | peak reported value, incl. the value standing at period start |
| `unique_count` | count of distinct seats with an interval overlapping `[from, to)` | zero — no identities |
| `weighted_sum` | `Σ clip(interval, switches) ÷ periodLength` | time-average of the reported level (symmetric only) |

`weighted_sum` requires `carry_over: true` — a time-averaged quantity is a
standing level by definition. On a flow meter it would reset to zero each period
and underbill every quiet period, so meter creation rejects it.

### 2.5 Billing cycle

No change to the *flow*: `InvoiceService.BuildForBillingPeriod` already walks the
subscription's metered lines and calls `MeteredUsageForSubscription(sub, price,
periodStart, periodEnd)`. Seat billing slots in there — the returned quantity is
just whole (integer line) or fractional (decimal line). The invoice line is built
by the existing `UsageLineFromPrice` (`quantity × rate`). Credits from
`credit_on_decrease` net into the period-close quantity — a seat's credited time
simply reduces its billable fraction before the line is built, so there is one
positive usage line per period.

---

## 3. Configuring each use case

### Use case A

**Full-period.**

```yaml
meter:   { carry_over: true, aggregation: latest }  # or max / unique_count
price:   { scheme: fixed, unit_price: 1000 }        # $10/seat
# no proration — whole seats only
```

Quantity is an integer count by the chosen [aggregation](#axis-1-whole-seat-counting).
June → 3 (`latest`) or 4 (`max` / `unique_count`).
See [Use case A](./use-case-a-full-period.md).

### Use case B

**Time-weighted.**

```yaml
meter:   { carry_over: true, aggregation: weighted_sum }
price:   { scheme: fixed, unit_price: 1000,
           prorate_on_increase: true, credit_on_decrease: true }
```

Each seat's interval is clipped to `[join, leave]`. June → **3.17** seats.
See [Use case B](./use-case-b-time-weighted.md).

### Use case C

**Hybrid.**

```yaml
meter:   { carry_over: true, aggregation: weighted_sum }
price:   { scheme: fixed, unit_price: 1000,
           prorate_on_increase: true, credit_on_decrease: false }
```

Joins start at the join date; leaves extend to period end (committed). June →
**3.50** seats. See [Use case C](./use-case-c-hybrid.md).

---

## 4. End-to-end worked example (hybrid)

Customer `cus_1`, $10/seat/month, June (30 days). Add/remove events (seat meter
`seats`, identity key `seat_id`):

```
May 20  add    alice     ─┐ (before the period — found via carry-over replay)
May 20  add    bob        │
May 20  add    carol      │
Jun 16  add    dave       │
Jun 21  remove bob       ─┘
```

Reconstructed intervals clipped to June with **C** switches
(`prorate_on_increase=true`, `credit_on_decrease=false`):

| seat | raw interval | effective (C) | fraction |
| --- | --- | --- | --- |
| alice | May 20 → (open) | Jun 1 → Jun 30 | 1.00 |
| carol | May 20 → (open) | Jun 1 → Jun 30 | 1.00 |
| bob | May 20 → Jun 21 | Jun 1 → **Jun 30** (committed) | 1.00 |
| dave | **Jun 16** → (open) | Jun 16 → Jun 30 (prorated join) | 0.50 |

`quantity = 3.50` → `UsageLineFromPrice(price, 3.50)` → `3.50 × $10 = $35.00`.
Swap to **B** (`credit_on_decrease=true`) and bob's interval clips to Jun 21
→ `0.67` → `quantity = 3.17` → `$31.67`.

---

## 5. Where it lives in code

| Capability | Code |
| --- | --- |
| Carry-over read path | `UsageService.AggregateForPeriod` → `aggregateCarryOver` (`internal/core/service/usage.go`) |
| History fetch | `EventStore.ListHistory` — both stores (`internal/adapter/postgres/event_store.go`, `internal/adapter/clickhouse/event_store.go`) |
| Interval reconstruction + level math | `internal/core/domain/usage_interval.go` (`ReconstructIntervals`, `CountStandingAtEnd`, `CountPeakConcurrent`, `CountDistinctActive`, `WeightIntervals`, level-report reads) |
| Operation convention + ingest validation | `domain.UsageOperationKey` consts; `UsageService.buildEvent` |
| Meter validation | `validateCarryOver` (`internal/core/service/meter.go`) |
| Proration switches | `Price.ProrateOnIncrease` / `Price.CreditOnDecrease`, carried onto the `UsageQuery` in `usageQueryFor` |
| Quantity → money | unchanged: `PriceUsage` → `UsageLineFromPrice`, whole or fractional |
| Tests | June-timeline unit tests (`usage_interval_test.go`) and e2e across a period boundary (`internal/adapter/postgres/seat_billing_e2e_test.go`) |

**Engine parity note.** All aggregation/proration logic lives in `core/`
(shared by both the Hatchet and Temporal adapters) so the two engines produce
identical bills — per the parity rule in the root `CLAUDE.md`.

---

## 6. Open items

- **First-class event columns:** promote `operation` / `seat_id` out of
  `metadata` for index/perf if seat volumes are high.
- **Anchor change + seats:** how time-weighted seats interact with a billing-anchor
  change mid-period (two prorations stacking).
