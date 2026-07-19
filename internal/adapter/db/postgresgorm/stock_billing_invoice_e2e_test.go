//go:build integration

// Full billing e2e for every usage-based use case, one self-contained test per
// use case: its own meter, price, timeline, and events, driven through the real
// path (RecordEvent → EventStore → ChargeForBillingPeriod → BuildForBillingPeriod
// → invoice lines) against a Postgres testcontainer.
//
// Covered:
//   - carry-over meters, add/remove events: end-of-period, peak, distinct,
//     time-weighted (B), hybrid (C)
//   - carry-over meters, level reports: end-of-period, peak, time-average
//   - flow meter (sum) for the non-carry-over contrast
//
// Policy semantics and the expected numbers: docs/internal/billing-model/.
// Reuses the harness from metered_billing_e2e_test.go / billing_charge_e2e_test.go.
package postgresgorm

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

// seedUsageFixture persists the parent chain for a usage-based subscription —
// customer, the GIVEN meter and price (wired together), order, order item, and an
// active subscription due for [periodStart, periodEnd) — and returns the chain.
// It is seedMeteredFixture generalised to any meter/price configuration.
func seedUsageFixture(t *testing.T, db *gorm.DB, orgId string, meter domain.BillableMetric, price domain.Price, periodStart, periodEnd time.Time) meteredFixture {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond)

	cust := domain.Customer{
		OrgId:      orgId,
		Id:         lib.GenerateId("cus"),
		ExternalId: lib.GenerateId("ext_cus"),
		FirstName:  "Ada",
		LastName:   "Lovelace",
		Email:      lib.GenerateId("ada") + "@example.com",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	custRow := customerRowFromDomain(cust)
	require.NoError(t, db.Omit("DefaultPaymentMethodId").Create(&custRow).Error)

	meter.OrgId = orgId
	meter.Id = lib.GenerateId("met")
	meter.CreatedAt, meter.UpdatedAt = now, now
	meterRow := billableMetricRowFromDomain(meter)
	require.NoError(t, db.Create(&meterRow).Error)

	price.OrgId = orgId
	price.Id = lib.GenerateId("price")
	price.VariantId = seedVariantChain(t, db, orgId)
	price.Category = domain.PriceCategorySubscription
	price.Currency = domain.USD
	price.BillingInterval = domain.BillingIntervalMonth
	price.BillingIntervalQty = 1
	price.TrialInterval = domain.BillingIntervalNone
	price.BillableMetricId = meter.Id
	price.CreatedAt, price.UpdatedAt = now, now
	priceRow := priceRowFromDomain(price)
	require.NoError(t, db.Create(&priceRow).Error)

	order := seedOrder(t, db, orgId, cust.Id)
	item := seedOrderItem(t, db, orgId, order.Id, price.Id)

	sub := domain.Subscription{
		OrgId:              orgId,
		Id:                 lib.GenerateId("sub"),
		PspId:              domain.Paystack,
		OrderId:            order.Id,
		CustomerId:         cust.Id,
		Status:             domain.SubscriptionStatusActive,
		Currency:           "USD",
		BillingInterval:    domain.BillingIntervalMonth,
		BillingIntervalQty: 1,
		TrialInterval:      domain.BillingIntervalNone,
		Cycles:             12,
		CyclesProcessed:    0,
		StartDate:          periodStart,
		CurrentPeriodStart: periodStart,
		CurrentPeriodEnd:   periodEnd,
		RenewsAt:           periodEnd,
		Metadata:           map[string]string{},
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	subRow := subscriptionRowFromDomain(sub)
	// payment_method_id is nullable with FK; omit when empty to avoid FK violation.
	require.NoError(t, db.Omit("PaymentMethodId").Create(&subRow).Error)
	require.NoError(t, db.Model(&orderItemRow{}).
		Where("org_id = ? AND id = ?", orgId, item.Id).
		Update("subscription_id", sub.Id).Error)

	return meteredFixture{customer: cust, meter: meter, price: price, order: order, item: item, sub: sub}
}

// chargeAndAssertInvoice runs the real charge for the fixture's current period and
// asserts the charge amount, then the persisted invoice's single usage line:
// quantity and line total, and the invoice total.
func chargeAndAssertInvoice(t *testing.T, db *gorm.DB, orgId string, fx meteredFixture, wantQty string, wantTotalCents int64) {
	t.Helper()
	ctx := context.Background()

	svc := buildSubscriptionService(t, db)
	result, err := svc.ChargeForBillingPeriod(ctx, fx.sub)
	require.NoError(t, err)
	assert.Equal(t, wantTotalCents, result.Amount, "charge amount")

	inv, err := NewInvoiceRepo(db).FindBySubscriptionCycle(ctx, orgId, fx.sub.Id, 0)
	require.NoError(t, err, "an invoice must exist for the billed cycle")
	assert.Equal(t, wantTotalCents, inv.Total, "invoice total")
	require.Len(t, inv.LineItems, 1, "single metered item → one usage line")
	line := inv.LineItems[0]
	assert.Equal(t, domain.InvoiceLineKindUsage, line.Kind)
	assert.True(t, line.Quantity.Equal(decimal.RequireFromString(wantQty)),
		"usage line quantity: got %s want %s", line.Quantity, wantQty)
	assert.Equal(t, wantTotalCents, line.Total, "usage line total")
}

// addRemove records one add/remove event through the full RecordEvent path.
func addRemove(t *testing.T, db *gorm.DB, fx meteredFixture, orgId, extId, op, identity string, ts time.Time) {
	t.Helper()
	usage := buildUsageService(t, db)
	recordUsage(t, usage, port.RecordEventInput{
		OrgId: orgId, CustomerId: fx.customer.Id, MetricCode: fx.meter.Code,
		SubscriptionId: fx.sub.Id, ExternalId: extId, Timestamp: ts,
		Metadata: map[string]string{domain.UsageOperationKey: op, fx.meter.FieldName: identity},
	})
}

// levelReport records one level report through the full RecordEvent path.
func levelReport(t *testing.T, db *gorm.DB, fx meteredFixture, orgId, extId, value string, ts time.Time) {
	t.Helper()
	usage := buildUsageService(t, db)
	recordUsage(t, usage, port.RecordEventInput{
		OrgId: orgId, CustomerId: fx.customer.Id, MetricCode: fx.meter.Code,
		SubscriptionId: fx.sub.Id, ExternalId: extId, Timestamp: ts,
		Metadata: map[string]string{fx.meter.FieldName: value},
	})
}

// --- carry-over, add/remove events ------------------------------------------

// Use case A, end of period: a support desk pays $12/seat for the seats standing
// when March closes. carol and dan have held seats since February (found only via
// carry-over replay); erin joins in March; frank joins mid-March and leaves again
// (never billed); dan leaves before month end. Standing on March 31: carol, erin
// → 2 seats → $24.00.
func TestStockBilling_EndOfPeriod_AddRemove_E2E(t *testing.T) {
	db := testDB(t)
	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	mar1 := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	apr1 := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	fx := seedUsageFixture(t, db, orgId,
		domain.BillableMetric{Code: "support_seats", Name: "Support seats",
			Aggregation: domain.AggregationLatest, FieldName: "agent_id", CarryOver: true},
		domain.Price{Label: "Support seat", Scheme: domain.Fixed, UnitPrice: 1200},
		mar1, apr1)
	seedMemoryPspForSub(t, db, orgId, &fx.sub)

	addRemove(t, db, fx, orgId, "s1", domain.UsageOperationAdd, "carol", time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC))
	addRemove(t, db, fx, orgId, "s2", domain.UsageOperationAdd, "dan", time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC))
	addRemove(t, db, fx, orgId, "s3", domain.UsageOperationAdd, "erin", time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC))
	addRemove(t, db, fx, orgId, "s4", domain.UsageOperationAdd, "frank", time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC))
	addRemove(t, db, fx, orgId, "s5", domain.UsageOperationRemove, "frank", time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC))
	addRemove(t, db, fx, orgId, "s6", domain.UsageOperationRemove, "dan", time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC))

	chargeAndAssertInvoice(t, db, orgId, fx, "2", 2*1200)
}

