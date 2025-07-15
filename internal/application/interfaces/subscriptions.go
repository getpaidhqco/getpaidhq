package interfaces

import (
	"context"
	"payloop/internal/api/dto/request"
	"payloop/internal/application/dto"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/payments"
	"payloop/internal/domain/entities/settings"
	"payloop/internal/domain/entities/subscriptions"
)

// SubscriptionOrchestrationService is a service that provides orchestration of subscription operations
// It's a superset of SubscriptionService and provides additional workflow operations
type SubscriptionOrchestrationService interface {
	SubscriptionService
	Update(ctx context.Context, input subscriptions.UpdateSubscriptionInput) (entities.Subscription, error)
	PauseSubscription(ctx context.Context, input subscriptions.PauseSubscriptionInput) (entities.Subscription, error)
	ResumeSubscription(ctx context.Context, input subscriptions.ResumeSubscriptionInput) (entities.Subscription, error)
	CancelSubscription(ctx context.Context, input subscriptions.CancelSubscriptionInput) (entities.Subscription, error)
	UpdateWorkflowState(ctx context.Context, orgId string, id string) (entities.Subscription, error)
}

type SubscriptionService interface {
	CreateSubscriptionsForOrder(ctx context.Context, orgId string, orderId string) ([]entities.Subscription, error)
	Create(ctx context.Context, orgId string, input dto.CreateSubscriptionInput) (entities.Subscription, error)
	FindById(ctx context.Context, orgId string, id string) (entities.Subscription, error)
	Activate(ctx context.Context, input subscriptions.ActivateSubscriptionInput) (entities.Subscription, error)
	PauseSubscription(ctx context.Context, input subscriptions.PauseSubscriptionInput) (entities.Subscription, error)
	List(ctx context.Context, orgId string, pagination request.Pagination) ([]entities.Subscription, int, error)
	FindSubscriptionPayments(ctx context.Context, pk entities.EntityKey, pagination request.Pagination) ([]entities.Payment, int, error)
	ResumeSubscription(ctx context.Context, input subscriptions.ResumeSubscriptionInput) (entities.Subscription, error)
	CancelSubscription(ctx context.Context, input subscriptions.CancelSubscriptionInput) (entities.Subscription, error)
	ChangeSubscriptionPlan(ctx context.Context, input subscriptions.ChangePlanInput) (*entities.Subscription, *entities.SubscriptionPlanChange, error)
	UpdateBillingAnchor(ctx context.Context, input dto.UpdateBillingAnchorInput) (dto.UpdateBillingAnchorResult, error)
	GetSubscriptionCustomer(ctx context.Context, subscription entities.Subscription) (entities.Customer, error)
	GetSubscriptionPaymentMethod(ctx context.Context, subscription entities.Subscription) (entities.SecurePaymentMethod, error)
	HandleSubscriptionChargeSuccess(ctx context.Context, input subscriptions.SubscriptionChargeInput) (entities.Subscription, error)
	HandleSubscriptionChargeFailure(ctx context.Context, input subscriptions.SubscriptionChargeInput) (entities.Subscription, error)
	GetOrgSubscriptionSettings(ctx context.Context, orgId string) (settings.Subscription, error)
	ProcessSubscriptionCharge(ctx context.Context, subscription entities.Subscription) (payments.ChargeResult, error)
}
