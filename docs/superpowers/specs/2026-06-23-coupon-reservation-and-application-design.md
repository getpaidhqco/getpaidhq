# Coupon Reservation & Discount Application — Design Spec

**Date:** 2026-06-23
**Status:** Settled — ready for implementation planning
**Builds on:** `docs/superpowers/specs/2026-06-15-coupons-design.md` (the `Coupon`/`CouponCode`/
`Discount` model + `Validate`/`Redeem`/`ApplyDiscounts` building blocks).

---

## 1. Why this exists

The 2026-06-15 spec built the coupon model and math but scoped out the flows that consume them.
So today nothing reserves, redeems, or applies a coupon: `Redeem` has no caller, the order has no
coupon field, and no bill is built with a discount. This spec defines that consuming flow, plus
**reserving a coupon code's capacity during checkout** so a limited coupon can't be oversold
before payment.

---

## 2. The Order is a record; everything else is an effect

**An `Order` is the record that a customer ordered specific items at a point in time — the
intent/fact of a purchase.** That is all it is. Payment, subscriptions, invoices, a checkout
pay-link, discounts — none are part of what an Order *is*; they are **effects** of what was
ordered and how the order is configured.

We have **one Order entity** that produces different effects, rather than Stripe's separate
Invoice / Checkout Session / Payment Link / Subscription objects. The effects are driven by:

