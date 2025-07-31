package services

import (
	"context"
	"encoding/json"
	"fmt"
	"payloop/internal/application/dto"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/events/topic"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
	"strings"
	"time"
)

type DiscountService struct {
	discountRepository          repositories.DiscountRepository
	discountRedemptionRepository repositories.DiscountRedemptionRepository
	pubsub                      events.NotificationPublisher
	logger                      logger.Logger
}

func NewDiscountService(
	discountRepository repositories.DiscountRepository,
	discountRedemptionRepository repositories.DiscountRedemptionRepository,
	pubsub events.NotificationPublisher,
	logger logger.Logger,
) interfaces.DiscountService {
	return &DiscountService{
		discountRepository:          discountRepository,
		discountRedemptionRepository: discountRedemptionRepository,
		pubsub:                      pubsub,
		logger:                      logger,
	}
}

// Discount CRUD operations
func (s *DiscountService) GetDiscount(ctx context.Context, orgId string, id string) (entities.Discount, error) {
	return s.discountRepository.FindById(ctx, orgId, id)
}

func (s *DiscountService) ListDiscounts(ctx context.Context, orgId string, pagination dto.Pagination) (dto.PaginatedResult[entities.Discount], error) {
	discounts, total, err := s.discountRepository.List(ctx, orgId, pagination)
	if err != nil {
		s.logger.Error("failed to list discounts", err)
		return dto.PaginatedResult[entities.Discount]{}, lib.NewCustomError(lib.InternalError, "Error listing discounts", err)
	}

	result := dto.PaginatedResult[entities.Discount]{
		Items:      discounts,
		TotalCount: total,
		Page:       pagination.Page,
		PageSize:   pagination.Limit,
	}

	return result, nil
}

func (s *DiscountService) CreateDiscount(ctx context.Context, orgId string, input dto.CreateDiscountInput) (entities.Discount, error) {
	// Validate input
	if err := s.validateDiscountInput(input); err != nil {
		return entities.Discount{}, err
	}

	// Normalize code to uppercase
	code := strings.ToUpper(input.Code)

	// Check if code already exists
	if code != "" {
		_, err := s.discountRepository.FindByCode(ctx, orgId, code)
		if err == nil {
			return entities.Discount{}, lib.NewCustomError(lib.ValidationError, "Discount code already exists", nil)
		}
	}

	// Parse dates
	var startsAt, endsAt time.Time
	var err error

	if input.StartsAt != "" {
		startsAt, err = time.Parse(time.RFC3339, input.StartsAt)
		if err != nil {
			return entities.Discount{}, lib.NewCustomError(lib.ValidationError, "Invalid starts_at date format", err)
		}
	}

	if input.EndsAt != "" {
		endsAt, err = time.Parse(time.RFC3339, input.EndsAt)
		if err != nil {
			return entities.Discount{}, lib.NewCustomError(lib.ValidationError, "Invalid ends_at date format", err)
		}
	}

	// Validate date range
	if !startsAt.IsZero() && !endsAt.IsZero() && endsAt.Before(startsAt) {
		return entities.Discount{}, lib.NewCustomError(lib.ValidationError, "ends_at must be after starts_at", nil)
	}

	// Convert metadata to JSON
	var metadataJson json.RawMessage
	if input.Metadata != nil {
		metadataBytes, err := json.Marshal(input.Metadata)
		if err != nil {
			return entities.Discount{}, lib.NewCustomError(lib.InternalError, "Failed to marshal metadata", err)
		}
		metadataJson = metadataBytes
	}

	// Create discount
	discount := entities.Discount{
		Id:             lib.GenerateId("disc"),
		OrgId:          orgId,
		Name:           input.Name,
		Type:           input.Type,
		Value:          input.Value,
		Code:           code,
		StartsAt:       startsAt,
		EndsAt:         endsAt,
		MaxRedemptions: input.MaxRedemptions,
		Recurring:      input.Recurring,
		Cycles:         input.Cycles,
		Currency:       input.Currency,
		Active:         input.Active,
		Metadata:       metadataJson,
	}

	createdDiscount, err := s.discountRepository.Create(ctx, discount)
	if err != nil {
		s.logger.Error("failed to create discount", err)
		return entities.Discount{}, err
	}

	// Publish event
	_ = s.pubsub.Publish(orgId, topic.DiscountCreated, createdDiscount)

	return createdDiscount, nil
}

