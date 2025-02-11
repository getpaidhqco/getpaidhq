package services

import (
	"context"
	"payloop/internal/application/lib/events"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/webhooks"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
	"time"
)

type WebhookSubscriptionService struct {
	logger          lib.Logger
	idempotencyRepo repositories.IdempotencyKeyRepository
	whsRepo         repositories.WebhookSubscriptionRepository
	pubsub          events.PubSub
}

func NewWebhookSubscriptionService(
	logger lib.Logger,
	whsRepo repositories.WebhookSubscriptionRepository,
	idempotencyRepo repositories.IdempotencyKeyRepository,
	pubsub events.PubSub,
) WebhookSubscriptionService {
	service := WebhookSubscriptionService{
		logger:          logger,
		whsRepo:         whsRepo,
		pubsub:          pubsub,
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

	_ = s.pubsub.PublishJSON(events.WebhookSubscriptionCreated, webhook)

	return webhook, nil
}
