package request

// CreateMeterRequest represents a request to create a new meter
type CreateMeterRequest struct {
	Name            string                 `json:"name" binding:"required"`
	Description     string                 `json:"description"`
	EventName       string                 `json:"event_name" binding:"required"`
	EventFilter     map[string]interface{} `json:"event_filter"`
	AggregationType string                 `json:"aggregation_type" binding:"required,oneof=sum max average last_during_period"`
	ValueProperty   string                 `json:"value_property"`
	UnitType        string                 `json:"unit_type" binding:"required"`
	DisplayName     string                 `json:"display_name"`
	WindowSize      string                 `json:"window_size" binding:"omitempty,oneof=minute hour day month"`
	ResetInterval   string                 `json:"reset_interval" binding:"omitempty,oneof=hourly daily monthly never"`
	Metadata        map[string]string      `json:"metadata"`
}

// UpdateMeterRequest represents a request to update an existing meter
type UpdateMeterRequest struct {
	Name            string                 `json:"name" binding:"required"`
	Description     string                 `json:"description"`
	EventName       string                 `json:"event_name" binding:"required"`
	EventFilter     map[string]interface{} `json:"event_filter"`
	AggregationType string                 `json:"aggregation_type" binding:"required,oneof=sum max average last_during_period"`
	ValueProperty   string                 `json:"value_property"`
	UnitType        string                 `json:"unit_type" binding:"required"`
	DisplayName     string                 `json:"display_name" binding:"required"`
	WindowSize      string                 `json:"window_size" binding:"omitempty,oneof=minute hour day month"`
	ResetInterval   string                 `json:"reset_interval" binding:"omitempty,oneof=hourly daily monthly never"`
	Metadata        map[string]string      `json:"metadata"`
}

// GetMeterByEventNameRequest represents a request to get a meter by event name
type GetMeterByEventNameRequest struct {
	EventName string `json:"event_name" binding:"required"`
}