# Coupon Reservation & Discount Application — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire coupons into the order→subscription→billing flow — reserve a code's capacity at order create, convert it to a `Discount` on payment success, and apply the discount when each cycle's invoice is built — so a 50%-off `repeating(2)` coupon on a subscription bills `2×$50 + 3×$100`.

**Architecture:** Ephemeral `coupon_reservations` table holds code capacity (no status; row-counted caps; atomic reserve under `FOR UPDATE`; lazy expiry). `CouponService` gains `Reserve`/`Consume`/`Release`. `InvoiceService.BuildForBillingPeriod` (shared by both engines) applies active discounts via the existing pure `domain.ApplyDiscounts`. All logic lives in `core/{domain,service}`, so Hatchet and Temporal get it for free.

**Tech Stack:** Go 1.24, GORM + hand-written pgx (both storage drivers, parity-tested via `storagetest`), Goose migrations, Postgres, testcontainers integration tests.

**Spec:** `docs/superpowers/specs/2026-06-23-coupon-reservation-and-application-design.md`.

**Scope (build-now):** inline `coupon_code` on `CreateOrder`, customer known, applied to a subscription via the `direct` path (the live rig). **Out of scope** (schema is shaped for them, not built): the checkout-session holder (`BindCustomer`/`AttachOrder`/`PreviewForHolder`), one-time-order invoices (`BuildForOrder`), and the order payment/invoice flags. For build-now the reservation holder is always the **order**, so the service/repo use order-based methods; the `checkout_session_id` column exists (nullable, unused).

**Cycle-0:** the e2e completes the order **without** a caller payment, so the existing billing sweep charges every cycle (0..4) through `BuildForBillingPeriod` — which now applies the discount. No change to `CompleteOrder`'s payment handling.

---

## File structure

| File | Responsibility |
| --- | --- |
| `schemas/app/migrations/000NN_coupon_reservations.sql` | the table |
| `internal/core/domain/coupon_reservation.go` | `CouponReservation` aggregate (`NewCouponReservation`, `IsLive`) |
| `internal/core/port/coupon.go` | `CouponReservationRepository`; lock methods on coupon/code repos |
| `internal/adapter/storage/postgresgorm/coupon_reservation_{row,repo}.go` | gorm persistence |
| `internal/adapter/storage/postgrespgx/coupon_reservation_{row,repo}.go` | pgx persistence |
| `internal/adapter/storage/storagetest/conformance.go` | cross-driver reservation sub-test |
| `internal/core/service/coupon.go` | reservation-aware gate + `Reserve`/`Consume`/`Release` |
| `internal/core/domain/invoice_build.go` | `Invoice.ApplyDiscountTotals` |
| `internal/core/service/invoice.go` | discount application in `BuildForBillingPeriod` |
| `internal/adapter/http/request.go`, `internal/core/port/*`, `internal/core/service/order.go` | `coupon_code` on `CreateOrder` → reserve; `Consume` in `CompleteOrder` |
| `internal/config/app.go` | wiring |

---

## Task 1: Goose migration — `coupon_reservations`

**Files:**
- Create: `schemas/app/migrations/000NN_coupon_reservations.sql` (use the next sequential number after the highest existing in that dir)

- [ ] **Step 1: Write the migration**

```sql
-- +goose Up
CREATE TABLE "coupon_reservations" (
    "org_id"              TEXT NOT NULL,
    "id"                  TEXT NOT NULL,
    "coupon_id"           TEXT NOT NULL,
    "coupon_code_id"      TEXT,
    "customer_id"         TEXT,
    "checkout_session_id" TEXT,
    "order_id"            TEXT,
    "expires_at"          TIMESTAMP(3) NOT NULL,
    "created_at"          TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "coupon_reservations_pkey" PRIMARY KEY ("org_id","id"),
    CONSTRAINT "coupon_reservations_has_holder" CHECK ("checkout_session_id" IS NOT NULL OR "order_id" IS NOT NULL),
    CONSTRAINT "coupon_reservations_coupon_fkey" FOREIGN KEY ("org_id","coupon_id") REFERENCES "coupons"("org_id","id") ON DELETE CASCADE
);
CREATE UNIQUE INDEX "coupon_reservations_org_coupon_order_key"   ON "coupon_reservations"("org_id","coupon_id","order_id")            WHERE "order_id" IS NOT NULL;
CREATE UNIQUE INDEX "coupon_reservations_org_coupon_session_key" ON "coupon_reservations"("org_id","coupon_id","checkout_session_id") WHERE "checkout_session_id" IS NOT NULL;
CREATE INDEX "coupon_reservations_org_coupon_idx" ON "coupon_reservations"("org_id","coupon_id");
CREATE INDEX "coupon_reservations_org_code_idx"   ON "coupon_reservations"("org_id","coupon_code_id");
CREATE INDEX "coupon_reservations_expires_idx"    ON "coupon_reservations"("expires_at");

-- +goose Down
DROP TABLE "coupon_reservations";
```

- [ ] **Step 2: Apply to a throwaway DB to verify it's valid**

