package interfaces

import (
	"context"
	"payloop/internal/application/dto"
	"payloop/internal/domain/entities"
)

// DiscountService defines the interface for discount operations
type DiscountService interface {
	// Discount CRUD operations
	GetDiscount(ctx context.Context, orgId string, id string) (entities.Discount, error)
	ListDiscounts(ctx context.Context, orgId string, pagination dto.Pagination) (dto.PaginatedResult[entities.Discount], error)
	CreateDiscount(ctx context.Context, orgId string, input dto.CreateDiscountInput) (entities.Discount, error)
	UpdateDiscount(ctx context.Context, orgId string, id string, input dto.UpdateDiscountInput) (entities.Discount, error)
	DeleteDiscount(ctx context.Context, orgId string, id string) error
	
	// Discount code validation
	ValidateDiscountCode(ctx context.Context, orgId string, input dto.ValidateDiscountCodeInput) (dto.DiscountValidationResult, error)
	
	// Discount application
	ApplyDiscount(ctx context.Context, orgId string, input dto.ApplyDiscountInput) (dto.DiscountRedemptionResult, error)
	
	// Discount redemption operations
	GetDiscountRedemption(ctx context.Context, orgId string, id string) (entities.DiscountRedemption, error)
	ListDiscountRedemptions(ctx context.Context, orgId string, discountId string, pagination dto.Pagination) (dto.PaginatedResult[entities.DiscountRedemption], error)
	ListCustomerRedemptions(ctx context.Context, orgId string, customerId string, pagination dto.Pagination) (dto.PaginatedResult[entities.DiscountRedemption], error)
}