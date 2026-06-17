//go:build integration

// Billing lifecycle e2e against real Postgres: what happens around the happy
// charge — a declined charge (retry schedule, unpaid invoice, failed payment),
// two consecutive real billing cycles over a carry-over meter, the trial
// flat-fee waiver (ADR 0003), the metered cadence clamp at subscription
// creation, and a period with zero usage.
//
// Dunning campaign creation is engine-owned (DunningOrchestrationService needs
// a workflow engine), so the failure test asserts up to this layer's boundary:
// HandleSubscriptionChargeFailure's persisted outcome.
package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"getpaidhq/internal/adapter/memory"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
	"getpaidhq/internal/lib"
)

// seedDecliningCard wires the subscription to the memory gateway with a payment
// method carrying the gateway's decline token, so every charge fails as a
// retryable card error.
func seedDecliningCard(t *testing.T, db *gorm.DB, orgId string, sub *domain.Subscription) {
	t.Helper()
	pspConfigId := seedMemoryPsp(t, db, orgId)
	now := time.Now().UTC().Truncate(time.Microsecond)
	pm := domain.PaymentMethod{
		OrgId:      orgId,
		Id:         lib.GenerateId("pm"),
		Status:     domain.PaymentMethodStatusActive,
		Psp:        string(domain.Memory),
		Name:       "Visa ****0002 (declines)",
		CustomerId: sub.CustomerId,
		Type:       domain.PaymentMethodTypeCard,
		Token:      memory.DeclineToken,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	row := paymentMethodRowFromDomain(pm)
	require.NoError(t, db.Create(&row).Error)
	sub.PspId = domain.Gateway(pspConfigId)
	sub.PaymentMethodId = pm.Id
	require.NoError(t, db.Model(&subscriptionRow{}).
		Where("org_id = ? AND id = ?", orgId, sub.Id).
		Updates(map[string]any{"psp_id": pspConfigId, "payment_method_id": pm.Id}).Error)
}

// buildInvoiceService mirrors buildSubscriptionService's invoice wiring for
// tests that drive the invoice builder directly.
func buildInvoiceService(t *testing.T, db *gorm.DB) *service.InvoiceService {
	t.Helper()
	return service.NewInvoiceService(NewInvoiceRepo(db), NewOrderRepo(db), NewPriceRepo(db),
		buildUsageService(t, db), NewTxManager(db), noopLogger{})
}

// A declined charge: the charge result is failed (not an error), the failure
// handler records a failed payment, marks the invoice unpaid, and schedules the
// first retry — subscription past_due, Retries = 1, NextRetryAt set (default
// policy: 3 attempts).
func TestChargeFailure_PastDueWithRetryScheduled_E2E(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	jan1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	feb1 := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	fx := seedUsageFixture(t, db, orgId,
		domain.BillableMetric{Code: "api_calls", Name: "API calls", Aggregation: domain.AggregationCount},
		domain.Price{Label: "API", Scheme: domain.Fixed, UnitPrice: 10},
		jan1, feb1)
	seedDecliningCard(t, db, orgId, &fx.sub)

	usage := buildUsageService(t, db)
	for _, ext := range []string{"cf1", "cf2"} {
		recordUsage(t, usage, port.RecordEventInput{
			OrgId: orgId, CustomerId: fx.customer.Id, MetricCode: fx.meter.Code,
			SubscriptionId: fx.sub.Id, ExternalId: ext, Timestamp: jan1.Add(time.Hour),
		})
	}

	svc := buildSubscriptionService(t, db)
	result, err := svc.ChargeForBillingPeriod(ctx, fx.sub)
	require.NoError(t, err, "a decline is a failed result, not an error")
	assert.Equal(t, domain.PaymentStatusFailed, result.Status)
	assert.Equal(t, "card_declined", result.ErrorCode)

	updated, err := svc.HandleSubscriptionChargeFailure(ctx, port.SubscriptionChargeInput{
		Subscription: fx.sub, ChargeResult: result,
	})
	require.NoError(t, err)
	assert.Equal(t, domain.SubscriptionStatusPastDue, updated.Status)
	assert.Equal(t, 1, updated.Retries)
	assert.False(t, updated.NextRetryAt.IsZero(), "first retry must be scheduled")

	persisted, err := NewSubscriptionRepo(db).FindById(ctx, orgId, fx.sub.Id)
	require.NoError(t, err)
	assert.Equal(t, domain.SubscriptionStatusPastDue, persisted.Status)
	assert.Equal(t, 0, persisted.CyclesProcessed, "a failed cycle must not advance")

	inv, err := NewInvoiceRepo(db).FindBySubscriptionCycle(ctx, orgId, fx.sub.Id, 0)
	require.NoError(t, err)
	assert.Equal(t, domain.InvoiceStatusUnpaid, inv.Status)

	payments, total, err := NewPaymentRepo(db).FindBySubscriptionId(ctx, orgId, fx.sub.Id, domain.Pagination{Page: 1, Limit: 10})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	assert.Equal(t, domain.PaymentStatusFailed, payments[0].Status)
	assert.Equal(t, inv.Id, payments[0].InvoiceId)
}

// Two consecutive REAL billing cycles over a carry-over meter: June bills the
// two seats standing since May; the success handler advances the period; a
// third seat joins in July and July's charge bills three — each cycle gets its
// own invoice, and the standing seats carry across the cycle boundary without
// emitting new events.
func TestMultiCycleBilling_CarryOver_E2E(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	jun1 := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	jul1 := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	aug1 := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	fx := seedUsageFixture(t, db, orgId,
		domain.BillableMetric{Code: "seats", Name: "Seats",
			Aggregation: domain.AggregationLatest, FieldName: "seat_id", CarryOver: true},
		domain.Price{Label: "Seat", Scheme: domain.Fixed, UnitPrice: 1000},
		jun1, jul1)
	seedMemoryPspForSub(t, db, orgId, &fx.sub)

	usage := buildUsageService(t, db)
	seat := func(extId, seatId, op string, ts time.Time) {
		recordUsage(t, usage, port.RecordEventInput{
			OrgId: orgId, CustomerId: fx.customer.Id, MetricCode: fx.meter.Code,
			SubscriptionId: fx.sub.Id, ExternalId: extId, Timestamp: ts,
			Metadata: map[string]string{domain.UsageOperationKey: op, "seat_id": seatId},
		})
	}
	may10 := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	seat("mc1", "ana", domain.UsageOperationAdd, may10)
	seat("mc2", "ben", domain.UsageOperationAdd, may10)

	svc := buildSubscriptionService(t, db)

	// Cycle 0: June bills the two standing seats.
	result, err := svc.ChargeForBillingPeriod(ctx, fx.sub)
	require.NoError(t, err)
	require.Equal(t, domain.PaymentStatusSucceeded, result.Status)
	assert.Equal(t, int64(2*1000), result.Amount)
	_, err = svc.HandleSubscriptionChargeSuccess(ctx, port.SubscriptionChargeInput{Subscription: fx.sub, ChargeResult: result})
	require.NoError(t, err)

	advanced, err := NewSubscriptionRepo(db).FindById(ctx, orgId, fx.sub.Id)
	require.NoError(t, err)
	assert.Equal(t, 1, advanced.CyclesProcessed)
	assert.True(t, advanced.CurrentPeriodStart.Equal(jul1), "period advanced to July, got %s", advanced.CurrentPeriodStart)
	assert.True(t, advanced.CurrentPeriodEnd.Equal(aug1), "got %s", advanced.CurrentPeriodEnd)

	// Cycle 1: a third seat joins in July; the May seats emit no new events.
	seat("mc3", "cyn", domain.UsageOperationAdd, time.Date(2026, 7, 5, 0, 0, 0, 0, time.UTC))

	result, err = svc.ChargeForBillingPeriod(ctx, advanced)
	require.NoError(t, err)
	require.Equal(t, domain.PaymentStatusSucceeded, result.Status)
	assert.Equal(t, int64(3*1000), result.Amount, "July bills the carried seats plus the new one")

	invJun, err := NewInvoiceRepo(db).FindBySubscriptionCycle(ctx, orgId, fx.sub.Id, 0)
	require.NoError(t, err)
	require.Len(t, invJun.LineItems, 1)
	assert.True(t, invJun.LineItems[0].Quantity.Equal(decimal.NewFromInt(2)), "June quantity, got %s", invJun.LineItems[0].Quantity)

	invJul, err := NewInvoiceRepo(db).FindBySubscriptionCycle(ctx, orgId, fx.sub.Id, 1)
	require.NoError(t, err)
	assert.Equal(t, 1, invJul.Cycle)
	require.Len(t, invJul.LineItems, 1)
	assert.True(t, invJul.LineItems[0].Quantity.Equal(decimal.NewFromInt(3)), "July quantity, got %s", invJul.LineItems[0].Quantity)
	assert.Equal(t, int64(3000), invJul.Total)
}

// A trial subscription's invoice waives the flat fee (ADR 0003): only the
// usage line is billed while Status == trial.
func TestTrialSubscription_FlatFeeWaived_E2E(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	apr1 := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	may1 := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	fx := seedUsageFixture(t, db, orgId,
		domain.BillableMetric{Code: "overage_units", Name: "Overage", Aggregation: domain.AggregationCount},
		domain.Price{Label: "Overage", Scheme: domain.Fixed, UnitPrice: 10},
		apr1, may1)

	// Add the flat fee the trial must waive.
	now := time.Now().UTC().Truncate(time.Microsecond)
	flat := domain.Price{
		OrgId: orgId, Id: lib.GenerateId("price"), VariantId: seedVariantChain(t, db, orgId),
		Label:    "Platform fee",
		Category: domain.PriceCategorySubscription, Scheme: domain.Fixed,
		Currency: domain.USD, UnitPrice: 2900,
		BillingInterval: domain.BillingIntervalMonth, BillingIntervalQty: 1,
		TrialInterval: domain.BillingIntervalNone,
		CreatedAt:     now, UpdatedAt: now,
	}
	flatRow := priceRowFromDomain(flat)
	require.NoError(t, db.Create(&flatRow).Error)
	flatItem := seedOrderItem(t, db, orgId, fx.order.Id, flat.Id)
	require.NoError(t, db.Model(&orderItemRow{}).
		Where("org_id = ? AND id = ?", orgId, flatItem.Id).
		Update("subscription_id", fx.sub.Id).Error)

	fx.sub.Status = domain.SubscriptionStatusTrial
	require.NoError(t, db.Model(&subscriptionRow{}).
		Where("org_id = ? AND id = ?", orgId, fx.sub.Id).
		Update("status", string(domain.SubscriptionStatusTrial)).Error)

	usage := buildUsageService(t, db)
	for _, ext := range []string{"tr1", "tr2", "tr3"} {
		recordUsage(t, usage, port.RecordEventInput{
			OrgId: orgId, CustomerId: fx.customer.Id, MetricCode: fx.meter.Code,
			SubscriptionId: fx.sub.Id, ExternalId: ext, Timestamp: apr1.Add(time.Hour),
		})
	}

	inv, err := buildInvoiceService(t, db).BuildForBillingPeriod(ctx, fx.sub)
	require.NoError(t, err)
	require.Len(t, inv.LineItems, 1, "trial waives the flat fee — usage line only")
	assert.Equal(t, domain.InvoiceLineKindUsage, inv.LineItems[0].Kind)
	assert.Equal(t, int64(30), inv.Total, "3 events × 10c, no platform fee")
}

// Subscription creation clamps a metered line's cadence to monthly: an order
// for an ANNUAL metered price produces a MONTHLY-billing subscription (usage
// must never accumulate unbilled for a year).
func TestCreateSubscriptionsForOrder_MeteredCadenceClamp_E2E(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	now := time.Now().UTC().Truncate(time.Microsecond)
	cust := domain.Customer{
		OrgId: orgId, Id: lib.GenerateId("cus"), FirstName: "Cad", LastName: "Clamp",
		Email: lib.GenerateId("cad") + "@example.com", CreatedAt: now, UpdatedAt: now,
	}
	custRow := customerRowFromDomain(cust)
	require.NoError(t, db.Omit("DefaultPaymentMethodId").Create(&custRow).Error)

	meter := domain.BillableMetric{
		OrgId: orgId, Id: lib.GenerateId("met"), Code: "annual_usage", Name: "Usage",
		Aggregation: domain.AggregationCount, CreatedAt: now, UpdatedAt: now,
	}
	meterRow := billableMetricRowFromDomain(meter)
	require.NoError(t, db.Create(&meterRow).Error)

	price := domain.Price{
		OrgId: orgId, Id: lib.GenerateId("price"), VariantId: seedVariantChain(t, db, orgId),
		Label:    "Annual metered",
		Category: domain.PriceCategorySubscription, Scheme: domain.Fixed,
		Currency: domain.USD, UnitPrice: 10,
		BillingInterval: domain.BillingIntervalYear, BillingIntervalQty: 1,
		TrialInterval:    domain.BillingIntervalNone,
		BillableMetricId: meter.Id, CreatedAt: now, UpdatedAt: now,
	}
	priceRow := priceRowFromDomain(price)
	require.NoError(t, db.Create(&priceRow).Error)

	order := seedOrder(t, db, orgId, cust.Id)
	item := seedOrderItem(t, db, orgId, order.Id, price.Id)

	subs, err := buildSubscriptionService(t, db).CreateSubscriptionsForOrder(ctx, orgId, order.Id)
	require.NoError(t, err)
	require.Len(t, subs, 1)
	assert.Equal(t, domain.BillingIntervalMonth, subs[0].BillingInterval, "metered cadence clamps to monthly")
	assert.Equal(t, 1, subs[0].BillingIntervalQty)

	linked, err := NewOrderRepo(db).FindOrderItemsBySubscriptionId(ctx, orgId, subs[0].Id)
	require.NoError(t, err)
	require.Len(t, linked, 1)
	assert.Equal(t, item.Id, linked[0].Id)
}

// A period with zero usage still builds the invoice: one usage line with
// quantity 0 and total 0, and the charge succeeds at amount 0.
func TestZeroUsage_ZeroAmountInvoice_E2E(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	sep1 := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	oct1 := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	fx := seedUsageFixture(t, db, orgId,
		domain.BillableMetric{Code: "api_calls", Name: "API calls", Aggregation: domain.AggregationCount},
		domain.Price{Label: "API", Scheme: domain.Fixed, UnitPrice: 10},
		sep1, oct1)
	seedMemoryPspForSub(t, db, orgId, &fx.sub)

	result, err := buildSubscriptionService(t, db).ChargeForBillingPeriod(ctx, fx.sub)
	require.NoError(t, err)
	assert.Equal(t, domain.PaymentStatusSucceeded, result.Status)
	assert.Equal(t, int64(0), result.Amount)

	inv, err := NewInvoiceRepo(db).FindBySubscriptionCycle(ctx, orgId, fx.sub.Id, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(0), inv.Total)
	require.Len(t, inv.LineItems, 1, "a zero-usage period still gets its usage line")
	assert.True(t, inv.LineItems[0].Quantity.IsZero(), "got %s", inv.LineItems[0].Quantity)
	assert.Equal(t, int64(0), inv.LineItems[0].Total)
}

// On a FLOW meter, "operation" is ordinary metadata — reserved only on
// carry-over meters. A count meter records and counts such events; a sum meter
// still demands a numeric value under its field_name.
func TestFlowMeter_OperationMetadataIsOrdinary_E2E(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	meters := NewMeterRepo(db)
	_, err := meters.Create(ctx, domain.BillableMetric{
		OrgId: orgId, Id: "met_flow_count", Code: "clicks", Name: "Clicks",
		Aggregation: domain.AggregationCount,
	})
	require.NoError(t, err)
	_, err = meters.Create(ctx, domain.BillableMetric{
		OrgId: orgId, Id: "met_flow_sum", Code: "gb_sum", Name: "GB",
		Aggregation: domain.AggregationSum, FieldName: "gb",
	})
	require.NoError(t, err)

	usage := buildUsageService(t, db)
	jan1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("count meter records operation as plain metadata", func(t *testing.T) {
		res, err := usage.RecordEvent(ctx, port.RecordEventInput{
			OrgId: orgId, MetricCode: "clicks", ExternalCustomerId: "ext_flow_op",
			ExternalId: "op1", Timestamp: jan1.Add(time.Hour),
			Metadata: map[string]string{domain.UsageOperationKey: domain.UsageOperationAdd, "seat_id": "x"},
		})
		require.NoError(t, err)
		assert.Equal(t, port.IngestRecorded, res.Status)

		got, err := usage.AggregateForPeriod(ctx,
			domain.BillableMetric{OrgId: orgId, Code: "clicks", Aggregation: domain.AggregationCount},
			port.UsageQuery{OrgId: orgId, ExternalCustomerId: "ext_flow_op", From: jan1, To: jan1.Add(24 * time.Hour)})
		require.NoError(t, err)
		assert.True(t, got.Equal(decimal.NewFromInt(1)), "got %s", got)
	})

	t.Run("sum meter still requires its numeric field, operation or not", func(t *testing.T) {
		_, err := usage.RecordEvent(ctx, port.RecordEventInput{
			OrgId: orgId, MetricCode: "gb_sum", ExternalCustomerId: "ext_flow_op",
			ExternalId: "op2", Timestamp: jan1.Add(time.Hour),
			Metadata: map[string]string{domain.UsageOperationKey: domain.UsageOperationAdd, "gb": "not_a_number"},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not numeric")
	})
}
