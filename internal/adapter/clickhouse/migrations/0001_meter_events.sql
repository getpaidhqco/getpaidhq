-- ClickHouse usage-event store. Mirrors the Postgres meter_events table; the two
-- back the same port.EventStore and must produce identical aggregation results.
--
-- ReplacingMergeTree(ingested_at): a background merge collapses rows that share the
-- full ORDER BY to the one with the highest ingested_at. `id` (unique per event) is
-- last in ORDER BY so two distinct events at the same instant are NEVER merged into
-- one. Resends that share an external_id are instead collapsed at READ time
-- (dedup_key = coalesce(nullif(external_id,''), id)), so dedup is exact even before
-- a merge runs. See docs/internal/clickhouse-primer.md §7.
CREATE TABLE IF NOT EXISTS meter_events (
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
