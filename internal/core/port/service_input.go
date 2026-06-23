package port

import "getpaidhq/internal/core/domain"

// CompleteCheckoutSessionInput is the input for OrderWorkflowService.CompleteCheckoutSession.
type CompleteCheckoutSessionInput struct {
	OrgId          string
	OrderId        string
	PaymentContext domain.PaymentWebhookContext
	Metadata       map[string]string
}

// SubscriptionChargeInput wraps a subscription and the result of a charge attempt for billing.
type SubscriptionChargeInput struct {
	Subscription domain.Subscription
	ChargeResult domain.ChargeResult
}
