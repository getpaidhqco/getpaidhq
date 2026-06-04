# Invoice-centric billing

**Date:** 2026-06-04
**Goal:** Make every billing run produce an itemized `Invoice` and have the `Payment` settle that invoice's total, instead of charging a flat `subscription.Amount`. Ships value for today's fixed-price subscriptions on its own — no metering — and is the billing model the usage-metering spec builds on.

**Authoritative decisions:** `docs/adr/0002-invoice-centric-billing.md` (invoice-centric billing; subscription carries no charge amount), `docs/adr/0001-invoice-line-item-decimal-quantity.md` (line-item decimal quantity / decimal unit amount), and `docs/adr/0004-decimal-for-fractional-quantities.md` (`decimal.Decimal` = `github.com/shopspring/decimal`, a **new dependency**; money stays `int64` cents). Glossary terms — Order, Subscription, Invoice, Payment, Invoice preview — are pinned in `CONTEXT.md` and used here exactly.

> **Code references** (`subscription.go:NNN` etc.) are indicative and will drift — **locate by symbol** (`ChargeForBillingPeriod`, `HandleSubscriptionChargeSuccess`, `NewSubscriptionFromOrderItem`, the `Amount` field), not by line number.

## Problem

Today a renewal charges a single flat amount stored on the subscription. `SubscriptionService.ChargeForBillingPeriod` (`internal/core/service/subscription.go:652`) re-reads the subscription and calls `gw.ChargePayment` with `Amount: subscription.Amount` (`subscription.go:683`). On success, `HandleSubscriptionChargeSuccess` records a `Payment` whose `NetAmount` is `subscription.Amount` and advances `TotalRevenue += subscription.Amount` (`subscription.go:466`, `:481`); the failure path mirrors it (`subscription.go:577`). `Subscription.Amount` is seeded from the linked price at creation — `NewSubscriptionFromOrderItem` sets `Amount: item.Price.UnitPrice` (`internal/core/domain/subscription.go:386`).

There is **no per-cycle record of what was owed**: only a `Payment` (one PSP attempt) exists, with no itemization and no link to the pricing that produced it. This blocks variable billing entirely — a usage subscription's period amount is variable and unknowable up front, so a single stored `Amount` is a fiction for it. There is also nowhere to attach a usage line. The flat-charge flow is documented in `docs/workflows/billing-cycle.md`.

## Decision

Introduce **`Invoice`** and **`InvoiceLineItem`** as new operational entities. The billing chain becomes:

> **Order → Subscription → Invoice (one per cycle) → Payment settles the Invoice total.**

On **every** billing run, for **all** subscriptions, `billing-cycle`:

1. **Builds an `Invoice`** for the subscription's current period — line items derived from the subscription's linked `Price` (reached via `OrderItem.Price`). For a fixed subscription that is exactly one base line: `Price.UnitPrice × quantity`.
2. **Totals** the invoice (sum of line `Total`s, `int64` cents).
3. **Creates a `Payment`** that attempts to settle that total. The `Payment` is still the record of a PSP attempt; a new `Payment.InvoiceId` links it to the invoice it settles.

Consequences encoded here (settled — see ADR 0002):

- **`Subscription` no longer stores a charge amount.** The `Amount` field is removed. Pricing authority is the linked `Price`. Historic actuals (`TotalRevenue`, `CyclesProcessed`) stay. A "base MRR" figure, if ever needed, is derived on demand from the linked price — never stored.
- **Engine parity.** The build→total→settle flow must produce the same observable outcome on both the Hatchet and Temporal adapters. All new logic lives in `core/`; only orchestration is per-adapter (per `CLAUDE.md` engine-parity rule and `docs/internal/engine-parity-and-subscription-lifecycle.md`).
- **Invoice preview / pro forma** is a *computed* estimate, never stored. Spec A does not implement it; the metering spec adds current-usage preview on top. Mentioned here only so the term is reserved.

## 1. Domain model

`Invoice`, `InvoiceLineItem`, and `Payment` are **operational, single-store** entities. They follow the same convention as `Order` / `OrderItem` / `Payment` / `Subscription` in `internal/core/domain/`: structs **with** gorm tags, `(OrgId, Id)` composite primary key, a `TableName()` method, `serializer:nulltime` for nullable times, `serializer:json` for maps, money as `int64` cents — except the two decimal line-item fields mandated by ADR 0001. They are **not** the tag-free pure types used for metering events (those have two storage adapters; the invoice has one store).

