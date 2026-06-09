//go:build integration

// End-to-end coverage of metered (usage-based) billing against real Postgres.
//
// The unit-level invoice tests (internal/core/service/invoice_test.go) build
// metered invoices against a MOCKED EventStore, so the aggregation SQL and the
// invoice math are never exercised together. The simple-billing e2e
// (simple_billing_e2e_test.go) runs the full charge tail but on a FIXED price.
// These tests close that gap: a metered subscription's usage is persisted in the
// real EventStore, then the SAME charge path the billing-cycle runner uses
// (SubscriptionService.ChargeForBillingPeriod → BuildForBillingPeriod →
// UsageForSubscription → real Sum/Count SQL) drives the charge amount and the
// invoice line items.
//
// Reuses the harness from billing_charge_e2e_test.go (buildSubscriptionService,
// seedMemoryPsp, seedPaymentMethod, noopLogger, noopPubSub) and testDB(t) — never
// the developer's local stack.
package postgres

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
	"getpaidhq/internal/core/service"
	"getpaidhq/internal/lib"
)

// meteredFixture is the parent chain for a usage-based subscription: a customer
// (with an ExternalId so external_customer_id matching can be tested), a count
// meter, a metered Fixed price (10c per counted event), and an order/order-item/
// subscription wired to them. The order carries a single metered item, so the
// subscription is itself the "primary" that bills the order's usage.
type meteredFixture struct {
	customer domain.Customer
	meter    domain.BillableMetric
	price    domain.Price
	order    domain.Order
	item     domain.OrderItem
	sub      domain.Subscription
}

// unitPriceCents is the metered rate: every counted event costs this many cents.
const meteredUnitPriceCents = 10

