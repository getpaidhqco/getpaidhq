package service

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

// ---- fakes specific to OrderService ----

// fakeTxManager runs the closure inline. CompleteOrder's contract is that
// post-commit side effects fire only when the closure returns nil, so a
// pass-through that propagates the closure error is enough to exercise both
// the commit and rollback branches.
type fakeTxManager struct{ ran bool }

func (m *fakeTxManager) RunInTx(ctx context.Context, fn func(context.Context) error) error {
	m.ran = true
	return fn(ctx)
}

// recordingEngine records StartSubscriptionWorkflow calls; the rest of the
// port.Engine surface is inert.
type recordingEngine struct {
	mu        sync.Mutex
	started   []domain.Subscription
	updates   []string
	startErr  error
	updateErr error
}

func (e *recordingEngine) StartWorkflow(context.Context, port.WorkflowType, any) (port.WorkflowResult, error) {
	return port.WorkflowResult{}, nil
}
func (e *recordingEngine) StartSubscriptionWorkflow(_ context.Context, sub domain.Subscription) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.startErr != nil {
		return e.startErr
	}
	e.started = append(e.started, sub)
	return nil
}
func (e *recordingEngine) UpdateSubscriptionWorkflow(_ context.Context, name string, _ domain.Subscription) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.updateErr != nil {
		return e.updateErr
	}
	e.updates = append(e.updates, name)
	return nil
}
func (e *recordingEngine) CancelSubscriptionWorkflow(context.Context, domain.Subscription) error {
	return nil
}
func (e *recordingEngine) SignalSubscriptionWorkflow(context.Context, string, domain.Subscription, any) error {
	return nil
}

type fakeOrderRepo struct {
	port.OrderRepository
	order        domain.Order
	items        []domain.OrderItem
	findErr      error
	updateErr    error
	created      []domain.Order
	updated      []domain.Order
	forUpdateHit int
	// hasCreated flips once Create is called, so FindById reflects the created
	// order (carrying its persisted Config) instead of the seeded r.order.
	hasCreated bool
}

func (r *fakeOrderRepo) FindById(_ context.Context, _, _ string) (domain.Order, error) {
	if r.findErr != nil {
		return domain.Order{}, r.findErr
	}
	return r.order, nil
}

func (r *fakeOrderRepo) Create(_ context.Context, o domain.Order) (domain.Order, error) {
	r.created = append(r.created, o)
	r.order = o
	r.hasCreated = true
	return o, nil
}

func (r *fakeOrderRepo) CreateOrderItem(_ context.Context, oi domain.OrderItem) (domain.OrderItem, error) {
	r.items = append(r.items, oi)
	return oi, nil
}

func (r *fakeOrderRepo) FindByIdForUpdate(ctx context.Context, orgId string, id string) (domain.Order, error) {
	r.forUpdateHit++
	return r.FindById(ctx, orgId, id)
}

func (r *fakeOrderRepo) FindOrderItemsByOrderId(_ context.Context, _, _ string) ([]domain.OrderItem, error) {
	return r.items, nil
}

func (r *fakeOrderRepo) FindOrderItemsBySubscriptionId(_ context.Context, _, _ string) ([]domain.OrderItem, error) {
	return r.items, nil
}

func (r *fakeOrderRepo) UpdateOrderItem(_ context.Context, oi domain.OrderItem) (domain.OrderItem, error) {
	return oi, nil
}

func (r *fakeOrderRepo) FindOrderItemById(_ context.Context, _, id string) (domain.OrderItem, error) {
	for _, it := range r.items {
		if it.Id == id {
			return it, nil
		}
	}
	// Fall back to the first item — tests that don't set Id on items still work.
	if len(r.items) > 0 {
		return r.items[0], nil
	}
	return domain.OrderItem{}, nil
}

func (r *fakeOrderRepo) Update(_ context.Context, o domain.Order) (domain.Order, error) {
	if r.updateErr != nil {
		return domain.Order{}, r.updateErr
	}
	r.updated = append(r.updated, o)
	r.order = o
	return o, nil
}

func (r *fakeOrderRepo) SetPaymentSession(_ context.Context, _, _ string, session any) error {
	r.order.PaymentSession = session
	return nil
}

// initPaymentGateway records InitPayment calls and returns a fixed session.
type initPaymentGateway struct {
	port.PaymentGateway
	calls int
	resp  port.InitPaymentResponse
}

