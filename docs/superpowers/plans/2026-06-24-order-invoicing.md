# Order Invoicing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** An order produces **one combined invoice** (subscription first period + one-time lines) at payment confirmation — or `open` up-front when `upfront_invoice` is set; discounts become order-owned; the invoice gets a configurable `reference`.

**Architecture:** Build a single `InvoiceService.BuildForOrder` that mirrors `BuildForBillingPeriod` but covers an order's first bill. Discounts move to order-ownership (`order_id` always set). `CompleteOrder` orchestrates: consume reservation → build combined invoice → link/settle payment. Reference is formatted from a settings-backed `InvoiceSettings`. Both storage drivers kept at parity via the `storagetest` conformance suite. No workflow-engine change (activation owns cycle 0; recurring engine bills ≥1, idempotent).

**Tech Stack:** Go 1.24, Fuego, GORM + jackc/pgx/v5 (both at parity), Goose, testcontainers.

**Spec:** `docs/superpowers/specs/2026-06-24-build-for-order-onceoff-invoice-design.md`

**Conventions:** every storage change lands in BOTH `postgresgorm` and `postgrespgx` and is exercised by `internal/adapter/storage/storagetest`. Migrations are Goose (`TIMESTAMP(3)`, quoted idents); latest is `00006`, so new files are `00007…`. Integration tests are `//go:build integration` and spin a throwaway testcontainer — never touch the dev DB.

---

## File structure

- **Migrations:** `00007_orders_config.sql`, `00008_invoices_reference_and_nullable_subscription.sql`.
- **Domain:** `order.go` (`OrderConfig`), `invoice.go` (`Reference`), `invoice_settings.go` (new — `InvoiceSettings` + `FormatReference`), `discount.go` (`NewDiscount` rule).
- **Port:** `repository.go` (`InvoiceRepository.FindOrderInvoice`), `order_input.go` (`CreateOrderInput.Config`, `CreateOrderResult.Invoice`).
- **Service:** `invoice.go` (`BuildForOrder`, reference formatting, `SettingRepository` dep), `coupon.go` (`Consume` order-always), `order.go` (`CreateOrder` config + upfront invoice; `CompleteOrder` orchestration; `*InvoiceService` dep).
- **Storage (×2 drivers):** `order_row.go`/`order_repo.go` (config), `invoice_row.go`/`invoice_repo.go` (reference, nullable subscription_id, `FindOrderInvoice`), `discount_repo.go` (`ActiveForOrder`).
- **HTTP:** `order_handler.go`/`request.go` (`upfront_invoice`, invoice in response).
- **Wiring:** `config/app.go`, `config/repos.go`.
- **Tests:** `storagetest/conformance.go`, service `*_test.go`, http `*_test.go`.

---

## Phase A — Storage & domain foundation

### Task 1: Migrations

**Files:** Create `schemas/app/migrations/00007_orders_config.sql`, `schemas/app/migrations/00008_invoices_reference_and_nullable_subscription.sql`

- [ ] **Step 1: `00007_orders_config.sql`**
```sql
-- +goose Up
ALTER TABLE "orders" ADD COLUMN "config" JSONB;
-- +goose Down
ALTER TABLE "orders" DROP COLUMN "config";
```

- [ ] **Step 2: `00008_invoices_reference_and_nullable_subscription.sql`**
```sql
-- +goose Up
ALTER TABLE "invoices" ADD COLUMN "reference" TEXT NOT NULL DEFAULT '';
CREATE INDEX "invoices_org_id_reference_idx" ON "invoices" ("org_id", "reference") WHERE "reference" <> '';
ALTER TABLE "invoices" ALTER COLUMN "subscription_id" DROP NOT NULL;
-- +goose Down
ALTER TABLE "invoices" ALTER COLUMN "subscription_id" SET NOT NULL;
DROP INDEX "invoices_org_id_reference_idx";
ALTER TABLE "invoices" DROP COLUMN "reference";
```

