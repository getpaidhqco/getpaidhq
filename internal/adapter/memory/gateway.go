package memory

import (
	"context"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

// GatewayAdapter is an in-memory implementation of port.GatewayAdapter.
//
// It is a deterministic, no-network PSP that always reports successful
// charges. Its purpose is twofold:
//
//   - Local/offline development: an org whose PSP config selects the
//     "memory" gateway can run the full charge path without talking to a
//     real processor.
//   - Testing: the billing charge->state-advance integration test drives
//     SubscriptionService.ChargeForBillingPeriod against this adapter so the
//     tail of the billing pipeline can be exercised without Paystack /
//     Checkout.com.
//
// It is harmless in production: it is only ever reached if an org's stored
// PSP configuration explicitly names domain.Memory.
type GatewayAdapter struct {
	logger port.Logger
}

// NewGatewayAdapter builds the in-memory gateway adapter. Only a logger is
// required; the gateway holds no external state.
func NewGatewayAdapter(logger port.Logger) *GatewayAdapter {
	return &GatewayAdapter{logger: logger}
}

// CreateGateway returns a provider that always succeeds. The settings JSON is
// ignored — the in-memory gateway has no configuration to parse — but the
// signature matches every other adapter so the GatewayFactory treats it
// uniformly.
func (a *GatewayAdapter) CreateGateway(_ string) (domain.GatewayProvider, error) {
	return &gatewayProvider{logger: a.logger}, nil
}

// CreateWebhookParser returns a minimal parser. The in-memory gateway never
// emits webhooks, so validation is a no-op (accept everything) and parsing
// returns an empty noop context. It exists only to satisfy the interface.
func (a *GatewayAdapter) CreateWebhookParser() domain.WebhookParser {
	return &webhookParser{}
}

// gatewayProvider implements domain.GatewayProvider with always-successful
// charges.
type gatewayProvider struct {
	logger port.Logger
}

// InitPayment is a no-op init that echoes back the originating command. The
// in-memory gateway has no hosted checkout, so there is nothing to redirect
// to; callers that only need the recurring ChargePayment path never reach
// this.
func (g *gatewayProvider) InitPayment(_ context.Context, input domain.InitPaymentCommand) (domain.InitPaymentResponse, error) {
	return domain.InitPaymentResponse{PspResponse: input}, nil
}

// DeclineToken is the memory gateway's test-card sentinel: a charge whose
// payment-method token equals it is declined (a retryable card error), the way
// PSP test cards decline. Every other token succeeds.
const DeclineToken = "tok_decline"

// ChargePayment returns a succeeded charge, unless the payment method carries
// DeclineToken — then it returns a declined, retryable charge.
//
// The response is shaped to satisfy SubscriptionService.ChargeForBillingPeriod:
// it reads Status (must be ChargePaymentStatusSuccess to map to
// PaymentStatusSucceeded), Psp, PspId, Reference and AmountCharged off this
// struct. A fresh reference is minted per call so distinct charges produce
// distinct payment rows.
func (g *gatewayProvider) ChargePayment(_ context.Context, input domain.ChargePaymentCommand) domain.ChargePaymentResponse {
	reference := input.Reference
	if reference == "" {
		reference = lib.GenerateId("memref")
	}
	if input.PaymentMethod.Token == DeclineToken {
		return domain.ChargePaymentResponse{
			Status:      domain.ChargePaymentStatusError,
			Retryable:   true,
			Psp:         domain.Memory,
			PspId:       lib.GenerateId("mempsp"),
			Reference:   reference,
			Currency:    domain.Currency(input.Currency),
			ErrorCode:   "card_declined",
			ErrorReason: "memory gateway decline token",
			PaymentType: "card",
			PspResponse: map[string]any{
				"gateway":   "memory",
				"succeeded": false,
				"amount":    input.Amount,
				"currency":  input.Currency,
			},
		}
	}
	return domain.ChargePaymentResponse{
		Status:        domain.ChargePaymentStatusSuccess,
		Retryable:     false,
		Psp:           domain.Memory,
		PspId:         lib.GenerateId("mempsp"),
		Reference:     reference,
		Currency:      domain.Currency(input.Currency),
		AmountCharged: input.Amount,
		PaymentType:   "card",
		PspResponse: map[string]any{
			"gateway":   "memory",
			"succeeded": true,
			"amount":    input.Amount,
			"currency":  input.Currency,
		},
	}
}

// webhookParser is a minimal, accept-everything parser. The in-memory gateway
// is not exercised via webhooks; this exists only to satisfy the interface.
type webhookParser struct{}

// ValidateWebhook accepts every payload (returns nil). The in-memory gateway
// has no signing secret.
func (p *webhookParser) ValidateWebhook(_ context.Context, _ []byte, _ string) error {
	return nil
}

// ParseWebhook returns an empty noop context.
func (p *webhookParser) ParseWebhook(_ context.Context, data []byte) (domain.PaymentWebhookContext, error) {
	return domain.PaymentWebhookContext{
		Type:    domain.Noop,
		Psp:     domain.Memory,
		RawData: data,
	}, nil
}
