//go:build integration

// End-to-end acceptance for the coupon reservation → discount → billing flow
// against real Postgres (testcontainer via testDB(t)). The scenario: a
// $100/cycle subscription with a 50%-off repeating(2) coupon must bill
// 2×$50 + 3×$100 over five cycles and end `completed`.
//
// The test drives the real service graph — OrderService (CreateOrder reserves
// the coupon, CompleteOrder consumes it into a Discount), then
// SubscriptionService.ChargeForBillingPeriod + HandleSubscriptionChargeSuccess
// per cycle (the same charge tail the billing sweep runs). The discount is
// applied inside InvoiceService.BuildForBillingPeriod, so both engines inherit
// it. The memory gateway charges whatever the per-cycle invoice totals, so the
// payment amounts are the observable proof the discount applied for cycles 0–1
// only.
package postgresgorm

import (
	"context"
	"getpaidhq/internal/lib/ids"
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
)

// noopEngine satisfies port.Engine without orchestration. CompleteOrder's
// post-commit StartSubscriptionWorkflow call logs (not returns) on error, and
// this test drives the billing cycles by hand, so the engine is inert.
type noopEngine struct{}

func (noopEngine) StartWorkflow(context.Context, port.WorkflowType, any) (port.WorkflowResult, error) {
	return port.WorkflowResult{}, nil
}
func (noopEngine) StartSubscriptionWorkflow(context.Context, domain.Subscription) error { return nil }
func (noopEngine) UpdateSubscriptionWorkflow(context.Context, string, domain.Subscription) error {
	return nil
}
func (noopEngine) CancelSubscriptionWorkflow(context.Context, domain.Subscription) error { return nil }
func (noopEngine) SignalSubscriptionWorkflow(context.Context, string, domain.Subscription, any) error {
	return nil
}

// buildCouponService mirrors app.go's NewCouponService wiring off the
// testcontainer db (real reservation/discount/code repos so the reserve →
// consume → discount path is exercised end to end).
func buildCouponService(t *testing.T, db *gorm.DB) *service.CouponService {
	t.Helper()
	return service.NewCouponService(
		NewCouponRepo(db),
		NewCouponCodeRepo(db),
		NewDiscountRepo(db),
		noopPriorPayments{},
		NewTxManager(db),
		noopLogger{},
		NewCouponReservationRepo(db),
	)
}

// noopPriorPayments backs the FirstTimeTransaction restriction; the coupon here
// has no such restriction, so it is never consulted.
type noopPriorPayments struct{}

func (noopPriorPayments) HasPriorSuccessfulPayment(context.Context, string, string) (bool, error) {
	return false, nil
}

// noopOrderInvoicing satisfies service.OrderInvoicing without building an
// invoice: BuildForOrder returns port.ErrNotFound (order with nothing to
// invoice), so order completion does no invoicing — the behaviour this
// coupon-billing flow asserts (it does not opt into order-level invoicing; the
// per-cycle invoices are produced later by the billing tail).
type noopOrderInvoicing struct{}

func (noopOrderInvoicing) BuildForOrder(ctx context.Context, order domain.Order) (domain.Invoice, error) {
	return domain.Invoice{}, port.ErrNotFound
}

func (noopOrderInvoicing) MarkOpen(ctx context.Context, orgId, invoiceId string) (domain.Invoice, error) {
	return domain.Invoice{}, nil
}

func (noopOrderInvoicing) SettleOrderInvoice(ctx context.Context, orgId, invoiceId string) error {
	return nil
}

// buildOrderService wires an engine-aware OrderService off the testcontainer db,
// with the memory gateway registered and the coupon service threaded in.
func buildOrderService(t *testing.T, db *gorm.DB, coupons *service.CouponService) *service.OrderService {
	t.Helper()
	logger := noopLogger{}
	pspRepo := NewPspRepo(db)
	gatewayFactory := service.NewGatewayFactory(
		pspRepo,
		nil,
		logger,
		map[domain.Gateway]port.GatewayAdapter{domain.Memory: memory.NewGatewayAdapter(logger)},
	)
	return service.NewOrderService(
		NewTxManager(db),
		noopEngine{},
		NewSessionRepo(db),
		NewPriceRepo(db),
		NewCartRepo(db),
		NewOrderRepo(db),
		NewCustomerRepo(db),
		NewSubscriptionRepo(db),
		NewPaymentRepo(db),
		NewPaymentMethodRepo(db),
		NewProductRepo(db),
		gatewayFactory,
		noopPubSub{},
		logger,
		coupons,
		noopOrderInvoicing{}, // this flow does not opt into order-level invoicing
	)
}

