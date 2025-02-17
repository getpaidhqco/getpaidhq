package types

import (
	"context"
	temporal_workflow "go.temporal.io/sdk/workflow"
	"payloop/internal/application/interfaces"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/payments"
	"payloop/internal/domain/payment_providers"
)

type StoreSubscriptionWorkflowContextInput struct {
	OrgId          string
	SubscriptionId string
	Execution      temporal_workflow.Execution
}

type OrderActivities interface {
	CompleteOrder(ctx context.Context, paymentContext payment_providers.PaymentWebhookContext) (interfaces.Result, error)
	GetOrderSubscriptions(ctx context.Context, orgId string, orderId string) ([]entities.Subscription, error)
	ChargeCustomerForBillingPeriod(ctx context.Context, subscription entities.Subscription) (payments.ChargeResult, error)
	HandleChargeResult(ctx context.Context, subscription entities.Subscription, chargeResult payments.ChargeResult) (entities.Subscription, error)
	StoreSubscriptionWorkflowContext(ctx context.Context, input StoreSubscriptionWorkflowContextInput) error
	GetSubscription(ctx context.Context, orgId string, id string) (entities.Subscription, error)
}
