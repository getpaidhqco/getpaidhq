//go:build integration

// Focused integration coverage for invoice status transitions through the full
// charge / cancel path against a real Postgres testcontainer.
//
// Scenarios:
//   - single failed charge, retries remaining → invoice open, sub past_due
//   - retries exhausted + FailureActionMarkUnpaid → invoice uncollectible, sub unpaid
//   - retries exhausted + FailureActionLeavePastDue → invoice stays open
//   - successful charge → invoice paid
//   - voluntary cancel on past_due sub: default → uncollectible; void → void; keep → open
//
// Reuses buildSubscriptionService, seedDecliningCard, seedUsageFixture, seedSubFixture,
// seedMemoryPsp, seedPaymentMethod, and the rest of the shared harness.
package postgresgorm

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// seedPastDueSubscriptionWithOpenInvoice drives the fixture through one failed
// charge so that:
//   - the subscription is past_due with Retries = 1 and an open invoice for cycle 0.
//
// Returns the updated (past_due) subscription so callers can cancel it.
func seedPastDueSubscriptionWithOpenInvoice(t *testing.T, orgId string) domain.Subscription {
	t.Helper()
	db := testDB(t)
	ctx := context.Background()

	jan1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	feb1 := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	fx := seedUsageFixture(t, db, orgId,
		domain.BillableMetric{Code: "api_calls_cis", Name: "API calls", Aggregation: domain.AggregationCount},
		domain.Price{Label: "API", Scheme: domain.Fixed, UnitPrice: 10},
		jan1, feb1)
	seedDecliningCard(t, db, orgId, &fx.sub)

	usage := buildUsageService(t, db)
	recordUsage(t, usage, port.RecordEventInput{
		OrgId: orgId, CustomerId: fx.customer.Id, MetricCode: fx.meter.Code,
		SubscriptionId: fx.sub.Id, ExternalId: "cis_ev1", Timestamp: jan1.Add(time.Hour),
	})

	svc := buildSubscriptionService(t, db)
	result, err := svc.ChargeForBillingPeriod(ctx, fx.sub)
	require.NoError(t, err)
	require.Equal(t, domain.PaymentStatusFailed, result.Status)

	updated, err := svc.HandleSubscriptionChargeFailure(ctx, port.SubscriptionChargeInput{
		Subscription: fx.sub, ChargeResult: result,
	})
	require.NoError(t, err)
	require.Equal(t, domain.SubscriptionStatusPastDue, updated.Status)
	require.Equal(t, 1, updated.Retries)

	return updated
}

// TestInvoiceStatus_SingleFailure_RetriesRemaining: a single declined charge with
// retries remaining leaves the invoice open and the subscription past_due.
func TestInvoiceStatus_SingleFailure_RetriesRemaining(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	jan1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	feb1 := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	fx := seedUsageFixture(t, db, orgId,
		domain.BillableMetric{Code: "calls_sfr", Name: "API calls", Aggregation: domain.AggregationCount},
		domain.Price{Label: "API", Scheme: domain.Fixed, UnitPrice: 10},
		jan1, feb1)
	seedDecliningCard(t, db, orgId, &fx.sub)

	usage := buildUsageService(t, db)
	recordUsage(t, usage, port.RecordEventInput{
		OrgId: orgId, CustomerId: fx.customer.Id, MetricCode: fx.meter.Code,
		SubscriptionId: fx.sub.Id, ExternalId: "sfr_ev1", Timestamp: jan1.Add(time.Hour),
	})

	svc := buildSubscriptionService(t, db)
	result, err := svc.ChargeForBillingPeriod(ctx, fx.sub)
	require.NoError(t, err, "a decline is a failed result, not a Go error")
	require.Equal(t, domain.PaymentStatusFailed, result.Status)

	updated, err := svc.HandleSubscriptionChargeFailure(ctx, port.SubscriptionChargeInput{
		Subscription: fx.sub, ChargeResult: result,
	})
	require.NoError(t, err)

	// Subscription: past_due with retry scheduled.
	assert.Equal(t, domain.SubscriptionStatusPastDue, updated.Status)
	assert.Equal(t, 1, updated.Retries)
	assert.False(t, updated.NextRetryAt.IsZero(), "first retry must be scheduled")

	// Invoice: open (retries remain — not written off yet).
	inv, err := NewInvoiceRepo(db).FindBySubscriptionCycle(ctx, orgId, fx.sub.Id, 0)
	require.NoError(t, err)
	assert.Equal(t, domain.InvoiceStatusOpen, inv.Status,
		"invoice stays open while retries remain")
}