// Use case A, peak concurrent: a call centre pays $20 per concurrent agent
// licence at the month's high-water mark. Three agents since January; two temps
// added for two days in April push the peak to 5; one agent leaves later. April
// bills the peak: 5 → $100.00.
func TestStockBilling_PeakConcurrent_AddRemove_E2E(t *testing.T) {
	db := testDB(t)
	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	apr1 := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	may1 := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	fx := seedUsageFixture(t, db, orgId,
		domain.BillableMetric{Code: "agent_licences", Name: "Agent licences",
			Aggregation: domain.AggregationMax, FieldName: "agent_id", CarryOver: true},
		domain.Price{Label: "Concurrent licence", Scheme: domain.Fixed, UnitPrice: 2000},
		apr1, may1)
	seedMemoryPspForSub(t, db, orgId, &fx.sub)

	jan5 := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	addRemove(t, db, fx, orgId, "p1", domain.UsageOperationAdd, "agent_1", jan5)
	addRemove(t, db, fx, orgId, "p2", domain.UsageOperationAdd, "agent_2", jan5)
	addRemove(t, db, fx, orgId, "p3", domain.UsageOperationAdd, "agent_3", jan5)
	addRemove(t, db, fx, orgId, "p4", domain.UsageOperationAdd, "temp_1", time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC))
	addRemove(t, db, fx, orgId, "p5", domain.UsageOperationAdd, "temp_2", time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC))
	addRemove(t, db, fx, orgId, "p6", domain.UsageOperationRemove, "temp_1", time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC))
	addRemove(t, db, fx, orgId, "p7", domain.UsageOperationRemove, "temp_2", time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC))
	addRemove(t, db, fx, orgId, "p8", domain.UsageOperationRemove, "agent_3", time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC))

	chargeAndAssertInvoice(t, db, orgId, fx, "5", 5*2000)
}

