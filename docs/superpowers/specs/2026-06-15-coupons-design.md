# Coupons & Discounts — Design Spec

**Date:** 2026-06-15
**Status:** Settled — ready for implementation planning
**Area:** Billing — coupon definitions, codes, and applied discounts

**Scope.** This spec covers the **data model** (`Coupon`, `CouponCode`, `Discount` + supporting
DB schema), the **discount calculation**, and the **service methods** for validation and
redemption. It deliberately does **not** specify the downstream *flows* that call these methods
(order-completion, billing-invoice, etc.) — where they get called is out of scope. The methods
here are building blocks those flows consume.

---

## 1. Summary

Merchant-defined **coupons** produce **discounts** on bills, following Stripe's separation:

| Aggregate    | Role                                                                                         |
| ------------ | ------------------------------------------------------------------------------------------- |
| `Coupon`     | The **definition** — the discount math (type, amount/percent, duration). **Immutable.**     |
| `CouponCode` | A **redeemable code**, N per coupon (1‑N). Carries all redemption-gating: string, active toggle, customer lock, expiry, its own cap, restrictions. (Stripe's *PromotionCode*.) |
| `Discount`   | The **applied instance** — recorded against a subscription/order, shown on invoices.         |

**Two load-bearing invariants, mutually reinforcing:**

- **The Coupon is immutable** — only `Name`, `Active`, and `Metadata` may change after creation,
  enforced strictly (domain has no economic setters → repo writes only those columns → a
  Postgres trigger is the un-bypassable backstop). The discount **terms** never change. This
  matches Stripe, whose Coupon is likewise editable only in `name`/`metadata`.
- **The Discount holds no snapshot** — because the Coupon can never change its terms, a
  `Discount` safely reads its math from the live `Coupon`. (A snapshot would only defend against
  mutable coupons; immutability removes the need.) Consequence: a Coupon referenced by any
  Discount can never be hard-deleted.

Discounts come off **before tax**, allocated across matching invoice lines.

---

## 2. Capabilities (traceability)

| #  | Capability                                                         | Provided by                                       |
| -- | ------------------------------------------------------------------ | ------------------------------------------------- |
| D1 | Flat amount **or** percentage off                                  | `Coupon.DiscountType` + DB mutual-exclusion check |
| D2 | Duration: one payment, N payments, or forever                      | `Coupon.Duration` + `DurationInCycles`            |
| D3 | A redeem-by calendar date                                          | `Coupon.RedeemBy` (global) + `CouponCode.ExpiresAt` (per-code) |
| D4 | Limit to specific plans/charges                                    | `Coupon.AppliesToProducts` (Product IDs)          |
| D5 | Cap total redemptions across all customers                         | `Coupon.MaxRedemptions` (global) + `CouponCode.MaxRedemptions` (per-code) |
| D6 | Reusable, or one-time per customer                                 | `Coupon.OncePerCustomer`                          |
| V1 | Validate a code against given lines, with the resulting discount   | `CouponService.Validate` (§5.1)                   |
| A1 | Discount math: before tax, per matching line, for the duration     | `domain.ApplyDiscounts` + duration cycle math (§4) |
| A2 | Targeted coupon discounts only matching lines; rest in full        | `domain.ApplyDiscounts` per-line allocation        |
| L1 | Refused when inactive, expired, cap hit, or already used           | Two-layer validation (§5.3)                       |
| L2 | Discount simply ends (e.g. on cancellation)                        | `Discount.Status` = `completed`; cycle math stops it |

**Code-layer extras:** lock a code to one customer (`CouponCode.CustomerId`), per-code expiry
and cap, and `Restrictions` — `FirstTimeTransaction` and `MinimumAmount`.

**Dropped (non-goal):** flat amount larger than the invoice carrying forward. See §9.

---

## 3. Domain model

IDs are `<prefix>_` + KSUID (`domain.GenerateId`). Money is `int64` minor units; currency is
ISO‑4217; percentages are `decimal.Decimal`. **Every aggregate carries `Metadata
map[string]string`** (JSON column).

### 3.1 `Coupon` — immutable definition (`internal/core/domain/coupon.go`)

```go
type Coupon struct {
    OrgId string // tenant shard key
    Id    string // "coup_" + KSUID

    // --- mutable: the ONLY editable fields ---
    Name     string
    Active   bool // disable the coupon (blocks new redemptions); does NOT change terms
    Metadata map[string]string
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
- **`Active` disables the coupon** — a non-economic availability switch; when `false` validation
  refuses new redemptions (§5.3). It never changes terms, so existing `Discount`s keep applying.
  It is the lever for retiring a coupon (including code-less ones) without a hard delete.
- `NewCoupon` enforces the percentage-vs-fixed exclusion, `repeating ⇒ DurationInCycles >= 1`,
  and `percent_off` range, returning an `ApiError` before the row hits Postgres. The DB `CHECK`
  + immutability trigger (§7.2) are the backstops.
- The struct exposes **no setters** for immutable fields — only `Rename`, `SetActive`,
  `SetMetadata`.

### 3.2 `CouponCode` — redeemable code & redemption gating (`internal/core/domain/coupon_code.go`)

Stripe's PromotionCode, mapped to our naming.

```go
type CouponCode struct {
    OrgId    string
    Id       string // "ccode_" + KSUID
    CouponId string // FK → Coupon (the discount math)
    Code     string // customer-facing, unique per org, matched case-insensitively (stored upper-cased)

    // --- mutable ---
    Active   bool
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

`Restrictions` is a JSON column. A coupon may have **zero** codes (redeemed programmatically
against the Coupon) or **many**. `TimesRedeemed` increments once per `Discount` creation — a
redemption-time counter, not a per-cycle one.

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

    StartCycle int            // subscription cycle at redemption (0 for orders)
    Status     DiscountStatus // active | completed | cancelled
    RedeemedAt time.Time
    EndedAt    time.Time // nullable

    Metadata  map[string]string
    CreatedAt time.Time
    UpdatedAt time.Time
}

type DiscountStatus string // "active" | "completed" | "cancelled"
```

**No snapshot** — economic terms are read from the referenced immutable `Coupon`. The cycle
window (§4.3) uses `Discount.StartCycle` with the Coupon's `Duration`/`DurationInCycles`.

### 3.4 Discount fields on invoices/orders

To make a discount visible on the bill and keep totals honest:

- `InvoiceLineItem.DiscountTotal int64` (new; `>= 0`, `<= Total`)
- `Invoice.DiscountTotal int64` (new; sum of line discounts; `Total = Subtotal − DiscountTotal`)
- `OrderItem.DiscountTotal int64` — **already exists**, reused.

(How and when these get populated is the consuming flow's concern, out of scope here.)

---

## 4. Discount calculation

### 4.1 Pure domain function

`internal/core/domain/discount_apply.go`:

```go
// AppliedDiscount: a Discount paired with its (immutable) Coupon. The caller resolves both.
type AppliedDiscount struct {
    Discount Discount
    Coupon   Coupon
}

// DiscountableLine: a line with its resolved Product.
type DiscountableLine struct {
    LineId    string
    ProductId string // resolved Price → Variant → Product
    Total     int64  // gross line total, minor units
}

// ApplyDiscounts returns the discount amount to record per line id.
// Pure, deterministic, side-effect free.
func ApplyDiscounts(lines []DiscountableLine, applied []AppliedDiscount, cycle int, currency string) map[string]int64
```

Algorithm:

1. Keep only **in-window** discounts for `cycle` (§4.3) and matching `currency` for fixed coupons.
2. Order by `RedeemedAt` ascending (stable; deterministic stacking).
3. Track a per-line **running net**, initialised to `line.Total`.
4. For each applied discount, using its Coupon's economics:
   - **Base** = sum of running nets of lines matching the Coupon's scope (`AppliesToProducts`
     contains `line.ProductId`; empty scope = all lines).
   - **Raw** = percentage: `round(base × percentOff/100)`; fixed: `min(amountOff, base)`
     (leftover not carried — §9).
   - **Allocate** raw across matching lines in proportion to running net (largest-remainder
     rounding → parts sum exactly to raw); subtract from each running net; accumulate into the
     line's `DiscountTotal`.
5. **Clamp invariant:** a line's cumulative `DiscountTotal` can never exceed its `Total` (running
   net floored at 0) — automatic, since each step works off the running net.

Stacking is well-defined and order-deterministic; no line can be discounted below zero.

### 4.2 Resolving a line's Product

`ApplyDiscounts` needs each line's `ProductId` (Price → Variant → Product). The caller resolves
it (via the price repo) before calling; the function itself stays pure.

### 4.3 Duration is derived, not counted

A discount applies to cycle *N* iff `StartCycle ≤ N < StartCycle + DurationInCycles`
(`once` = 1, `forever` = unbounded). **No mutable per-cycle counter.**

- **Idempotent:** rebuilding the same cycle's invoice yields the identical discount — no
  per-application decrement to double-count (matters under dunning retries).
- **Replay-safe:** derived from the deterministic cycle index → stable under workflow replay.

"N payments" means **N billing cycles**. Pauses correctly wait, since the cycle counter doesn't
advance. A discount is lazily marked `completed` once its window passes — for query/display
only; correctness comes from the cycle math.

> `CouponCode.TimesRedeemed` is a stored counter, but increments once per **redemption**, not per
> billing cycle — none of the duration counter's retry/replay hazards.

---

## 5. Validation & redemption methods

These are `CouponService` methods. **Callers (checkout/order/billing flows) are out of scope.**

### 5.1 `Validate` — check a code, compute the discount, write nothing

```go
type DiscountPreview struct {
    Valid         bool
    Reason        string // set when !Valid (see §5.3)
    DiscountTotal int64
    PerLine       map[string]int64 // lineId → discount
}

func (s *CouponService) Validate(
    ctx context.Context, orgId, code, customerId string, lines []domain.DiscountableLine,
) (DiscountPreview, error)
```

Resolves the code (org-scoped, case-insensitive) → `CouponCode` → `Coupon`, runs §5.3
validation, and computes the discount over `lines` via `domain.ApplyDiscounts`. **No writes.**
Used wherever a caller needs to validate/preview a code (e.g. before confirming a purchase).

### 5.2 `Redeem` — create the applied Discount

```go
type RedeemInput struct {
    OrgId          string
    Code           string // empty ⇒ programmatic; resolve CouponId directly
    CouponId       string // used when Code is empty
    CustomerId     string
    SubscriptionId string // exactly one target
    OrderId        string //   (subscription or order)
    StartCycle     int    // subscription cycle at redemption (0 for orders)
}

func (s *CouponService) Redeem(ctx context.Context, in RedeemInput) (domain.Discount, error)
```

Re-runs §5.3 validation, then creates the `Discount` (no snapshot — references the Coupon),
records `CouponCodeId` (if any), and increments `CouponCode.TimesRedeemed`. Wraps both writes in
`port.TxManager` so they commit atomically; supports being called inside a caller's existing
transaction via the ctx-propagated tx. Global redemption counts derive from `Discount` rows
(source of truth). **This spec does not prescribe who calls `Redeem` or when** — that is the
order/purchase flow's concern.

### 5.3 Validation / refusal — two layers

Run by both `Validate` and `Redeem`. Refused with a specific reason when any check fails.

**Coupon layer (global):**

| Reason              | Check                                                                |
| ------------------- | -------------------------------------------------------------------- |
| `coupon_inactive`   | `Coupon.Active == false`.                                            |
| `expired`           | `Coupon.RedeemBy` set and `now > RedeemBy`.                          |
| `cap_reached`       | `MaxRedemptions > 0` and `count(Discount where couponId) >= cap`.    |
| `already_used`      | `OncePerCustomer` and a `Discount` exists for (couponId, customerId).|
| `currency_mismatch` | Fixed coupon `Currency != ` target currency.                        |

**Code layer (per `CouponCode`):**

| Reason              | Check                                                              |
| ------------------- | ----------------------------------------------------------------- |
| `code_not_found`    | No `CouponCode` matches the string.                               |
| `inactive`          | `CouponCode.Active == false`.                                     |
| `code_expired`      | `ExpiresAt` set and `now > ExpiresAt`.                           |
| `code_cap_reached`  | `MaxRedemptions > 0` and `TimesRedeemed >= MaxRedemptions`.       |
| `wrong_customer`    | `CustomerId` set and `!= ` redeeming customer.                  |
| `not_first_time`    | `Restrictions.FirstTimeTransaction` and customer has a prior successful payment. |
| `below_minimum`     | `Restrictions.MinimumAmount > 0` and line subtotal `< MinimumAmount` (currency-checked). |

Programmatic (code-less) redemption runs only the **Coupon layer** checks.

---

## 6. Hexagonal placement

| Layer              | Additions                                                                                                   |
| ------------------ | ---------------------------------------------------------------------------------------------------------- |
| `core/domain`      | `coupon.go`, `coupon_code.go` (+`Restrictions`), `discount.go`, `discount_apply.go` (pure); invoice/order discount fields |
| `core/port`        | `CouponRepository`, `CouponCodeRepository`, `DiscountRepository`                                            |
| `core/service`     | `CouponService` — coupon/code CRUD (immutability-respecting), `Validate`, `Redeem`. **Narrow**, no engine   |
| `adapter/postgres` | `coupon_row.go`+`_repo.go`, `coupon_code_row.go`+`_repo.go`, `discount_row.go`+`_repo.go`; `discount_total` columns |
| `adapter/http`     | `coupon_handler.go` — coupon/code CRUD + discount reads; Cedar authz; routes in `config/server.go`          |
| `config/app.go`    | repos → `CouponService` → register `CouponHandler`                                                          |
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
    ActiveForSubscription(ctx, orgId, subscriptionId string) ([]domain.Discount, error)
    ActiveForOrder(ctx, orgId, orderId string) ([]domain.Discount, error)
    CountByCoupon(ctx, orgId, couponId string) (int, error)
    CountByCouponAndCustomer(ctx, orgId, couponId, customerId string) (int, error)
}
```

`CouponRepository` exposes **no general `Update`** — only `UpdateMutable` (name + active +
metadata), making immutability structural at the port boundary as well as the DB.

### 6.2 Authz (Cedar)

New actions: `ActionCreateCoupon`, `ActionUpdateCoupon` (name/active/metadata only),
`ActionDeleteCoupon`, `ActionReadCoupon`, `ActionManageCouponCode`. Handlers enforce before
mutating, matching existing handlers.

### 6.3 HTTP surface (coupon resource only)

```
POST   /api/coupons                  create coupon                         (merchant)
GET    /api/coupons                  list                                  (merchant)
GET    /api/coupons/{id}             get                                   (merchant)
PATCH  /api/coupons/{id}             update name/active/metadata ONLY      (merchant)
DELETE /api/coupons/{id}             delete (only if unreferenced)         (merchant)
POST   /api/coupons/{id}/codes       create a redeemable code              (merchant)
GET    /api/coupons/{id}/codes       list codes                            (merchant)
PATCH  /api/coupon-codes/{id}        update active/metadata                (merchant)
GET    /api/discounts/{id}           get a discount                        (merchant)
```

DTOs use `validate:"..."` off the single `lib.NewValidator`; handlers return `ApiError`.
`Validate`/`Redeem` are **service methods**, not endpoints — exposed by whatever flow consumes
them.

---

## 7. Database

Prisma defines the tables. **Prisma's schema language can't express `CHECK` constraints or
triggers**, so those invariants live in raw SQL alongside it:

1. **Prisma models** — tables, columns, types, indexes, FKs.
2. **`schemas/app/constraints.sql`** — `CHECK`/exclusion constraints **and** the coupon
   immutability trigger, applied after the schema is in place (via the build/migration step,
   or `make db-constraints` for a quick `db push`). Idempotent, so re-running is safe.

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

  customerId     String?   @map("customer_id")
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
ALTER TABLE discounts ADD CONSTRAINT discounts_target_xor    CHECK (
  (subscription_id IS NOT NULL AND order_id IS NULL) OR
  (subscription_id IS NULL     AND order_id IS NOT NULL));
ALTER TABLE discounts ADD CONSTRAINT discounts_start_cycle_nn CHECK (start_cycle >= 0);

-- invoice line discount sanity
ALTER TABLE invoice_line_items ADD CONSTRAINT ili_discount_nn  CHECK (discount_total >= 0);
ALTER TABLE invoice_line_items ADD CONSTRAINT ili_discount_cap CHECK (discount_total <= total);
ALTER TABLE invoices          ADD CONSTRAINT inv_discount_nn   CHECK (discount_total >= 0);

-- coupon immutability: only name, active, metadata, updated_at may change
CREATE OR REPLACE FUNCTION coupons_block_term_update() RETURNS trigger AS $$
BEGIN
  IF (NEW.discount_type, NEW.amount_off, NEW.currency, NEW.percent_off,
      NEW.duration, NEW.duration_in_cycles, NEW.applies_to_products,
      NEW.redeem_by, NEW.max_redemptions, NEW.once_per_customer)
   IS DISTINCT FROM
     (OLD.discount_type, OLD.amount_off, OLD.currency, OLD.percent_off,
      OLD.duration, OLD.duration_in_cycles, OLD.applies_to_products,
      OLD.redeem_by, OLD.max_redemptions, OLD.once_per_customer)
  THEN RAISE EXCEPTION 'coupon terms are immutable (only name/active/metadata may change)';
  END IF;
  RETURN NEW;
END; $$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS coupons_immutable ON coupons;
CREATE TRIGGER coupons_immutable BEFORE UPDATE ON coupons
  FOR EACH ROW EXECUTE FUNCTION coupons_block_term_update();
```

Each `ADD CONSTRAINT` is wrapped for idempotency (`DO $$ … EXCEPTION WHEN duplicate_object … $$`).
The `make db-constraints` target is chained after `db-push` and documented as a required re-run
after column-rewriting pushes.

> Web/checkout Prisma schemas are physically duplicated and **out of scope** — mirroring is a
> follow-up.

---

## 8. Engine parity

The discount math (`domain.ApplyDiscounts`) and duration derivation are pure `core/domain` code
with no engine awareness; duration derives from the deterministic cycle index, so it is stable
under workflow replay. Wherever a billing flow invokes them, both Hatchet and Temporal get the
identical result. (Those invocation flows are out of scope here, but the building blocks are
parity-safe by construction.)

---

## 9. Non-goals

- **Carry-forward of an over-large flat amount** (leftover lost; no customer credit balance).
- **Editing coupon terms** — immutable by design; "change" = create a new coupon.
- **Any downstream flow that consumes these methods** (order-completion, billing-invoice, etc.)
  — this spec provides the model and methods; wiring them in is separate work.
- **Coupons on usage/metered overages beyond product targeting.**
- **Mirroring to web/checkout Prisma schemas.**

---

## 10. Testing strategy

- **`domain` (strongest):** table-driven `ApplyDiscounts` — percentage, fixed,
  fixed-larger-than-base (clamp, no carry), product-targeted, **stacking** (order-dependence +
  cumulative clamp to zero), proportional-allocation rounding, in/out-of-window across
  `once`/`repeating`/`forever`. `NewCoupon` invariant tests; `Coupon` exposes no term setters.
- **`service`:** `Validate` two-layer refusal matrix (§5.3) and computed discount; `Redeem`
  creates the `Discount` + increments `TimesRedeemed` atomically and re-validates;
  `UpdateMutable` rejects term changes.
- **`adapter/http`:** httptest harness (Cedar authz + authn) for coupon/code CRUD; `PATCH`
  coupon ignores/rejects term fields; `DELETE` blocked when referenced.
- **Integration (`//go:build integration`, `testDB(t)`):** repo round-trips; `constraints.sql`
  rejects invalid rows (both `amount_off` and `percent_off` set → `23514`); the **immutability
  trigger** raises on a term UPDATE; the unique index blocks a second redemption per sub.

---

## 11. Decisions log

| Decision                         | Choice                                                                | Rationale |
| -------------------------------- | --------------------------------------------------------------------- | --------- |
| Coupon mutability                | **Immutable except name + active + metadata**, enforced at domain, port, and DB-trigger layers | Terms frozen (matches Stripe, lets us drop the snapshot); `active` is a non-economic disable switch. |
| Discount snapshot                | **Removed** — reads terms from the live, immutable Coupon              | Snapshot only defended against mutable coupons. |
| Field placement                  | Economic + global policy on **Coupon**; redemption gating on **CouponCode** | Mirrors Stripe Coupon vs PromotionCode. |
| `Restrictions`                   | `FirstTimeTransaction`, `MinimumAmount` (+`MinimumAmountCurrency`)     | Min-spend needs a currency to compare. |
| Metadata                         | On **all three** aggregates                                           | Per request. |
| Scope targeting unit             | **Product** (`AppliesToProducts`)                                     | Matches Stripe `applies_to.products`. |
| Over-large flat amount           | **Don't carry**                                                       | Avoids a credit ledger. |
| Duration mechanism               | **Derived from cycle math**                                           | Idempotent + replay-safe. |
| Stacking                         | **Allowed**, ordered by `RedeemedAt`, per-line running-net clamp      | Deterministic; no architectural barrier. |
| `CHECK` + immutability trigger   | **`constraints.sql` + `make db-constraints`** + domain validation     | Prisma can't express `CHECK`/triggers; DB is the backstop. |
| Duration unit                    | **N billing cycles**                                                  | The only sensible reading; not configurable. |
