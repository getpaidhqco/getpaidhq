package loops

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/email_providers"
	"payloop/internal/domain/entities"
	"time"
)

// LoopsConfig contains the configuration for the Loops email provider
type LoopsConfig struct {
	APIKey      string
	APIEndpoint string
	FromEmail   string
	FromName    string
}

// Validate validates the Loops configuration
func (c LoopsConfig) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("Loops API key is required")
	}
	if c.APIEndpoint == "" {
		return fmt.Errorf("Loops API endpoint is required")
	}
	if c.FromEmail == "" {
		return fmt.Errorf("From email is required")
	}
	return nil
}

// LoopsProvider implements the email_providers.Provider interface for Loops
type LoopsProvider struct {
	logger logger.Logger
	config LoopsConfig
	client *http.Client
}

// NewLoopsProvider creates a new Loops email provider
func NewLoopsProvider(logger logger.Logger, config LoopsConfig) (*LoopsProvider, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	return &LoopsProvider{
		logger: logger,
		config: config,
		client: client,
	}, nil
}

// loopsEmailRequest represents the request body for the Loops API
type loopsEmailRequest struct {
	To          string                 `json:"to"`
	From        string                 `json:"from"`
	FromName    string                 `json:"from_name,omitempty"`
	Subject     string                 `json:"subject"`
	HTML        string                 `json:"html,omitempty"`
	Text        string                 `json:"text,omitempty"`
	Attachments []loopsAttachment      `json:"attachments,omitempty"`
	Metadata    map[string]string      `json:"metadata,omitempty"`
}

// loopsAttachment represents an attachment in the Loops API
type loopsAttachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Data        string `json:"data"` // Base64 encoded data
}

// loopsEmailResponse represents the response from the Loops API
type loopsEmailResponse struct {
	Success   bool   `json:"success"`
	MessageID string `json:"message_id,omitempty"`
	Error     string `json:"error,omitempty"`
}

// SendEmail sends an email using the Loops API
func (p *LoopsProvider) SendEmail(ctx context.Context, input email_providers.SendEmailCommand) (email_providers.SendEmailResponse, error) {
	p.logger.Info("Sending email via Loops", "to", input.To, "subject", input.Subject)

	// Convert attachments to Loops format
	loopsAttachments := make([]loopsAttachment, 0, len(input.Attachments))
	for _, attachment := range input.Attachments {
		loopsAttachments = append(loopsAttachments, loopsAttachment{
			Filename:    attachment.Filename,
			ContentType: attachment.ContentType,
			Data:        base64.StdEncoding.EncodeToString(attachment.Data),
		})
	}

	// Create request body
	reqBody := loopsEmailRequest{
		To:          input.To,
		From:        p.config.FromEmail,
		FromName:    p.config.FromName,
		Subject:     input.Subject,
		HTML:        input.HtmlContent,
		Text:        input.TextContent,
		Attachments: loopsAttachments,
		Metadata:    input.Metadata,
	}

	// Convert request to JSON
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		p.logger.Error("Failed to marshal email request", "error", err)
		return email_providers.SendEmailResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to marshal request: %v", err),
		}, err
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", p.config.APIEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		p.logger.Error("Failed to create HTTP request", "error", err)
		return email_providers.SendEmailResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to create request: %v", err),
		}, err
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.config.APIKey))

	// Send request
	resp, err := p.client.Do(req)
	if err != nil {
		p.logger.Error("Failed to send email", "error", err)
		return email_providers.SendEmailResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to send request: %v", err),
		}, err
	}
	defer resp.Body.Close()

	// Parse response
	var loopsResp loopsEmailResponse
	if err := json.NewDecoder(resp.Body).Decode(&loopsResp); err != nil {
		p.logger.Error("Failed to decode response", "error", err)
		return email_providers.SendEmailResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to decode response: %v", err),
		}, err
	}

	// Check for errors
	if !loopsResp.Success {
		p.logger.Error("Loops API returned error", "error", loopsResp.Error)
		return email_providers.SendEmailResponse{
			Success:      false,
			ProviderID:   "loops",
			ErrorMessage: loopsResp.Error,
		}, fmt.Errorf("loops API error: %s", loopsResp.Error)
	}

	p.logger.Info("Email sent successfully", "message_id", loopsResp.MessageID)
	return email_providers.SendEmailResponse{
		Success:    true,
		ProviderID: "loops",
		MessageID:  loopsResp.MessageID,
	}, nil
}

// SendInvoiceNotification sends an invoice notification email with the invoice PDF attached
func (p *LoopsProvider) SendInvoiceNotification(ctx context.Context, customer entities.Customer, invoice entities.Invoice, orgName string, pdfData []byte) (email_providers.SendEmailResponse, error) {
	p.logger.Info("Sending invoice notification email", "customer_id", customer.Id, "invoice_id", invoice.Id)

	// Create PDF attachment
	attachments := []email_providers.EmailAttachment{
		{
			Filename:    fmt.Sprintf("invoice_%s.pdf", invoice.Id),
			ContentType: "application/pdf",
			Data:        pdfData,
		},
	}

	// Create email content
	subject := fmt.Sprintf("Invoice #%s from %s", invoice.DocNumber, orgName)

	// Simple HTML template for the email
	htmlContent := fmt.Sprintf(`
		<html>
		<body>
			<h1>Invoice #%s</h1>
			<p>Dear %s,</p>
			<p>Please find attached your invoice #%s for the amount of %s %d.</p>
			<p>Due date: %s</p>
			<p>Thank you for your business!</p>
			<p>Regards,<br>%s</p>
		</body>
		</html>
	`, invoice.DocNumber, customer.FirstName, invoice.DocNumber, 
	   invoice.Currency, invoice.Total, invoice.DueAt.Format("2006-01-02"), 
	   orgName)

	// Plain text version
	textContent := fmt.Sprintf(
		"Invoice #%s\n\nDear %s,\n\nPlease find attached your invoice #%s for the amount of %s %d.\n\nDue date: %s\n\nThank you for your business!\n\nRegards,\n%s",
		invoice.DocNumber, customer.FirstName, invoice.DocNumber,
		invoice.Currency, invoice.Total, invoice.DueAt.Format("2006-01-02"),
		orgName,
	)

	// Metadata for tracking
	metadata := map[string]string{
		"invoice_id":     invoice.Id,
		"customer_id":    customer.Id,
		"organization_id": invoice.OrgId,
	}

	// Send the email
	return p.SendEmail(ctx, email_providers.SendEmailCommand{
		To:          customer.Email,
		Subject:     subject,
		HtmlContent: htmlContent,
		TextContent: textContent,
		Attachments: attachments,
		Metadata:    metadata,
	})
}
