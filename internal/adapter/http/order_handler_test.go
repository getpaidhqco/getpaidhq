package handler

import (
	"errors"
	"net/http"
	"testing"

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
) *OrderHandler {
	t.Helper()
	factory := service.NewGatewayFactory(&fakePspRepo{}, &fakeSettingRepo{}, silentLogger{}, map[domain.Gateway]port.GatewayAdapter{})
	svc := service.NewOrderService(
		noopTxManager{}, engine, sessionRepo, priceRepo, cartRepo, orderRepo,
		custRepo, subRepo, payRepo, pmRepo, productRepo, factory, newPubSub(), silentLogger{},
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
	)

	ts := newTestServer(fixedAuthMiddleware(memberUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPost, "/api/orders", CreateOrderRequest{
		PspId:    "paystack",
		Customer: CreateOrderRequestCustomer{Email: "a@b.com"},
		Cart:     CartInput{Currency: "USD", Items: []CartItem{{ProductId: "prod_1", PriceId: "price_1", Quantity: 1}}},
	})

	assertErrorEnvelope(t, rec, http.StatusUnauthorized, string(lib.AuthenticationError))
}

func TestOrderHandler_CreateOrder_RequiresCartOrSession(t *testing.T) {
	// owner authorized; empty body fails the handler-level validation guard
	// before the service is reached.
	h := newOrderHandlerForTest(t,
		&fakeOrderRepo{}, &fakeCustomerRepo{}, &fakeSubRepo{}, &fakeCartRepo{},
		&fakeProductRepo{}, &fakePriceRepo{}, &fakePaymentMethodRepo{}, &fakePaymentRepo{},
		&fakeSessionRepo{}, &recordingEngine{},
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

func TestOrderHandler_ListSubscriptions(t *testing.T) {
	// authz: owner has no "ListOrderSubscriptions" permit; admin does (via the
	// unconditional admin rule).
	subRepo := &fakeSubRepo{byOrderId: []domain.Subscription{{Id: "sub_1"}}}
	h := newOrderHandlerForTest(t,
		&fakeOrderRepo{}, &fakeCustomerRepo{}, subRepo, &fakeCartRepo{},
		&fakeProductRepo{}, &fakePriceRepo{}, &fakePaymentMethodRepo{}, &fakePaymentRepo{},
		&fakeSessionRepo{}, &recordingEngine{},
	)
	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodGet, "/api/orders/ord_1/subscriptions", nil)

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
}
