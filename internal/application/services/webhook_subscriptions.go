package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/events/topic"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/webhooks"
	"payloop/internal/domain/repositories"
	"payloop/internal/domain/workflow"
	"payloop/internal/lib"
	"time"
)

type WebhookSubscriptionService struct {
	logger          logger.Logger
	idempotencyRepo repositories.IdempotencyKeyRepository
	whsRepo         repositories.WebhookSubscriptionRepository
	pubsub          events.PubSub
}

func NewWebhookSubscriptionService(
	logger logger.Logger,
	whsRepo repositories.WebhookSubscriptionRepository,
	idempotencyRepo repositories.IdempotencyKeyRepository,
	pubsub events.PubSub,
) interfaces.WebhookSubscriptionService {
	service := WebhookSubscriptionService{
		logger:          logger,
		whsRepo:         whsRepo,
		pubsub:          pubsub,
		idempotencyRepo: idempotencyRepo,
	}

	return service
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

func (s WebhookSubscriptionService) SendWebhook(ctx context.Context, input workflow.OutgoingWebhookPayload) error {

	webhook := input.WebhookSubscription
	// Convert the data to JSON
	jsonData, err := json.Marshal(input.Event)
	if err != nil {
		s.logger.Errorf("Failed to marshal JSON: %v", err)
		return err
	}

	// Create a new POST request with the JSON data
	req, err := http.NewRequest("POST", webhook.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		s.logger.Errorf("Failed to create new request: %v", err)
		return err
	}

	if webhook.Secret != "" {
		signature := generateHMACSignature(jsonData, webhook.Secret)
		req.Header.Set("X-Signature", signature)
	}

	req = req.WithContext(ctx)
	// Set the content type to application/json
	req.Header.Set("Content-Type", "application/json")
	// Add a timestamp to the request header
	timestamp := time.Now().UTC().Format(time.RFC3339)
	req.Header.Set("X-Timestamp", timestamp)

	// Send the POST request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		s.logger.Errorf("Failed to send request: %v", err.Error())
		return err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	// Print the response status
	s.logger.Infof("Webhook sent to %s. Response Status: %s", webhook.URL, resp.Status)
	return nil
}

func generateHMACSignature(data []byte, secretKey string) string {
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}
