package service

import (
	"context"
	"encoding/json"
	"getpaidhq/internal/core/port"
)

type WorkflowService struct {
	logger          port.Logger
	idempotencyRepo port.IdempotencyKeyRepository
	whsRepo         port.WebhookSubscriptionRepository
	pubsub          port.PubSub
	engine          port.Engine
}

func NewWorkflowService(
	logger port.Logger,
	whsRepo port.WebhookSubscriptionRepository,
	idempotencyRepo port.IdempotencyKeyRepository,
	pubsub port.PubSub,
	engine port.Engine,
) *WorkflowService {
	service := &WorkflowService{
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
func (s *WorkflowService) HandleOutboundWebhook(topic string, data []byte) {
	s.logger.Infof("[WorkflowService] checking topic: %s", topic)
	// Check if the org is subscribed to any outgoing messages and send them using a workflow

	var payload port.PubSubPayload
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
		_, err = s.engine.StartWorkflow(context.TODO(), port.WorkflowOutgoingWebhook, port.OutgoingWebhookPayload{
			WebhookSubscription: sub,
			Event:               payload,
		})
		if err != nil {
			s.logger.Errorf("Failed to start workflow %v", err.Error())
		}
	}

}
