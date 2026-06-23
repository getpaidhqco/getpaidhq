package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

func newOrderWorkflowService(
	orderRepo port.OrderRepository,
	subRepo port.SubscriptionRepository,
	pmRepo port.PaymentMethodRepository,
	payRepo port.PaymentRepository,
	ps port.PubSub,
) *OrderWorkflowService {
	if ps == nil {
		ps = &recordingPubSub{}
	}
	return NewOrderWorkflowService(orderRepo, &fakeCustomerRepo{}, subRepo, pmRepo, payRepo, &fakePriceRepo{}, &fakeTxManager{}, ps, silentLogger{})
}

func completeSessionInput() port.CompleteCheckoutSessionInput {
	return port.CompleteCheckoutSessionInput{
		OrgId:   "org_1",
		OrderId: "ord_1",
		PaymentContext: domain.PaymentWebhookContext{
			OrgId:         "org_1",
			Psp:           domain.Paystack,
			Payment:       domain.GatewayPayment{Amount: 5000, Currency: "USD", PspId: "psp_pay_1", Reference: "ref_1"},
			PaymentMethod: domain.GatewayPaymentMethod{Token: "tok_1", Type: "card"},
		},
	}
}

func TestOrderWorkflowService_CompleteCheckoutSession(t *testing.T) {
	t.Run("paid session: order completed, PM created, sub activated, payment row, event", func(t *testing.T) {
		orderRepo := &fakeOrderRepo{order: domain.Order{OrgId: "org_1", Id: "ord_1", CustomerId: "cus_1", Status: domain.OrderStatusPending}}
		subRepo := &fakeSubRepo{byOrderId: []domain.Subscription{{OrgId: "org_1", Id: "sub_1"}}}
		pmRepo := &fakePaymentMethodRepo{}
		payRepo := &fakePaymentRepo{}
		ps := &recordingPubSub{}
		svc := newOrderWorkflowService(orderRepo, subRepo, pmRepo, payRepo, ps)

		got, err := svc.CompleteCheckoutSession(context.Background(), completeSessionInput())

		require.NoError(t, err)
		assert.Equal(t, domain.OrderStatusCompleted, got.Status)
		assert.Equal(t, 1, orderRepo.forUpdateHit)
		require.Len(t, pmRepo.created, 1)
		assert.Equal(t, "tok_1", pmRepo.created[0].Token.Reveal())
		// Details must not smuggle the token alongside the redacting Token field.
		detailsJSON, _ := json.Marshal(pmRepo.created[0].Details)
		assert.NotContains(t, string(detailsJSON), "tok_1")
		require.Len(t, subRepo.updated, 1)
		assert.Equal(t, domain.SubscriptionStatusActive, subRepo.updated[0].Status)
		assert.Equal(t, pmRepo.created[0].Id, subRepo.updated[0].PaymentMethodId)
		require.Len(t, payRepo.created, 1)
		assert.True(t, payRepo.created[0].Recurring, "first recurring charge")
		assert.True(t, ps.hasTopic(port.TopicOrderCompleted))
	})

	t.Run("zero-amount session activates subs to trial without a payment row", func(t *testing.T) {
		orderRepo := &fakeOrderRepo{order: domain.Order{OrgId: "org_1", Id: "ord_1", CustomerId: "cus_1", Status: domain.OrderStatusPending}}
		subRepo := &fakeSubRepo{byOrderId: []domain.Subscription{{OrgId: "org_1", Id: "sub_1"}}}
		payRepo := &fakePaymentRepo{}
		ps := &recordingPubSub{}
		input := completeSessionInput()
		input.PaymentContext.Payment.Amount = 0
		svc := newOrderWorkflowService(orderRepo, subRepo, &fakePaymentMethodRepo{}, payRepo, ps)

		_, err := svc.CompleteCheckoutSession(context.Background(), input)

		require.NoError(t, err)
		require.Len(t, subRepo.updated, 1)
		assert.Equal(t, domain.SubscriptionStatusTrial, subRepo.updated[0].Status)
		assert.Empty(t, payRepo.created, "no payment row when amount is zero")
		assert.True(t, ps.hasTopic(port.TopicOrderCompleted))
	})

	t.Run("missing order is rejected before any mutation", func(t *testing.T) {
		orderRepo := &fakeOrderRepo{findErr: errors.New("not found")}
		pmRepo := &fakePaymentMethodRepo{}
		ps := &recordingPubSub{}
		svc := newOrderWorkflowService(orderRepo, &fakeSubRepo{}, pmRepo, &fakePaymentRepo{}, ps)

		_, err := svc.CompleteCheckoutSession(context.Background(), completeSessionInput())

		require.Error(t, err)
		assert.Empty(t, pmRepo.created)
		assert.Empty(t, ps.published)
	})

	t.Run("already completed order is an idempotent no-op", func(t *testing.T) {
		orderRepo := &fakeOrderRepo{order: domain.Order{OrgId: "org_1", Id: "ord_1", CustomerId: "cus_1", Status: domain.OrderStatusCompleted}}
		pmRepo := &fakePaymentMethodRepo{}
		payRepo := &fakePaymentRepo{}
		ps := &recordingPubSub{}
		svc := newOrderWorkflowService(orderRepo, &fakeSubRepo{byOrderId: []domain.Subscription{{OrgId: "org_1", Id: "sub_1"}}}, pmRepo, payRepo, ps)

		got, err := svc.CompleteCheckoutSession(context.Background(), completeSessionInput())

		require.NoError(t, err)
		assert.Equal(t, domain.OrderStatusCompleted, got.Status)
		assert.Equal(t, 1, orderRepo.forUpdateHit)
		assert.Empty(t, orderRepo.updated)
		assert.Empty(t, pmRepo.created)
		assert.Empty(t, payRepo.created)
		assert.Empty(t, ps.published)
	})

	t.Run("payment create failure returns error and prevents publish", func(t *testing.T) {
		orderRepo := &fakeOrderRepo{order: domain.Order{OrgId: "org_1", Id: "ord_1", CustomerId: "cus_1", Status: domain.OrderStatusPending}}
		subRepo := &fakeSubRepo{byOrderId: []domain.Subscription{{OrgId: "org_1", Id: "sub_1"}}}
		pmRepo := &fakePaymentMethodRepo{}
		payRepo := &fakePaymentRepo{createErr: errors.New("db down")}
		ps := &recordingPubSub{}
		svc := newOrderWorkflowService(orderRepo, subRepo, pmRepo, payRepo, ps)

		_, err := svc.CompleteCheckoutSession(context.Background(), completeSessionInput())

		require.Error(t, err)
		assert.Equal(t, 1, orderRepo.forUpdateHit)
		assert.Empty(t, ps.published)
	})
}
