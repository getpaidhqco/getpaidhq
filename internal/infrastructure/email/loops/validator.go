package loops

import (
	"encoding/json"
	"errors"
	"fmt"
	"payloop/internal/domain/settings"
)

type LoopsValidator struct {
	settings.BaseValidator
}

func NewLoopsValidator() *LoopsValidator {
	return &LoopsValidator{}
}

func (v *LoopsValidator) ValidateSettings(value interface{}) error {
	var settings LoopsSettings

	// Try to handle different input types
	switch val := value.(type) {
	case LoopsSettings:
		// Direct struct type
		settings = val
	case *LoopsSettings:
		// Pointer to struct
		if val == nil {
			return errors.New("loops settings cannot be nil")
		}
		settings = *val
	case map[string]interface{}:
		// Map from JSON unmarshaling
		// Convert map to JSON bytes
		jsonBytes, err := json.Marshal(val)
		if err != nil {
			return fmt.Errorf("failed to marshal settings map: %w", err)
		}

		// Unmarshal JSON bytes to struct
		if err := json.Unmarshal(jsonBytes, &settings); err != nil {
			return fmt.Errorf("failed to unmarshal loops settings: %w", err)
		}
	default:
		return errors.New("invalid loops settings type")
	}

	// Validate that at least one template ID is configured
	if settings.InvoicePaidTemplateID == "" {
		return errors.New("invoice_paid_template_id is required - create template in Loops dashboard")
	}

	return nil
}

func (v *LoopsValidator) GetSettingsSchema() settings.SettingsSchema {
	return settings.SettingsSchema{
		Name:        "Loops Email Configuration",
		Description: "Configure Loops.so email templates for transactional emails",
		Fields: []settings.SettingsField{
			{
				Name:        "invoice_paid_template_id",
				Type:        "string",
				Required:    true,
				Description: "Transactional email template ID for invoice paid notifications (created in Loops dashboard)",
			},
		},
	}
}

func (v *LoopsValidator) GetDefaultValue() interface{} {
	return &LoopsSettings{
		InvoicePaidTemplateID: "", // Must be configured manually
	}
}
