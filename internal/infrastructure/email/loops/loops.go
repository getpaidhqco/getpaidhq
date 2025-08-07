package loops

import (
	"context"
	"fmt"

	"github.com/tilebox/loops-go"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/email_providers"
)

// LoopsConfig contains the configuration for the Loops email provider
type LoopsConfig struct {
	APIKey string
}

// LoopsSettings contains the template configuration stored in database
type LoopsSettings struct {
	InvoicePaidTemplateID string `json:"invoice_paid_template_id"`
}

// DefaultLoopsSettings returns the default template configuration
func DefaultLoopsSettings() LoopsSettings {
	return LoopsSettings{
		InvoicePaidTemplateID: "cme14n0c048ntyp0ikdz6xn3h", // Must be configured in database or Loops dashboard
	}
}

// Validate validates the Loops configuration
func (c LoopsConfig) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("loops API key is required")
	}
	return nil
}

// LoopsProvider implements the email_providers.Provider interface using the loops-go SDK
type LoopsProvider struct {
	logger          logger.Logger
	config          LoopsConfig
	client          *loops.Client
	settingsService interfaces.SettingsService
}

// NewLoopsProvider creates a new Loops email provider using the loops-go SDK
func NewLoopsProvider(logger logger.Logger, config LoopsConfig, settingsService interfaces.SettingsService) (*LoopsProvider, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// Create the loops client
	client, err := loops.NewClient(loops.WithAPIKey(config.APIKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create loops client: %w", err)
	}

	return &LoopsProvider{
		logger:          logger,
		config:          config,
		client:          client,
		settingsService: settingsService,
	}, nil
}

// getLoopsSettings retrieves the loops template configuration from the database
func (p *LoopsProvider) getLoopsSettings(ctx context.Context, orgId string) (LoopsSettings, error) {
	var settings LoopsSettings

	// Try to get settings from database
	err := p.settingsService.GetSetting(ctx, orgId, "loops_config", "templates", &settings)
	if err != nil {
		p.logger.Warn("No loops settings found in database, using defaults", "org_id", orgId, "error", err.Error())
		return DefaultLoopsSettings(), nil
	}

	return settings, nil
}

// getTemplateID returns the template ID for the given email type
func (p *LoopsProvider) getTemplateID(emailType email_providers.EmailType, settings LoopsSettings) (string, error) {
	switch emailType {
	case email_providers.EmailTypeInvoicePaid:
		if settings.InvoicePaidTemplateID == "" {
			return "", fmt.Errorf("invoice_paid template ID not configured - create template in Loops dashboard and configure in database")
		}
		return settings.InvoicePaidTemplateID, nil
	default:
		return "", fmt.Errorf("unsupported email type: %s", emailType)
	}
}

// SendEmail sends an email using the Loops transactional email API
func (p *LoopsProvider) SendEmail(ctx context.Context, orgId string, emailType email_providers.EmailType, input email_providers.SendEmailInput) (email_providers.SendEmailResponse, error) {
	p.logger.Info("Sending email via Loops SDK", "org_id", orgId, "email_type", emailType, "to", input.To, "subject", input.Subject)

	// Get template settings from database
	settings, err := p.getLoopsSettings(ctx, orgId)
	if err != nil {
		return email_providers.SendEmailResponse{
			Success:      false,
			ProviderID:   "loops",
			ErrorMessage: fmt.Sprintf("Failed to get template settings: %v", err),
		}, err
	}

	// Get the template ID based on email type
	templateID, err := p.getTemplateID(emailType, settings)
	if err != nil {
		return email_providers.SendEmailResponse{
			Success:      false,
			ProviderID:   "loops",
			ErrorMessage: fmt.Sprintf("Failed to get template ID: %v", err),
		}, err
	}

	// Convert attachments to Loops format
	var loopsAttachments *[]loops.EmailAttachment
	if len(input.Attachments) > 0 {
		attachments := make([]loops.EmailAttachment, 0, len(input.Attachments))
		for _, attachment := range input.Attachments {
			attachments = append(attachments, loops.EmailAttachment{
				Filename:    attachment.Filename,
				ContentType: attachment.ContentType,
				Data:        string(attachment.Data), // loops-go expects string, not base64
			})
		}
		loopsAttachments = &attachments
	}

	// Prepare data variables for the template
	dataVariables := input.Variables

	// Helper to create boolean pointer
	addToAudience := false

	// Send the transactional email
	err = p.client.SendTransactionalEmail(ctx, &loops.TransactionalEmail{
		TransactionalID: templateID,
		Email:           input.To,
		AddToAudience:   &addToAudience, // Don't add to marketing audience
		DataVariables:   &dataVariables,
		Attachments:     loopsAttachments,
	})

	if err != nil {
		p.logger.Error("Failed to send email via Loops SDK", "error", err, "to", input.To)
		return email_providers.SendEmailResponse{
			Success:      false,
			ProviderID:   "loops",
			ErrorMessage: fmt.Sprintf("Failed to send email: %v", err),
		}, err
	}

	p.logger.Info("Email sent successfully via Loops SDK", "to", input.To)
	return email_providers.SendEmailResponse{
		Success:    true,
		ProviderID: "loops",
		MessageID:  "", // Loops doesn't return message ID in this SDK version
	}, nil
}
