package loops

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"payloop/internal/domain/email_providers"
	"payloop/internal/testing/mocks"
)


func TestLoopsProvider_SendEmail_Integration(t *testing.T) {
	// Note: This test uses a dummy API key, so actual Loops API calls will fail.
	// The focus is on testing database interaction and request preparation logic.
	apiKey := "test_loops_api_key"

	tests := []struct {
		name            string
		orgId           string
		emailType       email_providers.EmailType
		input           email_providers.SendEmailInput
		mockSetupFunc   func(*mocks.MockSettingsService)
		expectedSuccess bool
		expectedError   string
		expectLoopsCall bool // Whether we expect the Loops API to be called
	}{
		// Note: Tests that would succeed require valid API credentials
		// These test the database interaction and request preparation parts
		{
			name:      "unsupported email type",
			orgId:     "org_123",
			emailType: email_providers.EmailType("unsupported_type"),
			input: email_providers.SendEmailInput{
				To:      "customer@example.com",
				Subject: "Test Email",
				Variables: map[string]interface{}{
					"name": "John Doe",
				},
			},
			mockSetupFunc: func(m *MockSettingsService) {
				m.On("GetSetting", mock.Anything, "org_123", "loops_config", "templates", mock.AnythingOfType("*loops.LoopsSettings")).Return(nil)
			},
			expectedSuccess: false,
			expectedError:   "unsupported email type: unsupported_type",
			expectLoopsCall: false,
		},
		{
			name:      "empty template ID should fail before API call",
			orgId:     "org_789",
			emailType: email_providers.EmailTypeInvoicePaid,
			input: email_providers.SendEmailInput{
				To:      "customer@example.com",
				Subject: "Test Email",
				Variables: map[string]interface{}{
					"name": "John Doe",
				},
			},
			mockSetupFunc: func(m *MockSettingsService) {
				// Mock settings retrieval but with empty template ID
				m.On("GetSetting", mock.Anything, "org_789", "loops_config", "templates", mock.AnythingOfType("*loops.LoopsSettings")).Run(func(args mock.Arguments) {
					settings := args.Get(4).(*LoopsSettings)
					*settings = LoopsSettings{
						InvoicePaidTemplateID: "", // Empty template ID
					}
				}).Return(nil)
			},
			expectedSuccess: false,
			expectedError:   "invoice_paid template ID not configured",
			expectLoopsCall: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock settings service
			mockSettingsService := &MockSettingsService{}
			tt.mockSetupFunc(mockSettingsService)

			// Create test logger
			testLogger := NewTestLogger()

			// Create LoopsProvider with mocked dependencies
			provider, err := NewLoopsProvider(testLogger, LoopsConfig{
				APIKey: apiKey,
			}, mockSettingsService)
			require.NoError(t, err)
			require.NotNil(t, provider)

			// Create context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Execute the test
			response, err := provider.SendEmail(ctx, tt.orgId, tt.emailType, tt.input)

			// Verify results
			if tt.expectedSuccess {
				assert.NoError(t, err)
				assert.True(t, response.Success)
				assert.Equal(t, "loops", response.ProviderID)
				assert.Empty(t, response.ErrorMessage)
			} else {
				if tt.expectedError != "" {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), tt.expectedError)
					assert.False(t, response.Success)
					assert.Contains(t, response.ErrorMessage, tt.expectedError)
				}
			}

			// Verify mock expectations
			mockSettingsService.AssertExpectations(t)
		})
	}
}

func TestLoopsProvider_GetLoopsSettings_DatabaseInteraction(t *testing.T) {
	tests := []struct {
		name             string
		orgId            string
		mockSetupFunc    func(*MockSettingsService)
		expectedSettings LoopsSettings
		expectError      bool
	}{
		{
			name:  "successful settings retrieval from database",
			orgId: "org_123",
			mockSetupFunc: func(m *MockSettingsService) {
				m.On("GetSetting", mock.Anything, "org_123", "loops_config", "templates", mock.AnythingOfType("*loops.LoopsSettings")).Return(nil)
			},
			expectedSettings: LoopsSettings{
				InvoicePaidTemplateID: "test_template_123",
			},
			expectError: false,
		},
		{
			name:  "settings not found - should return defaults",
			orgId: "org_456",
			mockSetupFunc: func(m *MockSettingsService) {
				m.On("GetSetting", mock.Anything, "org_456", "loops_config", "templates", mock.AnythingOfType("*loops.LoopsSettings")).Return(assert.AnError)
			},
			expectedSettings: DefaultLoopsSettings(),
			expectError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock settings service
			mockSettingsService := &MockSettingsService{}
			tt.mockSetupFunc(mockSettingsService)

			// Create test logger
			testLogger := NewTestLogger()

			// Create LoopsProvider
			provider, err := NewLoopsProvider(testLogger, LoopsConfig{
				APIKey: "test_api_key",
			}, mockSettingsService)
			require.NoError(t, err)

			// Create context
			ctx := context.Background()

			// Test getLoopsSettings method
			settings, err := provider.getLoopsSettings(ctx, tt.orgId)

			// Verify results
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedSettings, settings)
			}

			// Verify mock expectations
			mockSettingsService.AssertExpectations(t)
		})
	}
}

func TestLoopsProvider_GetTemplateID(t *testing.T) {
	testLogger := NewTestLogger()
	mockSettingsService := &MockSettingsService{}

	provider, err := NewLoopsProvider(testLogger, LoopsConfig{
		APIKey: "test_api_key",
	}, mockSettingsService)
	require.NoError(t, err)

	tests := []struct {
		name          string
		emailType     email_providers.EmailType
		settings      LoopsSettings
		expectedID    string
		expectedError string
	}{
		{
			name:      "valid invoice_paid email type",
			emailType: email_providers.EmailTypeInvoicePaid,
			settings: LoopsSettings{
				InvoicePaidTemplateID: "template_123",
			},
			expectedID: "template_123",
		},
		{
			name:      "empty template ID",
			emailType: email_providers.EmailTypeInvoicePaid,
			settings: LoopsSettings{
				InvoicePaidTemplateID: "",
			},
			expectedError: "invoice_paid template ID not configured",
		},
		{
			name:      "unsupported email type",
			emailType: email_providers.EmailType("unknown_type"),
			settings: LoopsSettings{
				InvoicePaidTemplateID: "template_123",
			},
			expectedError: "unsupported email type: unknown_type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			templateID, err := provider.getTemplateID(tt.emailType, tt.settings)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Empty(t, templateID)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, templateID)
			}
		})
	}
}
