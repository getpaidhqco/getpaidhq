package factories

import (
	"fmt"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/email_providers"
	"payloop/internal/infrastructure/email/loops"
)

// EmailProviderFactory creates email providers based on configuration
type EmailProviderFactory struct {
	logger logger.Logger
}

// NewEmailProviderFactory creates a new email provider factory
func NewEmailProviderFactory(logger logger.Logger) *EmailProviderFactory {
	return &EmailProviderFactory{
		logger: logger,
	}
}

// CreateProvider creates an email provider based on the given configuration
func (f *EmailProviderFactory) CreateProvider(config map[string]interface{}) (email_providers.Provider, error) {
	providerName, ok := config["provider"].(string)
	if !ok {
		return nil, fmt.Errorf("email provider not specified in configuration")
	}

	f.logger.Info("Creating email provider", "provider", providerName)

	switch providerName {
	case "loops":
		loopsConfig, ok := config["loops"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("loops configuration not found")
		}

		apiKey, ok := loopsConfig["api_key"].(string)
		if !ok {
			return nil, fmt.Errorf("loops API key not specified")
		}

		apiEndpoint, ok := loopsConfig["api_endpoint"].(string)
		if !ok {
			return nil, fmt.Errorf("loops API endpoint not specified")
		}

		fromEmail, ok := loopsConfig["from_email"].(string)
		if !ok {
			return nil, fmt.Errorf("loops from email not specified")
		}

		fromName, ok := loopsConfig["from_name"].(string)
		if !ok {
			return nil, fmt.Errorf("loops from name not specified")
		}

		provider, err := loops.NewLoopsProvider(f.logger, loops.LoopsConfig{
			APIKey:      apiKey,
			APIEndpoint: apiEndpoint,
			FromEmail:   fromEmail,
			FromName:    fromName,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create loops provider: %w", err)
		}

		return provider, nil
	default:
		return nil, fmt.Errorf("unsupported email provider: %s", providerName)
	}
}