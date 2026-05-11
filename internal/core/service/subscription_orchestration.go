package service

import (
	"context"
	"errors"
	"payloop/internal/core/domain"
	"payloop/internal/core/port"
	"payloop/internal/lib"
)

// SubscriptionOrchestrationService wraps the narrow SubscriptionService and
// adds workflow-engine signaling for lifecycle transitions. HTTP handlers
// depend on this service; activities depend on the narrow one (via the
// port.SubscriptionService interface).
//
// The wrapping pattern keeps the engine off the narrow service so activities
// — which the engine itself dispatches — can be constructed before the engine
// exists. The construction-time cycle that would otherwise exist is broken at
// the type level, not papered over at the wiring level.
type SubscriptionOrchestrationService struct {
	*SubscriptionService
	engine port.Engine
	logger port.Logger
}

func NewSubscriptionOrchestrationService(
	subs *SubscriptionService,
	engine port.Engine,
	logger port.Logger,
) *SubscriptionOrchestrationService {
	return &SubscriptionOrchestrationService{
		SubscriptionService: subs,
		engine:              engine,
		logger:              logger,
	}
}

func (s *SubscriptionOrchestrationService) Activate(ctx context.Context, orgId string, id string) (domain.Subscription, error) {
	subscription, err := s.SubscriptionService.Activate(ctx, orgId, id)
	if err != nil {
		return domain.Subscription{}, err
	}

	err = s.engine.StartSubscriptionWorkflow(ctx, subscription)
	if err != nil {
		s.logger.Errorf("Failed to start workflow %v", err.Error())
		return domain.Subscription{}, err
	}

	return subscription, nil
}

func (s *SubscriptionOrchestrationService) PauseSubscription(ctx context.Context, input domain.PauseSubscriptionInput) (domain.Subscription, error) {
	subscription, err := s.SubscriptionService.PauseSubscription(ctx, input)
	if err != nil {
		return domain.Subscription{}, err
	}

	err = s.engine.UpdateSubscriptionWorkflow(ctx, "subscription.paused", subscription)
	if err != nil {
		if _, ok := errors.AsType[lib.CustomError](err); ok {
			return domain.Subscription{}, err
		}
		return domain.Subscription{}, lib.NewCustomError(lib.InternalError, "", err)
	}

	_ = s.pubsub.Publish(subscription.OrgId, port.TopicSubscriptionPaused, subscription)
	return subscription, nil
}

func (s *SubscriptionOrchestrationService) ResumeSubscription(ctx context.Context, input domain.ResumeSubscriptionInput) (domain.Subscription, error) {
	newSub, err := s.SubscriptionService.ResumeSubscription(ctx, input)
	if err != nil {
		return domain.Subscription{}, err
	}

	err = s.engine.UpdateSubscriptionWorkflow(ctx, port.TopicSubscriptionResumed, newSub)
	if err != nil {
		if _, ok := errors.AsType[lib.CustomError](err); ok {
			return domain.Subscription{}, err
		}
		return domain.Subscription{}, lib.NewCustomError(lib.InternalError, "", err)
	}

	_ = s.pubsub.Publish(newSub.OrgId, port.TopicSubscriptionResumed, newSub)
	return newSub, nil
}

func (s *SubscriptionOrchestrationService) CancelSubscription(ctx context.Context, input domain.CancelSubscriptionInput) (domain.Subscription, error) {
	subscription, err := s.SubscriptionService.CancelSubscription(ctx, input)
	if err != nil {
		return domain.Subscription{}, err
	}

	s.logger.Debugf("Updating workflow for subscription %s [%s]", subscription.Id, port.TopicSubscriptionCancelled)
	err = s.engine.UpdateSubscriptionWorkflow(ctx, port.TopicSubscriptionCancelled, subscription)
	if err != nil {
		if _, ok := errors.AsType[lib.CustomError](err); ok {
			return domain.Subscription{}, err
		}
		return domain.Subscription{}, lib.NewCustomError(lib.InternalError, "", err)
	}

	_ = s.pubsub.Publish(subscription.OrgId, port.TopicSubscriptionCancelled, subscription)
	return subscription, nil
}

// UpdateWorkflowState refreshes the workflow state from the database. Used for
// debugging and error recovery.
func (s *SubscriptionOrchestrationService) UpdateWorkflowState(ctx context.Context, orgId string, id string) (domain.Subscription, error) {
	s.logger.Infof("Updating workflow [%s][%s]", orgId, id)

	subscription, err := s.subscriptionRepository.FindById(ctx, orgId, id)
	if err != nil {
		return domain.Subscription{}, lib.NewCustomError(lib.NotFoundError, "Not found", err)
	}

	err = s.engine.UpdateSubscriptionWorkflow(ctx, "refresh-state", subscription)
	if err != nil {
		if _, ok := errors.AsType[lib.CustomError](err); ok {
			return domain.Subscription{}, err
		}
		return domain.Subscription{}, lib.NewCustomError(lib.InternalError, err.Error(), err)
	}

	return subscription, nil
}
