package port

import (
	"context"
	"payloop/internal/core/domain"
)

// OrderWorkflowService handles order-related workflow operations.
type OrderWorkflowService interface {
	CompleteCheckoutSession(ctx context.Context, input domain.CompleteCheckoutSessionInput) (domain.Order, error)
}

// SubscriptionService handles subscription-related operations needed by workflow activities.
type SubscriptionService interface {
	GetSubscriptionCustomer(ctx context.Context, subscription domain.Subscription) (domain.Customer, error)
	GetSubscriptionPaymentMethod(ctx context.Context, subscription domain.Subscription) (domain.PaymentMethod, error)
	HandleSubscriptionChargeSuccess(ctx context.Context, input domain.SubscriptionChargeInput) (domain.Subscription, error)
	HandleSubscriptionChargeFailure(ctx context.Context, input domain.SubscriptionChargeInput) (domain.Subscription, error)
}

// WebhookSubscriptionService handles outbound webhook delivery.
type WebhookSubscriptionService interface {
	SendWebhook(ctx context.Context, input OutgoingWebhookPayload) error
}

// GatewayFactory creates payment gateway instances from configuration.
type GatewayFactory interface {
	NewGateway(ctx context.Context, orgId string, id string) (domain.GatewayProvider, error)
}