// Use case A, distinct active (MAU-style): a contractor portal bills $15 for
// every person who held access at ANY point in May, however briefly. greta has
// had access since March; hank gets four days; iris joins late; greta leaves
// before month end. Distinct in May: greta, hank, iris → 3 → $45.00.
func TestStockBilling_DistinctActive_AddRemove_E2E(t *testing.T) {
	db := testDB(t)
	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	may1 := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	jun1 := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	fx := seedUsageFixture(t, db, orgId,
		domain.BillableMetric{Code: "portal_access", Name: "Portal access",
			Aggregation: domain.AggregationUniqueCount, FieldName: "user_id", CarryOver: true},
		domain.Price{Label: "Portal user", Scheme: domain.Fixed, UnitPrice: 1500},
		may1, jun1)
	seedMemoryPspForSub(t, db, orgId, &fx.sub)

	addRemove(t, db, fx, orgId, "d1", domain.UsageOperationAdd, "greta", time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC))
	addRemove(t, db, fx, orgId, "d2", domain.UsageOperationAdd, "hank", time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC))
	addRemove(t, db, fx, orgId, "d3", domain.UsageOperationRemove, "hank", time.Date(2026, 5, 9, 0, 0, 0, 0, time.UTC))
	addRemove(t, db, fx, orgId, "d4", domain.UsageOperationAdd, "iris", time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC))
	addRemove(t, db, fx, orgId, "d5", domain.UsageOperationRemove, "greta", time.Date(2026, 5, 28, 0, 0, 0, 0, time.UTC))

	chargeAndAssertInvoice(t, db, orgId, fx, "3", 3*1500)
}

