//go:build integration

// End-to-end acceptance for the order → combined-invoice flow against real
// Postgres (testcontainer via testDB(t)), driving a REAL wired InvoiceService
// (with a real InvoiceSettingsService resolver and the real gorm repos) through
// OrderService.CreateOrder/CompleteOrder.
//
// This closes the gap left by coupon_billing_e2e_test.go, which passes nil for
// invoiceService: here the new CompleteOrder → BuildForOrder → MarkOpen/
// MarkSettled path (and the nested RunInTx across CompleteOrder → Consume →
// BuildForOrder on a real gorm DB) is exercised against Postgres. We prove the
// ONE combined invoice is built and settled, the order discount is applied, the
// reference is formatted from invoice settings, the upfront invoice is opened at
// create time and reused (not duplicated) at completion, and engine cycle-0
// ownership holds (sub active with CyclesProcessed == 1 — the engine would bill
// cycle 1 next, never cycle 0).
package postgresgorm

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

// buildWiredInvoiceService constructs the real InvoiceService off the
// testcontainer db, with a real InvoiceSettingsService resolver (mirroring
// app.go's construction order: settings → invoice). This is the service the
// order flow drives end to end.
func buildWiredInvoiceService(t *testing.T, db *gorm.DB) *service.InvoiceService {
	t.Helper()
	logger := noopLogger{}
	settings := service.NewInvoiceSettingsService(NewSettingRepo(db), logger)
	return service.NewInvoiceService(
		NewInvoiceRepo(db),
		NewOrderRepo(db),
		NewPriceRepo(db),
		NewSubscriptionRepo(db),
		buildUsageService(t, db),
		NewTxManager(db),
		logger,
		NewDiscountRepo(db),
		NewCouponRepo(db),
		NewCouponReservationRepo(db),
		settings, // real resolver
	)
}

// buildWiredOrderService wires an engine-aware OrderService off the testcontainer
// db with the memory gateway, the coupon service, and a REAL InvoiceService —
// mirroring app.go's NewOrderService argument order. Unlike buildOrderService
// (which passes nil for invoicing), this exercises the full order-invoicing path.
func buildWiredOrderService(t *testing.T, db *gorm.DB, coupons *service.CouponService, invoices *service.InvoiceService) *service.OrderService {
	t.Helper()
	logger := noopLogger{}
	gatewayFactory := service.NewGatewayFactory(
		NewPspRepo(db),
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
		invoices,
	)
}

// seedOneTimePrice seeds a one-time (non-recurring) price of unitPrice cents on a
// fresh product/variant, returning the product and price ids. A one-time line
// starts no subscription — it is billed once on the combined invoice.
func seedOneTimePrice(t *testing.T, db *gorm.DB, orgId string, unitPrice int64) (productId, priceId string) {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond)
	variantId := seedVariantChain(t, db, orgId)
	var v variantRow
	require.NoError(t, db.Where("org_id = ? AND id = ?", orgId, variantId).First(&v).Error)
	productId = v.ProductId

	p := domain.Price{
		OrgId:              orgId,
		Id:                 lib.GenerateId("price"),
		VariantId:          variantId,
		Label:              "Setup fee",
		Category:           domain.OneTime,
		Scheme:             domain.Fixed,
		Currency:           domain.USD,
		UnitPrice:          unitPrice,
		UnitCount:          1,
		BillingInterval:    domain.BillingIntervalNone,
		BillingIntervalQty: 0,
		TrialInterval:      domain.BillingIntervalNone,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	row := priceRowFromDomain(p)
	require.NoError(t, db.Create(&row).Error)
	return productId, p.Id
}

