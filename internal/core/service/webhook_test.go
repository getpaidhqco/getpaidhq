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

// fakeIdempotencyRepo models the Claim/Release atomic pair the new
// repository contract exposes. `claimAvailable` controls whether the
// first Claim wins (`true` — caller proceeds) or loses to a sibling
// delivery (`false` — caller short-circuits).
type fakeIdempotencyRepo struct {
	claimAvailable bool
	claimErr       error
	releaseErr     error

	claims     int
	released   int
	releaseKey string
}

func (r *fakeIdempotencyRepo) Claim(_ context.Context, _ string, _ time.Time) (bool, error) {
	r.claims++
	if r.claimErr != nil {
		return false, r.claimErr
	}
	return r.claimAvailable, nil
}

func (r *fakeIdempotencyRepo) Release(_ context.Context, key string) error {
	r.released++
	r.releaseKey = key
	return r.releaseErr
}

// webhookEngine counts workflow starts (by type) and subscription signals so
// the dispatch switch in HandlePaymentWebhook can be asserted. It spawns no
// goroutines, keeping the package goleak-clean.
type webhookEngine struct {
	startedWorkflows map[port.WorkflowType]int
	signalledSubs    int

	// Failure injection — when set, the corresponding method returns
	// this error to drive the release-on-failure paths.
	startWorkflowErr error
	signalErr        error
}

func newWebhookEngine() *webhookEngine {
	return &webhookEngine{startedWorkflows: map[port.WorkflowType]int{}}
}

func (e *webhookEngine) StartWorkflow(_ context.Context, id port.WorkflowType, _ any) (port.WorkflowResult, error) {
	e.startedWorkflows[id]++
	return port.WorkflowResult{}, e.startWorkflowErr
}
func (e *webhookEngine) StartSubscriptionWorkflow(context.Context, domain.Subscription) error {
	return nil
}
func (e *webhookEngine) UpdateSubscriptionWorkflow(context.Context, string, domain.Subscription) error {
	return nil
}
func (e *webhookEngine) CancelSubscriptionWorkflow(context.Context, domain.Subscription) error {
	return nil
}
func (e *webhookEngine) SignalSubscriptionWorkflow(context.Context, string, domain.Subscription, any) error {
	e.signalledSubs++
	return e.signalErr
}

// stubParser is a configurable domain.WebhookParser used to drive the dispatch
// switch in HandlePaymentWebhook.
type stubParser struct {
	validateErr error
	parseErr    error
	parsed      domain.PaymentWebhookContext
}

func (p stubParser) ValidateWebhook(context.Context, []byte, string) error { return p.validateErr }
func (p stubParser) ParseWebhook(context.Context, []byte) (domain.PaymentWebhookContext, error) {
	if p.parseErr != nil {
		return domain.PaymentWebhookContext{}, p.parseErr
	}
	return p.parsed, nil
}

// factoryWithParser builds a real GatewayFactory whose Paystack adapter returns
// the given parser (the factory dependency on WebhookService is a concrete type).
func factoryWithParser(parser domain.WebhookParser) *GatewayFactory {
	return NewGatewayFactory(&fakePspRepo{}, &fakeSecretCipher{}, silentLogger{}, map[domain.Gateway]port.GatewayAdapter{
		domain.Paystack: &fakeGatewayAdapter{webhookParser: parser},
	})
}

func newWebhookService(factory *GatewayFactory, engine port.Engine, idem port.IdempotencyKeyRepository, subs port.SubscriptionRepository) *WebhookService {
	return NewWebhookService(silentLogger{}, factory, engine, idem, subs)
}

