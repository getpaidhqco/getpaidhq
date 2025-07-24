package repositories

import (
	"context"
	"payloop/internal/application/dto"
	"payloop/internal/domain/entities"
)

// DiscountRepository defines the interface for discount persistence operations
type DiscountRepository interface {
	// FindById retrieves a discount by its ID
	FindById(ctx context.Context, orgId string, id string) (entities.Discount, error)

	// FindByCode retrieves a discount by its code (case-insensitive)
	FindByCode(ctx context.Context, orgId string, code string) (entities.Discount, error)

	// Create creates a new discount
	Create(ctx context.Context, discount entities.Discount) (entities.Discount, error)

	// Update updates an existing discount
	Update(ctx context.Context, discount entities.Discount) (entities.Discount, error)

	// Delete deletes a discount
	Delete(ctx context.Context, orgId string, id string) error

	// List retrieves a paginated list of discounts
	List(ctx context.Context, orgId string, pagination dto.Pagination) ([]entities.Discount, int, error)

	// ListActive retrieves a paginated list of active discounts
	ListActive(ctx context.Context, orgId string, pagination dto.Pagination) ([]entities.Discount, int, error)

	// CountRedemptions counts the number of redemptions for a discount
	CountRedemptions(ctx context.Context, orgId string, discountId string) (int, error)
}
