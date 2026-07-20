//go:build integration

// End-to-end coverage of GRADUATED (tiered) usage billing against real Postgres,
// for the transactional-email-API use case documented in
// docs/internal/graduated-use-case.md.
//
// The unit test (internal/core/domain/pricing_test.go, if present) exercises
// priceGraduated in isolation; metered_billing_e2e_test.go runs the full charge
// tail but on a FIXED-scheme price. This test closes the remaining gap: a
// graduated price whose Tiers round-trip through the real prices table, usage
// persisted in the real EventStore, aggregated by real SUM SQL, then priced by
// the SAME charge path the billing-cycle runner uses
// (SubscriptionService.ChargeForBillingPeriod → BuildForBillingPeriod →
// MeteredUsageForSubscription → AggregateForPeriod(Sum) → UsageLineFromPrice →
// PriceUsage/priceGraduated). The recorded usage is sized so the period total
// spans ALL THREE tiers.
//
// Reuses the harness from metered_billing_e2e_test.go / billing_charge_e2e_test.go
// (buildUsageService, buildSubscriptionService, seedMemoryPspForSub, recordUsage,
// seedOrder, seedOrderItem, noopLogger, noopPubSub) and poolForTest(t) — never the
// developer's local stack.
package postgrespgx

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// TestGraduatedEmailBilling_AllTiers_E2E pins the graduated happy path across all
// three tiers. Three batch-send events totalling 250,000 emails are recorded for
// the period, then the subscription is charged. The period total lands in tier 3,
// so every tier contributes:
//
//	tier 1:   10,000 × $0.0010 = $10.00   (the first 10k)
//	tier 2:   90,000 × $0.0005 = $45.00   (10,001 → 100,000)
//	tier 3:  150,000 × $0.0002 = $30.00   (100,001 → 250,000)
//	                             ───────
//	                  total      $85.00   = 8,500¢
//
// The charge amount, the invoice usage line (quantity, blended unit amount, total),
// and the cycle advance must all reflect the graduated computation over the real
// aggregated usage — proving Tiers persistence, SUM SQL, the graduated math, and
// the charge tail agree.
func TestGraduatedEmailBilling_AllTiers_E2E(t *testing.T) {
	pool := poolForTest(t)
	ctx := context.Background()
	orgId := uniqueOrg(t, pool)
	cleanupOrg(t, pool, orgId)

	periodStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	nextEnd := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	fx := seedGraduatedEmailFixture(t, pool, orgId, periodStart, periodEnd)
	seedMemoryPspForSub(t, pool, orgId, &fx.sub)

	usage := buildUsageService(t, pool)
	// Several batch-send reports that sum to 250,000 emails. Individual batch sizes
	// are irrelevant to graduated pricing (only the period SUM matters); using a few
	// distinct events proves the SUM aggregation rather than a single pre-summed row.
	recordEmails(t, usage, orgId, fx, "batch-1", 100_000, periodStart.Add(1*time.Hour))
	recordEmails(t, usage, orgId, fx, "batch-2", 100_000, periodStart.Add(2*time.Hour))
	recordEmails(t, usage, orgId, fx, "batch-3", 50_000, periodStart.Add(3*time.Hour))

	const wantUnits = 250_000
	const wantTotal = int64(8_500) // $85.00 — see the per-tier breakdown above

	// Sanity-check the domain math directly: the same Price + units the charge path
	// will use must price to $85.00, with each tier contributing as documented.
	gotAmt, gotUnit := domain.PriceUsage(fx.price, decimal.NewFromInt(wantUnits))
	require.Equal(t, wantTotal, gotAmt, "graduated price of 250k emails must be 8,500c")
	require.True(t, gotUnit.Equal(decimal.RequireFromString("0.034")),
		"blended unit rate = 8,500c / 250,000 = 0.034c, got %s", gotUnit)

	svc := buildSubscriptionService(t, pool)

	result, err := svc.ChargeForBillingPeriod(ctx, fx.sub)
	require.NoError(t, err)
	require.Equal(t, domain.PaymentStatusSucceeded, result.Status)
	assert.Equal(t, wantTotal, result.Amount, "charge amount = graduated price over all three tiers")

	updated, err := svc.HandleSubscriptionChargeSuccess(ctx, port.SubscriptionChargeInput{Subscription: fx.sub, ChargeResult: result})
	require.NoError(t, err)

	// Invoice: one usage line for 250,000 units priced by the graduated ladder, paid.
	inv, err := NewInvoiceRepo(pool).FindBySubscriptionCycle(ctx, orgId, fx.sub.Id, 0)
	require.NoError(t, err, "an invoice must exist for the billed cycle")
	assert.Equal(t, domain.InvoiceStatusPaid, inv.Status)
	assert.Equal(t, wantTotal, inv.Total)
	require.Len(t, inv.LineItems, 1, "single metered item → one usage line")
	line := inv.LineItems[0]
	assert.Equal(t, domain.InvoiceLineKindUsage, line.Kind)
	assert.True(t, line.Quantity.Equal(decimal.NewFromInt(wantUnits)),
		"usage line quantity = aggregated emails, want %d got %s", wantUnits, line.Quantity)
	assert.True(t, line.UnitAmount.Equal(decimal.RequireFromString("0.034")),
		"usage line unit amount = blended rate 0.034c, got %s", line.UnitAmount)
	assert.Equal(t, wantTotal, line.Total)

	// Payment settles the invoice.
	payments, total, err := NewPaymentRepo(pool).FindBySubscriptionId(ctx, orgId, fx.sub.Id, domain.Pagination{Page: 1, Limit: 10})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	assert.Equal(t, domain.PaymentStatusSucceeded, payments[0].Status)
	assert.Equal(t, inv.Id, payments[0].InvoiceId)
	assert.Equal(t, wantTotal, payments[0].NetAmount)

	// Subscription advanced exactly one cycle, next period scheduled.
	assert.Equal(t, 1, updated.CyclesProcessed)
	persisted, err := NewSubscriptionRepo(pool).FindById(ctx, orgId, fx.sub.Id)
	require.NoError(t, err)
	assert.Equal(t, 1, persisted.CyclesProcessed, "cycle advance is durable")
	assert.True(t, persisted.CurrentPeriodEnd.Equal(nextEnd), "next period ends one interval later")
}

