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
	order   domain.Order
	items   []domain.OrderItem
	findErr error
	updated []domain.Order
}

func (r *fakeOrderRepo) FindById(_ context.Context, _, _ string) (domain.Order, error) {
	if r.findErr != nil {
		return domain.Order{}, r.findErr
	}
	return r.order, nil
}

func (r *fakeOrderRepo) FindOrderItemsByOrderId(_ context.Context, _, _ string) ([]domain.OrderItem, error) {
	return r.items, nil
}

func (r *fakeOrderRepo) Update(_ context.Context, o domain.Order) (domain.Order, error) {
	r.updated = append(r.updated, o)
	r.order = o
	return o, nil
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

func (r *fakeCustomerRepo) FindPaymentMethodById(_ context.Context, _, _ string) (domain.PaymentMethod, error) {
	if r.pmErr != nil {
		return domain.PaymentMethod{}, r.pmErr
	}
	return r.paymentMethod, nil
}

type fakePaymentMethodRepo struct {
	port.PaymentMethodRepository
	created []domain.PaymentMethod
}

func (r *fakePaymentMethodRepo) Create(_ context.Context, pm domain.PaymentMethod) (domain.PaymentMethod, error) {
	r.created = append(r.created, pm)
	return pm, nil
}

type fakePaymentRepo struct {
	port.PaymentRepository
	created []domain.Payment
}

func (r *fakePaymentRepo) Create(_ context.Context, p domain.Payment) (domain.Payment, error) {
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
	return NewOrderService(tx, engine, nil, nil, nil, orderRepo, custRepo, subRepo, payRepo, pmRepo, nil, nil, ps, silentLogger{})
}

func pendingOrder() domain.Order {
	return domain.Order{OrgId: "org_1", Id: "ord_1", CustomerId: "cust_1", Status: domain.OrderStatusPending}
}

func TestOrderService_CompleteOrder_Rejections(t *testing.T) {
	tests := []struct {
		name      string
		order     domain.Order
		orderErr  error
		input     domain.CompleteOrderInput
		assertErr func(t *testing.T, err error)
	}{
		{
			name:  "order not found",
			order: pendingOrder(),
			// FindById error wins regardless of other input.
			orderErr: errors.New("missing"),
			input:    domain.CompleteOrderInput{OrgId: "org_1", Id: "ord_1", PaymentMethodId: "pm_1"},
		},
		{
			name:  "order not pending",
			order: domain.Order{OrgId: "org_1", Id: "ord_1", Status: domain.OrderStatusCompleted},
			input: domain.CompleteOrderInput{OrgId: "org_1", Id: "ord_1", PaymentMethodId: "pm_1"},
		},
		{
			name:  "no payment method provided",
			order: pendingOrder(),
			input: domain.CompleteOrderInput{OrgId: "org_1", Id: "ord_1"}, // no PaymentMethodId, no token
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

	t.Run("existing payment method, first payment charged, activates all subs", func(t *testing.T) {
		orderRepo := &fakeOrderRepo{order: pendingOrder()}
		custRepo := &fakeCustomerRepo{paymentMethod: domain.PaymentMethod{Id: "pm_1"}}
		subRepo := &fakeSubRepo{byOrderId: subs}
		payRepo := &fakePaymentRepo{}
		engine := &recordingEngine{}
		ps := &recordingPubSub{}
		svc := newOrderServiceForTest(&fakeTxManager{}, engine, orderRepo, custRepo, subRepo, &fakePaymentMethodRepo{}, payRepo, ps)

		got, err := svc.CompleteOrder(context.Background(), domain.CompleteOrderInput{
			OrgId: "org_1", Id: "ord_1", PaymentMethodId: "pm_1",
			Payment: domain.CompleteOrderInputPayment{Amount: 5000, Currency: "USD"},
		})

		require.NoError(t, err)
		assert.Equal(t, domain.OrderStatusCompleted, got.Status)
		assert.Len(t, payRepo.created, 2, "one payment per activated subscription")
		assert.Len(t, subRepo.updated, 2, "both subscriptions updated")
		assert.Len(t, engine.started, 2, "a workflow started per activated subscription")
		assert.True(t, ps.hasTopic(port.TopicOrderCompleted))
		// The activated subs carry the resolved payment method id.
		for _, s := range subRepo.updated {
			assert.Equal(t, "pm_1", s.PaymentMethodId)
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

		_, err := svc.CompleteOrder(context.Background(), domain.CompleteOrderInput{
			OrgId: "org_1", Id: "ord_1", PaymentMethodId: "pm_1",
			Payment: domain.CompleteOrderInputPayment{Amount: 0},
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

		_, err := svc.CompleteOrder(context.Background(), domain.CompleteOrderInput{
			OrgId: "org_1", Id: "ord_1",
			PaymentMethod: domain.CompleteOrderInputPaymentMethod{
				Token: "tok_visa", Type: domain.PaymentMethodType("card"), Name: "Visa",
			},
		})

		require.NoError(t, err)
		require.Len(t, pmRepo.created, 1, "payment method created from token")
		assert.Equal(t, "tok_visa", pmRepo.created[0].Token)
		// The created PM id flows onto the activated subscription.
		require.Len(t, subRepo.updated, 1)
		assert.Equal(t, pmRepo.created[0].Id, subRepo.updated[0].PaymentMethodId)
		assert.Len(t, engine.started, 1)
	})
}
