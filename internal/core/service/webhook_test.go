package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// fakeIdempotencyRepo records idempotency-key writes and answers Exists.
type fakeIdempotencyRepo struct {
	exists    bool
	existsErr error
	createErr error
	createdN  int
}

func (r *fakeIdempotencyRepo) Exists(_ context.Context, _ string) (bool, error) {
	return r.exists, r.existsErr
}
func (r *fakeIdempotencyRepo) Create(_ context.Context, _ string, _ time.Time) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.createdN++
	return nil
}

// webhookEngine counts workflow starts (by type) and subscription signals so
// the dispatch switch in HandlePaymentWebhook can be asserted. It spawns no
// goroutines, keeping the package goleak-clean.
type webhookEngine struct {
	startedWorkflows map[port.WorkflowType]int
	signalledSubs    int
}

func newWebhookEngine() *webhookEngine {
	return &webhookEngine{startedWorkflows: map[port.WorkflowType]int{}}
}

func (e *webhookEngine) StartWorkflow(_ context.Context, id port.WorkflowType, _ any) (port.WorkflowResult, error) {
	e.startedWorkflows[id]++
	return port.WorkflowResult{}, nil
}
func (e *webhookEngine) StartSubscriptionWorkflow(context.Context, domain.Subscription) error { return nil }
func (e *webhookEngine) UpdateSubscriptionWorkflow(context.Context, string, domain.Subscription) error {
	return nil
}
func (e *webhookEngine) CancelSubscriptionWorkflow(context.Context, domain.Subscription) error {
	return nil
}
func (e *webhookEngine) SignalSubscriptionWorkflow(context.Context, string, domain.Subscription, any) error {
	e.signalledSubs++
	return nil
}

// stubParser is a configurable domain.WebhookParser used to drive the dispatch
// switch in HandlePaymentWebhook.
type stubParser struct {
	validateErr error
	parseErr    error
	parsed      domain.PaymentWebhookContext
}

func (p stubParser) ValidateWebhook(context.Context, []byte) error { return p.validateErr }
func (p stubParser) ParseWebhook(context.Context, []byte) (domain.PaymentWebhookContext, error) {
	if p.parseErr != nil {
		return domain.PaymentWebhookContext{}, p.parseErr
	}
	return p.parsed, nil
}

// factoryWithParser builds a real GatewayFactory whose Paystack adapter returns
// the given parser (the factory dependency on WebhookService is a concrete type).
func factoryWithParser(parser domain.WebhookParser) *GatewayFactory {
	return NewGatewayFactory(&fakePspRepo{}, &fakeSettingRepoRW{}, silentLogger{}, map[domain.Gateway]port.GatewayAdapter{
		domain.Paystack: &fakeGatewayAdapter{webhookParser: parser},
	})
}

func newWebhookService(factory *GatewayFactory, engine port.Engine, idem port.IdempotencyKeyRepository, subs port.SubscriptionRepository) *WebhookService {
	return NewWebhookService(silentLogger{}, factory, engine, idem, subs)
}

