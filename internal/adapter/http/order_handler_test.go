package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/eben-vranken/idempo"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/adapter/http/middleware"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
	"getpaidhq/internal/lib"
)

// fakeIdemStore is an in-memory port.IdempotencyStore for HTTP idempotency
// tests. No expiry: each test either uses a fresh store (so idempo no-ops
// without an Idempotency-Key header) or shares one instance across two
// requests to exercise the replay/conflict paths.
type fakeIdemStore struct {
	mu   sync.Mutex
	rows map[string]*fakeIdemRow
}

type fakeIdemRow struct {
	hash, state, token string
	code               int
	headers, body      []byte
}

func newFakeIdemStore() *fakeIdemStore { return &fakeIdemStore{rows: map[string]*fakeIdemRow{}} }

func (f *fakeIdemStore) Claim(_ context.Context, key, hash, token string) (port.IdempotencyClaim, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.rows[key]
	if !ok {
		f.rows[key] = &fakeIdemRow{hash: hash, state: "pending", token: token}
		return port.IdempotencyClaim{Status: port.IdempotencyNew}, nil
	}
	switch r.state {
	case "pending":
		return port.IdempotencyClaim{Status: port.IdempotencyPending}, nil
	case "completed":
		if r.hash != hash {
			return port.IdempotencyClaim{Status: port.IdempotencyConflict}, nil
		}
		return port.IdempotencyClaim{Status: port.IdempotencyCompleted, Code: r.code, Headers: r.headers, Body: r.body}, nil
	}
	return port.IdempotencyClaim{Status: port.IdempotencyConflict}, nil
}

func (f *fakeIdemStore) Complete(_ context.Context, key, token string, code int, headers, body []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r, ok := f.rows[key]; ok && r.token == token && r.state == "pending" {
		r.state, r.code, r.headers, r.body = "completed", code, headers, body
	}
	return nil
}

func (f *fakeIdemStore) Abandon(_ context.Context, key, token string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r, ok := f.rows[key]; ok && r.token == token && r.state == "pending" {
		delete(f.rows, key)
	}
	return nil
}

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
	idemStore ...*fakeIdemStore,
) *OrderHandler {
	t.Helper()
	factory := service.NewGatewayFactory(&fakePspRepo{}, fakeSecretCipher{}, silentLogger{}, map[domain.Gateway]port.GatewayAdapter{})
	// Tests that don't exercise coupons pass a nil *CouponService; wire a no-op
	// OrderCoupons so OrderService never holds a nil dependency.
	var orderCoupons service.OrderCoupons = noopCoupons{}
	if coupons != nil {
		orderCoupons = coupons
	}
	svc := service.NewOrderService(
		noopTxManager{}, engine, sessionRepo, priceRepo, cartRepo, orderRepo,
		custRepo, subRepo, payRepo, pmRepo, productRepo, factory, newPubSub(), silentLogger{}, orderCoupons, noopInvoicing{},
	)
	// Each existing test gets a fresh store (idempo no-ops without the header);
	// idempotency tests inject one shared store across two requests.
	store := newFakeIdemStore()
	if len(idemStore) > 0 && idemStore[0] != nil {
		store = idemStore[0]
	}
	idemMW := middleware.NewIdempotencyMiddleware(store, idempo.Options{})
	return NewOrderHandler(svc, silentLogger{}, newRealAuthz(t), idemMW)
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

func TestOrderHandler_CreateOrder_ThreadsUpfrontInvoiceFlag(t *testing.T) {
	// upfront_invoice:true in the request must thread through to
	// port.CreateOrderInput.Config and be persisted onto the created order.
	// (The harness wires a nil invoiceService, so no invoice is actually built;
	// the response-side mapping of rsp.Invoice is covered by a direct unit test.)
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
		PspId:          "paystack",
		Customer:       CreateOrderRequestCustomer{Email: "a@b.com", FirstName: "A"},
		UpfrontInvoice: true,
		Cart: CartInput{
			Currency: "USD",
			Items:    []CartItem{{ProductId: "prod_1", PriceId: "price_1", Quantity: 1}},
		},
	})

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	require.Len(t, orderRepo.created, 1, "an order row was created")
	assert.True(t, orderRepo.created[0].Config.UpfrontInvoice,
		"upfront_invoice request flag threads to Order.Config.UpfrontInvoice")

	// No invoiceService is wired in this harness, so no invoice is raised and
	// the response omits the field.
	var resp CreateOrderResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Nil(t, resp.Invoice, "no invoice raised → invoice field omitted")
	assert.NotContains(t, rec.Body.String(), "\"invoice\"")
}