// TestGraduatedEmailBilling_TierBoundaries_E2E proves the ladder is correct at and
// around the tier boundaries — that slices abut (no gap, no double-count) and the
// open-ended top tier extends — by charging several independent subscriptions at
// chosen usage levels and asserting the exact cents:
//
//	10,000  → only tier 1 full:            10,000×0.1c                       =  1,000c
//	10,001  → tier 1 + 1 unit of tier 2:   1,000 + 1×0.05c                   =  1,000c (rounds)
//	100,000 → tiers 1+2 full:              1,000 + 90,000×0.05c              =  5,500c
//	250,000 → tiers 1+2 full + tier 3:     5,500 + 150,000×0.02c             =  8,500c
//	1,000,000 → deep into tier 3:          5,500 + 900,000×0.02c             = 23,500c
func TestGraduatedEmailBilling_TierBoundaries_E2E(t *testing.T) {
	pool := poolForTest(t)
	ctx := context.Background()

	periodStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name      string
		emails    int
		wantCents int64
	}{
		{"tier1 full edge", 10_000, 1_000},
		{"one unit into tier2", 10_001, 1_000}, // +0.05c rounds down to the same cent
		{"tier1+2 full edge", 100_000, 5_500},  // tier-2 upper boundary
		{"into tier3 (use case)", 250_000, 8_500},
		{"deep in tier3", 1_000_000, 23_500}, // open-ended top tier
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Each case gets its own org: the fixture reuses a fixed meter Code, so
			// sharing an org would let RecordEvent's FindByCode resolve a sibling
			// case's meter. Fresh-org-per-test is the package's isolation model.
			orgId := uniqueOrg(t, pool)
			cleanupOrg(t, pool, orgId)
			fx := seedGraduatedEmailFixture(t, pool, orgId, periodStart, periodEnd)
			seedMemoryPspForSub(t, pool, orgId, &fx.sub)

			usage := buildUsageService(t, pool)
			recordEmails(t, usage, orgId, fx, "send", tc.emails, periodStart.Add(time.Hour))

			svc := buildSubscriptionService(t, pool)
			result, err := svc.ChargeForBillingPeriod(ctx, fx.sub)
			require.NoError(t, err)
			assert.Equal(t, tc.wantCents, result.Amount,
				"%d emails graduated", tc.emails)

			inv, err := NewInvoiceRepo(pool).FindBySubscriptionCycle(ctx, orgId, fx.sub.Id, 0)
			require.NoError(t, err)
			require.Len(t, inv.LineItems, 1)
			assert.True(t, inv.LineItems[0].Quantity.Equal(decimal.NewFromInt(int64(tc.emails))),
				"usage line quantity = %d, got %s", tc.emails, inv.LineItems[0].Quantity)
			assert.Equal(t, tc.wantCents, inv.Total)
		})
	}
}
