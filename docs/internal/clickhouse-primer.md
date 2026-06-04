# ClickHouse primer — how it stores, how it queries, where it fits in gphq

Reference notes for the usage-metering work. Explains ClickHouse mechanics from
first principles, how Lago uses it, and how it slots behind our `EventStore` port as
one of two parity adapters (Postgres + ClickHouse). Companion to the design spec
`docs/superpowers/specs/2026-06-04-usage-based-metering-design.md` and
`research/lago/usage-based-billing.md`.

---

## 1. Mental model: columns, not rows

Postgres stores a row together on disk — `(org, sub, code, ts, value, props)` for
row 1, then row 2. Great for "fetch/update this record"; bad for "sum one column
over 10M records," because you drag every other column through disk/cache to reach
`value`.

ClickHouse stores each **column separately**. All `value`s are contiguous in one
file, all `timestamp`s in another. `SUM(value)` reads only the `value` file. That
single fact is ~90% of why it's fast for aggregation and slow for transactional
work.

```
ROW STORE (Postgres)                COLUMN STORE (ClickHouse)
┌───────────────────────┐          value:     [12, 5, 30, 7, 99, ...]
│org sub code ts value..│ row1      timestamp: [t1, t2, t3, t4, t5, ...]
│org sub code ts value..│ row2      code:      [api, api, gb, api, gb,...]
└───────────────────────┘           ...each column its own compressed file
SUM(value) touches everything       SUM(value) reads one file
```

A column of similar values (a month of timestamps, one repeated `code`) compresses
10×+, so there is even less to read.

---

## 2. Storage — the MergeTree engine

The engine family you'll use (`MergeTree`, `ReplacingMergeTree`, ...). Four ideas:

**Parts.** Every `INSERT` writes a new immutable directory (a *part*) holding a
sorted, compressed copy of the columns. Parts are never edited in place; a
background process **merges** small parts into bigger ones (hence "MergeTree").
Consequence: **many tiny inserts → many tiny parts → "too many parts" errors.**
Insert in batches.

**ORDER BY (sorting key).** Within every part, rows are physically sorted by the
table's `ORDER BY`. Most important schema decision: queries that filter on a
**prefix** of `ORDER BY` are cheap.

**Sparse primary index + granules.** Unlike Postgres's per-row B-tree, ClickHouse's
primary index is *sparse* — it records the sort-key value once per
`index_granularity` rows (default 8192). Each 8192-row block is a **granule**; the
recorded entry is a **mark**. A query binary-searches marks and reads **only the
granules** whose key range overlaps the `WHERE`, skipping the rest unread.

```
ORDER BY (org_id, subscription_id, code, timestamp)
marks:  [A,1,api,t0] [A,1,api,t9000] [A,2,gb,t0] ...
         └ granule 0 ┘└ granule 1 ───┘└ granule 2┘
WHERE org=A AND sub=1 AND code=api AND ts in [t0,t9000)
  → read granules 0–1 only
```

**PARTITION BY.** Coarser split on top of parts, e.g. `PARTITION BY
toYYYYMM(timestamp)`. Queries with a time filter **prune** whole partitions, and
`DROP PARTITION` expires old data instantly (cheaper than `DELETE`). Don't
over-partition (monthly/daily fine; hourly is a mistake).

Our query shape — "this org/sub/code over this time range" — maps perfectly:
partition-prune by month, then mark-skip to the exact granules.

---

## 3. Querying — and the deduplication wrinkle

Reads are SQL (plus window functions for weighted-sum). Speed comes from
mark-skipping, so **the WHERE must lead with the ORDER BY prefix.**

The wrinkle: ClickHouse **cannot enforce uniqueness**. Postgres gives exactly-once
for free (`UNIQUE(org,sub,transaction_id)` + `ON CONFLICT DO NOTHING`). ClickHouse
dedups one of two ways:

**`ReplacingMergeTree(version)`** — during a background merge, rows with identical
`ORDER BY` collapse to the one with the highest `version`. But dedup is **eventual**
(only on merge). So a naive `SELECT sum(value)` can double-count a re-sent event.
Fix at **read time**:

```sql
-- A) FINAL — forces dedup at query time. Correct, heavier.
SELECT sum(value) FROM meter_events FINAL WHERE ...

-- B) dedup subquery — usually faster than FINAL.
SELECT sum(value) FROM (
  SELECT argMax(value, ingested_at) AS value
  FROM meter_events
  WHERE org_id=? AND subscription_id=? AND code=? AND timestamp >= ? AND timestamp < ?
  GROUP BY transaction_id
)
```

Several aggregations are **dedup-immune** and need no FINAL:
- count → `uniqExact(transaction_id)` (counting distinct txn ids *is* the dedup)
- max → `max(value)` (a duplicate max changes nothing)
- unique_count → `uniqExact(value)`
- latest → `argMax(value, timestamp)`
- **sum** is the only one needing the dedup subquery.

Headline parity fact: **Postgres dedups on write, ClickHouse dedups on read.** Same
outcome, opposite mechanism.

---

## 4. Ingestion

**Batch INSERT** (start here): accumulate and insert in chunks (thousands of rows),
or enable `async_insert` so the server buffers small inserts into bigger parts.
Either way avoids the tiny-part problem.

