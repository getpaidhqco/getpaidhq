package services

import (
	"context"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/payment_providers"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
)

type SubscriptionService struct {
	sessionRepository      repositories.SessionRepository
	cartRepository         repositories.CartRepository
	orderRepository        repositories.OrderRepository
	customerRepository     repositories.CustomerRepository
	subscriptionRepository repositories.SubscriptionRepository
	paymentGateway         payment_providers.Gateway
	logger                 lib.Logger
}

func NewSubscriptionService(
	sessionRepository repositories.SessionRepository,
	cartRepository repositories.CartRepository,
	subscriptionRepository repositories.SubscriptionRepository,
	customerRepository repositories.CustomerRepository,
	orderRepository repositories.OrderRepository,
	paymentGateway payment_providers.Gateway,
	logger lib.Logger,
) SubscriptionService {
	return SubscriptionService{
		customerRepository:     customerRepository,
		sessionRepository:      sessionRepository,
		cartRepository:         cartRepository,
		orderRepository:        orderRepository,
		subscriptionRepository: subscriptionRepository,
		logger:                 logger,
		paymentGateway:         paymentGateway,
	}
}

func (s *SubscriptionService) Activate(ctx context.Context, orgId string, id string) (entities.Subscription, error) {
	s.logger.Info("Marking subscription active", "orgId", orgId, "id", id)

	subscription, err := s.subscriptionRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to find subscriptions", err.Error())
		return entities.Subscription{}, err
	}

	subscription.Status = entities.SubscriptionStatusActive
	subscription, err = s.subscriptionRepository.Update(ctx, subscription)
	if err != nil {
		s.logger.Error("Failed to update subscription", err.Error())
		return entities.Subscription{}, err
	}
	
	return subscription, nil
}
