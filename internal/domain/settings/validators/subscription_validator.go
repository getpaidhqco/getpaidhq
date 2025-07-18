package validators

import (
	"encoding/json"
	"errors"
	"fmt"
	"payloop/internal/domain/settings"
)

// SubscriptionSettings represents subscription configuration
type SubscriptionSettings struct {
	EnableInvoicePdfs bool         `json:"enable_invoice_pdfs"`
	InvoicePrefix     string       `json:"invoice_prefix"`
	EmailReminders    bool         `json:"email_reminders"`
	ReminderDays      int          `json:"reminder_days"`
	CancelOnFailure   bool         `json:"cancel_on_failure"`
	RetryPolicy       *RetryPolicy `json:"retry_policy,omitempty"`
}

type RetryPolicy struct {
	RetryAttempts int    `json:"attempts"`
	RetryPeriod   int    `json:"retry_period"`
	FailureAction string `json:"failure_action"` // cancel, mark_unpaid, past_due
}

type SubscriptionValidator struct {
	settings.BaseValidator
}

func NewSubscriptionValidator() *SubscriptionValidator {
	return &SubscriptionValidator{}
}

func (v *SubscriptionValidator) ValidateSettings(value interface{}) error {
	var settings SubscriptionSettings

	// Try to handle different input types
	switch val := value.(type) {
	case SubscriptionSettings:
		// Direct struct type
		settings = val
	case *SubscriptionSettings:
		// Pointer to struct
		if val == nil {
			return errors.New("subscription settings cannot be nil")
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
			return fmt.Errorf("failed to unmarshal subscription settings: %w", err)
		}
	default:
		return errors.New("invalid subscription settings type")
	}

	// Validate reminder days
	if settings.ReminderDays < 0 || settings.ReminderDays > 30 {
		return fmt.Errorf("reminder_days must be between 0 and 30, got %d", settings.ReminderDays)
	}

	// Validate invoice prefix
	if len(settings.InvoicePrefix) > 10 {
		return errors.New("invoice_prefix must be 10 characters or less")
	}

	// Validate retry policy if provided
	if settings.RetryPolicy != nil {
		if settings.RetryPolicy.RetryAttempts < 0 || settings.RetryPolicy.RetryAttempts > 10 {
			return errors.New("retry_attempts must be between 0 and 10")
		}

		if settings.RetryPolicy.RetryPeriod < 1 || settings.RetryPolicy.RetryPeriod > 30 {
			return errors.New("retry_period must be between 1 and 30 days")
		}

		validActions := map[string]bool{
			"cancel":      true,
			"mark_unpaid": true,
			"past_due":    true,
		}
		if !validActions[settings.RetryPolicy.FailureAction] {
			return fmt.Errorf("invalid failure_action: %s", settings.RetryPolicy.FailureAction)
		}
	}

	return nil
}

func (v *SubscriptionValidator) GetSettingsSchema() settings.SettingsSchema {
	return settings.SettingsSchema{
		Name:        "Subscription Settings",
		Description: "Configure subscription billing behavior and retry policies",
		Fields: []settings.SettingsField{
			{
				Name:        "enable_invoice_pdfs",
				Type:        "boolean",
				Required:    true,
				Description: "Enable PDF generation for invoices",
				Default:     true,
			},
			{
				Name:        "invoice_prefix",
				Type:        "string",
				Required:    false,
				Description: "Prefix for invoice numbers (max 10 chars)",
				Validation:  "max:10",
			},
			{
				Name:        "email_reminders",
				Type:        "boolean",
				Required:    true,
				Description: "Send email reminders for upcoming charges",
				Default:     true,
			},
			{
				Name:        "reminder_days",
				Type:        "number",
				Required:    true,
				Description: "Days before charge to send reminder",
				Default:     3,
				Validation:  "min:0,max:30",
			},
			{
				Name:        "cancel_on_failure",
				Type:        "boolean",
				Required:    true,
				Description: "Automatically cancel subscription on payment failure",
				Default:     false,
			},
			{
				Name:        "retry_policy",
				Type:        "object",
				Required:    false,
				Description: "Payment retry configuration",
				Children: []settings.SettingsField{
					{
						Name:        "attempts",
						Type:        "number",
						Required:    true,
						Description: "Number of retry attempts",
						Default:     3,
						Validation:  "min:0,max:10",
					},
					{
						Name:        "retry_period",
						Type:        "number",
						Required:    true,
						Description: "Days between retry attempts",
						Default:     3,
						Validation:  "min:1,max:30",
					},
					{
						Name:        "failure_action",
						Type:        "string",
						Required:    true,
						Description: "Action after all retries fail",
						Default:     "past_due",
						Validation:  "in:cancel,mark_unpaid,past_due",
					},
				},
			},
		},
	}
}

func (v *SubscriptionValidator) GetDefaultValue() interface{} {
	return &SubscriptionSettings{
		EnableInvoicePdfs: true,
		InvoicePrefix:     "",
		EmailReminders:    true,
		ReminderDays:      3,
		CancelOnFailure:   false,
		RetryPolicy: &RetryPolicy{
			RetryAttempts: 3,
			RetryPeriod:   3,
			FailureAction: "past_due",
		},
	}
}
