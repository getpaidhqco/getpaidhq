# Transactional Outbox for Domain Events — Design

Date: 2026-07-19
Status: approved (design review in conversation)

## Problem

Domain events publish straight to NATS core via `port.PubSub.Publish(orgId, topic, message)`.
The signature has no `ctx`, so a publish can never be atomic with the database write it
announces. Nearly all ~60 call sites in `internal/core/service/` ignore the returned error
(`_ = s.pubsub.Publish(...)`), so a failed publish is silently dropped. Flows that use
`RunInTx` publish after commit by convention, which still loses the event if the process
dies between commit and publish; everything else publishes with no transaction at all.

The fix is the transactional outbox pattern: write the event to an `outbox_events` table in
the same transaction as the business write, and have a background relay deliver it to NATS.

## Decisions

- **Scope**: all publishes go through the outbox — one code path, durability everywhere.
- **Placement**: the outbox is default behavior, not an application service. `Outbox` and
  `Relay` live in `internal/lib/pubsub` (the pubsub infrastructure package) and depend only
  on ports (`port.OutboxRepository`, `port.TxManager`, and a narrow raw-publish port
  implemented by the NATS adapter). NATS becomes pure transport; any service holding
  `port.PubSub` gets durability by construction.
- **Org scoping**: rows carry `org_id NOT NULL` (it is part of the stored envelope and needed
  for inspection and future per-org replay), but the relay is a single global queue — no
  per-org partitioning, ordering, or fairness in v1. NATS subjects carry no org id and
  consumers have never had ordering guarantees, so global best-effort order is a strict
  improvement; per-aggregate blocking ordering stays a future opt-in if a consumer ever
  needs it.
- **Relay**: in-process polling relay (`FOR UPDATE SKIP LOCKED`), no CDC, no new infrastructure.
- **Row lifecycle**: mark `published_at` on success; periodic purge of published rows older
  than 24h; failing rows retry with backoff and are left for inspection after max attempts.
- **Ordering**: best-effort by insertion order; a failing row is skipped and retried later,
  it never blocks later rows.

## Schema

New Goose migration `schemas/app/migrations/00010_outbox_events.sql`:

```sql
CREATE TABLE outbox_events (
    id              BIGSERIAL PRIMARY KEY,        -- publish order
    event_id        TEXT        NOT NULL,         -- evt_<ULID>, stable envelope id
    org_id          TEXT        NOT NULL,
    topic           TEXT        NOT NULL,
    payload         JSONB       NOT NULL,         -- full PubSubPayload envelope
    attempts        INT         NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ,
    last_error      TEXT,
    published_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX outbox_events_pending_idx ON outbox_events (id) WHERE published_at IS NULL;
```

There is no status column. States are derived:

- pending: `published_at IS NULL AND attempts < max`
- failed (left for inspection, excluded by the relay): `published_at IS NULL AND attempts >= max`
- published: `published_at IS NOT NULL` (purged once older than the retention window)

## Write path

`port.PubSub.Publish` gains a context:

```go
Publish(ctx context.Context, orgId string, topic string, message any) error
```

A new `Outbox` (`internal/lib/pubsub`) implements `port.PubSub`:

- `Publish` builds the `domain.PubSubPayload` envelope — the `evt_` id and `created_at`
  are generated at insert time (moved out of the NATS adapter) — and inserts a row via a
  new `port.OutboxRepository.Create(ctx, ...)`. The repository uses `dbFromCtx`, so the
  insert automatically joins the ambient transaction inside `RunInTx` and is a standalone
  insert otherwise.
- `Subscribe` delegates to the NATS adapter unchanged.

`OutboxPubSub` is wired in `app.go` in place of the NATS adapter, so every service keeps
its `port.PubSub` dependency; call sites change only mechanically (add `ctx`).

Semantic change in transactional flows (order completion, subscription charge
success/failure, billing-anchor change): the publish **moves inside the `RunInTx` closure**
and its error propagates, rolling back the business write with the event. This is the
atomicity the pattern exists for. The AGENTS.md convention "pubsub after commit" is updated:
outbox publishes belong inside the transaction; only non-DB side effects (workflow starts)
stay post-commit.

## Relay

`Relay` (`internal/lib/pubsub`), a background goroutine started in `app.go`, holding
`port.TxManager`, `port.OutboxRepository`, and the real NATS adapter through a narrow
raw-publish port defined in `internal/core/port`
(`PublishPayload(topic string, data []byte) error`) so the stored envelope is not
double-wrapped and core never imports the adapter.

Loop, every poll interval:

1. `RunInTx`:
   - `SELECT ... WHERE published_at IS NULL AND attempts < maxAttempts AND
     (next_attempt_at IS NULL OR next_attempt_at <= now())
     ORDER BY id LIMIT batchSize FOR UPDATE SKIP LOCKED`
   - For each row: publish the stored envelope bytes to the row's topic on NATS.
     - success → set `published_at`
     - failure → `attempts++`, set `next_attempt_at` (exponential backoff), `last_error`
   - commit
2. Every ~10 minutes, a purge pass deletes published rows older than 24h.

`SKIP LOCKED` makes concurrent server instances safe. Publishing inside the lock-holding
transaction is deliberate: a crash after publish but before commit means the row is
republished — at-least-once delivery.

Relay tuning values are constants, not env vars: poll interval 1s, batch size 100,
max attempts 10. The purge pass is configurable via env (added to `lib.Env` /
`.env.example` per convention): `OUTBOX_PURGE_INTERVAL` (default `10m`) and
`OUTBOX_RETENTION` (default `24h`).

## Storage adapters

`port.OutboxRepository` is implemented in both `postgresgorm` and `postgrespgx`, added to
both `RepoSet`s, with shared conformance coverage in `storagetest`:

- `Create` joins the ambient tx; a rolled-back tx leaves no row
- lock/claim semantics (pending selection honors attempts/backoff, `SKIP LOCKED`)
- mark published / record failure
- purge respects the retention cutoff

## Delivery semantics and engine parity

Producer side becomes at-least-once (today: at-most-once with silent drops). Consumers
(`SubscriptionEventBridge`, webhook fan-out, dunning orchestration, customer handler) may
see duplicates after a relay crash, and retry-with-backoff can reorder events within an
org (a failed row is delivered after later rows). Consumers already tolerate drops with
no ordering guarantee from NATS core, so neither is a new burden. `UpdateSubscriptionWorkflow` is fire-and-forget and
workflow starts are idempotent via deterministic ids, so no engine-adapter changes are
needed — observable behavior is identical on Hatchet and Temporal.

## Testing

- `storagetest` conformance for `OutboxRepository`, run by both drivers.
- Relay unit tests with a fake publisher: success marks published; failure bumps
  attempts/backoff; rows at max attempts are excluded; purge deletes only old published rows.
- Integration test: a rolled-back business tx leaves no outbox row; a committed one results
  in a NATS publish by the relay.
