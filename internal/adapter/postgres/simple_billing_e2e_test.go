//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// TestSimpleSubscriptionBilling_E2E pins the fixed (non-metered) recurring billing
// flow end-to-end against real Postgres: charge a due cycle → memory gateway succeeds
// → an invoice is built and marked paid → a payment row links to it → the
// subscription advances exactly one cycle and the next period is scheduled correctly.
// Complements TestBillingChargeAdvancesState, which checks cycle/payment/idempotency
// but never asserts the invoice or the exact schedule.
func TestSimpleSubscriptionBilling_E2E(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	fx := seedSubFixture(t, db, orgId)
	pspConfigId := seedMemoryPsp(t, db, orgId)
	pm := seedPaymentMethod(t, db, orgId, fx.customer.Id)

	// A due, active monthly subscription with explicit period boundaries so we can
	// assert the *exact* next schedule. Period being billed: [Jan 1, Feb 1) 2026.
	periodStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	nextEnd := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC) // Feb 1 + 1 month

	sub := fx.sub
	sub.PspId = domain.Gateway(pspConfigId)
	sub.PaymentMethodId = pm.Id
	sub.Status = domain.SubscriptionStatusActive
	sub.Currency = "USD"
	sub.BillingInterval = domain.BillingIntervalMonth
	sub.BillingIntervalQty = 1
	sub.Cycles = 12
	sub.CyclesProcessed = 0
	sub.StartDate = periodStart
	sub.CurrentPeriodStart = periodStart
	sub.CurrentPeriodEnd = periodEnd
	sub.RenewsAt = periodEnd
	subRow := subscriptionRowFromDomain(sub)
	require.NoError(t, db.Create(&subRow).Error)
	// The per-cycle invoice bills the subscription's OWN lines: stamp the order
	// item with this subscription's id.
	require.NoError(t, db.Model(&orderItemRow{}).
		Where("org_id = ? AND id = ?", orgId, fx.item.Id).
		Update("subscription_id", sub.Id).Error)

	// Expected charge total = the linked price's unit price × quantity (read from DB
	// so the assertion doesn't hard-code seed values).
	items, err := NewOrderRepo(db).FindOrderItemsByOrderId(ctx, orgId, sub.OrderId)
	require.NoError(t, err)
	require.Len(t, items, 1)
	price, err := NewPriceRepo(db).FindById(ctx, orgId, items[0].PriceId)
	require.NoError(t, err)
	qty := int64(items[0].Quantity)
	if qty <= 0 {
		qty = 1
	}
	wantTotal := price.UnitPrice * qty

	svc := buildSubscriptionService(t, db)

	// Charge + apply success (the path the billing-cycle runner uses).
	result, err := svc.ChargeForBillingPeriod(ctx, sub)
	require.NoError(t, err)
	require.Equal(t, domain.PaymentStatusSucceeded, result.Status)
	assert.Equal(t, wantTotal, result.Amount, "charge amount comes from the invoice total")

	updated, err := svc.HandleSubscriptionChargeSuccess(ctx, port.SubscriptionChargeInput{Subscription: sub, ChargeResult: result})
	require.NoError(t, err)

	// --- Invoice: built for the billed cycle, marked paid, one base line. ---
	inv, err := NewInvoiceRepo(db).FindBySubscriptionCycle(ctx, orgId, sub.Id, 0)
	require.NoError(t, err, "an invoice must exist for the billed cycle")
	assert.Equal(t, domain.InvoiceStatusPaid, inv.Status, "invoice marked paid after a successful charge")
	assert.Equal(t, wantTotal, inv.Total)
	assert.Equal(t, wantTotal, inv.Subtotal)
	assert.Equal(t, sub.Id, inv.SubscriptionId)
	assert.Equal(t, fx.customer.Id, inv.CustomerId)
	assert.True(t, inv.PeriodStart.Equal(periodStart), "invoice bills the just-closed period start")
	assert.True(t, inv.PeriodEnd.Equal(periodEnd), "invoice bills the just-closed period end")
	require.Len(t, inv.LineItems, 1, "fixed price → exactly one base line")
	assert.Equal(t, domain.InvoiceLineKindBase, inv.LineItems[0].Kind)
	assert.Equal(t, wantTotal, inv.LineItems[0].Total)

	// --- Payment: succeeded, settles the invoice. ---
	payments, total, err := NewPaymentRepo(db).FindBySubscriptionId(ctx, orgId, sub.Id, domain.Pagination{Page: 1, Limit: 10})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	assert.Equal(t, domain.PaymentStatusSucceeded, payments[0].Status)
	assert.Equal(t, inv.Id, payments[0].InvoiceId, "payment links to the invoice it settled")
	assert.Equal(t, inv.Total, payments[0].NetAmount)

	// --- Schedule: exactly one cycle advanced; next period [Feb 1, Mar 1). ---
	assert.Equal(t, 1, updated.CyclesProcessed)
	assert.Equal(t, domain.SubscriptionStatusActive, updated.Status)
	assert.Equal(t, inv.Total, updated.TotalRevenue)

	persisted, err := NewSubscriptionRepo(db).FindById(ctx, orgId, sub.Id)
	require.NoError(t, err)
	assert.Equal(t, 1, persisted.CyclesProcessed, "cycle advance is durable")
	assert.True(t, persisted.CurrentPeriodStart.Equal(periodEnd), "new period starts at the old period end")
	assert.True(t, persisted.CurrentPeriodEnd.Equal(nextEnd), "new period ends one interval later")
	assert.True(t, persisted.RenewsAt.Equal(nextEnd), "renews at the new period end")
	assert.False(t, persisted.LastCharge.IsZero(), "last-charge timestamp recorded")
}