// TestInvoiceStatus_RetriesExhausted_MarkUnpaid: when the retry counter reaches
// RetryAttempts and the FailureAction is mark_unpaid, the invoice is written off
// as uncollectible and the subscription becomes unpaid.
func TestInvoiceStatus_RetriesExhausted_MarkUnpaid(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	jan1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	feb1 := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	fx := seedUsageFixture(t, db, orgId,
		domain.BillableMetric{Code: "calls_remu", Name: "API calls", Aggregation: domain.AggregationCount},
		domain.Price{Label: "API", Scheme: domain.Fixed, UnitPrice: 10},
		jan1, feb1)
	seedDecliningCard(t, db, orgId, &fx.sub)

	usage := buildUsageService(t, db)
	recordUsage(t, usage, port.RecordEventInput{
		OrgId: orgId, CustomerId: fx.customer.Id, MetricCode: fx.meter.Code,
		SubscriptionId: fx.sub.Id, ExternalId: "remu_ev1", Timestamp: jan1.Add(time.Hour),
	})

	// Seed a mark_unpaid retry policy with 3 attempts (the default count). We
	// exhaust the policy by pre-advancing sub.Retries to 3 before calling the
	// failure handler, so GetNextCharge sees Retries >= RetryAttempts → zero.
	_, err := NewSettingRepo(db).Create(ctx, domain.Setting{
		OrgId:    orgId,
		ParentId: "subscriptions",
		Id:       "retry_policy",
		Value:    `{"attempts":3,"interval":"minute","retry_period":4,"failure_action":"mark_unpaid"}`,
	})
	require.NoError(t, err)

	svc := buildSubscriptionService(t, db)
	result, err := svc.ChargeForBillingPeriod(ctx, fx.sub)
	require.NoError(t, err)
	require.Equal(t, domain.PaymentStatusFailed, result.Status)

	// Simulate retries already exhausted: advance sub.Retries to 3 (== RetryAttempts)
	// so that GetNextCharge returns zero and the failure handler writes off the invoice.
	exhaustedSub := fx.sub
	exhaustedSub.Retries = 3
	require.NoError(t, db.Model(&subscriptionRow{}).
		Where("org_id = ? AND id = ?", orgId, fx.sub.Id).
		Update("retries", 3).Error)

	updated, err := svc.HandleSubscriptionChargeFailure(ctx, port.SubscriptionChargeInput{
		Subscription: exhaustedSub, ChargeResult: result,
	})
	require.NoError(t, err)

	assert.Equal(t, domain.SubscriptionStatusUnpaid, updated.Status,
		"mark_unpaid action → subscription unpaid")

	inv, err := NewInvoiceRepo(db).FindBySubscriptionCycle(ctx, orgId, fx.sub.Id, 0)
	require.NoError(t, err)
	assert.Equal(t, domain.InvoiceStatusUncollectible, inv.Status,
		"mark_unpaid exhausted → invoice uncollectible")
}

// TestInvoiceStatus_RetriesExhausted_LeavePastDue: when FailureAction is
// past_due the invoice is NOT written off — it stays open even after exhaustion.
func TestInvoiceStatus_RetriesExhausted_LeavePastDue(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	jan1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	feb1 := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	fx := seedUsageFixture(t, db, orgId,
		domain.BillableMetric{Code: "calls_relpd", Name: "API calls", Aggregation: domain.AggregationCount},
		domain.Price{Label: "API", Scheme: domain.Fixed, UnitPrice: 10},
		jan1, feb1)
	seedDecliningCard(t, db, orgId, &fx.sub)

	recordUsage(t, buildUsageService(t, db), port.RecordEventInput{
		OrgId: orgId, CustomerId: fx.customer.Id, MetricCode: fx.meter.Code,
		SubscriptionId: fx.sub.Id, ExternalId: "relpd_ev1", Timestamp: jan1.Add(time.Hour),
	})

	// Seed a past_due policy with 3 attempts; exhaust by pre-advancing Retries.
	_, err := NewSettingRepo(db).Create(ctx, domain.Setting{
		OrgId:    orgId,
		ParentId: "subscriptions",
		Id:       "retry_policy",
		Value:    `{"attempts":3,"interval":"minute","retry_period":4,"failure_action":"past_due"}`,
	})
	require.NoError(t, err)

	svc := buildSubscriptionService(t, db)
	result, err := svc.ChargeForBillingPeriod(ctx, fx.sub)
	require.NoError(t, err)
	require.Equal(t, domain.PaymentStatusFailed, result.Status)

	// Exhaust retries: set Retries = RetryAttempts (3) so GetNextCharge → zero.
	exhaustedSub := fx.sub
	exhaustedSub.Retries = 3
	require.NoError(t, db.Model(&subscriptionRow{}).
		Where("org_id = ? AND id = ?", orgId, fx.sub.Id).
		Update("retries", 3).Error)

	_, err = svc.HandleSubscriptionChargeFailure(ctx, port.SubscriptionChargeInput{
		Subscription: exhaustedSub, ChargeResult: result,
	})
	require.NoError(t, err)

	inv, err := NewInvoiceRepo(db).FindBySubscriptionCycle(ctx, orgId, fx.sub.Id, 0)
	require.NoError(t, err)
	assert.Equal(t, domain.InvoiceStatusOpen, inv.Status,
		"past_due failure action leaves invoice open even after exhaustion")
}