// Use case B, time-weighted (prorate up, credit down): the canonical June
// timeline at $10/seat. alice, bob, carol seated since May 20; dave joins June 16
// (accrues 15/30); bob leaves June 21 (accrues 20/30, remainder credited).
// Quantity 1 + 1 + 0.6667 + 0.5 → rounded 3.17 → $31.70.
func TestStockBilling_TimeWeighted_AddRemove_E2E(t *testing.T) {
	db := testDB(t)
	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	jun1 := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	jul1 := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	fx := seedUsageFixture(t, db, orgId,
		domain.BillableMetric{Code: "seats", Name: "Seats",
			Aggregation: domain.AggregationWeightedSum, FieldName: "seat_id", CarryOver: true,
			RoundingMode: "round", RoundingScale: 2},
		domain.Price{Label: "Seat (fair billing)", Scheme: domain.Fixed, UnitPrice: 1000,
			ProrateOnIncrease: true, CreditOnDecrease: true},
		jun1, jul1)
	seedMemoryPspForSub(t, db, orgId, &fx.sub)

	may20 := time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC)
	addRemove(t, db, fx, orgId, "b1", domain.UsageOperationAdd, "alice", may20)
	addRemove(t, db, fx, orgId, "b2", domain.UsageOperationAdd, "bob", may20)
	addRemove(t, db, fx, orgId, "b3", domain.UsageOperationAdd, "carol", may20)
	addRemove(t, db, fx, orgId, "b4", domain.UsageOperationAdd, "dave", time.Date(2026, 6, 16, 0, 0, 0, 0, time.UTC))
	addRemove(t, db, fx, orgId, "b5", domain.UsageOperationRemove, "bob", time.Date(2026, 6, 21, 0, 0, 0, 0, time.UTC))

	chargeAndAssertInvoice(t, db, orgId, fx, "3.17", 3170)
}

// Use case C, hybrid (prorate up, commit down): a design tool at $8/seat,
// September (30 days). jo and kim seated since August; lee joins September 16
// (prorated: 15/30 = 0.5); kim leaves September 11 but is committed to month end
// (no credit: 1.0). Quantity 1 + 1 + 0.5 = 2.5 → $20.00.
func TestStockBilling_Hybrid_AddRemove_E2E(t *testing.T) {
	db := testDB(t)
	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	sep1 := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	oct1 := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	fx := seedUsageFixture(t, db, orgId,
		domain.BillableMetric{Code: "design_seats", Name: "Design seats",
			Aggregation: domain.AggregationWeightedSum, FieldName: "member_id", CarryOver: true,
			RoundingMode: "round", RoundingScale: 2},
		domain.Price{Label: "Design seat", Scheme: domain.Fixed, UnitPrice: 800,
			ProrateOnIncrease: true, CreditOnDecrease: false},
		sep1, oct1)
	seedMemoryPspForSub(t, db, orgId, &fx.sub)

	aug1 := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	addRemove(t, db, fx, orgId, "c1", domain.UsageOperationAdd, "jo", aug1)
	addRemove(t, db, fx, orgId, "c2", domain.UsageOperationAdd, "kim", aug1)
	addRemove(t, db, fx, orgId, "c3", domain.UsageOperationRemove, "kim", time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC))
	addRemove(t, db, fx, orgId, "c4", domain.UsageOperationAdd, "lee", time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC))

	chargeAndAssertInvoice(t, db, orgId, fx, "2.5", 2000)
}

// --- carry-over, level reports ------------------------------------------------

// End of period from level reports: licences reported as totals by a provisioning
// job, $5 each, billed on the count standing when July closes. 8 reported back in
// May (carries over), 12 on July 9, 11 on July 28 → standing at end: 11 → $55.00.
func TestStockBilling_EndOfPeriod_LevelReports_E2E(t *testing.T) {
	db := testDB(t)
	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	jul1 := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	aug1 := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	fx := seedUsageFixture(t, db, orgId,
		domain.BillableMetric{Code: "licences", Name: "Licences",
			Aggregation: domain.AggregationLatest, FieldName: "count", CarryOver: true},
		domain.Price{Label: "Licence", Scheme: domain.Fixed, UnitPrice: 500},
		jul1, aug1)
	seedMemoryPspForSub(t, db, orgId, &fx.sub)

	levelReport(t, db, fx, orgId, "l1", "8", time.Date(2026, 5, 14, 0, 0, 0, 0, time.UTC))
	levelReport(t, db, fx, orgId, "l2", "12", time.Date(2026, 7, 9, 0, 0, 0, 0, time.UTC))
	levelReport(t, db, fx, orgId, "l3", "11", time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC))

	chargeAndAssertInvoice(t, db, orgId, fx, "11", 11*500)
}