```go
// internal/core/domain/invoice.go
type InvoiceStatus string

const (
	InvoiceStatusDraft   InvoiceStatus = "draft"   // built, not yet settled
	InvoiceStatusOpen    InvoiceStatus = "open"    // a Payment attempt is outstanding
	InvoiceStatusPaid    InvoiceStatus = "paid"    // settled by a succeeded Payment
	InvoiceStatusUnpaid  InvoiceStatus = "unpaid"  // settlement failed / exhausted
	InvoiceStatusVoid    InvoiceStatus = "void"    // cancelled before settlement
)

type Invoice struct {
	OrgId          string            `gorm:"column:org_id;primaryKey" json:"org_id"`
	Id             string            `gorm:"column:id;primaryKey" json:"id"`
	SubscriptionId string            `gorm:"column:subscription_id" json:"subscription_id"`
	CustomerId     string            `gorm:"column:customer_id" json:"customer_id"`
	OrderId        string            `gorm:"column:order_id" json:"order_id"`
	Status         InvoiceStatus     `gorm:"column:status" json:"status"`
	Currency       string            `gorm:"column:currency" json:"currency"`
	// Total is the amount a Payment attempts to settle — sum of line Totals,
	// rounded once at line level (ADR 0001), in int64 cents.
	Subtotal    int64       `gorm:"column:subtotal" json:"subtotal"`
	Total       int64       `gorm:"column:total" json:"total"`
	LineItems   []InvoiceLineItem `gorm:"foreignKey:InvoiceId,OrgId;references:Id,OrgId" json:"line_items,omitempty"`
	// The cycle this invoice bills. CyclesProcessed at build time makes the
	// (subscription, cycle) pair unique, mirroring the billing-cycle RunKey.
	Cycle              int       `gorm:"column:cycle" json:"cycle"`
	PeriodStart        time.Time `gorm:"column:period_start;serializer:nulltime" json:"period_start"`
	PeriodEnd          time.Time `gorm:"column:period_end;serializer:nulltime" json:"period_end"`
	Metadata    map[string]string `gorm:"column:metadata;serializer:json" json:"metadata"`
	CreatedAt   time.Time         `gorm:"column:created_at" json:"created_at"`
	UpdatedAt   time.Time         `gorm:"column:updated_at" json:"updated_at"`
}

func (Invoice) TableName() string { return "invoices" }
```

```go
// internal/core/domain/invoice_line_item.go
//
// Per ADR 0001: Quantity is decimal (whole for product lines, fractional for
// usage lines added by Spec B); UnitAmount is decimal cents, sub-cent capable
// (a usage rate can be below a cent); Total is int64 cents, the actually-charged
// amount, rounded once.
type InvoiceLineItemKind string

const (
	InvoiceLineKindBase  InvoiceLineItemKind = "base"  // the fixed/recurring base charge (Spec A)
	InvoiceLineKindUsage InvoiceLineItemKind = "usage" // metered usage (Spec B)
)

type InvoiceLineItem struct {
	OrgId       string              `gorm:"column:org_id;primaryKey" json:"org_id"`
	Id          string              `gorm:"column:id;primaryKey" json:"id"`
	InvoiceId   string              `gorm:"column:invoice_id" json:"invoice_id"`
	PriceId     string              `gorm:"column:price_id" json:"price_id"`
	Kind        InvoiceLineItemKind `gorm:"column:kind" json:"kind"`
	Description string              `gorm:"column:description" json:"description"`
	// decimal.Decimal — same numeric type the metering spec uses for usage
	// quantities; stored as Postgres numeric (see schema §2).
	Quantity   decimal.Decimal `gorm:"column:quantity;type:numeric" json:"quantity"`
	UnitAmount decimal.Decimal `gorm:"column:unit_amount;type:numeric" json:"unit_amount"` // decimal cents
	Total      int64           `gorm:"column:total" json:"total"`                          // int64 cents, rounded once
	Metadata   map[string]string `gorm:"column:metadata;serializer:json" json:"metadata"`
	CreatedAt  time.Time         `gorm:"column:created_at" json:"created_at"`
	UpdatedAt  time.Time         `gorm:"column:updated_at" json:"updated_at"`
}

func (InvoiceLineItem) TableName() string { return "invoice_line_items" }
```

### `Subscription.Amount` removal

Delete the `Amount int64` field from `Subscription` (`internal/core/domain/subscription.go:76`) and stop seeding it in `NewSubscriptionFromOrderItem` (`:386`) and `NewFromCreateInput` (`:427`). `CreateSubscriptionInput.Amount` (`subscription.go:15`) is dropped from the create path's pricing role — pricing comes from the linked `Price`. `TotalRevenue` and `CyclesProcessed` remain. See §6 for every read site that must move.

