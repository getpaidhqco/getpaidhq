# Idempotent CreateOrder (via idempo) + separate `/orders/{id}/pay` — Design Spec

**Date:** 2026-06-23
**Status:** Settled — ready for implementation planning
**Area:** Orders — make creation idempotent using the `idempo` HTTP middleware backed by our own Postgres `Store`, and split payment-session init into its own retryable call.

---

## 1. Why

`OrderService.CreateOrder` today (read `internal/core/service/order.go:76`) is **not idempotent** and does an **external PSP call mid-flow**:

- `orderId` is freshly generated every call, so **re-calling `CreateOrder` with the same details creates a duplicate** order + customer + subscriptions + reservation. Nothing on the order path handles an `Idempotency-Key`.
- The PSP `InitPayment` call lives **inside** `CreateOrder` (`order.go:307`) and there is **no re-init endpoint**, so a committed order whose PSP init failed can only be "retried" by calling `CreateOrder` again → a duplicate.

This spec fixes both by separating two concerns and **not hand-rolling idempotency**:

- **`CreateOrder`** becomes an idempotent **write** — idempotency is provided by the [`idempo`](https://github.com/eben-vranken/idempo) HTTP middleware, backed by a Postgres `Store` **we** implement against **our** table and **our** storage adapters. No bespoke claim/replay/fencing logic of our own.
- **`POST /orders/{id}/pay`** initialises (or returns) the PSP payment session for an existing order. Independently retryable. No gateway call inside `CreateOrder`.

> Transactional atomicity of `CreateOrder`'s writes (wrapping them in `RunInTx`, and the gorm `RunInTx` ctx/SAVEPOINT fix) is a **separate, prerequisite** piece of work, tracked independently. This spec assumes it lands and does not respecify it.

---

## 2. Idempotency via `idempo` + our own Postgres `Store`

### 2.1 What `idempo` gives us (off the shelf)

`idempo` is framework-agnostic `net/http` middleware (`func(http.Handler) http.Handler`) implementing the IETF `Idempotency-Key` draft. It owns **all** the hard logic — request fingerprinting, fencing tokens, single-winner concurrency, response capture/replay, and the 409/422 status handling — and it has its own tests. We do **not** reimplement any of it.

Per request carrying an `Idempotency-Key` header (no header → it passes straight through):

1. It reads the body, computes `requestHash = sha256(method + "\n" + path + "\n" + body)`, and mints a fencing `token` (uuid).
2. It calls our `Store.Claim(ctx, key, requestHash, token)`:
   - **`StatusNew`** → it runs the handler (`CreateOrder`), captures the response, then calls our `Store.Complete(ctx, key, token, code, headers, body)`.
   - **`StatusCompleted`** (replay, same fingerprint) → it replays the stored `code`/`headers`/`body` + `Idempotency-Replayed: true`; **the handler never runs**.
   - **`StatusPending`** (a duplicate is mid-flight) → it returns **409**.
   - **`StatusConflict`** (same key, different body) → it returns **422**.
3. If the handler panics or returns **5xx**, idempo calls our `Store.Abandon(ctx, key, token)` → the claim is released → the key is retryable.

Because idempo stores and replays the **whole response**, a replayed `CreateOrder` returns the **exact original order JSON** with no lookup on our side — there is no `FindByIdempotencyKey`, no result column, no key→order mapping to maintain.

### 2.2 The dependency

Add `github.com/eben-vranken/idempo` as a **pinned module dependency** (MIT). We import it for the `Store` interface and `ClaimResult`/`ClaimStatus` types and the `New`/`Handler` middleware. We do **not** use its `pg`/`redis`/`inmem` backends — we supply our own `Store`. If the dependency ever goes stale, the middleware core is ~350 LOC and can be vendored later without touching our `Store` or table.

### 2.3 Our `Store` — Postgres, our adapters, hexagonal

idempo's `Store` is a third-party interface; to keep adapters implementing **our** ports (repo convention) and to keep the idempo import at the edge, we add a mirror port and a thin shim.

**Port** (`internal/core/port`):

```go
type IdempotencyClaimStatus string

const (
    IdempotencyNew       IdempotencyClaimStatus = "new"
    IdempotencyPending   IdempotencyClaimStatus = "pending"
    IdempotencyCompleted IdempotencyClaimStatus = "completed"
    IdempotencyConflict  IdempotencyClaimStatus = "conflict"
)

type IdempotencyClaim struct {
    Status  IdempotencyClaimStatus
    Code    int
    Headers []byte
    Body    []byte
}

// IdempotencyStore is the persistence behind the idempo middleware. Mirrors
// idempo.Store one-to-one so a tiny shim can adapt it without importing idempo
// into core. Claim must make exactly one concurrent caller win (StatusNew);
// Complete/Abandon must be no-ops on a token mismatch or non-pending row.
type IdempotencyStore interface {
    Claim(ctx context.Context, key, requestHash, token string) (IdempotencyClaim, error)
    Complete(ctx context.Context, key, token string, statusCode int, headers, body []byte) error
    Abandon(ctx context.Context, key, token string) error
}
```

The status string values are identical to idempo's (`"new"`, `"pending"`, `"completed"`, `"conflict"`) so the shim casts directly.

**Shim** (`internal/adapter/http/middleware`, where it may read auth ctx) — adapts our port to `idempo.Store` **and** scopes the key by org so two orgs can never collide:

```go
type idempoStore struct{ store port.IdempotencyStore }

func (a idempoStore) Claim(ctx context.Context, key, requestHash, token string) (idempo.ClaimResult, error) {
    c, err := a.store.Claim(ctx, scopeKey(ctx, key), requestHash, token)
    return idempo.ClaimResult{Status: idempo.ClaimStatus(c.Status), Code: c.Code, Headers: c.Headers, Body: c.Body}, err
}
func (a idempoStore) Complete(ctx context.Context, key, token string, code int, h, b []byte) error {
    return a.store.Complete(ctx, scopeKey(ctx, key), token, code, h, b)
}
func (a idempoStore) Abandon(ctx context.Context, key, token string) error {
    return a.store.Abandon(ctx, scopeKey(ctx, key), token)
}

// scopeKey prefixes the org so the stored key is "<orgId>:<clientKey>". The
// middleware runs AFTER authn (route-scoped middleware runs inside the global
// chain), so AuthUser is on ctx here.
func scopeKey(ctx context.Context, key string) string {
    if u, ok := middleware.AuthUserFrom(ctx); ok {
        return u.OrgId + ":" + key
    }
    return ":" + key
}
```

Because org is folded into the stored key, the storage adapter stays org-agnostic — it persists whatever key string it's handed. No `org_id` column, no cross-tenant leak.

**Adapters** (`postgresgorm`, `postgrespgx`) each implement `port.IdempotencyStore` against the table below. Logic mirrors idempo's reference `pg` store (read for reference only), adapted to our pool, our row-mapping, and Goose:

- `Claim`: delete any expired row for this key, then `INSERT ... ON CONFLICT (key) DO NOTHING`. Inserted ⇒ `StatusNew`. Conflict ⇒ `SELECT` the existing row: `state='pending'` ⇒ `StatusPending`; `state='completed'` and `request_hash` matches ⇒ `StatusCompleted` (with the stored response); hash differs ⇒ `StatusConflict`. The single conditional INSERT is the atomic single-winner gate.
- `Complete`: `UPDATE ... SET state='completed', response_*, expires_at = now()+retentionTTL WHERE key=$ AND token=$ AND state='pending'` (token-fenced, pending-only — a stale request can't overwrite a newer claim).
- `Abandon`: `DELETE WHERE key=$ AND token=$ AND state='pending'` (token-fenced).
- All three run on `dbFromCtx(ctx, db)` and honor ctx cancellation.

The two TTLs live on the store (constructor args): **lockTTL** (pending in-flight window; default `1m`, env `IDEMPOTENCY_LOCK_TTL`) and **retentionTTL** (how long a completed response is replayable; default `24h`, env `IDEMPOTENCY_RETENTION_TTL`). Expiry is lazy (swept on `Claim`), matching the existing webhook idempotency repo — no background goroutine.

### 2.4 Wiring

- `app.go`: build the `port.IdempotencyStore` from the selected `RepoSet` (gorm or pgx), then `idem := idempo.New(idempoStore{store}, idempo.Options{Logger: logger})`. Inject `idem.Handler` into `OrderHandler`.
- `OrderHandler.RegisterRoutes`: attach it to the order group so it runs **after** global authn (route/group middleware runs inside the global chain → `AuthUser` is on ctx for `scopeKey`):

```go
g := fuego.Group(s, "/orders",
    option.Tags("Orders"),
    option.Middleware(o.idem), // idempo, scoped to /orders; no-ops without the header
)
```

`idempo` no-ops when the header is absent, so group-level placement leaves the GETs untouched.

---

## 3. `POST /orders/{id}/pay` — payment-session init

A new route on the orders group: initialise the PSP payment session for an existing **pending** order and return it.

```
POST /api/orders/{id}/pay  →  { "psp": <InitPaymentResponse> }
```

`OrderService.InitOrderPayment(ctx, orgId, orderId, opts)`:
1. Load the order; require `status == pending` (else `ConflictError`).
2. **Idempotent on the order's session:** if the order already has a stored live payment session, return it. Otherwise resolve the gateway (`gatewayFactory.NewGateway` for the order's PSP), call `gw.InitPayment(...)` with the order + cart + customer, **persist the session on the order**, and return it.
3. PSP/gateway failure → return the error; the order is untouched; `/pay` is simply **retried**.

`/pay` is idempotent at the **domain** level (the stored session is the anchor), so it works even without an `Idempotency-Key`. The group's idempo middleware still applies if a client sends a key — the two layers coexist.

The **direct / card-on-file path** (no hosted checkout) does **not** use `/pay` — it goes straight to `CompleteOrder` with a payment method, unchanged.

### 3.1 Response change

`CreateOrder` no longer initialises a PSP session, so `domain.CreateOrderResponse` drops `Psp` (returns only `Order`). Callers needing a payment session now call `/pay`. This is a **breaking API change** for clients that read `resp.psp` (SDK / web / checkout, and the order tests that assert `resp.Psp`).

---

## 4. Data model

One new table in the app DB (Goose forward migration, mapped by **both** storage adapters). Purpose-built for the idempo `Store` — nothing legacy constrains it, and the existing `idempotency_keys`/webhook path is untouched.

```sql
CREATE TABLE "idempotency_requests" (
    "key"              TEXT        NOT NULL PRIMARY KEY,  -- "<orgId>:<clientKey>"
    "request_hash"     TEXT        NOT NULL,              -- idempo's method+path+body fingerprint
    "state"            TEXT        NOT NULL,              -- 'pending' | 'completed'
    "token"            TEXT        NOT NULL,              -- fencing token
    "response_code"    INTEGER,
    "response_headers" BYTEA,
    "response_body"    BYTEA,
    "expires_at"       TIMESTAMPTZ NOT NULL,
    "created_at"       TIMESTAMPTZ NOT NULL DEFAULT now(),
    "updated_at"       TIMESTAMPTZ NOT NULL
);
CREATE INDEX "idempotency_requests_expires_at" ON "idempotency_requests" ("expires_at");
```

- `key` is the org-scoped storage key (PK) — the single-winner gate is `INSERT ... ON CONFLICT (key) DO NOTHING`.
- `response_*` are NULL while `pending`, populated by `Complete`.
- `request_hash` lets `Claim` distinguish a true replay (`StatusCompleted`) from a key reused for a different body (`StatusConflict`).

`order_row.go` is unchanged for idempotency (the response is replayed by idempo, not reconstructed). For `/pay`, `orders` gains a nullable `payment_session JSONB` (the stored `InitPaymentResponse`), mapped in both adapters; `domain.Order` gains `PaymentSession any`, and `OrderRepository` gains a way to persist it (extend `Update` or a dedicated write).

---

## 5. Retry story (the whole point)

| Failure | Before | After |
| --- | --- | --- |
| Client retries `CreateOrder` (same key, same body) | duplicate order | idempo replays the stored response → same order |
| Concurrent duplicate `CreateOrder` (same key, in-flight) | two orders | one wins; the other gets **409** |
| Same key, different body | n/a | **422** (`StatusConflict`) |
| `CreateOrder` handler returns 5xx / panics | partial/duplicate | idempo `Abandon`s → key retryable |
| PSP session init fails | committed order, no retry path → re-call duplicates | order already exists; **retry `/pay`** → one session |
| Client retries `/pay` | n/a | returns the stored session (domain-idempotent) |

---

## 6. Hexagonal placement

| Layer | Change |
| --- | --- |
| `go.mod` | add `github.com/eben-vranken/idempo` (pinned). |
| `core/port` | `IdempotencyStore` + `IdempotencyClaim`/`IdempotencyClaimStatus`. For `/pay`: `OrderRepository` persists `payment_session`; `OrderService.InitOrderPayment`. |
| `core/domain` | `Order.PaymentSession any`. `CreateOrderResponse` drops `Psp`. |
| `core/service` | `CreateOrder` no longer calls the PSP (pure write); new `InitOrderPayment`. |
| `adapter/storage/{postgresgorm,postgrespgx}` | implement `port.IdempotencyStore` against `idempotency_requests`; map `orders.payment_session`; both in `RepoSet`, exercised by the shared `storagetest` conformance suite. |
| `adapter/http/middleware` | `idempoStore` shim (port→`idempo.Store`) + `scopeKey` org-prefixing. |
| `adapter/http` | `OrderHandler` takes the idempo `Handler`, attaches it to the order group; `CreateOrderResponse` drops `Psp`; new `POST /orders/{id}/pay` handler + Cedar action. |
| `config/app.go` | build the store, `idempo.New`, inject into `OrderHandler`. |
| `config/server.go` | (no global change — middleware is group-scoped via the handler.) |
| `schemas/app/migrations` | `create idempotency_requests`; `add orders.payment_session`. |
| `internal/lib/env.go` | `IDEMPOTENCY_LOCK_TTL`, `IDEMPOTENCY_RETENTION_TTL`. |

No workflow-engine code changes (the order flow isn't engine-specific).

---

## 7. Transactions

The idempo `Store` calls happen at the **HTTP-middleware layer on the bare request ctx — never inside a `RunInTx`** — so each adapter call resolves `dbFromCtx(ctx, db)` to the pool and runs as a **single autocommit statement**. This is required by idempo's model, not incidental:

- `Claim` must commit **immediately**, before the handler, so a concurrent duplicate sees the `pending` row (→ 409). It cannot be deferred into a later transaction.
- `Complete` runs **after the handler returns**, i.e. after the order's own transaction has already committed. It physically cannot share that tx.

`CreateOrder` opens its **own** `RunInTx` (the separate atomicity fix) for the order writes; that transaction is scoped to the service closure and fully resolved before control returns to idempo. The Store never nests inside it.

Happy path = **three sequential, independent commits**: `Claim` (pending) → order writes (`RunInTx`) → `Complete` (response). The window between the order commit and `Complete` is the accepted at-most-once gap: a crash there leaves the row `pending` until `lockTTL` (retries get 409), after which a retry may re-run. This is inherent to edge idempotency and is the deliberate trade-off for not threading the key through the domain transaction.

---

## 8. Testing

- **adapter/storage (both drivers, `storagetest` conformance):** `Claim` is single-winner under concurrency (N goroutines, exactly one `StatusNew`); a second `Claim` while `pending` returns `StatusPending`; after `Complete`, a same-hash `Claim` returns `StatusCompleted` with the exact stored `code`/`headers`/`body`; a different-hash `Claim` returns `StatusConflict`; `Complete`/`Abandon` are no-ops on a token mismatch and on a non-pending row; an expired `pending`/`completed` row is reclaimable as `StatusNew`.
- **adapter/http:** with the middleware mounted, two `POST /orders` with the same `Idempotency-Key` + body yield **one** order (second is a replay carrying `Idempotency-Replayed: true`); same key + different body → 422; missing header → normal create. `POST /orders/{id}/pay` returns a session and is idempotent (second call returns the stored session); `/pay` on a non-pending order → 409.
- **service:** `InitOrderPayment` inits once, returns the stored session on a second call, errors on a non-pending order. `CreateOrder` no longer returns `Psp`.
- **e2e (integration):** create order (no PSP) → `/pay` → session; retry `/pay` → same session; replay `CreateOrder` with the same key → same order, no second subscription.

We do **not** test idempo's internal logic (fingerprinting, fencing, replay, status codes) — that's the library's responsibility. We test only our `Store` (via conformance) and the wiring.

---

## 9. Non-goals

- **Transactional `CreateOrder` atomicity** — separate prerequisite work (the `RunInTx` wrap + gorm ctx/SAVEPOINT fix).
- **Closing the `Complete` window** — only domain-tx idempotency could, and we've chosen the middleware approach; the window is accepted.
- **Payment-session expiry / refresh** — `/pay` v1 returns the stored session if present; TTL/`force` refresh is a later refinement.
- **Idempotency for `CompleteOrder`** and other mutations — once the middleware exists, extending it to other routes is trivial but out of scope here.
- **Reusing/altering the webhook `idempotency_keys` table** — left untouched; different concern, different lifecycle.
- **SDK / web / checkout updates** for the response change — downstream follow-up.

---

## 10. Decisions log

| Decision | Choice | Rationale |
| --- | --- | --- |
| Idempotency mechanism | `idempo` HTTP middleware (off the shelf) | Don't hand-roll fingerprinting/fencing/replay/concurrency; the library owns and tests it. |
| idempo intake | pinned module dependency, not vendored | Least to maintain (user goal); MIT; can vendor the ~350-LOC core later if it stales. |
| Storage backend | our own Postgres `Store`, our table, both adapters | Use our DB/migrations/parity model — not idempo's `pg`/`redis` backends. |
| Hexagonal shape | `port.IdempotencyStore` + shim to `idempo.Store` | Adapters implement our port; idempo import stays at the http edge; decoupled if we swap libs. |
| Org scoping | shim prefixes `"<orgId>:<key>"` | No cross-tenant collision; storage adapter stays org-agnostic; no `org_id` column. |
| Middleware placement | group-scoped on `/orders` | Runs after global authn (org on ctx for scoping); no-ops without the header. |
| Split PSP out of create | `CreateOrder` (DB-only) + `POST /orders/{id}/pay` (PSP) | Removes the network call from creation; each step independently retryable. |
| `/pay` idempotency | store the session on the order; return it if present | Domain-level retry-safety even without a key. |
| Response shape | `CreateOrderResponse` drops `Psp` | The session now comes from `/pay`; breaking change accepted. |
| Transaction boundary | Store calls autocommit outside any `RunInTx` | Required by idempo's claim-before / complete-after model; documented at-most-once window. |
