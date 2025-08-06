package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"payloop/internal/api/dto/request"
	"payloop/internal/application/dto"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/events/topic"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/payment_links"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
	"time"
)

type PaymentLinkService struct {
	paymentLinkRepository      repositories.PaymentLinkRepository
	paymentLinkUsageRepository repositories.PaymentLinkUsageRepository
	pubsub                     events.NotificationPublisher
	logger                     logger.Logger
}

func NewPaymentLinkService(
	paymentLinkRepository repositories.PaymentLinkRepository,
	paymentLinkUsageRepository repositories.PaymentLinkUsageRepository,
	pubsub events.NotificationPublisher,
	logger logger.Logger,
) interfaces.PaymentLinkService {
	return PaymentLinkService{
		paymentLinkRepository:      paymentLinkRepository,
		paymentLinkUsageRepository: paymentLinkUsageRepository,
		pubsub:                     pubsub,
		logger:                     logger,
	}
}

// generateSecureToken generates a cryptographically secure 32-byte token
func (s PaymentLinkService) generateSecureToken() (string, string, error) {
	// Generate 32 random bytes
	tokenBytes := make([]byte, 32)
	_, err := rand.Read(tokenBytes)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate secure token: %w", err)
	}

	// Convert to hex string for the token
	token := hex.EncodeToString(tokenBytes)

	// Generate SHA256 hash for storage
	hasher := sha256.New()
	hasher.Write([]byte(token))
	tokenHash := hex.EncodeToString(hasher.Sum(nil))

	return token, tokenHash, nil
}

// Payment Link CRUD operations
func (s PaymentLinkService) GetPaymentLink(ctx context.Context, orgId string, id string) (entities.PaymentLink, error) {
	return s.paymentLinkRepository.FindById(ctx, orgId, id)
}

func (s PaymentLinkService) GetPaymentLinkBySlug(ctx context.Context, slug string) (entities.PaymentLink, error) {
	return s.paymentLinkRepository.FindBySlug(ctx, slug)
}

func (s PaymentLinkService) ListPaymentLinks(ctx context.Context, orgId string, pagination dto.Pagination) (dto.PaginatedResult[entities.PaymentLink], error) {
	// Convert application DTO pagination to request pagination
	requestPagination := request.Pagination{
		Page:          pagination.Page,
		Limit:         pagination.Limit,
		Offset:        pagination.Offset,
		SortDirection: pagination.SortDirection,
		SortBy:        pagination.SortBy,
	}

	paymentLinks, total, err := s.paymentLinkRepository.List(ctx, orgId, requestPagination)
	if err != nil {
		s.logger.Error("failed to list payment links", err)
		return dto.PaginatedResult[entities.PaymentLink]{}, lib.NewCustomError(lib.InternalError, "Error listing payment links", err)
	}

	result := dto.PaginatedResult[entities.PaymentLink]{
		Items:      paymentLinks,
		TotalCount: total,
		Page:       pagination.Page,
		PageSize:   pagination.Limit,
	}

	return result, nil
}

func (s PaymentLinkService) CreatePaymentLink(ctx context.Context, orgId string, input payment_links.CreatePaymentLinkInput) (payment_links.PaymentLinkCreationResult, error) {
	// Parse expires_at if provided
	var expiresAt time.Time
	if input.ExpiresAt != "" {
		parsedExpiresAt, err := time.Parse(time.RFC3339, input.ExpiresAt)
		if err != nil {
			return payment_links.PaymentLinkCreationResult{}, fmt.Errorf("invalid expires_at format: %w", err)
		}
		expiresAt = parsedExpiresAt
	}

	// Generate secure token and hash for token-protected access
	token, tokenHash, err := s.generateSecureToken()
	if err != nil {
		s.logger.Error("failed to generate secure token", err)
		return payment_links.PaymentLinkCreationResult{}, fmt.Errorf("failed to generate secure token: %w", err)
	}

	// Create payment link
	paymentLink, err := s.paymentLinkRepository.Create(ctx, entities.PaymentLink{
		OrgId:     orgId,
		Id:        lib.GenerateId("pay"),
		Slug:      input.Slug,
		Data:      input.Data,
		Config:    input.Config,
		SingleUse: input.SingleUse,
		Status:    "active", // Default status
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	})

	if err != nil {
		s.logger.Error("failed to create payment link", err)
		return payment_links.PaymentLinkCreationResult{}, err
	}

	// Log the generated token for the user (in production, this should be returned to the client)
	s.logger.Info("Payment link created with secure token", "payment_link_id", paymentLink.Id, "token", token)

	// Publish event
	_ = s.pubsub.Publish(orgId, topic.PaymentLinkCreated, paymentLink)

	// Return both the payment link and the token
	return payment_links.PaymentLinkCreationResult{
		PaymentLink: paymentLink,
		Token:       token,
	}, nil
}

func (s PaymentLinkService) UpdatePaymentLink(ctx context.Context, orgId string, id string, input payment_links.UpdatePaymentLinkInput) (entities.PaymentLink, error) {
	// Get existing payment link
	paymentLink, err := s.paymentLinkRepository.FindById(ctx, orgId, id)
	if err != nil {
		return entities.PaymentLink{}, fmt.Errorf("payment link not found: %w", err)
	}

	// Update fields if provided
	if input.Slug != "" {
		paymentLink.Slug = input.Slug
	}

	if input.Data != nil {
		paymentLink.Data = input.Data
	}

	if input.Config != nil {
		paymentLink.Config = input.Config
	}

	paymentLink.SingleUse = input.SingleUse

	if input.Status != "" {
		paymentLink.Status = input.Status
	}

	if input.ExpiresAt != "" {
		expiresAt, err := time.Parse(time.RFC3339, input.ExpiresAt)
		if err != nil {
			return entities.PaymentLink{}, fmt.Errorf("invalid expires_at format: %w", err)
		}
		paymentLink.ExpiresAt = expiresAt
	}

	// Update payment link
	updatedPaymentLink, err := s.paymentLinkRepository.Update(ctx, paymentLink)
	if err != nil {
		s.logger.Error("failed to update payment link", err)
		return entities.PaymentLink{}, err
	}

	// Publish event
	_ = s.pubsub.Publish(orgId, topic.PaymentLinkUpdated, updatedPaymentLink)

	return updatedPaymentLink, nil
}

