package interfaces

import (
	"context"
	"payloop/internal/application/dto"
	"payloop/internal/domain/entities"
)

// MeterService defines the interface for meter operations
type MeterService interface {
	// Create creates a new meter
	Create(ctx context.Context, orgId string, input dto.CreateMeterInput) (entities.Meter, error)

	// Update updates an existing meter
	Update(ctx context.Context, orgId, meterId string, input dto.UpdateMeterInput) (entities.Meter, error)

	// Get gets a meter by ID
	Get(ctx context.Context, orgId, meterId string) (entities.Meter, error)

	// GetByEventName gets a meter by event name
	GetByEventName(ctx context.Context, orgId, eventName string) (entities.Meter, error)

	// List lists all meters for an organization with pagination
	List(ctx context.Context, orgId string, pagination dto.Pagination) (dto.PaginatedResult[entities.Meter], error)

	// Delete deletes a meter
	Delete(ctx context.Context, orgId, meterId string) error

	// ValidateEventAgainstMeter checks if an event matches the meter's configuration
	ValidateEventAgainstMeter(ctx context.Context, meter entities.Meter, event map[string]interface{}) bool
}