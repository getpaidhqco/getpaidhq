package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"time"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

// outgoingWebhookTimeout bounds the total time spent on a single
// delivery attempt. Customer endpoints that hang would otherwise pin
// worker goroutines indefinitely.
const outgoingWebhookTimeout = 30 * time.Second

// outgoingWebhookMaxResponseBytes caps the response body we read back.
// We don't actually use the response, but we must drain it for
// keep-alive — a malicious endpoint streaming /dev/urandom would
// exhaust memory.
const outgoingWebhookMaxResponseBytes = 64 * 1024

type WebhookSubscriptionService struct {
	logger          port.Logger
	idempotencyRepo port.IdempotencyKeyRepository
	whsRepo         port.WebhookSubscriptionRepository
	pubsub          port.PubSub

	// httpClient is built once with the SSRF-safe DialContext and a
	// hard total timeout. Reused across deliveries so we benefit from
	// keep-alive against well-behaved tenant endpoints.
	httpClient *http.Client

	// ipPredicate decides which IPs are OK to dial. Defaults to
	// isPublicUnicast (rejects loopback / private / link-local).
	// Tests replace this with allowAllIPs since httptest binds to
	// 127.0.0.1 — the predicate is package-private so the gate can
	// never be relaxed from outside the package.
	ipPredicate func(net.IP) bool
}

func NewWebhookSubscriptionService(
	logger port.Logger,
	whsRepo port.WebhookSubscriptionRepository,
	idempotencyRepo port.IdempotencyKeyRepository,
	pubsub port.PubSub,
) *WebhookSubscriptionService {
	s := &WebhookSubscriptionService{
		logger:          logger,
		whsRepo:         whsRepo,
		pubsub:          pubsub,
		idempotencyRepo: idempotencyRepo,
		ipPredicate:     isPublicUnicast,
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		DialContext:           safeDialContextWith(dialer, func() func(net.IP) bool { return s.ipPredicate }),
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConns:          50,
		IdleConnTimeout:       90 * time.Second,
	}
	s.httpClient = &http.Client{
		Transport: transport,
		Timeout:   outgoingWebhookTimeout,
	}
	return s
}

func (s *WebhookSubscriptionService) Create(ctx context.Context, input port.CreateWebhookSubscriptionInput) (domain.WebhookSubscription, error) {
	// Refuse to STORE a webhook URL that points at internal infra. The
	// runtime DialContext also re-checks at send time (DNS rebinding
	// defense) but rejecting at create gives the customer a useful
	// 4xx instead of silent delivery failures days later.
	if err := validateOutgoingWebhookURL(ctx, input.Url, s.ipPredicate); err != nil {
		s.logger.Warn("rejected webhook URL at create", "orgId", input.OrgId, "err", err.Error())
		return domain.WebhookSubscription{}, err
	}

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

	if pubErr := s.pubsub.Publish(input.OrgId, port.TopicWebhookSubscriptionCreated, webhook); pubErr != nil {
		// Pubsub is best-effort here — the row exists, the API returns
		// success, but observers won't get the create event. Log
		// loudly so it surfaces in alerts.
		s.logger.Error("failed to publish webhook-created event", "err", pubErr.Error(), "webhookId", webhook.Id)
	}

	return webhook, nil
}

func (s *WebhookSubscriptionService) SendWebhook(ctx context.Context, input port.OutgoingWebhookPayload) error {
	webhook := input.WebhookSubscription

	// Validate again at send — a URL that was safe at registration
	// might point at a different IP now (or the customer might've
	// pivoted their DNS). The DialContext below also enforces this at
	// the socket level, but doing it here gives us a cleaner error.
	if err := validateOutgoingWebhookURL(ctx, webhook.URL, s.ipPredicate); err != nil {
		s.logger.Warn("refusing to deliver to unsafe webhook URL", "webhookId", webhook.Id, "err", err.Error())
		return err
	}

	jsonData, err := json.Marshal(input.Event)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook.URL, bytes.NewReader(jsonData))
	if err != nil {
		return err
	}

	if webhook.Secret != "" {
		req.Header.Set("X-Signature", generateHMACSignature(jsonData, webhook.Secret))
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Timestamp", time.Now().UTC().Format(time.RFC3339))

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Error("outgoing webhook delivery failed", "webhookId", webhook.Id, "err", err.Error())
		return err
	}
	defer func() {
		// Drain capped to avoid memory blowup from a hostile endpoint,
		// but still drain so the connection can return to the pool.
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, outgoingWebhookMaxResponseBytes))
		_ = resp.Body.Close()
	}()

	// Note: status is just logged. Caller (dunning runner / outgoing
	// webhook workflow) owns retry policy.
	s.logger.Info("webhook delivered", "webhookId", webhook.Id, "status", resp.StatusCode)
	return nil
}

func generateHMACSignature(data []byte, secretKey string) string {
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}
