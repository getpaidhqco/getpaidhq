package types

import (
	"context"
	temporal_workflow "go.temporal.io/sdk/workflow"
	"payloop/internal/core/domain"
	"payloop/internal/core/port"
)

type StoreSubscriptionWorkflowContextInput struct {
	OrgId          string
	SubscriptionId string
	Execution      temporal_workflow.Execution
}

type OrderActivities interface {
	CompleteOrder(ctx context.Context, paymentContext domain.PaymentWebhookContext) (port.WorkflowResult, error)
	GetOrderSubscriptions(ctx context.Context, orgId string, orderId string) ([]domain.Subscription, error)
	ChargeCustomerForBillingPeriod(ctx context.Context, subscription domain.Subscription) (domain.ChargeResult, error)
	HandleChargeResult(ctx context.Context, subscription domain.Subscription, chargeResult domain.ChargeResult) (domain.Subscription, error)
	StoreSubscriptionWorkflowContext(ctx context.Context, input StoreSubscriptionWorkflowContextInput) error
	GetSubscription(ctx context.Context, orgId string, id string) (domain.Subscription, error)
}
