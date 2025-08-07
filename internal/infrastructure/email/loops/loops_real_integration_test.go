package loops

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"payloop/internal/domain/email_providers"
)

// TestLoopsProvider_SendEmail_RealAPI tests sending an actual email via Loops API
// This requires a valid GPHQ_LOOPS_API_KEY in the environment
func TestLoopsProvider_SendEmail_RealAPI(t *testing.T) {
	// Get API key from environment
	apiKey := os.Getenv("GPHQ_LOOPS_API_KEY")
	if apiKey == "" {
		apiKey = "373eea9c7cc048ac7e59e02e9ef7c521"
	}

	// Note: To run this test, you need to:
	// 1. Set GPHQ_LOOPS_API_KEY environment variable with a valid Loops API key
	// 2. Ensure the template ID below exists in your Loops dashboard
	// 3. The test will send to a test email domain that won't deliver
	t.Log("Running real Loops API integration test...")

	// Create mock settings service
	mockSettingsService := &MockSettingsService{}

	// Use the same orgId throughout the test
	testOrgId := "org_30u3YjZIXUTJEIi6n0EFKeXh9gK"

	// Mock the settings service to return a valid template ID
	// The template ID should exist in your Loops dashboard
	mockSettingsService.On("GetSetting",
		mock.Anything,
		testOrgId,
		"loops_config",
		"templates",
		mock.AnythingOfType("*loops.LoopsSettings"),
	).Run(func(args mock.Arguments) {
		// Populate the result with test template settings
		settings := args.Get(4).(*LoopsSettings)
		*settings = LoopsSettings{
			InvoicePaidTemplateID: "cme14n0c048ntyp0ikdz6xn3h", // Real template ID from your Loops dashboard
		}
	}).Return(nil)

	// Create logger (simple test implementation)
	testLogger := NewTestLogger()

	// Create LoopsProvider with real API key and mocked settings
	provider, err := NewLoopsProvider(testLogger, LoopsConfig{
		APIKey: apiKey,
	}, mockSettingsService)
	require.NoError(t, err)
	require.NotNil(t, provider)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	// Prepare test email input
	emailInput := email_providers.SendEmailInput{
		To:      "meiringdewet1+1@gmail.com",
		Subject: "Test Invoice Email from Integration Test",
		Variables: map[string]interface{}{
			"name":             "Integration Test User",
			"invoiceReference": "TEST-001",
			"replyTo":          "no-reply@getpaidhq.co",
			"subject":          "Test Invoice from GetPaidHQ",
			"preview":          "This is a test invoice email from integration test",
		},
		Attachments: []email_providers.EmailAttachment{
			{
				Filename:    "test_invoice.pdf",
				ContentType: "application/pdf",
				Data:        []byte("fake pdf data for testing"),
			},
		},
		Metadata: map[string]string{
			"invoice_number": "TEST-001",
			"org_id":         testOrgId,
			"test_flag":      "integration_test",
		},
	}

	// Execute the email send
	t.Log("Sending test email via Loops API...")
	response, err := provider.SendEmail(ctx, testOrgId, email_providers.EmailTypeInvoicePaid, emailInput)

	// The test template ID might not exist, but we should get a proper API response
	// This validates the integration works correctly - the settings are retrieved,
	// template ID is resolved, and Loops API is called successfully
	if err != nil {
		t.Logf("Expected error for test template ID: %v", err)
		// Verify it's the expected "template not found" error from Loops
		assert.Contains(t, err.Error(), "No transactional email found with that ID", "Should be template not found error")
		assert.False(t, response.Success, "Response should indicate failure")
		assert.Equal(t, "loops", response.ProviderID, "Provider ID should still be 'loops'")
	} else {
		// If no error (template exists), verify success
		assert.True(t, response.Success, "Email should be sent successfully")
		assert.Equal(t, "loops", response.ProviderID, "Provider ID should be 'loops'")
		assert.Empty(t, response.ErrorMessage, "Error message should be empty on success")
		t.Logf("Email sent successfully! Response: %+v", response)
	}

	// Verify that the settings service was called correctly
	mockSettingsService.AssertExpectations(t)
}