func (g *initPaymentGateway) InitPayment(_ context.Context, _ port.InitPaymentInput) (port.InitPaymentResponse, error) {
	g.calls++
	return g.resp, nil
}

// initPaymentFactory hands back a fixed gateway.
type initPaymentFactory struct {
	port.GatewayFactory
	gw port.PaymentGateway
}

func (f *initPaymentFactory) NewGateway(_ context.Context, _, _ string) (port.PaymentGateway, error) {
	return f.gw, nil
}

type fakeCustomerRepo struct {
	port.CustomerRepository
	customer      domain.Customer
	paymentMethod domain.PaymentMethod
	findErr       error
	pmErr         error
}

func (r *fakeCustomerRepo) FindById(_ context.Context, _, _ string) (domain.Customer, error) {
	if r.findErr != nil {
		return domain.Customer{}, r.findErr
	}
	return r.customer, nil
}

func (r *fakeCustomerRepo) FindByIds(_ context.Context, _ string, ids []string) ([]domain.Customer, error) {
	return []domain.Customer{r.customer}, nil
}

func (r *fakeCustomerRepo) FindPaymentMethodById(_ context.Context, _, _ string) (domain.PaymentMethod, error) {
	if r.pmErr != nil {
		return domain.PaymentMethod{}, r.pmErr
	}
	return r.paymentMethod, nil
}

type fakePaymentMethodRepo struct {
	port.PaymentMethodRepository
	createErr error
	created   []domain.PaymentMethod
}

func (r *fakePaymentMethodRepo) Create(_ context.Context, pm domain.PaymentMethod) (domain.PaymentMethod, error) {
	if r.createErr != nil {
		return domain.PaymentMethod{}, r.createErr
	}
	r.created = append(r.created, pm)
	return pm, nil
}

type fakePaymentRepo struct {
	port.PaymentRepository
	createErr error
	created   []domain.Payment
}

func (r *fakePaymentRepo) Create(_ context.Context, p domain.Payment) (domain.Payment, error) {
	if r.createErr != nil {
		return domain.Payment{}, r.createErr
	}
	r.created = append(r.created, p)
	return p, nil
}

func newOrderServiceForTest(
	tx port.TxManager,
	engine port.Engine,
	orderRepo port.OrderRepository,
	custRepo port.CustomerRepository,
	subRepo port.SubscriptionRepository,
	pmRepo port.PaymentMethodRepository,
	payRepo port.PaymentRepository,
	ps port.PubSub,
) *OrderService {
	// session/cart/price/product repos and gateway factory are unused by
	// CompleteOrder.
	return NewOrderService(tx, engine, nil, &fakePriceRepo{}, nil, orderRepo, custRepo, subRepo, payRepo, pmRepo, nil, nil, ps, silentLogger{}, nil, nil)
}

// newOrderServiceWithInvoice wires an invoice-aware OrderService for CompleteOrder
// tests: a real InvoiceService (over a fake invoice repo + the given order
// items/prices) so the combined invoice is built and settled, plus an optional
// CouponService.
func newOrderServiceWithInvoice(
	tx port.TxManager,
	engine port.Engine,
	orderRepo port.OrderRepository,
	custRepo port.CustomerRepository,
	subRepo port.SubscriptionRepository,
	pmRepo port.PaymentMethodRepository,
	payRepo port.PaymentRepository,
	ps port.PubSub,
	priceRepo port.PriceRepository,
	invRepo *fakeInvoiceRepo,
	coupons *CouponService,
) *OrderService {
	invSvc := NewInvoiceService(invRepo, orderRepo, priceRepo, subRepo, noopUsage{}, tx, silentLogger{}, noopDiscountRepo{}, noopCouponRepo{}, noopReservationRepo{}, defaultSettingsResolver{})
	return NewOrderService(tx, engine, nil, priceRepo, nil, orderRepo, custRepo, subRepo, payRepo, pmRepo, nil, nil, ps, silentLogger{}, coupons, invSvc)
}

func pendingOrder() domain.Order {
	return domain.Order{OrgId: "org_1", Id: "ord_1", CustomerId: "cust_1", Status: domain.OrderStatusPending}
}