func (s *DiscountService) UpdateDiscount(ctx context.Context, orgId string, id string, input dto.UpdateDiscountInput) (entities.Discount, error) {
	// Get existing discount
	discount, err := s.discountRepository.FindById(ctx, orgId, id)
	if err != nil {
		return entities.Discount{}, lib.NewCustomError(lib.NotFoundError, "Discount not found", err)
	}

	// Update fields if provided
	if input.Name != "" {
		discount.Name = input.Name
	}

	if input.Type != "" {
		discount.Type = input.Type
	}

	if input.Value != nil {
		discount.Value = *input.Value
	}

	if input.Code != "" {
		// Normalize code to uppercase
		code := strings.ToUpper(input.Code)

		// Check if code already exists and is not the current discount
		if code != discount.Code {
			existingDiscount, err := s.discountRepository.FindByCode(ctx, orgId, code)
			if err == nil && existingDiscount.Id != id {
				return entities.Discount{}, lib.NewCustomError(lib.ValidationError, "Discount code already exists", nil)
			}
		}

		discount.Code = code
	}

	// Parse dates
	if input.StartsAt != "" {
		startsAt, err := time.Parse(time.RFC3339, input.StartsAt)
		if err != nil {
			return entities.Discount{}, lib.NewCustomError(lib.ValidationError, "Invalid starts_at date format", err)
		}
		discount.StartsAt = startsAt
	}

	if input.EndsAt != "" {
		endsAt, err := time.Parse(time.RFC3339, input.EndsAt)
		if err != nil {
			return entities.Discount{}, lib.NewCustomError(lib.ValidationError, "Invalid ends_at date format", err)
		}
		discount.EndsAt = endsAt
	}

	// Validate date range
	if !discount.StartsAt.IsZero() && !discount.EndsAt.IsZero() && discount.EndsAt.Before(discount.StartsAt) {
		return entities.Discount{}, lib.NewCustomError(lib.ValidationError, "ends_at must be after starts_at", nil)
	}

	if input.MaxRedemptions != nil {
		discount.MaxRedemptions = *input.MaxRedemptions
	}

	if input.Recurring != "" {
		discount.Recurring = input.Recurring
	}

	if input.Cycles != nil {
		discount.Cycles = *input.Cycles
	}

	if input.Currency != "" {
		discount.Currency = input.Currency
	}

	if input.Active != nil {
		discount.Active = *input.Active
	}

	// Update metadata if provided
	if input.Metadata != nil {
		metadataBytes, err := json.Marshal(input.Metadata)
		if err != nil {
			return entities.Discount{}, lib.NewCustomError(lib.InternalError, "Failed to marshal metadata", err)
		}
		discount.Metadata = metadataBytes
	}

	// Validate the updated discount
	if err := s.validateDiscount(discount); err != nil {
		return entities.Discount{}, err
	}

	// Update discount
	updatedDiscount, err := s.discountRepository.Update(ctx, discount)
	if err != nil {
		s.logger.Error("failed to update discount", err)
		return entities.Discount{}, err
	}

	// Publish event
	_ = s.pubsub.Publish(orgId, topic.DiscountUpdated, updatedDiscount)

	return updatedDiscount, nil
}

