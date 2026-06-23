# Coupon Reservation & Order/Subscription Application — Design Spec

**Date:** 2026-06-23
**Status:** Settled — ready for implementation planning
**Area:** Billing — wiring coupons into the order → subscription → billing-cycle flow
**Builds on:** `docs/superpowers/specs/2026-06-15-coupons-design.md` (the coupon/code/discount
data model + `Validate`/`Redeem`/`ApplyDiscounts` building blocks).

---

## 1. Why this spec exists

The 2026-06-15 spec deliberately scoped **out** the consuming flows (§9 non-goals: "Any
downstream flow that consumes these methods — order-completion, billing-invoice — is separate
work"). As a result the building blocks exist but **nothing wires them up**:

- `CouponService.Redeem` has no caller and no route.
- The order/cart payload has no coupon field.
- `InvoiceService.BuildForBillingPeriod` never reads discounts, so every cycle bills full price.

This spec defines that consuming flow, with one extra concern the 2026-06-15 spec didn't
address: **reserving a coupon code's redemption capacity during checkout, before the discount
is committed.** A limited coupon (`max_redemptions`) must not be oversold while a customer is
mid-checkout, and the discount must only become real once payment succeeds.

---

## 2. Core model — three distinct concepts

| Concept | Role | Lifetime |
| --- | --- | --- |
| **Reservation** | An ephemeral **hold on a coupon code's redemption capacity** during one checkout. | Transient — deleted on convert/release; self-frees at `expires_at`. |
| **Discount** | The **applied record** of a redemption against a subscription/order; drives per-cycle discount math. | Permanent (the audit). Created **only on payment success**. |
| **Invoice line `DiscountTotal`** | The **money on the bill** — the discount actually charged for one cycle. | Per-invoice; can exist pre-payment. |

Two invariants, both load-bearing:

- **A `Discount` is only ever the applied record.** It is created on payment success and never
  exists in a "reserved/pending" limbo. (Reservation is a separate concept; it does not live on
  the Discount.)
- **The discount on the invoice is computed from the coupon math, not from the existence of a
  `Discount` row.** The math is deterministic on `(coupon terms, start_cycle, cycle)`, so it is
  identical whether resolved from a live reservation (pre-payment) or a committed Discount
  (post-payment). This lets a hosted checkout charge the discounted amount *before* the Discount
  record exists.

---

## 3. Data model — `coupon_reservations`

A new ephemeral table. No status column — **presence + `expires_at` encode the entire state.**

A coupon can be added before the customer or order exists (an anonymous checkout cart being
built up). So the holders are all **nullable** and filled in as they become known:
`checkout_session_id` (the top-level owner — sessions don't exist yet, so it's a forward-looking
nullable column), `order_id` (set when the order is created), `customer_id` (set at bind). A
reservation must be anchored to **at least one** of `checkout_session_id` / `order_id`.

```sql
CREATE TABLE coupon_reservations (
  org_id              TEXT NOT NULL,
  id                  TEXT NOT NULL,            -- "cres_" + KSUID
  coupon_id           TEXT NOT NULL,            -- capacity owner (always)
  coupon_code_id      TEXT,                     -- NULL = programmatic / code-less hold
  customer_id         TEXT,                     -- NULL until the customer is bound
  checkout_session_id TEXT,                     -- holder (future top-level entity)
  order_id            TEXT,                     -- holder, set when the order is created
  expires_at          TIMESTAMP(3) NOT NULL,    -- checkout-hold TTL
  created_at          TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

  CONSTRAINT coupon_reservations_pkey PRIMARY KEY (org_id, id),
  CONSTRAINT coupon_reservations_has_holder CHECK (checkout_session_id IS NOT NULL OR order_id IS NOT NULL),
  FOREIGN KEY (org_id, coupon_id) REFERENCES coupons(org_id, id) ON DELETE CASCADE
);
-- one live hold per coupon per holder (partial unique on whichever holder is set):
CREATE UNIQUE INDEX coupon_reservations_org_coupon_order_key   ON coupon_reservations(org_id, coupon_id, order_id)            WHERE order_id IS NOT NULL;
CREATE UNIQUE INDEX coupon_reservations_org_coupon_session_key ON coupon_reservations(org_id, coupon_id, checkout_session_id) WHERE checkout_session_id IS NOT NULL;
CREATE INDEX coupon_reservations_org_coupon_idx       ON coupon_reservations(org_id, coupon_id);
CREATE INDEX coupon_reservations_org_code_idx         ON coupon_reservations(org_id, coupon_code_id);
CREATE INDEX coupon_reservations_org_customer_cpn_idx ON coupon_reservations(org_id, customer_id, coupon_id);
CREATE INDEX coupon_reservations_expires_at_idx       ON coupon_reservations(expires_at);
```

Row states (no status field):
- **live** — row exists and `expires_at > now()` → holds a slot.
- **stale** — row exists but `expires_at <= now()` → already ignored by every count; not yet reclaimed.
- **gone** — converted (payment success) or released (failure/cancel) → row deleted.

**Goose:** a new forward migration `schemas/app/migrations/000NN_coupon_reservations.sql`
(Prisma is retired; migrations are the schema source of truth). No web/checkout schema mirroring
in scope.

---

## 4. Caps — counted in rows, never a stored counter

Every redemption cap counts **committed + live-reserved**, so a checkout in progress holds its
slot:

| Cap | Limit (`0` = ∞) | Counted |
| --- | --- | --- |
| Per-code | `coupon_code.max_redemptions` | `code.times_redeemed` + `count(reservations WHERE coupon_code_id = code AND expires_at > now)` |
| Coupon global | `coupon.max_redemptions` | `count(discounts WHERE coupon_id)` + `count(reservations WHERE coupon_id AND expires_at > now)` |
| Once-per-customer | `coupon.once_per_customer` | block if a `Discount` **or** live reservation exists for `(coupon_id, customer_id)` — *customer-scoped, see §4.2* |

Rules:
- **Coupon-global counts `Discount` rows**, not `times_redeemed` (it spans all codes +
  programmatic). `times_redeemed` is the committed per-code count.
- **Rows, not a counter** — release/convert just deletes the row; a counter would drift on a
  crash between row-delete and decrement, and could not lazily expire.
- **Lazy expiry** — the `expires_at > now()` predicate drops stale holds from every count
  automatically. Correctness never depends on the cleanup job.

### 4.1 Atomic reserve (closes today's TOCTOU race)

The existing gate counts-then-creates with a race window. Reserve fixes it:

```
BEGIN
  SELECT … FROM coupons      WHERE (org_id, id) = (…) FOR UPDATE;          -- lock capacity owner
  SELECT … FROM coupon_codes WHERE (org_id, id) = (…) FOR UPDATE;          -- when code-based
  -- run the full §5.3 gate, with cap counts = committed + live-reserved (above)
  INSERT INTO coupon_reservations (…);                                     -- only if every cap passes
COMMIT
```

The `FOR UPDATE` on the coupon (and code) row serializes concurrent reserves for the same
coupon/code, so the row counts are consistent at decision time. This is the whole point of
reservations: a correct, race-free hold.

### 4.2 Two-pass gate (the customer may be unknown at reserve)

A coupon can be added to an anonymous cart before the customer exists, so the gate runs in two
parts — each part runs at the point its inputs are available:

- **Capacity pass** (at reserve): coupon active, not expired, currency match, code active/not
  expired, **per-code cap**, **coupon-global cap**. None need a customer → runs on an anonymous
  checkout. This is what holds the slot.
- **Customer pass** (at bind, when the customer becomes known): `once_per_customer`,
  `first_time_transaction`, code customer-lock. Run the moment the customer is set — together
  with the capacity pass in the inline `CreateOrder` flow (customer known up front), or later in
  the session flow when a customer is attached.

Each pass is a pass/fail check; a failure is surfaced to the caller (the order/`addCoupon`/
`setCustomer` method fails). The capacity hold stays in place across the gap between the two
passes.

---

## 5. Reservation lifecycle

```
add coupon_code to order ──reserve──▶ [live reservation]
                                          │
        payment success ──convert──▶ delete reservation + create Discount + code.times_redeemed++
                                          │
   payment fail / cancel ──release──▶ delete reservation
                                          │
       no signal (abandon) ──expiry──▶ stops counting at expires_at; cleanup reclaims later
```

- **Reserve** — atomic gate + insert (§4.1). Runs the full two-layer §5.3 gate (cap counts now
  reservation-aware). `expires_at = now + holdWindow`.
- **Convert** — on payment success. **Unconditional and never re-gates caps** — the hold already
  secured the slot, and a paid customer is always honoured (even if the hold expired in a slow
  webhook; a rare ±1 cap overshoot favours the merchant). Deletes the reservation, creates the
  `Discount` (the applied record, `start_cycle` = the subscription's cycle at conversion, `0` for
  first-cycle redemption), increments `code.times_redeemed`. All in one tx; joins the caller's tx
  via ctx.
- **Release** — on payment failure or order cancellation: delete the reservation.
- **Expiry** — abandoned checkouts self-free at `expires_at` (lazy). A periodic
  `DELETE FROM coupon_reservations WHERE expires_at <= now()` is **housekeeping only**.

### 5.1 Hold TTL (`expires_at`)

The **checkout-hold window** — how long the slot is held for one in-flight order. A config knob
(default **30 minutes**, `COUPON_RESERVATION_TTL`), ideally aligned to the checkout-session
lifetime so the hold dies exactly when the checkout can no longer complete. Distinct from the
coupon's own redemption cutoffs (`coupon.RedeemBy`, `code.ExpiresAt`), which are calendar limits
checked in the gate, not holds.

---

## 6. Service surface (`CouponService`, narrow — no engine)

The existing `Validate` (preview, no writes) and `Redeem` (gate + create Discount immediately)
stay. New reservation-aware methods drive the order flow:

```go
// Reserve runs the capacity pass (§4.1/§4.2) and inserts a hold. CustomerId may be empty
// (anonymous); when set, the customer pass runs here too. Holder is the order (inline flow now)
// or the checkout session (later). A refusal aborts the order / fails the addCoupon method.
func (s *CouponService) Reserve(ctx, in ReserveInput) (domain.CouponReservation, error)

// BindCustomer runs the customer pass against an existing anonymous hold once the customer is
// known, and stamps customer_id on it. (Future session flow; in the inline flow the customer is
// already set at Reserve, so this is a no-op.)
func (s *CouponService) BindCustomer(ctx, in BindInput) error

// Consume converts a live (or expired — never re-gated) reservation into the applied Discount:
// deletes the reservation, creates the Discount (StartCycle + customer from the bound holder),
// increments code.times_redeemed. Called on payment success. Joins the caller's tx via ctx.
func (s *CouponService) Consume(ctx, in ConsumeInput) (domain.Discount, error)

// Release deletes a reservation (payment failure / order cancel). Idempotent.
func (s *CouponService) Release(ctx, orgId string, holder Holder) error

// PreviewForHolder computes the discount the reserved coupon would apply to the given lines at a
// cycle — used to build the hosted cycle-0 invoice from the reservation, pre-payment. Pure read.
func (s *CouponService) PreviewForHolder(ctx, orgId string, holder Holder, lines []domain.DiscountableLine, cycle int, currency string) (DiscountPreview, error)
```

```go
// Holder is the order (now) or the checkout session (later) — exactly one is set.
type Holder struct { OrderId, CheckoutSessionId string }

type ReserveInput struct { OrgId, Code, CouponId, CustomerId, Currency string; Holder Holder; Amount int64 } // CustomerId optional
type BindInput    struct { OrgId, CustomerId string; Holder Holder; Amount int64 }
type ConsumeInput struct { OrgId, SubscriptionId string; Holder Holder; StartCycle int }
```

New port: `CouponReservationRepository`

```go
type CouponReservationRepository interface {
    Create(ctx, domain.CouponReservation) (domain.CouponReservation, error)
    FindByHolder(ctx, orgId string, holder Holder) ([]domain.CouponReservation, error)
    BindCustomer(ctx, orgId string, holder Holder, customerId string) error // anonymous → known
    DeleteByHolder(ctx, orgId string, holder Holder) error
    CountLiveByCoupon(ctx, orgId, couponId string, now time.Time) (int, error)
    CountLiveByCode(ctx, orgId, couponCodeId string, now time.Time) (int, error)
    ExistsLiveForCustomer(ctx, orgId, couponId, customerId string, now time.Time) (bool, error)
    DeleteExpired(ctx, now time.Time) (int, error) // housekeeping
}
```

The gate's cap checks (§4) now consult `DiscountRepository.CountByCoupon` **plus**
`CouponReservationRepository.CountLive*`. The `FOR UPDATE` lock on the coupon/code row makes the
count-then-insert atomic.

---

## 7. Where it plugs into the order flow

### 7.1 Add — inline `CreateOrder` (what we build now)

- New optional field `coupon_code` on `CreateOrderRequest` → `port.CreateOrderInput.CouponCode`.
- Inside `OrderService.CreateOrder`'s existing tx, after the order row exists, call
  `CouponService.Reserve(...)` with the order as holder, the **customer (known up front in this
  flow)**, the cart currency, and the cart subtotal (for `MinimumAmount`). Because the customer
  is known, **both gate passes run together** here.
- **A reservation refusal fails the entire order** — the tx rolls back, no order/subscription is
  created, and the handler returns the refusal reason as an `ApiError`.

> The later **session flow** (`applyDiscount`/`setCustomer` cart methods, anonymous reserve →
> bind → create order from session) reuses the same `Reserve`/`BindCustomer`/`Consume`/`Release`
> with a session holder. It's out of scope to build now (§14) but the schema + service are
> shaped for it.

### 7.2 Convert — payment success

Both completion paths convert the reservation → `Discount`:

- **Synchronous** `OrderService.CompleteOrder`: inside its existing tx, after activating the
  subscription, call `Consume(Holder{OrderId}, subscriptionId, StartCycle: subscription's current
  cycle)`. The `Discount` commits in the same tx, *before* the post-commit
  `StartSubscriptionWorkflow`, so cycle-0 billing already sees it.
- **Webhook** `OrderWorkflowService.CompleteCheckoutSession`: same `Consume` call when the PSP
  reports success.

### 7.3 Release — failure / cancel

On a failed/cancelled order completion, call `Release(orgId, Holder{OrderId})`. Abandoned orders
that never resolve are handled by lazy expiry (no explicit release needed).

---

## 8. Cycle-0 unified into the billing/invoice flow

The legacy "caller supplies the first-payment amount at `CompleteOrder`" path is dropped. **Cycle
0 becomes a real billing cycle** so the discount applies to it uniformly. The invoice is the
single source of the charged amount in both checkout modes.

| | Server-charge (card-on-file) | Hosted checkout (PSP page) |
| --- | --- | --- |
| cycle-0 invoice built, discount applied | at `CompleteOrder` | at checkout-session creation (before redirect) |
| discount source for the invoice | committed `Discount` (converted just above, in the same tx) via `BuildForBillingPeriod` | the **reservation's coupon** via `PreviewForOrder` |
| who charges `invoice.Total` | server via the gateway (the existing billing runner) | the PSP on its page |
| on success | `Consume` (→ Discount) + settle invoice + record Payment | webhook → `Consume` (→ Discount) + settle invoice + record Payment |

**Server-charge mechanics:** `CompleteOrder` activates the subscription **due-now**
(`CyclesProcessed=0`, `RenewsAt=now`) and **no longer records a caller payment**. The existing
post-commit `StartSubscriptionWorkflow` sees `IsDueForBilling()==true` and runs the billing
runner, which charges cycle 0 through `BuildForBillingPeriod` like any other cycle. The `Discount`
was committed in the `CompleteOrder` tx, so the cycle-0 invoice applies it.

**Hosted mechanics:** at session creation the cycle-0 invoice is built with the discount from the
reservation (`PreviewForOrder`); the PSP charges `invoice.Total`; the success webhook records the
Payment against that invoice and `Consume`s the reservation into the `Discount`.

Both paths agree numerically because the discount math is deterministic on
`(coupon, start_cycle=0, cycle 0)`.

> The hosted-checkout *session plumbing* (build session → redirect → webhook) is its own surface;
> this spec specifies **where the discount fits** in it. The cycle-0 invoice-build + `Consume`
> hook are shared; the session/redirect machinery can land alongside or after.

---

## 9. Per-cycle discount application in billing (the other missing wire)

`InvoiceService.BuildForBillingPeriod` (shared `core/service`, called by **both** engines) gains
discount application, after building lines and before returning:

1. Load `DiscountRepository.ActiveForSubscription(orgId, sub.Id)` (status `active` only) and each
   discount's immutable `Coupon` → `[]domain.AppliedDiscount`.
2. Build `[]domain.DiscountableLine` from the invoice lines, resolving each line's `ProductId`
   (Price → Variant → Product; the order item already carries `product_id`).
3. `perLine := domain.ApplyDiscounts(lines, applied, inv.Cycle, sub.Currency)` (`inv.Cycle =
   sub.CyclesProcessed`).
4. Set each line's `DiscountTotal = perLine[lineId]`; `inv.recalculate()` (`Total = Subtotal −
   DiscountTotal`).

`ApplyDiscounts` already gates each discount to its window
(`StartCycle ≤ cycle < StartCycle + DurationInCycles`; `once`/`forever` special cases), so a
`repeating(2)` coupon redeemed at `start_cycle=0` discounts cycles 0 and 1 and nothing after.

`InvoiceService` gains a `DiscountRepository` + `CouponRepository` dependency (wired in
`app.go`). Build remains idempotent (looked up by `(orgId, subId, CyclesProcessed)`), so replay /
dunning retries re-derive the identical discount — no double-application.

---

## 10. Engine parity

Everything lands in `core/domain` (pure `ApplyDiscounts`, the reservation/discount aggregates)
and `core/service` (`CouponService`, `InvoiceService.BuildForBillingPeriod`). Both Hatchet and
Temporal charge through the same `BuildForBillingPeriod`, so they get identical discounts with no
per-adapter work. The duration window derives from the deterministic cycle index → stable under
Temporal replay and Hatchet retries. No workflow/step/activity changes are required beyond the
shared service calls already on the path.

The reservation cleanup (`DeleteExpired`) is housekeeping; it can be a small periodic job on
whichever scheduler is active (or piggyback the existing billing sweep). Because expiry is lazy,
its cadence is not correctness-critical and need not be identical across engines.

---

## 11. Hexagonal placement

| Layer | Additions |
| --- | --- |
| `core/domain` | `coupon_reservation.go` (aggregate + `NewCouponReservation`, `IsLive(now)`); reuse `discount_apply.go`, `Discount`. |
| `core/port` | `CouponReservationRepository`; extend `CouponService` inputs (`ReserveInput`, `ConsumeInput`); `CreateOrderInput.CouponCode`. |
| `core/service` | `CouponService.Reserve/BindCustomer/Consume/Release/PreviewForHolder`; two-pass reservation-aware gate; `InvoiceService` discount application; `OrderService` reserve/convert/release hooks. |
| `adapter/storage/{postgresgorm,postgrespgx}` | `coupon_reservation_row.go` + `_repo.go` in **both** drivers (the pgx/gorm parity from the storage work); reservation conformance sub-test in `storagetest`. |
| `adapter/http` | `coupon_code` on `CreateOrderRequest`; refusal → `ApiError`. |
| `config/app.go` | wire `CouponReservationRepository` → `CouponService`; `DiscountRepository`+`CouponRepository` → `InvoiceService`. |
| `schemas/app/migrations` | `000NN_coupon_reservations.sql`. |

---

## 12. Error handling

- **Reserve refusal** (cap/expiry/restriction): `CreateOrder` returns the §5.3 reason as an
  `ApiError` (e.g. `lib.ConflictError` for `cap_reached`/`code_cap_reached`/`already_used`,
  `lib.ValidationError` for `code_not_found`/`below_minimum`). The order is not created.
- **Consume after expiry** is allowed (unconditional convert).
- **Release is idempotent** (deleting a missing reservation is not an error).
- **Discount unique index** `(org_id, coupon_id, subscription_id)` still guards a double
  redemption of the same coupon on one subscription; `Consume` treats a unique-violation as
  already-consumed (idempotent under workflow retry).

---

## 13. Testing strategy

- **`domain`:** `CouponReservation.IsLive`; (reuse existing `ApplyDiscounts` window tests).
- **`service`:**
  - `Reserve` — cap matrix counting committed + live-reserved; refusal reasons; atomic gate.
  - `Consume` — creates Discount + increments `times_redeemed` + deletes reservation; idempotent;
    does **not** re-gate (succeeds even when the reservation is expired/absent on retry).
  - `Release` — idempotent delete.
  - `InvoiceService.BuildForBillingPeriod` — a `repeating(2)` discount yields discounted cycles 0
    and 1 and full price from cycle 2; idempotent rebuild.
- **`adapter/http`:** `CreateOrder` with a valid code reserves; with an exhausted/invalid code the
  **whole order fails** with the right `ApiError`.
- **Integration (`//go:build integration`, both drivers via `storagetest`):** reservation repo
  round-trips; concurrent reserve respects the cap under `FOR UPDATE`; lazy-expiry count excludes
  stale holds; `DeleteExpired` reclaims.
- **End-to-end (the live rig):** a $100/10-min sub with a 50%-off `repeating(2)` coupon →
  `2×$50 + 3×$100` over 5 cycles, sub `completed`, payments correct.

---

## 14. Non-goals

- **Multiple coupons per order / stacking at checkout** — one `coupon_code` per order for now
  (`ApplyDiscounts` already supports stacking for a later multi-coupon surface).
- **The checkout-session model** (top-level session → cart → `addProduct`/`changeCurrency`/
  `applyDiscount`/`setCustomer` methods → create-order-from-session) — doesn't exist yet. This
  spec builds the **inline `coupon_code` on `CreateOrder`** path now; the reservation schema +
  service are shaped so the session flow later reuses them (nullable holders, two-pass gate).
- **Full hosted-checkout session machinery** (session creation, redirect, PSP page) — this spec
  fixes *where the discount sits* in it; the session plumbing is its own work.
- **Mirroring coupon schema to web/checkout Prisma/TS** (per the 2026-06-15 spec).
- **Credit/carry-forward of an over-large flat amount** (unchanged from 2026-06-15).
- **Reservation in Redis/external store** — kept in the data model (Postgres) for transactional
  cap consistency.

---

## 15. Decisions log

| Decision | Choice | Rationale |
| --- | --- | --- |
| What is reserved | A **coupon-code redemption slot**, not a Discount | `Discount` = applied record only; reservation is capacity. |
| Reservation storage | Separate **ephemeral `coupon_reservations`** table | Keeps `Discount` pure; rows are hard-deleted on resolve. |
| Reservation status | **None** — presence + `expires_at` | Transient; lazy expiry; no state machine. |
| Cap counting | **Rows** (`committed + live-reserved`), not a stored counter | No drift; lazy expiry; cheap on a small table. |
| Reserve atomicity | **`FOR UPDATE` on coupon/code** then gate+insert | Closes the count-then-create TOCTOU race. |
| Convert (on success) | **Unconditional**, never re-gates caps | Honour paid customers; rare ±1 overshoot favours merchant. |
| Reserve failure at CreateOrder | **Fails the whole order** | A coupon that can't be held shouldn't create a half-order. |
| Customer unknown at reserve | **Holders all nullable** (`checkout_session_id`/`order_id`/`customer_id`); **two-pass gate** | A coupon can be added to an anonymous cart before the customer/order exist; capacity holds without a customer, customer checks run at bind. |
| Reservation holder | **Checkout session** (top-level), `order_id` filled later | The session owns the cart; for now it's a nullable column until sessions are built. |
| Cycle-0 charge | **Folded into the billing/invoice flow**; drop caller-supplied amount | Uniform, server-computed, discount-aware. |
| Discount on invoice vs Discount record | **Invoice carries the discount (can be pre-payment); `Discount` record is the post-success audit** | Lets a hosted PSP charge the discounted amount before the record exists; math is deterministic so they agree. |
| Hold TTL | Config (**30 min default**), aligned to checkout-session lifetime | The pre-payment hold window. |
| Engine parity | All logic in `core/{domain,service}`; both engines bill via `BuildForBillingPeriod` | No per-adapter discount code. |
