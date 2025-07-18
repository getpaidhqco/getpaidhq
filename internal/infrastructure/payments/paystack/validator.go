package paystack

import (
	"errors"
	"payloop/internal/domain/payment_providers"
)

// PaystackValidator implements the GatewayValidator interface for Paystack
type PaystackValidator struct{}

// ValidateSettings validates the settings for Paystack
func (v PaystackValidator) ValidateSettings(settings map[string]string) error {
	config := PaystackConfig{}

	// Convert map to struct
	if apiKey, ok := settings["api_key"]; ok {
		config.ApiKey = apiKey
	} else {
		return errors.New("api_key is required")
	}

	if connectId, ok := settings["connect_id"]; ok {
		config.ConnectId = connectId
	}

	// Validate the config
	if config.ApiKey == "" {
		return errors.New("api_key cannot be empty")
	}

	// Add more validation rules as needed
	return nil
}

// GetSettingsSchema returns the schema for Paystack settings
func (v PaystackValidator) GetSettingsSchema() payment_providers.SettingsSchema {
	return payment_providers.SettingsSchema{
		Fields: []payment_providers.SettingsField{
			{
				Name:        "api_key",
				Type:        "string",
				Required:    true,
				Description: "Paystack API key",
				Sensitive:   true,
			},
			{
				Name:        "connect_id",
				Type:        "string",
				Required:    false,
				Description: "Paystack Connect ID for marketplace",
			},
		},
	}
}