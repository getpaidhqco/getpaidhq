package resend

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"payloop/internal/domain/email_providers"
)

// TestResendProvider_SendEmail_RealAPI tests sending an actual email via Resend API
// This requires a valid RESEND_API_KEY in the environment
func TestResendProvider_SendEmail_RealAPI(t *testing.T) {
	// Get API key from environment
	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey == "" {
		t.Skip("RESEND_API_KEY not set, skipping real API test")
	}

	// Note: To run this test, you need to:
	// 1. Set RESEND_API_KEY environment variable with a valid Resend API key
	// 2. Ensure you have a verified domain in Resend dashboard
	// 3. Update the from_email to use your verified domain
	t.Log("Running real Resend API integration test...")

	// Use the same orgId throughout the test
	testOrgId := "org_30u3YjZIXUTJEIi6n0EFKeXh9gK"

	// Create logger (simple test implementation)
	testLogger := NewTestLogger()

	// Create ResendProvider with real API key
	provider, err := NewResendProvider(testLogger, ResendConfig{
		APIKey:    apiKey,
		FromEmail: "onboarding@resend.dev", // Use Resend's test domain or your verified domain
	})
	require.NoError(t, err)
	require.NotNil(t, provider)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Prepare test email input
	emailInput := email_providers.SendEmailInput{
		To:      "delivered@resend.dev", // Use Resend's test email or your real email
		Subject: "Test Invoice Payment Confirmation - Resend Integration Test",
		Variables: map[string]interface{}{
			"name":             "John Doe",
			"invoiceReference": "INV-2024-001",
			"replyTo":          "support@getpaidhq.co",
			"amount":           "$1,234.56",
			"paymentDate":      time.Now().Format("January 2, 2006"),
		},
		Attachments: []email_providers.EmailAttachment{
			{
				Filename:    "test_invoice.pdf",
				ContentType: "application/pdf",
				Data:        []byte("%PDF-1.4\n1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n3 0 obj\n<< /Type /Page /Parent 2 0 R /Resources << /Font << /F1 << /Type /Font /Subtype /Type1 /BaseFont /Helvetica >> >> >> /MediaBox [0 0 612 792] /Contents 4 0 R >>\nendobj\n4 0 obj\n<< /Length 44 >>\nstream\nBT /F1 12 Tf 100 700 Td (Test Invoice) Tj ET\nendstream\nendobj\nxref\n0 5\n0000000000 65535 f\n0000000009 00000 n\n0000000058 00000 n\n0000000115 00000 n\n0000000274 00000 n\ntrailer\n<< /Size 5 /Root 1 0 R >>\nstartxref\n365\n%%EOF"),
			},
		},
		Metadata: map[string]string{
			"invoice_number": "INV-2024-001",
			"customer_id":    "cust_123456",
			"org_id":         testOrgId,
			"test_flag":      "integration_test",
		},
	}

	// Execute the email send
	t.Log("Sending test email via Resend API...")
	response, err := provider.SendEmail(ctx, testOrgId, email_providers.EmailTypeInvoicePaid, emailInput)

	// Verify the response
	require.NoError(t, err, "Should send email successfully")
	assert.True(t, response.Success, "Email should be sent successfully")
	assert.Equal(t, "resend", response.ProviderID, "Provider ID should be 'resend'")
	assert.NotEmpty(t, response.MessageID, "Should have a message ID")
	assert.Empty(t, response.ErrorMessage, "Error message should be empty on success")
	
	t.Logf("Email sent successfully! Message ID: %s", response.MessageID)
}

// TestResendProvider_SendEmail_WithInvalidAPIKey tests error handling with invalid API key
func TestResendProvider_SendEmail_WithInvalidAPIKey(t *testing.T) {
	testOrgId := "org_test"

	// Create logger
	testLogger := NewTestLogger()

	// Create ResendProvider with invalid API key
	provider, err := NewResendProvider(testLogger, ResendConfig{
		APIKey:    "re_invalid_key_123",
		FromEmail: "test@example.com",
	})
	require.NoError(t, err)
	require.NotNil(t, provider)

	// Create context
	ctx := context.Background()

	// Prepare test email input
	emailInput := email_providers.SendEmailInput{
		To:      "test@example.com",
		Subject: "Test Email",
		Variables: map[string]interface{}{
			"name": "Test User",
		},
	}

	// Execute the email send - should fail with invalid API key
	response, err := provider.SendEmail(ctx, testOrgId, email_providers.EmailTypeInvoicePaid, emailInput)

	// Verify error response
	assert.Error(t, err, "Should return error with invalid API key")
	assert.False(t, response.Success, "Should not be successful")
	assert.Equal(t, "resend", response.ProviderID, "Provider ID should still be 'resend'")
	assert.NotEmpty(t, response.ErrorMessage, "Should have error message")
	
	t.Logf("Expected error received: %v", err)
}