- [ ] **Step 3:** `make db-migrate-all` (best-effort; skip if no local DB — the testcontainer applies all migrations). Commit:
```bash
git add schemas/app/migrations/00007_orders_config.sql schemas/app/migrations/00008_invoices_reference_and_nullable_subscription.sql
git commit -m "feat(db): orders.config; invoices.reference + nullable subscription_id"
```

---

### Task 2: `OrderConfig` domain + order_row mapping (both drivers) + conformance

**Files:** `internal/core/domain/order.go`; `postgresgorm/order_row.go`; `postgrespgx/order_row.go` + `order_repo.go`; `storagetest/conformance.go`

- [ ] **Step 1: domain** — add to `order.go`:
```go
type OrderConfig struct {
	UpfrontInvoice bool `json:"upfront_invoice"`
}
```
Add `Config OrderConfig` to `Order` (after `PaymentSession`).

- [ ] **Step 2: gorm row** — in `order_row.go` add `Config domain.OrderConfig `gorm:"column:config;serializer:json"`` and map it in `toDomain`/`orderRowFromDomain`. (A typed struct value is nil-safe for `Save`.)

- [ ] **Step 3: pgx row** — in `order_row.go` add `Config jsonCol[domain.OrderConfig]`; append `config` to `orderColumns` (last) and `&r.Config` to `scanInto` (last); map `.V` / `newJSON(o.Config)`. In `order_repo.go` add the `$N` placeholder to the INSERT (and arg) — and to the UPDATE SET if order Update should persist config (it should: add `config=$N`). Double-check every placeholder number.

- [ ] **Step 4: conformance** — extend `testCartOrderItem` (or add a focused case): create an order with `Config: domain.OrderConfig{UpfrontInvoice: true}`, `FindById`, assert it round-trips on both drivers.

- [ ] **Step 5:** `go build ./...` then `go test -tags integration ./internal/adapter/storage/... -run TestConformance`. Commit:
```bash
git commit -am "feat(orders): OrderConfig persisted as orders.config (both drivers)"
```

---

### Task 3: `Invoice.Reference` + nullable subscription_id mapping (both drivers) + conformance

**Files:** `internal/core/domain/invoice.go`; `postgresgorm/invoice_row.go`; `postgrespgx/invoice_row.go` + `invoice_repo.go`; `storagetest/conformance.go`

- [ ] **Step 1: domain** — add `Reference string` to `Invoice` (after `Number`).

