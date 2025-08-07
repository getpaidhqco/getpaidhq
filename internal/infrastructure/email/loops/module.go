package loops

import (
	"go.uber.org/fx"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/logger"
	"payloop/internal/application/services"
	"payloop/internal/domain/email_providers"
	"payloop/internal/lib"
)

// Module provides the Loops email provider and registers its validator
var Module = fx.Options(
	fx.Provide(NewLoopsProviderFromConfig),
	fx.Invoke(RegisterLoopsValidator),
)

// NewLoopsProviderFromConfig creates a new Loops email provider from configuration
func NewLoopsProviderFromConfig(logger logger.Logger, env lib.Env, settingsService interfaces.SettingsService) (email_providers.Provider, error) {
	// Check if email provider is configured
	if env.EmailProvider == "" {
		logger.Warn("Email provider not configured, using no-op provider")
		return nil, nil
	}

	// Check if the configured provider is Loops
	if env.EmailProvider != "loops" {
		logger.Warn("Loops provider not configured, using no-op provider")
		return nil, nil
	}

	// Check if Loops API key is configured
	if env.LoopsApiKey == "" {
		logger.Warn("Loops API key not specified, using no-op provider")
		return nil, nil
	}

	logger.Info("Initializing Loops email provider with SDK", "api_key_set", env.LoopsApiKey != "")

	return NewLoopsProvider(logger, LoopsConfig{
		APIKey: env.LoopsApiKey,
	}, settingsService)
}

// RegisterLoopsValidator registers the Loops validator with the settings registry
func RegisterLoopsValidator(registry services.SettingsRegistryInterface, logger logger.Logger) {
	logger.Info("Registering Loops validator with settings registry")
	registry.Register("loops_config", NewLoopsValidator())
}
