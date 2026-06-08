# Usage ingest (rename + batch) and subscription current-period usage read

## Context

Two changes to the usage/metering HTTP surface:

1. **Rename + batch ingest.** Today `POST /api/usage/events` accepts exactly one usage event
   per request and has **no authorization** enforced (unlike every other handler). Rename it to
   `POST /api/usage/ingest` and accept many events per request, so callers can send usage in
   batches instead of one HTTP round trip per event.
2. **Read current-period usage.** There is currently **no** read API for usage — ingestion is
   write-only and aggregation only happens internally during invoice generation. Add
   `GET /api/subscriptions/{id}/usage` returning the usage quantity per meter for the
   subscription's current billing period, matching exactly what will be invoiced.

Both are pre-1.0, local-only. The only external consumer of the ingest path is the hand-written
TypeScript SDK (which separately has a mismatched path bug) and the web dashboard via the SDK —
updating those is a follow-up, not part of this change.

### Decisions locked in (brainstorming)

- **Hard rename** — `/events` ceases to exist; only `/ingest` remains.
- **Batch wrapper** `{"events":[...]}`, 1–N events; per-event results; partial success.
- **Per-event results, HTTP 200** — a single invalid event (e.g. unknown `metric_code`) never
  fails the others; it returns a `rejected` result with a reason. Whole-request problems (empty
  array, over the cap, malformed JSON) are a 400.
