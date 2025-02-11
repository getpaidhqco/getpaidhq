package services

import (
	"context"
	"encoding/json"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/events/topic"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/webhooks"
	"payloop/internal/domain/repositories"
	"payloop/internal/domain/workflow"
	"payloop/internal/lib"
	"time"
)

type WebhookSubscriptionService struct {
	logger          lib.Logger
	idempotencyRepo repositories.IdempotencyKeyRepository
	whsRepo         repositories.WebhookSubscriptionRepository
	pubsub          events.PubSub
	engine          workflow.Engine
}

func NewWebhookSubscriptionService(
	logger lib.Logger,
	whsRepo repositories.WebhookSubscriptionRepository,
	idempotencyRepo repositories.IdempotencyKeyRepository,
	pubsub events.PubSub,
	engine workflow.Engine,
) WebhookSubscriptionService {
	service := WebhookSubscriptionService{
		logger:          logger,
		whsRepo:         whsRepo,
		pubsub:          pubsub,
		engine:          engine,
		idempotencyRepo: idempotencyRepo,
	}

	_, err := pubsub.Subscribe(">", service.HandlePubSubMessage)
	if err != nil {
		logger.Error("Failed to subscribe to topic", err.Error())
		panic(err)
	}

	return service
}

// HandlePubSubMessage handles incoming PubSub messages
func (s WebhookSubscriptionService) HandlePubSubMessage(topic string, data []byte) {
	s.logger.Infof("[Outgoing Webhook::HandlePubSubMessage]: %s", topic)
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
		result, err := s.engine.StartWorkflow(context.TODO(), workflow.OutgoingWebhook, workflow.OutgoingWebhookPayload{
			WebhookSubscription: sub,
			Event:               payload,
		})
		if err != nil {
			s.logger.Errorf("Failed to start workflow", err.Error())
		}
		s.logger.Infof("Workflow result: %v", result)
	}

}

func (s WebhookSubscriptionService) Create(ctx context.Context, input webhooks.CreateWebhookSubscriptionInput) (entities.WebhookSubscription, error) {
	webhook, err := s.whsRepo.Create(ctx, entities.WebhookSubscription{
		OrgID:     input.OrgId,
		Id:        lib.GenerateId("webhook"),
		Events:    input.Events,
		URL:       input.Url,
		Secret:    input.Secret,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	})
	if err != nil {
		return entities.WebhookSubscription{}, err
	}

	_ = s.pubsub.Publish(input.OrgId, topic.WebhookSubscriptionCreated, webhook)

	return webhook, nil
}