func TestOrderHandler_CreateOrder_OmitsInvoiceByDefault(t *testing.T) {
	// upfront_invoice omitted (false) → Config flag is false and the response
	// carries no invoice field.
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
	require.Len(t, orderRepo.created, 1)
	assert.False(t, orderRepo.created[0].Config.UpfrontInvoice)
	assert.NotContains(t, rec.Body.String(), "\"invoice\"", "invoice field omitted by default")
}

// TestCreateOrderResponse_InvoiceMapping covers the response-side mapping of
// rsp.Invoice → CreateOrderResponse.Invoice {id,url} directly, since the HTTP
// harness wires a nil invoiceService and never raises an invoice.
func TestCreateOrderResponse_InvoiceMapping(t *testing.T) {
	t.Run("invoice raised → field present with id and (placeholder) url", func(t *testing.T) {
		rsp := port.CreateOrderResult{Invoice: &domain.Invoice{Id: "inv_123"}}

		resp := CreateOrderResponse{}
		if rsp.Invoice != nil {
			resp.Invoice = &CreateOrderInvoice{Id: rsp.Invoice.Id, Url: ""}
		}

		require.NotNil(t, resp.Invoice)
		assert.Equal(t, "inv_123", resp.Invoice.Id)
		assert.Equal(t, "", resp.Invoice.Url)

		out, err := json.Marshal(resp)
		require.NoError(t, err)
		assert.Contains(t, string(out), `"invoice":{"id":"inv_123","url":""}`)
	})

	t.Run("no invoice → field omitted", func(t *testing.T) {
		rsp := port.CreateOrderResult{Invoice: nil}

		resp := CreateOrderResponse{}
		if rsp.Invoice != nil {
			resp.Invoice = &CreateOrderInvoice{Id: rsp.Invoice.Id, Url: ""}
		}

		assert.Nil(t, resp.Invoice)
		out, err := json.Marshal(resp)
		require.NoError(t, err)
		assert.NotContains(t, string(out), `"invoice"`)
	})
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

// doJSONWithHeaders is doJSON plus arbitrary request headers (used to set
// Idempotency-Key). Mirrors doJSON's middleware composition so the
// AuthUser-injecting fixedAuthMiddleware still runs.
func doJSONWithHeaders(t *testing.T, ts *testSrv, method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	req := httptest.NewRequest(method, path, strings.NewReader(string(raw)))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	asHandler(ts.srv, ts.mws).ServeHTTP(rec, req)
	return rec
}

func validOrderBody() CreateOrderRequest {
	return CreateOrderRequest{
		PspId:    "paystack",
		Customer: CreateOrderRequestCustomer{Email: "a@b.com", FirstName: "A"},
		Cart: CartInput{
			Currency: "USD",
			Items:    []CartItem{{ProductId: "prod_1", PriceId: "price_1", Quantity: 1}},
		},
	}
}

func newOrderHandlerForIdemTest(t *testing.T, store *fakeIdemStore) (*OrderHandler, *fakeOrderRepo) {
	t.Helper()
	orderRepo := &fakeOrderRepo{}
	prod := &fakeProductRepo{byId: domain.Product{Id: "prod_1", Name: "Plan"}}
	price := &fakePriceRepo{byId: domain.Price{Id: "price_1", UnitPrice: 1000, Category: domain.OneTime}}
	h := newOrderHandlerForTest(t,
		orderRepo, &fakeCustomerRepo{}, &fakeSubRepo{}, &fakeCartRepo{},
		prod, price, &fakePaymentMethodRepo{}, &fakePaymentRepo{},
		&fakeSessionRepo{}, &recordingEngine{},
		nil,
		store,
	)
	return h, orderRepo
}

func TestOrderHandler_CreateOrder_Idempotent_Replay(t *testing.T) {
	// Same key + same body twice → the order is created once and the second
	// response is replayed verbatim with the Idempotency-Replayed header.
	store := newFakeIdemStore()
	h, orderRepo := newOrderHandlerForIdemTest(t, store)
	ts := newTestServer(fixedAuthMiddleware(ownerUser()))
	h.RegisterRoutes(ts.api())

	hdr := map[string]string{"Idempotency-Key": "k1"}
	first := doJSONWithHeaders(t, ts, http.MethodPost, "/api/orders", validOrderBody(), hdr)
	require.Equal(t, http.StatusOK, first.Code, "body=%s", first.Body.String())

	second := doJSONWithHeaders(t, ts, http.MethodPost, "/api/orders", validOrderBody(), hdr)
	require.Equal(t, http.StatusOK, second.Code, "body=%s", second.Body.String())

	assert.Equal(t, "true", second.Header().Get("Idempotency-Replayed"), "the replay must carry the replayed header")
	assert.Empty(t, first.Header().Get("Idempotency-Replayed"), "the original response is not a replay")
	assert.Equal(t, first.Body.String(), second.Body.String(), "the replay body matches the original")
	require.Len(t, orderRepo.created, 1, "the handler ran exactly once; the replay did not re-create the order")
}

func TestOrderHandler_CreateOrder_Idempotent_BodyMismatch(t *testing.T) {
	// Same key + a different body on the second call → 422 (the request does
	// not match the fingerprint of the completed claim).
	store := newFakeIdemStore()
	h, orderRepo := newOrderHandlerForIdemTest(t, store)
	ts := newTestServer(fixedAuthMiddleware(ownerUser()))
	h.RegisterRoutes(ts.api())

	hdr := map[string]string{"Idempotency-Key": "k1"}
	first := doJSONWithHeaders(t, ts, http.MethodPost, "/api/orders", validOrderBody(), hdr)
	require.Equal(t, http.StatusOK, first.Code, "body=%s", first.Body.String())

	changed := validOrderBody()
	changed.Customer.Email = "different@b.com"
	second := doJSONWithHeaders(t, ts, http.MethodPost, "/api/orders", changed, hdr)

	require.Equal(t, http.StatusUnprocessableEntity, second.Code, "body=%s", second.Body.String())
	require.Len(t, orderRepo.created, 1, "the mismatched replay never reaches the handler")
}

func TestOrderHandler_CreateOrder_NoKey_IndependentCreates(t *testing.T) {
	// No Idempotency-Key header → today's behaviour: two independent creates,
	// no replay header.
	store := newFakeIdemStore()
	h, orderRepo := newOrderHandlerForIdemTest(t, store)
	ts := newTestServer(fixedAuthMiddleware(ownerUser()))
	h.RegisterRoutes(ts.api())

	first := doJSON(t, ts, http.MethodPost, "/api/orders", validOrderBody())
	require.Equal(t, http.StatusOK, first.Code, "body=%s", first.Body.String())
	second := doJSON(t, ts, http.MethodPost, "/api/orders", validOrderBody())
	require.Equal(t, http.StatusOK, second.Code, "body=%s", second.Body.String())

	assert.Empty(t, first.Header().Get("Idempotency-Replayed"))
	assert.Empty(t, second.Header().Get("Idempotency-Replayed"))
	require.Len(t, orderRepo.created, 2, "without a key, each request creates its own order")
}

// stubInitGateway is a port.PaymentGateway that records InitPayment calls and
// returns a fixed session. Only InitPayment is exercised by /pay.
type stubInitGateway struct {
	port.PaymentGateway
	calls   int
	session any
}

func (g *stubInitGateway) InitPayment(_ context.Context, _ port.InitPaymentInput) (port.InitPaymentResponse, error) {
	g.calls++
	return port.InitPaymentResponse{PspResponse: g.session}, nil
}

// stubGatewayAdapter satisfies port.GatewayAdapter, handing back a fixed gateway.
type stubGatewayAdapter struct {
	gw port.PaymentGateway
}

func (a stubGatewayAdapter) CreateGateway(_ map[string]string, _ map[string]domain.Secret) (port.PaymentGateway, error) {
	return a.gw, nil
}

func (a stubGatewayAdapter) CreateWebhookParser() domain.WebhookParser { return nil }

// newOrderHandlerForPayTest assembles an OrderService whose GatewayFactory
// resolves a stub gateway. The fakePspRepo returns a PspConfig whose PspId is
// "memory", matching the single adapter registered in the factory's map, so
// factory.NewGateway hands back gw. orderRepo is returned so tests can seed the
// order and assert the persisted session.
func newOrderHandlerForPayTest(t *testing.T, order domain.Order, gw port.PaymentGateway) (*OrderHandler, *fakeOrderRepo) {
	t.Helper()
	orderRepo := &fakeOrderRepo{byId: order}
	pspRepo := &fakePspRepo{byId: domain.PspConfig{PspId: domain.Memory}}
	factory := service.NewGatewayFactory(pspRepo, fakeSecretCipher{}, silentLogger{},
		map[domain.Gateway]port.GatewayAdapter{domain.Memory: stubGatewayAdapter{gw: gw}})
	svc := service.NewOrderService(
		noopTxManager{}, &recordingEngine{}, &fakeSessionRepo{}, &fakePriceRepo{}, &fakeCartRepo{},
		orderRepo, &fakeCustomerRepo{}, &fakeSubRepo{}, &fakePaymentRepo{}, &fakePaymentMethodRepo{},
		&fakeProductRepo{}, factory, newPubSub(), silentLogger{}, noopCoupons{}, noopInvoicing{},
	)
	idemMW := middleware.NewIdempotencyMiddleware(newFakeIdemStore(), idempo.Options{})
	return NewOrderHandler(svc, silentLogger{}, newRealAuthz(t), idemMW), orderRepo
}

func TestOrderHandler_Pay_HappyPath(t *testing.T) {
	session := map[string]any{"reference": "ps_1", "url": "https://pay/x"}
	gw := &stubInitGateway{session: session}
	order := domain.Order{OrgId: "org_1", Id: "ord_1", Status: domain.OrderStatusPending}
	h, orderRepo := newOrderHandlerForPayTest(t, order, gw)
	ts := newTestServer(fixedAuthMiddleware(ownerUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPost, "/api/orders/ord_1/pay", PayOrderRequest{Psp: "memory", Options: map[string]any{}})

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	var got PayOrderResponse
	decodeJSON(t, rec, &got)
	require.NotNil(t, got.Psp, "the response carries the PSP session payload")
	assert.Equal(t, 1, gw.calls, "the gateway initialised the session once")
	assert.NotNil(t, orderRepo.byId.PaymentSession, "the session was persisted on the order")
}

func TestOrderHandler_Pay_Idempotent_ServedFromStoredSession(t *testing.T) {
	// Two /pay calls on the same order init the gateway once; the second is
	// served from the stored payment_session (fakeOrderRepo.SetPaymentSession
	// writes byId, so the second FindById sees the stored session).
	session := map[string]any{"reference": "ps_1"}
	gw := &stubInitGateway{session: session}
	order := domain.Order{OrgId: "org_1", Id: "ord_1", Status: domain.OrderStatusPending}
	h, _ := newOrderHandlerForPayTest(t, order, gw)
	ts := newTestServer(fixedAuthMiddleware(ownerUser()))
	h.RegisterRoutes(ts.api())

	first := doJSON(t, ts, http.MethodPost, "/api/orders/ord_1/pay", PayOrderRequest{Psp: "memory"})
	require.Equal(t, http.StatusOK, first.Code, "body=%s", first.Body.String())
	second := doJSON(t, ts, http.MethodPost, "/api/orders/ord_1/pay", PayOrderRequest{Psp: "memory"})
	require.Equal(t, http.StatusOK, second.Code, "body=%s", second.Body.String())

	assert.Equal(t, 1, gw.calls, "the gateway is initialised exactly once across both calls")
	assert.Equal(t, first.Body.String(), second.Body.String(), "both calls return the same session")
}

func TestOrderHandler_Pay_NonPending_Conflict(t *testing.T) {
	gw := &stubInitGateway{session: map[string]any{"reference": "ps_1"}}
	order := domain.Order{OrgId: "org_1", Id: "ord_1", Status: domain.OrderStatusCompleted}
	h, _ := newOrderHandlerForPayTest(t, order, gw)
	ts := newTestServer(fixedAuthMiddleware(ownerUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPost, "/api/orders/ord_1/pay", PayOrderRequest{Psp: "memory"})

	assertErrorEnvelope(t, rec, http.StatusConflict, string(lib.ConflictError))
	assert.Equal(t, 0, gw.calls, "no gateway call for a non-pending order")
}

func TestOrderHandler_Pay_AuthzDenied(t *testing.T) {
	gw := &stubInitGateway{session: map[string]any{"reference": "ps_1"}}
	order := domain.Order{OrgId: "org_1", Id: "ord_1", Status: domain.OrderStatusPending}
	h, _ := newOrderHandlerForPayTest(t, order, gw)
	// member is denied CreateOrder, which /pay reuses.
	ts := newTestServer(fixedAuthMiddleware(memberUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPost, "/api/orders/ord_1/pay", PayOrderRequest{Psp: "memory"})

	assertErrorEnvelope(t, rec, http.StatusForbidden, string(lib.ForbiddenError))
	assert.Equal(t, 0, gw.calls, "denied requests never reach the service")
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