// Peak from level reports: monitoring agents reported as totals, $3 per agent at
// the August high-water mark. 4 agents standing since February (the level in
// force entering August), a spike to 9 on August 12, back to 6 on August 23.
// Peak: 9 → $27.00.
func TestStockBilling_Peak_LevelReports_E2E(t *testing.T) {
	db := testDB(t)
	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	aug1 := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	sep1 := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	fx := seedUsageFixture(t, db, orgId,
		domain.BillableMetric{Code: "monitor_agents", Name: "Monitoring agents",
			Aggregation: domain.AggregationMax, FieldName: "count", CarryOver: true},
		domain.Price{Label: "Agent", Scheme: domain.Fixed, UnitPrice: 300},
		aug1, sep1)
	seedMemoryPspForSub(t, db, orgId, &fx.sub)

	levelReport(t, db, fx, orgId, "m1", "4", time.Date(2026, 2, 3, 0, 0, 0, 0, time.UTC))
	levelReport(t, db, fx, orgId, "m2", "9", time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC))
	levelReport(t, db, fx, orgId, "m3", "6", time.Date(2026, 8, 23, 0, 0, 0, 0, time.UTC))

	chargeAndAssertInvoice(t, db, orgId, fx, "9", 9*300)
}

// Time-average from level reports: provisioned db at $0.10 per GB-month,
// June (30 days). 300 GB provisioned back in April (in force entering June),
// 600 GB from June 11, 150 GB from June 21: 10 days at each level →
// average (300 + 600 + 150) / 3 = 350 GB → $35.00.
func TestStockBilling_TimeAverage_LevelReports_E2E(t *testing.T) {
	db := testDB(t)
	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	jun1 := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	jul1 := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	fx := seedUsageFixture(t, db, orgId,
		domain.BillableMetric{Code: "db", Name: "Provisioned db",
			Aggregation: domain.AggregationWeightedSum, FieldName: "gb", CarryOver: true,
			RoundingMode: "round", RoundingScale: 2},
		domain.Price{Label: "Storage GB-month", Scheme: domain.Fixed, UnitPrice: 10},
		jun1, jul1)
	seedMemoryPspForSub(t, db, orgId, &fx.sub)

	levelReport(t, db, fx, orgId, "g1", "300", time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC))
	levelReport(t, db, fx, orgId, "g2", "600", time.Date(2026, 6, 11, 0, 0, 0, 0, time.UTC))
	levelReport(t, db, fx, orgId, "g3", "150", time.Date(2026, 6, 21, 0, 0, 0, 0, time.UTC))

	chargeAndAssertInvoice(t, db, orgId, fx, "350", 350*10)
}

// --- flow meter (the non-carry-over contrast) ----------------------------------

// Flow sum: outbound data transfer at $0.25/GB, summed over October's events
// only — a flow meter resets each period, so the September transfer is not
// billed. October events: 1.5 + 2.5 + 6 = 10 GB → $2.50.
func TestStockBilling_FlowSum_E2E(t *testing.T) {
	db := testDB(t)
	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	oct1 := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	nov1 := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	fx := seedUsageFixture(t, db, orgId,
		domain.BillableMetric{Code: "data_transfer", Name: "Data transfer",
			Aggregation: domain.AggregationSum, FieldName: "gb"},
		domain.Price{Label: "Transfer GB", Scheme: domain.Fixed, UnitPrice: 25},
		oct1, nov1)
	seedMemoryPspForSub(t, db, orgId, &fx.sub)

	transfer := func(extId, gb string, ts time.Time) {
		usage := buildUsageService(t, db)
		recordUsage(t, usage, port.RecordEventInput{
			OrgId: orgId, CustomerId: fx.customer.Id, MetricCode: fx.meter.Code,
			SubscriptionId: fx.sub.Id, ExternalId: extId, Timestamp: ts,
			Metadata: map[string]string{"gb": gb},
		})
	}
	transfer("f0", "99", time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)) // last period — not billed
	transfer("f1", "1.5", time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC))
	transfer("f2", "2.5", time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC))
	transfer("f3", "6", time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC))

	chargeAndAssertInvoice(t, db, orgId, fx, "10", 250)
}
