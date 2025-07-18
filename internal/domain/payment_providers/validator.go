package payment_providers

// GatewayValidator is an interface that each payment processor must implement
// to validate settings and provide a schema for the settings.
type GatewayValidator interface {
	ValidateSettings(settings map[string]string) error
	GetSettingsSchema() SettingsSchema
}

// SettingsSchema represents the schema for payment processor settings
type SettingsSchema struct {
	Fields []SettingsField `json:"fields"`
}

// SettingsField represents a field in the settings schema
type SettingsField struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // string, number, boolean
	Required    bool   `json:"required"`
	Description string `json:"description"`
	Sensitive   bool   `json:"sensitive,omitempty"`
}