func TestWebhookService_HandlePaymentWebhook(t *testing.T) {
	t.Run("duplicate delivery (claim returns false) short-circuits without dispatch", func(t *testing.T) {
		idem := &fakeIdempotencyRepo{claimAvailable: false}
		engine := newWebhookEngine()
		svc := newWebhookService(factoryWithParser(stubParser{}), engine, idem, &fakeSubRepo{})

		err := svc.HandlePaymentWebhook(context.Background(), port.PaymentWebhookPayload{Psp: domain.Paystack, Data: "{}"})

		require.NoError(t, err)
		assert.Equal(t, 1, idem.claims, "claim attempted exactly once")
		assert.Equal(t, 0, idem.released, "duplicate doesn't release — the prior delivery owns the claim")
		assert.Equal(t, 0, len(engine.startedWorkflows), "no dispatch on duplicate")
	})

	t.Run("idempotency claim error is surfaced", func(t *testing.T) {
		idem := &fakeIdempotencyRepo{claimErr: errors.New("redis down")}
		svc := newWebhookService(factoryWithParser(stubParser{}), newWebhookEngine(), idem, &fakeSubRepo{})

		err := svc.HandlePaymentWebhook(context.Background(), port.PaymentWebhookPayload{Psp: domain.Paystack, Data: "{}"})

		require.Error(t, err)
	})

	t.Run("validation failure releases the claim so the PSP retry can re-run", func(t *testing.T) {
		idem := &fakeIdempotencyRepo{claimAvailable: true}
		parser := stubParser{validateErr: errors.New("bad signature")}
		svc := newWebhookService(factoryWithParser(parser), newWebhookEngine(), idem, &fakeSubRepo{})

		err := svc.HandlePaymentWebhook(context.Background(), port.PaymentWebhookPayload{Psp: domain.Paystack, Data: "{}"})

		require.Error(t, err)
		assert.Equal(t, 1, idem.released, "failed claim must be released for PSP retry to work")
	})

	t.Run("parse failure releases the claim", func(t *testing.T) {
		idem := &fakeIdempotencyRepo{claimAvailable: true}
		parser := stubParser{parseErr: errors.New("malformed json")}
		svc := newWebhookService(factoryWithParser(parser), newWebhookEngine(), idem, &fakeSubRepo{})

		err := svc.HandlePaymentWebhook(context.Background(), port.PaymentWebhookPayload{Psp: domain.Paystack, Data: "{}"})

		require.Error(t, err)
		assert.Equal(t, 1, idem.released)
	})

	t.Run("payment.success starts the payment-success workflow and keeps the claim", func(t *testing.T) {
		idem := &fakeIdempotencyRepo{claimAvailable: true}
		parser := stubParser{parsed: domain.PaymentWebhookContext{Type: domain.PaymentSuccess, OrgId: "org_1", OrderId: "ord_1"}}
		engine := newWebhookEngine()
		svc := newWebhookService(factoryWithParser(parser), engine, idem, &fakeSubRepo{})

		err := svc.HandlePaymentWebhook(context.Background(), port.PaymentWebhookPayload{Psp: domain.Paystack, Data: "{}"})

		require.NoError(t, err)
		assert.Equal(t, 1, engine.startedWorkflows[port.WorkflowPaymentSuccess], "payment-success workflow started")
		assert.Equal(t, 0, idem.released, "successful processing keeps the claim — duplicates must short-circuit")
	})

	t.Run("payment.success engine failure surfaces the error AND releases the claim", func(t *testing.T) {
		idem := &fakeIdempotencyRepo{claimAvailable: true}
		parser := stubParser{parsed: domain.PaymentWebhookContext{Type: domain.PaymentSuccess, OrgId: "org_1", OrderId: "ord_1"}}
		engine := newWebhookEngine()
		engine.startWorkflowErr = errors.New("hatchet unreachable")
		svc := newWebhookService(factoryWithParser(parser), engine, idem, &fakeSubRepo{})

		err := svc.HandlePaymentWebhook(context.Background(), port.PaymentWebhookPayload{Psp: domain.Paystack, Data: "{}"})

		require.Error(t, err, "engine failure must propagate to the PSP as a retryable error")
		assert.Equal(t, 1, idem.released, "engine failure releases the claim so the PSP retry can re-run")
	})

	t.Run("recurring.success signals the subscription workflow for the order's first sub", func(t *testing.T) {
		idem := &fakeIdempotencyRepo{claimAvailable: true}
		parser := stubParser{parsed: domain.PaymentWebhookContext{Type: domain.RecurringSuccess, OrgId: "org_1", OrderId: "ord_1"}}
		engine := newWebhookEngine()
		subs := &fakeSubRepo{byOrderId: []domain.Subscription{{OrgId: "org_1", Id: "sub_1"}}}
		svc := newWebhookService(factoryWithParser(parser), engine, idem, subs)

		err := svc.HandlePaymentWebhook(context.Background(), port.PaymentWebhookPayload{Psp: domain.Paystack, Data: "{}"})

		require.NoError(t, err)
		assert.Equal(t, 1, engine.signalledSubs, "subscription workflow signalled")
		assert.Equal(t, 0, idem.released)
	})

	t.Run("recurring.success signal failure releases the claim", func(t *testing.T) {
		idem := &fakeIdempotencyRepo{claimAvailable: true}
		parser := stubParser{parsed: domain.PaymentWebhookContext{Type: domain.RecurringSuccess, OrgId: "org_1", OrderId: "ord_1"}}
		engine := newWebhookEngine()
		engine.signalErr = errors.New("engine signal failed")
		subs := &fakeSubRepo{byOrderId: []domain.Subscription{{OrgId: "org_1", Id: "sub_1"}}}
		svc := newWebhookService(factoryWithParser(parser), engine, idem, subs)

		err := svc.HandlePaymentWebhook(context.Background(), port.PaymentWebhookPayload{Psp: domain.Paystack, Data: "{}"})

		require.Error(t, err)
		assert.Equal(t, 1, idem.released)
	})

	t.Run("recurring.success with no subscriptions retains the claim (no signal to send)", func(t *testing.T) {
		idem := &fakeIdempotencyRepo{claimAvailable: true}
		parser := stubParser{parsed: domain.PaymentWebhookContext{Type: domain.RecurringSuccess, OrgId: "org_1", OrderId: "ord_1"}}
		engine := newWebhookEngine()
		svc := newWebhookService(factoryWithParser(parser), engine, idem, &fakeSubRepo{})

		err := svc.HandlePaymentWebhook(context.Background(), port.PaymentWebhookPayload{Psp: domain.Paystack, Data: "{}"})

		require.NoError(t, err)
		assert.Equal(t, 0, engine.signalledSubs)
		assert.Equal(t, 0, idem.released, "no work was attempted — retrying via PSP would do the same nothing")
	})

	t.Run("payment.refunded starts the refund workflow", func(t *testing.T) {
		idem := &fakeIdempotencyRepo{claimAvailable: true}
		parser := stubParser{parsed: domain.PaymentWebhookContext{Type: domain.PaymentRefunded, OrgId: "org_1", OrderId: "ord_1"}}
		engine := newWebhookEngine()
		svc := newWebhookService(factoryWithParser(parser), engine, idem, &fakeSubRepo{})

		err := svc.HandlePaymentWebhook(context.Background(), port.PaymentWebhookPayload{Psp: domain.Paystack, Data: "{}"})

		require.NoError(t, err)
		assert.Equal(t, 1, engine.startedWorkflows[port.WorkflowPaymentRefunded])
		assert.Equal(t, 0, idem.released)
	})
}

func TestWebhookService_HandleAuthnWebhook(t *testing.T) {
	svc := newWebhookService(factoryWithParser(stubParser{}), newWebhookEngine(), &fakeIdempotencyRepo{}, &fakeSubRepo{})
	err := svc.HandleAuthnWebhook(context.Background(), port.AuthnWebhookPayload{Provider: "clerk", Data: "{}"})
	require.NoError(t, err)
}
