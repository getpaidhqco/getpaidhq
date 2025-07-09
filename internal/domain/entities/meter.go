package entities

import (
	"fmt"
	"payloop/internal/lib"
	"time"
)

// Meter defines how to measure usage
type Meter struct {
	OrgId       string `json:"org_id"`
	Id          string `json:"id"`
	Slug        string `json:"slug"`        // Unique machine-readable identifier (e.g., "api_calls", "storage_gb_hours")
	Name        string `json:"name"`        // Human-readable name (e.g., "API Calls", "Storage Usage")
	Description string `json:"description"` // Detailed description of what this meter measures

	// Event Configuration
	EventName   string                 `json:"event_name"`   // The event type to track (e.g., "api.request", "storage.snapshot")
	EventFilter map[string]interface{} `json:"event_filter"` // Optional filters to apply (e.g., {"method": "POST", "tier": "premium"})

	// Aggregation Configuration
	AggregationType AggregationType `json:"aggregation_type"` // How to aggregate: sum, count, max, average, last_during_period
	ValueProperty   string          `json:"value_property"`   // Which event property to aggregate (e.g., "duration", "bytes", "tokens")

	// Display Configuration
	UnitType    UnitType `json:"unit_type"`    // Unit of measurement: gb_hours, api_calls, minutes, etc.
	DisplayName string   `json:"display_name"` // How to display in invoices/UI (e.g., "API Calls", "GB-Hours")

	// Window Configuration
	WindowSize    string `json:"window_size"`    // Aggregation window: "minute", "hour", "day", "month"
	ResetInterval string `json:"reset_interval"` // When to reset counters: "hourly", "daily", "monthly", "never"

	Metadata  map[string]string `json:"metadata"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// CreateMeterInput is the input for creating a new meter
type CreateMeterInput struct {
	Slug            string                 `json:"slug"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	EventName       string                 `json:"event_name"`
	EventFilter     map[string]interface{} `json:"event_filter"`
	AggregationType AggregationType        `json:"aggregation_type"`
	ValueProperty   string                 `json:"value_property"`
	UnitType        UnitType               `json:"unit_type"`
	DisplayName     string                 `json:"display_name"`
	WindowSize      string                 `json:"window_size"`
	ResetInterval   string                 `json:"reset_interval"`
	Metadata        map[string]string      `json:"metadata"`
}

// NewMeter creates a new meter entity
func NewMeter(orgId string, input CreateMeterInput) (Meter, error) {
	if err := validateMeterInput(input); err != nil {
		return Meter{}, err
	}

	return Meter{
		OrgId:           orgId,
		Id:              lib.GenerateId("meter"),
		Slug:            input.Slug,
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
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}, nil
}

// validateMeterInput validates the input for creating a meter
func validateMeterInput(input CreateMeterInput) error {
	if input.Slug == "" {
		return fmt.Errorf("slug is required")
	}
	if input.Name == "" {
		return fmt.Errorf("name is required")
	}
	if input.EventName == "" {
		return fmt.Errorf("event name is required")
	}
	if input.AggregationType == "" {
		return fmt.Errorf("aggregation type is required")
	}
	if input.UnitType == "" {
		return fmt.Errorf("unit type is required")
	}

	return nil
}

// ValidateEventAgainstMeter checks if an event matches the meter's configuration
func (m *Meter) ValidateEventAgainstMeter(event map[string]interface{}) bool {
	// Check if event matches meter's event name
	eventName, ok := event["event_name"].(string)
	if !ok || eventName != m.EventName {
		return false
	}

	// Apply event filters if any
	if m.EventFilter != nil && len(m.EventFilter) > 0 {
		for key, expectedValue := range m.EventFilter {
			actualValue, exists := event[key]
			if !exists || actualValue != expectedValue {
				return false
			}
		}
	}

	return true
}
