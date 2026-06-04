//go:build integration

// End-to-end coverage of the billing *charge tail*: a due, active subscription
// is charged through SubscriptionService.ChargeForBillingPeriod (which resolves
// a gateway via the GatewayFactory) and its state is advanced by
// HandleSubscriptionChargeSuccess. The cron+fan-out orchestration that leads up
// to this point is verified separately; this test pins the part that actually
// touches money and mutates subscription state.
//
// The charge runs against an in-memory gateway (internal/adapter/memory) so no
// real PSP is contacted. The DB is a per-run Testcontainer Postgres via
// testDB(t) — never the developer's local stack, never config.NewApp, never
// env-derived DSNs.
package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"getpaidhq/internal/adapter/memory"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
	"getpaidhq/internal/lib"
)

// noopLogger is a Logger that drops everything. The charge path is chatty
// (Infof/Errorf on every step) but the test asserts on returned state, not
// logs, so silence keeps the output readable. Panicf must actually halt to
// preserve the interface contract relied on elsewhere.
type noopLogger struct{}

func (noopLogger) Debug(string, ...any)      {}
func (noopLogger) Info(string, ...any)       {}
func (noopLogger) Warn(string, ...any)       {}
func (noopLogger) Error(string, ...any)      {}
func (noopLogger) Fatal(string, ...any)      {}
func (noopLogger) Debugf(string, ...any)     {}
func (noopLogger) Infof(string, ...any)      {}
func (noopLogger) Warnf(string, ...any)      {}
func (noopLogger) Errorf(string, ...any)     {}
func (noopLogger) Panicf(t string, a ...any) { panic(t) }
func (noopLogger) Fatalf(string, ...any)     {}
func (noopLogger) Sync() error               { return nil }

// noopPubSub satisfies port.PubSub without any transport. SubscriptionService's
// constructor subscribes to "subscription.workflow.>", and the charge handlers
// publish success/failure events; none of that is asserted here, so every
// method is a no-op. Subscribe returns a real (no-op) subscription so the
// constructor's nil-check on the returned subscription is satisfied.
type noopPubSub struct{}

func (noopPubSub) Publish(string, string, any) error { return nil }
func (noopPubSub) Subscribe(string, func(string, []byte)) (port.PubSubSubscription, error) {
	return noopSubscription{}, nil
}
func (noopPubSub) Close() error { return nil }

type noopSubscription struct{}

func (noopSubscription) Unsubscribe() error { return nil }

// buildSubscriptionService mirrors app.go's NewSubscriptionService wiring, but
// with the memory gateway registered in the GatewayFactory and no-op pubsub /
// error reporter. Repos are constructed straight off the testcontainer db.
func buildSubscriptionService(t *testing.T, db *gorm.DB) *service.SubscriptionService {
	t.Helper()

	logger := noopLogger{}
	pubsub := noopPubSub{}
	reporter := lib.NewErrorReporter(logger)

	pspRepo := NewPspRepo(db)
	settingRepo := NewSettingRepo(db)
	memoryAdapter := memory.NewGatewayAdapter(logger)
	gatewayFactory := service.NewGatewayFactory(
		pspRepo,
		settingRepo,
		logger,
		map[domain.Gateway]port.GatewayAdapter{domain.Memory: memoryAdapter},
	)

	svc, err := service.NewSubscriptionService(
		NewSessionRepo(db),
		settingRepo,
		NewCartRepo(db),
		NewSubscriptionRepo(db),
		NewCustomerRepo(db),
		NewOrderRepo(db),
		NewPaymentRepo(db),
		NewPriceRepo(db),
		gatewayFactory,
		pubsub,
		reporter,
		logger,
		NewTxManager(db),
	)
	require.NoError(t, err)
	return svc
}

// seedMemoryPsp configures the org so the GatewayFactory resolves to the memory
// gateway. The factory reads gateways.FindById(orgId, id) then
// settings.FindById(orgId, gateway.Id, "settings"); it dispatches on the
// gateway row's PspId. We give the gateway row id `pspConfigId` and PspId
// domain.Memory, and a (content-irrelevant) settings row hanging off it. The
// subscription's PspId must equal `pspConfigId` so ChargeForBillingPeriod's
// NewGateway(orgId, string(sub.PspId)) lookup hits this row.
func seedMemoryPsp(t *testing.T, db *gorm.DB, orgId string) string {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond)

	pspConfigId := lib.GenerateId("gw")
	require.NoError(t, db.Create(pspConfigRowFromDomain(domain.PspConfig{
		OrgId:     orgId,
		Id:        pspConfigId,
		PspId:     domain.Memory,
		Name:      "Memory (test)",
		Active:    true,
		CreatedAt: now,
		UpdatedAt: now,
	})).Error)

	require.NoError(t, db.Create(settingRowFromDomain(domain.Setting{
		OrgId:     orgId,
		ParentId:  pspConfigId,
		Id:        "settings",
		Type:      "json",
		Value:     "{}",
		CreatedAt: now,
		UpdatedAt: now,
	})).Error)

	return pspConfigId
}