// TestOrderInvoicing_MixedCart_DirectPayment_E2E: a recurring $100/mo line + a
// one-time $50 line, completed with a $150 caller payment. Exactly ONE combined
// invoice is built and settled paid, with two line items, the subscription's id,
// a default-formatted reference, and the persisted Payment linked to it. The
// subscription activates with CyclesProcessed == 1 (cycle 0 is owned here; the
// engine bills cycle 1 next).
func TestOrderInvoicing_MixedCart_DirectPayment_E2E(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	// $100/mo recurring price (1-cycle cap is irrelevant; uncapped here).
	subProductId, subPriceId := seedSubscriptionPrice(t, db, orgId, 0)
	// Overwrite the seeded $100/minute price's cadence to a clean monthly $100.
	require.NoError(t, db.Model(&priceRow{}).
		Where("org_id = ? AND id = ?", orgId, subPriceId).
		Updates(map[string]any{
			"unit_price":       10000,
			"billing_interval": string(domain.BillingIntervalMonth),
			"cycles":           0,
		}).Error)
	oneTimeProductId, oneTimePriceId := seedOneTimePrice(t, db, orgId, 5000)

	customer := seedCustomer(t, db, orgId)
	pspConfigId := seedMemoryPsp(t, db, orgId)
	pm := seedPaymentMethod(t, db, orgId, customer.Id)

	coupons := buildCouponService(t, db)
	invoices := buildWiredInvoiceService(t, db)
	orders := buildWiredOrderService(t, db, coupons, invoices)

	created, err := orders.CreateOrder(ctx, port.CreateOrderInput{
		OrgId:           orgId,
		Customer:        port.CreateOrderInputCustomer{Id: customer.Id},
		Currency:        "USD",
		PspId:           domain.Gateway(pspConfigId),
		PaymentMethodId: pm.Id,
		CartItems: []domain.CartItem{
			{ProductId: subProductId, PriceId: subPriceId, Quantity: 1},
			{ProductId: oneTimeProductId, PriceId: oneTimePriceId, Quantity: 1},
		},
	})
	require.NoError(t, err)
	orderId := created.Order.Id
	require.Nil(t, created.Invoice, "no upfront invoice unless opted in")

	completed, err := orders.CompleteOrder(ctx, port.CompleteOrderInput{
		OrgId:           orgId,
		Id:              orderId,
		PaymentMethodId: pm.Id,
		Payment: port.CompleteOrderInputPayment{
			PspId:       pspConfigId,
			Amount:      15000,
			Currency:    "USD",
			CompletedAt: time.Now().UTC(),
			Reference:   "ref-mixed",
		},
	})
	require.NoError(t, err)
	require.Equal(t, domain.OrderStatusCompleted, completed.Status)

	// The created subscription.
	subs, err := NewSubscriptionRepo(db).FindByOrderId(ctx, orgId, orderId)
	require.NoError(t, err)
	require.Len(t, subs, 1, "the recurring line yields one subscription")
	sub := subs[0]
	assert.Equal(t, domain.SubscriptionStatusActive, sub.Status)
	assert.Equal(t, 1, sub.CyclesProcessed, "cycle 0 is owned by the order; engine bills cycle 1 next")

	// Exactly ONE combined invoice for the order, settled paid.
	inv, err := NewInvoiceRepo(db).FindOrderInvoice(ctx, orgId, orderId)
	require.NoError(t, err)
	assert.Equal(t, int64(15000), inv.Total, "$100 recurring + $50 one-time")
	assert.Equal(t, domain.InvoiceStatusPaid, inv.Status)
	assert.Equal(t, 0, inv.Cycle)
	require.Len(t, inv.LineItems, 2, "one base line per cart line")
	assert.Equal(t, sub.Id, inv.SubscriptionId, "single-sub order → the invoice IS the sub's cycle-0 invoice")

	// Reference is the default-formatted INV- + zero-padded number.
	require.NotEmpty(t, inv.Reference)
	assert.Equal(t, domain.DefaultInvoiceSettings().FormatReference(inv.Number), inv.Reference)

	// Only one invoice exists for the order (no duplicate from the build path).
	all, total, err := NewInvoiceRepo(db).List(ctx, orgId, domain.Pagination{Page: 1, Limit: 50})
	require.NoError(t, err)
	require.Equal(t, 1, total, "exactly one invoice for the order")
	require.Len(t, all, 1)

	// The persisted Payment links to that invoice.
	payments, ptotal, err := NewPaymentRepo(db).FindBySubscriptionId(ctx, orgId, sub.Id, domain.Pagination{Page: 1, Limit: 10})
	require.NoError(t, err)
	require.Equal(t, 1, ptotal)
	assert.Equal(t, inv.Id, payments[0].InvoiceId)
	assert.Equal(t, int64(15000), payments[0].Amount)
}

