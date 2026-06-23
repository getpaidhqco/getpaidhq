# Idempotent CreateOrder + separate `/orders/{id}/pay` — Design Spec

**Date:** 2026-06-23
**Status:** Settled — ready for implementation planning
**Area:** Orders — make creation atomic + idempotent, and split payment-session init into its own retryable call.

---

## 1. Why

`OrderService.CreateOrder` today (read `internal/core/service/order.go:76`) is **not transactional** and **not idempotent**, and it does an **external PSP call mid-flow**:

- It writes cart → customer → order → order items → subscriptions → coupon reservation as **independent commits** (no `RunInTx`), then calls `gw.InitPayment` (order.go:307). Any failure after the first write **orphans** a partial order; the `subscription.created` event is even published mid-flow (order.go:280).
- `orderId` is freshly generated every call, so **re-calling `CreateOrder` with the same details creates a duplicate** order + customer + subscriptions + reservation. There is no `Idempotency-Key` handling anywhere on the order path (the `IdempotencyKeyRepository` is webhook-only).
- Because the PSP call lives **inside** `CreateOrder` and there is **no re-init endpoint** (routes are create / `{id}/complete` / get / list / list-subs), a committed-order-whose-PSP-init-failed can only be "retried" by calling `CreateOrder` again → a duplicate.

This spec fixes all three by **separating the two concerns**:

- **`CreateOrder`** = a pure, transactional, idempotent **write**. Returns the order. No gateway call.
- **`POST /orders/{id}/pay`** = initialise (or return) the PSP payment session for an existing order. Independently retryable.

> This also closes the gap flagged in `2026-06-23-coupon-reservation-and-application-design.md`: the coupon reservation now commits atomically inside the `CreateOrder` tx, so a refused coupon rolls back the whole order.

---

## 2. `CreateOrder` — idempotent pure write

### 2.1 Transactional

Reads/validation stay **before** the tx (fail fast, no tx held): session/cart lookup, archived-product guard, customer existence. The **write sequence** goes in one `s.tx.RunInTx`:

```
cart.Create (direct path) → customer find/create → order.Create(idempotency_key)
  → order items → subscriptions (collect them) → link items → coupons.Reserve(...)
```

- All repo calls inside use the closure's `ctx` → they join the tx via `dbFromCtx`.
- `coupons.Reserve` already wraps `RunInTx`; inside the outer tx it opens a **savepoint** (gorm nested-tx / the pgx `RunInTx`), so a coupon refusal rolls back its savepoint *and* returns the error → the outer tx rolls back → **no order**.
- **No external call inside the tx.**

**Post-commit side-effects** (after the tx returns nil): publish `TopicSubscriptionCreated` for each created subscription (moved out of mid-flow). No PSP call here at all.

### 2.2 Idempotency

- An optional **`Idempotency-Key`** HTTP header → `port.CreateOrderInput.IdempotencyKey` → stored on the order in a new `orders.idempotency_key` column, with a **unique partial index `(org_id, idempotency_key) WHERE idempotency_key IS NOT NULL`**.
- The order `INSERT` inside the tx carries the key. On a **replay** the insert raises a unique violation → the tx rolls back (nothing partial) → we `SELECT` the existing order by `(org_id, idempotency_key)` and **return it** (HTTP 200, same order). The DB constraint *is* the atomic dedup — no TOCTOU, and two concurrent replays resolve to the one winner.
- **No key supplied** → behaves as today (each call is a new order). Dedup is opt-in.

```go
err := s.tx.RunInTx(ctx, func(ctx) error { /* writes + Reserve, order carries key */ })
if isUniqueViolation(err) && input.IdempotencyKey != "" {
    existing, ferr := s.orderRepository.FindByIdempotencyKey(ctx, orgId, input.IdempotencyKey)
    if ferr != nil { return resp{}, ferr }
    return domain.CreateOrderResponse{Order: existing}, nil   // idempotent replay
}
if err != nil { return resp{}, err }
// post-commit: publish subscription.created; return the order
```

### 2.3 Response change

`CreateOrder` no longer initialises a PSP session, so `domain.CreateOrderResponse` drops `Psp` (returns only `Order`). Callers that needed a payment session now call `/pay` (§3). This is a **breaking API change** for clients that read `resp.psp` (SDK / web / checkout update; the order e2e/tests that assert `resp.Psp` change).

---

## 3. `POST /orders/{id}/pay` — payment-session init

A new route on the orders group: initialise the PSP payment session for an existing **pending** order and return it.

```
POST /api/orders/{id}/pay  →  { "psp": <InitPaymentResponse> }
```