func TestOrderService_CompleteOrder_Rejections(t *testing.T) {
	tests := []struct {
		name      string
		order     domain.Order
		orderErr  error
		input     port.CompleteOrderInput
		assertErr func(t *testing.T, err error)
	}{
		{
			name:  "order not found",
			order: pendingOrder(),
			// FindById error wins regardless of other input.
			orderErr: errors.New("missing"),
			input:    port.CompleteOrderInput{OrgId: "org_1", Id: "ord_1", PaymentMethodId: "pm_1"},
		},
		{
			name:  "order not pending",
			order: domain.Order{OrgId: "org_1", Id: "ord_1", Status: domain.OrderStatusCompleted},
			input: port.CompleteOrderInput{OrgId: "org_1", Id: "ord_1", PaymentMethodId: "pm_1"},
		},
		{
			name:  "no payment method provided",
			order: pendingOrder(),
			input: port.CompleteOrderInput{OrgId: "org_1", Id: "ord_1"}, // no PaymentMethodId, no token
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderRepo := &fakeOrderRepo{order: tt.order, findErr: tt.orderErr}
			engine := &recordingEngine{}
			ps := &recordingPubSub{}
			svc := newOrderServiceForTest(&fakeTxManager{}, engine, orderRepo,
				&fakeCustomerRepo{}, &fakeSubRepo{}, &fakePaymentMethodRepo{}, &fakePaymentRepo{}, ps)

			_, err := svc.CompleteOrder(context.Background(), tt.input)

			require.Error(t, err)
			assert.Empty(t, engine.started, "no workflow starts on rejection")
			assert.Empty(t, ps.published, "no order.completed publish on rejection")
		})
	}
}

func TestOrderService_CompleteOrder_HappyPath(t *testing.T) {
	subs := []domain.Subscription{
		{OrgId: "org_1", Id: "sub_1", Status: domain.SubscriptionStatusPending},
		{OrgId: "org_1", Id: "sub_2", Status: domain.SubscriptionStatusPending},
	}

	t.Run("existing payment method, first payment charged, builds+settles one invoice", func(t *testing.T) {
		// Single $50 one-time line so BuildForOrder has something to invoice. With
		// two subscriptions the combined invoice carries no single SubscriptionId,
		// so the one order payment links to the invoice (not a sub).
		orderRepo := &fakeOrderRepo{order: pendingOrder(), items: []domain.OrderItem{
			{OrgId: "org_1", Id: "oi_1", OrderId: "ord_1", PriceId: "price_1", Quantity: 1},
		}}
		priceRepo := &mapPriceRepo{byId: map[string]domain.Price{
			"price_1": {OrgId: "org_1", Id: "price_1", Scheme: domain.Fixed, UnitPrice: 5000},
		}}
		custRepo := &fakeCustomerRepo{paymentMethod: domain.PaymentMethod{Id: "pm_1"}}
		subRepo := &fakeSubRepo{byOrderId: subs}
		payRepo := &fakePaymentRepo{}
		invRepo := newFakeInvoiceRepo()
		engine := &recordingEngine{}
		ps := &recordingPubSub{}
		svc := newOrderServiceWithInvoice(&fakeTxManager{}, engine, orderRepo, custRepo, subRepo, &fakePaymentMethodRepo{}, payRepo, ps, priceRepo, invRepo, nil)

		got, err := svc.CompleteOrder(context.Background(), port.CompleteOrderInput{
			OrgId: "org_1", Id: "ord_1", PaymentMethodId: "pm_1",
			Payment: port.CompleteOrderInputPayment{Amount: 5000, Currency: "USD"},
		})

		require.NoError(t, err)
		assert.Equal(t, domain.OrderStatusCompleted, got.Status)
		require.Len(t, payRepo.created, 1, "exactly one payment for the order")
		require.Len(t, invRepo.byOrder, 1, "exactly one combined invoice")
		inv := invRepo.byOrder["ord_1"]
		assert.Equal(t, inv.Id, payRepo.created[0].InvoiceId, "payment links to the combined invoice")
		assert.Equal(t, domain.InvoiceStatusPaid, invRepo.byId[inv.Id].Status, "invoice settled to paid")
		assert.Len(t, subRepo.updated, 2, "both subscriptions updated")
		assert.Len(t, engine.started, 2, "a workflow started per activated subscription")
		assert.True(t, ps.hasTopic(port.TopicOrderCompleted))
		// The activated subs carry the resolved payment method id and reached cycle 1.
		for _, s := range subRepo.updated {
			assert.Equal(t, "pm_1", s.PaymentMethodId)
			assert.Equal(t, 1, s.CyclesProcessed, "first charge advances the sub to cycle 1")
		}
	})

	t.Run("no first payment means no payment rows but subs still activate", func(t *testing.T) {
		orderRepo := &fakeOrderRepo{order: pendingOrder()}
		custRepo := &fakeCustomerRepo{paymentMethod: domain.PaymentMethod{Id: "pm_1"}}
		subRepo := &fakeSubRepo{byOrderId: subs}
		payRepo := &fakePaymentRepo{}
		engine := &recordingEngine{}
		ps := &recordingPubSub{}
		svc := newOrderServiceForTest(&fakeTxManager{}, engine, orderRepo, custRepo, subRepo, &fakePaymentMethodRepo{}, payRepo, ps)

		_, err := svc.CompleteOrder(context.Background(), port.CompleteOrderInput{
			OrgId: "org_1", Id: "ord_1", PaymentMethodId: "pm_1",
			Payment: port.CompleteOrderInputPayment{Amount: 0},
		})

		require.NoError(t, err)
		assert.Empty(t, payRepo.created, "no payment created when amount is zero")
		assert.Len(t, engine.started, 2)
	})

	t.Run("new payment method via token is created", func(t *testing.T) {
		orderRepo := &fakeOrderRepo{order: pendingOrder()}
		subRepo := &fakeSubRepo{byOrderId: subs[:1]}
		pmRepo := &fakePaymentMethodRepo{}
		engine := &recordingEngine{}
		ps := &recordingPubSub{}
		svc := newOrderServiceForTest(&fakeTxManager{}, engine, orderRepo, &fakeCustomerRepo{}, subRepo, pmRepo, &fakePaymentRepo{}, ps)

		_, err := svc.CompleteOrder(context.Background(), port.CompleteOrderInput{
			OrgId: "org_1", Id: "ord_1",
			PaymentMethod: port.CompleteOrderInputPaymentMethod{
				Token: "tok_visa", Type: domain.PaymentMethodType("card"), Name: "Visa",
			},
		})

		require.NoError(t, err)
		require.Len(t, pmRepo.created, 1, "payment method created from token")
		assert.Equal(t, "tok_visa", pmRepo.created[0].Token.Reveal())
		// The created PM id flows onto the activated subscription.
		require.Len(t, subRepo.updated, 1)
		assert.Equal(t, pmRepo.created[0].Id, subRepo.updated[0].PaymentMethodId)
		assert.Len(t, engine.started, 1)
	})
}