// TestOrderInvoicing_OneTimeCoupon_DirectPayment_E2E: a pure one-time $200 line
// with a 25%-off coupon applied at CreateOrder, completed with payment. The ONE
// invoice is paid with DiscountTotal $50 and Total $150, carries no subscription,
// and the coupon reservation is consumed into an order-owned Discount (OrderId
// set, SubscriptionId empty) with no leftover reservation.
func TestOrderInvoicing_OneTimeCoupon_DirectPayment_E2E(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	productId, priceId := seedOneTimePrice(t, db, orgId, 20000)
	customer := seedCustomer(t, db, orgId)
	pspConfigId := seedMemoryPsp(t, db, orgId)
	pm := seedPaymentMethod(t, db, orgId, customer.Id)

	coupons := buildCouponService(t, db)
	invoices := buildWiredInvoiceService(t, db)
	orders := buildWiredOrderService(t, db, coupons, invoices)

	coupon, err := coupons.Create(ctx, orgId, port.CreateCouponInput{
		Name:         "Quarter off",
		DiscountType: string(domain.DiscountTypePercentage),
		PercentOff:   decimal.NewFromInt(25),
		Duration:     string(domain.DurationOnce),
	})
	require.NoError(t, err)
	_, err = coupons.CreateCode(ctx, orgId, coupon.Id, port.CreateCouponCodeInput{Code: "QUARTER25"})
	require.NoError(t, err)

	created, err := orders.CreateOrder(ctx, port.CreateOrderInput{
		OrgId:           orgId,
		Customer:        port.CreateOrderInputCustomer{Id: customer.Id},
		Currency:        "USD",
		PspId:           domain.Gateway(pspConfigId),
		PaymentMethodId: pm.Id,
		CartItems:       []domain.CartItem{{ProductId: productId, PriceId: priceId, Quantity: 1}},
		CouponCode:      "QUARTER25",
	})
	require.NoError(t, err)
	orderId := created.Order.Id

	reservations, err := NewCouponReservationRepo(db).FindByOrder(ctx, orgId, orderId)
	require.NoError(t, err)
	require.Len(t, reservations, 1, "CreateOrder holds one reservation")

	_, err = orders.CompleteOrder(ctx, port.CompleteOrderInput{
		OrgId:           orgId,
		Id:              orderId,
		PaymentMethodId: pm.Id,
		Payment: port.CompleteOrderInputPayment{
			PspId:       pspConfigId,
			Amount:      15000,
			Currency:    "USD",
			CompletedAt: time.Now().UTC(),
			Reference:   "ref-coupon",
		},
	})
	require.NoError(t, err)

	// A pure one-time order has no subscription.
	subs, err := NewSubscriptionRepo(db).FindByOrderId(ctx, orgId, orderId)
	require.NoError(t, err)
	require.Empty(t, subs, "a one-time-only order starts no subscription")

	// The combined invoice: $200 subtotal, $50 discount, $150 total, paid, no sub.
	inv, err := NewInvoiceRepo(db).FindOrderInvoice(ctx, orgId, orderId)
	require.NoError(t, err)
	assert.Equal(t, int64(5000), inv.DiscountTotal, "25%% of $200")
	assert.Equal(t, int64(15000), inv.Total)
	assert.Equal(t, domain.InvoiceStatusPaid, inv.Status)
	assert.Empty(t, inv.SubscriptionId, "no subscription anchors this invoice")

	// The reservation converted to an order-owned Discount.
	ds, err := NewDiscountRepo(db).ActiveForOrder(ctx, orgId, orderId)
	require.NoError(t, err)
	require.Len(t, ds, 1, "the reservation consumes into exactly one order-owned Discount")
	assert.Equal(t, orderId, ds[0].OrderId, "order owns the discount")
	assert.Empty(t, ds[0].SubscriptionId, "a one-time order's discount targets no subscription")
	assert.Equal(t, coupon.Id, ds[0].CouponId)

	gone, err := NewCouponReservationRepo(db).FindByOrder(ctx, orgId, orderId)
	require.NoError(t, err)
	assert.Empty(t, gone, "no leftover reservation after consume")
}

