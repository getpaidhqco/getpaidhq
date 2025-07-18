package checkout_com

import (
	"errors"
	"payloop/internal/domain/payment_providers"
)

// CheckoutDotComValidator implements the GatewayValidator interface for CheckoutDotCom
type CheckoutDotComValidator struct{}

// ValidateSettings validates the settings for CheckoutDotCom
func (v CheckoutDotComValidator) ValidateSettings(settings map[string]string) error {
	config := CheckoutDotComConfig{}

	// Convert map to struct
	if secretKey, ok := settings["secret_key"]; ok {
		config.SecretKey = secretKey
	} else {
		return errors.New("secret_key is required")
	}

	if processingChannelId, ok := settings["processing_channel_id"]; ok {
		config.ProcessingChannelId = processingChannelId
	} else {
		return errors.New("processing_channel_id is required")
	}

	// Validate the config
	if config.SecretKey == "" {
		return errors.New("secret_key cannot be empty")
	}

	if config.ProcessingChannelId == "" {
		return errors.New("processing_channel_id cannot be empty")
	}

	// Add more validation rules as needed
	return nil
}

// GetSettingsSchema returns the schema for CheckoutDotCom settings
func (v CheckoutDotComValidator) GetSettingsSchema() payment_providers.SettingsSchema {
	return payment_providers.SettingsSchema{
		Fields: []payment_providers.SettingsField{
			{
				Name:        "secret_key",
				Type:        "string",
				Required:    true,
				Description: "CheckoutDotCom Secret Key",
				Sensitive:   true,
			},
			{
				Name:        "processing_channel_id",
				Type:        "string",
				Required:    true,
				Description: "CheckoutDotCom Processing Channel ID",
			},
		},
	}
}