// CompleteOrder builds exactly ONE combined invoice for a mixed order (a
// $100/mo subscription + a $50 one-time line), links the single order payment
// to it, settles it to paid, and advances the subscription to cycle 1.
func TestOrderService_CompleteOrder_MixedInvoice(t *testing.T) {
	const orgId = "org_1"
	plan := domain.Price{
		OrgId: orgId, Id: "price_plan", Category: domain.PriceCategorySubscription,
		Scheme: domain.Fixed, UnitPrice: 10000, Currency: domain.USD,
		BillingInterval: domain.BillingIntervalMonth, BillingIntervalQty: 1,
	}
	oneTime := domain.Price{OrgId: orgId, Id: "price_setup", Scheme: domain.Fixed, UnitPrice: 5000, Currency: domain.USD}
	priceRepo := &mapPriceRepo{byId: map[string]domain.Price{"price_plan": plan, "price_setup": oneTime}}

	orderRepo := &fakeOrderRepo{order: pendingOrder(), items: []domain.OrderItem{
		{OrgId: orgId, Id: "oi_plan", OrderId: "ord_1", ProductId: "prod_a", PriceId: "price_plan", Quantity: 1},
		{OrgId: orgId, Id: "oi_setup", OrderId: "ord_1", ProductId: "prod_b", PriceId: "price_setup", Quantity: 1},
	}}
	// Exactly one subscription so the combined invoice IS its cycle-0 invoice.
	sub := domain.NewSubscriptionFromLines(orgId, "ord_1", "cust_1", []domain.Price{plan})
	subRepo := &fakeSubRepo{byOrderId: []domain.Subscription{sub}}
	custRepo := &fakeCustomerRepo{paymentMethod: domain.PaymentMethod{Id: "pm_1"}}
	payRepo := &fakePaymentRepo{}
	invRepo := newFakeInvoiceRepo()
	engine := &recordingEngine{}
	ps := &recordingPubSub{}
	svc := newOrderServiceWithInvoice(&fakeTxManager{}, engine, orderRepo, custRepo, subRepo, &fakePaymentMethodRepo{}, payRepo, ps, priceRepo, invRepo, nil)

	_, err := svc.CompleteOrder(context.Background(), port.CompleteOrderInput{
		OrgId: orgId, Id: "ord_1", PaymentMethodId: "pm_1",
		Payment: port.CompleteOrderInputPayment{Amount: 15000, Currency: "USD"},
	})
	require.NoError(t, err)

	require.Len(t, invRepo.byOrder, 1, "exactly one combined invoice")
	inv := invRepo.byOrder["ord_1"]
	assert.EqualValues(t, 15000, inv.Total, "100/mo + 50 one-time = 15000c")
	assert.Equal(t, domain.InvoiceStatusPaid, invRepo.byId[inv.Id].Status)
	require.Len(t, payRepo.created, 1, "one order payment")
	assert.Equal(t, inv.Id, payRepo.created[0].InvoiceId)
	assert.Equal(t, sub.Id, payRepo.created[0].SubscriptionId, "single-sub order links the payment to the sub")
	require.Len(t, subRepo.updated, 1)
	assert.Equal(t, 1, subRepo.updated[0].CyclesProcessed, "subscription advanced to cycle 1")
}

