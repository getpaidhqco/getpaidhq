package handler

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
	"getpaidhq/internal/lib"
)

// newOrderHandlerForTest assembles an OrderService against fake repos plus a
// real cedar authz so authz paths can be exercised the same way production
// runs them.
func newOrderHandlerForTest(
	t *testing.T,
	orderRepo *fakeOrderRepo,
	custRepo *fakeCustomerRepo,
	subRepo *fakeSubRepo,
	cartRepo *fakeCartRepo,
	productRepo *fakeProductRepo,
	priceRepo *fakePriceRepo,
	pmRepo *fakePaymentMethodRepo,
	payRepo *fakePaymentRepo,
	sessionRepo *fakeSessionRepo,
	engine *recordingEngine,
	coupons *service.CouponService,
) *OrderHandler {
	t.Helper()
	factory := service.NewGatewayFactory(&fakePspRepo{}, fakeSecretCipher{}, silentLogger{}, map[domain.Gateway]port.GatewayAdapter{})
	svc := service.NewOrderService(
		noopTxManager{}, engine, sessionRepo, priceRepo, cartRepo, orderRepo,
		custRepo, subRepo, payRepo, pmRepo, productRepo, factory, newPubSub(), silentLogger{}, coupons,
	)
	return NewOrderHandler(svc, silentLogger{}, newRealAuthz(t))
}