// TestOrderInvoicing_UpfrontInvoice_E2E: Config.UpfrontInvoice opens the combined
// invoice immediately at CreateOrder; CompleteOrder reuses the SAME invoice
// (idempotent via FindOrderInvoice) and settles it paid — no second invoice.
func TestOrderInvoicing_UpfrontInvoice_E2E(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	productId, priceId := seedOneTimePrice(t, db, orgId, 7500)
	customer := seedCustomer(t, db, orgId)
	pspConfigId := seedMemoryPsp(t, db, orgId)
	pm := seedPaymentMethod(t, db, orgId, customer.Id)

	coupons := buildCouponService(t, db)
	invoices := buildWiredInvoiceService(t, db)
	orders := buildWiredOrderService(t, db, coupons, invoices)

	created, err := orders.CreateOrder(ctx, port.CreateOrderInput{
		OrgId:           orgId,
		Customer:        port.CreateOrderInputCustomer{Id: customer.Id},
		Currency:        "USD",
		PspId:           domain.Gateway(pspConfigId),
		PaymentMethodId: pm.Id,
		CartItems:       []domain.CartItem{{ProductId: productId, PriceId: priceId, Quantity: 1}},
		Config:          domain.OrderConfig{UpfrontInvoice: true},
	})
	require.NoError(t, err)
	orderId := created.Order.Id

	// The upfront invoice exists and is open immediately after CreateOrder.
	require.NotNil(t, created.Invoice, "upfront invoice returned on the create result")
	upfront := *created.Invoice
	assert.Equal(t, domain.InvoiceStatusOpen, upfront.Status)
	assert.Equal(t, int64(7500), upfront.Total)

	persistedUpfront, err := NewInvoiceRepo(db).FindOrderInvoice(ctx, orgId, orderId)
	require.NoError(t, err)
	assert.Equal(t, upfront.Id, persistedUpfront.Id)
	assert.Equal(t, domain.InvoiceStatusOpen, persistedUpfront.Status)

	_, err = orders.CompleteOrder(ctx, port.CompleteOrderInput{
		OrgId:           orgId,
		Id:              orderId,
		PaymentMethodId: pm.Id,
		Payment: port.CompleteOrderInputPayment{
			PspId:       pspConfigId,
			Amount:      7500,
			Currency:    "USD",
			CompletedAt: time.Now().UTC(),
			Reference:   "ref-upfront",
		},
	})
	require.NoError(t, err)

	// The SAME invoice is now paid — no second invoice was created.
	settled, err := NewInvoiceRepo(db).FindOrderInvoice(ctx, orgId, orderId)
	require.NoError(t, err)
	assert.Equal(t, upfront.Id, settled.Id, "completion reuses the upfront invoice")
	assert.Equal(t, domain.InvoiceStatusPaid, settled.Status)

	_, total, err := NewInvoiceRepo(db).List(ctx, orgId, domain.Pagination{Page: 1, Limit: 50})
	require.NoError(t, err)
	assert.Equal(t, 1, total, "exactly one invoice across create + complete")
}

// TestOrderInvoicing_UpfrontInvoiceWithCoupon_E2E: with Config.UpfrontInvoice AND
// a coupon, the OPEN invoice built at CreateOrder time is ALREADY discounted from
// the order's live reservation (before the reservation is consumed into a
// Discount). CompleteOrder reuses the SAME invoice, settles it paid (still
// discounted), and the reservation converts to an order-owned Discount. This is
// the exact pre-payment-discount case the review flagged as untested.
func TestOrderInvoicing_UpfrontInvoiceWithCoupon_E2E(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	productId, priceId := seedOneTimePrice(t, db, orgId, 20000)
	customer := seedCustomer(t, db, orgId)
	pspConfigId := seedMemoryPsp(t, db, orgId)
	pm := seedPaymentMethod(t, db, orgId, customer.Id)

	coupons := buildCouponService(t, db)
	invoices := buildWiredInvoiceService(t, db)
	orders := buildWiredOrderService(t, db, coupons, invoices)

	coupon, err := coupons.Create(ctx, orgId, port.CreateCouponInput{
		Name:         "Quarter off",
		DiscountType: string(domain.DiscountTypePercentage),
		PercentOff:   decimal.NewFromInt(25),
		Duration:     string(domain.DurationOnce),
	})
	require.NoError(t, err)
	_, err = coupons.CreateCode(ctx, orgId, coupon.Id, port.CreateCouponCodeInput{Code: "QUARTER25"})
	require.NoError(t, err)

	created, err := orders.CreateOrder(ctx, port.CreateOrderInput{
		OrgId:           orgId,
		Customer:        port.CreateOrderInputCustomer{Id: customer.Id},
		Currency:        "USD",
		PspId:           domain.Gateway(pspConfigId),
		PaymentMethodId: pm.Id,
		CartItems:       []domain.CartItem{{ProductId: productId, PriceId: priceId, Quantity: 1}},
		CouponCode:      "QUARTER25",
		Config:          domain.OrderConfig{UpfrontInvoice: true},
	})
	require.NoError(t, err)
	orderId := created.Order.Id

	// At this point the reservation exists but is NOT yet a committed Discount.
	res, err := NewCouponReservationRepo(db).FindByOrder(ctx, orgId, orderId)
	require.NoError(t, err)
	require.Len(t, res, 1, "CreateOrder holds one reservation")
	ds0, err := NewDiscountRepo(db).ActiveForOrder(ctx, orgId, orderId)
	require.NoError(t, err)
	require.Empty(t, ds0, "no committed Discount before completion — only a reservation")

	// The OPEN upfront invoice is ALREADY discounted from the reservation's coupon.
	require.NotNil(t, created.Invoice, "upfront invoice returned on the create result")
	upfront := *created.Invoice
	assert.Equal(t, domain.InvoiceStatusOpen, upfront.Status)
	assert.Equal(t, int64(5000), upfront.DiscountTotal, "open invoice discounted from the reservation: 25%% of $200")
	assert.Equal(t, int64(15000), upfront.Total, "$200 less 25%% = $150 BEFORE payment")

	persistedUpfront, err := NewInvoiceRepo(db).FindOrderInvoice(ctx, orgId, orderId)
	require.NoError(t, err)
	assert.Equal(t, upfront.Id, persistedUpfront.Id)
	assert.Equal(t, int64(5000), persistedUpfront.DiscountTotal)
	assert.Equal(t, int64(15000), persistedUpfront.Total)

	_, err = orders.CompleteOrder(ctx, port.CompleteOrderInput{
		OrgId:           orgId,
		Id:              orderId,
		PaymentMethodId: pm.Id,
		Payment: port.CompleteOrderInputPayment{
			PspId:       pspConfigId,
			Amount:      15000,
			Currency:    "USD",
			CompletedAt: time.Now().UTC(),
			Reference:   "ref-upfront-coupon",
		},
	})
	require.NoError(t, err)

	// SAME invoice, now paid, still discounted — totals stay consistent.
	settled, err := NewInvoiceRepo(db).FindOrderInvoice(ctx, orgId, orderId)
	require.NoError(t, err)
	assert.Equal(t, upfront.Id, settled.Id, "completion reuses the upfront invoice")
	assert.Equal(t, domain.InvoiceStatusPaid, settled.Status)
	assert.Equal(t, int64(5000), settled.DiscountTotal, "discount preserved through completion")
	assert.Equal(t, int64(15000), settled.Total)

	_, total, err := NewInvoiceRepo(db).List(ctx, orgId, domain.Pagination{Page: 1, Limit: 50})
	require.NoError(t, err)
	assert.Equal(t, 1, total, "exactly one invoice across create + complete")

	// The reservation converted into the committed order-owned Discount.
	ds, err := NewDiscountRepo(db).ActiveForOrder(ctx, orgId, orderId)
	require.NoError(t, err)
	require.Len(t, ds, 1, "the reservation consumes into exactly one order-owned Discount")
	assert.Equal(t, coupon.Id, ds[0].CouponId)
	gone, err := NewCouponReservationRepo(db).FindByOrder(ctx, orgId, orderId)
	require.NoError(t, err)
	assert.Empty(t, gone, "no leftover reservation after consume")
}

