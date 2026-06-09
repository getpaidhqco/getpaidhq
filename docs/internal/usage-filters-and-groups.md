# Usage filters & groups

How a single meter prices different slices of its usage differently, and how one
priced charge is broken out into per-segment invoice lines. Companion to
`billing-model.md` (the four-record spine) — this doc only covers the two metered
dimensions and the model that carries them.

## The one-line principle

- **Filter** = the dimension that *sets the rate*. Few, fixed categories; different
  prices per value.
- **Group** = a dimension you want *broken out on the invoice* at the **same rate**.
  Many, open-ended identities; a different price per value would make no sense.

> The decision test — ask one question about a dimension:
> **"Would I ever set a different per-unit price for two values of this?"**
> **YES → Filter** (message type SMS vs MMS, US vs intl, hot vs cold storage).
> **NO → Group** (project, api_key, user_id, environment, phone number).

Filter is a *pricing* axis; group is an *attribution / visibility* axis. The rate is
held constant across a group; only the count changes.

## Worked example — a messaging API

Bill messages; the rate depends on message **type** (a filter). The customer is an
agency that wants each client **project** itemised (a group) — but every project pays
the same per-message rate.

```
BillableMetric "messages"  (count)
   Filters:  [ { Field: "type", Values: ["SMS","MMS"] } ]   ← rate axis (enumerated)
   GroupBy:  [ "project" ]                                    ← breakout axis (open)

Variant "Pro"
   ├─ Price metered →messages  FilterField=type FilterValue="SMS"  $0.01/msg
   └─ Price metered →messages  FilterField=type FilterValue="MMS"  $0.05/msg
```

Events in a period (everything lives in `MeterEvent.Metadata`, a jsonb map — no event
schema change):

| type | project  | count |
|------|----------|-------|
| SMS  | acme     | 1000  |
| SMS  | globex   | 500   |
| MMS  | acme     | 200   |
| MMS  | initech  | 100   |

Billing is two nested loops — **filter picks the rate (outer), group splits the line
(inner), rate constant inside the group**:

```
for each Price (filter → WHERE metadata->>'type' = value, picks rate):
    for each distinct project (group → GROUP BY metadata->>'project', splits line):
        units  = count(this type, this project)
        amount = units × rate          # rate identical for every project
        emit one invoice line
```

Resulting invoice:

```
SMS  project=acme     1000 × $0.01 = $10.00
SMS  project=globex    500 × $0.01 =  $5.00
MMS  project=acme      200 × $0.05 = $10.00
MMS  project=initech   100 × $0.05 =  $5.00
                                       ──────  $30.00
```

Without the group, identical total / rates — only line granularity differs:

```
SMS  1500 × $0.01 = $15.00
MMS   300 × $0.05 = $15.00
                     ──────  $30.00
```

### Why `project` can't just be a filter

1. **New projects appear constantly.** A filter must enumerate every value up front;
   an unlisted value matches nothing and falls to the default price. A group discovers
   values from the events.
2. **You'd duplicate the same rate across every value**, and they'd drift on a price
   change.
3. **You never want a different price per project** — the split is for cost
   attribution, not pricing.

## The model

Both axes are declared on the **Metric** (it is "what to measure and how to slice it");
the **Price** only ever names one filter value and carries that slice's rate.

### `domain.BillableMetric` (meter.go)

```go
type MetricFilter struct {
    Field  string   // event metadata key, e.g. "type"
    Values []string // enumerated values that get their own Price; default = NOT IN these
}

type BillableMetric struct {
    ...
    Filters []MetricFilter // declared rate dimensions + their known values
    GroupBy []string       // open breakout dimensions (key names only)
}
```

`FilterValues(field)` returns a field's declared values — used to compute the
default/catch-all charge's `NOT IN` set without a Price having to peek at its siblings.

### `domain.Price` (price.go)

```go
type Price struct {
    ...
    BillableMetricId string // metered when set
    FilterField      string // metadata key this charge filters on (matches a MetricFilter); "" = whole meter
    FilterValue      string // the value; "" with FilterField set = the default/catch-all charge
}
```

- `FilterField == ""` → no filter; bills the whole meter (today's behaviour, unchanged).
- `FilterField="type", FilterValue="SMS"` → `WHERE metadata->>'type' = 'SMS'`.
- `FilterField="type", FilterValue=""` → **default**: `metadata->>'type' NOT IN (declared values) OR IS NULL`.

### Read path — `port.UsageQuery` / `EventStore`

`UsageQuery` gains `FilterField`, `FilterValue`, `FilterExclude` (the default's NOT-IN
set), and `GroupBy`. The filter is one extra `WHERE` in `EventStore.scope`, so it
applies to every aggregation uniformly. Grouping is a new store method:

```go
type GroupedUsage struct { Key, Value string; Quantity decimal.Decimal }
AggregateGrouped(ctx, q UsageQuery, agg AggregationType, groupKey string) ([]GroupedUsage, error)
```

The invoice builder (`service.InvoiceService.BuildForBillingPeriod`) resolves each
metered Price, applies its filter, and:
- **no GroupBy** → one usage line (`UsageLineFromPrice`), as today;
- **GroupBy set** → one line per returned `GroupedUsage` (`UsageLineFromPriceGrouped`),
  each at the Price's single rate, the group key/value stamped into the line's
  `Metadata` and description.

The default charge's exclude set is `Metric.FilterValues(price.FilterField)`.
Unclassified events (filter field absent) fall to the default charge.

## v1 bounds (deliberate)

- **Single group dimension** honoured (`GroupBy[0]`); the column stores a slice for
  forward-compat but a metric with >1 group key errors at aggregation time.
- **Grouped aggregation** supports `count`, `sum`, `unique_count`, `max`. `latest` and
  `weighted_sum` grouped error (they need window/DISTINCT-ON; `weighted_sum` is
  unimplemented even ungrouped).
- **No new API surface yet** — this is the model + billing computation. Exposing
  filters/groups on the meter-create DTO + OpenAPI + SDK is a follow-up.
- **Ingest unchanged** — dimensions are ordinary `Metadata` keys; events need not be
  re-validated against a metric's declared filters.
- **Back-compat** — empty `Filters`/`GroupBy` and empty `Price.FilterField` reproduce
  today's exact behaviour.

