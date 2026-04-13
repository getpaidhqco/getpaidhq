package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"payloop/internal/core/domain"
	"payloop/internal/core/port"
	"payloop/internal/lib"
	"time"
)

type WebhookSubscriptionService struct {
	logger          port.Logger
	idempotencyRepo port.IdempotencyKeyRepository
	whsRepo         port.WebhookSubscriptionRepository
	pubsub          port.PubSub
}

func NewWebhookSubscriptionService(
	logger port.Logger,
	whsRepo port.WebhookSubscriptionRepository,
	idempotencyRepo port.IdempotencyKeyRepository,
	pubsub port.PubSub,
) *WebhookSubscriptionService {
	return &WebhookSubscriptionService{
		logger:          logger,
		whsRepo:         whsRepo,
		pubsub:          pubsub,
		idempotencyRepo: idempotencyRepo,
	}
}

func (s *WebhookSubscriptionService) Create(ctx context.Context, input domain.CreateWebhookSubscriptionInput) (domain.WebhookSubscription, error) {
	webhook, err := s.whsRepo.Create(ctx, domain.WebhookSubscription{
		OrgID:     input.OrgId,
		Id:        lib.GenerateId("webhook"),
		Events:    input.Events,
		URL:       input.Url,
		Secret:    input.Secret,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	})
	if err != nil {
		return domain.WebhookSubscription{}, err
	}

	_ = s.pubsub.Publish(input.OrgId, port.TopicWebhookSubscriptionCreated, webhook)

	return webhook, nil
}

func (s *WebhookSubscriptionService) SendWebhook(ctx context.Context, input port.OutgoingWebhookPayload) error {

	webhook := input.WebhookSubscription
	// Convert the data to JSON
	jsonData, err := json.Marshal(input.Event)
	if err != nil {
		s.logger.Error("failed to marshal json", "error", err)
		return err
	}

	// Create a new POST request with the JSON data
	req, err := http.NewRequest("POST", webhook.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		s.logger.Error("failed to create new request", "error", err)
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
		s.logger.Error("failed to send request", "error", err)
		return err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	// Print the response status
	s.logger.Info("webhook sent", "url", webhook.URL, "status", resp.Status)
	return nil
}

func generateHMACSignature(data []byte, secretKey string) string {
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}