// TestOrderInvoicing_CustomInvoiceSettings_E2E: a custom prefix + padding set via
// the real InvoiceSettingsService is honoured by the build path — the order
// invoice's reference starts with the prefix and is zero-padded to the width.
func TestOrderInvoicing_CustomInvoiceSettings_E2E(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	productId, priceId := seedOneTimePrice(t, db, orgId, 4200)
	customer := seedCustomer(t, db, orgId)
	pspConfigId := seedMemoryPsp(t, db, orgId)
	pm := seedPaymentMethod(t, db, orgId, customer.Id)

	settings := service.NewInvoiceSettingsService(NewSettingRepo(db), noopLogger{})
	require.NoError(t, settings.SetInvoiceSettings(ctx, orgId, domain.InvoiceSettings{Prefix: "ACME-", Padding: 4}))

	coupons := buildCouponService(t, db)
	// Reuse the same settings resolver instance so the build path reads ACME-.
	invoices := service.NewInvoiceService(
		NewInvoiceRepo(db), NewOrderRepo(db), NewPriceRepo(db), NewSubscriptionRepo(db),
		buildUsageService(t, db), NewTxManager(db), noopLogger{},
		NewDiscountRepo(db), NewCouponRepo(db), NewCouponReservationRepo(db), settings,
	)
	orders := buildWiredOrderService(t, db, coupons, invoices)

	created, err := orders.CreateOrder(ctx, port.CreateOrderInput{
		OrgId:           orgId,
		Customer:        port.CreateOrderInputCustomer{Id: customer.Id},
		Currency:        "USD",
		PspId:           domain.Gateway(pspConfigId),
		PaymentMethodId: pm.Id,
		CartItems:       []domain.CartItem{{ProductId: productId, PriceId: priceId, Quantity: 1}},
		Config:          domain.OrderConfig{UpfrontInvoice: true},
	})
	require.NoError(t, err)
	require.NotNil(t, created.Invoice)

	inv, err := NewInvoiceRepo(db).FindOrderInvoice(ctx, orgId, created.Order.Id)
	require.NoError(t, err)
	assert.Equal(t, domain.InvoiceSettings{Prefix: "ACME-", Padding: 4}.FormatReference(inv.Number), inv.Reference)
	assert.Equal(t, "ACME-", inv.Reference[:5], "custom prefix honoured")
}
