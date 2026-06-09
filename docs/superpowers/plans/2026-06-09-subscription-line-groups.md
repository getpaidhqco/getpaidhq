# Subscription Owns a Group of Order Lines — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the 1:1 `Subscription`↔`OrderItem` link with "a subscription owns the recurring order lines that share one billing cadence," so flat + metered bill together with no `primary`/`isOwn`/`subscriptionAnchors` hacks.

**Architecture:** Add nullable `OrderItem.SubscriptionId`; group an order's recurring lines by cadence at order-creation, one subscription per group. Invoicing and usage read a subscription's *own* lines. Drop `Subscription.OrderItemId` and `Subscription.Amount` (per ADR 0002). All logic stays in `core/`, so Hatchet and Temporal inherit it unchanged; the only adapter change is making both engines' payment-success start a runner per subscription.

**Tech Stack:** Go 1.24, GORM, Prisma schema via `db push` (no migrations), testify, `//go:build integration` Testcontainers.

**Spec:** `docs/superpowers/specs/2026-06-08-subscription-owns-order-line-group-design.md`

**Branch:** `feat/subscription-line-groups` (already created off `main`).

**Run tests:** `make test` (unit), `make test-integration` (Postgres), `go build ./...`, `go vet ./...`.

---

## Task 1: Schema — order_item.subscription_id; drop subscription.order_item_id + amount

**Files:**
- Modify: `schemas/app/schema.prisma` (Product/OrderItem ~399-430, Subscription ~524-578)

- [ ] **Step 1: Edit `OrderItem`** — add the nullable back-link and relation:

```prisma
model OrderItem {
  // ...existing fields...
  subscriptionId String?       @map("subscription_id")
  subscription   Subscription? @relation("SubscriptionLines", fields: [orgId, subscriptionId], references: [orgId, id])
  // remove the old `Subscription Subscription[]` back-relation line
}
```

- [ ] **Step 2: Edit `Subscription`** — remove `orderItemId`, its relation, and `amount`; add the lines back-relation:

```prisma
model Subscription {
  // remove: orderItemId String @map("order_item_id")
  // remove: orderItem   OrderItem @relation(...)
  // remove: amount   Int
  lines OrderItem[] @relation("SubscriptionLines")
  // keep orderId + order relation, cadence fields, etc.
}
```

- [ ] **Step 3: Validate the schema**

Run: `node_modules/.bin/prisma validate --schema schemas/app/schema.prisma`
Expected: `The schema ... is valid 🚀`

- [ ] **Step 4: Commit**

```bash
git add schemas/app/schema.prisma
git commit -m "schema(subscription): order_item.subscription_id; drop order_item_id + amount"
```

> DB apply (`make db-push`) happens in Task 12 after the code compiles; integration tests use AutoMigrate of the row structs, not Prisma.

---

## Task 2: Domain — OrderItem.SubscriptionId, Subscription drops OrderItemId/Amount, cadence constructor

**Files:**
- Modify: `internal/core/domain/order_item.go`
- Modify: `internal/core/domain/subscription.go` (struct ~26-66, `NewSubscriptionFromOrderItem` ~349, `SetActivationDates` ~257)
- Test: `internal/core/domain/subscription_test.go`

- [ ] **Step 1: Add `SubscriptionId` to `OrderItem`** (`order_item.go`), after `PriceId`:

```go
	PriceId        string
	SubscriptionId string // the subscription that bills this recurring line; "" for one-time lines
```

- [ ] **Step 2: Edit `Subscription` struct** (`subscription.go`): remove the `OrderItemId string` field and the `Amount int64` field.

- [ ] **Step 3: Write the failing test** for the new cadence constructor (`subscription_test.go`):