func (s *DiscountService) DeleteDiscount(ctx context.Context, orgId string, id string) error {
	// Get discount to publish event
	discount, err := s.discountRepository.FindById(ctx, orgId, id)
	if err != nil {
		return lib.NewCustomError(lib.NotFoundError, "Discount not found", err)
	}

	// Delete discount
	err = s.discountRepository.Delete(ctx, orgId, id)
	if err != nil {
		s.logger.Error("failed to delete discount", err)
		return err
	}

	// Publish event
	_ = s.pubsub.Publish(orgId, topic.DiscountDeleted, discount)

	return nil
}

// Discount code validation
func (s *DiscountService) ValidateDiscountCode(ctx context.Context, orgId string, input dto.ValidateDiscountCodeInput) (dto.DiscountValidationResult, error) {
	// Normalize code to uppercase
	code := strings.ToUpper(input.Code)

	// Find discount by code
	discount, err := s.discountRepository.FindByCode(ctx, orgId, code)
	if err != nil {
		return dto.DiscountValidationResult{
			Valid:   false,
			Message: "Invalid discount code",
		}, nil
	}

	// Check if discount is valid
	if !discount.IsValid() {
		return dto.DiscountValidationResult{
			Valid:   false,
			Message: "Discount is not active or has expired",
		}, nil
	}

	// Check redemption limits
	if discount.MaxRedemptions > 0 {
		count, err := s.discountRepository.CountRedemptions(ctx, orgId, discount.Id)
		if err != nil {
			s.logger.Error("failed to count redemptions", err)
			return dto.DiscountValidationResult{}, lib.NewCustomError(lib.InternalError, "Error validating discount code", err)
		}

		if count >= discount.MaxRedemptions {
			return dto.DiscountValidationResult{
				Valid:   false,
				Message: "Discount code has reached maximum redemptions",
			}, nil
		}
	}

	// Check customer-specific redemption limits if customer ID is provided
	if input.CustomerId != "" {
		count, err := s.discountRedemptionRepository.CountByCustomerAndDiscount(ctx, orgId, input.CustomerId, discount.Id)
		if err != nil {
			s.logger.Error("failed to count customer redemptions", err)
			return dto.DiscountValidationResult{}, lib.NewCustomError(lib.InternalError, "Error validating discount code", err)
		}

		if count > 0 && discount.Recurring == "once" {
			return dto.DiscountValidationResult{
				Valid:   false,
				Message: "Discount code has already been used by this customer",
			}, nil
		}
	}

	// Calculate discount amount if amount and currency are provided
	var discountAmount int
	if input.Amount > 0 && input.Currency != "" {
		// For fixed discounts, check currency match
		if discount.IsFixedAmount() && discount.Currency != input.Currency {
			return dto.DiscountValidationResult{
				Valid:   false,
				Message: fmt.Sprintf("Discount currency (%s) does not match transaction currency (%s)", discount.Currency, input.Currency),
			}, nil
		}

		discountAmount = discount.CalculateDiscountAmount(input.Amount)
	}

	return dto.DiscountValidationResult{
		Valid:          true,
		DiscountId:     discount.Id,
		DiscountAmount: discountAmount,
	}, nil
}