// InitOrderPayment initialises the PSP session once and is idempotent: a second
// call returns the stored session without a second gateway call.
func TestOrderService_InitOrderPayment(t *testing.T) {
	newSvc := func(order domain.Order, gw *initPaymentGateway) (*OrderService, *fakeOrderRepo) {
		orderRepo := &fakeOrderRepo{order: order}
		factory := &initPaymentFactory{gw: gw}
		svc := NewOrderService(nil, nil, nil, &fakePriceRepo{}, &fakeCartRepo{}, orderRepo,
			&fakeCustomerRepo{}, nil, nil, nil, nil, factory, &recordingPubSub{}, silentLogger{}, nil, nil)
		return svc, orderRepo
	}

	t.Run("initialises once then returns the stored session on retry", func(t *testing.T) {
		session := map[string]any{"reference": "ps_1", "url": "https://pay/x"}
		gw := &initPaymentGateway{resp: port.InitPaymentResponse{PspResponse: session}}
		svc, orderRepo := newSvc(pendingOrder(), gw)

		resp, err := svc.InitOrderPayment(context.Background(), "org_1", "ord_1", "paystack", nil)
		require.NoError(t, err)
		assert.Equal(t, session, resp.PspResponse)
		assert.Equal(t, 1, gw.calls, "gateway called once")
		assert.Equal(t, session, orderRepo.order.PaymentSession, "session persisted on the order")

		// Second call: the order now has a stored session, so no gateway call.
		resp2, err := svc.InitOrderPayment(context.Background(), "org_1", "ord_1", "paystack", nil)
		require.NoError(t, err)
		assert.Equal(t, session, resp2.PspResponse)
		assert.Equal(t, 1, gw.calls, "no second gateway call when a session already exists")
	})

	t.Run("rejects a non-pending order with a conflict", func(t *testing.T) {
		order := pendingOrder()
		order.Status = domain.OrderStatusCompleted
		gw := &initPaymentGateway{}
		svc, _ := newSvc(order, gw)

		_, err := svc.InitOrderPayment(context.Background(), "org_1", "ord_1", "paystack", nil)
		require.Error(t, err)
		var ce lib.CustomError
		require.ErrorAs(t, err, &ce)
		assert.Equal(t, lib.ConflictError, ce.Type)
		assert.Equal(t, 0, gw.calls, "no gateway call for a non-pending order")
	})
}