### `Payment.InvoiceId`

Add to `Payment` (`internal/core/domain/payment.go:7`):

```go
InvoiceId string `gorm:"column:invoice_id" json:"invoice_id"`
```

The `Payment` keeps `OrderId` and `SubscriptionId`; `InvoiceId` is the new link to the per-cycle invoice it settles. `NetAmount` is set from the invoice total, not `subscription.Amount`.

## 2. Schema (`schemas/app/schema.prisma`)

Prisma is the source of truth for the operational DB `getpaidhq` (`DATABASE_URL`); pushed with `pnpm prisma:push` (clean-slate `db push`, no migrations). New models mirror the existing convention: composite `@@id([orgId, id])`, `@map("snake_case")`, `@relation` on `(orgId, fk)`.

> Note: the operational schema lives at `schemas/app/schema.prisma` (ADR 0002 refers to it as `schemas/getpaidhq/`; same database, `DATABASE_URL`).

```prisma
model Invoice {
  orgId String @map("org_id")
  id    String @default(cuid())

  subscriptionId String       @map("subscription_id")
  subscription   Subscription @relation(fields: [orgId, subscriptionId], references: [orgId, id])

  customerId String   @map("customer_id")
  customer   Customer @relation(fields: [orgId, customerId], references: [orgId, id])

  orderId String @map("order_id")
  order   Order  @relation(fields: [orgId, orderId], references: [orgId, id])

  status   InvoiceStatus
  currency String
  subtotal Int           @default(0)
  total    Int           @default(0)

  cycle       Int
  periodStart DateTime? @map("period_start")
  periodEnd   DateTime? @map("period_end")

  lineItems InvoiceLineItem[]
  payments  Payment[]

  metadata Json?

  createdAt DateTime @default(now()) @map("created_at")
  updatedAt DateTime @updatedAt @map("updated_at")

  org Org @relation(fields: [orgId], references: [id], onDelete: Cascade)

  @@id([orgId, id])
  @@unique([orgId, subscriptionId, cycle]) // one invoice per cycle, mirrors the billing RunKey
  @@map("invoices")
}

enum InvoiceStatus {
  draft
  open
  paid
  unpaid
  void
}

model InvoiceLineItem {
  orgId String @map("org_id")
  id    String @default(cuid())

  invoiceId String  @map("invoice_id")
  invoice   Invoice @relation(fields: [orgId, invoiceId], references: [orgId, id], onDelete: Cascade)

  priceId     String  @map("price_id")
  kind        InvoiceLineItemKind
  description String

  // ADR 0001: decimal quantity + sub-cent-capable decimal unit amount;
  // Total is int64 cents, rounded once.
  quantity   Decimal @db.Decimal(38, 9)
  unitAmount Decimal @db.Decimal(38, 9) @map("unit_amount") // decimal cents
  total      Int     @default(0)

  metadata Json?

  createdAt DateTime @default(now()) @map("created_at")
  updatedAt DateTime @updatedAt @map("updated_at")

  org Org @relation(fields: [orgId], references: [id], onDelete: Cascade)

  @@id([orgId, id])
  @@map("invoice_line_items")
}

enum InvoiceLineItemKind {
  base
  usage
}
```

Edits to existing models:

```prisma
// model Payment — add the invoice link and back-relation:
//   invoiceId String?  @map("invoice_id")
//   invoice   Invoice? @relation(fields: [orgId, invoiceId], references: [orgId, id])

// model Subscription — DROP the charge amount:
//   amount Int   ← remove (schemas/app/schema.prisma:470)
//   add back-relation: invoices Invoice[]

// model Customer / Order — add back-relations: invoices Invoice[]
```

**Deferred to Spec B (referenced, not added here):** the `metered` value on `PriceCategory`, the metered-price fields, and the `usage` line items that the metering spec attaches to the same `Invoice`. The `kind = usage` enum value and the decimal line fields exist now precisely so Spec B adds rows, not columns.

## 3. Ports / repos

New `InvoiceRepository` in `internal/core/port/repository.go`, same style as the existing repositories there (ctx first; `(ctx, orgId, …)`; `(value, error)` returns; `port.ErrNotFound` on miss):

