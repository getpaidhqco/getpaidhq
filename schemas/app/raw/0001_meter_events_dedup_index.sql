-- Partial unique index backing write-time dedup of usage events.
--
-- Prisma cannot express a partial unique index (the WHERE clause), so `db push` does
-- NOT create it. Without it, a resend carrying a previously-seen external_id is
-- inserted again instead of raising a duplicate error, and Count/Sum double-count —
-- which also breaks parity with the ClickHouse backend (which dedups on read).
--
-- The application also ensures this index at boot (postgres.EnsureUsageSchema), so it
-- is created automatically; this file is the canonical DDL for deploy pipelines and
-- review. Apply against whichever database holds meter_events (the operational DB by
-- default, or USAGE_DATABASE_URL when the usage store is split out).
--
-- Optional ids are stored as NULL when absent (never ''), so events without a client
-- id are never deduped (each is a distinct event) — the partial index only covers
-- rows that actually carry an external_id.
CREATE UNIQUE INDEX IF NOT EXISTS meter_events_external_id_uq
    ON meter_events (org_id, external_id)
    WHERE external_id IS NOT NULL;
