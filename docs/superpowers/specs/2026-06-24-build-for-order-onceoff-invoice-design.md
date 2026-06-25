# Order flow: combined invoicing, order-owned discounts, opt-in upfront invoice — Design Spec

**Date:** 2026-06-24
**Status:** Settled — ready for implementation planning
**Area:** Fix create/complete-order invoicing. An order produces **one combined invoice** at payment confirmation (or `open` up-front, opt-in). Discounts become order-owned. The subscription's first bill is that same invoice.

---

## 1. Why

`CompleteOrder` today builds **no** invoice for the first bill, records a `Payment` with an empty `InvoiceId`, never invoices one-time lines, and consumes a coupon only onto a subscription (one-time coupons orphaned). Order and payment are already split (order create ≠ payment); invoicing should follow from that:

- **Payment is not an order concern.** Creating an order says nothing about how/when it is paid. The order sits `pending` with no invoice until payment is confirmed — so an abandoned order leaves nothing to void (the old up-front-invoice-per-checkout void mess is gone).
- **One combined invoice per order.** When paid, the order produces a single invoice covering all its first-bill lines (each subscription's first period + every one-time line). The customer paid once for one cart → one bill.
- **Opt-in upfront invoice.** For "send someone an invoice with a pay link", the order can be created with the invoice raised **`open`** now; it settles to `paid` when paid. Opt-in and deliberate, so no void pile-up.

---

## 2. Order configuration

The order carries one typed setting, set at `CreateOrder`, persisted in a single `config` JSONB column on `orders` (mirrors `orders.payment_session`; grows without migrations; never queried on; a typed struct is nil-safe for gorm `Save`):

```go
type OrderConfig struct {
    UpfrontInvoice bool `json:"upfront_invoice"` // raise the invoice OPEN now (send-an-invoice); default false
    // future order configuration lands here
}
```
- `false` (default) → no invoice at create; the combined invoice is built **`paid`** when payment confirms.
- `true` → build the combined invoice **`open`** at create; `CreateOrder` returns its details (§3); it settles to `paid` when paid.

**Nothing about payment** lives on the order. Payment is driven separately — `POST /orders/{id}/pay` issues a PSP-hosted payment session/link, or `CompleteOrder` records a supplied/charged payment. `domain.Order` gains `Config OrderConfig`; both adapters map the column (gorm `serializer:json`, pgx `jsonCol[domain.OrderConfig]`); conformance round-trips it. `CreateOrderRequest` gains `upfront_invoice bool`; `port.CreateOrderInput` gains `Config`.

---

## 3. Invoice identity: number + reference

An invoice has three identifiers:
- **`Id`** — internal system id (`inv_…`); URLs and relations.
- **`Number`** (`int64`, already on `main`) — the raw **counter value**, from the org-scoped `invoice_counters` + `InvoiceRepository.NextInvoiceNumber(orgId)` (set inside the build tx so the counter bump and the insert are atomic). It is *just the counter value* — **not** the public identity and **not** relied on for uniqueness (numbering scope may become per-customer/other downstream; a single per-org sequence is not assumed permanent).
- **`Reference`** (text) — **NEW**: the **public, searchable identity** a customer copies off the invoice and searches by. **Stored and indexed** (`(org_id, reference)`); the durable identity lives here. Its default form derives from the number; a configurable/scoped format is downstream, and because `reference` is stored, that needs no model change later.

Both `Number` and `Reference` are set at build, in the tx, by **`BuildForOrder`** and **`BuildForBillingPeriod`** (the latter updated to also set `Reference`).

When `upfront_invoice = true`, the `CreateOrder` response includes the invoice:
```json
"invoice": { "id": "inv_…", "number": 42, "reference": "INV-000042", "url": "" }
```
`url` is a **placeholder** (empty for now; it will point at the hosted invoice page when that exists). When `upfront_invoice = false` there is no invoice yet, so the field is omitted/null.

---

## 4. Discounts become order-owned

Today `NewDiscount` enforces "exactly one of subscription or order" — inconsistent with the order being the topmost owner (subscriptions/invoices are created from an order; the reservation is held on the order; `invoices.order_id` is `NOT NULL`).

- `Discount`: **`OrderId` always set**; **`SubscriptionId` optional** (set when targeting a subscription's recurring invoices).
- `NewDiscount`: require `OrgId`, `CouponId`, `CustomerId`, **`OrderId`**; `SubscriptionId` optional; drop the `hasSub == hasOrder` rule.
- **No discounts migration** — `discounts.subscription_id`/`order_id` already nullable, no CHECK.
- `DiscountRepository.ActiveForOrder` → `WHERE order_id = ? AND subscription_id IS NULL AND status='active'`; `ActiveForSubscription` unchanged. Both drivers + conformance.
- `CouponService.Consume` **always** sets `OrderId`; sets `SubscriptionId` only when consuming onto a subscription.

---

## 5. The combined order invoice

One invoice per order, covering its **first bill**: each subscription's first-period (cycle-0) base/usage line(s) **plus** every one-time line. Built by a single `InvoiceService.BuildForOrder(ctx, order)`:

1. Idempotency: `FindOrderInvoice(orgId, orderId)` — the invoice for this order (`order_id` set, `cycle = 0`). If found, return it.
2. Gather the order's lines (`FindOrderItemsByOrderId`): recurring lines contribute their first-period base/usage line; one-time lines contribute a base line. One `domain.Invoice` (`OrgId`, `OrderId`, `CustomerId`, `Currency`, `Cycle: 0`, period = first period / order time). `Number` + `Reference` are assigned in the tx (step 5).
3. **Subscription linkage:** if the order has exactly one subscription, set `SubscriptionId` on the invoice (so it **is** that subscription's cycle-0 invoice — the recurring engine's `FindBySubscriptionCycle(sub, 0)` finds it and never rebuilds/recharges cycle 0). Pure one-time order → `SubscriptionId` NULL. (Multi-subscription orders: each subscription owns its own cycle-0 invoice; one-time lines combine onto the first — an edge, kept explicit.)
4. **Discount:** apply via `domain.ApplyDiscounts(lines, applied, cycle=0, currency)`. Resolve discounts from the committed `Discount` (post-payment) **or** the order's live reservation's coupon when building `open` pre-payment, so the bill total *is* the discounted amount.
5. In the tx: `inv.Number = NextInvoiceNumber(orgId)`, set `inv.Reference` (default form from the number), then `invoiceRepository.Create`.

`BuildForOrder` reuses the subscription builder's per-line + discount mechanics (shared helper). `BuildForBillingPeriod` stays **recurring-only** for cycles ≥ 1 (unchanged).

---

## 6. When the invoice is built & settled

- **`upfront_invoice = false` (default):** order created `pending`, no invoice. On **payment confirmation** — `CompleteOrder` (supplied/charged payment) or the payment-success path (`/pay` → PSP webhook) — `BuildForOrder` builds the combined invoice, it is marked **`paid`**, the `Payment.InvoiceId` is linked, and the reservation is consumed → `Discount`.
- **`upfront_invoice = true`:** at `CreateOrder`, `BuildForOrder` builds the combined invoice **`open`** (discount from the live reservation); the response returns its number/url. On payment confirmation it is marked **`paid`** and linked; the reservation is consumed.

Either way: **one** combined invoice, discount applied once, settled when paid. No invoice is ever built that isn't either paid or a deliberately-raised open invoice.

---

## 7. Subscription first invoice = the combined invoice

Today `SetActive` advances `CyclesProcessed` 0→1 on an upfront payment and the engine starts recurring at cycle 1, but no cycle-0 invoice exists and `Payment.InvoiceId` is empty. Fix: the combined order invoice (§5) **is** the subscription's cycle-0 invoice (it carries `subscription_id` + `cycle 0`). So:
- Build it before/at activation; link the first `Payment.InvoiceId`; mark paid (or open up-front).
- `SetActive` → `CyclesProcessed 1`, `RenewsAt` future → `IsDueForBilling` false → the recurring engine bills cycle 1+ and, being idempotent on `(sub, cycle)`, never rebuilds/recharges cycle 0.

**Engine parity:** no per-engine change. Both Hatchet and Temporal already skip cycle 0 after an upfront payment (`IsDueForBilling` false) and rely on `BuildForBillingPeriod` idempotency. The no-upfront-payment path (engine builds+charges cycle 0) is unchanged — and now finds the combined invoice if one exists.

---

## 8. `CompleteOrder` / payment-success orchestration

Inside the existing `RunInTx`, on payment confirmation:
1. **Consume the reservation** (run for one-time orders too — drop the `len(activated)>0` gate): subscriptions exist → `Consume{OrderId, SubscriptionId, StartCycle:0}`; pure one-time → `Consume{OrderId, StartCycle:0}`. Before any bill build.
2. **`BuildForOrder`** → the combined invoice (unless one was already built `open` up-front — then load it). Apply discount.
3. Create the `Payment` with `InvoiceId =` the invoice; `MarkOpen`+`MarkSettled` (now paid).
4. `SetActive` each subscription; update.

All within the order-completion transaction (the merged `RunInTx` ctx fix lets nested `Consume`/`Create` join it). Post-commit: start subscription workflows, publish `order.completed` (unchanged).

---

## 9. Invoice ↔ Payment linking

`domain.Payment` already has `InvoiceId`. Set it to the combined invoice for the order's payment. The recurring path (`HandleSubscriptionChargeSuccess`) already links + `MarkSettled` for cycles ≥ 1 (unchanged).

---

## 10. Data model & migrations

- `orders`: `+ config JSONB` (typed `OrderConfig`). `domain.Order.Config` + both mappers + conformance.
- `invoices`: `+ reference TEXT` (public searchable identity) with index `(org_id, reference)`; `subscription_id` → **nullable** (`DROP NOT NULL`) for pure-once-off invoices (`order_id` stays `NOT NULL`); confirm empty `subscription_id` writes NULL, not `""`. `domain.Invoice.Reference`; both adapters map it; conformance round-trips it. (The `number`/counter columns are already on `main`; `reference` is new. `BuildForBillingPeriod` is updated to also set `Reference`.)
- **No discounts migration.**
- `InvoiceRepository.FindOrderInvoice(orgId, orderId)` (the order's cycle-0 invoice) — port + both adapters + conformance.

---

## 11. Hexagonal placement

| Layer | Change |
| --- | --- |
| `core/domain` | `Order.Config` (`OrderConfig`, validated); `Invoice.Reference`; `NewDiscount` order-always; the combined-invoice build helpers. |
| `core/service` | `InvoiceService.BuildForOrder` (sets `Number`+`Reference`) + shared discount/line helper + reservation-coupon resolution; `BuildForBillingPeriod` also sets `Reference`; `CouponService.Consume` order-always; `OrderService.CreateOrder` persists `Config` and, if `UpfrontInvoice`, builds the open invoice + returns it; `OrderService.CompleteOrder` orchestration (§8); `OrderService` gains `*InvoiceService`. |
| `core/port` | `CreateOrderInput.Config`; `InvoiceRepository.FindOrderInvoice`; `CreateOrderResult` carries the optional invoice. |
| `adapter/storage/{postgresgorm,postgrespgx}` | `orders.config`; `invoices.reference` (+ index) + `subscription_id` nullable; `FindOrderInvoice`; `ActiveForOrder` sub-null. Both drivers + conformance. |
| `adapter/http` | `CreateOrderRequest.upfront_invoice`; `CreateOrderResponse` returns `invoice {id, number, url}` when raised. |
| `config/app.go` | inject `*InvoiceService` into `OrderService`. |
| `schemas/app/migrations` | `orders.config`; `invoices.reference` + index; `invoices.subscription_id` nullable. |

No workflow-engine code change. Parity preserved.

---

## 12. Behaviour matrix

| Order | upfront_invoice | At create | At payment confirmation |
| --- | --- | --- | --- |
| Pure subscription ($100/mo) | false | pending, no invoice | one paid invoice (cycle-0 $100); engine bills cycle 1+ |
| Mixed ($100/mo + $50 once-off) | false | pending, no invoice | **one combined paid invoice $150** (sub first period + once-off); engine bills $100 cycle 1+ |
| Pure once-off ($50), coupon | false | pending, no invoice | one paid invoice $50−discount; order-owned discount; reservation consumed |
| Any, send-an-invoice | true | combined **open** invoice built (discounted); response returns `{id, number, url}` | invoice → paid, linked, reservation consumed |
| Abandoned (never paid) | false | pending, no invoice | nothing to void |
| Retried completion | any | — | idempotent: `FindOrderInvoice` reuses; no duplicate invoice/charge |

---

## 13. Testing

- **domain:** `NewDiscount` (order-only ✓, order+sub ✓, missing order ✗, sub-only ✗); `OrderConfig` validation; `Invoice.Number` (counter) + `Invoice.Reference` set at build.
- **storage (both drivers, conformance):** `orders.config` round-trips; `invoices.reference` round-trips and is searchable by `(org_id, reference)`; NULL `subscription_id` round-trips; `FindOrderInvoice` returns the order's cycle-0 invoice; `ActiveForOrder` excludes sub-targeted discounts sharing the order_id.
- **service:** `BuildForOrder` — combined lines (sub first period + once-off), discount once, idempotent, sets `SubscriptionId` for a single-sub order. `CompleteOrder` — mixed `$100/mo + $50` → one combined paid `$150` invoice, `Payment.InvoiceId` set, sub `CyclesProcessed=1`; pure once-off+coupon → one paid discounted invoice, reservation consumed; `upfront_invoice` → open invoice at create, paid on completion; idempotent re-complete.
- **engine parity (integration, both engines):** after activation the recurring engine bills cycle 1 next, never rebuilds/recharges cycle 0 (the combined invoice is reused).
- **e2e:** mixed cart with a coupon → `/pay` (or complete) → one combined paid discounted invoice; subscription recurs from cycle 1; `upfront_invoice` → open invoice returned, then settled.

---

## 14. Decisions log

| Decision | Choice | Rationale |
| --- | --- | --- |
| Payment on the order | none | Order create ≠ payment; payment is separate (`/pay`, complete). |
| Invoice timing | built at payment confirmation (`paid`), or `open` up-front if `upfront_invoice` | No dangling/void invoices (order/payment split); send-an-invoice is opt-in. |
| Order config | one `config` JSONB field: `upfront_invoice` | Only genuine knob; grows without migration; nil-safe struct. |
| Mixed cart | **one combined invoice** (all first-bill lines) | Paid once for one cart → one bill. |
| Subscription first invoice | = the combined order invoice (`subscription_id`+`cycle 0`) | Subscription owns its first bill; engine bills 1+; idempotent. |
| Discount ownership | `order_id` always, `subscription_id` optional | Order is topmost owner; mirrors `invoices`; no migration. |
| Invoice identity | `Number` (counter value, from `main`) **and** `Reference` (text, stored + indexed) | `Reference` is the public, searchable identity (survives format changes, encodes future per-entity scope); `Number` is just the counter, not relied on for uniqueness. `url` placeholder until the hosted page exists. |
| Engine parity | no per-engine change | Activation owns cycle 0; recurring bills ≥1; replay-safe. |