// Discount application
func (s *DiscountService) ApplyDiscount(ctx context.Context, orgId string, input dto.ApplyDiscountInput) (dto.DiscountRedemptionResult, error) {
	// Get discount
	discount, err := s.discountRepository.FindById(ctx, orgId, input.DiscountId)
	if err != nil {
		return dto.DiscountRedemptionResult{}, lib.NewCustomError(lib.NotFoundError, "Discount not found", err)
	}

	// Validate discount
	if !discount.IsValid() {
		return dto.DiscountRedemptionResult{}, lib.NewCustomError(lib.ValidationError, "Discount is not active or has expired", nil)
	}

	// Check redemption limits
	if discount.MaxRedemptions > 0 {
		count, err := s.discountRepository.CountRedemptions(ctx, orgId, discount.Id)
		if err != nil {
			s.logger.Error("failed to count redemptions", err)
			return dto.DiscountRedemptionResult{}, lib.NewCustomError(lib.InternalError, "Error applying discount", err)
		}

		if count >= discount.MaxRedemptions {
			return dto.DiscountRedemptionResult{}, lib.NewCustomError(lib.ValidationError, "Discount has reached maximum redemptions", nil)
		}
	}

	// Check customer-specific redemption limits
	count, err := s.discountRedemptionRepository.CountByCustomerAndDiscount(ctx, orgId, input.CustomerId, discount.Id)
	if err != nil {
		s.logger.Error("failed to count customer redemptions", err)
		return dto.DiscountRedemptionResult{}, lib.NewCustomError(lib.InternalError, "Error applying discount", err)
	}

	if count > 0 && discount.Recurring == "once" {
		return dto.DiscountRedemptionResult{}, lib.NewCustomError(lib.ValidationError, "Discount has already been used by this customer", nil)
	}

	// For fixed discounts, check currency match
	if discount.IsFixedAmount() && discount.Currency != input.Currency {
		return dto.DiscountRedemptionResult{}, lib.NewCustomError(lib.ValidationError, 
			fmt.Sprintf("Discount currency (%s) does not match transaction currency (%s)", discount.Currency, input.Currency), nil)
	}

	// Calculate discount amount
	discountAmount := discount.CalculateDiscountAmount(input.Amount)
	discountedAmount := input.Amount - discountAmount

	// Convert metadata to JSON
	var metadataJson json.RawMessage
	if input.Metadata != nil {
		metadataBytes, err := json.Marshal(input.Metadata)
		if err != nil {
			return dto.DiscountRedemptionResult{}, lib.NewCustomError(lib.InternalError, "Failed to marshal metadata", err)
		}
		metadataJson = metadataBytes
	}

	// Create redemption record
	redemption := entities.DiscountRedemption{
		Id:             lib.GenerateId("dred"),
		OrgId:          orgId,
		DiscountId:     discount.Id,
		CustomerId:     input.CustomerId,
		ResourceType:   input.ResourceType,
		ResourceId:     input.ResourceId,
		DiscountAmount: discountAmount,
		Currency:       input.Currency,
		Metadata:       metadataJson,
	}

	createdRedemption, err := s.discountRedemptionRepository.Create(ctx, redemption)
	if err != nil {
		s.logger.Error("failed to create discount redemption", err)
		return dto.DiscountRedemptionResult{}, err
	}

	// Publish event
	_ = s.pubsub.Publish(orgId, topic.DiscountApplied, createdRedemption)

	return dto.DiscountRedemptionResult{
		RedemptionId:     createdRedemption.Id,
		DiscountId:       discount.Id,
		DiscountAmount:   discountAmount,
		OriginalAmount:   input.Amount,
		DiscountedAmount: discountedAmount,
		Currency:         input.Currency,
	}, nil
}

// Discount redemption operations
func (s *DiscountService) GetDiscountRedemption(ctx context.Context, orgId string, id string) (entities.DiscountRedemption, error) {
	return s.discountRedemptionRepository.FindById(ctx, orgId, id)
}

func (s *DiscountService) ListDiscountRedemptions(ctx context.Context, orgId string, discountId string, pagination dto.Pagination) (dto.PaginatedResult[entities.DiscountRedemption], error) {
	redemptions, total, err := s.discountRedemptionRepository.ListByDiscount(ctx, orgId, discountId, pagination)
	if err != nil {
		s.logger.Error("failed to list discount redemptions", err)
		return dto.PaginatedResult[entities.DiscountRedemption]{}, lib.NewCustomError(lib.InternalError, "Error listing discount redemptions", err)
	}

	result := dto.PaginatedResult[entities.DiscountRedemption]{
		Items:      redemptions,
		TotalCount: total,
		Page:       pagination.Page,
		PageSize:   pagination.Limit,
	}

	return result, nil
}