func (s PaymentLinkService) DeletePaymentLink(ctx context.Context, orgId string, id string) error {
	// Get payment link to publish event
	paymentLink, err := s.paymentLinkRepository.FindById(ctx, orgId, id)
	if err != nil {
		return fmt.Errorf("payment link not found: %w", err)
	}

	// Delete payment link
	err = s.paymentLinkRepository.Delete(ctx, orgId, id)
	if err != nil {
		s.logger.Error("failed to delete payment link", err)
		return err
	}

	// Publish event
	_ = s.pubsub.Publish(orgId, topic.PaymentLinkDeleted, paymentLink)

	return nil
}

// Public Access operations
func (s PaymentLinkService) ValidatePaymentLinkAccess(ctx context.Context, slug, token string) (entities.PaymentLink, error) {
	// Get payment link by slug
	paymentLink, err := s.paymentLinkRepository.FindBySlug(ctx, slug)
	if err != nil {
		s.logger.Error("payment link not found", err)
		return entities.PaymentLink{}, lib.NewCustomError(lib.NotFoundError, "Payment link not found", err)
	}

	// Validate payment link status
	if paymentLink.Status != "active" {
		s.logger.Warn("payment link is not active", "slug", slug, "status", paymentLink.Status)
		return entities.PaymentLink{}, lib.NewCustomError(lib.ValidationError, "Payment link is not active", nil)
	}

	// Validate expiration
	if !paymentLink.ExpiresAt.IsZero() && time.Now().After(paymentLink.ExpiresAt) {
		s.logger.Warn("payment link is expired", "slug", slug, "expires_at", paymentLink.ExpiresAt)
		return entities.PaymentLink{}, lib.NewCustomError(lib.ValidationError, "Payment link has expired", nil)
	}

	// Validate single use
	if paymentLink.SingleUse && !paymentLink.UsedAt.IsZero() {
		s.logger.Warn("payment link has already been used", "slug", slug, "used_at", paymentLink.UsedAt)
		return entities.PaymentLink{}, lib.NewCustomError(lib.ValidationError, "Payment link has already been used", nil)
	}

	// Validate token if TokenHash is present
	if paymentLink.TokenHash != "" {
		if token == "" {
			s.logger.Warn("token required for payment link access", "slug", slug)
			return entities.PaymentLink{}, lib.NewCustomError(lib.ValidationError, "Token required for payment link access", nil)
		}

		// Hash the provided token and compare with stored hash
		hasher := sha256.New()
		hasher.Write([]byte(token))
		tokenHash := hex.EncodeToString(hasher.Sum(nil))

		if tokenHash != paymentLink.TokenHash {
			s.logger.Warn("invalid token for payment link", "slug", slug)
			return entities.PaymentLink{}, lib.NewCustomError(lib.ValidationError, "Invalid token", nil)
		}
	}

	return paymentLink, nil
}

// Payment Link Usage operations
func (s PaymentLinkService) RecordPaymentLinkUsage(ctx context.Context, orgId string, input payment_links.RecordPaymentLinkUsageInput) (entities.PaymentLinkUsage, error) {
	// Convert metadata to JSON
	var metadataJson []byte
	var err error
	if input.Metadata != nil {
		metadataJson, err = json.Marshal(input.Metadata)
		if err != nil {
			return entities.PaymentLinkUsage{}, fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	// Create payment link usage
	usage, err := s.paymentLinkUsageRepository.Create(ctx, entities.PaymentLinkUsage{
		Id:            lib.GenerateId("usage"),
		OrgId:         orgId,
		PaymentLinkId: input.PaymentLinkId,
		SessionId:     input.SessionId,
		CustomerId:    input.CustomerId,
		EventType:     input.EventType,
		IpAddress:     input.IpAddress,
		UserAgent:     input.UserAgent,
		Referer:       input.Referer,
		Country:       input.Country,
		Metadata:      metadataJson,
		Timestamp:     time.Now(),
	})

	if err != nil {
		s.logger.Error("failed to record payment link usage", err)
		return entities.PaymentLinkUsage{}, err
	}

	// Publish event
	_ = s.pubsub.Publish(orgId, topic.PaymentLinkUsageRecorded, usage)

	// If this is a payment_succeeded event and the payment link is single-use, mark it as used
	if input.EventType == "payment_succeeded" {
		paymentLink, err := s.paymentLinkRepository.FindById(ctx, orgId, input.PaymentLinkId)
		if err == nil && paymentLink.SingleUse {
			paymentLink.Status = "used"
			paymentLink.UsedAt = time.Now()
			_, _ = s.paymentLinkRepository.Update(ctx, paymentLink)
		}
	}

	return usage, nil
}

func (s PaymentLinkService) GetPaymentLinkUsage(ctx context.Context, orgId string, id string) (entities.PaymentLinkUsage, error) {
	return s.paymentLinkUsageRepository.FindById(ctx, orgId, id)
}

func (s PaymentLinkService) ListPaymentLinkUsages(ctx context.Context, orgId string, paymentLinkId string) ([]entities.PaymentLinkUsage, error) {
	return s.paymentLinkUsageRepository.ListByPaymentLinkId(ctx, orgId, paymentLinkId)
}