```go
// InvoiceRepository manages invoice + line-item persistence (operational DB).
type InvoiceRepository interface {
	Create(ctx context.Context, entity domain.Invoice) (domain.Invoice, error) // persists Invoice + its LineItems
	FindById(ctx context.Context, orgId string, id string) (domain.Invoice, error)
	// FindBySubscriptionCycle returns the invoice already built for a
	// (subscription, cycle) pair, or ErrNotFound. The build step uses it as
	// the idempotency guard so a replayed billing run reuses one invoice.
	FindBySubscriptionCycle(ctx context.Context, orgId, subscriptionId string, cycle int) (domain.Invoice, error)
	FindBySubscriptionId(ctx context.Context, orgId, subscriptionId string, p domain.Pagination) ([]domain.Invoice, int, error)
	Update(ctx context.Context, entity domain.Invoice) (domain.Invoice, error) // status / total transitions
}
```

Postgres implementation: `internal/adapter/postgres/invoice_repo.go`, registered in `app.go`. `Create` writes the `Invoice` and its `InvoiceLineItem`s in one `tx.RunInTx` so the invoice and its lines commit atomically (same `port.TxManager` pattern already used by `OrderService.CompleteOrder`).

## 4. Pure pricing (`internal/core/domain`)

Building the base line from the linked `Price` is a pure function — no DB — so both engines share it identically and it is unit-testable like the existing `subscription*_test.go` surface:

```go
// internal/core/domain/invoice_build.go
//
// BaseLineFromPrice computes the fixed base line for a subscription's period
// from its linked Price. Today every subscription price is Fixed scheme:
//   Total = round(Price.UnitPrice × quantity)   // int64 cents, rounded once
// Quantity is the OrderItem quantity (whole today); UnitAmount = Price.UnitPrice
// carried as decimal cents.
func BaseLineFromPrice(p Price, quantity decimal.Decimal) InvoiceLineItem

// BuildInvoice assembles a draft Invoice for a subscription's current period
// from its base line(s) and totals it. Spec A passes only the base line.
func BuildInvoice(sub Subscription, lines []InvoiceLineItem) Invoice
```

**Pricing scheme note.** The `PriceScheme` enum exists (`internal/core/domain/price_types.go:25` — `Fixed`, `Tiered`, `Volume`, `Graduated`) but has **no implementation today**. Spec A only needs `Fixed` (`Total = UnitPrice × quantity`). The general tier math (Graduated/Volume) is net-new and is shared with Spec B — the metering spec defines `domain.PriceUsage(p Price, units) (amountCents, unitAmountCents)` switching on `p.Scheme`. Spec A does not build it; it only computes the Fixed base line. When Spec B lands, the base line for a non-Fixed scheme will route through the same shared function.

## 5. Service & wiring

A new **narrow** `InvoiceService` in `internal/core/service/invoice.go` (no workflow engine, per the narrow-vs-orchestration split in `internal/core/service/subscription_orchestration.go` and `CLAUDE.md`). It is built in the narrow-services block in `internal/config/app.go` **before** the engine, so it can be handed to the steps/activities the engine dispatches.

```go
// InvoiceService — narrow. Builds and persists the per-cycle invoice and
// reports its total. No engine, no signaling.
type InvoiceService struct {
	invoiceRepository      port.InvoiceRepository
	subscriptionRepository port.SubscriptionRepository
	orderRepository        port.OrderRepository // for OrderItem.Price (pricing authority)
	priceRepository        port.PriceRepository
	tx                     port.TxManager
	pubsub                 port.PubSub
	logger                 port.Logger
}

// BuildForBillingPeriod builds (or returns the already-built) invoice for the
// subscription's current cycle and persists it as draft. Idempotent on
// (orgId, subscriptionId, cycle) via FindBySubscriptionCycle.
func (s *InvoiceService) BuildForBillingPeriod(ctx context.Context, sub domain.Subscription) (domain.Invoice, error)

// MarkSettled / MarkUnpaid flip status after the Payment result is known.
func (s *InvoiceService) MarkSettled(ctx context.Context, orgId, invoiceId string) (domain.Invoice, error)
func (s *InvoiceService) MarkUnpaid(ctx context.Context, orgId, invoiceId string) (domain.Invoice, error)
```

`BuildForBillingPeriod` resolves the linked `Price` through the subscription's `OrderItem.Price` (the same path `NewSubscriptionFromOrderItem` reads today), calls `domain.BaseLineFromPrice` + `domain.BuildInvoice`, and persists via `InvoiceRepository.Create`. The `cycle` is `sub.CyclesProcessed` (matching the billing-cycle RunKey so a replayed run reuses the same invoice).

`InvoiceService` is injected into `SubscriptionService` (the existing narrow service that owns `ChargeForBillingPeriod` and the charge-result handlers), since the charge flow is where build→total→settle happens. Wiring is by hand in `app.go` (manual DI), as with every other service.

## 6. Billing flow change