func TestOrderHandler_AuthzDenied_OnCreate(t *testing.T) {
	// Owner permits CreateOrder; member does not. Drive a member to assert
	// the cedar denial path.
	h := newOrderHandlerForTest(t,
		&fakeOrderRepo{}, &fakeCustomerRepo{}, &fakeSubRepo{}, &fakeCartRepo{},
		&fakeProductRepo{}, &fakePriceRepo{}, &fakePaymentMethodRepo{}, &fakePaymentRepo{},
		&fakeSessionRepo{}, &recordingEngine{},
		nil,
	)

	ts := newTestServer(fixedAuthMiddleware(memberUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPost, "/api/orders", CreateOrderRequest{
		PspId:    "paystack",
		Customer: CreateOrderRequestCustomer{Email: "a@b.com"},
		Cart:     CartInput{Currency: "USD", Items: []CartItem{{ProductId: "prod_1", PriceId: "price_1", Quantity: 1}}},
	})

	assertErrorEnvelope(t, rec, http.StatusForbidden, string(lib.ForbiddenError))
}

func TestOrderHandler_CreateOrder_RequiresCartOrSession(t *testing.T) {
	// owner authorized; empty body fails the handler-level validation guard
	// before the service is reached.
	h := newOrderHandlerForTest(t,
		&fakeOrderRepo{}, &fakeCustomerRepo{}, &fakeSubRepo{}, &fakeCartRepo{},
		&fakeProductRepo{}, &fakePriceRepo{}, &fakePaymentMethodRepo{}, &fakePaymentRepo{},
		&fakeSessionRepo{}, &recordingEngine{},
		nil,
	)
	ts := newTestServer(fixedAuthMiddleware(ownerUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPost, "/api/orders", CreateOrderRequest{
		PspId:    "paystack",
		Customer: CreateOrderRequestCustomer{Email: "a@b.com"},
	})

	// The handler returns an ApiError with code "validation_error" → 422.
	// Fuego's error serializer pipeline replaces the human-readable message
	// with the canonical HTTP title; the contract worth pinning here is the
	// code + status, not the wording.
	_ = assertErrorEnvelope(t, rec, http.StatusUnprocessableEntity, string(lib.ValidationError))
}

func TestOrderHandler_CreateOrder_HappyPath(t *testing.T) {
	// Direct-cart create: skip session/PSP-init by leaving SessionId empty;
	// service still creates the cart from the items and writes the order.
	orderRepo := &fakeOrderRepo{}
	prod := &fakeProductRepo{byId: domain.Product{Id: "prod_1", Name: "Plan"}}
	price := &fakePriceRepo{byId: domain.Price{
		Id: "price_1", UnitPrice: 1000, Category: domain.OneTime,
	}}
	h := newOrderHandlerForTest(t,
		orderRepo, &fakeCustomerRepo{}, &fakeSubRepo{}, &fakeCartRepo{},
		prod, price, &fakePaymentMethodRepo{}, &fakePaymentRepo{},
		&fakeSessionRepo{}, &recordingEngine{},
		nil,
	)
	ts := newTestServer(fixedAuthMiddleware(ownerUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPost, "/api/orders", CreateOrderRequest{
		PspId:    "paystack",
		Customer: CreateOrderRequestCustomer{Email: "a@b.com", FirstName: "A"},
		Cart: CartInput{
			Currency: "USD",
			Items:    []CartItem{{ProductId: "prod_1", PriceId: "price_1", Quantity: 1}},
		},
	})

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	require.Len(t, orderRepo.created, 1, "an order row was created")
}

func TestOrderHandler_Get(t *testing.T) {
	orderRepo := &fakeOrderRepo{byId: domain.Order{
		OrgId: "org_1", Id: "ord_1", Reference: "ref_1", Status: domain.OrderStatusPending,
	}}
	h := newOrderHandlerForTest(t,
		orderRepo, &fakeCustomerRepo{}, &fakeSubRepo{}, &fakeCartRepo{},
		&fakeProductRepo{}, &fakePriceRepo{}, &fakePaymentMethodRepo{}, &fakePaymentRepo{},
		&fakeSessionRepo{}, &recordingEngine{},
		nil,
	)
	ts := newTestServer(fixedAuthMiddleware(ownerUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodGet, "/api/orders/ord_1", nil)

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	var got OrderResponse
	decodeJSON(t, rec, &got)
	assert.Equal(t, "ord_1", got.Id)
	assert.Equal(t, "ref_1", got.Reference)
}

func TestOrderHandler_Get_NotFound(t *testing.T) {
	// OrderService.FindById wraps the underlying error as `errors.New("order not found")`
	// (a plain error), so the envelope falls through to "bad_request".
	orderRepo := &fakeOrderRepo{byIdErr: errors.New("missing")}
	h := newOrderHandlerForTest(t,
		orderRepo, &fakeCustomerRepo{}, &fakeSubRepo{}, &fakeCartRepo{},
		&fakeProductRepo{}, &fakePriceRepo{}, &fakePaymentMethodRepo{}, &fakePaymentRepo{},
		&fakeSessionRepo{}, &recordingEngine{},
		nil,
	)
	ts := newTestServer(fixedAuthMiddleware(ownerUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodGet, "/api/orders/ord_x", nil)

	assertErrorEnvelope(t, rec, http.StatusBadRequest, "bad_request")
}

func TestOrderHandler_List(t *testing.T) {
	orderRepo := &fakeOrderRepo{listResult: []domain.Order{
		{Id: "ord_1"}, {Id: "ord_2"}, {Id: "ord_3"},
	}}
	h := newOrderHandlerForTest(t,
		orderRepo, &fakeCustomerRepo{}, &fakeSubRepo{}, &fakeCartRepo{},
		&fakeProductRepo{}, &fakePriceRepo{}, &fakePaymentMethodRepo{}, &fakePaymentRepo{},
		&fakeSessionRepo{}, &recordingEngine{},
		nil,
	)
	ts := newTestServer(fixedAuthMiddleware(ownerUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodGet, "/api/orders?page=1&limit=5", nil)

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	var got ListResponse
	decodeJSON(t, rec, &got)
	assert.Equal(t, 3, got.Meta.Total)
	assert.Equal(t, 1, got.Meta.Page)
	assert.Equal(t, 5, got.Meta.Limit)
}

func TestOrderHandler_CompleteOrder(t *testing.T) {
	// Order must be pending and have a known payment method on the customer.
	orderRepo := &fakeOrderRepo{byId: domain.Order{
		OrgId: "org_1", Id: "ord_1", Status: domain.OrderStatusPending, CustomerId: "cus_1",
	}}
	custRepo := &fakeCustomerRepo{
		paymentMethod: domain.PaymentMethod{Id: "pm_1"},
	}
	subRepo := &fakeSubRepo{byOrderId: []domain.Subscription{
		{OrgId: "org_1", Id: "sub_1", Status: domain.SubscriptionStatusPending},
	}}
	engine := &recordingEngine{}
	h := newOrderHandlerForTest(t,
		orderRepo, custRepo, subRepo, &fakeCartRepo{},
		&fakeProductRepo{}, &fakePriceRepo{}, &fakePaymentMethodRepo{}, &fakePaymentRepo{},
		&fakeSessionRepo{}, engine,
		nil,
	)

	ts := newTestServer(fixedAuthMiddleware(ownerUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPost, "/api/orders/ord_1/complete", CompleteOrderRequest{
		PaymentMethodId: "pm_1",
	})

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	require.NotEmpty(t, engine.started, "completing an order with a payment method must start the subscription workflow")
}

func TestOrderHandler_CompleteOrder_InvalidCompletedAt(t *testing.T) {
	orderRepo := &fakeOrderRepo{byId: domain.Order{Id: "ord_1", Status: domain.OrderStatusPending}}
	h := newOrderHandlerForTest(t,
		orderRepo, &fakeCustomerRepo{}, &fakeSubRepo{}, &fakeCartRepo{},
		&fakeProductRepo{}, &fakePriceRepo{}, &fakePaymentMethodRepo{}, &fakePaymentRepo{},
		&fakeSessionRepo{}, &recordingEngine{},
		nil,
	)

	ts := newTestServer(fixedAuthMiddleware(ownerUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPost, "/api/orders/ord_1/complete", CompleteOrderRequest{
		PaymentMethodId: "pm_1",
		Payment:         CompleteOrderRequestPayment{CompletedAt: "not-an-rfc3339"},
	})

	assertErrorEnvelope(t, rec, http.StatusUnprocessableEntity, string(lib.ValidationError))
}

// newCouponServiceForOrderTest builds a real CouponService over in-memory fakes
// and seeds one coupon (MaxRedemptions) + one code ("LAUNCH50"). resRepo is
// returned so the caller can seed/inspect the reservation side.
func newCouponServiceForOrderTest(t *testing.T, maxRedemptions int) (*service.CouponService, *hFakeReservationRepo) {
	t.Helper()
	cr := newHFakeCouponRepo()
	ccr := newHFakeCouponCodeRepo()
	res := &hFakeReservationRepo{}
	svc := service.NewCouponService(cr, ccr, &hFakeDiscountRepo{}, &hFakePriorPayments{}, noopTxManager{}, silentLogger{}, res)

	coupon, err := svc.Create(context.Background(), "org_1", port.CreateCouponInput{
		Name:             "Launch 50",
		DiscountType:     "percentage",
		PercentOff:       decimal.NewFromInt(50),
		Duration:         "repeating",
		DurationInCycles: 2,
		MaxRedemptions:   maxRedemptions,
	})
	require.NoError(t, err)
	_, err = svc.CreateCode(context.Background(), "org_1", coupon.Id, port.CreateCouponCodeInput{Code: "LAUNCH50"})
	require.NoError(t, err)
	return svc, res
}

func TestOrderHandler_CreateOrder_CouponReserved(t *testing.T) {
	// A valid coupon_code reserves capacity and the order is created (200).
	coupons, res := newCouponServiceForOrderTest(t, 1)

	orderRepo := &fakeOrderRepo{}
	prod := &fakeProductRepo{byId: domain.Product{Id: "prod_1", Name: "Plan"}}
	price := &fakePriceRepo{byId: domain.Price{Id: "price_1", UnitPrice: 1000, Category: domain.OneTime}}
	h := newOrderHandlerForTest(t,
		orderRepo, &fakeCustomerRepo{}, &fakeSubRepo{}, &fakeCartRepo{},
		prod, price, &fakePaymentMethodRepo{}, &fakePaymentRepo{},
		&fakeSessionRepo{}, &recordingEngine{},
		coupons,
	)
	ts := newTestServer(fixedAuthMiddleware(ownerUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPost, "/api/orders", CreateOrderRequest{
		PspId:      "paystack",
		Customer:   CreateOrderRequestCustomer{Email: "a@b.com", FirstName: "A"},
		CouponCode: "LAUNCH50",
		Cart: CartInput{
			Currency: "USD",
			Items:    []CartItem{{ProductId: "prod_1", PriceId: "price_1", Quantity: 1}},
		},
	})

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	require.Len(t, orderRepo.created, 1, "an order row was created")
	require.Len(t, res.created, 1, "a reservation was recorded for the order")
}

func TestOrderHandler_CreateOrder_CouponExhausted(t *testing.T) {
	// An exhausted coupon (live holds already at the cap) refuses the reserve
	// with cap_reached → 409, and no reservation is recorded.
	coupons, res := newCouponServiceForOrderTest(t, 1)
	res.couponCount = 1 // one live hold already == MaxRedemptions

	orderRepo := &fakeOrderRepo{}
	prod := &fakeProductRepo{byId: domain.Product{Id: "prod_1", Name: "Plan"}}
	price := &fakePriceRepo{byId: domain.Price{Id: "price_1", UnitPrice: 1000, Category: domain.OneTime}}
	h := newOrderHandlerForTest(t,
		orderRepo, &fakeCustomerRepo{}, &fakeSubRepo{}, &fakeCartRepo{},
		prod, price, &fakePaymentMethodRepo{}, &fakePaymentRepo{},
		&fakeSessionRepo{}, &recordingEngine{},
		coupons,
	)
	ts := newTestServer(fixedAuthMiddleware(ownerUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPost, "/api/orders", CreateOrderRequest{
		PspId:      "paystack",
		Customer:   CreateOrderRequestCustomer{Email: "a@b.com", FirstName: "A"},
		CouponCode: "LAUNCH50",
		Cart: CartInput{
			Currency: "USD",
			Items:    []CartItem{{ProductId: "prod_1", PriceId: "price_1", Quantity: 1}},
		},
	})

	assertErrorEnvelope(t, rec, http.StatusConflict, string(lib.ConflictError))
	require.Empty(t, res.created, "no reservation is recorded when the coupon is refused")
}

func TestOrderHandler_ListSubscriptions(t *testing.T) {
	// authz: owner has no "ListOrderSubscriptions" permit; admin does (via the
	// unconditional admin rule).
	subRepo := &fakeSubRepo{byOrderId: []domain.Subscription{{Id: "sub_1"}}}
	h := newOrderHandlerForTest(t,
		&fakeOrderRepo{}, &fakeCustomerRepo{}, subRepo, &fakeCartRepo{},
		&fakeProductRepo{}, &fakePriceRepo{}, &fakePaymentMethodRepo{}, &fakePaymentRepo{},
		&fakeSessionRepo{}, &recordingEngine{},
		nil,
	)
	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodGet, "/api/orders/ord_1/subscriptions", nil)

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
}