Run: `make db-migrate-status` then apply against a scratch DB (or rely on Task 6's testcontainer, which runs the full baseline + this migration). Expected: no SQL errors.

- [ ] **Step 3: Commit**

```bash
git add schemas/app/migrations
git commit -m "feat(coupons): coupon_reservations migration"
```

---

## Task 2: Domain — `CouponReservation`

**Files:**
- Create: `internal/core/domain/coupon_reservation.go`
- Test: `internal/core/domain/coupon_reservation_test.go`

- [ ] **Step 1: Write the failing test**

```go
package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCouponReservation_RequiresHolder(t *testing.T) {
	_, err := NewCouponReservation(NewCouponReservationInput{
		OrgId: "o", CouponId: "c", ExpiresAt: time.Now().Add(time.Hour),
	})
	require.Error(t, err, "a reservation with no order and no session is invalid")
}

func TestCouponReservation_IsLive(t *testing.T) {
	r := CouponReservation{ExpiresAt: time.Now().Add(time.Hour)}
	assert.True(t, r.IsLive(time.Now()))
	expired := CouponReservation{ExpiresAt: time.Now().Add(-time.Hour)}
	assert.False(t, expired.IsLive(time.Now()))
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/core/domain/ -run TestCouponReservation -v` — Expected: FAIL (undefined).

- [ ] **Step 3: Implement**

```go
package domain

import (
	"time"

	"getpaidhq/internal/lib"
)

// CouponReservation is an ephemeral hold on a coupon code's redemption capacity
// for one checkout. Held by the order (build-now) or a checkout session
// (forward). No status — presence + ExpiresAt encode the state.
type CouponReservation struct {
	OrgId             string
	Id                string
	CouponId          string
	CouponCodeId      string // "" = programmatic / code-less
	CustomerId        string // "" until bound
	CheckoutSessionId string // holder (forward)
	OrderId           string // holder (build-now)
	ExpiresAt         time.Time
	CreatedAt         time.Time
}

type NewCouponReservationInput struct {
	OrgId             string
	CouponId          string
	CouponCodeId      string
	CustomerId        string
	CheckoutSessionId string
	OrderId           string
	ExpiresAt         time.Time
}

func NewCouponReservation(in NewCouponReservationInput) (CouponReservation, error) {
	if in.OrgId == "" || in.CouponId == "" {
		return CouponReservation{}, lib.NewCustomError(lib.ValidationError, "reservation requires org and coupon", nil)
	}
	if in.OrderId == "" && in.CheckoutSessionId == "" {
		return CouponReservation{}, lib.NewCustomError(lib.ValidationError, "reservation requires a holder (order or checkout session)", nil)
	}
	if in.ExpiresAt.IsZero() {
		return CouponReservation{}, lib.NewCustomError(lib.ValidationError, "reservation requires expires_at", nil)
	}
	now := time.Now().UTC()
	return CouponReservation{
		OrgId: in.OrgId, Id: lib.GenerateId("cres"),
		CouponId: in.CouponId, CouponCodeId: in.CouponCodeId, CustomerId: in.CustomerId,
		CheckoutSessionId: in.CheckoutSessionId, OrderId: in.OrderId,
		ExpiresAt: in.ExpiresAt, CreatedAt: now,
	}, nil
}

// IsLive reports whether the hold still counts at now.
func (r CouponReservation) IsLive(now time.Time) bool { return r.ExpiresAt.After(now) }
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/core/domain/ -run TestCouponReservation -v` — Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/core/domain/coupon_reservation.go internal/core/domain/coupon_reservation_test.go
git commit -m "feat(coupons): CouponReservation domain aggregate"
```

---

## Task 3: Port — `CouponReservationRepository` + coupon/code lock methods

**Files:**
- Modify: `internal/core/port/coupon.go`

- [ ] **Step 1: Add the interfaces** (no test — interface only; compile gate is Task 4)

Append to `internal/core/port/coupon.go`:

```go
// CouponReservationRepository persists ephemeral capacity holds (build-now: order-held).
type CouponReservationRepository interface {
	Create(ctx context.Context, r domain.CouponReservation) (domain.CouponReservation, error)
	FindByOrder(ctx context.Context, orgId, orderId string) ([]domain.CouponReservation, error)
	DeleteByOrder(ctx context.Context, orgId, orderId string) error
	CountLiveByCoupon(ctx context.Context, orgId, couponId string, now time.Time) (int, error)
	CountLiveByCode(ctx context.Context, orgId, couponCodeId string, now time.Time) (int, error)
	ExistsLiveForCustomer(ctx context.Context, orgId, couponId, customerId string, now time.Time) (bool, error)
	DeleteExpired(ctx context.Context, now time.Time) (int, error)
}
```

Add a lock method to the existing `CouponRepository` and `CouponCodeRepository` interfaces (same file):

```go
// In CouponRepository:
	FindByIdForUpdate(ctx context.Context, orgId, id string) (domain.Coupon, error) // SELECT ... FOR UPDATE
// In CouponCodeRepository:
	FindByCodeForUpdate(ctx context.Context, orgId, code string) (domain.CouponCode, error) // SELECT ... FOR UPDATE, case-insensitive
```

Ensure `time` is imported in `port/coupon.go`.

- [ ] **Step 2: Commit**

```bash
git add internal/core/port/coupon.go
git commit -m "feat(coupons): reservation repo port + coupon/code FOR UPDATE lookups"
```

---

## Task 4: GORM persistence — reservation repo + lock lookups

**Files:**
- Create: `internal/adapter/storage/postgresgorm/coupon_reservation_row.go`, `coupon_reservation_repo.go`
- Modify: `internal/adapter/storage/postgresgorm/coupon_repo.go`, `coupon_code_repo.go`

Follow the existing `postgresgorm/coupon_repo.go` + `coupon_row.go` pattern exactly (GORM row struct with `gorm:"column:…"` tags, `toDomain`/`fromDomain`, `dbFromCtx(ctx, r.db)`).

- [ ] **Step 1: Row** — `coupon_reservation_row.go`: struct mirroring the table columns (`org_id,id,coupon_id,coupon_code_id,customer_id,checkout_session_id,order_id,expires_at,created_at`), `TableName() "coupon_reservations"`, nullable text columns as `*string` written via `nilIfEmpty` (mirror `customerRow.ExternalId`), `toDomain`/`couponReservationRowFromDomain`.

- [ ] **Step 2: Repo** — `coupon_reservation_repo.go`, `NewCouponReservationRepo(db) port.CouponReservationRepository`:
  - `Create` → `dbFromCtx(ctx,r.db).Create(&row)`, return `FindById`-style reselect.
  - `FindByOrder` → `Where("org_id = ? AND order_id = ?", …)`.
  - `DeleteByOrder` → `Where("org_id = ? AND order_id = ?", …).Delete(&couponReservationRow{})`.
  - `CountLiveByCoupon` → `Model(&couponReservationRow{}).Where("org_id = ? AND coupon_id = ? AND expires_at > ?", orgId, couponId, now).Count(&n)`.
  - `CountLiveByCode` → same with `coupon_code_id = ?`.
  - `ExistsLiveForCustomer` → `Where("org_id = ? AND coupon_id = ? AND customer_id = ? AND expires_at > ?", …).Limit(1).Count`; return `n>0`.
  - `DeleteExpired` → `Where("expires_at <= ?", now).Delete(...)`, return `int(res.RowsAffected)`.

- [ ] **Step 3: Lock lookups** — in `coupon_repo.go` add `FindByIdForUpdate` = the existing `FindById` body plus `.Clauses(clause.Locking{Strength: "UPDATE"})`; in `coupon_code_repo.go` add `FindByCodeForUpdate` = `FindByCode` body (it upper-cases `code`) plus the same locking clause. Import `gorm.io/gorm/clause`.

- [ ] **Step 4: Build**

Run: `go build ./internal/adapter/storage/postgresgorm/` — Expected: OK.

- [ ] **Step 5: Commit**

```bash
git add internal/adapter/storage/postgresgorm/
git commit -m "feat(coupons): gorm reservation repo + FOR UPDATE lookups"
```

---

## Task 5: pgx persistence — reservation repo + lock lookups

**Files:**
- Create: `internal/adapter/storage/postgrespgx/coupon_reservation_row.go`, `coupon_reservation_repo.go`
- Modify: `internal/adapter/storage/postgrespgx/coupon_repo.go`, `coupon_code_repo.go`

Follow the existing `postgrespgx/coupon_repo.go` pattern (hand-written `$1` SQL, `dbFromCtx(ctx,r.pool)`, `scanInto`, `*string` for nullable columns via `nilIfEmpty`/`strOrEmpty`, `pgx.ErrNoRows → translateErr`).

- [ ] **Step 1: Row + repo** mirroring Task 4's methods, e.g.:
  - `const couponReservationColumns = "org_id, id, coupon_id, coupon_code_id, customer_id, checkout_session_id, order_id, expires_at, created_at"`.
  - `Create` → `INSERT INTO coupon_reservations (…) VALUES ($1..$9)`, reselect by `(org_id,id)`.
  - `CountLiveByCoupon` → `SELECT count(*) FROM coupon_reservations WHERE org_id=$1 AND coupon_id=$2 AND expires_at>$3`.
  - `CountLiveByCode`, `ExistsLiveForCustomer` (`SELECT EXISTS(SELECT 1 …)`), `FindByOrder`, `DeleteByOrder`, `DeleteExpired` (`DELETE … RETURNING` count via `tag.RowsAffected()`).

- [ ] **Step 2: Lock lookups** — `FindByIdForUpdate` / `FindByCodeForUpdate`: the existing single-row `SELECT … WHERE …` plus ` FOR UPDATE` appended to the SQL (no extra param).

- [ ] **Step 3: Build**

Run: `go build ./internal/adapter/storage/postgrespgx/` — Expected: OK.

- [ ] **Step 4: Commit**

```bash
git add internal/adapter/storage/postgrespgx/
git commit -m "feat(coupons): pgx reservation repo + FOR UPDATE lookups"
```

---

## Task 6: Conformance sub-test (both drivers)

**Files:**
- Modify: `internal/adapter/storage/storagetest/conformance.go`
- Modify: `internal/adapter/storage/postgrespgx/conformance_test.go`, `internal/adapter/storage/postgresgorm/conformance_test.go` (add `CouponReservation` to the factories)

- [ ] **Step 1: Add the port to `RepoSet`** in `conformance.go`: `CouponReservation port.CouponReservationRepository` and a `Coupon port.CouponRepository` field if not already present (the suite already builds coupons via `seedCoupon`-style helpers in the gorm package; add a coupon repo to the RepoSet for reservation FKs).

- [ ] **Step 2: Add `testCouponReservation`** to `RunConformance`:

```go
func testCouponReservation(t *testing.T, ctx context.Context, rs RepoSet) {
	orgId := seedOrg(t, ctx, rs)
	// seed a coupon (reservation FKs to it)
	coupon, err := rs.Coupon.Create(ctx, mustCoupon(t, orgId))
	require.NoError(t, err)
	now := now()
	r, err := domain.NewCouponReservation(domain.NewCouponReservationInput{
		OrgId: orgId, CouponId: coupon.Id, CustomerId: lib.GenerateId("cus"),
		OrderId: lib.GenerateId("ord"), ExpiresAt: now.Add(time.Hour),
	})
	require.NoError(t, err)
	_, err = rs.CouponReservation.Create(ctx, r)
	require.NoError(t, err)

	n, err := rs.CouponReservation.CountLiveByCoupon(ctx, orgId, coupon.Id, now)
	require.NoError(t, err)
	assert.Equal(t, 1, n, "live reservation counts")

	n, err = rs.CouponReservation.CountLiveByCoupon(ctx, orgId, coupon.Id, now.Add(2*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 0, n, "expired reservation does not count (lazy expiry)")

	require.NoError(t, rs.CouponReservation.DeleteByOrder(ctx, orgId, r.OrderId))
	n, err = rs.CouponReservation.CountLiveByCoupon(ctx, orgId, coupon.Id, now)
	require.NoError(t, err)
	assert.Equal(t, 0, n, "deleted on release")
}
```

Add a `mustCoupon(t, orgId)` helper building a valid `domain.Coupon` via `domain.NewCoupon` (percentage, `repeating`, `DurationInCycles: 2`, `PercentOff: decimal.NewFromInt(50)`), and register `t.Run("CouponReservation", …)` in `RunConformance`.

- [ ] **Step 3: Run both drivers**

Run: `go test -tags=integration -run 'TestConformance/CouponReservation' ./internal/adapter/storage/postgrespgx/ ./internal/adapter/storage/postgresgorm/` — Expected: PASS on both.

- [ ] **Step 4: Commit**

```bash
git add internal/adapter/storage/storagetest/ internal/adapter/storage/postgres*/conformance_test.go
git commit -m "test(coupons): cross-driver reservation conformance"
```

---

## Task 7: `CouponService` — reservation-aware gate + `Reserve`

**Files:**
- Modify: `internal/core/service/coupon.go` (struct, constructor, `gate`, new `Reserve`)
- Test: `internal/core/service/coupon_reserve_test.go`

- [ ] **Step 1: Failing test** — `coupon_reserve_test.go` with a fake `CouponReservationRepository` + the existing fakes. Assert:
  - `Reserve` of a valid code creates one reservation (fake records it) and returns no error.
  - When `coupon.MaxRedemptions == 1` and one live reservation already exists, `Reserve` is refused with `cap_reached`.
  - `Reserve` with an unknown code returns `code_not_found`.

(Mirror the structure of the existing `coupon_validate_test.go` fakes; the reservation fake's `CountLiveByCoupon` returns a settable count.)

- [ ] **Step 2: Run to verify it fails** — `go test ./internal/core/service/ -run TestCouponReserve -v` → FAIL.

- [ ] **Step 3: Implement.** Add `reservations port.CouponReservationRepository` to `CouponService` + `NewCouponService` (new last param). Make the cap checks in `gate` reservation-aware — change the three blocks to:

```go
	// coupon global cap: committed Discounts + live reservations
	if coupon.MaxRedemptions > 0 {
		n, err := s.discounts.CountByCoupon(ctx, orgId, coupon.Id)
		if err != nil { return gateResult{}, err }
		held, err := s.reservations.CountLiveByCoupon(ctx, orgId, coupon.Id, time.Now().UTC())
		if err != nil { return gateResult{}, err }
		if n+held >= coupon.MaxRedemptions {
			return gateResult{reason: "cap_reached"}, nil
		}
	}
	// once-per-customer: a Discount OR a live reservation
	if coupon.OncePerCustomer {
		n, err := s.discounts.CountByCouponAndCustomer(ctx, orgId, coupon.Id, customerId)
		if err != nil { return gateResult{}, err }
		if n == 0 {
			held, herr := s.reservations.ExistsLiveForCustomer(ctx, orgId, coupon.Id, customerId, time.Now().UTC())
			if herr != nil { return gateResult{}, herr }
			if held { n = 1 }
		}
		if n > 0 { return gateResult{reason: "already_used"}, nil }
	}
```

And the code-cap block (the `code != ""` branch):

```go
		held, herr := s.reservations.CountLiveByCode(ctx, orgId, cc.Id, time.Now().UTC())
		if herr != nil { return gateResult{}, herr }
		if cc.MaxRedemptions > 0 && cc.TimesRedeemed+held >= cc.MaxRedemptions {
			return gateResult{reason: "code_cap_reached"}, nil
		}
```

Add `Reserve` (atomic — lock the capacity owner under `FOR UPDATE`, then gate, then insert):

```go
type ReserveInput struct {
	OrgId, Code, CouponId, CustomerId, OrderId, Currency string
	Amount  int64
	HoldTTL time.Duration // 0 → default 30m
}

func (s *CouponService) Reserve(ctx context.Context, in ReserveInput) (domain.CouponReservation, error) {
	ttl := in.HoldTTL
	if ttl == 0 { ttl = 30 * time.Minute }
	var out domain.CouponReservation
	err := s.tx.RunInTx(ctx, func(ctx context.Context) error {
		// lock the capacity owner so the cap count + insert is atomic
		couponId := in.CouponId
		var codeId string
		if in.Code != "" {
			cc, err := s.codes.FindByCodeForUpdate(ctx, in.OrgId, in.Code)
			if err != nil {
				if errors.Is(err, port.ErrNotFound) {
					return lib.NewCustomError(lib.ValidationError, "coupon refused: code_not_found", nil)
				}
				return err
			}
			couponId, codeId = cc.CouponId, cc.Id
		}
		if _, err := s.coupons.FindByIdForUpdate(ctx, in.OrgId, couponId); err != nil {
			if errors.Is(err, port.ErrNotFound) {
				return lib.NewCustomError(lib.ValidationError, "coupon refused: code_not_found", nil)
			}
			return err
		}
		gate, err := s.gate(ctx, in.OrgId, in.Code, in.CouponId, in.CustomerId, in.Currency, in.Amount)
		if err != nil { return err }
		if gate.reason != "" {
			return lib.NewCustomError(refusalStatus(gate.reason), "coupon refused: "+gate.reason, nil)
		}
		ccId := ""
		if gate.hasCode { ccId = gate.code.Id }
		_ = codeId
		r, err := domain.NewCouponReservation(domain.NewCouponReservationInput{
			OrgId: in.OrgId, CouponId: gate.coupon.Id, CouponCodeId: ccId,
			CustomerId: in.CustomerId, OrderId: in.OrderId,
			ExpiresAt: time.Now().UTC().Add(ttl),
		})
		if err != nil { return err }
		out, err = s.reservations.Create(ctx, r)
		return err
	})
	if err != nil { return domain.CouponReservation{}, err }
	return out, nil
}

// refusalStatus maps a gate reason to an ApiError kind.
func refusalStatus(reason string) lib.ErrorCode {
	switch reason {
	case "cap_reached", "code_cap_reached", "already_used":
		return lib.ConflictError
	default:
		return lib.ValidationError
	}
}
```

(Confirm `lib.ErrorCode` is the type returned by `lib.ConflictError`/`lib.ValidationError`; if those are untyped consts, change `refusalStatus`'s return type to match.)

- [ ] **Step 4: Run** — `go test ./internal/core/service/ -run TestCouponReserve -v` → PASS. Then `go build ./...` (the new constructor param breaks `app.go` — fix in Task 11's wiring step or temporarily pass `nil`; defer the wire to Task 12).

- [ ] **Step 5: Commit**

```bash
git add internal/core/service/coupon.go internal/core/service/coupon_reserve_test.go
git commit -m "feat(coupons): reservation-aware gate + CouponService.Reserve"
```

---

## Task 8: `CouponService.Consume` + `Release`

**Files:**
- Modify: `internal/core/service/coupon.go`
- Test: `internal/core/service/coupon_consume_test.go`

- [ ] **Step 1: Failing test** — assert `Consume(order, sub, startCycle=0)` for an existing reservation: creates one `Discount` (fake discount repo records it with `SubscriptionId`/`StartCycle`), increments the code's `times_redeemed`, deletes the reservation; and is a no-op when no reservation exists. `Release(order)` deletes the reservation and is idempotent.

- [ ] **Step 2: Run to verify it fails** — FAIL.

- [ ] **Step 3: Implement**

```go
type ConsumeInput struct {
	OrgId, OrderId, SubscriptionId string
	StartCycle int
}

// Consume converts the order's reservation into a Discount on payment success.
// One reservation → one Discount → the given subscription. Never re-gates caps.
func (s *CouponService) Consume(ctx context.Context, in ConsumeInput) (domain.Discount, error) {
	var out domain.Discount
	err := s.tx.RunInTx(ctx, func(ctx context.Context) error {
		rs, err := s.reservations.FindByOrder(ctx, in.OrgId, in.OrderId)
		if err != nil { return err }
		if len(rs) == 0 { return nil } // no coupon on this order, or already consumed
		r := rs[0]
		discount, err := domain.NewDiscount(domain.NewDiscountInput{
			OrgId: in.OrgId, CouponId: r.CouponId, CouponCodeId: r.CouponCodeId,
			CustomerId: r.CustomerId, SubscriptionId: in.SubscriptionId, StartCycle: in.StartCycle,
		})
		if err != nil { return err }
		created, err := s.discounts.Create(ctx, discount)
		if err != nil {
			// idempotent under workflow retry: the (org,coupon,subscription) unique
			// index means it was already consumed — just clear the reservation.
			if errors.As(err, new(*lib.CustomError)) && isConflict(err) {
				return s.reservations.DeleteByOrder(ctx, in.OrgId, in.OrderId)
			}
			return err
		}
		if r.CouponCodeId != "" {
			if err := s.codes.IncrementRedeemed(ctx, in.OrgId, r.CouponCodeId); err != nil { return err }
		}
		if err := s.reservations.DeleteByOrder(ctx, in.OrgId, in.OrderId); err != nil { return err }
		out = created
		return nil
	})
	if err != nil { return domain.Discount{}, err }
	return out, nil
}

func (s *CouponService) Release(ctx context.Context, orgId, orderId string) error {
	return s.reservations.DeleteByOrder(ctx, orgId, orderId)
}
```

(`isConflict(err)` — reuse the codebase's conflict detection: the repos already map a unique violation to `lib.ConflictError`; assert on that — e.g. `lib.IsConflict(err)` if it exists, otherwise check the `*lib.CustomError` code. Confirm the helper name when implementing.)

- [ ] **Step 4: Run** — `go test ./internal/core/service/ -run 'TestCouponConsume|TestCouponRelease' -v` → PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/core/service/coupon.go internal/core/service/coupon_consume_test.go
git commit -m "feat(coupons): CouponService.Consume + Release"
```

---

## Task 9: Apply discounts in `BuildForBillingPeriod`

**Files:**
- Modify: `internal/core/domain/invoice_build.go` (add `ApplyDiscountTotals`)
- Modify: `internal/core/service/invoice.go` (deps + apply step)
- Test: `internal/core/domain/invoice_discount_apply_test.go`, `internal/core/service/invoice_discount_test.go`

- [ ] **Step 1: Domain method + test** — `invoice_discount_apply_test.go`:

```go
func TestInvoice_ApplyDiscountTotals(t *testing.T) {
	inv := Invoice{LineItems: []InvoiceLineItem{{Id: "l1", Total: 1000}, {Id: "l2", Total: 500}}}
	inv.recalculate()
	inv.ApplyDiscountTotals(map[string]int64{"l1": 250})
	assert.EqualValues(t, 250, inv.DiscountTotal)
	assert.EqualValues(t, 1250, inv.Total) // 1500 - 250
}
```

Add to `invoice_build.go`:

```go
// ApplyDiscountTotals sets each line's DiscountTotal from perLine (keyed by line
// id), then recomputes invoice totals. Lines absent from the map keep 0.
func (inv *Invoice) ApplyDiscountTotals(perLine map[string]int64) {
	for i := range inv.LineItems {
		if d, ok := perLine[inv.LineItems[i].Id]; ok {
			inv.LineItems[i].DiscountTotal = d
		}
	}
	inv.recalculate()
}
```

Run: `go test ./internal/core/domain/ -run TestInvoice_ApplyDiscountTotals -v` → PASS.

- [ ] **Step 2: Service deps** — add `discounts port.DiscountRepository` and `coupons port.CouponRepository` to `InvoiceService` + `NewInvoiceService` (new params, after `priceRepository`). Existing call sites get the new args in Task 12 wiring; in tests they may be `nil` (guarded below).

- [ ] **Step 3: Apply step.** In `BuildForBillingPeriod`, build a `productByPrice` map inside the items loop (`productByPrice[it.PriceId] = it.ProductId`), and after the loop (before persisting `inv`) call:

```go
	if err := s.applyDiscounts(ctx, sub, &inv, productByPrice); err != nil {
		return domain.Invoice{}, err
	}
```

Add the method:

```go
func (s *InvoiceService) applyDiscounts(ctx context.Context, sub domain.Subscription, inv *domain.Invoice, productByPrice map[string]string) error {
	if s.discounts == nil || s.coupons == nil {
		return nil // not wired (unit tests without discounts)
	}
	ds, err := s.discounts.ActiveForSubscription(ctx, sub.OrgId, sub.Id)
	if err != nil { return err }
	if len(ds) == 0 { return nil }
	applied := make([]domain.AppliedDiscount, 0, len(ds))
	for _, d := range ds {
		c, err := s.coupons.FindById(ctx, sub.OrgId, d.CouponId)
		if err != nil { return err }
		applied = append(applied, domain.AppliedDiscount{Discount: d, Coupon: c})
	}
	lines := make([]domain.DiscountableLine, 0, len(inv.LineItems))
	for _, l := range inv.LineItems {
		lines = append(lines, domain.DiscountableLine{LineId: l.Id, ProductId: productByPrice[l.PriceId], Total: l.Total})
	}
	inv.ApplyDiscountTotals(domain.ApplyDiscounts(lines, applied, sub.CyclesProcessed, sub.Currency))
	return nil
}
```

`inv` is currently a value (`inv := domain.NewInvoice(...)`); pass `&inv`.

- [ ] **Step 4: Service test** — `invoice_discount_test.go` (in-memory fakes for invoice/order/price/discount/coupon repos): seed a subscription with one fixed $100 line + an active `repeating(2)` 50% discount at `start_cycle=0`. Assert `BuildForBillingPeriod` at `CyclesProcessed=0` and `=1` produce `Total == 5000`, and at `=2` produces `Total == 10000`.

Run: `go test ./internal/core/service/ -run TestInvoiceDiscount -v` → PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/core/domain/invoice_build.go internal/core/domain/invoice_discount_apply_test.go internal/core/service/invoice.go internal/core/service/invoice_discount_test.go
git commit -m "feat(coupons): apply active discounts in BuildForBillingPeriod"
```

---

## Task 10: `coupon_code` on `CreateOrder` → reserve

**Files:**
- Modify: `internal/adapter/http/request.go` (add `CouponCode` to `CreateOrderRequest`)
- Modify: `internal/adapter/http/order_handler.go` (thread it into `port.CreateOrderInput`)
- Modify: `internal/core/port/order_input.go` (add `CouponCode` to `CreateOrderInput`)
- Modify: `internal/core/service/order.go` (`OrderService` gains `coupons *CouponService`; reserve in the create tx)
- Test: `internal/adapter/http/order_handler_test.go`

- [ ] **Step 1: DTO + input.** `CreateOrderRequest` gets `CouponCode string \`json:"coupon_code"\``. `order_handler.go` `CreateOrder` sets `CouponCode: input.CouponCode` on the `port.CreateOrderInput`. `CreateOrderInput` gets `CouponCode string`.

- [ ] **Step 2: Reserve in `CreateOrder`.** Add `coupons *CouponService` to `OrderService` + its constructor. Inside the existing `CreateOrder` transaction, **after the order + subscriptions are created**, if `input.CouponCode != ""`:

```go
		if input.CouponCode != "" {
			if _, err := s.coupons.Reserve(ctx, ReserveInput{
				OrgId:      input.OrgId,
				Code:       input.CouponCode,
				CustomerId: customerEntity.Id,
				OrderId:    orderId,
				Currency:   input.Currency,
				Amount:     order.Total, // cart subtotal for MinimumAmount
			}); err != nil {
				return err // refusal rolls back the whole order
			}
		}
```

(Place it where `orderId`, `customerEntity`, and `order.Total` are in scope, before the tx closure returns nil. Confirm exact local names.)

- [ ] **Step 3: Handler test.** In `order_handler_test.go`, drive `POST /api/orders` with a `coupon_code` for an exhausted coupon (fake CouponService returns a `cap_reached` `ApiError`) → assert the response is the conflict `ApiError` and no order persists. With a valid code → 200 and a reservation recorded.

Run: `go test ./internal/adapter/http/ -run TestCreateOrder -v` → PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/adapter/http/request.go internal/adapter/http/order_handler.go internal/core/port/order_input.go internal/core/service/order.go internal/adapter/http/order_handler_test.go
git commit -m "feat(coupons): reserve a coupon at CreateOrder; refusal fails the order"
```

---

## Task 11: `Consume` in `CompleteOrder`

**Files:**
- Modify: `internal/core/service/order.go`

- [ ] **Step 1: Implement.** Inside `CompleteOrder`'s tx, after subscriptions are activated (the `activated` slice is built), convert the reservation against the matched subscription. Build-now (single subscription): the first activated subscription is the target.

```go
		if len(activated) > 0 {
			if _, err := s.coupons.Consume(ctx, ConsumeInput{
				OrgId:          input.OrgId,
				OrderId:        order.Id,
				SubscriptionId: activated[0].Id,
				StartCycle:     activated[0].CyclesProcessed, // 0 at activation
			}); err != nil {
				return err
			}
		}
```

(`Consume` is a no-op when the order has no reservation, so this is safe for coupon-less orders. Multi-subscription targeting — "the first subscription holding a line the coupon targets" — is a later refinement; single-sub is the build-now case.)

- [ ] **Step 2: Build + run the order service tests**

Run: `go build ./... && go test ./internal/core/service/ -run TestOrder -v` — Expected: OK/PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/core/service/order.go
git commit -m "feat(coupons): convert reservation to Discount on CompleteOrder"
```

---

## Task 12: Wire everything in `app.go`

**Files:**
- Modify: `internal/config/app.go`, `internal/config/repos.go`

- [ ] **Step 1: Build the reservation repo** in both `repoSet` builders (`newGormRepoSet` / `newPgxRepoSet` in `repos.go`): add `couponReservation port.CouponReservationRepository` to `repoSet`, set from `postgresgorm.NewCouponReservationRepo(db)` / `postgrespgx.NewCouponReservationRepo(pool)`.

- [ ] **Step 2: Wire services** in `app.go`:
  - `couponService := service.NewCouponService(repos.coupon, repos.couponCode, repos.discount, repos.priorPaymentChecker, txManager, logger, repos.couponReservation)` (new last arg).
  - `invoiceService := service.NewInvoiceService(repos.invoice, repos.order, repos.price, usageService, txManager, logger, repos.discount, repos.coupon)` (new last two args).
  - Pass `couponService` into `NewOrderService(...)` (new dependency).

  Confirm the local `repoSet` field names (the storage work named them `coupon`, `couponCode`, `discount`, `priorPaymentChecker`, `invoice`, `order`, `price`).

- [ ] **Step 2b: Cleanup sweep (optional but cheap):** register a periodic `repos.couponReservation.DeleteExpired(ctx, time.Now())` on the cron scheduler already in `app.go` (e.g. every 10m). Lazy expiry means this is housekeeping; skip if it complicates wiring.

- [ ] **Step 3: Build + boot**

Run: `go build ./... && go vet ./...` — Expected: OK. Then `DB_DRIVER=gorm` and `DB_DRIVER=pgx` both boot (Task 13 exercises pgx via the e2e).

- [ ] **Step 4: Commit**

```bash
git add internal/config/
git commit -m "feat(coupons): wire reservation repo + discount-aware invoice + order coupon"
```

---

## Task 13: End-to-end — the live scenario

**Files:**
- Create: `internal/adapter/storage/postgresgorm/coupon_billing_e2e_test.go` (`//go:build integration`)

This is the acceptance test: a $100/cycle subscription with a 50%-off `repeating(2)` coupon bills `2×$50 + 3×$100`. It runs through the **services** (not the live server) against the testcontainer, mirroring the existing `billing_lifecycle_e2e_test.go` harness (which builds the service graph from `testDB(t)`).

- [ ] **Step 1: Write the test.** Using the existing e2e harness in `postgresgorm` (see `billing_charge_e2e_test.go` for how it constructs `OrderService`/`SubscriptionService`/`InvoiceService`/`CouponService` from `testDB(t)` and the memory gateway):
  1. Seed org, a $100 `subscription` price (`unit_price: 10000`, `billing_interval: minute`, qty 1), a customer + payment method.
  2. `coupon, _ := CouponService.Create(...)` — 50% `percentage`, `repeating`, `DurationInCycles: 2`.
  3. `coupon_code` via `CouponService.CreateCode(coupon.Id, "LAUNCH50")`.
  4. `CreateOrder` with the cart + `CouponCode: "LAUNCH50"` → assert a reservation exists for the order.
  5. `CompleteOrder` (no caller payment) → assert: order completed, subscription active, **one `Discount`** exists for the subscription (`start_cycle=0`), the reservation is gone, `code.times_redeemed == 1`.
  6. Drive 5 billing cycles by invoking `SubscriptionService.ChargeForBillingPeriod` directly for `CyclesProcessed = 0..4` (advance the sub between charges as the billing flow does), or call the cycle helper the existing billing e2e uses.
  7. Assert the five `payments` are `5000, 5000, 10000, 10000, 10000` in order, the subscription ends `completed` with `CyclesProcessed == 5`, and `invoices` totals match.

- [ ] **Step 2: Run**

Run: `go test -tags=integration -run TestCouponBillingE2E -v ./internal/adapter/storage/postgresgorm/` — Expected: PASS.

- [ ] **Step 3: Run the whole integration suite (regression, both drivers)**

Run: `make test-integration` — Expected: PASS (the new conformance reservation test runs under pgx too).

- [ ] **Step 4: Commit**

```bash
git add internal/adapter/storage/postgresgorm/coupon_billing_e2e_test.go
git commit -m "test(coupons): e2e 50%-off repeating(2) → 2x50 + 3x100"
```

---

## Task 14: Docs + finalize

- [ ] **Step 1:** Note the `coupon_code` field on `POST /api/orders` is now in the generated spec — regenerate the contract: `make openapi` and commit `docs/openapi.yml` if it changed.
- [ ] **Step 2:** Add a one-line mention to `AGENTS.md` under coupons (reservation → discount → billing) if the coupons area is documented there.
- [ ] **Step 3: Commit**

```bash
git add docs/ AGENTS.md
git commit -m "docs(coupons): regenerate spec + note reservation flow"
```

---

## Notes for the implementer

- **TDD order matters:** domain → port → both repos → conformance → service → billing → order hooks → wiring → e2e. Each task builds on the previous; run `go build ./...` after the service-signature changes (Tasks 7/9/12) since new constructor params ripple into `app.go`.
- **Both storage drivers** must stay at parity — every repo method lands in `postgresgorm` and `postgrespgx`, verified by `storagetest`. Don't skip the pgx side.
- **`isConflict` / `lib.ErrorCode` / conflict helper names:** confirm the exact names in `internal/lib/` when you reach Tasks 7–8 (the codebase already maps unique violations to a conflict `ApiError`; reuse that, don't invent one).
- **`CompleteOrder` without a payment** is the build-now cycle-0 path; the full "subscription owns its first invoice at activation" + checkout modes (spec §6, §15) are deliberately out of scope here.
