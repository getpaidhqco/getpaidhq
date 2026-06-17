//go:build integration

// Invoice e2e for the billable surfaces no other test drives through a real
// charge: the volume pricing scheme, grouped usage (one invoice line per group
// value), and a hybrid plan (flat + metered price on one subscription). Same
// harness and style as stock_billing_invoice_e2e_test.go.
package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

// Volume scheme: call minutes summed over January, the WHOLE quantity priced at
// the band it reaches. Bands: 0–100 minutes at 5c, beyond at 3c. Three calls
// totalling 150 minutes land in the 3c band → 150 × 3c = $4.50.
func TestVolumeScheme_Invoice_E2E(t *testing.T) {
	db := testDB(t)
	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	jan1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	feb1 := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	fx := seedUsageFixture(t, db, orgId,
		domain.BillableMetric{Code: "call_minutes", Name: "Call minutes",
			Aggregation: domain.AggregationSum, FieldName: "minutes"},
		domain.Price{Label: "Minutes (volume)", Scheme: domain.Volume, Tiers: []domain.PriceTier{
			{FromValue: decimal.Zero, ToValue: decimal.NewFromInt(100), PerUnitAmount: decimal.NewFromInt(5)},
			{FromValue: decimal.NewFromInt(100), ToValue: decimal.Zero, PerUnitAmount: decimal.NewFromInt(3)},
		}},
		jan1, feb1)
	seedMemoryPspForSub(t, db, orgId, &fx.sub)

	usage := buildUsageService(t, db)
	for i, minutes := range []string{"50", "60", "40"} {
		recordUsage(t, usage, port.RecordEventInput{
			OrgId: orgId, CustomerId: fx.customer.Id, MetricCode: fx.meter.Code,
			SubscriptionId: fx.sub.Id, ExternalId: lib.GenerateId("vol"),
			Timestamp: jan1.Add(time.Duration(i+1) * 24 * time.Hour),
			Metadata:  map[string]string{"minutes": minutes},
		})
	}

	chargeAndAssertInvoice(t, db, orgId, fx, "150", 150*3)
}

// Grouped usage: a count meter with group_by ["region"] splits the one priced
// charge into one invoice line per discovered region, all at the same 7c rate.
// February: three eu requests, two us → two lines (21c + 14c), invoice 35c.
func TestGroupedUsage_Invoice_E2E(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	feb1 := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	mar1 := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	fx := seedUsageFixture(t, db, orgId,
		domain.BillableMetric{Code: "api_requests", Name: "API requests",
			Aggregation: domain.AggregationCount, GroupBy: []string{"region"}},
		domain.Price{Label: "Requests", Scheme: domain.Fixed, UnitPrice: 7},
		feb1, mar1)
	seedMemoryPspForSub(t, db, orgId, &fx.sub)

	usage := buildUsageService(t, db)
	for i, region := range []string{"eu", "eu", "eu", "us", "us"} {
		recordUsage(t, usage, port.RecordEventInput{
			OrgId: orgId, CustomerId: fx.customer.Id, MetricCode: fx.meter.Code,
			SubscriptionId: fx.sub.Id, ExternalId: lib.GenerateId("grp"),
			Timestamp: feb1.Add(time.Duration(i+1) * time.Hour),
			Metadata:  map[string]string{"region": region},
		})
	}

	svc := buildSubscriptionService(t, db)
	result, err := svc.ChargeForBillingPeriod(ctx, fx.sub)
	require.NoError(t, err)
	assert.Equal(t, int64(5*7), result.Amount, "charge = all regions' usage at one rate")

	inv, err := NewInvoiceRepo(db).FindBySubscriptionCycle(ctx, orgId, fx.sub.Id, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(35), inv.Total)
	require.Len(t, inv.LineItems, 2, "one usage line per discovered region")

	byRegion := map[string]domain.InvoiceLineItem{}
	for _, l := range inv.LineItems {
		assert.Equal(t, domain.InvoiceLineKindUsage, l.Kind)
		byRegion[l.Metadata["region"]] = l
	}
	require.Contains(t, byRegion, "eu")
	require.Contains(t, byRegion, "us")
	assert.True(t, byRegion["eu"].Quantity.Equal(decimal.NewFromInt(3)), "eu quantity, got %s", byRegion["eu"].Quantity)
	assert.Equal(t, int64(21), byRegion["eu"].Total)
	assert.Contains(t, byRegion["eu"].Description, "(region=eu)")
	assert.True(t, byRegion["us"].Quantity.Equal(decimal.NewFromInt(2)), "us quantity, got %s", byRegion["us"].Quantity)
	assert.Equal(t, int64(14), byRegion["us"].Total)
}

// Hybrid plan: ONE subscription carrying a flat $29 platform fee AND a 10c
// metered overage price. March: four metered events → the cycle's invoice has a
// base line ($29.00) and a usage line (40c), charged together as $29.40.
func TestHybridPlan_FlatPlusMetered_Invoice_E2E(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	mar1 := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	apr1 := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	fx := seedUsageFixture(t, db, orgId,
		domain.BillableMetric{Code: "overage_units", Name: "Overage units",
			Aggregation: domain.AggregationCount},
		domain.Price{Label: "Overage", Scheme: domain.Fixed, UnitPrice: 10},
		mar1, apr1)
	seedMemoryPspForSub(t, db, orgId, &fx.sub)

	// The flat platform fee: a second, non-metered price on the same order, its
	// item owned by the same subscription.
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

	usage := buildUsageService(t, db)
	for i := range 4 {
		recordUsage(t, usage, port.RecordEventInput{
			OrgId: orgId, CustomerId: fx.customer.Id, MetricCode: fx.meter.Code,
			SubscriptionId: fx.sub.Id, ExternalId: lib.GenerateId("hyb"),
			Timestamp: mar1.Add(time.Duration(i+1) * time.Hour),
		})
	}

	wantTotal := int64(2900 + 4*10)

	svc := buildSubscriptionService(t, db)
	result, err := svc.ChargeForBillingPeriod(ctx, fx.sub)
	require.NoError(t, err)
	assert.Equal(t, wantTotal, result.Amount, "one charge covers the flat fee plus usage")

	inv, err := NewInvoiceRepo(db).FindBySubscriptionCycle(ctx, orgId, fx.sub.Id, 0)
	require.NoError(t, err)
	assert.Equal(t, wantTotal, inv.Total)
	require.Len(t, inv.LineItems, 2, "base line + usage line")

	var base, usageLine *domain.InvoiceLineItem
	for i := range inv.LineItems {
		switch inv.LineItems[i].Kind {
		case domain.InvoiceLineKindBase:
			base = &inv.LineItems[i]
		case domain.InvoiceLineKindUsage:
			usageLine = &inv.LineItems[i]
		}
	}
	require.NotNil(t, base, "flat fee must produce a base line")
	require.NotNil(t, usageLine, "metered price must produce a usage line")
	assert.Equal(t, int64(2900), base.Total)
	assert.True(t, base.Quantity.Equal(decimal.NewFromInt(1)), "got %s", base.Quantity)
	assert.Equal(t, int64(40), usageLine.Total)
	assert.True(t, usageLine.Quantity.Equal(decimal.NewFromInt(4)), "got %s", usageLine.Quantity)
}
