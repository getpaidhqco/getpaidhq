package response

import (
	"payloop/internal/domain/entities"
	"time"
)

// Meter represents a meter response
type Meter struct {
	Id              string                 `json:"id"`
	Slug            string                 `json:"slug"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	EventName       string                 `json:"event_name"`
	EventFilter     map[string]interface{} `json:"event_filter"`
	AggregationType string                 `json:"aggregation_type"`
	ValueProperty   string                 `json:"value_property"`
	UnitType        string                 `json:"unit_type"`
	DisplayName     string                 `json:"display_name"`
	WindowSize      string                 `json:"window_size"`
	ResetInterval   string                 `json:"reset_interval"`
	Metadata        map[string]string      `json:"metadata"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// NewMeterFromEntity creates a new meter response from a meter entity
func NewMeterFromEntity(entity entities.Meter) Meter {
	return Meter{
		Id:              entity.Id,
		Slug:            entity.Slug,
		Name:            entity.Name,
		Description:     entity.Description,
		EventName:       entity.EventName,
		EventFilter:     entity.EventFilter,
		AggregationType: string(entity.AggregationType),
		ValueProperty:   entity.ValueProperty,
		UnitType:        string(entity.UnitType),
		DisplayName:     entity.DisplayName,
		WindowSize:      entity.WindowSize,
		ResetInterval:   entity.ResetInterval,
		Metadata:        entity.Metadata,
		CreatedAt:       entity.CreatedAt,
		UpdatedAt:       entity.UpdatedAt,
	}
}

// NewMetersFromEntities creates a new list of meter responses from a list of meter entities
func NewMetersFromEntities(entities []entities.Meter) []Meter {
	meters := make([]Meter, len(entities))
	for i, entity := range entities {
		meters[i] = NewMeterFromEntity(entity)
	}
	return meters
}