// seedPaymentMethod creates an active card the recurring charge can reference.
// ChargeForBillingPeriod fetches the payment method by sub.PaymentMethodId, so
// the sub must point at this row.
func seedPaymentMethod(t *testing.T, db *gorm.DB, orgId, customerId string) domain.PaymentMethod {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond)
	pm := domain.PaymentMethod{
		OrgId:      orgId,
		Id:         lib.GenerateId("pm"),
		Status:     domain.PaymentMethodStatusActive,
		Psp:        string(domain.Memory),
		Name:       "Visa ****4242",
		CustomerId: customerId,
		Type:       domain.PaymentMethodTypeCard,
		Token:      lib.GenerateId("tok"),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	row := paymentMethodRowFromDomain(pm)
	require.NoError(t, db.Create(&row).Error)
	return pm
}

func TestBillingChargeAdvancesState(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	// Seed the subscription graph (customer / price / order / order item / sub).
	fx := seedSubFixture(t, db, orgId)

	// Wire the sub to a memory PSP + an active payment method, and make it due:
	// active status, renewal in the past, sane interval/amount/currency.
	pspConfigId := seedMemoryPsp(t, db, orgId)
	pm := seedPaymentMethod(t, db, orgId, fx.customer.Id)

	sub := fx.sub
	sub.PspId = domain.Gateway(pspConfigId)
	sub.PaymentMethodId = pm.Id
	sub.Status = domain.SubscriptionStatusActive
	sub.Amount = 1999
	sub.Currency = "USD"
	sub.BillingInterval = domain.BillingIntervalMonth
	sub.BillingIntervalQty = 1
	sub.Cycles = 12
	sub.CyclesProcessed = 0
	sub.RenewsAt = time.Now().UTC().Add(-24 * time.Hour) // due
	subRow := subscriptionRowFromDomain(sub)
	require.NoError(t, db.Omit("Customer", "OrderItem").Create(&subRow).Error)

	svc := buildSubscriptionService(t, db)
	subRepo := NewSubscriptionRepo(db)
	paymentRepo := NewPaymentRepo(db)

	// (a) Charge the billing period -> succeeded.
	result, err := svc.ChargeForBillingPeriod(ctx, sub)
	require.NoError(t, err)
	assert.Equal(t, domain.PaymentStatusSucceeded, result.Status, "memory gateway should report a succeeded charge")
	assert.Equal(t, domain.Memory, result.Psp)
	assert.Equal(t, int64(1999), result.Amount)
	assert.NotEmpty(t, result.Reference)

	// (b) Apply the success -> state advances + a payment row is written.
	updated, err := svc.HandleSubscriptionChargeSuccess(ctx, port.SubscriptionChargeInput{
		Subscription: sub,
		ChargeResult: result,
	})
	require.NoError(t, err)
	assert.Equal(t, sub.CyclesProcessed+1, updated.CyclesProcessed, "one successful charge advances exactly one cycle")
	assert.Equal(t, domain.SubscriptionStatusActive, updated.Status, "not at cycle cap, stays active")
	assert.Equal(t, sub.TotalRevenue+sub.Amount, updated.TotalRevenue, "revenue accrues by the charged amount")
	assert.True(t, updated.RenewsAt.After(sub.RenewsAt), "renewal moves forward after a successful charge")

	payments, total, err := paymentRepo.FindBySubscriptionId(ctx, orgId, sub.Id, domain.Pagination{Page: 1, Limit: 10})
	require.NoError(t, err)
	require.Equal(t, 1, total, "exactly one payment row exists for the subscription")
	require.Len(t, payments, 1)
	assert.Equal(t, domain.PaymentStatusSucceeded, payments[0].Status)
	assert.Equal(t, int64(1999), payments[0].Amount)

	// Confirm the advance is durable (re-read the row, not the returned value).
	persisted, err := subRepo.FindById(ctx, orgId, sub.Id)
	require.NoError(t, err)
	assert.Equal(t, 1, persisted.CyclesProcessed)

	// (c) Idempotency: re-applying the SAME (now-stale) pre-charge snapshot must
	// be a no-op. The handler's per-cycle guard sees the persisted row already
	// advanced past the snapshot's cycle and skips, so cycles do NOT advance
	// again and no duplicate payment is written.
	again, err := svc.HandleSubscriptionChargeSuccess(ctx, port.SubscriptionChargeInput{
		Subscription: sub, // still CyclesProcessed == 0
		ChargeResult: result,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, again.CyclesProcessed, "stale replay must not double-advance the cycle")

	repersisted, err := subRepo.FindById(ctx, orgId, sub.Id)
	require.NoError(t, err)
	assert.Equal(t, 1, repersisted.CyclesProcessed, "persisted cycle count unchanged after replay")

	_, totalAfter, err := paymentRepo.FindBySubscriptionId(ctx, orgId, sub.Id, domain.Pagination{Page: 1, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, totalAfter, "stale replay must not write a duplicate payment")
}

// TestImmediateFirstCharge pins the "no upfront checkout payment, due now" case —
// the one the Hatchet activation-spawn of billing-cycle-runner exists to serve.
// A subscription is activated without an upfront payment (e.g. system-charges-now
// or a just-ended trial), so SetActive with a zero-amount payment (which calls
// the no-arg SetActivationDates) seeds CyclesProcessed=0, RenewsAt = StartDate
// (= now, via the cycle-0 rule), and CurrentPeriodStart = CurrentPeriodEnd = StartDate.
//
// This proves two things:
//   - the subscription IS due (IsDueForBilling == true), which is what gates the
//     Hatchet spawn;
//   - charging it via the SAME path the runner uses produces correct cycle-1
//     state AND correct period boundaries (CurrentPeriodStart == StartDate,
//     CurrentPeriodEnd == StartDate + one interval) WITHOUT any handler change —
//     i.e. the period-init guard (A1) is not needed when CurrentPeriodEnd is
//     seeded from StartDate (the non-zero value SetActive/SetActivationDates produce).
func TestImmediateFirstCharge(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	fx := seedSubFixture(t, db, orgId)

	pspConfigId := seedMemoryPsp(t, db, orgId)
	pm := seedPaymentMethod(t, db, orgId, fx.customer.Id)

	// Reconstruct the state SetActive(payment{amount:0}) leaves behind:
	// active, cycle 0, StartDate ≈ now, RenewsAt = StartDate (cycle-0 rule),
	// CurrentPeriodStart = CurrentPeriodEnd = StartDate.
	startDate := time.Now().UTC().Add(-time.Minute).Truncate(time.Microsecond)
	sub := fx.sub
	sub.PspId = domain.Gateway(pspConfigId)
	sub.PaymentMethodId = pm.Id
	sub.Status = domain.SubscriptionStatusActive
	sub.Amount = 1999
	sub.Currency = "USD"
	sub.BillingInterval = domain.BillingIntervalMonth
	sub.BillingIntervalQty = 1
	sub.Cycles = 12
	sub.CyclesProcessed = 0
	sub.StartDate = startDate
	sub.RenewsAt = startDate           // due now/past (no upfront payment)
	sub.CurrentPeriodStart = startDate // what SetActive (zero-amount) seeds
	sub.CurrentPeriodEnd = startDate   // (NOT zero — this is the load-bearing seed)
	subRow := subscriptionRowFromDomain(sub)
	require.NoError(t, db.Omit("Customer", "OrderItem").Create(&subRow).Error)

	// The activation gate: this is exactly the predicate the Hatchet
	// StartSubscriptionWorkflow checks before spawning billing-cycle-runner.
	assert.True(t, sub.IsDueForBilling(time.Now().UTC()), "no-upfront-payment sub must be immediately due")

	svc := buildSubscriptionService(t, db)
	paymentRepo := NewPaymentRepo(db)

	// (a) Charge the first period -> succeeded.
	result, err := svc.ChargeForBillingPeriod(ctx, sub)
	require.NoError(t, err)
	assert.Equal(t, domain.PaymentStatusSucceeded, result.Status)

	// (b) Apply the success -> cycle 1, still active, payment row written, and —
	// critically — correct period boundaries without any handler fix.
	updated, err := svc.HandleSubscriptionChargeSuccess(ctx, port.SubscriptionChargeInput{
		Subscription: sub,
		ChargeResult: result,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, updated.CyclesProcessed, "immediate first charge advances to cycle 1")
	assert.Equal(t, domain.SubscriptionStatusActive, updated.Status, "stays active below the cycle cap")

	payments, total, err := paymentRepo.FindBySubscriptionId(ctx, orgId, sub.Id, domain.Pagination{Page: 1, Limit: 10})
	require.NoError(t, err)
	require.Equal(t, 1, total, "exactly one payment row exists for the first charge")
	require.Len(t, payments, 1)
	assert.Equal(t, domain.PaymentStatusSucceeded, payments[0].Status)

	// The period assertion — proves period-init is already correct for cycle 1
	// without the A1 guard. CurrentPeriodStart rolls from the seeded
	// CurrentPeriodEnd (= StartDate), and CurrentPeriodEnd advances one interval.
	expectedPeriodEnd := startDate.AddDate(0, 1, 0) // StartDate + 1 month
	assert.WithinDuration(t, startDate, updated.CurrentPeriodStart, time.Second,
		"cycle-1 CurrentPeriodStart must equal StartDate")
	assert.WithinDuration(t, expectedPeriodEnd, updated.CurrentPeriodEnd, time.Second,
		"cycle-1 CurrentPeriodEnd must be StartDate + one billing interval, not zero")
	assert.False(t, updated.CurrentPeriodEnd.IsZero(), "period end must not be the zero time")
}
