package resend

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/resend/resend-go/v2"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/email_providers"
)

// ResendConfig contains the configuration for the Resend email provider
type ResendConfig struct {
	APIKey    string
	FromEmail string // Default from email address
}

// Validate validates the Resend configuration
func (c ResendConfig) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("resend API key is required")
	}
	if c.FromEmail == "" {
		return fmt.Errorf("resend from email is required")
	}
	return nil
}

// ResendProvider implements the email_providers.Provider interface using the resend-go SDK
type ResendProvider struct {
	logger logger.Logger
	config ResendConfig
	client *resend.Client
}

// NewResendProvider creates a new Resend email provider using the resend-go SDK
func NewResendProvider(logger logger.Logger, config ResendConfig) (*ResendProvider, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// Create the resend client
	client := resend.NewClient(config.APIKey)

	return &ResendProvider{
		logger: logger,
		config: config,
		client: client,
	}, nil
}

// SendEmail sends an email using the Resend transactional email API
func (p *ResendProvider) SendEmail(ctx context.Context, orgId string, emailType email_providers.EmailType, input email_providers.SendEmailInput) (email_providers.SendEmailResponse, error) {
	p.logger.Info("Sending email via Resend SDK", "org_id", orgId, "email_type", emailType, "to", input.To, "subject", input.Subject)

	// For now, we'll use minimal placeholder content
	// TODO: Implement proper HTML/text generation based on email type
	htmlContent := "<p>Email content placeholder</p>"
	textContent := "Email content placeholder"

	// Convert attachments to Resend format
	var resendAttachments []*resend.Attachment
	if len(input.Attachments) > 0 {
		resendAttachments = make([]*resend.Attachment, 0, len(input.Attachments))
		for _, attachment := range input.Attachments {
			// Resend expects base64 encoded content as []byte
			encodedContent := base64.StdEncoding.EncodeToString(attachment.Data)
			resendAttachments = append(resendAttachments, &resend.Attachment{
				Filename: attachment.Filename,
				Content:  []byte(encodedContent),
			})
		}
	}

	// Prepare the email request
	emailRequest := &resend.SendEmailRequest{
		From:        p.config.FromEmail,
		To:          []string{input.To},
		Subject:     input.Subject,
		Html:        htmlContent,
		Text:        textContent,
		Attachments: resendAttachments,
	}

	// Add reply-to if specified in variables
	if replyTo, ok := input.Variables["replyTo"].(string); ok && replyTo != "" {
		emailRequest.ReplyTo = replyTo
	}

	// Send the email
	sent, err := p.client.Emails.Send(emailRequest)
	if err != nil {
		p.logger.Error("Failed to send email via Resend SDK", "error", err, "to", input.To)
		return email_providers.SendEmailResponse{
			Success:      false,
			ProviderID:   "resend",
			ErrorMessage: fmt.Sprintf("Failed to send email: %v", err),
		}, err
	}

	p.logger.Info("Email sent successfully via Resend SDK", "to", input.To, "message_id", sent.Id)
	return email_providers.SendEmailResponse{
		Success:    true,
		ProviderID: "resend",
		MessageID:  sent.Id,
	}, nil
}