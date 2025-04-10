package services

import (
	"context"
	"encoding/json"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/repositories"
	"payloop/internal/domain/workflow"
)

type WorkflowService struct {
	logger          logger.Logger
	idempotencyRepo repositories.IdempotencyKeyRepository
	whsRepo         repositories.WebhookSubscriptionRepository
	pubsub          events.PubSub
	engine          interfaces.Engine
}

func NewWorkflowService(
	logger logger.Logger,
	whsRepo repositories.WebhookSubscriptionRepository,
	idempotencyRepo repositories.IdempotencyKeyRepository,
	pubsub events.PubSub,
	engine interfaces.Engine,
) interfaces.WorkflowService {
	service := WorkflowService{
		logger:          logger,
		whsRepo:         whsRepo,
		pubsub:          pubsub,
		engine:          engine,
		idempotencyRepo: idempotencyRepo,
	}
	logger.Debugf("[WorkflowService] Subscribing to all topics")
	_, err := pubsub.Subscribe(">", service.HandleOutboundWebhook)
	if err != nil {
		logger.Error("Failed to subscribe to topic", err.Error())
		panic(err)
	}

	return service
}

// HandleOutboundWebhook listens for all published messages and starts an outgoing webhook workflow
// if the org is subscribed to the event.
func (s WorkflowService) HandleOutboundWebhook(topic string, data []byte) {
	s.logger.Infof("[WorkflowService] checking topic: %s", topic)
	// Check if the org is subscribed to any outgoing messages and send them using a workflow

	var payload events.Payload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		s.logger.Errorf("Failed to unmarshal payload: %v", err)
		return
	}

	subs, err := s.whsRepo.FindByEvent(context.Background(), payload.OrgId, payload.Topic)
	if err != nil {
		s.logger.Errorf("Failed to get webhook subscriptions: %v", err)
		return
	}

	for _, sub := range subs {
		_, err = s.engine.StartWorkflow(context.TODO(), interfaces.OutgoingWebhook, workflow.OutgoingWebhookPayload{
			WebhookSubscription: sub,
			Event:               payload,
		})
		if err != nil {
			s.logger.Errorf("Failed to start workflow %v", err.Error())
		}
	}

}
