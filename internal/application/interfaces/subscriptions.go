package interfaces

import (
	"context"
	"payloop/internal/api/dto/request"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/subscriptions"
)

type SubscriptionService interface {
	CreateSubscriptionsForOrder(ctx context.Context, orgId string, orderId string) ([]entities.Subscription, error)
	Create(ctx context.Context, input entities.CreateSubscriptionInput) (entities.Subscription, error)
	Update(ctx context.Context, input subscriptions.UpdateSubscriptionInput) (entities.Subscription, error)
	FindById(ctx context.Context, orgId string, id string) (entities.Subscription, error)
	Activate(ctx context.Context, orgId string, id string) (entities.Subscription, error)
	Pause(ctx context.Context, input subscriptions.PauseSubscriptionInput) (entities.Subscription, error)
	List(ctx context.Context, orgId string, pagination request.Pagination) ([]entities.Subscription, error)
	ResumeSubscription(ctx context.Context, input subscriptions.ResumeSubscriptionInput) (entities.Subscription, error)
	CancelSubscription(ctx context.Context, input subscriptions.CancelSubscriptionInput) (entities.Subscription, error)
}

type SubscriptionActivityService interface {
	CreateSubscriptionsForOrder(ctx context.Context, orgId string, orderId string) ([]entities.Subscription, error)
	FindById(ctx context.Context, orgId string, id string) (entities.Subscription, error)
	Activate(ctx context.Context, orgId string, id string) (entities.Subscription, error)
	Pause(ctx context.Context, input subscriptions.PauseSubscriptionInput) (entities.Subscription, error)
	List(ctx context.Context, orgId string, pagination request.Pagination) ([]entities.Subscription, error)
	ResumeSubscription(ctx context.Context, input subscriptions.ResumeSubscriptionInput) (entities.Subscription, error)
	CancelSubscription(ctx context.Context, input subscriptions.CancelSubscriptionInput) (entities.Subscription, error)
	GetSubscriptionCustomer(ctx context.Context, subscription entities.Subscription) (entities.Customer, error)
	GetSubscriptionPaymentMethod(ctx context.Context, subscription entities.Subscription) (entities.PaymentMethod, error)
	HandleSubscriptionChargeSuccess(ctx context.Context, input subscriptions.SubscriptionChargeInput) (entities.Subscription, error)
	HandleSubscriptionChargeFailure(ctx context.Context, input subscriptions.SubscriptionChargeInput) (entities.Subscription, error)
}