// CreateOrder must refuse to sell an archived product. The guard runs after the
// cart is assembled and before the customer/order is created, so a minimal
// service (no engine/session/customer repos) is enough to exercise it via the
// direct cart-items path.
func TestOrderService_CreateOrder_RejectsArchivedProduct(t *testing.T) {
	prod := &fakeProductRepo{byId: domain.Product{OrgId: "org_1", Id: "prod_1", Status: domain.ProductStatusArchived}}
	price := &fakePriceRepo{byId: domain.Price{OrgId: "org_1", Id: "price_1", UnitPrice: 1000}}
	orderRepo := &fakeOrderRepo{}
	svc := NewOrderService(nil, nil, nil, price, &fakeCartRepo{}, orderRepo, &fakeCustomerRepo{}, nil, nil, nil, prod, nil, &recordingPubSub{}, silentLogger{}, nil, nil)

	_, err := svc.CreateOrder(context.Background(), port.CreateOrderInput{
		OrgId:     "org_1",
		CartItems: []domain.CartItem{{ProductId: "prod_1", PriceId: "price_1", Quantity: 1}},
	})

	require.Error(t, err)
	var ce lib.CustomError
	require.ErrorAs(t, err, &ce)
	assert.Equal(t, lib.ConflictError, ce.Type)
	assert.Empty(t, orderRepo.updated, "archived product must be rejected before the order is created")
}

// CreateOrder persists the order's Config and, when upfront_invoice is set,
// builds the combined invoice open at create time and returns it. A one-time
// (non-recurring) price keeps the order free of subscriptions, so the invoice is
// a pure one-time order invoice.
func TestOrderService_CreateOrder_UpfrontInvoice(t *testing.T) {
	newSvc := func() (*OrderService, *fakeOrderRepo, *fakeInvoiceRepo) {
		prod := &fakeProductRepo{byId: domain.Product{OrgId: "org_1", Id: "prod_1", Name: "Plan", Status: domain.ProductStatusActive}}
		// one-time price: no billing interval, no metric → not recurring.
		price := &fakePriceRepo{byId: domain.Price{OrgId: "org_1", Id: "price_1", UnitPrice: 1000, UnitCount: 1}}
		orderRepo := &fakeOrderRepo{}
		subRepo := &fakeSubRepo{}
		custRepo := &fakeCustomerRepo{customer: domain.Customer{OrgId: "org_1", Id: "cust_1"}}
		invRepo := newFakeInvoiceRepo()
		invSvc := NewInvoiceService(invRepo, orderRepo, price, subRepo, noopUsage{}, noopTx{}, silentLogger{}, noopDiscountRepo{}, noopCouponRepo{}, noopReservationRepo{}, defaultSettingsResolver{})
		svc := NewOrderService(
			nil, &recordingEngine{}, nil, price, &fakeCartRepo{}, orderRepo,
			custRepo, subRepo, nil, nil, prod, nil, &recordingPubSub{}, silentLogger{}, nil, invSvc,
		)
		return svc, orderRepo, invRepo
	}

	input := func(upfront bool) port.CreateOrderInput {
		return port.CreateOrderInput{
			OrgId:     "org_1",
			Currency:  "USD",
			Customer:  port.CreateOrderInputCustomer{Id: "cust_1"},
			CartItems: []domain.CartItem{{ProductId: "prod_1", PriceId: "price_1", Quantity: 1}},
			Config:    domain.OrderConfig{UpfrontInvoice: upfront},
		}
	}

	t.Run("upfront_invoice true returns an open invoice", func(t *testing.T) {
		svc, orderRepo, _ := newSvc()

		res, err := svc.CreateOrder(context.Background(), input(true))
		require.NoError(t, err)
		require.NotNil(t, res.Invoice, "upfront order must return an invoice")
		assert.Equal(t, domain.InvoiceStatusOpen, res.Invoice.Status)
		assert.Equal(t, res.Order.Id, res.Invoice.OrderId)
		require.Len(t, orderRepo.created, 1)
		assert.True(t, orderRepo.created[0].Config.UpfrontInvoice, "Config.UpfrontInvoice persisted on the order")
	})

	t.Run("upfront_invoice false returns no invoice", func(t *testing.T) {
		svc, orderRepo, _ := newSvc()

		res, err := svc.CreateOrder(context.Background(), input(false))
		require.NoError(t, err)
		assert.Nil(t, res.Invoice, "non-upfront order must not return an invoice")
		require.Len(t, orderRepo.created, 1)
		assert.False(t, orderRepo.created[0].Config.UpfrontInvoice, "Config.UpfrontInvoice persisted as false")
	})
}
