# Order flow: once-off + subscription, with discounts, invoicing & order-config flags — Design Spec

**Date:** 2026-06-24
**Status:** Settled — ready for implementation planning
**Area:** Fix the create/complete-order flow end to end: order-config flags (payment mode + invoice behaviour), order-owned discounts, one-time-order invoices (`BuildForOrder`), and "the subscription owns its first invoice at activation". Everything the coupon design (`2026-06-23-coupon-reservation-and-application-design.md` §2/§6/§7) agreed, built now.

**Only external gap:** the actual **hosted-checkout payment page** (the customer-facing pay-link UI + its redirect). All order-side machinery for the `checkout` mode (the flag, pending order, open invoices built up-front, the `session_id` link, settlement on payment success) is built so the page can drop in later.

---

## 1. Why

`CompleteOrder` today: activates subscriptions, records a `Payment` with **no `InvoiceId`**, builds **no invoice** for the first charge, builds **no invoice** for one-time lines, and consumes a coupon only onto a subscription (one-time coupons orphaned). There is no way to choose how an order is paid or whether/when it is invoiced. This spec completes the agreed model: an `Order` configured by flags, whose contents + flags drive payments, subscriptions, and invoices.

---

## 2. Order-config flags

Two flags, set at `CreateOrder`, persisted on the order (real columns — not `Metadata`).

