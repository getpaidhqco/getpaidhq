package dto

import (
	"payloop/internal/domain/entities"
)

// CreateMeterInput is the input for creating a new meter
type CreateMeterInput struct {
	Name            string                   `json:"name"`
	Description     string                   `json:"description"`
	EventName       string                   `json:"event_name"`
	EventFilter     map[string]interface{}   `json:"event_filter"`
	AggregationType entities.AggregationType `json:"aggregation_type"`
	ValueProperty   string                   `json:"value_property"`
	UnitType        entities.UnitType        `json:"unit_type"`
	DisplayName     string                   `json:"display_name"`
	WindowSize      string                   `json:"window_size"`
	ResetInterval   string                   `json:"reset_interval"`
	Metadata        map[string]string        `json:"metadata"`
}

// UpdateMeterInput is the input for updating an existing meter
type UpdateMeterInput struct {
	Name            string                   `json:"name"`
	Description     string                   `json:"description"`
	EventName       string                   `json:"event_name"`
	EventFilter     map[string]interface{}   `json:"event_filter"`
	AggregationType entities.AggregationType `json:"aggregation_type"`
	ValueProperty   string                   `json:"value_property"`
	UnitType        entities.UnitType        `json:"unit_type"`
	DisplayName     string                   `json:"display_name"`
	WindowSize      string                   `json:"window_size"`
	ResetInterval   string                   `json:"reset_interval"`
	Metadata        map[string]string        `json:"metadata"`
}

// MeterResponse is the response for meter operations
type MeterResponse struct {
	Id              string                 `json:"id"`
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
	CreatedAt       string                 `json:"created_at"`
	UpdatedAt       string                 `json:"updated_at"`
}

// GetMeterInput is the input for getting a meter
type GetMeterInput struct {
	MeterId string `json:"meter_id"`
}

// GetMeterByEventNameInput is the input for getting a meter by event name
type GetMeterByEventNameInput struct {
	EventName string `json:"event_name"`
}

// ListMetersInput is the input for listing meters
type ListMetersInput struct {
	Pagination Pagination `json:"pagination"`
}

// DeleteMeterInput is the input for deleting a meter
type DeleteMeterInput struct {
	MeterId string `json:"meter_id"`
}