func TestWebhookService_HandlePaymentWebhook(t *testing.T) {
	t.Run("already-processed webhook short-circuits without dispatch", func(t *testing.T) {
		idem := &fakeIdempotencyRepo{exists: true}
		engine := newWebhookEngine()
		svc := newWebhookService(factoryWithParser(stubParser{}), engine, idem, &fakeSubRepo{})

		err := svc.HandlePaymentWebhook(context.Background(), port.PaymentWebhookPayload{Psp: domain.Paystack, Data: "{}"})

		require.NoError(t, err)
		assert.Equal(t, 0, idem.createdN, "no key written on a duplicate")
	})

	t.Run("idempotency check failure is surfaced", func(t *testing.T) {
		idem := &fakeIdempotencyRepo{existsErr: errors.New("redis down")}
		svc := newWebhookService(factoryWithParser(stubParser{}), newWebhookEngine(), idem, &fakeSubRepo{})

		err := svc.HandlePaymentWebhook(context.Background(), port.PaymentWebhookPayload{Psp: domain.Paystack, Data: "{}"})
		require.Error(t, err)
	})

	t.Run("validation failure aborts before storing the key", func(t *testing.T) {
		idem := &fakeIdempotencyRepo{}
		parser := stubParser{validateErr: errors.New("bad signature")}
		svc := newWebhookService(factoryWithParser(parser), newWebhookEngine(), idem, &fakeSubRepo{})

		err := svc.HandlePaymentWebhook(context.Background(), port.PaymentWebhookPayload{Psp: domain.Paystack, Data: "{}"})

		require.Error(t, err)
		assert.Equal(t, 0, idem.createdN)
	})

	t.Run("payment.success starts the payment-success workflow and stores the key", func(t *testing.T) {
		idem := &fakeIdempotencyRepo{}
		parser := stubParser{parsed: domain.PaymentWebhookContext{Type: domain.PaymentSuccess, OrgId: "org_1", OrderId: "ord_1"}}
		engine := newWebhookEngine()
		svc := newWebhookService(factoryWithParser(parser), engine, idem, &fakeSubRepo{})

		err := svc.HandlePaymentWebhook(context.Background(), port.PaymentWebhookPayload{Psp: domain.Paystack, Data: "{}"})

		require.NoError(t, err)
		assert.Equal(t, 1, engine.startedWorkflows[port.WorkflowPaymentSuccess], "payment-success workflow started")
		assert.Equal(t, 1, idem.createdN, "key stored after successful processing")
	})

	t.Run("recurring.success signals the subscription workflow for the order's first sub", func(t *testing.T) {
		idem := &fakeIdempotencyRepo{}
		parser := stubParser{parsed: domain.PaymentWebhookContext{Type: domain.RecurringSuccess, OrgId: "org_1", OrderId: "ord_1"}}
		engine := newWebhookEngine()
		subs := &fakeSubRepo{byOrderId: []domain.Subscription{{OrgId: "org_1", Id: "sub_1"}}}
		svc := newWebhookService(factoryWithParser(parser), engine, idem, subs)

		err := svc.HandlePaymentWebhook(context.Background(), port.PaymentWebhookPayload{Psp: domain.Paystack, Data: "{}"})

		require.NoError(t, err)
		assert.Equal(t, 1, engine.signalledSubs, "subscription workflow signalled")
		assert.Equal(t, 1, idem.createdN)
	})

	t.Run("recurring.success with no subscriptions is a no-op success without storing a key", func(t *testing.T) {
		idem := &fakeIdempotencyRepo{}
		parser := stubParser{parsed: domain.PaymentWebhookContext{Type: domain.RecurringSuccess, OrgId: "org_1", OrderId: "ord_1"}}
		engine := newWebhookEngine()
		svc := newWebhookService(factoryWithParser(parser), engine, idem, &fakeSubRepo{})

		err := svc.HandlePaymentWebhook(context.Background(), port.PaymentWebhookPayload{Psp: domain.Paystack, Data: "{}"})

		require.NoError(t, err)
		assert.Equal(t, 0, engine.signalledSubs)
		assert.Equal(t, 0, idem.createdN, "early return before key storage")
	})

	t.Run("payment.refunded starts the refund workflow", func(t *testing.T) {
		idem := &fakeIdempotencyRepo{}
		parser := stubParser{parsed: domain.PaymentWebhookContext{Type: domain.PaymentRefunded, OrgId: "org_1", OrderId: "ord_1"}}
		engine := newWebhookEngine()
		svc := newWebhookService(factoryWithParser(parser), engine, idem, &fakeSubRepo{})

		err := svc.HandlePaymentWebhook(context.Background(), port.PaymentWebhookPayload{Psp: domain.Paystack, Data: "{}"})

		require.NoError(t, err)
		assert.Equal(t, 1, engine.startedWorkflows[port.WorkflowPaymentRefunded])
		assert.Equal(t, 1, idem.createdN)
	})
}

func TestWebhookService_HandleAuthnWebhook(t *testing.T) {
	svc := newWebhookService(factoryWithParser(stubParser{}), newWebhookEngine(), &fakeIdempotencyRepo{}, &fakeSubRepo{})
	err := svc.HandleAuthnWebhook(context.Background(), port.AuthnWebhookPayload{Provider: "clerk", Data: "{}"})
	require.NoError(t, err)
}