`docs/workflows/billing-cycle.md` describes the current flow; the per-cycle orchestration is unchanged in shape — the one-step `billing-cycle` DAG/workflow (`internal/adapter/hatchet/workflows/billing_cycle.go`, `internal/adapter/temporal/workflows/billing_cycle.go`) still calls a single `SubscriptionService` method. Only what that method does changes, and it stays in `core/` so both engines share it.

**Before** (`SubscriptionService.ChargeForBillingPeriod`, `subscription.go:652`):
1. `FindById` the subscription.
2. `gw.ChargePayment{ Amount: subscription.Amount }` (`:683`).
3. On result, `HandleSubscriptionChargeSuccess`/`Failure` record a `Payment` with `NetAmount: subscription.Amount` and advance `TotalRevenue += subscription.Amount`.

**After:**
1. `FindById` the subscription.
2. `InvoiceService.BuildForBillingPeriod(ctx, sub)` → draft `Invoice` with a base line from the linked `Price`, totaled (idempotent per cycle).
3. `gw.ChargePayment{ Amount: invoice.Total }`.
4. `HandleSubscriptionChargeSuccess` records a `Payment` with `InvoiceId: invoice.Id`, `Amount`/`NetAmount` from `invoice.Total`, advances `TotalRevenue += invoice.Total` and `CyclesProcessed++`, then `InvoiceService.MarkSettled`. `HandleSubscriptionChargeFailure` records the failed `Payment` linked to the invoice and `InvoiceService.MarkUnpaid` (dunning path is unchanged).

Read sites that move off `subscription.Amount` (the refactor surface):
- `subscription.go:653` log line.
- `subscription.go:683` gateway charge amount → `invoice.Total`.
- `subscription.go:466`, `:481` (success) and `:577` (failure) `NetAmount` / `TotalRevenue` → `invoice.Total`.
- `internal/core/domain/subscription.go:135` `CalculateProrationDetails` uses `s.Amount` for the credit base; proration must derive the period amount from the linked price (or, post-Spec-B, the invoice). For Spec A, source it from `Price.UnitPrice`.
- `domain.SetActive` (`subscription.go:327`) sets `TotalRevenue`/`LastCharge` from the first `Payment.Amount`, not `subscription.Amount`, so it already reads the payment — confirm it no longer references the removed field.

**Engine-parity note.** No workflow-shape change: the DAG/activity still calls one `core/` method. Build→total→settle is entirely in `InvoiceService` + `SubscriptionService` + pure `domain` functions, so Hatchet and Temporal produce the same invoice, the same total, and the same settling payment by construction. The `(orgId, subscriptionId, cycle)` uniqueness on `invoices` plus `FindBySubscriptionCycle` make the build idempotent under the durable replay / DAG retry both engines do (same guard reasoning as the `CyclesProcessed` idempotency check at `subscription.go:446`).

## 7. Migration note

No Prisma migrations are checked in; schema syncs via clean-slate `pnpm prisma:push` (`CLAUDE.md`). The change is: add the `invoices`, `invoice_line_items` tables (+ `InvoiceStatus`, `InvoiceLineItemKind` enums), add `payments.invoice_id`, and **drop `subscriptions.amount`**. Because `db push` is clean-slate locally there is no data backfill in dev. The code-side work is the refactor in §6 — every path that reads `subscription.Amount` as the charge must move to the invoice total (gateway charge, payment net amount, revenue accrual, proration). ADR 0002 explicitly waives backward compatibility.

## 8. Phasing

1. **Domain + schema.** `Invoice`, `InvoiceLineItem`, `Payment.InvoiceId`; drop `Subscription.Amount`; `domain.BaseLineFromPrice` + `BuildInvoice` (Fixed only) with unit tests modeled on `subscription*_test.go`; `db push`.
2. **Repo + service.** `InvoiceRepository` (Postgres) + `InvoiceService` (narrow); wire in `app.go`.
3. **Billing flow.** Rewire `ChargeForBillingPeriod` and the charge-result handlers to build→total→settle; verify identical behaviour on both engines (Hatchet default, Temporal) — fixed subscription bills the same total it did before, now with an itemized invoice behind it.

## 9. Depends on / depended-on-by

- **Depends on:** nothing new — uses the existing Order → Subscription → Price catalog and the existing `billing-cycle` workflow surface on both engines.
- **Depended-on-by:** the usage-metering spec (`docs/superpowers/specs/2026-06-04-usage-based-metering-design.md`, "Spec B"). It adds `kind = usage` line items to the same `Invoice` at billing time and implements **Invoice preview** (current-usage pro forma) on top of the structures defined here. The general `PriceScheme` math (Graduated/Volume) is net-new there and shared back with the base-line builder.
