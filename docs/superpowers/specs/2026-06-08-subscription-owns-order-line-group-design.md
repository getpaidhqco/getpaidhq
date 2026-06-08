# Subscription owns a group of order lines (one cadence), not a single item

## Context / problem

A `Subscription` is currently glued to **one** order item: `Subscription.OrderItemId` (singular),
set by `NewSubscriptionFromOrderItem(item, price)`, with `Amount = price.UnitPrice`. The Prisma
relation is `subscription.orderItemId → order_item`.

Because the link is 1:1, billing a **flat + metered** plan together (the normal SaaS shape:
"$100/mo + $X per token") has nowhere to live. The codebase works around it with two hacks:

- `subscriptionAnchors` (order.go) makes the *fixed* item the subscription and makes metered
  items "ride" it (`continue // metered usage rides the order's fixed plan`).
- `InvoiceService.BuildForBillingPeriod` walks the **whole order** and uses
  `primaryItemId` / `isPrimary` / `isOwn` to decide which subscription claims which line and to
  bill a shared meter "once" by a designated primary subscription.

This contradicts the code's own stated intent (order.go:221: *"a metered price is a recurring
subscription billed by usage, **not a rider on a fixed plan**"*) and misattributes usage the
moment an order has two plans with distinct meters (all metered usage lands on the primary).

## Target model

A **Subscription is a recurring agreement billed at one cadence that owns a group of order
lines** (flat and/or metered). Grouping is by **billing cadence** (interval + qty).

- `OrderItem` gains a nullable `subscriptionId`. A subscription owns the order items pointing to
  it; one-time / free / variable lines have `subscriptionId = null` (charged once, no
  subscription).
- `Subscription` drops the single-item assumption (`OrderItemId`). It keeps its own cadence
  (`BillingInterval` / `BillingIntervalQty`) — that *is* the group's cadence.
- **Order creation** partitions the order's *recurring* lines (any price with a real
  `BillingInterval`, i.e. fixed-subscription or metered) by `(BillingInterval, BillingIntervalQty)`.
  Each partition → one subscription; stamp each line's `subscriptionId`. Normal order = one
  cadence = **one** subscription with all its lines. A second cadence (e.g. a yearly add-on) =
  a second subscription. `subscriptionAnchors` is deleted.
