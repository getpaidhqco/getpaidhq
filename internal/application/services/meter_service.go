package services

import (
	"context"
	"fmt"
	"strings"
	"payloop/internal/application/dto"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"time"
)

// MeterService implements the MeterService interface
type meterService struct {
	meterRepo repositories.MeterRepository
	logger    logger.Logger
}

// NewMeterService creates a new MeterService
func NewMeterService(meterRepo repositories.MeterRepository, logger logger.Logger) interfaces.MeterService {
	return &meterService{
		meterRepo: meterRepo,
		logger:    logger,
	}
}

// Create creates a new meter
func (s *meterService) Create(ctx context.Context, orgId string, input dto.CreateMeterInput) (entities.Meter, error) {
	// Check if event name already exists
	_, err := s.meterRepo.FindByEventName(ctx, orgId, input.EventName)
	if err == nil {
		return entities.Meter{}, fmt.Errorf("meter with event name %s already exists", input.EventName)
	}
	// If the error is "not found", we can proceed with creation
	// Otherwise, return the error
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return entities.Meter{}, err
	}

	// Create meter entity
	meter, err := entities.NewMeter(orgId, entities.CreateMeterInput{
		Name:            input.Name,
		Description:     input.Description,
		EventName:       input.EventName,
		EventFilter:     input.EventFilter,
		AggregationType: input.AggregationType,
		ValueProperty:   input.ValueProperty,
		UnitType:        input.UnitType,
		DisplayName:     input.DisplayName,
		WindowSize:      input.WindowSize,
		ResetInterval:   input.ResetInterval,
		Metadata:        input.Metadata,
	})
	if err != nil {
		return entities.Meter{}, err
	}

	// Save to repository
	return s.meterRepo.Create(ctx, meter)
}

// Update updates an existing meter
func (s *meterService) Update(ctx context.Context, orgId, meterId string, input dto.UpdateMeterInput) (entities.Meter, error) {
	// Get existing meter
	meter, err := s.meterRepo.FindById(ctx, orgId, meterId)
	if err != nil {
		return entities.Meter{}, fmt.Errorf("meter not found: %w", err)
	}

	// Update fields
	meter.Name = input.Name
	meter.Description = input.Description
	meter.EventName = input.EventName
	meter.EventFilter = input.EventFilter
	meter.AggregationType = input.AggregationType
	meter.ValueProperty = input.ValueProperty
	meter.UnitType = input.UnitType
	meter.DisplayName = input.DisplayName
	meter.WindowSize = input.WindowSize
	meter.ResetInterval = input.ResetInterval
	meter.Metadata = input.Metadata
	meter.UpdatedAt = time.Now().UTC()

	// Save to repository
	return s.meterRepo.Update(ctx, meter)
}

// Get gets a meter by ID
func (s *meterService) Get(ctx context.Context, orgId, meterId string) (entities.Meter, error) {
	return s.meterRepo.FindById(ctx, orgId, meterId)
}

// GetByEventName gets a meter by event name
func (s *meterService) GetByEventName(ctx context.Context, orgId, eventName string) (entities.Meter, error) {
	meter, err := s.meterRepo.FindByEventName(ctx, orgId, eventName)
	if err != nil {
		return entities.Meter{}, err
	}

	return meter, nil
}

// List lists all meters for an organization with pagination
func (s *meterService) List(ctx context.Context, orgId string, pagination dto.Pagination) (dto.PaginatedResult[entities.Meter], error) {
	return s.meterRepo.List(ctx, orgId, pagination)
}

// Delete deletes a meter
func (s *meterService) Delete(ctx context.Context, orgId, meterId string) error {
	// Check if meter exists
	_, err := s.meterRepo.FindById(ctx, orgId, meterId)
	if err != nil {
		return fmt.Errorf("meter not found: %w", err)
	}

	// Delete meter
	return s.meterRepo.Delete(ctx, orgId, meterId)
}

// ValidateEventAgainstMeter checks if an event matches the meter's configuration
func (s *meterService) ValidateEventAgainstMeter(ctx context.Context, meter entities.Meter, event map[string]interface{}) bool {
	return meter.ValidateEventAgainstMeter(event)
}