func (s *DiscountService) ListCustomerRedemptions(ctx context.Context, orgId string, customerId string, pagination dto.Pagination) (dto.PaginatedResult[entities.DiscountRedemption], error) {
	redemptions, total, err := s.discountRedemptionRepository.ListByCustomer(ctx, orgId, customerId, pagination)
	if err != nil {
		s.logger.Error("failed to list customer redemptions", err)
		return dto.PaginatedResult[entities.DiscountRedemption]{}, lib.NewCustomError(lib.InternalError, "Error listing customer redemptions", err)
	}

	result := dto.PaginatedResult[entities.DiscountRedemption]{
		Items:      redemptions,
		TotalCount: total,
		Page:       pagination.Page,
		PageSize:   pagination.Limit,
	}

	return result, nil
}

// Helper methods
func (s *DiscountService) validateDiscountInput(input dto.CreateDiscountInput) error {
	// Check required fields
	if input.Name == "" {
		return lib.NewCustomError(lib.ValidationError, "Name is required", nil)
	}

	if input.Type == "" {
		return lib.NewCustomError(lib.ValidationError, "Type is required", nil)
	}

	if input.Type != "fixed" && input.Type != "percentage" {
		return lib.NewCustomError(lib.ValidationError, "Type must be 'fixed' or 'percentage'", nil)
	}

	if input.Recurring == "" {
		return lib.NewCustomError(lib.ValidationError, "Recurring is required", nil)
	}

	if input.Recurring != "once" && input.Recurring != "forever" && input.Recurring != "cycles" {
		return lib.NewCustomError(lib.ValidationError, "Recurring must be 'once', 'forever', or 'cycles'", nil)
	}

	// Validate percentage value
	if input.Type == "percentage" && (input.Value < 0 || input.Value > 100) {
		return lib.NewCustomError(lib.ValidationError, "Percentage value must be between 0 and 100", nil)
	}

	// Validate fixed discount
	if input.Type == "fixed" && input.Currency == "" {
		return lib.NewCustomError(lib.ValidationError, "Currency is required for fixed discounts", nil)
	}

	// Validate cycles
	if input.Recurring == "cycles" && input.Cycles <= 0 {
		return lib.NewCustomError(lib.ValidationError, "Cycles must be greater than 0", nil)
	}

	return nil
}

func (s *DiscountService) validateDiscount(discount entities.Discount) error {
	// Check required fields
	if discount.Name == "" {
		return lib.NewCustomError(lib.ValidationError, "Name is required", nil)
	}

	if discount.Type == "" {
		return lib.NewCustomError(lib.ValidationError, "Type is required", nil)
	}

	if discount.Type != "fixed" && discount.Type != "percentage" {
		return lib.NewCustomError(lib.ValidationError, "Type must be 'fixed' or 'percentage'", nil)
	}

	if discount.Recurring == "" {
		return lib.NewCustomError(lib.ValidationError, "Recurring is required", nil)
	}

	if discount.Recurring != "once" && discount.Recurring != "forever" && discount.Recurring != "cycles" {
		return lib.NewCustomError(lib.ValidationError, "Recurring must be 'once', 'forever', or 'cycles'", nil)
	}

	// Validate percentage value
	if discount.Type == "percentage" && (discount.Value < 0 || discount.Value > 100) {
		return lib.NewCustomError(lib.ValidationError, "Percentage value must be between 0 and 100", nil)
	}

	// Validate fixed discount
	if discount.Type == "fixed" && discount.Currency == "" {
		return lib.NewCustomError(lib.ValidationError, "Currency is required for fixed discounts", nil)
	}

	// Validate cycles
	if discount.Recurring == "cycles" && discount.Cycles <= 0 {
		return lib.NewCustomError(lib.ValidationError, "Cycles must be greater than 0", nil)
	}

	return nil
}