// TestInvoiceStatus_SuccessfulCharge: a successful charge marks the invoice paid.
func TestInvoiceStatus_SuccessfulCharge(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	jan1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	feb1 := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	fx := seedUsageFixture(t, db, orgId,
		domain.BillableMetric{Code: "calls_succ", Name: "API calls", Aggregation: domain.AggregationCount},
		domain.Price{Label: "API", Scheme: domain.Fixed, UnitPrice: 10},
		jan1, feb1)
	seedMemoryPspForSub(t, db, orgId, &fx.sub)

	recordUsage(t, buildUsageService(t, db), port.RecordEventInput{
		OrgId: orgId, CustomerId: fx.customer.Id, MetricCode: fx.meter.Code,
		SubscriptionId: fx.sub.Id, ExternalId: "succ_ev1", Timestamp: jan1.Add(time.Hour),
	})

	svc := buildSubscriptionService(t, db)
	result, err := svc.ChargeForBillingPeriod(ctx, fx.sub)
	require.NoError(t, err)
	require.Equal(t, domain.PaymentStatusSucceeded, result.Status)

	_, err = svc.HandleSubscriptionChargeSuccess(ctx, port.SubscriptionChargeInput{
		Subscription: fx.sub, ChargeResult: result,
	})
	require.NoError(t, err)

	inv, err := NewInvoiceRepo(db).FindBySubscriptionCycle(ctx, orgId, fx.sub.Id, 0)
	require.NoError(t, err)
	assert.Equal(t, domain.InvoiceStatusPaid, inv.Status,
		"a succeeded charge marks the invoice paid")
}

// TestInvoiceStatus_CancelPastDueSub_OutstandingInvoiceActions exercises all
// three OutstandingInvoiceAction values when cancelling a past_due subscription
// that has an open (not yet written off) current-cycle invoice.
func TestInvoiceStatus_CancelPastDueSub_OutstandingInvoiceActions(t *testing.T) {
	cases := []struct {
		name       string
		action     port.OutstandingInvoiceAction
		wantStatus domain.InvoiceStatus
	}{
		{
			name:       "default (empty) → uncollectible",
			action:     "", // empty ⇒ applyOutstandingInvoiceAction defaults to uncollectible
			wantStatus: domain.InvoiceStatusUncollectible,
		},
		{
			name:       "OutstandingInvoiceUncollectible → uncollectible",
			action:     port.OutstandingInvoiceUncollectible,
			wantStatus: domain.InvoiceStatusUncollectible,
		},
		{
			name:       "OutstandingInvoiceVoid → void",
			action:     port.OutstandingInvoiceVoid,
			wantStatus: domain.InvoiceStatusVoid,
		},
		{
			name:       "OutstandingInvoiceKeep → stays open",
			action:     port.OutstandingInvoiceKeep,
			wantStatus: domain.InvoiceStatusOpen,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			db := testDB(t)
			ctx := context.Background()
			orgId := uniqueOrg(t)
			cleanupOrg(t, db, orgId)

			// Drive to a past_due state with an open current-cycle invoice.
			pastDueSub := seedPastDueSubscriptionWithOpenInvoice(t, orgId)

			// Confirm the invoice is open before we cancel.
			inv, err := NewInvoiceRepo(db).FindBySubscriptionCycle(ctx, orgId, pastDueSub.Id, 0)
			require.NoError(t, err)
			require.Equal(t, domain.InvoiceStatusOpen, inv.Status,
				"pre-cancel: invoice must be open")

			svc := buildSubscriptionService(t, db)
			_, err = svc.CancelSubscription(ctx, port.CancelSubscriptionInput{
				OrgId:              orgId,
				Id:                 pastDueSub.Id,
				OutstandingInvoice: tc.action,
			})
			require.NoError(t, err)

			// Re-read the invoice.
			inv, err = NewInvoiceRepo(db).FindBySubscriptionCycle(ctx, orgId, pastDueSub.Id, 0)
			require.NoError(t, err)
			assert.Equal(t, tc.wantStatus, inv.Status,
				"after cancel with action %q invoice status", tc.action)
		})
	}
}