- **Contents** — subscription-priced lines produce subscriptions; one-time lines don't.
- **How the first payment is collected** (a flag):
  - `direct` — a payment is supplied (already processed) or a saved payment method is charged.
    No link. (Today's `CompleteOrder`-with-a-payment path; card-on-file.)
  - `checkout` — the order is `pending` with a hosted pay-link the customer pays; it **expires**
    if unpaid. A "payment link" is just this with a shared URL.
- **Invoice behaviour** (a flag):
  - a subscription **always** has invoices (the subscription owns them — §6);
  - a one-time order's invoice is created **up-front** (send-an-invoice) or **only after
    payment** (receipt).

> Important consequence: **a recurring subscription charge is NOT an order.** The order is the
> one-time record of what was bought; the subscription it produced owns its own recurring
> invoices/payments.

### 2.1 Checkout session vs order

A **checkout session** is the in-progress assembly of a purchase — a cart being built on a
checkout page (add/remove items, change currency, apply a coupon), **before any order exists**.
It is anonymous-capable and ephemeral (expires if abandoned). When the customer finalises, the
session becomes an **order** (the finalized record). The inline path skips the session and creates
the order directly.

So during checkout there may be **no order yet** — which is why a coupon hold cannot require an
order (§4).

### 2.1 Scope of this spec

- **Build now:** the **inline `coupon_code` on `CreateOrder`**, `direct` payment, applied to a
  subscription (the live-rig case: reserve at create, convert on first-payment success, discount
  on the subscription's first + recurring invoices).
- **Forward (schema/service shaped for it, not built now):** `checkout`/hosted pay-link payment,
  one-time-order invoices (`BuildForOrder`), and the order-configuration flags above.

---

## 3. Three concepts, kept separate

| Concept | What it is | Lifetime |
| --- | --- | --- |
| **Reservation** | An ephemeral **hold on a coupon code's redemption capacity** for one checkout (held by the session, or the order once finalized). | Transient — deleted on convert/release; self-frees at `expires_at`. |
| **Discount** | The **applied record** of a redemption against a subscription/order. | Permanent (the audit). Created **only on payment success**. |
| **Invoice line `DiscountTotal`** | The discount **money on a bill**. | Per invoice; may exist pre-payment. |

Two invariants:
- **A `Discount` is only ever the applied record** — created on payment success, never in a
  reserved/pending state.
- **The discount on a bill is computed from the coupon math, not from a `Discount` row existing.**
  `ApplyDiscounts` is deterministic on `(coupon, start_cycle, cycle)`, so the number is identical
  whether resolved from a live reservation (pre-payment) or a committed `Discount` (post-payment).
  This lets a hosted PSP charge the discounted amount before the `Discount` record exists.

---

## 4. Data model — `coupon_reservations`

Ephemeral. **No status column** — presence + `expires_at` encode the state. The holder is the
**checkout session** (during assembly) **or the order** (once finalized / inline) — both nullable,
at least one set; `order_id` is filled when a session becomes an order. `customer_id` is nullable
(anonymous checkout binds it later).

```sql
CREATE TABLE coupon_reservations (
  org_id              TEXT NOT NULL,
  id                  TEXT NOT NULL,            -- "cres_" + KSUID
  coupon_id           TEXT NOT NULL,            -- capacity owner (always)
  coupon_code_id      TEXT,                     -- NULL = programmatic / code-less hold
  customer_id         TEXT,                     -- NULL until bound
  checkout_session_id TEXT,                     -- holder while the order is being assembled
  order_id            TEXT,                     -- holder once finalized (filled when the session becomes an order)
  expires_at          TIMESTAMP(3) NOT NULL,    -- checkout-hold TTL
  created_at          TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

  CONSTRAINT coupon_reservations_pkey PRIMARY KEY (org_id, id),
  CONSTRAINT coupon_reservations_has_holder CHECK (checkout_session_id IS NOT NULL OR order_id IS NOT NULL),
  FOREIGN KEY (org_id, coupon_id) REFERENCES coupons(org_id, id) ON DELETE CASCADE
);
-- one live hold per coupon per holder:
CREATE UNIQUE INDEX coupon_reservations_org_coupon_order_key   ON coupon_reservations(org_id, coupon_id, order_id)            WHERE order_id IS NOT NULL;
CREATE UNIQUE INDEX coupon_reservations_org_coupon_session_key ON coupon_reservations(org_id, coupon_id, checkout_session_id) WHERE checkout_session_id IS NOT NULL;
CREATE INDEX coupon_reservations_org_coupon_idx ON coupon_reservations(org_id, coupon_id);
CREATE INDEX coupon_reservations_org_code_idx   ON coupon_reservations(org_id, coupon_code_id);
CREATE INDEX coupon_reservations_expires_idx    ON coupon_reservations(expires_at);
```

Goose forward migration `schemas/app/migrations/000NN_coupon_reservations.sql`. Built in **both**
storage drivers (`postgresgorm`, `postgrespgx`) per the existing parity rule.

Row states: **live** (`expires_at > now()`), **stale** (expired, ignored by counts, not yet
reclaimed), **gone** (deleted on convert/release).

---

## 5. Caps — counted in rows, atomic, lazy-expiring

Every cap counts **committed Discounts + live reservations**:

| Cap | Limit (`0`=∞) | Counts |
| --- | --- | --- |
| Per-code | `coupon_code.max_redemptions` | `code.times_redeemed` + live reservations for that code |
| Coupon global | `coupon.max_redemptions` | `count(discounts WHERE coupon_id)` + live reservations for the coupon |
| Once-per-customer | `coupon.once_per_customer` | a `Discount` **or** live reservation exists for `(coupon, customer)` — *customer-scoped, §5.2* |

- **Rows, not a stored counter** — release/convert deletes the row (no drift); `expires_at > now()`
  drops stale holds from every count (lazy expiry; cleanup is housekeeping only).

### 5.1 Atomic reserve (closes the existing TOCTOU race)

`SELECT … FOR UPDATE` the `coupons` (and `coupon_codes`, when code-based) row → run the gate with
the counts above → insert the reservation only if every cap passes — all in one tx. The lock
serializes concurrent reserves per coupon/code.

### 5.2 Two-pass gate (customer may be unknown at reserve)

A coupon can be added to an anonymous order before the customer is known, so the gate runs in two
parts, each where its inputs exist:
- **Capacity pass (at reserve):** coupon/code active + not expired, currency, per-code + global
  caps. No customer needed → holds the slot.
- **Customer pass (at bind):** `once_per_customer`, `first_time`, code customer-lock. Runs when
  the customer is set — together with the capacity pass in the inline flow (customer known up
  front), later in the checkout flow.

---

## 6. Bills — who owns which invoice

- **Subscription owns its invoices, including the first.** The first invoice is the subscription's
  cycle-0 invoice, built when the order's first payment is collected; the recurring engine builds
  cycles 1+ via `InvoiceService.BuildForBillingPeriod`. (Matches Stripe: creating the subscription
  creates + pays its first invoice now, then recurring continues.)
- **A one-time order's invoice** is built by `InvoiceService.BuildForOrder` (forward) per the
  order's invoice flag — up-front (`open`, for send-an-invoice) or after payment (the record).

When the first invoice is materialized depends on the payment flag:
- `direct` — built at payment time (server charges it / records the supplied payment).
- `checkout` — built **open before** the customer pays, so the hosted PSP charges that bill.

`ChargeForBillingPeriod` stays **recurring-only**; it is not the first-payment path.

---

## 7. Discount application — one rule, at every bill build

**Whenever a bill is built — a subscription cycle invoice, or a one-time order invoice — the
coupon's discount is computed on that bill's lines and subtracted, so the total it produces *is*
the discounted amount charged.** It is part of building the bill, not a later step. Payment
success writes the `Discount` record; it recalculates nothing.

In the build (shared `core/service`, both engines):
1. Resolve the applicable discounts → `[]domain.AppliedDiscount` — from the committed
   `DiscountRepository.Active*` (post-payment, and recurring cycles), or from the live
   **reservation's** coupon when building a bill pre-payment (`checkout`).
2. Build `[]domain.DiscountableLine` from the lines (resolve each `ProductId` via
   Price→Variant→Product / the order item).
3. `perLine := domain.ApplyDiscounts(lines, applied, cycle, currency)` (`cycle =
   sub.CyclesProcessed`; for a one-time order, order-targeting per `Discount.OrderId`).
4. Set each line's `DiscountTotal`; `inv.recalculate()` (`Total = Subtotal − DiscountTotal`).

`ApplyDiscounts` gates each discount to its window (`StartCycle ≤ cycle < StartCycle +
DurationInCycles`; `once`/`forever` cases), so a `repeating(2)` coupon at `start_cycle=0`
discounts cycles 0 and 1, full price after. Build stays idempotent (keyed
`(orgId, subId, CyclesProcessed)`), so replay/dunning re-derive the identical discount.

---

## 8. Lifecycle — reserve → convert / release

```
add coupon (checkout session, or inline order) ──reserve──▶ [live reservation, held by session or order]
                       session finalised ──attachOrder──▶ stamp order_id on the hold
                  first payment succeeds ──convert──▶ delete reservation + create Discount + code.times_redeemed++
       payment fails / session|order cancelled ──release──▶ delete reservation
                      abandoned/expired ──────▶ reservation lapses at expires_at (lazy); cleanup reclaims
```

- **Reserve** — atomic capacity gate (§5.1) + insert. `expires_at = now + holdTTL` (config
  default **30 min**, aligned to a `checkout` order's expiry). Customer pass runs here if the
  customer is known.
- **Convert** — on **first-payment success**. **Unconditional, never re-gates caps** (the hold
  secured the slot; a paid customer is always honoured even if the hold expired). Deletes the
  reservation, creates the `Discount` (`start_cycle` = the subscription's cycle at conversion, or
  the order target), increments `code.times_redeemed`. One tx; joins the caller's tx.
- **Release** — on payment failure / order cancel: delete the reservation. Idempotent.

`coupon.RedeemBy` / `code.ExpiresAt` are redemption cutoffs checked in the gate — distinct from
the reservation hold.

---

## 9. Service surface (`CouponService`, narrow — no engine)

`Validate` / `Redeem` stay. New reservation methods:

```go
// Holder is the checkout session (pre-order) OR the order (finalized) — exactly one set.
type Holder struct { CheckoutSessionId, OrderId string }

func (s *CouponService) Reserve(ctx, in ReserveInput) (domain.CouponReservation, error) // capacity (+customer if known); atomic
func (s *CouponService) BindCustomer(ctx, in BindInput) error                            // customer pass once known (checkout flow)
func (s *CouponService) AttachOrder(ctx, orgId string, session, order string) error      // session → order: stamp order_id on the hold
func (s *CouponService) Consume(ctx, in ConsumeInput) (domain.Discount, error)           // on payment success; never re-gates
func (s *CouponService) Release(ctx, orgId string, holder Holder) error                  // idempotent
func (s *CouponService) PreviewForHolder(ctx, orgId string, holder Holder, lines []domain.DiscountableLine, cycle int, currency string) (DiscountPreview, error) // pre-payment bill

type ReserveInput struct { OrgId, Code, CouponId, CustomerId, Currency string; Holder Holder; Amount int64 } // CustomerId optional
type BindInput    struct { OrgId, CustomerId string; Holder Holder; Amount int64 }
type ConsumeInput struct { OrgId, SubscriptionId string; Holder Holder; StartCycle int }
```

```go
type CouponReservationRepository interface {
    Create(ctx, domain.CouponReservation) (domain.CouponReservation, error)
    FindByHolder(ctx, orgId string, holder Holder) ([]domain.CouponReservation, error)
    BindCustomer(ctx, orgId string, holder Holder, customerId string) error
    AttachOrder(ctx, orgId, checkoutSessionId, orderId string) error // session hold → order
    DeleteByHolder(ctx, orgId string, holder Holder) error
    CountLiveByCoupon(ctx, orgId, couponId string, now time.Time) (int, error)
    CountLiveByCode(ctx, orgId, couponCodeId string, now time.Time) (int, error)
    ExistsLiveForCustomer(ctx, orgId, couponId, customerId string, now time.Time) (bool, error)
    DeleteExpired(ctx, now time.Time) (int, error) // housekeeping
}
```

The gate's cap checks consult `DiscountRepository.CountByCoupon` **plus**
`CouponReservationRepository.CountLive*`, under the `FOR UPDATE` lock.

---

## 10. Order-flow hooks

- **Reserve:**
  - *Inline (build now)* — `coupon_code` on `CreateOrderRequest`. Inside `CreateOrder`'s tx,
    reserve with the **order** as holder; customer is known → both gate passes run. **A refusal
    fails the whole order** (tx rolls back, `ApiError`).
  - *Checkout page (forward)* — the `applyDiscount` cart method reserves with the **session** as
    holder (customer maybe unknown → capacity pass only; `BindCustomer` later). On finalise,
    `AttachOrder(session → order)` stamps `order_id`. Same `Reserve`/`Consume`/`Release`.
- **Convert — first-payment success:** `direct` → in `CompleteOrder`'s tx, after activating the
  subscription, `Consume(Holder{OrderId})` (commits the `Discount` before cycle-0 billing reads
  it). `checkout` → the pay webhook does the same.
- **Release — failure/cancel:** `Release(Holder)`; abandoned sessions/orders lapse via expiry.

---

## 11. Engine parity

All logic is in `core/domain` (`ApplyDiscounts`, the reservation/discount aggregates) and
`core/service` (`CouponService`, the invoice builds). Both Hatchet and Temporal build bills
through the same service, so discounts are identical with no per-adapter code; the cycle window is
derived from the deterministic cycle index → stable under replay/retry. Reservation cleanup
(`DeleteExpired`) is housekeeping on whichever scheduler is active; lazy expiry means its cadence
is not correctness-critical.

---

## 12. Hexagonal placement

| Layer | Additions |
| --- | --- |
| `core/domain` | `coupon_reservation.go` (`NewCouponReservation`, `IsLive(now)`); reuse `discount_apply.go`, `Discount`. |
| `core/port` | `CouponReservationRepository`; `CouponService` reserve inputs; `CreateOrderInput.CouponCode`. |
| `core/service` | `CouponService.Reserve/BindCustomer/AttachOrder/Consume/Release/PreviewForHolder`; two-pass gate; discount application in `InvoiceService` builds; `OrderService` hooks. |
| `adapter/storage/{postgresgorm,postgrespgx}` | `coupon_reservation_row.go` + `_repo.go` (both drivers); reservation conformance sub-test in `storagetest`. |
| `adapter/http` | `coupon_code` on `CreateOrderRequest`; refusal → `ApiError`. |
| `config/app.go` | wire `CouponReservationRepository` → `CouponService`; `DiscountRepository`+`CouponRepository` → `InvoiceService`. |
| `schemas/app/migrations` | `000NN_coupon_reservations.sql`. |

---

## 13. Error handling

- **Reserve refusal** → `CreateOrder` returns the gate reason as `ApiError`
  (`ConflictError` for caps/`already_used`, `ValidationError` for `code_not_found`/`below_minimum`);
  no order created.
- **Consume** after expiry is allowed (unconditional). Treats the `(org,coupon,subscription)`
  unique-violation as already-consumed (idempotent under retry).
- **Release** is idempotent.

---

## 14. Testing

- **domain:** `CouponReservation.IsLive`; reuse `ApplyDiscounts` window tests.
- **service:** `Reserve` cap matrix (committed + live-reserved; refusal reasons; atomic);
  `Consume` creates Discount + increments + deletes, idempotent, never re-gates; `Release`
  idempotent; the invoice build discounts cycles 0–1 and full price from cycle 2 for a
  `repeating(2)`; idempotent rebuild.
- **adapter/http:** `CreateOrder` with a valid code reserves; an exhausted/invalid code fails the
  whole order with the right `ApiError`.
- **integration (both drivers, `storagetest`):** reservation round-trips; concurrent reserve
  respects the cap under `FOR UPDATE`; lazy-expiry count excludes stale; `DeleteExpired` reclaims.
- **e2e (live rig):** $100/10-min sub + 50%-off `repeating(2)` → **`2×$50 + 3×$100`** over 5
  cycles, sub `completed`, payments correct.

---

## 15. Non-goals

- **`checkout`/hosted pay-link plumbing** (the hosted page, redirect, expiry sweep) and
  **one-time-order invoices (`BuildForOrder`)** — the model + schema support them; not built now.
- **The order-configuration flags machinery** (payment mode, invoice behaviour) beyond what the
  inline subscription path needs.
- **Multiple coupons per order / stacking at checkout** — one `coupon_code` per order
  (`ApplyDiscounts` already stacks for later).
- **Refunds of discounted payments**; **carry-forward of an over-large flat amount**;
  **web/checkout schema mirroring**; **reservation in Redis** (kept in Postgres for cap
  consistency).

---

## 16. Open items

- **One order producing multiple subscriptions** + one coupon — `Consume` makes one `Discount` per
  `subscription_id`; need the rule for which sub(s) get it (likely one `Discount` per sub whose
  lines the coupon targets).

---

## 17. Decisions log

| Decision | Choice | Rationale |
| --- | --- | --- |
| Order definition | **A record of what was ordered, when** — payment/subscription/invoice/discount are effects | Keeps the entity neutral; avoids baking payment assumptions into it. |
| One entity vs many | **One `Order`** configured by flags, not Stripe's separate Invoice/Session/Link/Subscription | Simpler to reason about; effects derive from contents + flags. |
| Recurring charge | **Not an order** — the subscription owns its recurring invoices/payments | An order is the one-time purchase record. |
| What is reserved | A **coupon-code slot**, not a Discount | `Discount` = applied record only. |
| Reservation storage | Ephemeral `coupon_reservations`, **no status**, held by the **checkout session or order** (nullable holders) | Transient; lazy expiry; a checkout page has no order yet, so the hold can't require one. |
| Cap counting | **Rows** (`committed + live-reserved`); atomic reserve under `FOR UPDATE` | No drift; lazy expiry; closes the TOCTOU race. |
| Customer unknown at reserve | `customer_id` nullable; **two-pass gate** | Anonymous carts: capacity holds without a customer; customer checks at bind. |
| Convert | **Unconditional**, on first-payment success | Honour paid customers; the hold already guarded the window. |
| Reserve failure | **Fails the whole order** | A coupon that can't be held shouldn't half-create an order. |
| Discount on bill vs record | **Bill carries the discount (pre-payment ok); `Discount` is the post-success record** | Hosted PSP charges the discounted amount before the record exists; math is deterministic. |
| First bill ownership | **Subscription owns its first invoice** (built at activation, paid now); recurring after; one-time → `BuildForOrder` | Matches Stripe; `ChargeForBillingPeriod` stays recurring-only. |
| Engine parity | All logic in `core/{domain,service}` | No per-adapter discount code. |
