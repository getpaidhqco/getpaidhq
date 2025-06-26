package email_providers

import (
	"context"
	"payloop/internal/domain/entities"
)

// ProviderConfig is an interface for email provider configurations
type ProviderConfig interface {
	Validate() error
}

// EmailAttachment represents a file attachment for an email
type EmailAttachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

// SendEmailCommand contains the data needed to send an email
type SendEmailCommand struct {
	To          string
	Subject     string
	HtmlContent string
	TextContent string
	Attachments []EmailAttachment
	Metadata    map[string]string
}

// SendEmailResponse contains the response from sending an email
type SendEmailResponse struct {
	Success      bool
	ProviderID   string
	MessageID    string
	ErrorMessage string
}

// Provider is the interface that all email providers must implement
type Provider interface {
	// SendEmail sends an email with optional attachments
	SendEmail(ctx context.Context, input SendEmailCommand) (SendEmailResponse, error)

	// SendInvoiceNotification sends an invoice notification email with the invoice PDF attached
	SendInvoiceNotification(ctx context.Context, customer entities.Customer, invoice entities.Invoice, orgName string, pdfData []byte) (SendEmailResponse, error)
}