```go
func TestNewSubscriptionForCadence(t *testing.T) {
	sub := domain.NewSubscriptionForCadence("org_1", "ord_1", "cus_1", domain.BillingIntervalMonth, 1, 12, "USD")
	assert.Equal(t, "org_1", sub.OrgId)
	assert.Equal(t, "ord_1", sub.OrderId)
	assert.Equal(t, "cus_1", sub.CustomerId)
	assert.Equal(t, domain.BillingIntervalMonth, sub.BillingInterval)
	assert.Equal(t, 1, sub.BillingIntervalQty)
	assert.Equal(t, domain.SubscriptionStatusPending, sub.Status)
	assert.NotEmpty(t, sub.Id)
}
```

- [ ] **Step 4: Run it — expect FAIL** (`NewSubscriptionForCadence` undefined): `go test ./internal/core/domain/ -run TestNewSubscriptionForCadence`

- [ ] **Step 5: Replace `NewSubscriptionFromOrderItem`** with `NewSubscriptionForCadence` (`subscription.go`):

```go
// NewSubscriptionForCadence creates a pending subscription for one billing
// cadence within an order. Its lines (order items) are linked separately by
// setting OrderItem.SubscriptionId. The subscription stores no charge amount
// (ADR 0002) — the per-cycle total is computed onto the Invoice.
func NewSubscriptionForCadence(orgId, orderId, customerId string, interval BillingInterval, qty, cycles int, currency string) Subscription {
	return Subscription{
		OrgId:              orgId,
		Id:                 lib.GenerateId("sub"),
		OrderId:            orderId,
		CustomerId:         customerId,
		Status:             SubscriptionStatusPending,
		BillingInterval:    interval,
		BillingIntervalQty: qty,
		Cycles:             cycles,
		Currency:           currency,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
}
```

- [ ] **Step 6: Change `SetActivationDates`** to use the subscription's own cadence instead of a passed `Price`. New signature `func (s *Subscription) SetActivationDates() *Subscription`, using `s.BillingInterval` / `s.BillingIntervalQty` where it previously read `price.BillingInterval` / `price.BillingIntervalQty`. (Trial fields that came from the price move to Task 5's group resolution if needed.)

- [ ] **Step 7: Run domain tests** — fix any compile fallout in `subscription_test.go` / `subscription_state_test.go` that referenced `OrderItemId`, `Amount`, or `NewSubscriptionFromOrderItem`. Expected: PASS.

Run: `go test ./internal/core/domain/...`

- [ ] **Step 8: Commit**

```bash
git add internal/core/domain/
git commit -m "domain: OrderItem.SubscriptionId; subscription drops OrderItemId/Amount; cadence constructor"
```

---

## Task 3: Row mappers — order_item_row + subscription_row

**Files:**
- Modify: `internal/adapter/postgres/order_item_row.go`
- Modify: `internal/adapter/postgres/subscription_row.go` (Amount col ~42, mappers ~59/79/95/115)

- [ ] **Step 1:** In `order_item_row.go` add `SubscriptionId string \`gorm:"column:subscription_id"\`` to the struct and map it in both `toDomain` and `...FromDomain`.

- [ ] **Step 2:** In `subscription_row.go` remove the `Amount int64 \`gorm:"column:amount"\`` field and the `OrderItemId` field, and remove `Amount:`/`OrderItemId:` from both `toDomain` and `...FromDomain`.