### `payment_mode` — how the first payment is collected
- **`direct`** (default) — a payment is supplied / a saved method is charged now (today's `CompleteOrder`-with-a-payment path). Invoices are built **and paid** at completion.
- **`checkout`** — the order stays `pending` with a hosted pay-link the customer pays. Invoices are built **`open` up-front** so the hosted PSP charges that exact bill; they settle when payment is confirmed. The hosted page is the external gap; everything else is built.

### `invoice_behaviour` — whether/how a one-time order's lines are invoiced
- **`record`** (default) — build the one-time invoice as the **receipt** (paid in `direct`; open→paid via the hosted flow in `checkout`).
- **`open`** — build the one-time invoice **`open`** (send-an-invoice / pay-later), settled when paid.
- **`none`** — no invoice for one-time lines (just the `Payment`). Invoicing is optional, per use case.

A **subscription always** owns its invoices (§5) regardless of `invoice_behaviour`; the flag governs only the **one-time** lines of an order.

### Surface
Order configuration is a **single typed `config` JSONB column** on `orders` — not discrete columns — because it is an explicitly growing flag bag, is never queried/filtered on, and mirrors the existing `orders.payment_session` JSONB pattern. New order-config flags are added as struct fields with **no further migration**.

```go
type PaymentMode string      // "direct" | "checkout"
type InvoiceBehaviour string // "none" | "record" | "open"

type OrderConfig struct {
    PaymentMode      PaymentMode      `json:"payment_mode"`
    InvoiceBehaviour InvoiceBehaviour `json:"invoice_behaviour"`
    // future order-config flags land here
}
```
- Migration: `orders` gains `config JSONB` (one column, holds the serialized `OrderConfig`).
- `domain.Order` gains `Config OrderConfig`. A typed struct value is never nil, so the gorm `serializer:json` `Save` path is safe (unlike the bare-`any` `payment_session`). `OrderConfig` has a `withDefaults()`/validation in `domain` (empty `PaymentMode` → `direct`, empty `InvoiceBehaviour` → `record`). Both storage adapters map the column (gorm `serializer:json`, pgx `jsonCol[domain.OrderConfig]`); conformance round-trips it.
- `port.CreateOrderInput` gains `Config OrderConfig` (or the two flag fields, assembled into `OrderConfig`); `CreateOrderRequest` (HTTP) gains `payment_mode` / `invoice_behaviour` (`validate:"omitempty,oneof=..."`). Defaults applied + validated in the domain at create; persisted on the order. `CompleteOrder` reads `order.Config` off the stored order (not re-supplied). Distinct from the existing `CreateOrderInput.Options` (the PSP-options map passed to `InitPayment`), which is unchanged.

---

## 3. Discount becomes order-owned

Today `NewDiscount` enforces "exactly one of subscription or order". Inconsistent — the order is the topmost owner (subscriptions/invoices are created from an order; `invoices.order_id` is `NOT NULL`, `subscription_id` nullable). A coupon reservation is held on the **order**.

- `Discount`: **`OrderId` always set**; **`SubscriptionId` optional** (set when the discount targets a subscription's recurring invoices). Update the struct comment.
- `NewDiscount`: require `OrgId`, `CouponId`, `CustomerId`, **`OrderId`**; `SubscriptionId` optional; drop the `hasSub == hasOrder` rule.
- **No discounts migration** — `discounts.subscription_id` and `order_id` are already both nullable, no CHECK.
- `DiscountRepository.ActiveForOrder` → `WHERE order_id = ? AND subscription_id IS NULL AND status='active'` (the order-level discounts), so a subscription's discount — which now also carries `order_id` — never leaks into a one-time order invoice. `ActiveForSubscription` unchanged. Both drivers; conformance updated.
- `CouponService.Consume` **always** sets `OrderId`; sets `SubscriptionId` only when consuming onto a subscription.

---

## 4. One rule: discount is applied at every bill build (§7)

Whenever a bill is built — a subscription cycle invoice **or** a one-time order invoice — the coupon's discount is computed on that bill's lines and subtracted, so the produced total *is* the discounted amount. Reuse the existing `domain.ApplyDiscounts(lines, applied, cycle, currency)` from `InvoiceService`:
- subscription cycle invoice → discounts from `ActiveForSubscription`, `cycle = sub.CyclesProcessed` (existing behaviour, unchanged).
- one-time order invoice → discounts from `ActiveForOrder` (sub-null), `cycle = 0` (every duration is in-window once).
- **Pre-payment (`checkout`)**: when an invoice is built `open` before the `Discount` record exists, resolve the discount from the order's live **reservation's** coupon, so the open bill already reflects the discounted total the hosted PSP will charge. On payment success the `Discount` record is written (no recompute).

Factor the per-line discount mechanics so the subscription build, the order build, and the pre-payment reservation build share one helper.

---

## 5. Subscription owns its first invoice at activation

Today an upfront payment at activation increments `CyclesProcessed` 0→1 (`subscription.go SetActive`) and the engine starts recurring at cycle 1 — but **no cycle-0 invoice is built and the `Payment` has no `InvoiceId`** (`order.go`). Fix:

In `CompleteOrder`, for each subscription, when a first payment is collected (`direct`) — **before** `SetActive` increments the cycle:
1. Build the cycle-0 invoice via `InvoiceService.BuildForBillingPeriod` against the pre-activation subscription (CyclesProcessed 0, first period). It applies the subscription's discount (the reservation is consumed first — see §6).
2. Create the `Payment` with `InvoiceId = inv.Id`.
3. `MarkOpen` then `MarkSettled` the invoice (it's paid now). For `checkout` (no payment yet at this point) the invoice is left `open` and settled by the payment-success path.
4. `SetActive(payment)` → CyclesProcessed 1, RenewsAt in the future → `IsDueForBilling` false → the engine starts at cycle 1.

**Engine parity (both Hatchet & Temporal):** unchanged dispatch. The first invoice is owned by activation; the recurring engine builds cycles ≥1. `BuildForBillingPeriod` is idempotent on `(sub, cycle)`, so any engine replay of cycle 0 returns the existing paid invoice and never re-charges. No per-engine code change is required beyond confirming neither engine charges cycle 0 after an upfront payment (it doesn't: `IsDueForBilling` is false). The no-upfront-payment path (engine builds+charges cycle 0) is unchanged.

---

## 6. `InvoiceService.BuildForOrder` — one-time order invoice

```go
func (s *InvoiceService) BuildForOrder(ctx context.Context, order domain.Order) (domain.Invoice, error)
```
1. Idempotency: `FindOrderInvoice(orgId, orderId)` (new repo method — the invoice with this `order_id` and `subscription_id` NULL). If found, return it.
2. Load `FindOrderItemsByOrderId`; keep only **non-recurring** lines (`!price.IsRecurring()`). None → return `port.ErrNotFound` (caller skips).
3. `NewInvoiceForOrder(order)` — sets `OrgId`, `OrderId`, `CustomerId`, `Currency`, `SubscriptionId: ""`, `Cycle: 0`, period = order completion time. One `BaseLineFromPrice` per one-time item.
4. Discounts: `ActiveForOrder` → `ApplyDiscounts(..., cycle=0, ...)`.
5. Persist (`invoiceRepository.Create`) inside the ambient tx.

Status/flow is driven by the order's `invoice_behaviour` (§7).

---

## 7. `CompleteOrder` orchestration (the whole flow)

Inside the existing `RunInTx`, after the order is marked completed and the payment method resolved:

1. **Consume the reservation** (run for one-time orders too — drop the `len(activated)>0` gate):
   - subscriptions exist → `Consume{OrderId, SubscriptionId: activated[0].Id, StartCycle: 0}`.
   - pure one-time → `Consume{OrderId, StartCycle: 0}` (order-owned discount).
   Run **before** any bill build so the `Discount` exists.
2. **Subscriptions** (§5): build cycle-0 invoice, link the `Payment.InvoiceId`, mark open+settled (`direct`) / open (`checkout`), `SetActive`, update.
3. **One-time lines** per `invoice_behaviour`:
   - `none` → no invoice (just the `Payment`, if any).
   - `record` → `BuildForOrder`; `direct` ⇒ mark open+settled and link a one-time `Payment.InvoiceId`; `checkout` ⇒ leave `open`.
   - `open` → `BuildForOrder`, leave `open` (settled when paid).
4. Mixed order coupon → goes to the subscription (step 1); `ActiveForOrder` is sub-null so the order invoice is undiscounted (one reservation → one discount).

`OrderService` gains an `*InvoiceService` dependency (wired in `app.go`). All invoice/discount/payment writes are inside the order-completion transaction (the merged `RunInTx` ctx fix makes the nested `Consume`/`Create` join it).

For `payment_mode == checkout`: `CompleteOrder` is not the payment step; the order stays `pending` with its invoices built `open` (at create/await time), and the existing payment-success path (webhook → `Payment` succeeded) marks them settled. The **only** missing piece is the hosted page that produces that payment.

---

## 8. Invoice ↔ Payment linking

`domain.Payment` already has `InvoiceId`. Set it for the first subscription invoice (§5) and the one-time order invoice (`record`/`direct`). The subscription-recurring path already links + `MarkSettled` (`subscription.go HandleSubscriptionChargeSuccess`) — unchanged. A one-time order may produce its own `Payment` (`OrderId` set, `SubscriptionId` empty, `Recurring:false`, `InvoiceId` = order invoice) when paid directly.

---

## 9. Data model & migrations

- `orders`: `+ config JSONB` (one column holding the typed `OrderConfig`; future flags need no migration). `domain.Order.Config` + both row mappers + conformance.
- `invoices`: `subscription_id` → **nullable** (`DROP NOT NULL`) for order-only invoices. `order_id` stays `NOT NULL`. Confirm both adapters write NULL (not `""`) for empty `subscription_id`.
- **No discounts migration** (columns already nullable).
- `InvoiceRepository.FindOrderInvoice(orgId, orderId)` (order_id set, subscription_id NULL) — port + both adapters + conformance.

---

## 10. Hexagonal placement

| Layer | Change |
| --- | --- |
| `core/domain` | `Order.Config` (`OrderConfig` enums + validation); `NewDiscount` order-always; `NewInvoiceForOrder`. |
| `core/service` | `InvoiceService.BuildForOrder` + shared discount helper + reservation-coupon pre-payment resolution; `CouponService.Consume` order-always; `OrderService.CreateOrder` persists flags; `OrderService.CompleteOrder` full orchestration (§7) incl. first sub invoice (§5); `OrderService` gains `*InvoiceService`. |
| `core/port` | `CreateOrderInput.PaymentMode/InvoiceBehaviour`; `InvoiceRepository.FindOrderInvoice`. |
| `adapter/storage/{postgresgorm,postgrespgx}` | order `config` jsonb column; `FindOrderInvoice`; `ActiveForOrder` sub-null; invoice nullable `subscription_id`. Both drivers + conformance. |
| `adapter/http` | `CreateOrderRequest` flags (validated, defaulted). |
| `config/app.go` | inject `*InvoiceService` into `OrderService`. |
| `schemas/app/migrations` | `orders.config` jsonb; `invoices.subscription_id` nullable. |

No workflow-engine code change (activation owns the first invoice; recurring engine unchanged & idempotent). Parity preserved.

---

## 11. Behaviour matrix

| Order | payment_mode | invoice_behaviour | Result |
| --- | --- | --- | --- |
| Pure subscription | direct | (n/a) | cycle-0 invoice built+paid at activation, `Payment.InvoiceId` set; engine bills cycle 1+ |
| Pure subscription | checkout | (n/a) | cycle-0 invoice built `open` up-front; settled on hosted payment; engine bills 1+ |
| Pure one-time, coupon | direct | record | one paid order invoice, order-owned discount applied; reservation consumed |
| Pure one-time | direct | none | `Payment` only, no invoice |
| Pure one-time | direct | open | `open` order invoice (pay-later) |
| Pure one-time | checkout | record/open | `open` order invoice up-front; settled on hosted payment |
| Mixed (sub + one-time), coupon | direct | record | sub cycle-0 invoice (discounted, coupon→sub) + paid one-time order invoice (undiscounted) |
| Retried `CompleteOrder` | any | any | idempotent: `FindBySubscriptionCycle` / `FindOrderInvoice` reuse; no double invoice/charge |

---

## 12. Testing

- **domain:** `NewDiscount` (order-only ✓, order+sub ✓, missing order ✗, sub-only ✗); `Order` flag validation + defaults; `NewInvoiceForOrder`.
- **storage (both drivers, conformance):** order `config` jsonb round-trips; `FindOrderInvoice` returns the sub-null order invoice & `ErrNotFound` when only a sub invoice exists; `ActiveForOrder` excludes sub-targeted discounts sharing the order_id; invoice round-trips NULL `subscription_id`.
- **service:** `BuildForOrder` (one-time lines only, discount once, idempotent); `CompleteOrder` — pure one-time `direct/record` → one paid discounted order invoice, reservation consumed; `direct/none` → no invoice; `open` → open invoice; **subscription `direct` → cycle-0 invoice built+paid, `Payment.InvoiceId` set, CyclesProcessed=1**; mixed order routes coupon to sub; idempotent re-complete.
- **engine parity (integration, both engines):** after activation with an upfront payment, the recurring engine bills cycle 1 next and does NOT rebuild/recharge cycle 0 (the existing paid invoice is reused). The no-upfront-payment path still builds+charges cycle 0.
- **e2e:** create once-off order (coupon, direct/record) → complete → one paid discounted invoice, reservation gone, replay no-ops; create subscription order (direct) → complete → cycle-0 paid invoice + recurring continues discounted.

---

## 13. The one external gap

The hosted-checkout **page** (customer-facing pay-link UI + redirect) is not built — it cannot exist yet. Everything it needs is built and ready: the `payment_mode=checkout` flag, the `pending` order, invoices built `open` up-front (discounted via the reservation), the order's `session_id` link, and settlement via the existing payment-success path. When the page lands it populates the session and the open invoices settle through the path that already exists. **Nothing else is deferred.**

---

## 14. Decisions log

| Decision | Choice | Rationale |
| --- | --- | --- |
| Discount ownership | order_id always; subscription_id optional | Order is topmost owner; mirrors `invoices`; no migration. |
| Order flags storage | one typed `config` JSONB column (`OrderConfig`) | Growing flag bag, never queried on; mirrors `payment_session`; new flags need no migration; typed struct is nil-safe for gorm Save. |
| `payment_mode` default | `direct` | Today's working path; no surprise. |
| `invoice_behaviour` default | `record` | Most once-off sales want a receipt; `none`/`open` opt-in. |
| First subscription invoice | built+paid at activation (`direct`) / open (`checkout`) | §6 "subscription owns its first invoice"; links `Payment.InvoiceId`. |
| Engine parity | no per-engine change; rely on `CyclesProcessed`=1 + build idempotency | Activation owns cycle 0; recurring engine bills ≥1; replay-safe. |
| One-time scope | non-recurring lines only | Subscription owns recurring; no double-bill. |
| Mixed-order coupon | → subscription | One reservation → one discount; `ActiveForOrder` sub-null. |
| Pre-payment discount (checkout) | from the live reservation's coupon | Open bill reflects the discounted charge before the `Discount` record exists. |