**Kafka engine + materialized view** (Lago's pure-SQL ETL):
```sql
CREATE TABLE events_queue (...) ENGINE = Kafka SETTINGS kafka_topic_list='raw_events', ...;
CREATE MATERIALIZED VIEW events_mv TO meter_events AS SELECT ... FROM events_queue;
```
The MV fires on each batch the Kafka table consumes and writes to the MergeTree
target — no app code in the write path. Only needed at firehose rate; for us, NATS →
a small consumer doing batched INSERTs is the equivalent with far less infra.

---

## 5. What it's bad at (keep these in Postgres)

- **Updates / deletes** — `ALTER TABLE ... UPDATE/DELETE` are async "mutations" that
  rewrite whole parts. Treat events as **append-only**.
- **Uniqueness / FK / constraints** — none; integrity is the app's job.
- **Transactions** — no multi-statement/multi-table ACID.
- **Point lookups / single-row writes** — possible, not the point.

So all relational data (BillableMetric definitions, Price, Subscription, Order)
stays in Postgres. ClickHouse only holds the append-only `meter_events` firehose and
answers aggregation queries — exactly the boundary the `EventStore` port draws.

---

## 6. How Lago uses it (recap)

- `events_raw` (MergeTree) = firehose; `events_enriched` (ReplacingMergeTree) = what
  aggregation reads, with a typed `decimal_value` column produced by the Go enricher.
- Ingest is pure SQL: Kafka-engine queue table → materialized view → target.
- `ORDER BY (organization_id, code, external_subscription_id, toDate(timestamp),
  timestamp, transaction_id)`; partitioned to allow pruning.
- Aggregation queries are plain column aggregates over the period; weighted-sum uses
  window functions (`leadInFrame` + running `SUM() OVER`).
- Per-org switch `clickhouse_events_store?` chooses ClickHouse vs Postgres store —
  the precedent for our two-adapter approach.

Details: `research/lago/usage-based-billing.md` §6–7.

---

## 7. Where it fits in gphq — two parity adapters behind one port

`port.EventStore` is the seam. Both adapters implement the same interface; config
picks one (or both, to compare). Same `UsageQuery` in, same `float64` out; different
mechanics inside, so we can compare them. Events attach to a **customer** and a
**metric**, and may carry an optional **`external_id`** that serves as the dedup key
(see the design spec for the field meanings).

```
                  UsageService / billing-cycle workflow
                        │  (depends only on port.EventStore)
              ┌─────────┴──────────┐
  USAGE_EVENT_STORE=postgres   USAGE_EVENT_STORE=clickhouse
              │                    │
  internal/adapter/usage/      internal/adapter/usage/
    postgres/event_store.go      clickhouse/event_store.go
  - dedup on WRITE             - dedup on READ
    (ON CONFLICT)                (ReplacingMergeTree + argMax)
  - row store, B-tree         - column store, sparse index
  - read-after-write exact    - batched insert, eventual merge
```

**ClickHouse schema** (gphq `meter_events`):
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
ORDER BY (org_id, customer_id, metric_code, timestamp, id);
```
`id` (unique) is last in `ORDER BY`, so two distinct events at the same instant are
never merged into one. Resends that share an `external_id` are collapsed at read time
instead — group by `external_id` when it's set, else by `id` (so events without one
are all kept). `ingested_at` is the version, so a collapsed duplicate keeps the latest
copy.

**Methods, side by side** (same outcome, different SQL). `dedup_key` below is
`coalesce(nullif(external_id,''), id)`:

| port method | Postgres adapter | ClickHouse adapter |
|---|---|---|
| `Ingest` | `INSERT ... ON CONFLICT (org, external_id) DO NOTHING` | `INSERT` with `async_insert`; dedup deferred to read |
| `Count` | `count(*)` | `uniqExact(dedup_key)` |
| `Sum` | `COALESCE(SUM(value),0)` | `SUM` over `argMax(value, ingested_at) GROUP BY dedup_key` |
| `Max` | `MAX(value)` | `max(value)` |
| `UniqueCount` | `COUNT(DISTINCT value)` | `uniqExact(value)` |
| `Latest` | `value ORDER BY timestamp DESC LIMIT 1` | `argMax(value, timestamp)` |
| `WeightedSum` | `SUM() OVER (ORDER BY ts)` × `LEAD(ts)` gap / period | same logic, CH `leadInFrame` |

All time filters are **half-open `[from, to)`** in both — pin this in a shared helper
so boundary semantics can't drift (a classic parity bug).

**Parity wrinkles to watch:**

| concern | Postgres | ClickHouse | same outcome via |
|---|---|---|---|
| dedup | write-time constraint | read-time `argMax`/`uniqExact` | identical totals; CH pays at query |
| read-after-write | immediate, exact | immediate but may include dupes pre-merge | read-time dedup makes it exact |
| precision | `numeric` | `Decimal(38,9)` | both decimal; round to cents once in Go |
| insert latency | per-row fine | needs batching | buffer/flush in CH adapter |
| expiry | partitioning / `DELETE` | `DROP PARTITION` | same retention behavior |

**Config switch** (extend `Env` + `viper.BindEnv`, per repo convention):
```
USAGE_EVENT_STORE=postgres    # default
USAGE_EVENT_STORE=clickhouse
USAGE_EVENT_STORE=compare     # write both, read postgres, check clickhouse in background, log diffs
```
`compare` checks the two on real data without affecting billing: every aggregation
runs both, returns Postgres, logs mismatches + timings.

**Parity test harness:** one table-driven test feeds the same event set (including
resent duplicates, out-of-window events, boundary timestamps) into both adapters,
runs every aggregation across several `UsageQuery`s, and asserts equality within an
epsilon. Same test body, two adapters — that's how we know they agree.

---

## 8. Recommendation

Ship the **Postgres adapter first**, but **define the port and write the parity
harness up front** so the ClickHouse adapter slots in and can be checked via `compare`
mode. Don't stand up the Kafka-engine/MV tier — a NATS consumer doing batched
INSERTs gives the same ClickHouse benefits without running Kafka. You then get a
real, measured comparison (latency, correctness, ops cost) on your own data before
committing to either backend long-term.
