package services

import (
	"context"
	"errors"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/events/topic"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/subscriptions"

	"payloop/internal/domain/payment_providers"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
)

type SubscriptionOrchestrationService struct {
	interfaces.SubscriptionService
	workflowEngine         interfaces.Engine
	sessionRepository      repositories.SessionRepository
	cartRepository         repositories.CartRepository
	orderRepository        repositories.OrderRepository
	customerRepository     repositories.CustomerRepository
	subscriptionRepository repositories.SubscriptionRepository
	paymentRepository      repositories.PaymentRepository
	orderItemRepository    repositories.OrderItemRepository
	workflowService        interfaces.WorkflowService
	paymentGateway         payment_providers.Gateway
	pubsub                 events.PubSub
	logger                 logger.Logger
}

func NewSubscriptionOrchestrationService(
	sessionRepository repositories.SessionRepository,
	cartRepository repositories.CartRepository,
	subscriptionRepository repositories.SubscriptionRepository,
	orderItemRepository repositories.OrderItemRepository,
	customerRepository repositories.CustomerRepository,
	orderRepository repositories.OrderRepository,
	paymentRepository repositories.PaymentRepository,
	pubsub events.PubSub,
	paymentGateway payment_providers.Gateway,
	logger logger.Logger,
	subs interfaces.SubscriptionService,
	workflowEngine interfaces.Engine,
) interfaces.SubscriptionOrchestrationService {

	_, err := pubsub.Subscribe("subscription.workflow.>", func(topic string, data []byte) {
		logger.Infof("Received message from %s", topic)
	})
	if err != nil {
		logger.Error("Failed to subscribe to topic", err.Error())
		panic(err)
	}

	return SubscriptionOrchestrationService{
		SubscriptionService:    subs,
		workflowEngine:         workflowEngine,
		customerRepository:     customerRepository,
		sessionRepository:      sessionRepository,
		paymentRepository:      paymentRepository,
		cartRepository:         cartRepository,
		orderRepository:        orderRepository,
		orderItemRepository:    orderItemRepository,
		subscriptionRepository: subscriptionRepository,
		pubsub:                 pubsub,
		logger:                 logger,
		paymentGateway:         paymentGateway,
	}
}

func (s SubscriptionOrchestrationService) Update(ctx context.Context, input subscriptions.UpdateSubscriptionInput) (entities.Subscription, error) {
	s.logger.Info("Updating subscription", "orgId", input.OrgId, "id", input.Id)

	subscription, err := s.subscriptionRepository.FindById(ctx, input.OrgId, input.Id)
	if err != nil {
		s.logger.Error("Failed to find subscriptions", err.Error())
		return entities.Subscription{}, err
	}

	if input.Status != subscription.Status {
		s.logger.Infof("Updating status %s", input.Status)
		subscription.Status = input.Status
	}

	newSub, err := s.subscriptionRepository.Update(ctx, subscription)
	if err != nil {
		s.logger.Error("Failed to update subscription", err.Error())
		return entities.Subscription{}, err
	}

	_ = s.pubsub.Publish(subscription.OrgId, entities.GetTopicFromStatus(subscription.Status), newSub)
	return newSub, err
}

// Activate a subscription and update the Entity Workflow
func (s SubscriptionOrchestrationService) Activate(ctx context.Context, orgId string, id string) (entities.Subscription, error) {
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

	_, err = s.workflowEngine.StartSubscriptionWorkflow(ctx, subscription)
	if err != nil {
		s.logger.Error("Failed to start workflow", err.Error())
		return entities.Subscription{}, err
	}

	return subscription, nil
}

// Pause a subscription and update the Entity Workflow
func (s SubscriptionOrchestrationService) PauseSubscription(ctx context.Context, input subscriptions.PauseSubscriptionInput) (entities.Subscription, error) {
	s.logger.Info("Pausing subscription", "orgId", input.OrgId, "id", input.Id)

	subscription, err := s.SubscriptionService.PauseSubscription(ctx, input)
	if err != nil {
		s.logger.Error("Failed to pause subscription", err.Error())
		var serr lib.CustomError
		if errors.As(err, &serr) {
			return entities.Subscription{}, err
		}
		return entities.Subscription{}, lib.NewCustomError(lib.InternalError, "", err)
	}

	// update the subscription workflow
	err = s.workflowEngine.UpdateSubscriptionWorkflow(ctx, "subscription.paused", subscription)
	if err != nil {
		s.logger.Error("Failed to update workflow", err.Error())
		var serr lib.CustomError
		if errors.As(err, &serr) {
			return entities.Subscription{}, err
		}
		return entities.Subscription{}, lib.NewCustomError(lib.InternalError, "", err)
	}

	// publi
	_ = s.pubsub.Publish(subscription.OrgId, topic.TopicSubscriptionPaused, subscription)

	return subscription, nil
}

func (s SubscriptionOrchestrationService) ResumeSubscription(ctx context.Context, input subscriptions.ResumeSubscriptionInput) (entities.Subscription, error) {
	s.logger.Info("Resuming subscription", "orgId", input.OrgId, "id", input.Id)

	newSub, err := s.SubscriptionService.ResumeSubscription(ctx, input)
	if err != nil {
		var serr lib.CustomError
		if errors.As(err, &serr) {
			return entities.Subscription{}, err
		}
		return entities.Subscription{}, lib.NewCustomError(lib.InternalError, "", err)
	}

	// update the subscription workflow
	err = s.workflowEngine.UpdateSubscriptionWorkflow(ctx, topic.TopicSubscriptionResumed, newSub)
	if err != nil {
		s.logger.Error("Failed to update workflow", err.Error())
		var serr lib.CustomError
		if errors.As(err, &serr) {
			return entities.Subscription{}, err
		}
		return entities.Subscription{}, lib.NewCustomError(lib.InternalError, "", err)
	}

	// Publish the resume event
	_ = s.pubsub.Publish(newSub.OrgId, topic.TopicSubscriptionResumed, newSub)

	return newSub, nil
}

// CancelSubscription
// A canceled subscription will continue through its current billing cycle. At the end of the current billing cycle the subscription will expire and the customer will no longer be billed.
// Canceled subscriptions can be reactivated until the end of the billing cycle
func (s SubscriptionOrchestrationService) CancelSubscription(ctx context.Context, input subscriptions.CancelSubscriptionInput) (entities.Subscription, error) {

	subscription, err := s.SubscriptionService.FindById(ctx, input.OrgId, input.Id)
	if err != nil {
		var serr lib.CustomError
		if errors.As(err, &serr) {
			return entities.Subscription{}, err
		}
		return entities.Subscription{}, lib.NewCustomError(lib.InternalError, "", err)
	}

	// update the workflow
	err = s.workflowEngine.UpdateSubscriptionWorkflow(ctx, "cancel", subscription)
	if err != nil {
		s.logger.Error("Failed to update workflow", err.Error())
		var serr lib.CustomError
		if errors.As(err, &serr) {
			return entities.Subscription{}, err
		}
		return entities.Subscription{}, lib.NewCustomError(lib.InternalError, "", err)
	}

	// publi
	_ = s.pubsub.Publish(subscription.OrgId, topic.TopicSubscriptionCancelled, subscription)

	return subscription, nil
}
