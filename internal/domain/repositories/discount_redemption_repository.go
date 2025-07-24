package repositories

import (
	"context"
	"payloop/internal/application/dto"
	"payloop/internal/domain/entities"
)

// DiscountRedemptionRepository defines the interface for discount redemption persistence operations
type DiscountRedemptionRepository interface {
	// FindById retrieves a discount redemption by its ID
	FindById(ctx context.Context, orgId string, id string) (entities.DiscountRedemption, error)

	// Create creates a new discount redemption
	Create(ctx context.Context, redemption entities.DiscountRedemption) (entities.DiscountRedemption, error)

	// Delete deletes a discount redemption
	Delete(ctx context.Context, orgId string, id string) error

	// ListByDiscount retrieves all redemptions for a discount
	ListByDiscount(ctx context.Context, orgId string, discountId string, pagination dto.Pagination) ([]entities.DiscountRedemption, int, error)

	// ListByCustomer retrieves all redemptions for a customer
	ListByCustomer(ctx context.Context, orgId string, customerId string, pagination dto.Pagination) ([]entities.DiscountRedemption, int, error)

	// ListByResource retrieves all redemptions for a resource
	ListByResource(ctx context.Context, orgId string, resourceType string, resourceId string) ([]entities.DiscountRedemption, error)

	// CountByDiscount counts the number of redemptions for a discount
	CountByDiscount(ctx context.Context, orgId string, discountId string) (int, error)

	// CountByCustomerAndDiscount counts the number of redemptions for a customer and discount
	CountByCustomerAndDiscount(ctx context.Context, orgId string, customerId string, discountId string) (int, error)
}
