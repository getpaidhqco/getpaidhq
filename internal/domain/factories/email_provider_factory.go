package factories

import (
	"fmt"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/email_providers"
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
		// DEPRECATED: This factory is deprecated and incompatible with the new settings-based approach.
		// Use the module-based configuration (internal/infrastructure/email/loops/module.go) instead.
		// Template configuration is now handled through the settings service and database.
		return nil, fmt.Errorf("loops factory is deprecated - use module-based configuration with settings service")
	default:
		return nil, fmt.Errorf("unsupported email provider: %s", providerName)
	}
}