package services

import (
	"context"
	"errors"
	"payloop/internal/application/dto"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/events/topic"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/subscriptions"
	"payloop/internal/domain/factories"
	"payloop/internal/lib/apperrors"

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
	gatewayFactory         factories.GatewayFactory
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
	gatewayFactory factories.GatewayFactory,
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
		gatewayFactory:         gatewayFactory,
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
		s.logger.Error("Failed to update subscription", "err", err.Error())
		return entities.Subscription{}, err
	}

	_ = s.pubsub.Publish(subscription.OrgId, topic.GetSubscriptionTopic(subscription.Status), newSub)
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
		s.logger.Error("Failed to update subscription", "err", err.Error())
		return entities.Subscription{}, err
	}
	err = s.workflowEngine.StartSubscriptionWorkflow(ctx, subscription)
	if err != nil {
		s.logger.Errorf("Failed to start workflow %v", err.Error())
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

// UpdateWorkflowState Calls the workflow engine to update the workflow state. This is used to refresh the workflow state with what is in the database
// and is used for debugging and error recovery purposes.
func (s SubscriptionOrchestrationService) UpdateWorkflowState(ctx context.Context, orgId string, id string) (entities.Subscription, error) {
	s.logger.Infof("Updating workflow [%s][%s]", orgId, id)

	subscription, err := s.subscriptionRepository.FindById(ctx, orgId, id)
	if err != nil {
		return entities.Subscription{}, lib.NewCustomError(lib.NotFoundError, "Not found", err)
	}

	// update the subscription workflow
	err = s.workflowEngine.UpdateSubscriptionWorkflow(ctx, "refresh-state", subscription)
	if err != nil {
		var serr lib.CustomError
		if errors.As(err, &serr) {
			return entities.Subscription{}, err
		}
		return entities.Subscription{}, lib.NewCustomError(lib.InternalError, err.Error(), err)
	}

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

	subscription, err := s.SubscriptionService.CancelSubscription(ctx, input)
	if err != nil {
		var serr lib.CustomError
		if errors.As(err, &serr) {
			return entities.Subscription{}, err
		}
		return entities.Subscription{}, lib.NewCustomError(lib.InternalError, "", err)
	}

	// update the workflow
	s.logger.Debugf("Updating workflow for subscription %s [%s]", subscription.Id, topic.TopicSubscriptionCancelled)
	err = s.workflowEngine.UpdateSubscriptionWorkflow(ctx, topic.TopicSubscriptionCancelled, subscription)
	if err != nil {
		switch e := err.(type) {
		case apperrors.NotFound:
			// since we're cancelling, we don't care if the workflow is not found
			s.logger.Warnf("Workflow not found for subscription %s: %v", subscription.Id, e)
			break

		default:
			var serr lib.CustomError
			if errors.As(err, &serr) {
				return entities.Subscription{}, err
			}
			return entities.Subscription{}, lib.NewCustomError(lib.InternalError, "", err)
		}

	}

	// publi
	_ = s.pubsub.Publish(subscription.OrgId, topic.TopicSubscriptionCancelled, subscription)

	return subscription, nil
}

func (s SubscriptionOrchestrationService) UpdateBillingAnchor(ctx context.Context, input dto.UpdateBillingAnchorInput) (dto.UpdateBillingAnchorResult, error) {
	s.logger.Infof("Updating billing anchor for subscription %s", input.Id)

	result, err := s.SubscriptionService.UpdateBillingAnchor(ctx, input)
	if err != nil {
		var serr lib.CustomError
		if errors.As(err, &serr) {
			return dto.UpdateBillingAnchorResult{}, err
		}
		return dto.UpdateBillingAnchorResult{}, lib.NewCustomError(lib.InternalError, "", err)
	}

	// update the workflow
	sub, err := s.UpdateWorkflowState(ctx, result.Subscription.OrgId, result.Subscription.Id)
	if err != nil {
		var serr lib.CustomError
		if errors.As(err, &serr) {
			return dto.UpdateBillingAnchorResult{}, err
		}
		return dto.UpdateBillingAnchorResult{}, lib.NewCustomError(lib.InternalError, "", err)
	}

	// Publish the event
	_ = s.pubsub.Publish(result.Subscription.OrgId, topic.SubscriptionBillingAnchorChanged, sub)

	return result, nil
}