// seedMeteredFixture persists a metered subscription due for the given period.
// The subscription is active, cycle 0, with explicit period boundaries so usage
// window scoping can be asserted exactly.
func seedMeteredFixture(t *testing.T, db *gorm.DB, orgId string, periodStart, periodEnd time.Time) meteredFixture {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond)

	cust := domain.Customer{
		OrgId:      orgId,
		Id:         lib.GenerateId("cus"),
		ExternalId: lib.GenerateId("ext_cus"), // the merchant's own id, matched on usage events
		FirstName:  "Grace",
		LastName:   "Hopper",
		Email:      lib.GenerateId("grace") + "@example.com",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	custRow := customerRowFromDomain(cust)
	require.NoError(t, db.Create(&custRow).Error)

	meter := domain.BillableMetric{
		OrgId:       orgId,
		Id:          lib.GenerateId("met"),
		Code:        "api_calls",
		Name:        "API Calls",
		Aggregation: domain.AggregationCount,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	meterRow := billableMetricRowFromDomain(meter)
	require.NoError(t, db.Create(&meterRow).Error)

	price := domain.Price{
		OrgId:              orgId,
		Id:                 lib.GenerateId("price"),
		Label:              "Metered API",
		Category:           domain.PriceCategorySubscription,
		Scheme:             domain.Fixed,
		Currency:           domain.USD,
		UnitPrice:          meteredUnitPriceCents,
		BillingInterval:    domain.BillingIntervalMonth,
		BillingIntervalQty: 1,
		BillableMetricId:   meter.Id, // <- makes the price metered
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	priceRow := priceRowFromDomain(price)
	require.NoError(t, db.Create(&priceRow).Error)

	order := seedOrder(t, db, orgId, cust.Id)
	item := seedOrderItem(t, db, orgId, order.Id, price.Id)

	sub := domain.Subscription{
		OrgId:              orgId,
		Id:                 lib.GenerateId("sub"),
		OrderId:            order.Id,
		CustomerId:         cust.Id,
		Status:             domain.SubscriptionStatusActive,
		Currency:           "USD",
		BillingInterval:    domain.BillingIntervalMonth,
		BillingIntervalQty: 1,
		Cycles:             12,
		CyclesProcessed:    0,
		StartDate:          periodStart,
		CurrentPeriodStart: periodStart,
		CurrentPeriodEnd:   periodEnd,
		RenewsAt:           periodEnd,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	subRow := subscriptionRowFromDomain(sub)
	require.NoError(t, db.Create(&subRow).Error)
	// The subscription owns its metered line.
	require.NoError(t, db.Model(&orderItemRow{}).
		Where("org_id = ? AND id = ?", orgId, item.Id).
		Update("subscription_id", sub.Id).Error)

	return meteredFixture{customer: cust, meter: meter, price: price, order: order, item: item, sub: sub}
}

// buildUsageService wires a UsageService off the testcontainer db, with the
// EventStore as both the durable ingestor and the read/aggregation backend (the
// USAGE_DATABASE_URL-unset, sync-ingest production default).
func buildUsageService(t *testing.T, db *gorm.DB) *service.UsageService {
	t.Helper()
	store := NewEventStore(db)
	return service.NewUsageService(
		NewMeterRepo(db),
		NewCustomerRepo(db),
		NewSubscriptionRepo(db),
		NewOrderRepo(db),
		NewPriceRepo(db),
		store, // ingestor
		store, // event store
		noopPubSub{},
		noopLogger{},
	)
}

// recordUsage drives a usage event through the full RecordEvent validation +
// ingest path (the same path the HTTP handler uses) and asserts the outcome
// status, returning it for further assertions.
func recordUsage(t *testing.T, usage *service.UsageService, in port.RecordEventInput) port.IngestResult {
	t.Helper()
	res, err := usage.RecordEvent(context.Background(), in)
	require.NoError(t, err)
	return res
}

// seedMemoryPspForSub wires the subscription to a memory gateway + active payment
// method, persisting the psp_id / payment_method_id columns so the re-read inside
// ChargeForBillingPeriod resolves them, and updating the in-memory sub the caller holds.
func seedMemoryPspForSub(t *testing.T, db *gorm.DB, orgId string, sub *domain.Subscription) {
	t.Helper()
	pspConfigId := seedMemoryPsp(t, db, orgId)
	pm := seedPaymentMethod(t, db, orgId, sub.CustomerId)
	sub.PspId = domain.Gateway(pspConfigId)
	sub.PaymentMethodId = pm.Id
	require.NoError(t, db.Model(&subscriptionRow{}).
		Where("org_id = ? AND id = ?", orgId, sub.Id).
		Updates(map[string]any{"psp_id": pspConfigId, "payment_method_id": pm.Id}).Error)
}

// TestMeteredSubscriptionBilling_E2E pins the metered billing happy path: three
// usage events are recorded for the period, then the subscription is charged. The
// charge amount, the invoice usage line, and the cycle advance must all reflect
// the persisted usage (3 events × 10c = 30c) — proving the real aggregation SQL,
// the invoice build, and the charge tail agree.
func TestMeteredSubscriptionBilling_E2E(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	periodStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	nextEnd := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	fx := seedMeteredFixture(t, db, orgId, periodStart, periodEnd)
	seedMemoryPspForSub(t, db, orgId, &fx.sub)

	usage := buildUsageService(t, db)
	// Three distinct counted events inside the period.
	for i, ext := range []string{"u1", "u2", "u3"} {
		recordUsage(t, usage, port.RecordEventInput{
			OrgId:          orgId,
			CustomerId:     fx.customer.Id,
			MetricCode:     fx.meter.Code,
			SubscriptionId: fx.sub.Id,
			ExternalId:     ext,
			Timestamp:      periodStart.Add(time.Duration(i+1) * time.Hour),
		})
	}

	wantTotal := int64(3 * meteredUnitPriceCents) // 30c

	svc := buildSubscriptionService(t, db)

	result, err := svc.ChargeForBillingPeriod(ctx, fx.sub)
	require.NoError(t, err)
	require.Equal(t, domain.PaymentStatusSucceeded, result.Status)
	assert.Equal(t, wantTotal, result.Amount, "charge amount = aggregated usage × rate")

	updated, err := svc.HandleSubscriptionChargeSuccess(ctx, port.SubscriptionChargeInput{Subscription: fx.sub, ChargeResult: result})
	require.NoError(t, err)

	// Invoice: one usage line for 3 units @ 10c, marked paid.
	inv, err := NewInvoiceRepo(db).FindBySubscriptionCycle(ctx, orgId, fx.sub.Id, 0)
	require.NoError(t, err, "an invoice must exist for the billed cycle")
	assert.Equal(t, domain.InvoiceStatusPaid, inv.Status)
	assert.Equal(t, wantTotal, inv.Total)
	require.Len(t, inv.LineItems, 1, "single metered item → one usage line")
	line := inv.LineItems[0]
	assert.Equal(t, domain.InvoiceLineKindUsage, line.Kind)
	assert.True(t, line.Quantity.Equal(decimal.NewFromInt(3)), "usage line quantity = 3 counted events, got %s", line.Quantity)
	assert.Equal(t, wantTotal, line.Total)

	// Payment settles the invoice.
	payments, total, err := NewPaymentRepo(db).FindBySubscriptionId(ctx, orgId, fx.sub.Id, domain.Pagination{Page: 1, Limit: 10})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	assert.Equal(t, domain.PaymentStatusSucceeded, payments[0].Status)
	assert.Equal(t, inv.Id, payments[0].InvoiceId)
	assert.Equal(t, wantTotal, payments[0].NetAmount)

	// Subscription advanced exactly one cycle, next period scheduled.
	assert.Equal(t, 1, updated.CyclesProcessed)
	persisted, err := NewSubscriptionRepo(db).FindById(ctx, orgId, fx.sub.Id)
	require.NoError(t, err)
	assert.Equal(t, 1, persisted.CyclesProcessed, "cycle advance is durable")
	assert.True(t, persisted.CurrentPeriodEnd.Equal(nextEnd), "next period ends one interval later")
}

// TestMeteredBilling_DuplicateEventNotDoubleCharged proves a resend with the same
// external_id is deduped at write time and therefore billed once: two events share
// external_id "dup" (second reported Duplicate) plus one distinct event → 2 billable
// units → 20c, NOT 30c.
func TestMeteredBilling_DuplicateEventNotDoubleCharged(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	periodStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	fx := seedMeteredFixture(t, db, orgId, periodStart, periodEnd)
	seedMemoryPspForSub(t, db, orgId, &fx.sub)

	usage := buildUsageService(t, db)
	base := port.RecordEventInput{OrgId: orgId, CustomerId: fx.customer.Id, MetricCode: fx.meter.Code, SubscriptionId: fx.sub.Id}

	first := base
	first.ExternalId = "dup"
	first.Timestamp = periodStart.Add(time.Hour)
	res := recordUsage(t, usage, first)
	assert.Equal(t, port.IngestRecorded, res.Status)

	resend := base
	resend.ExternalId = "dup" // same id, later timestamp — must be ignored at write
	resend.Timestamp = periodStart.Add(2 * time.Hour)
	res = recordUsage(t, usage, resend)
	assert.Equal(t, port.IngestDuplicate, res.Status, "resend with a seen external_id must dedup")

	distinct := base
	distinct.ExternalId = "other"
	distinct.Timestamp = periodStart.Add(3 * time.Hour)
	recordUsage(t, usage, distinct)

	wantTotal := int64(2 * meteredUnitPriceCents) // 2 billable units, not 3

	svc := buildSubscriptionService(t, db)
	result, err := svc.ChargeForBillingPeriod(ctx, fx.sub)
	require.NoError(t, err)
	assert.Equal(t, wantTotal, result.Amount, "duplicate event must not double-charge")

	inv, err := NewInvoiceRepo(db).FindBySubscriptionCycle(ctx, orgId, fx.sub.Id, 0)
	require.NoError(t, err)
	require.Len(t, inv.LineItems, 1)
	assert.True(t, inv.LineItems[0].Quantity.Equal(decimal.NewFromInt(2)), "billable units = 2 (deduped), got %s", inv.LineItems[0].Quantity)
	assert.Equal(t, wantTotal, inv.Total)
}

// TestMeteredBilling_PeriodScoping proves the charge for cycle N bills only usage
// inside [CurrentPeriodStart, CurrentPeriodEnd): an event before the period, an
// event on the [periodEnd] boundary (half-open → excluded), and an event after the
// period are all ignored; only the two in-window events are billed.
func TestMeteredBilling_PeriodScoping(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	periodStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	fx := seedMeteredFixture(t, db, orgId, periodStart, periodEnd)
	seedMemoryPspForSub(t, db, orgId, &fx.sub)

	usage := buildUsageService(t, db)
	base := port.RecordEventInput{OrgId: orgId, CustomerId: fx.customer.Id, MetricCode: fx.meter.Code, SubscriptionId: fx.sub.Id}

	events := []struct {
		ext string
		ts  time.Time
	}{
		{"before", periodStart.Add(-time.Hour)},       // previous period — excluded
		{"in1", periodStart},                          // on [from] boundary — included
		{"in2", periodStart.Add(10 * 24 * time.Hour)}, // mid-period — included
		{"on_end", periodEnd},                         // on [to] boundary — excluded (half-open)
		{"after", periodEnd.Add(time.Hour)},           // next period — excluded
	}
	for _, e := range events {
		in := base
		in.ExternalId = e.ext
		in.Timestamp = e.ts
		recordUsage(t, usage, in)
	}

	wantTotal := int64(2 * meteredUnitPriceCents) // only in1 + in2

	svc := buildSubscriptionService(t, db)
	result, err := svc.ChargeForBillingPeriod(ctx, fx.sub)
	require.NoError(t, err)
	assert.Equal(t, wantTotal, result.Amount, "only in-window usage is billed")

	inv, err := NewInvoiceRepo(db).FindBySubscriptionCycle(ctx, orgId, fx.sub.Id, 0)
	require.NoError(t, err)
	require.Len(t, inv.LineItems, 1)
	assert.True(t, inv.LineItems[0].Quantity.Equal(decimal.NewFromInt(2)), "only 2 in-window events, got %s", inv.LineItems[0].Quantity)
}

// TestMeteredBilling_ExternalCustomerIdAttribution proves usage recorded against
// the merchant's own customer id (external_customer_id) — with NO internal
// customer_id, the "orphan" case for events sent before the customer existed — is
// still billed, because UsageForSubscription matches on (customer_id OR
// external_customer_id). A foreign external id must NOT leak in.
func TestMeteredBilling_ExternalCustomerIdAttribution(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	periodStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	fx := seedMeteredFixture(t, db, orgId, periodStart, periodEnd)
	seedMemoryPspForSub(t, db, orgId, &fx.sub)

	// Ingest directly so we control the stored columns: RecordEvent would resolve a
	// known external id to the internal customer_id. The orphan path is external-only.
	store := NewEventStore(db)
	mk := func(id, extId, extCustomer string) domain.MeterEvent {
		return domain.MeterEvent{
			OrgId:              orgId,
			Id:                 id,
			ExternalCustomerId: extCustomer,
			MetricCode:         fx.meter.Code,
			ExternalId:         extId,
			Timestamp:          periodStart.Add(time.Hour),
			CreatedAt:          periodStart.Add(time.Hour),
		}
	}
	// One attributed to our internal id, one orphan against our external id, and one
	// against a foreign external id that must never be billed.
	_, err := store.Ingest(ctx, domain.MeterEvent{OrgId: orgId, Id: "internal", CustomerId: fx.customer.Id, MetricCode: fx.meter.Code, ExternalId: "i1", Timestamp: periodStart.Add(time.Hour), CreatedAt: periodStart.Add(time.Hour)})
	require.NoError(t, err)
	_, err = store.Ingest(ctx, mk("orphan", "o1", fx.customer.ExternalId))
	require.NoError(t, err)
	_, err = store.Ingest(ctx, mk("foreign", "f1", "someone_elses_external_id"))
	require.NoError(t, err)

	wantTotal := int64(2 * meteredUnitPriceCents) // internal + orphan; not foreign

	svc := buildSubscriptionService(t, db)
	result, err := svc.ChargeForBillingPeriod(ctx, fx.sub)
	require.NoError(t, err)
	assert.Equal(t, wantTotal, result.Amount, "external_customer_id usage is billed; foreign id is not")

	inv, err := NewInvoiceRepo(db).FindBySubscriptionCycle(ctx, orgId, fx.sub.Id, 0)
	require.NoError(t, err)
	require.Len(t, inv.LineItems, 1)
	assert.True(t, inv.LineItems[0].Quantity.Equal(decimal.NewFromInt(2)), "internal + orphan external match, got %s", inv.LineItems[0].Quantity)
}
