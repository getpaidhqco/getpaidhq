package email_providers

import (
	"context"
)

// EmailType represents the type of email being sent
type EmailType string

const (
	// EmailTypeInvoicePaid represents an invoice payment confirmation email
	EmailTypeInvoicePaid EmailType = "invoice_paid"
)

// String returns the string representation of the EmailType
func (e EmailType) String() string {
	return string(e)
}

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

// SendEmailInput contains the data needed to send an email
type SendEmailInput struct {
	To          string
	Subject     string
	Variables   map[string]interface{}
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
	// SendEmail sends an email based on the specified email type
	SendEmail(ctx context.Context, orgId string, emailType EmailType, input SendEmailInput) (SendEmailResponse, error)
}
