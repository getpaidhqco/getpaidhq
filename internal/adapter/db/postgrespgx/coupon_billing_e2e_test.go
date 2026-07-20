//go:build integration

// End-to-end acceptance for the coupon reservation → discount → billing flow
// against real Postgres (testcontainer via poolForTest(t)). The scenario: a
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
package postgrespgx

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// TestCouponBillingE2E is the acceptance scenario: a $100/cycle subscription
// with a 50%-off repeating(2) coupon bills 2×$50 + 3×$100 over five cycles and
// ends `completed`.
func TestCouponBillingE2E(t *testing.T) {
	pool := poolForTest(t)
	ctx := context.Background()
	orgId := uniqueOrg(t, pool)
	cleanupOrg(t, pool, orgId)

	// 1. Seed the product/price, a customer, a memory PSP, and an active card.
	productId, priceId := seedSubscriptionPrice(t, pool, orgId, 5)
	customer := seedCustomer(t, pool, orgId)
	pspConfigId := seedMemoryPsp(t, pool, orgId)
	pm := seedPaymentMethod(t, pool, orgId, customer.Id)

	coupons := buildCouponService(t, pool)
	orders := buildOrderService(t, pool, coupons)

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

	reservations, err := NewCouponReservationRepo(pool).FindByOrder(ctx, orgId, orderId)
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

	subs, err := NewSubscriptionRepo(pool).FindByOrderId(ctx, orgId, orderId)
	require.NoError(t, err)
	require.Len(t, subs, 1)
	sub := subs[0]
	assert.Equal(t, domain.SubscriptionStatusActive, sub.Status, "no-upfront-payment sub activates")
	assert.Equal(t, 0, sub.CyclesProcessed)
	assert.Equal(t, 5, sub.Cycles)

	discounts, err := NewDiscountRepo(pool).ActiveForSubscription(ctx, orgId, sub.Id)
	require.NoError(t, err)
	require.Len(t, discounts, 1, "the reservation converts to exactly one Discount")
	assert.Equal(t, coupon.Id, discounts[0].CouponId)
	assert.Equal(t, 0, discounts[0].StartCycle)

	gone, err := NewCouponReservationRepo(pool).FindByOrder(ctx, orgId, orderId)
	require.NoError(t, err)
	assert.Empty(t, gone, "the reservation is cleared on consume")

	redeemedCode, err := NewCouponCodeRepo(pool).FindByCode(ctx, orgId, "LAUNCH50")
	require.NoError(t, err)
	assert.Equal(t, 1, redeemedCode.TimesRedeemed, "consume increments the code redemption count")

	// 5. Drive five billing cycles through the charge tail (the same path the
	//    billing sweep runs), reloading the sub between cycles as the flow does.
	subSvc := buildSubscriptionService(t, pool)
	subRepo := NewSubscriptionRepo(pool)

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
	payments, total, err := NewPaymentRepo(pool).FindBySubscriptionId(ctx, orgId, sub.Id, domain.Pagination{Page: 1, Limit: 50})
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
		inv, err := NewInvoiceRepo(pool).FindBySubscriptionCycle(ctx, orgId, sub.Id, cycle)
		require.NoErrorf(t, err, "invoice for cycle %d", cycle)
		assert.Equalf(t, want, inv.Total, "cycle %d invoice total", cycle)
	}

	final, err := subRepo.FindById(ctx, orgId, sub.Id)
	require.NoError(t, err)
	assert.Equal(t, domain.SubscriptionStatusCompleted, final.Status, "sub ends completed at the cycle cap")
	assert.Equal(t, 5, final.CyclesProcessed)
	assert.EqualValues(t, 5000+5000+10000+10000+10000, final.TotalRevenue)
}