- [ ] **Step 2: gorm row** — add `Reference string `gorm:"column:reference"`` to the invoice row; map both ways. Confirm `subscription_id` maps via a nil-safe optional string (it can now be empty → must write NULL, not `""` — follow the repo's existing optional-id convention; check how `coupon_code_id`/other nullable ids are written).

- [ ] **Step 3: pgx row** — add `reference` to the invoice columns + scan + mapping; make `subscription_id` read/write nullable (`*string` / `nilIfEmpty` per existing pattern). Update INSERT/UPDATE placeholders.

- [ ] **Step 4: conformance** — in `testInvoice`: round-trip an invoice with a `Reference` and with an **empty `SubscriptionId`** (asserting it reads back `""` and that an order-only invoice persists). Add a search assertion: find by `(org_id, reference)` returns it (via a repo method if one exists, else a direct read in the driver test).

- [ ] **Step 5:** build + `TestConformance` both drivers. Commit:
```bash
git commit -am "feat(invoice): Reference column + nullable subscription_id (both drivers)"
```

---

### Task 4: `InvoiceRepository.FindOrderInvoice` (both drivers) + conformance

**Files:** `internal/core/port/repository.go`; both `invoice_repo.go`; `storagetest/conformance.go`

- [ ] **Step 1: port** — add to `InvoiceRepository`:
```go
// FindOrderInvoice returns the order's combined cycle-0 invoice (order_id set),
// or port.ErrNotFound. The build-idempotency guard for an order's invoice.
FindOrderInvoice(ctx context.Context, orgId, orderId string) (domain.Invoice, error)
```

- [ ] **Step 2: gorm** — `WHERE org_id = ? AND order_id = ? AND cycle = 0` first row (with line items, like `FindById`), mapping `port.ErrNotFound` on no rows.

- [ ] **Step 3: pgx** — equivalent SQL; `pgx.ErrNoRows` → `port.ErrNotFound`.

- [ ] **Step 4: conformance** — add `IdempotencyStore`-style case in `testInvoice`: create an order invoice (`order_id` set, `cycle 0`, no subscription) → `FindOrderInvoice` returns it; for an order with only a subscription cycle invoice (`cycle 1`), `FindOrderInvoice` → `ErrNotFound`.

- [ ] **Step 5:** build + conformance both drivers. Commit:
```bash
git commit -am "feat(invoice): FindOrderInvoice idempotency lookup (both drivers)"
```

---

### Task 5: Order-owned discounts — `NewDiscount` + `ActiveForOrder`

**Files:** `internal/core/domain/discount.go` (+ `discount_test.go`); both `discount_repo.go`; `storagetest/conformance.go`

- [ ] **Step 1 (TDD): domain test** — update `discount_test.go`: order-only ✓; **order+subscription now ✓**; missing order ✗; subscription-only (no order) ✗. Run → fails.

- [ ] **Step 2: domain** — in `NewDiscount`, require `OrderId` (with org/coupon/customer); make `SubscriptionId` optional; delete the `hasSub == hasOrder` check. Update the `Discount.SubscriptionId`/`OrderId` struct comment to "order_id always set; subscription_id set when targeting a subscription". Run test → passes.

- [ ] **Step 3: repos** — change `ActiveForOrder` on BOTH drivers to `WHERE order_id = ? AND subscription_id IS NULL AND status = 'active'` (gorm: add `.Where("subscription_id IS NULL")`; pgx: add to the SQL). `ActiveForSubscription` unchanged.

- [ ] **Step 4: conformance** — in `testCoupon`/discount cases: a discount with `order_id` + `subscription_id` set is returned by `ActiveForSubscription` but NOT `ActiveForOrder`; an order-only discount (sub-null) is returned by `ActiveForOrder`.

- [ ] **Step 5:** build + `make test` (domain) + conformance both drivers. Commit:
```bash
git commit -am "feat(discount): order-owned (order_id always, subscription_id optional); ActiveForOrder sub-null"
```

---

### Task 6: `InvoiceSettings` domain + reference formatter

**Files:** Create `internal/core/domain/invoice_settings.go` (+ `invoice_settings_test.go`)

- [ ] **Step 1 (TDD): test** — `FormatReference`:
```go
func TestFormatReference(t *testing.T) {
	s := InvoiceSettings{Prefix: "INV-", Padding: 6}
	assert.Equal(t, "INV-000042", s.FormatReference(42))
	// defaults when zero-value
	assert.Equal(t, "INV-000042", InvoiceSettings{}.WithDefaults().FormatReference(42))
}
```

- [ ] **Step 2: impl**
```go
package domain

import "fmt"

type InvoiceSettings struct {
	Prefix  string `json:"prefix"`
	Padding int    `json:"padding"`
}

func (s InvoiceSettings) WithDefaults() InvoiceSettings {
	if s.Prefix == "" { s.Prefix = "INV-" }
	if s.Padding <= 0 { s.Padding = 6 }
	return s
}

func (s InvoiceSettings) FormatReference(number int64) string {
	s = s.WithDefaults()
	return fmt.Sprintf("%s%0*d", s.Prefix, s.Padding, number)
}
```

- [ ] **Step 3:** `go test ./internal/core/domain/ -run FormatReference`. Commit:
```bash
git commit -am "feat(invoice): InvoiceSettings + FormatReference"
```

---

## Phase B — Service layer

### Task 7: `CouponService.Consume` sets the order always

**Files:** `internal/core/service/coupon.go` (+ `coupon_test.go`)

- [ ] **Step 1 (TDD):** add/extend a `Consume` test: with `SubscriptionId` empty, the created `Discount` has `OrderId` set and `SubscriptionId` empty (order-owned); with `SubscriptionId` set, both are set. Run → fails (current code only sets subscription).

- [ ] **Step 2: impl** — in `Consume`, build `NewDiscount` with `OrderId: in.OrderId` always, plus `SubscriptionId: in.SubscriptionId` (already in `ConsumeInput`). No signature change.

- [ ] **Step 3:** `make test` (service). Commit:
```bash
git commit -am "feat(coupon): Consume creates an order-owned Discount (optionally subscription-targeted)"
```

---

### Task 8: `InvoiceService` — `SettingRepository` dep + reference on `BuildForBillingPeriod`

**Files:** `internal/core/service/invoice.go` (+ `invoice_test.go`); `internal/config/app.go`

- [ ] **Step 1:** add `settings port.SettingRepository` to `InvoiceService` + constructor param (wire in `app.go`). Add a helper:
```go
// invoiceSettings loads the org's InvoiceSettings from the settings store
// (parent = orgId, id = "invoice"); returns defaults when unset/malformed.
func (s *InvoiceService) invoiceSettings(ctx context.Context, orgId string) domain.InvoiceSettings {
	var out domain.InvoiceSettings
	if s.settings != nil {
		if st, err := s.settings.FindById(ctx, orgId, orgId, "invoice"); err == nil {
			_ = json.Unmarshal([]byte(st.Value), &out)
		}
	}
	return out.WithDefaults()
}
```

- [ ] **Step 2:** in `BuildForBillingPeriod`, after `inv.Number = NextInvoiceNumber(...)`, set `inv.Reference = s.invoiceSettings(ctx, sub.OrgId).FormatReference(inv.Number)`.

- [ ] **Step 3 (TDD):** extend an invoice service test to assert a built invoice has `Reference == "INV-000001"` (counter starts at 1) with default settings.

- [ ] **Step 4:** `make test`. Commit:
```bash
git commit -am "feat(invoice): format reference from InvoiceSettings on build"
```

---

### Task 9: `InvoiceService.BuildForOrder` (the combined invoice)

**Files:** `internal/core/service/invoice.go` (+ `invoice_test.go`)

- [ ] **Step 1: shared helper** — factor the per-line build (base/usage line from a price + qty) and discount application out of `BuildForBillingPeriod` into a helper both builders call (DRY). Keep `BuildForBillingPeriod` behaviour identical.

- [ ] **Step 2: `BuildForOrder`**
```go
// BuildForOrder builds (or returns) the order's combined cycle-0 invoice: each
// subscription's first-period line(s) + every one-time line, with the order's
// discount applied. Idempotent on the order. Status is set by the caller.
func (s *InvoiceService) BuildForOrder(ctx context.Context, order domain.Order) (domain.Invoice, error)
```
Logic:
1. `existing, err := s.invoiceRepository.FindOrderInvoice(ctx, order.OrgId, order.Id)`; if found return it; if not `ErrNotFound`, return err.
2. `items := FindOrderItemsByOrderId(order.OrgId, order.Id)`. If none → return `domain.Invoice{}, port.ErrNotFound`.
3. Resolve the order's subscriptions (`subscriptionRepository.FindByOrderId`) — used for first-period dates + linkage. Build a `domain.Invoice` with `OrgId/OrderId/CustomerId/Currency`, `Cycle: 0`, period = the single subscription's first period if present else order time. For each item: recurring → base/usage line over the first period (reuse the shared helper; usage will be ~0 at cycle 0 and that's correct); one-time → base line.
4. **Subscription linkage:** if exactly one subscription on the order, set `inv.SubscriptionId = sub.Id`.
5. **Discount:** resolve via `ActiveForOrder` (committed) — and for the `open`/pre-payment path also accept discounts from the order's live reservation's coupon (helper param). Apply with `cycle = 0`.
6. In the tx: `inv.Number = NextInvoiceNumber`; `inv.Reference = invoiceSettings.FormatReference(...)`; `invoiceRepository.Create`.

`InvoiceService` will need the `subscriptionRepository` (add the dep + wire) for first-period dates + linkage.

- [ ] **Step 3 (TDD):** service tests (no DB / fakes where possible, else integration): a mixed order ($100/mo sub + $50 one-off) → one invoice, two lines, total $150, `SubscriptionId` set, `Cycle 0`; a pure one-time order with an order-discount → discounted total; idempotent (second call returns the same invoice).

- [ ] **Step 4:** `make test` (+ integration if the test needs a DB). Commit:
```bash
git commit -am "feat(invoice): BuildForOrder — combined order invoice with discount"
```

---

### Task 10: `OrderService.CreateOrder` — persist `Config`, optional upfront invoice

**Files:** `internal/core/port/order_input.go`; `internal/core/service/order.go` (+ test); `internal/config/app.go`

- [ ] **Step 1: port** — add `Config domain.OrderConfig` to `CreateOrderInput`; add `Invoice *domain.Invoice` (optional) to `CreateOrderResult`.

- [ ] **Step 2:** `OrderService` gains `invoiceService *InvoiceService` (constructor + `app.go` wiring — see Task 13). `CreateOrder` sets `Config: input.Config` on the created order.

- [ ] **Step 3:** after the order + subscriptions + reservation are created, if `input.Config.UpfrontInvoice`: `inv, err := s.invoiceService.BuildForOrder(ctx, order)` (open status — `BuildForOrder` builds `draft`/`open`; mark `open`), include it in `CreateOrderResult.Invoice`. (No payment here — upfront invoice is `open`.) Keep this inside the existing create flow's transaction boundary where the order is written.

- [ ] **Step 4 (TDD):** service test: `CreateOrder` with `UpfrontInvoice:true` returns an `open` invoice covering the lines; with `false` returns no invoice.

- [ ] **Step 5:** `make test`. Commit:
```bash
git commit -am "feat(orders): persist Config; build open invoice when upfront_invoice"
```

---

### Task 11: `CompleteOrder` orchestration

**Files:** `internal/core/service/order.go` (+ test)

- [ ] **Step 1:** in the `RunInTx` body, replace the coupon-consume block so it runs for **all** orders (drop `len(activated) > 0`): subscriptions exist → `Consume{OrgId, OrderId, SubscriptionId: activated[0].Id, StartCycle: 0}`; else → `Consume{OrgId, OrderId, StartCycle: 0}`. Before the invoice build.

- [ ] **Step 2: first invoice = combined invoice.** Before `SetActive` increments cycles, build the combined invoice once for the order: `inv, err := s.invoiceService.BuildForOrder(ctx, order)` (returns the existing one if `upfront_invoice` already built it). Then:
  - if a payment is supplied (`input.Payment.Amount > 0`): create the `Payment` with `InvoiceId = inv.Id`; `MarkOpen` + `MarkSettled` the invoice.
  - link the subscription activation to it; `SetActive`; update.
  (For a pure one-time order, same: build invoice, create a one-time `Payment` with `InvoiceId`, mark settled.)

- [ ] **Step 3:** ensure cycle-0 is not rebuilt by the engine — covered by `BuildForBillingPeriod` idempotency + the combined invoice carrying `subscription_id`+`cycle 0` (no code change in engines; assert in Task 14).

- [ ] **Step 4 (TDD):** service test: mixed order `direct` payment → one combined `paid` invoice `$150`, `Payment.InvoiceId` set, sub `CyclesProcessed == 1`; pure once-off + coupon → one paid discounted invoice, reservation consumed (no orphan); re-complete is idempotent (no second invoice).

- [ ] **Step 5:** `make test`. Commit:
```bash
git commit -am "feat(orders): CompleteOrder builds+settles the combined invoice; consumes reservation for all orders"
```

---

## Phase C — HTTP + wiring

### Task 12: HTTP — `upfront_invoice` in, invoice out

**Files:** `internal/adapter/http/request.go`, `internal/adapter/http/order_handler.go` (+ test)

- [ ] **Step 1:** add `UpfrontInvoice bool `json:"upfront_invoice"`` to `CreateOrderRequest`; thread it into `port.CreateOrderInput{Config: domain.OrderConfig{UpfrontInvoice: input.UpfrontInvoice}}`.

- [ ] **Step 2:** add an optional `Invoice` to `CreateOrderResponse`:
```go
type CreateOrderInvoice struct {
	Id  string `json:"id"`
	Url string `json:"url"`
}
// in CreateOrderResponse: Invoice *CreateOrderInvoice `json:"invoice,omitempty"`
```
Populate from `rsp.Invoice` when present (`Url: ""` placeholder).

- [ ] **Step 3 (TDD):** http test (existing order harness): `upfront_invoice:true` → response carries `invoice {id, url:""}`; `false` → no `invoice` field.

- [ ] **Step 4:** `go test ./internal/adapter/http/ -run Order`. Commit:
```bash
git commit -am "feat(http): upfront_invoice request flag; invoice {id,url} in create response"
```

---

### Task 13: Wiring (`app.go`, `repos.go`)

**Files:** `internal/config/app.go`

- [ ] **Step 1:** construct `InvoiceService` with the `SettingRepository` and `SubscriptionRepository` deps (added in Tasks 8–9). Pass the `*InvoiceService` into `NewOrderService`. Ensure construction order: `InvoiceService` before `OrderService`.

- [ ] **Step 2:** `go build ./...` + `make test`. Commit:
```bash
git commit -am "feat(config): wire SettingRepository/SubscriptionRepository into InvoiceService; InvoiceService into OrderService"
```

---

## Phase D — Verification

### Task 14: Engine-parity + e2e integration

**Files:** an integration e2e (extend `postgresgorm` e2e or `storagetest`)

- [ ] **Step 1:** e2e — mixed cart ($100/mo + $50) with a coupon, `direct` payment → exactly **one** combined `paid` invoice with the discounted total, `reference` formatted, `Payment.InvoiceId` set; the subscription's next engine charge is **cycle 1** and does NOT rebuild/recharge cycle 0 (assert `FindBySubscriptionCycle(sub,0)` returns the combined invoice and the engine produces cycle 1 next).
- [ ] **Step 2:** e2e — usage subscription (base + metered, no one-off): activation invoice = base only (usage 0 at cycle 0); cycle 1 invoice includes the month's usage (existing metering path unaffected).
- [ ] **Step 3:** e2e — `upfront_invoice` order → `open` invoice at create → `/pay`/complete settles it to `paid`.

- [ ] **Step 4:** `make ci` then `make test-integration` — green across all packages, both drivers. Commit:
```bash
git commit -am "test(e2e): combined order invoicing, usage sub, upfront invoice; parity"
```

---

## Self-review notes
- **Spec coverage:** §2 OrderConfig → Tasks 1,2,10,12; §3/§3.1 number+reference+settings → Tasks 1,3,6,8; §4 order-owned discounts → Task 5; §5 BuildForOrder → Task 9; §6 build/settle timing → Tasks 10,11; §7 first invoice at activation → Task 11; §8 orchestration → Task 11; §10 data model → Tasks 1–4; §11 placement → all; §12 testing → Tasks 2–14.
- **Parity:** every storage change (Tasks 2–5) lands in both drivers and is gated by `storagetest`.
- **Usage correctness:** cycle-0 usage is naturally ~0; metering for cycles ≥1 is untouched (Task 14 asserts).
- **Engine parity:** no engine code changed; idempotency + `CyclesProcessed=1` keep cycle 0 owned by activation (Task 14 asserts both engines).
- **Type consistency:** `OrderConfig`, `InvoiceSettings`, `Invoice.Reference`, `FindOrderInvoice`, `BuildForOrder` defined once (Tasks 1–6/9) and used verbatim downstream.