- [ ] **Step 3: Build** — `go build ./internal/adapter/postgres/` (will fail later until repos updated; that's expected at this step, so just confirm these two files have no leftover references).

- [ ] **Step 4: Commit**

```bash
git add internal/adapter/postgres/order_item_row.go internal/adapter/postgres/subscription_row.go
git commit -m "postgres(rows): order_item.subscription_id; drop subscription order_item_id/amount"
```

---

## Task 4: Repository — FindOrderItemsBySubscriptionId + rewrite FindActiveMeteredForMeter

**Files:**
- Modify: `internal/core/port/repository.go` (OrderRepository + SubscriptionRepository interfaces)
- Modify: `internal/adapter/postgres/order_repo.go`
- Modify: `internal/adapter/postgres/subscription_repo.go` (FindActiveMeteredForMeter ~80-104)
- Test: `internal/adapter/postgres/subscription_repo_test.go` (integration) or a new `order_repo_test.go` case

- [ ] **Step 1: Add to `port.OrderRepository`:**

```go
	FindOrderItemsBySubscriptionId(ctx context.Context, orgId, subscriptionId string) ([]domain.OrderItem, error)
```

- [ ] **Step 2: Implement it** in `order_repo.go` (mirror `FindOrderItemsByOrderId`, filtering `WHERE subscription_id = ?`).

- [ ] **Step 3: Rewrite `FindActiveMeteredForMeter`** (`subscription_repo.go`) to key on subscription *ownership* of the metered line, not the whole order:

```go
	err := dbFromCtx(ctx, r.db).
		Model(&subscriptionRow{}).
		Distinct("subscriptions.*").
		Joins("JOIN order_items oi ON oi.org_id = subscriptions.org_id AND oi.subscription_id = subscriptions.id").
		Joins("JOIN prices p ON p.org_id = oi.org_id AND p.id = oi.price_id").
		Where("subscriptions.org_id = ? AND subscriptions.customer_id = ?", orgId, customerId).
		Where("p.billable_metric_id = ?", billableMetricId).
		Where("subscriptions.status IN ?", []string{
			string(domain.SubscriptionStatusActive),
			string(domain.SubscriptionStatusTrial),
			string(domain.SubscriptionStatusPastDue),
		}).
		Order("subscriptions.start_date ASC, subscriptions.created_at ASC").
		Find(&rows).Error
```

(Join changed from `oi.order_id = subscriptions.order_id` to `oi.subscription_id = subscriptions.id`.)

- [ ] **Step 4: Integration test** (`//go:build integration`): seed an order with a subscription owning a metered line (set `order_item.subscription_id`), assert `FindActiveMeteredForMeter` returns it; a metered line owned by a *different* subscription is not returned. Use `testDB(t)` + `uniqueOrg`/`cleanupOrg`; register `&orderItemRow{}` and `&subscriptionRow{}` (already in `allModels`).

- [ ] **Step 5: Run** `go test -tags=integration -run 'FindActiveMetered' ./internal/adapter/postgres/...` — expect PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/core/port/repository.go internal/adapter/postgres/
git commit -m "repo: FindOrderItemsBySubscriptionId; attribute metered usage by subscription ownership"
```

---

## Task 5: Order creation — group recurring lines by cadence

**Files:**
- Modify: `internal/core/service/order.go` (`subscriptionAnchors` ~606-630, `startSubscription` ~228-238, loop ~272-277, `orderLine` ~601)
- Test: `internal/core/service/order_test.go`

- [ ] **Step 1: Write failing tests** for grouping (table-driven, `order_test.go`):

```go
func TestGroupIntoSubscriptions(t *testing.T) {
	monthlyFixed := domain.Price{Id: "p_fix", Category: domain.PriceCategorySubscription, BillingInterval: domain.BillingIntervalMonth, BillingIntervalQty: 1}
	monthlyMetered := domain.Price{Id: "p_met", Category: domain.PriceCategorySubscription, BillingInterval: domain.BillingIntervalMonth, BillingIntervalQty: 1, BillableMetricId: "m1"}
	yearlyFixed := domain.Price{Id: "p_yr", Category: domain.PriceCategorySubscription, BillingInterval: domain.BillingIntervalYear, BillingIntervalQty: 1}
	oneTime := domain.Price{Id: "p_one", Category: domain.OneTime, BillingInterval: domain.BillingIntervalNone}

	line := func(id string, p domain.Price) orderLine { return orderLine{item: domain.OrderItem{Id: id}, price: p} }

	t.Run("flat + metered same cadence => one group of two", func(t *testing.T) {
		groups := groupIntoSubscriptions([]orderLine{line("a", monthlyFixed), line("b", monthlyMetered)})
		require.Len(t, groups, 1)
		require.Len(t, groups[0], 2)
	})
	t.Run("two cadences => two groups", func(t *testing.T) {
		groups := groupIntoSubscriptions([]orderLine{line("a", monthlyFixed), line("c", yearlyFixed)})
		require.Len(t, groups, 2)
	})
	t.Run("one-time line is not grouped", func(t *testing.T) {
		groups := groupIntoSubscriptions([]orderLine{line("o", oneTime)})
		require.Empty(t, groups)
	})
}
```

- [ ] **Step 2: Run — expect FAIL** (`groupIntoSubscriptions` undefined): `go test ./internal/core/service/ -run TestGroupIntoSubscriptions`

- [ ] **Step 3: Replace `subscriptionAnchors` with `groupIntoSubscriptions`** (`order.go`). A line is recurring iff its price has a real billing interval; group by `(interval, qty)`; preserve line order within a group; return groups in first-seen cadence order:

```go
// groupIntoSubscriptions partitions an order's recurring lines (any price with a
// real billing interval — fixed-subscription or metered) into one group per
// billing cadence. Each group becomes one subscription that bills all its lines.
// One-time / free / no-interval lines are not grouped (charged once, no subscription).
func groupIntoSubscriptions(lines []orderLine) [][]orderLine {
	type cadence struct {
		interval domain.BillingInterval
		qty      int
	}
	order := []cadence{}
	byCadence := map[cadence][]orderLine{}
	for _, l := range lines {
		if l.price.BillingInterval == "" || l.price.BillingInterval == domain.BillingIntervalNone {
			continue
		}
		c := cadence{l.price.BillingInterval, l.price.BillingIntervalQty}
		if _, ok := byCadence[c]; !ok {
			order = append(order, c)
		}
		byCadence[c] = append(byCadence[c], l)
	}
	groups := make([][]orderLine, 0, len(order))
	for _, c := range order {
		groups = append(groups, byCadence[c])
	}
	return groups
}
```

- [ ] **Step 4: Run grouping test — expect PASS.**

- [ ] **Step 5: Rewire subscription creation** in `CreateOrder` (replace the `startSubscription` closure + anchor loop, ~228-277). For each group: build the subscription from the group's cadence, create it, then stamp `SubscriptionId` on each line's order item:

```go
for _, group := range groupIntoSubscriptions(lines) {
	head := group[0].price
	sub := domain.NewSubscriptionForCadence(orgId, orderId, customerEntity.Id, head.BillingInterval, head.BillingIntervalQty, head.Cycles, currency)
	sub.PspId = input.PspId
	sub.PaymentMethodId = input.PaymentMethodId
	created, err := s.subscriptionRepository.Create(ctx, sub)
	if err != nil {
		s.logger.Error("Failed to create subscription", "err", err.Error())
		return domain.CreateOrderResponse{}, err
	}
	for _, l := range group {
		item := l.item
		item.SubscriptionId = created.Id
		if _, err := s.orderRepository.UpdateOrderItem(ctx, item); err != nil {
			s.logger.Error("Failed to link order item to subscription", "item", item.Id, "err", err.Error())
			return domain.CreateOrderResponse{}, err
		}
	}
	_ = s.pubsub.Publish(orgId, port.TopicSubscriptionCreated, created)
}
```

- [ ] **Step 6: Fix `CompleteOrder` / `order_workflow.CompleteCheckoutSession`** — they currently fetch `FindOrderItemById(sub.OrderItemId)` to get a price for activation. Replace with `sub.SetActivationDates()` (cadence already on the subscription). Remove the `OrderItemId` reads.

- [ ] **Step 7: Run** `go test ./internal/core/service/ -run 'Order'` — fix fallout. Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/core/service/order.go internal/core/service/order_workflow.go internal/core/service/order_test.go
git commit -m "order: group recurring lines by cadence into subscriptions; drop subscriptionAnchors"
```

---

## Task 6: Invoice — bill the subscription's own lines; delete primary/isOwn

**Files:**
- Modify: `internal/core/service/invoice.go` (`BuildForBillingPeriod` ~48-136)
- Test: `internal/core/service/invoice_test.go`

- [ ] **Step 1: Update/replace the failing test** in `invoice_test.go` so the fake order repo returns the subscription's lines via `FindOrderItemsBySubscriptionId` (a fixed line + a metered line), and assert the invoice has a base line + a usage line. Add a case: a `trial` subscription waives the base line but keeps the usage line (ADR 0003).

- [ ] **Step 2: Run — expect FAIL** (still calling old logic): `go test ./internal/core/service/ -run Invoice`

- [ ] **Step 3: Rewrite the body** of `BuildForBillingPeriod` after the idempotency check:

```go
	items, err := s.orderRepository.FindOrderItemsBySubscriptionId(ctx, sub.OrgId, sub.Id)
	if err != nil {
		return domain.Invoice{}, err
	}
	inv := domain.NewInvoice(sub, sub.CurrentPeriodStart, sub.CurrentPeriodEnd)
	for _, it := range items {
		price, perr := s.priceRepository.FindById(ctx, sub.OrgId, it.PriceId)
		if perr != nil {
			return domain.Invoice{}, perr
		}
		if price.IsMetered() {
			units, uerr := s.usageService.UsageForSubscription(ctx, sub, price, sub.CurrentPeriodStart, sub.CurrentPeriodEnd)
			if uerr != nil {
				return domain.Invoice{}, uerr
			}
			inv.AddLine(domain.UsageLineFromPrice(sub.OrgId, inv.Id, price, units))
			continue
		}
		// Fixed line — trial waives the flat fee (ADR 0003).
		if sub.Status != domain.SubscriptionStatusTrial {
			qty := int64(it.Quantity)
			if qty <= 0 {
				qty = 1
			}
			inv.AddLine(domain.BaseLineFromPrice(sub.OrgId, inv.Id, price, decimal.NewFromInt(qty)))
		}
	}
```

Delete `primaryItemId`, `isPrimary`, `isOwn`, and the `FindOrderItemsByOrderId` walk.

- [ ] **Step 4: Run** `go test ./internal/core/service/ -run Invoice` — expect PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/core/service/invoice.go internal/core/service/invoice_test.go
git commit -m "invoice: bill the subscription's own lines; remove primary/isOwn order-walk"
```

---

## Task 7: Usage — UsageForSubscription unchanged contract; add CurrentPeriodUsage over the sub's lines

**Files:**
- Modify: `internal/core/service/usage.go`
- Test: `internal/core/service/usage_test.go`

> `UsageForSubscription(sub, price, from, to)` already works per-meter and is unchanged. Add the period read keyed on the subscription's own metered lines (this supersedes PR #31's order-item version).

- [ ] **Step 1: Ensure `UsageService` has `orderRepository` + `priceRepository`** (constructor params, wired in Task 11). If PR #31 already added them on its branch, mirror here.

- [ ] **Step 2: Write the failing test** (`usage_test.go`): a subscription owning one metered line (`FindOrderItemsBySubscriptionId` returns it; price metered; event store `count: 9`) → `CurrentPeriodUsage` returns one meter with quantity 9; zero period → empty; a subscription owning only a fixed line → empty.

- [ ] **Step 3: Implement `CurrentPeriodUsage`:**

```go
func (s *UsageService) CurrentPeriodUsage(ctx context.Context, orgId, subscriptionId string) (SubscriptionUsage, error) {
	sub, err := s.subscriptionRepository.FindById(ctx, orgId, subscriptionId)
	if err != nil {
		return SubscriptionUsage{}, err
	}
	out := SubscriptionUsage{SubscriptionId: sub.Id, CurrentPeriodStart: sub.CurrentPeriodStart, CurrentPeriodEnd: sub.CurrentPeriodEnd, Meters: []MeterUsage{}}
	if sub.CurrentPeriodStart.IsZero() {
		return out, nil
	}
	items, err := s.orderRepository.FindOrderItemsBySubscriptionId(ctx, orgId, sub.Id)
	if err != nil {
		return SubscriptionUsage{}, err
	}
	for _, it := range items {
		price, perr := s.priceRepository.FindById(ctx, orgId, it.PriceId)
		if perr != nil {
			return SubscriptionUsage{}, perr
		}
		if !price.IsMetered() {
			continue
		}
		units, uerr := s.UsageForSubscription(ctx, sub, price, sub.CurrentPeriodStart, sub.CurrentPeriodEnd)
		if uerr != nil {
			return SubscriptionUsage{}, uerr
		}
		metric, merr := s.meterRepository.FindById(ctx, orgId, price.BillableMetricId)
		if merr != nil {
			return SubscriptionUsage{}, merr
		}
		out.Meters = append(out.Meters, MeterUsage{MetricCode: metric.Code, Aggregation: metric.Aggregation, Quantity: units})
	}
	return out, nil
}
```

(Plus the `MeterUsage` / `SubscriptionUsage` structs if not already present from PR #31.)

- [ ] **Step 4: Run** `go test ./internal/core/service/ -run Usage` — expect PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/core/service/usage.go internal/core/service/usage_test.go
git commit -m "usage: current-period usage over the subscription's own metered lines"
```

---

## Task 8: Proration + revenue — derive base from fixed lines, not subscription.Amount

**Files:**
- Modify: `internal/core/service/subscription.go` (UpdateBillingAnchor caller ~386)
- Modify: `internal/core/service/order_workflow.go` (TotalRevenue ~127)
- Modify: `internal/adapter/temporal/activities/order_activities.go` (log ~83)
- Modify: `internal/core/domain/subscription.go` (`UpdateBillingAnchor` signature if it took the amount)
- Test: `internal/core/service/subscription_test.go`

- [ ] **Step 1:** Add a helper on `SubscriptionService` to compute the recurring fixed base for a subscription (sum of its fixed lines' unit prices):

```go
func (s *SubscriptionService) fixedBaseAmount(ctx context.Context, sub domain.Subscription) (int64, error) {
	items, err := s.orderRepository.FindOrderItemsBySubscriptionId(ctx, sub.OrgId, sub.Id)
	if err != nil {
		return 0, err
	}
	var total int64
	for _, it := range items {
		p, perr := s.priceRepository.FindById(ctx, sub.OrgId, it.PriceId)
		if perr != nil {
			return 0, perr
		}
		if !p.IsMetered() {
			q := int64(it.Quantity)
			if q <= 0 {
				q = 1
			}
			total += p.UnitPrice * q
		}
	}
	return total, nil
}
```

- [ ] **Step 2: Proration** — at subscription.go:386, replace `subscription.Amount` with the computed base from `fixedBaseAmount(ctx, subscription)`. If `UpdateBillingAnchor` (domain) took the amount as a param, keep the param and pass the computed base.

- [ ] **Step 3: Revenue** — `order_workflow.go:127` `subscription.TotalRevenue = subscription.Amount` becomes the activation invoice total (or the computed base if no invoice yet); simplest: `tr, _ := fixedBaseAmount(...)`. Prefer the first invoice total if available — pick one and make it explicit.

- [ ] **Step 4: Temporal log** — `order_activities.go:83`: drop the `"Total", currentSub.Amount` field (or log `currentSub.Id` only).

- [ ] **Step 5:** Update any tests referencing `subscription.Amount`. Run `go test ./internal/core/... ./internal/adapter/temporal/...`. Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/core/service/subscription.go internal/core/service/order_workflow.go internal/adapter/temporal/activities/order_activities.go internal/core/domain/subscription.go
git commit -m "billing: derive recurring base from fixed lines; drop subscription.Amount reads"
```

---

## Task 9: Engine parity — both payment-success workflows start every subscription's runner

**Files:**
- Modify: `internal/adapter/hatchet/workflows/payment_success.go` (~48-69)
- Modify: `internal/adapter/temporal/workflows/payment_success.go` (~56-70)

- [ ] **Step 1: Hatchet** — change the `start-subscription-lifecycle` step from using `subs[0]` to looping all subscriptions, calling `engine.StartSubscriptionWorkflow` (idempotent via deterministic id) for each.

- [ ] **Step 2: Temporal** — change the workflow to start a `SubscriptionWorkflow` detached child per subscription (deterministic id per `sub.Id`), not just `subs[0]`.

- [ ] **Step 3:** Run `go build ./...` and any workflow tests: `go test ./internal/adapter/hatchet/... ./internal/adapter/temporal/...`. Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/adapter/hatchet/workflows/payment_success.go internal/adapter/temporal/workflows/payment_success.go
git commit -m "engines: start a runner for every order subscription (both Hatchet and Temporal)"
```

---

## Task 10: HTTP response — drop subscription order_item_id

**Files:**
- Modify: `internal/adapter/http/response.go` (~171)
- Test: `internal/adapter/http/subscription_handler_test.go` (if it asserts the field)

- [ ] **Step 1:** Remove `OrderItemId: s.OrderItemId` from the subscription response DTO and the field from the struct.

- [ ] **Step 2:** Run `go test ./internal/adapter/http/...`. Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/adapter/http/response.go
git commit -m "http: drop subscription order_item_id from response"
```

---

## Task 11: Wiring — UsageService gets order/price repos; full build

**Files:**
- Modify: `internal/config/app.go` (`NewUsageService` call)

- [ ] **Step 1:** Pass `orderRepo` + `priceRepo` into `service.NewUsageService(...)` (matching Task 7's constructor).

- [ ] **Step 2: Full build + vet + unit tests:**

Run: `go build ./... && go vet ./... && make test`
Expected: all pass.

- [ ] **Step 3: Commit**

```bash
git add internal/config/app.go
git commit -m "wire: UsageService order/price repos"
```

---

## Task 12: DB apply + integration + end-to-end verification

**Files:** none (verification)

- [ ] **Step 1: Backfill plan for the dev DB.** Existing rows: each old subscription has `order_item_id`; set that item's `subscription_id` to the sub, then drop the columns. Since this is local-only/pre-1.0 with hand-seeded data, the simplest correct path is a clean reseed: `make db-push` (Prisma will warn it drops `subscriptions.order_item_id`/`amount`; accept). If the seeded subscriptions matter, run a one-off SQL backfill first:

```sql
UPDATE order_items oi SET subscription_id = s.id
FROM subscriptions s WHERE s.order_item_id = oi.id AND s.org_id = oi.org_id;
```

(run before `db push` drops the column).

- [ ] **Step 2: Apply schema** — `make db-push`. Expected: in sync.

- [ ] **Step 3: Integration suite** — `make test-integration`. Expected: PASS (register `orderItemRow`/`subscriptionRow` already done; the new `subscription_id` column round-trips).

- [ ] **Step 4: Manual end-to-end** — `make run`, then:
  - Create an order with two prices on one monthly cadence: `A = $100 fixed`, `B = per-token metered`. Confirm **one** subscription is created and both order items carry its `subscription_id`.
  - Record token usage for the customer/subscription via `POST /api/usage/ingest`.
  - Confirm the subscription's invoice for the period (or `GET /api/subscriptions/{id}/usage`) shows the base $100 + token usage — on that single subscription, with no `primary` logic involved.
  - Add a yearly add-on price → second order item on a different cadence → confirm a **second** subscription.

- [ ] **Step 5: Engine parity note** — re-read `docs/internal/engine-parity-and-subscription-lifecycle.md`; confirm the change is core-only and both engines start a runner per subscription. Update that doc + `CONTEXT.md` (Subscription entry: agreement = one cadence's lines; no single anchor item) in this task's commit.

- [ ] **Step 6: Final commit**

```bash
git add CONTEXT.md docs/
git commit -m "docs: subscription owns a cadence's order lines; update CONTEXT + parity notes"
```

---

## Notes for the implementer

- **PR #31** (usage ingest rename + the old order-item usage read) is on `feat/usage-ingest-batch-and-subscription-usage`. Either land it first and let Task 7 correct the read, or rebase it onto this branch. Don't duplicate the ingest-rename here.
- **No `subscription.Amount` anywhere** after Task 8 — grep `\.Amount` on subscriptions to confirm zero readers before Task 12.
- Keep behaviour identical on Hatchet and Temporal (parity rule). The only adapter change is Task 9.
