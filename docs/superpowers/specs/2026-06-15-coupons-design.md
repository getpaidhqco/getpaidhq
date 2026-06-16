# Coupons & Discounts — Design Spec

**Date:** 2026-06-15
**Status:** Draft for review
**Area:** Billing — subscription & one-time order discounting
**Engines affected:** Both (Hatchet + Temporal) — by construction, see [Engine parity](#8-engine-parity)

---

## 1. Summary

Add merchant-defined **coupons** that customers redeem (by code or programmatically) to
receive **discounts** on subscription invoices and one-time orders. The model follows Stripe's
separation of concerns:

| Aggregate    | Role                                                                                         |
| ------------ | ------------------------------------------------------------------------------------------- |
| `Coupon`     | The **definition** — the discount math (type, amount/percent, duration). **Immutable.**     |
| `CouponCode` | A **redeemable code**, N per coupon (1‑N). Carries all redemption-gating: string, active toggle, customer lock, expiry, its own cap, restrictions. (Stripe's *PromotionCode*.) |
| `Discount`   | The **applied instance** — recorded against a subscription/order, shown on invoices.         |

The `Coupon` is *only* the discount definition and never the code. The `CouponCode` is the
customer-facing string plus every "can this be redeemed right now?" concern. The `Discount` is
the redeemed result, attached to a specific subscription or order, and is what reduces an
invoice. Discounts come off **before tax**, allocated across matching invoice lines.

**Two load-bearing invariants, mutually reinforcing:**

- **The Coupon is immutable** — only `name` and `metadata` may change after creation, enforced
  strictly (domain has no economic setters → repo writes only those columns → a Postgres
  trigger is the un-bypassable backstop). This matches Stripe, whose Coupon is likewise
  editable only in `name`/`metadata`.
- **The Discount holds no snapshot** — because the Coupon can never change its terms, a
  `Discount` safely reads its math from the live `Coupon` at apply time. (The previous
  snapshot design existed only to defend against mutable coupons; immutability removes the
  need.) A consequence: a Coupon referenced by any Discount can never be hard-deleted.

---

## 2. Goals (traceability)

| #  | User story / use case                                              | Where satisfied                                       |
| -- | ------------------------------------------------------------------ | ----------------------------------------------------- |
| D1 | Create a coupon as flat amount **or** percentage off               | `Coupon.DiscountType` + DB mutual-exclusion check     |
| D2 | Duration: one payment, N payments, or forever                      | `Coupon.Duration` + `DurationInCycles`                |
| D3 | A redeem-by calendar date                                          | `Coupon.RedeemBy` (global) + `CouponCode.ExpiresAt` (per-code) |
| D4 | Limit a coupon to specific plans/charges                           | `Coupon.AppliesToProducts` (Product IDs)              |
| D5 | Cap total redemptions across all customers                         | `Coupon.MaxRedemptions` (global) + `CouponCode.MaxRedemptions` (per-code) |
| D6 | Reusable, or one-time per customer                                 | `Coupon.OncePerCustomer`                              |
| R1 | Enter a code at checkout and see the discount before confirming    | `POST /api/coupons:preview`                            |
| R2 | Discount attaches to the subscription and counts from its start    | `Discount.SubscriptionId` + `StartCycle`              |
| A1 | Comes off each bill before tax, for the duration, then stops       | `InvoiceService.BuildForBillingPeriod` + cycle math   |
| A2 | Targeted coupon discounts only matching plan/charge; rest in full  | `domain.ApplyDiscounts` per-line allocation           |
| L1 | Refused when expired, cap hit, or already used (one-time)          | Two-layer redemption validation (§5.3)                |
| L2 | Cancel mid-discount → remaining discount simply ends               | Subscription stops billing; discount → `completed`    |

**Added by the code layer (beyond original stories):** lock a code to one customer
(`CouponCode.CustomerId`), per-code expiry and cap, and `Restrictions` —
`FirstTimeTransaction` (no prior payments) and `MinimumAmount` (min spend to qualify).

**Dropped (non-goal):** flat amount larger than the invoice carrying forward. See §9.

---

## 3. Domain model

IDs are `<prefix>_` + KSUID (`domain.GenerateId`). Money is `int64` minor units; currency is
ISO‑4217; percentages are `decimal.Decimal`. **Every aggregate carries `Metadata
map[string]string`** (stored as a JSON column).

### 3.1 `Coupon` — immutable definition (`internal/core/domain/coupon.go`)

```go
type Coupon struct {
    OrgId string // tenant shard key
    Id    string // "coup_" + KSUID

    // --- mutable: the ONLY editable fields ---
    Name     string
    Metadata map[string]string
    Active   bool // disable the coupon (blocks new redemptions); does NOT change terms
    // -----------------------------------------

    // --- immutable after creation (economic + global policy) ---
    DiscountType DiscountType    // percentage | fixed
    PercentOff   decimal.Decimal // when percentage: 0 < p <= 100
    AmountOff    int64           // when fixed: > 0, minor units
    Currency     string          // when fixed: ISO-4217; must equal invoice currency at apply

    Duration         Duration // once | repeating | forever
    DurationInCycles int      // set iff Duration == repeating, >= 1

    RedeemBy          time.Time // nullable; global redeem-by cutoff
    AppliesToProducts []string  // Product IDs; empty = whole bill
    MaxRedemptions    int       // global cap; 0 = unlimited
    OncePerCustomer   bool      // false = reusable across a customer's redemptions
    // -----------------------------------------------------------

    CreatedAt time.Time
    UpdatedAt time.Time
}

type DiscountType string // "percentage" | "fixed"
type Duration     string // "once" | "repeating" | "forever"
```

Notes:
- **`Active` disables the coupon** — when `false`, no new redemptions are accepted (checked at
  the coupon layer, §5.3). It is a non-economic availability switch: it never changes discount
  terms, so existing `Discount`s keep applying. This is the lever for retiring a coupon
  (including code-less/programmatic ones) without a hard delete.
- The constructor `NewCoupon` enforces the percentage-vs-fixed exclusion, `repeating ⇒
  DurationInCycles >= 1`, and `percent_off` range, returning an `ApiError` before the row hits
  Postgres. The DB `CHECK` + immutability trigger (§7.2) are the backstops.
- The struct exposes **no setters** for immutable fields — only `Rename(string)`,
  `SetMetadata(map[string]string)`, and `SetActive(bool)`.

### 3.2 `CouponCode` — redeemable code & redemption gating (`internal/core/domain/coupon_code.go`)

Stripe's PromotionCode, mapped to our naming.

```go
type CouponCode struct {
    OrgId    string
    Id       string // "ccode_" + KSUID
    CouponId string // FK → Coupon (the discount math)
    Code     string // customer-facing, unique per org, matched case-insensitively (stored upper-cased)

    // --- mutable ---
    Active   bool              // toggle off without deleting
    Metadata map[string]string
    // ---------------

    // --- set at creation, then fixed ---
    CustomerId     string       // nullable — lock the code to one customer
    ExpiresAt      time.Time    // nullable — redeem-by cutoff at the code layer
    MaxRedemptions int          // per-code cap; 0 = unlimited
    Restrictions   Restrictions // additional eligibility gates
    // ------------------------------------

    TimesRedeemed int // system-managed running count, incremented on redeem

    CreatedAt time.Time
    UpdatedAt time.Time
}

type Restrictions struct {
    FirstTimeTransaction  bool   // only customers with no prior successful payment
    MinimumAmount         int64  // nullable (0 = none) — minimum spend to qualify, minor units
    MinimumAmountCurrency string // currency for MinimumAmount (required when MinimumAmount > 0)
}
```

`Restrictions` is stored as a JSON column (`serializer:json`), like `Metadata`. A coupon may
have **zero** codes (redeemed programmatically against the Coupon directly) or **many**.
`TimesRedeemed` is a redemption-time counter (incremented once per `Discount` creation) — not a
per-billing-cycle counter, so it has no dunning-retry hazard.

### 3.3 `Discount` — the applied instance (`internal/core/domain/discount.go`)

```go
type Discount struct {
    OrgId        string
    Id           string // "disc_" + KSUID
    CouponId     string // the discount math (read live; Coupon is immutable)
    CouponCodeId string // nullable; empty when redeemed programmatically
    CustomerId   string

    // exactly one target (ctor + DB CHECK):
    SubscriptionId string // recurring discount
    OrderId        string // one-time order discount

    StartCycle int            // subscription.CyclesProcessed at redemption (0 for orders)
    Status     DiscountStatus // active | completed | cancelled
    RedeemedAt time.Time
    EndedAt    time.Time // nullable

    Metadata  map[string]string
    CreatedAt time.Time
    UpdatedAt time.Time
}

type DiscountStatus string // "active" | "completed" | "cancelled"
```

**No snapshot.** Economic terms are read from the referenced `Coupon` at apply time; safe
because the Coupon is immutable. The cycle window (§4.4) uses `Discount.StartCycle` together
with the Coupon's `Duration`/`DurationInCycles`.

### 3.4 Invoice changes (`internal/core/domain/invoice.go`, `invoice_line_item.go`)

```go
// InvoiceLineItem
DiscountTotal int64 // >= 0, <= Total; amount discounted from this line

// Invoice
DiscountTotal int64 // sum of line DiscountTotal
```

`Invoice.recalculate()`:

```
Subtotal      = Σ line.Total                 // gross, unchanged
DiscountTotal = Σ line.DiscountTotal
Total         = Subtotal − DiscountTotal     // tax, when implemented, applies on Total here
```

One-time orders reuse the **already-present** `OrderItem.DiscountTotal` column.

---

## 4. Discount application (the load-bearing seam)

### 4.1 Pure domain function

`internal/core/domain/discount_apply.go`:

```go
// Resolved by the caller (InvoiceService): a Discount paired with its (immutable) Coupon.
type AppliedDiscount struct {
    Discount Discount
    Coupon   Coupon
}

// DiscountableLine: the minimal line view, with its resolved Product.
type DiscountableLine struct {
    LineId    string
    ProductId string // resolved Price → Variant → Product
    Total     int64  // gross line total, minor units
}

// ApplyDiscounts returns the discount amount to record per line id.
// Pure, deterministic, side-effect free.
func ApplyDiscounts(lines []DiscountableLine, applied []AppliedDiscount, invoiceCycle int, currency string) map[string]int64
```

Algorithm:

1. Keep only **in-window** discounts for `invoiceCycle` (§4.4) and matching `currency` for
   fixed coupons.
2. Order them by `RedeemedAt` ascending (stable; deterministic stacking).
3. Track a per-line **running net**, initialised to `line.Total`.
4. For each applied discount, in order, using its Coupon's economics:
   - **Base** = sum of running nets of lines matching the Coupon's scope
     (`AppliesToProducts` contains `line.ProductId`; empty scope = all lines).
   - **Raw** = percentage: `round(base × percentOff/100)`; fixed: `min(amountOff, base)`
     (leftover is **not** carried — §9).
   - **Allocate** the raw amount across matching lines in proportion to each line's running net
     (largest-remainder rounding → parts sum exactly to raw).
   - Subtract each allocation from that line's running net; accumulate into the line's
     `DiscountTotal`.
5. **Clamp invariant:** a line's cumulative `DiscountTotal` can never exceed its `Total`
   (running net floored at 0) — holds automatically because each step works off the running net.

Stacking is well-defined and order-deterministic; no line can be discounted below zero.

### 4.2 Where it's invoked — subscriptions

`InvoiceService.BuildForBillingPeriod` (`internal/core/service/invoice.go`), **after** lines are
built and **before** `recalculate()`:

1. Build gross lines (today's behaviour).
2. Resolve each line's `ProductId` (Price → Variant → Product) via the price repo.
3. Load **active** `Discount`s for the subscription (`DiscountReader`), and load each one's
   `Coupon` (immutable; cacheable). Pair into `[]AppliedDiscount`.
4. `domain.ApplyDiscounts(lines, applied, invoice.Cycle, sub.Currency)` → write each
   `line.DiscountTotal`.
5. `recalculate()` → `Total = Subtotal − DiscountTotal`.

All in `core/` → both engines charge the discounted `Invoice.Total` with **zero adapter code**.

### 4.3 Where it's invoked — one-time orders

Same pure function once at checkout/order completion (`OrderService.CompleteOrder`), writing
`OrderItem.DiscountTotal`. Duration is irrelevant for a single charge (`once`).

### 4.4 Duration is derived, not counted

A discount applies to cycle *N* iff `StartCycle ≤ N < StartCycle + DurationInCycles`
(`once` = 1, `forever` = unbounded). **No mutable per-cycle counter.**

- **Dunning-idempotent:** the same cycle's invoice can be rebuilt/retried any number of times
  and yields the identical discount — no per-application decrement to double-count.
- **Parity-safe:** derived from the deterministic cycle index → stable under Temporal
  replay/`ContinueAsNew`; a stored counter would be extra state both engines must keep in sync.

**Known semantic:** "N payments" is realised as "N billing **cycles**" (= N paid invoices when
every covered cycle is paid; pauses correctly wait, since `CyclesProcessed` doesn't advance).
Strict "N successful payments" is out of scope (§9). A discount is lazily marked `completed`
(status + `EndedAt`) once its window passes or on subscription cancellation — for
query/display only; correctness comes from the cycle math.

> Note: `CouponCode.TimesRedeemed` (§3.2) *is* a stored counter, but it increments once per
> **redemption**, not per billing cycle — so it carries none of the duration counter's
> retry/parity hazards. Different lifecycle, different decision.

---

## 5. Redemption

### 5.1 Preview (checkout, no writes)

`POST /api/coupons:preview`

```
Request:  { code: string, customerId?: string, orderId?: string, lines?: [{ priceId, quantity }] }
Response: { valid: bool, reason?: string, discountTotal: int64, perLine: [{ priceId, discount }] }
```

Resolves the code (org-scoped, case-insensitive) → `CouponCode` → `Coupon`, runs §5.3
validation, and computes the discount over the prospective lines with the same `ApplyDiscounts`
function. Shows the amount the customer sees **before** confirming.

### 5.2 Redeem (on confirm)

Creates a `Discount` linked to the subscription (or order) with `StartCycle =
subscription.CyclesProcessed` (0 for orders), records `CouponCodeId` (if any), and increments
`CouponCode.TimesRedeemed`. Redeem runs inside `port.TxManager` so the Discount insert and the
`TimesRedeemed` increment commit atomically; post-commit side-effects follow the existing
pattern. Global redemption counts derive from `Discount` rows (source of truth).

### 5.3 Validation / refusal — two layers

A redemption (and preview) is refused with a specific reason when any check fails.

**Coupon layer (global):**

| Reason            | Check                                                                |
| ----------------- | -------------------------------------------------------------------- |
| `coupon_inactive` | `Coupon.Active == false`.                                            |
| `expired`         | `Coupon.RedeemBy` set and `now > RedeemBy`.                          |
| `cap_reached`     | `MaxRedemptions > 0` and `count(Discount where couponId) >= cap`.    |
| `already_used`    | `OncePerCustomer` and a `Discount` exists for (couponId, customerId).|
| `currency_mismatch` | Fixed coupon `Currency != ` target currency.                      |

**Code layer (per `CouponCode`):**

| Reason              | Check                                                              |
| ------------------- | ----------------------------------------------------------------- |
| `code_not_found`    | No `CouponCode` matches the string.                               |
| `inactive`          | `CouponCode.Active == false`.                                     |
| `code_expired`      | `ExpiresAt` set and `now > ExpiresAt`.                            |
| `code_cap_reached`  | `MaxRedemptions > 0` and `TimesRedeemed >= MaxRedemptions`.       |
| `wrong_customer`    | `CustomerId` set and `!= ` redeeming customer.                   |
| `not_first_time`    | `Restrictions.FirstTimeTransaction` and customer has a prior successful payment. |
| `below_minimum`     | `Restrictions.MinimumAmount > 0` and cart/invoice subtotal `< MinimumAmount` (currency-checked). |

Programmatic (code-less) redemption runs only the **Coupon layer** checks. A DB unique index
prevents the same coupon being redeemed twice on one subscription (§7.1).

---

## 6. Hexagonal placement

| Layer              | Additions                                                                                                   |
| ------------------ | ---------------------------------------------------------------------------------------------------------- |
| `core/domain`      | `coupon.go`, `coupon_code.go` (+`Restrictions`), `discount.go`, `discount_apply.go` (pure); invoice fields  |
| `core/port`        | `CouponRepository`, `CouponCodeRepository`, `DiscountRepository`; **`DiscountReader`** (narrow read port for `InvoiceService`) |
| `core/service`     | `CouponService` — coupon/code CRUD (respecting immutability), `ValidateAndPreview`, `Redeem`. **Narrow**, no engine |
| `adapter/postgres` | `coupon_row.go`+`_repo.go`, `coupon_code_row.go`+`_repo.go`, `discount_row.go`+`_repo.go`; `discount_total` columns |
| `adapter/http`     | `coupon_handler.go` — coupon/code CRUD, `:preview`, discount reads; Cedar authz; routes in `config/server.go` |
| `config/app.go`    | repos → `CouponService` → inject `DiscountReader` into `InvoiceService` → register `CouponHandler`           |
| `schemas/app`      | `Coupon`, `CouponCode`, `Discount` Prisma models; `discountTotal` on invoice models; `constraints.sql`      |

### 6.1 Ports

```go
type CouponRepository interface {
    Create(ctx, domain.Coupon) (domain.Coupon, error)
    UpdateMutable(ctx, orgId, id, name string, active bool, metadata map[string]string) (domain.Coupon, error) // name + active + metadata ONLY
    FindById(ctx, orgId, id string) (domain.Coupon, error)
    Find(ctx, orgId string, p domain.Pagination) ([]domain.Coupon, int, error)
    DeleteIfUnreferenced(ctx, orgId, id string) error // errors if any Discount references it
}

type CouponCodeRepository interface {
    Create(ctx, domain.CouponCode) (domain.CouponCode, error)
    UpdateMutable(ctx, orgId, id string, active bool, metadata map[string]string) (domain.CouponCode, error)
    IncrementRedeemed(ctx, orgId, id string) error
    FindByCode(ctx, orgId, code string) (domain.CouponCode, error) // case-insensitive
    FindByCouponId(ctx, orgId, couponId string) ([]domain.CouponCode, error)
}

type DiscountRepository interface {
    Create(ctx, domain.Discount) (domain.Discount, error)
    Update(ctx, domain.Discount) (domain.Discount, error)
    FindById(ctx, orgId, id string) (domain.Discount, error)
    CountByCoupon(ctx, orgId, couponId string) (int, error)
    CountByCouponAndCustomer(ctx, orgId, couponId, customerId string) (int, error)
}

// Narrow read port for InvoiceService — never holds the full CouponService.
type DiscountReader interface {
    ActiveForSubscription(ctx, orgId, subscriptionId string) ([]domain.Discount, error)
    ActiveForOrder(ctx, orgId, orderId string) ([]domain.Discount, error)
}
```

`CouponRepository` exposes **no general `Update`** — only `UpdateMutable` (name + metadata),
making immutability structural at the port boundary as well as the DB.

### 6.2 Authz (Cedar)

New actions: `ActionCreateCoupon`, `ActionUpdateCoupon` (name/metadata only),
`ActionDeleteCoupon`, `ActionReadCoupon`, `ActionManageCouponCode`, `ActionRedeemCoupon`.
Handlers enforce before mutating, matching existing handlers.

### 6.3 HTTP surface

```
POST   /api/coupons                  create coupon                         (merchant)
GET    /api/coupons                  list                                  (merchant)
GET    /api/coupons/{id}             get                                   (merchant)
PATCH  /api/coupons/{id}             update name/active/metadata ONLY      (merchant)
DELETE /api/coupons/{id}             delete (only if unreferenced)         (merchant)
POST   /api/coupons/{id}/codes       create a redeemable code              (merchant)
GET    /api/coupons/{id}/codes       list codes                            (merchant)
PATCH  /api/coupon-codes/{id}        update active/metadata                (merchant)
POST   /api/coupons:preview          validate + preview                    (checkout)
GET    /api/subscriptions/{id}/discounts   list a sub's discounts
GET    /api/discounts/{id}           get a discount
```

`PATCH` (not `PUT`) on coupon reflects partial, restricted-field updates. DTOs use
`validate:"..."` off the single `lib.NewValidator`; handlers return `ApiError`.

---

## 7. Database

Prisma is schema source-of-truth (`db push`, no migrations). **Prisma cannot express `CHECK`
constraints or triggers**, so invariants live in two complementary places:

1. **Prisma models** — tables, columns, types, indexes, FKs.
2. **`schemas/app/constraints.sql`** — `CHECK`/exclusion constraints **and** the coupon
   immutability trigger, applied by a new `make db-constraints` step run **after** `db push`
   (idempotent; re-run after any push that rewrites these columns).

Domain constructors + `UpdateMutable` ports enforce the same invariants first (clean
`ApiError`); the DB objects are the un-bypassable backstop.

### 7.1 Prisma models (`schemas/app/schema.prisma`)

```prisma
model Coupon {
  orgId String @map("org_id")
  id    String @default(cuid())

  name     String
  metadata Json?
  active   Boolean @default(true)

  discountType DiscountType @map("discount_type")
  amountOff    Int?         @map("amount_off")
  currency     String?
  percentOff   Decimal?     @map("percent_off") @db.Decimal(5, 2)

  duration         Duration
  durationInCycles Int?     @map("duration_in_cycles")

  redeemBy          DateTime? @map("redeem_by")
  appliesToProducts String[]  @map("applies_to_products")
  maxRedemptions    Int       @default(0) @map("max_redemptions")
  oncePerCustomer   Boolean   @default(false) @map("once_per_customer")

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

  customerId     String? @map("customer_id")
  expiresAt      DateTime? @map("expires_at")
  maxRedemptions Int       @default(0) @map("max_redemptions")
  timesRedeemed  Int       @default(0) @map("times_redeemed")
  restrictions   Json?     // { firstTimeTransaction, minimumAmount, minimumAmountCurrency }

  metadata  Json?
  createdAt DateTime @default(now()) @map("created_at")
  updatedAt DateTime @updatedAt @map("updated_at")

  @@id([orgId, id])
  @@unique([orgId, code]) // store upper-cased
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

  startCycle Int            @default(0) @map("start_cycle")
  status     DiscountStatus @default(active)
  redeemedAt DateTime       @default(now()) @map("redeemed_at")
  endedAt    DateTime?      @map("ended_at")

  metadata  Json?
  createdAt DateTime @default(now()) @map("created_at")
  updatedAt DateTime @updatedAt @map("updated_at")

  @@id([orgId, id])
  @@index([orgId, couponId])
  @@index([orgId, subscriptionId])
  @@unique([orgId, couponId, subscriptionId]) // one redemption of a coupon per subscription
  @@map("discounts")
}

enum DiscountType   { percentage fixed }
enum Duration       { once repeating forever }
enum DiscountStatus { active completed cancelled }
```

`discountTotal Int @default(0)` added to existing `Invoice` and `InvoiceLineItem` models.

### 7.2 `schemas/app/constraints.sql`

Applied idempotently after `db push` (each guarded so re-running is a no-op).

```sql
-- coupons: economic invariants
ALTER TABLE coupons ADD CONSTRAINT coupons_amount_off_pos     CHECK (amount_off > 0);
ALTER TABLE coupons ADD CONSTRAINT coupons_currency_len       CHECK (currency IS NULL OR char_length(currency) = 3);
ALTER TABLE coupons ADD CONSTRAINT coupons_percent_off_range  CHECK (percent_off > 0 AND percent_off <= 100);
ALTER TABLE coupons ADD CONSTRAINT coupons_max_redemptions_nn CHECK (max_redemptions >= 0);
ALTER TABLE coupons ADD CONSTRAINT coupons_discount_type_xor  CHECK (
  (amount_off IS NOT NULL AND currency IS NOT NULL AND percent_off IS NULL) OR
  (amount_off IS NULL     AND currency IS NULL     AND percent_off IS NOT NULL));
ALTER TABLE coupons ADD CONSTRAINT coupons_repeating_cycles   CHECK (
  (duration = 'repeating' AND duration_in_cycles >= 1) OR
  (duration <> 'repeating' AND duration_in_cycles IS NULL));

-- coupon_codes
ALTER TABLE coupon_codes ADD CONSTRAINT codes_max_redemptions_nn CHECK (max_redemptions >= 0);
ALTER TABLE coupon_codes ADD CONSTRAINT codes_times_redeemed_nn  CHECK (times_redeemed >= 0);

-- discounts: exactly one target
ALTER TABLE discounts ADD CONSTRAINT discounts_target_xor   CHECK (
  (subscription_id IS NOT NULL AND order_id IS NULL) OR
  (subscription_id IS NULL     AND order_id IS NOT NULL));
ALTER TABLE discounts ADD CONSTRAINT discounts_start_cycle_nn CHECK (start_cycle >= 0);

-- invoice line discount sanity
ALTER TABLE invoice_line_items ADD CONSTRAINT ili_discount_nn  CHECK (discount_total >= 0);
ALTER TABLE invoice_line_items ADD CONSTRAINT ili_discount_cap CHECK (discount_total <= total);
ALTER TABLE invoices          ADD CONSTRAINT inv_discount_nn   CHECK (discount_total >= 0);

-- coupon immutability: only name, metadata, active, updated_at may change
CREATE OR REPLACE FUNCTION coupons_block_term_update() RETURNS trigger AS $$
BEGIN
  IF (NEW.discount_type, NEW.amount_off, NEW.currency, NEW.percent_off,
      NEW.duration, NEW.duration_in_cycles, NEW.applies_to_products,
      NEW.redeem_by, NEW.max_redemptions, NEW.once_per_customer)
   IS DISTINCT FROM
     (OLD.discount_type, OLD.amount_off, OLD.currency, OLD.percent_off,
      OLD.duration, OLD.duration_in_cycles, OLD.applies_to_products,
      OLD.redeem_by, OLD.max_redemptions, OLD.once_per_customer)
  THEN RAISE EXCEPTION 'coupon terms are immutable (only name/metadata may change)';
  END IF;
  RETURN NEW;
END; $$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS coupons_immutable ON coupons;
CREATE TRIGGER coupons_immutable BEFORE UPDATE ON coupons
  FOR EACH ROW EXECUTE FUNCTION coupons_block_term_update();
```

Each `ADD CONSTRAINT` is wrapped for idempotency (e.g. a `DO $$ … EXCEPTION WHEN
duplicate_object … $$` block). The `make db-constraints` target is chained after `db-push` in
the `Makefile` and documented as a required re-run after column-rewriting pushes.

> Web/checkout Prisma schemas are physically duplicated and **out of scope** here — mirroring is
> a follow-up.

---

## 8. Engine parity

All discount behaviour — window selection, `ApplyDiscounts`, and total recomputation — lives in
`core/` (`domain` + `InvoiceService`/`OrderService`). Hatchet (cron + per-org fan-out) and
Temporal (long-lived workflow + `ContinueAsNew`) both bill through `ChargeForBillingPeriod` →
`BuildForBillingPeriod`, so both produce the **identical discounted invoice with no
adapter-specific code**. Duration derives from the deterministic cycle index → stable under
replay. Redemption is an HTTP/service concern, engine-agnostic. No new workflow, signal, or
topic is introduced.

---

## 9. Non-goals (v1)

- **Carry-forward of an over-large flat amount** (leftover lost; no customer credit balance).
- **Strict "N successful payments" counting** — duration is N billing cycles (§4.4).
- **Editing coupon terms** — terms are immutable by design; "change" = create a new coupon.
- **Coupons on usage/metered overages beyond product targeting** — product-scope only.
- **Mirroring to web/checkout Prisma schemas** — follow-up.

---

## 10. Testing strategy

- **`domain` (strongest):** table-driven `ApplyDiscounts` — percentage, fixed,
  fixed-larger-than-base (clamp, no carry), product-targeted, **stacking** (order-dependence +
  cumulative clamp to zero), proportional-allocation rounding, in/out-of-window across
  `once`/`repeating`/`forever`. `NewCoupon` invariant tests; `Coupon` exposes no term setters.
- **`service`:** two-layer refusal matrix (§5.3); `Redeem` increments `TimesRedeemed` atomically;
  `UpdateMutable` rejects term changes; `BuildForBillingPeriod` applies in-window and skips
  out-of-window discounts; dunning re-build idempotency (same cycle → same discount).
- **`adapter/http`:** httptest harness (Cedar authz + authn) for CRUD, `:preview`, refusal codes;
  `PATCH` coupon ignores/rejects term fields.
- **Integration (`//go:build integration`, `testDB(t)`):** repo round-trips; `constraints.sql`
  rejects invalid rows (both `amount_off` and `percent_off` set → `23514`); the **immutability
  trigger** raises on a term UPDATE; the unique index blocks a second redemption per sub.

---

## 11. Decisions log

| Decision                         | Choice                                                                | Rationale |
| -------------------------------- | --------------------------------------------------------------------- | --------- |
| Coupon mutability                | **Immutable except name + metadata + active**, enforced at domain, port, and DB-trigger layers | Terms frozen (matches Stripe, lets us drop the snapshot); `active` is a non-economic disable switch. |
| Discount snapshot                | **Removed** — reads terms from the live, immutable Coupon              | Snapshot only defended against mutable coupons; immutability removes the need. |
| Field placement                  | Economic + global policy on **Coupon**; redemption gating on **CouponCode** | Mirrors Stripe Coupon vs PromotionCode. |
| Code layer                       | Stripe-shaped: `Active`, `CustomerId`, `ExpiresAt`, `MaxRedemptions`, `TimesRedeemed`, `Restrictions`, `Metadata` | Per request. |
| `Restrictions`                   | `FirstTimeTransaction`, `MinimumAmount` (+`MinimumAmountCurrency`)     | Min-spend needs a currency to compare. |
| Metadata                         | On **all three** aggregates                                           | Per request. |
| Scope targeting unit             | **Product** (`AppliesToProducts`)                                     | Matches Stripe `applies_to.products`. |
| Over-large flat amount           | **Don't carry**                                                       | User choice; avoids a credit ledger. |
| Apply scope                      | **Subscriptions + one-time orders**                                   | Orders reuse existing `OrderItem.DiscountTotal`. |
| Duration mechanism               | **Derived from cycle math**                                           | Dunning-idempotent + deterministic under replay. |
| Stacking                         | **Allowed**, ordered by `RedeemedAt`, per-line running-net clamp      | Deterministic; no architectural barrier. |
| `CHECK` + immutability trigger   | **`constraints.sql` + `make db-constraints`** + domain validation     | Prisma can't express `CHECK`/triggers; DB is the backstop. |

---

## 12. Open questions

1. **"N payments" semantics** — realised as N billing cycles (§4.4), not N strictly-paid
   invoices. Flag if that must change before implementation.
```