- **Invoice per cycle** = the subscription's own lines (`order_items WHERE subscription_id =
  sub.id`): each fixed line → a base line at its flat amount; each metered line → a usage line
  for its meter. **`primaryItemId` / `isPrimary` / `isOwn` and the whole-order walk are
  deleted** — a line is on the invoice iff it belongs to the subscription.

### Worked example — "$100 flat + per-token"
One product, two prices (A = $100/mo fixed, B = per-token metered, same monthly cadence). Order
has two lines. Order creation groups both (same cadence) → **one** subscription owning A and B.
Each cycle its invoice = `$100` (A) + `tokens × rate` (B). No primary, no rider.

## Components & changes (all in `core/` — engine parity is automatic)

Both Hatchet and Temporal create subscriptions and run billing through the same core services
(`OrderService.CreateOrder`/`CompleteOrder`, `SubscriptionService.ChargeForBillingPeriod`,
`InvoiceService.BuildForBillingPeriod`) and only pass `Subscription` values, so no adapter logic
changes. Verify the parity docs after.

### 1. Schema (`schemas/app/schema.prisma`, db push)
- `OrderItem`: add `subscriptionId String? @map("subscription_id")` + relation to `Subscription`.
- `Subscription`: remove `orderItemId` + its relation; the `OrderItem[]` back-relation flips to
  "items this subscription bills." (Local-only / no migrations: db push; dev data is reset or
  backfilled — see Migration.)

### 2. Domain (`internal/core/domain/`)
- `OrderItem`: add `SubscriptionId string`.
- `Subscription`: remove `OrderItemId`. Replace `NewSubscriptionFromOrderItem(item, price)` with
  `NewSubscriptionForCadence(orgId, orderId, customerId, interval, qty, cycles, currency)` (cadence
  comes from the group, not one price). `Amount` becomes the **sum of the group's fixed unit
  prices** (a derived base figure per ADR 0002), or is dropped in favour of deriving from the
  invoice — decide in the plan; lean to "sum of fixed lines."

### 3. Order creation (`internal/core/service/order.go`)
- Delete `subscriptionAnchors`. Add `groupIntoSubscriptions(lines)` → map keyed by cadence.
- For each group: create the subscription (cadence from the group), then set `subscriptionId` on
  each of the group's order items (`UpdateOrderItem`, or create the items already pointing at the
  sub). Publish `subscription.created` per subscription (unchanged).
- `CompleteOrder` / `order_workflow.CompleteCheckoutSession`: stop fetching the single
  `OrderItemId` price for activation — use the subscription's own cadence fields (already set) for
  activation dates.

### 4. Invoice (`internal/core/service/invoice.go`)
- `BuildForBillingPeriod`: load `order_items WHERE subscription_id = sub.id` (new repo method
  `FindOrderItemsBySubscriptionId`). For each: fixed → `BaseLineFromPrice` (trial waives it, ADR
  0003); metered → `UsageForSubscription` → `UsageLineFromPrice`. Remove primary/isOwn entirely.
  Idempotency key (orgId, subId, cycle) unchanged.

### 5. Usage attribution (`internal/core/service/usage.go`, `subscription_repo.go`)
- `FindActiveMeteredForMeter`: change the join from "any item in the subscription's **order**
  carries a metered price for M" to "the subscription **owns** an item with a metered price for M"
  (`order_items.subscription_id = subscriptions.id AND prices.billable_metric_id = M`). Earliest-
  first ordering and the `IncludeUnattributed` catch-all rule are preserved (still "the earliest
  metered subscription for the customer+meter folds in unattributed usage once").

### 6. Current-period usage read (folds in PR #31)
With the group model, `UsageService.CurrentPeriodUsage(sub)` = the subscription's metered lines,
each summed over `[CurrentPeriodStart, CurrentPeriodEnd)`. No order-walk, no anchor. This replaces
the (incorrect) own-price version on PR #31; that PR's endpoint + ingest-rename either rebase onto
this branch or land first and get corrected here.

### 7. HTTP response (`internal/adapter/http/response.go`)
- Subscription response drops `order_item_id`; optionally add the line list / metered summary.

## Migration

`db push`, no migrations, dev DB is hand-seeded. Adding `order_item.subscription_id` is additive;
removing `subscription.order_item_id` is a drop. Existing rows need backfill (point each
subscription's old single item at it, set that item's `subscription_id`). Given local-only/pre-1.0,
the plan will either ship a one-off backfill script or document a clean reseed. No production data.

## Pre-existing bug noted (decide in plan)
Payment-success workflows on **both** engines start the runner for `subs[0]` only
(`FindByOrderId()` then first). With grouping the common order is one subscription, so this stops
mattering for the normal case; a multi-cadence order (2 subscriptions) would still only start the
first runner. The HTTP `CompleteOrder` path already loops all. Decide whether to make both engines'
payment-success loop all subscriptions (recommended) or defer.

## Tests
- **Order creation**: fixed+metered same cadence → one subscription owning both lines; two
  cadences → two subscriptions; one-time line → no subscription / no `subscriptionId`.
- **Invoice**: a subscription with a fixed + a metered line → base + usage line, no primary logic;
  trial waives base but bills usage (ADR 0003); idempotent per cycle.
- **Usage attribution**: `FindActiveMeteredForMeter` keyed on ownership; `IncludeUnattributed`
  still billed once across multiple metered subscriptions.
- **CurrentPeriodUsage**: sums the subscription's metered lines; non-metered/zero-period → empty.
- **Integration** (`//go:build integration`): order with fixed+metered → one subscription, two
  order items carrying its `subscription_id`; invoice round-trip.

## Verification
1. `make test` + `make test-integration` green.
2. `make run`; create an order with a $100 fixed + per-token metered price (one cadence) → one
   subscription; record token usage; `BuildForBillingPeriod` (or the usage read) shows base +
   token usage on that one subscription. Confirm identical behaviour notes for Hatchet and
   Temporal per the parity doc.