// seedSubscriptionPrice seeds a product + variant + a fixed $100/cycle
// subscription price capped at `cycles` cycles, billed every minute, and
// returns the product and price.
func seedSubscriptionPrice(t *testing.T, db *gorm.DB, orgId string, cycles int) (productId, priceId string) {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond)
	variantId := seedVariantChain(t, db, orgId)
	// seedVariantChain creates its own product; reuse that product as the cart's.
	var v variantRow
	require.NoError(t, db.Where("org_id = ? AND id = ?", orgId, variantId).First(&v).Error)
	productId = v.ProductId

	p := domain.Price{
		OrgId:              orgId,
		Id:                 ids.Generate("price"),
		VariantId:          variantId,
		Label:              "Warp Drive Pro",
		Category:           domain.PriceCategorySubscription,
		Scheme:             domain.Fixed,
		Currency:           domain.USD,
		UnitPrice:          10000, // $100.00
		UnitCount:          1,
		BillingInterval:    domain.BillingIntervalMinute,
		BillingIntervalQty: 1,
		TrialInterval:      domain.BillingIntervalNone,
		Cycles:             cycles,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	row := priceRowFromDomain(p)
	require.NoError(t, db.Create(&row).Error)
	return productId, p.Id
}

// TestCouponBillingE2E is the acceptance scenario: a $100/cycle subscription
// with a 50%-off repeating(2) coupon bills 2×$50 + 3×$100 over five cycles and
// ends `completed`.
func TestCouponBillingE2E(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	// 1. Seed the product/price, a customer, a memory PSP, and an active card.
	productId, priceId := seedSubscriptionPrice(t, db, orgId, 5)
	customer := seedCustomer(t, db, orgId)
	pspConfigId := seedMemoryPsp(t, db, orgId)
	pm := seedPaymentMethod(t, db, orgId, customer.Id)

	coupons := buildCouponService(t, db)
	orders := buildOrderService(t, db, coupons)

	// 2. Create the 50% percentage repeating(2) coupon + code LAUNCH50.
	coupon, err := coupons.Create(ctx, orgId, port.CreateCouponInput{
		Name:             "Launch 50",
		DiscountType:     string(domain.DiscountTypePercentage),
		PercentOff:       decimal.NewFromInt(50),
		Duration:         string(domain.DurationRepeating),
		DurationInCycles: 2,
	})
	require.NoError(t, err)
	code, err := coupons.CreateCode(ctx, orgId, coupon.Id, port.CreateCouponCodeInput{Code: "LAUNCH50"})
	require.NoError(t, err)
	require.Equal(t, 0, code.TimesRedeemed)

	// 3. CreateOrder with the cart + coupon code → a reservation is held.
	created, err := orders.CreateOrder(ctx, port.CreateOrderInput{
		OrgId:           orgId,
		Customer:        port.CreateOrderInputCustomer{Id: customer.Id},
		Currency:        "USD",
		PspId:           domain.Gateway(pspConfigId),
		PaymentMethodId: pm.Id,
		CartItems:       []domain.CartItem{{ProductId: productId, PriceId: priceId, Quantity: 1}},
		CouponCode:      "LAUNCH50",
	})
	require.NoError(t, err)
	orderId := created.Order.Id

	reservations, err := NewCouponReservationRepo(db).FindByOrder(ctx, orgId, orderId)
	require.NoError(t, err)
	require.Len(t, reservations, 1, "CreateOrder must hold one reservation for the order")
	assert.Equal(t, coupon.Id, reservations[0].CouponId)
	assert.Equal(t, code.Id, reservations[0].CouponCodeId)

	// 4. CompleteOrder with no caller payment → subscription activates, the
	//    reservation converts to one Discount at start_cycle 0, the code's
	//    times_redeemed becomes 1, and the reservation is gone.
	completed, err := orders.CompleteOrder(ctx, port.CompleteOrderInput{
		OrgId:           orgId,
		Id:              orderId,
		PaymentMethodId: pm.Id,
	})
	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusCompleted, completed.Status)

	subs, err := NewSubscriptionRepo(db).FindByOrderId(ctx, orgId, orderId)
	require.NoError(t, err)
	require.Len(t, subs, 1)
	sub := subs[0]
	assert.Equal(t, domain.SubscriptionStatusActive, sub.Status, "no-upfront-payment sub activates")
	assert.Equal(t, 0, sub.CyclesProcessed)
	assert.Equal(t, 5, sub.Cycles)

	discounts, err := NewDiscountRepo(db).ActiveForSubscription(ctx, orgId, sub.Id)
	require.NoError(t, err)
	require.Len(t, discounts, 1, "the reservation converts to exactly one Discount")
	assert.Equal(t, coupon.Id, discounts[0].CouponId)
	assert.Equal(t, 0, discounts[0].StartCycle)

	gone, err := NewCouponReservationRepo(db).FindByOrder(ctx, orgId, orderId)
	require.NoError(t, err)
	assert.Empty(t, gone, "the reservation is cleared on consume")

	redeemedCode, err := NewCouponCodeRepo(db).FindByCode(ctx, orgId, "LAUNCH50")
	require.NoError(t, err)
	assert.Equal(t, 1, redeemedCode.TimesRedeemed, "consume increments the code redemption count")

	// 5. Drive five billing cycles through the charge tail (the same path the
	//    billing sweep runs), reloading the sub between cycles as the flow does.
	subSvc := buildSubscriptionService(t, db)
	subRepo := NewSubscriptionRepo(db)

	cur := sub
	for cycle := 0; cycle < 5; cycle++ {
		result, err := subSvc.ChargeForBillingPeriod(ctx, cur)
		require.NoErrorf(t, err, "charge cycle %d", cycle)
		require.Equalf(t, domain.PaymentStatusSucceeded, result.Status, "cycle %d status", cycle)

		_, err = subSvc.HandleSubscriptionChargeSuccess(ctx, port.SubscriptionChargeInput{
			Subscription: cur,
			ChargeResult: result,
		})
		require.NoErrorf(t, err, "apply success cycle %d", cycle)

		cur, err = subRepo.FindById(ctx, orgId, sub.Id)
		require.NoErrorf(t, err, "reload sub after cycle %d", cycle)
	}

	// 6. Assert the payment sequence, the terminal state, and the invoice totals.
	payments, total, err := NewPaymentRepo(db).FindBySubscriptionId(ctx, orgId, sub.Id, domain.Pagination{Page: 1, Limit: 50})
	require.NoError(t, err)
	require.Equal(t, 5, total, "five cycles → five payments")

	// FindBySubscriptionId orders newest-first; sort by cycle via creation order
	// is not guaranteed, so assert on the multiset and the per-cycle invoices.
	amounts := make([]int64, len(payments))
	for i, p := range payments {
		amounts[i] = p.Amount
	}
	assert.ElementsMatch(t, []int64{5000, 5000, 10000, 10000, 10000}, amounts,
		"50%% off for cycles 0–1, full price thereafter")

	// Per-cycle invoice totals — the authoritative, ordered proof.
	wantByCycle := []int64{5000, 5000, 10000, 10000, 10000}
	for cycle, want := range wantByCycle {
		inv, err := NewInvoiceRepo(db).FindBySubscriptionCycle(ctx, orgId, sub.Id, cycle)
		require.NoErrorf(t, err, "invoice for cycle %d", cycle)
		assert.Equalf(t, want, inv.Total, "cycle %d invoice total", cycle)
	}

	final, err := subRepo.FindById(ctx, orgId, sub.Id)
	require.NoError(t, err)
	assert.Equal(t, domain.SubscriptionStatusCompleted, final.Status, "sub ends completed at the cycle cap")
	assert.Equal(t, 5, final.CyclesProcessed)
	assert.EqualValues(t, 5000+5000+10000+10000+10000, final.TotalRevenue)
}