- **Read returns quantity only** — per meter `{metric_code, aggregation, quantity}` plus the
  period window. No pricing/amount math (that's invoice-preview, deferred).
- **Add Cedar authz to both** endpoints. API keys map to `RoleAdmin` → covered by the admin
  wildcard, so server-to-server ingestion keeps working unchanged.
- **Read is a plain per-subscription sum** — load the subscription, resolve its own metered
  price (sub → order item → price → meter), and sum its usage over the current period via the
  existing `UsageService.UsageForSubscription`. This is the 1:1 subscription↔price model from
  `CONTEXT.md`. (Invoice-time aggregation of an order's *sibling* metered items onto a primary
  subscription is a billing-run concern and is deliberately **not** in this read.)

## Endpoints

### `POST /api/usage/ingest`

Request — always the batch wrapper (a single event is a one-element array):
```json
{ "events": [
  { "metric_code": "api_calls", "external_customer_id": "cust-42",
    "subscription_id": "sub_x", "external_id": "evt-1",
    "timestamp": "2026-06-08T10:00:00Z", "metadata": { "value": "10" } }
] }
```
- Validation: `events` `required,min=1,max=500,dive` (each element is the existing
  `RecordEventRequest` with `metric_code` required). Empty / >500 / malformed → **400**.

Response — always **200**, results aligned by request index:
```json
{ "results": [
  { "index": 0, "id": "mev_...", "status": "recorded" },
  { "index": 1, "status": "rejected", "error": "unknown metric code" }
] }
```
- `status` ∈ `recorded` | `duplicate` (seen `external_id`) | `accepted` (durably queued, async
  ingest mode) | `rejected` (validation failed; `error` set, no `id`).

### `GET /api/subscriptions/{id}/usage`

```json
{ "subscription_id": "sub_x",
  "current_period_start": "2026-06-01T00:00:00Z",
  "current_period_end":   "2026-07-01T00:00:00Z",
  "meters": [ { "metric_code": "api_calls", "aggregation": "sum", "quantity": "1234" } ] }
```
- `quantity` is a decimal string (preserves precision — usage units are `decimal.Decimal`).
- `404` if the subscription doesn't exist; `meters: []` when the subscription has no metered
  lines it owns.

## Components & changes

### Port: `EventIngestor` gains batch
`internal/core/port/usage.go` — add
`IngestBatch(ctx, []domain.MeterEvent) ([]IngestResult, error)` to `EventIngestor`.
- Sync path (`postgres.EventStore`) **already implements `IngestBatch`** (one INSERT with
  `ON CONFLICT DO NOTHING`); just satisfy the wider interface.
- JetStream ingestor: implement `IngestBatch` as loop-publish, returning `accepted` per event
  (mirrors its single-`Ingest` semantics).

### Service: batch record
`internal/core/service/usage.go` — add
`RecordEvents(ctx, []port.RecordEventInput) ([]port.IngestResult, error)`:
1. For each input, run the existing per-event validation (meter lookup by code, customer /
   subscription resolution, value/field extraction). **Cache meter lookups by code** within the
   batch. Invalid input → an `IngestResult{Status: rejected, ...}` carrying the reason; it does
   not reach the store.
2. Batch-ingest the valid events via `EventIngestor.IngestBatch`; map results back to their
   original request index.
3. Publish `usage.recorded` for each recorded event (preserve current behaviour).
The single-event `RecordEvent` is removed; the handler always calls `RecordEvents`.

The per-event validation currently inline in `RecordEvent` is extracted into a helper
(`buildEvent(ctx, in) (domain.MeterEvent, error)`) reused per batch element.

### Service: current-period usage
New `UsageService.CurrentPeriodUsage(ctx, orgId, subscriptionId) ([]MeterUsage, error)`:
1. Load the subscription (`subscriptionRepository.FindById`) → `port.ErrNotFound` → handler 404.
2. If `CurrentPeriodStart` is zero (pending/cancelled) → return empty (no period, no usage).
3. Resolve the subscription's own price: `sub.OrderItemId` → `OrderItem` (orderRepository) →
   `Price` (priceRepository). If `!price.IsMetered()` → return empty.
4. `units = UsageForSubscription(ctx, sub, price, sub.CurrentPeriodStart, sub.CurrentPeriodEnd)`
   (already encapsulates per-meter attribution); look up the meter for its code + aggregation.
5. Return `[]MeterUsage{ {MetricCode, Aggregation, Quantity: units} }`.

`MeterUsage` is a small read-model struct in the service package. `UsageService` gains
`orderRepository` + `priceRepository` (DI in `app.go`) to do the sub→price resolution. No invoice
is built; this is a straight per-subscription usage sum over the current period.

### HTTP handler & authz
`internal/adapter/http/usage_handler.go`:
- `UsageHandler` gains an `authz port.Authz` field (wired in `internal/config/app.go`).
- `RegisterRoutes`: `POST /usage/ingest` (was `/usage/events`), and a second group registering
  `GET /subscriptions/{id}/usage`.
- `Ingest` handler: `enforce(c, h.authz, port.ActionRecordUsage)`; binds
  `IngestEventsRequest{ Events []RecordEventRequest }`; calls `RecordEvents`; returns
  `IngestEventsResponse{ Results []IngestEventResult }`.
- `SubscriptionUsage` handler: `enforce(c, h.authz, port.ActionReadUsage)`; calls
  `CurrentPeriodUsage`; returns `SubscriptionUsageResponse`.
- DTOs: `IngestEventsRequest`, `IngestEventResult{Index, Id, Status, Error omitempty}`,
  `IngestEventsResponse`, `SubscriptionUsageResponse`, `MeterUsageResponse{MetricCode,
  Aggregation, Quantity}`.

### Authz constants & policy
- `internal/core/port/auth.go`: `ActionRecordUsage = "RecordUsage"`,
  `ActionReadUsage = "ReadUsage"`.
- `policy.cedar`: admin wildcard already covers both (API key + Clerk org:admin). Add
  `Action::"ReadUsage"` to the **member** permit list so dashboard users can view usage;
  `RecordUsage` stays admin/api-key only.

## Error handling

- Whole-request: empty `events`, `> 500`, or unbindable body → `400` via the validator / binder.
- Per-event: unknown `metric_code`, missing required value field for a non-count aggregation,
  unresolved customer/subscription → `rejected` result with `error`, HTTP still `200`.
- Read: unknown subscription → `404` (`port.ErrNotFound` translated by the envelope).
- Authz failures → `403` before any service call (existing `enforce` pattern).

## Testing

- **HTTP** (`usage_handler_test.go`): batch happy path (mixed `recorded`/`duplicate`/`rejected`,
  index alignment); `400` on empty array and on > 500; authz guard (non-permitted role → 403 on
  both routes, before service); read happy path; read `404`; a `member` user **can** read usage
  but **cannot** ingest.
- **Service** (`usage_test.go`): `RecordEvents` validation + index mapping + meter-lookup
  caching; `CurrentPeriodUsage` returns the metered price's usage quantity over the current
  period; zero period → empty; non-metered subscription → empty; unknown subscription →
  `ErrNotFound`.
- **Integration** (`//go:build integration`): `EventStore.IngestBatch` round-trip with an
  in-batch duplicate `external_id` (deduped, not double-counted).

## Verification (end-to-end)

1. `make test` and `make test-integration` green.
2. `make run`, confirm `openapi.json` regenerates with `POST /usage/ingest` (no `/usage/events`)
   and `GET /subscriptions/{id}/usage`. Exercise:
   - Ingest a 3-event batch with one unknown metric → `200`, results `recorded, recorded,
     rejected`.
   - Re-send one event → `duplicate`.
   - `GET /subscriptions/{id}/usage` → the subscription's metered usage quantity for its current
     period (and `meters: []` for a non-metered or pending subscription).
   - An API-key request still ingests (admin wildcard); a `member` Clerk user can read but not
     ingest.

## Follow-up (not in this change)

- Regenerate and commit `openapi.json` (server boot).
- Update the hand-written SDK `getpaidhq-sdk/src/resources/usage.ts` (also fix its mismatched
  `/api/usage-events` path) and bump the SDK; web picks it up via the SDK.
- Update `CLAUDE.md` and `docs/superpowers/specs/2026-06-04-usage-based-metering-design.md` path
  references.