Service method `OrderService.InitOrderPayment(ctx, orgId, orderId, opts)`:
1. Load the order; require `status == pending` (else `ConflictError`).
2. **Idempotent on the session:** if the order already has a stored live payment session, return it. Otherwise resolve the gateway (`gatewayFactory.NewGateway` for the order's PSP), call `gw.InitPayment(...)` with the order + its cart + customer, **persist the session on the order**, and return it.
3. PSP/gateway failure → return the error; the order is untouched and `/pay` is simply **retried** (no duplication — it operates on an existing order id).

So repeated `/pay` (client retry, or retry after a failed first attempt) yields **one** session, retried safely. The order's stored session is the idempotency anchor.

The **direct / card-on-file path** (no hosted checkout) does **not** use `/pay` — it goes straight to `CompleteOrder` with a payment method, unchanged.

---

## 4. Data model

Two new nullable columns on `orders` (Goose forward migration, both storage drivers):

```sql
ALTER TABLE "orders" ADD COLUMN "idempotency_key" TEXT;
ALTER TABLE "orders" ADD COLUMN "payment_session" JSONB;   -- stored InitPaymentResponse (+ enough to detect "live")
CREATE UNIQUE INDEX "orders_org_idempotency_key" ON "orders"("org_id","idempotency_key") WHERE "idempotency_key" IS NOT NULL;
```

- `idempotency_key` — opt-in dedup key; the partial unique index allows many NULLs (no key) but one row per `(org, key)`.
- `payment_session` — the gateway's `InitPaymentResponse` payload (opaque `any`), so `/pay` can return the existing session on replay. Whether a session is "live" is initially "present" (expiry-aware refresh is a refinement — §10).

`domain.Order` gains `IdempotencyKey string` and `PaymentSession any` (or a small typed wrapper). `order_row.go` in **both** `postgresgorm` and `postgrespgx` map the columns (nullable text via `*string`/`nilIfEmpty`; jsonb via the existing json column handling). `OrderRepository` gains `FindByIdempotencyKey(ctx, orgId, key)` and a way to persist `payment_session` (extend `Update` or an `Order` write that includes it).

---

## 5. Retry story (the whole point)

| Failure | Before | After |
| --- | --- | --- |
| Mid-write error (item/sub/reserve) | partial orphan order | whole tx rolls back — no order |
| Coupon refused | order + subs committed, error returned | tx rolls back — no order |
| Client retries `CreateOrder` (same key) | duplicate order | unique violation → returns the existing order |
| PSP session init fails | committed order, no retry path → re-call duplicates | order already exists; **retry `/pay`** → one session, no duplicate |
| Client retries `/pay` | n/a | returns the stored session (idempotent) |

---

## 6. Hexagonal placement

| Layer | Change |
| --- | --- |
| `core/domain` | `Order.IdempotencyKey`, `Order.PaymentSession`. |
| `core/port` | `CreateOrderInput.IdempotencyKey`; `OrderRepository.FindByIdempotencyKey`; persist payment_session. |
| `core/service` | `CreateOrder` → transactional + idempotent, no PSP; new `InitOrderPayment`. |
| `adapter/storage/{postgresgorm,postgrespgx}` | `order_row` columns + `FindByIdempotencyKey` + payment_session persistence (both drivers, conformance sub-test). |
| `adapter/http` | read `Idempotency-Key` header → `CreateOrderInput`; `CreateOrderResponse` drops `Psp`; new `POST /orders/{id}/pay` handler + Cedar action. |
| `config/server.go` | register the `pay` route. |
| `schemas/app/migrations` | `add orders.idempotency_key + payment_session + partial unique index`. |

No workflow-engine code changes (the order flow isn't engine-specific; `CompleteOrder`/billing are unchanged).

---

## 7. Error handling

- **Replay (unique violation) with a key** → return the existing order (200), not an error.
- **Unique violation without a key** → genuine conflict; surface as is (shouldn't happen — `idempotency_key` is the only added unique).
- **`/pay` on a non-pending order** → `ConflictError`.
- **`/pay` gateway failure** → return the error; order untouched; retryable.
- The pre-existing swallowed errors in `CreateOrder` (`pubsub.Publish` and the trailing `FindById`) are fixed in passing (log/return appropriately) since we're rewriting the flow.

---

## 8. Testing

- **service:** `CreateOrder` commits atomically — a forced mid-tx error (e.g. a refused coupon) leaves **no** order/cart/subscription/reservation (`FindById` not-found); a replay with the same `Idempotency-Key` returns the **same** order id and creates no second subscription; no key → two distinct orders. `InitOrderPayment` inits once, returns the stored session on a second call, and errors on a non-pending order.
- **adapter/storage (both drivers, `storagetest`):** `FindByIdempotencyKey` round-trip; the partial unique index rejects a duplicate `(org, key)` and allows multiple NULLs; `payment_session` round-trips.
- **adapter/http:** `Idempotency-Key` header threads through; same key on two `POST /orders` returns one order; `POST /orders/{id}/pay` returns a session and is idempotent.
- **e2e (integration):** create order (no PSP) → `/pay` → session; retry `/pay` → same session; a PSP-failure stub → retry succeeds with no duplicate order.

---

## 9. Non-goals

- **Payment-session expiry / refresh** — v1 returns the stored session if present; expiring/replacing a stale session (e.g. a `force` flag or TTL check) is a refinement.
- **Idempotency for `CompleteOrder`** and other mutations — this spec scopes the create + pay path only.
- **A general idempotency middleware** (storing arbitrary response bodies per key) — the order carries its own key; the webhook `IdempotencyKeyRepository` is unchanged and unrelated.
- **SDK / web / checkout updates** for the response change — downstream follow-up.

---

## 10. Decisions log

| Decision | Choice | Rationale |
| --- | --- | --- |
| Split PSP out of create | `CreateOrder` (DB-only) + `POST /orders/{id}/pay` (PSP) | Removes the dual-write; each step is independently retryable. |
| CreateOrder idempotency | `Idempotency-Key` → `orders.idempotency_key` + partial unique index | DB constraint is atomic dedup; replay returns the existing order; opt-in. |
| `/pay` idempotency | store the session on the order; return it if present | Retry-after-PSP-failure yields one session, no duplicate order. |
| Transaction boundary | writes + `Reserve` in one `RunInTx`; pubsub post-commit; PSP out of band | Atomic create (incl. coupon rollback); no network call in a tx. |
| Response shape | `CreateOrderResponse` drops `Psp` | The session now comes from `/pay`; breaking change accepted. |
| Direct/card-on-file | no `/pay` (goes to `CompleteOrder`) | `/pay` is for the hosted-checkout session only. |
