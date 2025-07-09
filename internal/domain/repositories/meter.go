package repositories

import (
	"context"
	"payloop/internal/domain/entities"
	"payloop/internal/application/dto"
)

// MeterRepository defines the interface for meter persistence operations
type MeterRepository interface {
	// Create creates a new meter
	Create(ctx context.Context, meter entities.Meter) (entities.Meter, error)

	// Update updates an existing meter
	Update(ctx context.Context, meter entities.Meter) (entities.Meter, error)

	// FindById finds a meter by ID
	FindById(ctx context.Context, orgId, meterId string) (entities.Meter, error)

	// FindBySlug finds a meter by slug
	FindBySlug(ctx context.Context, orgId, slug string) (entities.Meter, error)

	// FindByEventName finds meters by event name
	FindByEventName(ctx context.Context, orgId, eventName string) ([]entities.Meter, error)

	// List lists all meters for an organization with pagination
	List(ctx context.Context, orgId string, pagination dto.Pagination) (dto.PaginatedResult[entities.Meter], error)

	// Delete deletes a meter
	Delete(ctx context.Context, orgId, meterId string) error
}