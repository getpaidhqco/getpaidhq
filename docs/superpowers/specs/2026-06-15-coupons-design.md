# Coupons & Discounts — Design Spec

**Date:** 2026-06-15
**Status:** Draft for review
**Area:** Billing — subscription & one-time order discounting
**Engines affected:** Both (Hatchet + Temporal) — by construction, see [Engine parity](#engine-parity)

---

## 1. Summary

Add merchant-defined **coupons** that customers redeem (by code or programmatically) to
receive **discounts** on subscription invoices and one-time orders. The model follows
Stripe's separation of concerns:

| Aggregate    | Role                                                                                  |
| ------------ | ------------------------------------------------------------------------------------- |
| `Coupon`     | The **definition** — discount rules and type. Holds no code.                          |
| `CouponCode` | A **redeemable code**, N per coupon (1‑N). Defines the string a customer types.       |
| `Discount`   | The **applied instance** — recorded against a subscription/order, shown on invoices.  |

A `Coupon` is *only* the definition. A `CouponCode` is *only* the redeemable string. A
`Discount` is the redeemed coupon, snapshotted onto a specific subscription or order, and is
what actually reduces an invoice. This keeps "what the merchant configured", "how a customer
enters it", and "what was applied to this customer's bill" as three independent things.

Discounts come off **before tax**, allocated across matching invoice lines.

---

## 2. Goals (traceability to user stories)

| # | User story / use case                                                            | Where satisfied                                  |
| - | -------------------------------------------------------------------------------- | ------------------------------------------------ |
| D1 | Create a coupon as flat amount **or** percentage off                            | `Coupon.DiscountType` + DB mutual-exclusion check |
| D2 | Set duration: one payment, N payments, or forever                               | `Coupon.Duration` + `DurationInCycles`           |
| D3 | Set a redeem-by calendar date                                                   | `Coupon.RedeemBy`                                |
| D4 | Limit a coupon to specific plans/charges                                         | `Coupon.AppliesToProducts` (Product IDs)         |
| D5 | Cap total redemptions across all customers                                      | `Coupon.MaxRedemptions`                          |
| D6 | Reusable, or one-time per customer                                              | `Coupon.OncePerCustomer`                         |
| R1 | Enter a code at checkout and see the discount before confirming                 | `POST /api/coupons:preview`                       |
| R2 | Discount attaches to the subscription and counts from its start                 | `Discount.SubscriptionId` + `StartCycle`          |
| A1 | Discount comes off each bill before tax, for the duration, then stops           | `InvoiceService.BuildForBillingPeriod` + cycle math |
| A2 | Targeted coupon discounts only matching plan/charge; rest bills full            | `domain.ApplyDiscounts` per-line allocation       |
| L1 | Refused when expired, global cap hit, or already used (one-time)                | Redemption validation                            |
| L2 | Cancel mid-discount → remaining discount simply ends                            | Subscription stops billing; discount goes `completed` |

**Dropped (non-goal, per decision):** flat amount larger than the invoice carrying forward to
the next invoice. See [Non-goals](#9-non-goals-v1).

---

## 3. Domain model

All IDs are `<prefix>_` + KSUID (matching `domain.GenerateId`). Money is `int64` minor units
(cents); currency is ISO‑4217. Percentages are `decimal.Decimal`.

### 3.1 `Coupon` — the definition (`internal/core/domain/coupon.go`)

```go
type Coupon struct {
    OrgId   string          // tenant shard key
    Id      string          // "coup_" + KSUID
    Name    string

    DiscountType DiscountType // percentage | fixed

    // Exactly one of the two groups is set (enforced in ctor + DB CHECK):
    PercentOff   decimal.Decimal // when percentage, 0 < p <= 100
    AmountOff    int64           // when fixed, > 0, minor units
    Currency     string          // when fixed, ISO-4217; must equal invoice currency at apply time

    Duration         Duration // once | repeating | forever
    DurationInCycles int      // set iff Duration == repeating, >= 1; nil/0 otherwise

    RedeemBy time.Time // nullable; last instant the coupon may be redeemed

    AppliesToProducts []string // Product IDs; empty = applies to the whole bill

    MaxRedemptions  int  // 0 = unlimited global cap
    OncePerCustomer bool // false = reusable by a customer

    Active    bool
    CreatedAt time.Time
    UpdatedAt time.Time
}

type DiscountType string // "percentage" | "fixed"
type Duration     string // "once" | "repeating" | "forever"
```

The constructor (`NewCoupon`) enforces the mutual exclusion (percentage vs fixed), the
`repeating ⇒ DurationInCycles >= 1` rule, and `percent_off` range — returning an `ApiError`
before the row ever reaches Postgres. The DB `CHECK` constraints are the backstop (§7.2).

### 3.2 `CouponCode` — the redeemable code (`internal/core/domain/coupon_code.go`)

```go
type CouponCode struct {
    OrgId     string
    Id        string // "ccode_" + KSUID
    CouponId  string
    Code      string // unique per org, matched case-insensitively (stored upper-cased)
    Active    bool
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

A coupon may have **zero** codes (applied programmatically/automatically via the redeem API
referencing the coupon directly) or **many**. Redemption-by-code resolves a `CouponCode` →
`Coupon`. Per-code caps/expiry are intentionally **not** modelled in v1 — limits live on the
`Coupon` (§9).

### 3.3 `Discount` — the applied instance (`internal/core/domain/discount.go`)

```go
type Discount struct {
    OrgId        string
    Id           string // "disc_" + KSUID
    CouponId     string // provenance (display/audit)
    CouponCodeId string // nullable; empty when redeemed programmatically
    CustomerId   string

    // Exactly one target (enforced in ctor + DB CHECK):
    SubscriptionId string // recurring discount
    OrderId        string // one-time order discount

    // --- snapshot of the coupon's rules at redemption time ---
    DiscountType      DiscountType
    PercentOff        decimal.Decimal
    AmountOff         int64
    Currency          string
    Duration          Duration
    DurationInCycles  int
    AppliesToProducts []string
    // ---------------------------------------------------------

    StartCycle int       // subscription.CyclesProcessed at redemption (0 for orders)
    Status     DiscountStatus // active | completed | cancelled
    RedeemedAt time.Time
    EndedAt    time.Time // nullable

    CreatedAt time.Time
    UpdatedAt time.Time
}

type DiscountStatus string // "active" | "completed" | "cancelled"
```

**The snapshot is load-bearing.** Once redeemed, a `Discount` carries its own copy of the
coupon rules, so editing or deleting the `Coupon` afterwards never changes a live discount's
behaviour. This is also what makes application a pure function of the `Discount` + the invoice.

### 3.4 Invoice changes (`internal/core/domain/invoice.go`, `invoice_line_item.go`)

Add discount fields to make the discount visible on the invoice and keep totals honest:

```go
// InvoiceLineItem
DiscountTotal int64 // >= 0, <= Total; amount discounted from this line (minor units)

// Invoice
DiscountTotal int64 // sum of line DiscountTotal
```

`Invoice.recalculate()` becomes:

```
Subtotal      = Σ line.Total                 // gross, unchanged
DiscountTotal = Σ line.DiscountTotal
Total         = Subtotal − DiscountTotal     // tax, when implemented, applies on Total here
```

One-time orders reuse the **already-present** `OrderItem.DiscountTotal` column (currently
unused in recurring billing).

---

## 4. Discount application (the load-bearing seam)

### 4.1 Pure domain function

`internal/core/domain/discount_apply.go`:

```go
// DiscountableLine is the minimal view ApplyDiscounts needs — resolved by the caller.
type DiscountableLine struct {
    LineId    string
    ProductId string // resolved Price → Variant → Product
    Total     int64  // gross line total, minor units
}

// ApplyDiscounts returns the discount amount to record per line id.
// Pure, deterministic, side-effect free.
func ApplyDiscounts(lines []DiscountableLine, discounts []Discount, currency string) map[string]int64
```

Algorithm:

1. Order `discounts` by `RedeemedAt` ascending (stable; deterministic for stacking).
2. Track a per-line **running net**, initialised to `line.Total`.
3. For each discount, in order:
   - **Base** = sum of running nets of lines that match the discount's scope
     (`AppliesToProducts` contains `line.ProductId`; empty scope = all lines).
   - **Raw discount** = `percentage`: `round(base × percentOff/100)`; `fixed`:
     `min(amountOff, base)` (leftover is **not** carried — see Non-goals).
   - **Allocate** the raw discount back across the matching lines in proportion to each line's
     running net (largest-remainder rounding so the parts sum exactly to the raw discount).
   - Subtract each line's allocated part from its running net; accumulate into the line's
     recorded `DiscountTotal`.
4. **Clamp invariant:** a line's cumulative `DiscountTotal` can never exceed its `Total`
   (running net floored at 0). This holds automatically because each step works off the
   running net.

Stacking is therefore well-defined: "fixed then percentage" vs "percentage then fixed" differ
only by redemption order, and no line can be discounted below zero.

### 4.2 Where it's invoked — subscriptions

`InvoiceService.BuildForBillingPeriod` (`internal/core/service/invoice.go`), **after** lines
are built and **before** `recalculate()`:

1. Build gross lines (today's behaviour).
2. Resolve `ProductId` for each line (Price → Variant → Product) via the price repo.
3. Load **active** discounts for the subscription whose cycle window covers this invoice
   (`StartCycle ≤ invoice.Cycle < StartCycle + DurationInCycles`; `once` ⇒ window of 1;
   `forever` ⇒ unbounded) via the injected `DiscountReader` (§6).
4. `domain.ApplyDiscounts(...)` → write each `line.DiscountTotal`.
5. `recalculate()` → `Total = Subtotal − DiscountTotal`.

Because this lives entirely in `core/`, both engines charge the discounted `Invoice.Total`
with **zero adapter code**.

### 4.3 Where it's invoked — one-time orders

The same pure function runs once at checkout/order completion (`OrderService.CompleteOrder`
path), writing `OrderItem.DiscountTotal` and the order totals. Duration is irrelevant for a
single charge (`once` semantics).

### 4.4 Duration is derived, not counted

A discount applies to cycle *N* iff `StartCycle ≤ N < StartCycle + DurationInCycles`
(`once` = 1, `forever` = unbounded). **No mutable counter.**

- **Dunning-idempotent:** the same cycle's invoice can be rebuilt/retried any number of times
  and yields the identical discount — there is no per-application decrement to double-count.
- **Parity-safe:** a value derived from the (deterministic) cycle index is stable under
  Temporal replay/`ContinueAsNew`; a stored counter would be extra state both engines must
  keep in lockstep.

**Known semantic:** "N payments" is realised as "N billing **cycles**". This equals N paid
invoices when every covered cycle is paid. Pauses are handled correctly (`CyclesProcessed`
does not advance while paused, so the discount waits). If a covered cycle is shown on an
ultimately-unpaid invoice, it still counted against the duration — accepted for v1. Switching
to "N strictly-successful payments" would require a post-success counter with an idempotency
guard; explicitly out of scope.

A discount is lazily marked `completed` (status + `EndedAt`) once its window has passed, or on
subscription cancellation. Marking is for query/display only — application correctness comes
from the cycle math, not the status.

---

## 5. Redemption

### 5.1 Preview (checkout, no writes)

`POST /api/coupons:preview`

```
Request:  { code: string, orderId?: string, lines?: [{ priceId, quantity }] }
Response: { valid: bool, reason?: string, discountTotal: int64, perLine: [{ priceId, discount }] }
```

Resolves the code (org-scoped, case-insensitive) → coupon, runs the §5.3 validation, and
computes the discount against the prospective order lines using the same `ApplyDiscounts`
pure function. Returns the amount the customer will see **before** they confirm.

### 5.2 Redeem (on confirm)

Creates a `Discount` that **snapshots** the coupon, links it to the subscription (or order),
and sets `StartCycle = subscription.CyclesProcessed` (0 for orders). Redemption counts are
derived from `Discount` rows — there is no separate redemption ledger.

For subscriptions, redemption typically happens as part of the order/subscription creation
flow; the discount is then visible on the first and subsequent in-window invoices.

### 5.3 Validation / refusal rules (`CouponService.validate`)

A redemption (and preview) is refused, with a specific reason, when:

| Reason             | Check                                                                 |
| ------------------ | --------------------------------------------------------------------- |
| `code_not_found`   | No active `CouponCode` matches (when redeeming by code).              |
| `inactive`         | `Coupon.Active == false`.                                             |
| `expired`          | `RedeemBy` set and `now > RedeemBy`.                                  |
| `cap_reached`      | `MaxRedemptions > 0` and `count(Discount where couponId) >= cap`.     |
| `already_used`     | `OncePerCustomer` and a `Discount` exists for (couponId, customerId). |
| `currency_mismatch`| Fixed coupon `Currency != ` target currency.                         |

The global cap and per-customer checks count `Discount` rows (the source of truth). The
unique guard against the *same* coupon being redeemed twice on one subscription is covered by
the per-customer / cap checks plus a DB unique index (§7.1).

---

## 6. Hexagonal placement

| Layer            | Additions                                                                                                   |
| ---------------- | ---------------------------------------------------------------------------------------------------------- |
| `core/domain`    | `coupon.go`, `coupon_code.go`, `discount.go`, `discount_apply.go` (pure calc), enums; invoice field additions |
| `core/port`      | `CouponRepository`, `CouponCodeRepository`, `DiscountRepository`; **`DiscountReader`** (narrow read port consumed by `InvoiceService`) |
| `core/service`   | `CouponService` — coupon/code CRUD, `ValidateAndPreview`, `Redeem`. **Narrow** (no engine) → no orchestration wrapper |
| `adapter/postgres` | `coupon_row.go`+`coupon_repo.go`, `coupon_code_row.go`+`coupon_code_repo.go`, `discount_row.go`+`discount_repo.go`; `discount_total` columns on invoice rows |
| `adapter/http`   | `coupon_handler.go` — merchant CRUD, code management, `:preview`, discount reads; Cedar authz; routes in `config/server.go` |
| `config/app.go`  | construct repos → `CouponService` → inject `DiscountReader` into `InvoiceService` → register `CouponHandler` |
| `schemas/app`    | `Coupon`, `CouponCode`, `Discount` Prisma models; `discountTotal` on `Invoice`/`InvoiceLineItem`; `constraints.sql` |

### 6.1 Ports

```go
// internal/core/port/repository.go
type CouponRepository interface {
    Create(ctx, domain.Coupon) (domain.Coupon, error)
    Update(ctx, domain.Coupon) (domain.Coupon, error)
    FindById(ctx, orgId, id string) (domain.Coupon, error)
    Find(ctx, orgId string, p domain.Pagination) ([]domain.Coupon, int, error)
    Delete(ctx, orgId, id string) error
}

type CouponCodeRepository interface {
    Create(ctx, domain.CouponCode) (domain.CouponCode, error)
    FindByCode(ctx, orgId, code string) (domain.CouponCode, error) // case-insensitive
    FindByCouponId(ctx, orgId, couponId string) ([]domain.CouponCode, error)
    Deactivate(ctx, orgId, id string) error
}

type DiscountRepository interface {
    Create(ctx, domain.Discount) (domain.Discount, error)
    Update(ctx, domain.Discount) (domain.Discount, error)
    FindById(ctx, orgId, id string) (domain.Discount, error)
    CountByCoupon(ctx, orgId, couponId string) (int, error)
    CountByCouponAndCustomer(ctx, orgId, couponId, customerId string) (int, error)
}

// internal/core/port/service.go — narrow read port for InvoiceService
type DiscountReader interface {
    ActiveForSubscription(ctx, orgId, subscriptionId string) ([]domain.Discount, error)
    ActiveForOrder(ctx, orgId, orderId string) ([]domain.Discount, error)
}
```

`InvoiceService` depends on `DiscountReader` only (read-only narrow port) — it never holds the
full `CouponService`. `CouponService` implements `DiscountReader` (or a thin reader does),
avoiding any construction cycle.

### 6.2 Authz (Cedar)

New actions in `policy.cedar` + `port` action constants: `ActionCreateCoupon`,
`ActionUpdateCoupon`, `ActionDeleteCoupon`, `ActionReadCoupon`, `ActionRedeemCoupon`. Handlers
call `authz.Enforce` before mutating actions, matching existing handlers. Coupon preview/redeem
is available to the checkout path under the existing onboarding/auth conventions.

### 6.3 HTTP surface

```
POST   /api/coupons                 create coupon            (merchant)
GET    /api/coupons                 list                     (merchant)
GET    /api/coupons/{id}            get                      (merchant)
PUT    /api/coupons/{id}            update                   (merchant)
DELETE /api/coupons/{id}            delete                   (merchant)
POST   /api/coupons/{id}/codes      add a redeemable code    (merchant)
GET    /api/coupons/{id}/codes      list codes               (merchant)
DELETE /api/coupon-codes/{id}       deactivate a code        (merchant)
POST   /api/coupons:preview         validate + preview       (checkout)
GET    /api/subscriptions/{id}/discounts   list a sub's discounts
GET    /api/discounts/{id}          get a discount
```

DTOs use `validate:"..."` tags off the single `lib.NewValidator`; handlers return `ApiError`.

---

## 7. Database

Prisma is schema source-of-truth (`db push`, no migrations). **Prisma cannot express arbitrary
`CHECK` constraints**, so invariants are enforced in two complementary places:

1. **Prisma models** — tables, columns, types, indexes, FKs.
2. **`schemas/app/constraints.sql`** — the `CHECK`/exclusion constraints, applied by a new
   `make db-constraints` step run **after** `db push` (idempotent; safe to re-run).

Domain constructors validate the same invariants first, so the API returns a clean `ApiError`
rather than a raw Postgres `23514` — the DB constraints are the un-bypassable backstop.

### 7.1 Prisma models (`schemas/app/schema.prisma`)

```prisma
model Coupon {
  orgId String @map("org_id")
  id    String @default(cuid())

  name         String
  discountType DiscountType @map("discount_type")

  amountOff  Int?     @map("amount_off")
  currency   String?
  percentOff  Decimal? @map("percent_off") @db.Decimal(5, 2)

  duration         Duration
  durationInCycles Int?     @map("duration_in_cycles")

  redeemBy DateTime? @map("redeem_by")

  appliesToProducts String[] @map("applies_to_products")

  maxRedemptions  Int     @default(0) @map("max_redemptions")
  oncePerCustomer Boolean @default(false) @map("once_per_customer")

  active    Boolean  @default(true)
  createdAt DateTime @default(now()) @map("created_at")
  updatedAt DateTime @updatedAt @map("updated_at")

  codes     CouponCode[]
  discounts Discount[]

  @@id([orgId, id])
  @@map("coupons")
}

model CouponCode {
  orgId String @map("org_id")
  id    String @default(cuid())

  couponId String @map("coupon_id")
  coupon   Coupon @relation(fields: [orgId, couponId], references: [orgId, id], onDelete: Cascade)

  code   String
  active Boolean @default(true)

  createdAt DateTime @default(now()) @map("created_at")
  updatedAt DateTime @updatedAt @map("updated_at")

  @@id([orgId, id])
  @@unique([orgId, code]) // codes unique per org (store upper-cased)
  @@map("coupon_codes")
}

model Discount {
  orgId String @map("org_id")
  id    String @default(cuid())

  couponId     String  @map("coupon_id")
  coupon       Coupon  @relation(fields: [orgId, couponId], references: [orgId, id])
  couponCodeId String? @map("coupon_code_id")
  customerId   String  @map("customer_id")

  subscriptionId String? @map("subscription_id")
  orderId        String? @map("order_id")

  // snapshot
  discountType      DiscountType @map("discount_type")
  amountOff         Int?         @map("amount_off")
  currency          String?
  percentOff        Decimal?     @map("percent_off") @db.Decimal(5, 2)
  duration          Duration
  durationInCycles  Int?         @map("duration_in_cycles")
  appliesToProducts String[]     @map("applies_to_products")

  startCycle Int            @default(0) @map("start_cycle")
  status     DiscountStatus @default(active)
  redeemedAt DateTime       @default(now()) @map("redeemed_at")
  endedAt    DateTime?      @map("ended_at")

  createdAt DateTime @default(now()) @map("created_at")
  updatedAt DateTime @updatedAt @map("updated_at")

  @@id([orgId, id])
  @@index([orgId, couponId])
  @@index([orgId, subscriptionId])
  // one redemption of a given coupon per subscription:
  @@unique([orgId, couponId, subscriptionId])
  @@map("discounts")
}

enum DiscountType { percentage fixed }
enum Duration     { once repeating forever }
enum DiscountStatus { active completed cancelled }
```

`discountTotal` added to existing `Invoice` (`@default(0)`) and `InvoiceLineItem`
(`@default(0)`) models.

### 7.2 `schemas/app/constraints.sql`

Applied idempotently after `db push` (each wrapped so re-running is a no-op):

```sql
-- coupons
ALTER TABLE coupons ADD CONSTRAINT coupons_amount_off_pos      CHECK (amount_off > 0);
ALTER TABLE coupons ADD CONSTRAINT coupons_currency_len        CHECK (currency IS NULL OR char_length(currency) = 3);
ALTER TABLE coupons ADD CONSTRAINT coupons_percent_off_range   CHECK (percent_off > 0 AND percent_off <= 100);
ALTER TABLE coupons ADD CONSTRAINT coupons_max_redemptions_nn  CHECK (max_redemptions >= 0);
ALTER TABLE coupons ADD CONSTRAINT coupons_discount_type_xor   CHECK (
  (amount_off IS NOT NULL AND currency IS NOT NULL AND percent_off IS NULL) OR
  (amount_off IS NULL     AND currency IS NULL     AND percent_off IS NOT NULL));
ALTER TABLE coupons ADD CONSTRAINT coupons_repeating_cycles    CHECK (
  (duration = 'repeating' AND duration_in_cycles >= 1) OR
  (duration <> 'repeating' AND duration_in_cycles IS NULL));

-- discounts (same money invariants on the snapshot + exactly-one target)
ALTER TABLE discounts ADD CONSTRAINT discounts_discount_type_xor CHECK (
  (amount_off IS NOT NULL AND currency IS NOT NULL AND percent_off IS NULL) OR
  (amount_off IS NULL     AND currency IS NULL     AND percent_off IS NOT NULL));
ALTER TABLE discounts ADD CONSTRAINT discounts_target_xor        CHECK (
  (subscription_id IS NOT NULL AND order_id IS NULL) OR
  (subscription_id IS NULL     AND order_id IS NOT NULL));
ALTER TABLE discounts ADD CONSTRAINT discounts_start_cycle_nn    CHECK (start_cycle >= 0);

-- invoice line discount sanity
ALTER TABLE invoice_line_items ADD CONSTRAINT ili_discount_nn  CHECK (discount_total >= 0);
ALTER TABLE invoice_line_items ADD CONSTRAINT ili_discount_cap CHECK (discount_total <= total);
ALTER TABLE invoices          ADD CONSTRAINT inv_discount_nn   CHECK (discount_total >= 0);
```

Each `ADD CONSTRAINT` is guarded for idempotency (e.g. a `DO $$ ... EXCEPTION WHEN duplicate_object`
block, or `DROP CONSTRAINT IF EXISTS` + `ADD`). The `make db-constraints` target is added to
the `Makefile` and chained after `db-push`; documented as a required re-run after any push that
rewrites these columns.

> Note: web/checkout Prisma schemas (`gphq-web`, `gphq-checkout`) are physically duplicated and
> are **out of scope** for this spec — mirroring there is a follow-up.

---

## 8. Engine parity

All discount behaviour — redemption-window selection, the `ApplyDiscounts` calculation, and
total recomputation — lives in `core/` (`domain` + `InvoiceService`/`OrderService`). Hatchet
(cron + per-org fan-out) and Temporal (long-lived workflow + `ContinueAsNew`) both reach
billing through `ChargeForBillingPeriod` → `BuildForBillingPeriod`, so both produce the
**identical discounted invoice with no adapter-specific code**. Duration is derived from the
deterministic cycle index, so it is stable under Temporal replay. Redemption is an HTTP/service
concern, engine-agnostic. No new workflow, signal, or topic is introduced.

---

## 9. Non-goals (v1)

- **Carry-forward of an over-large flat amount.** A fixed discount caps at the discountable
  base; leftover is lost (Stripe default). No customer credit balance is introduced.
- **Per-code caps/expiry.** Limits live on the `Coupon`; `CouponCode` carries only the string +
  active flag.
- **Strict "N successful payments" counting.** Duration is N billing cycles (§4.4).
- **Mirroring to web/checkout Prisma schemas** — follow-up.
- **Coupons on usage/metered overages beyond product targeting** — product-scope only in v1.

---

## 10. Testing strategy

- **`domain` (strongest):** table-driven tests for `ApplyDiscounts` — percentage, fixed,
  fixed-larger-than-base (clamp, no carry), product-targeted (only matching lines), **stacking**
  (order-dependence, cumulative clamp to zero), proportional allocation rounding (parts sum to
  raw). `NewCoupon` invariant tests (type XOR, repeating-cycles, percent range).
- **`service`:** `CouponService.validate` refusal matrix (§5.3); `Redeem` snapshot correctness;
  `BuildForBillingPeriod` applies in-window discounts and skips out-of-window ones across
  `once`/`repeating`/`forever`; dunning re-build idempotency (same cycle → same discount).
- **`adapter/http`:** real httptest harness (Cedar authz + authn) for coupon CRUD, `:preview`,
  and refusal status codes.
- **Integration (`//go:build integration`, `testDB(t)`):** repo round-trips and — importantly —
  that `constraints.sql` rejects an invalid row (e.g. both `amount_off` and `percent_off` set
  → `23514`).

---

## 11. Decisions log

| Decision                                  | Choice                                      | Rationale |
| ----------------------------------------- | ------------------------------------------- | --------- |
| Scope targeting unit                      | **Product** (`AppliesToProducts`)           | Matches Stripe `applies_to.products`; coarse but simple for merchants. |
| Over-large flat amount                    | **Don't carry** (leftover lost)             | User choice; avoids introducing a customer-credit ledger. |
| Apply scope                               | **Subscriptions + one-time orders**         | User choice; orders reuse existing `OrderItem.DiscountTotal`. |
| Code requirement                          | **Codes optional** (0..N per coupon)        | Mirrors Stripe coupon vs promotion_code; enables programmatic redemption. |
| Duration mechanism                        | **Derived from cycle math** (no counter)    | Dunning-idempotent + deterministic under Temporal replay (§4.4). |
| Stacking multiple discounts               | **Allowed**, ordered by `RedeemedAt`, per-line running-net clamp | No architectural barrier; deterministic order makes it well-defined. |
| `CHECK` constraints                       | **`constraints.sql` + `make db-constraints`** + domain validation | Prisma can't express `CHECK`; DB is the un-bypassable backstop. |
```
