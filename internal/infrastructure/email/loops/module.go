package loops

import (
	"go.uber.org/fx"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/email_providers"
	"payloop/internal/lib"
)

// Module provides the Loops email provider
var Module = fx.Provide(
	NewLoopsProviderFromConfig,
)

// NewLoopsProviderFromConfig creates a new Loops email provider from configuration
func NewLoopsProviderFromConfig(logger logger.Logger, env lib.Env) (email_providers.Provider, error) {
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

	// Check if Loops API endpoint is configured
	if env.LoopsApiEndpoint == "" {
		logger.Warn("Loops API endpoint not specified, using no-op provider")
		return nil, nil
	}

	// Check if from email is configured
	if env.EmailFromEmail == "" {
		logger.Warn("From email not specified, using no-op provider")
		return nil, nil
	}

	// Check if from name is configured
	if env.EmailFromName == "" {
		logger.Warn("From name not specified, using no-op provider")
		return nil, nil
	}

	return NewLoopsProvider(logger, LoopsConfig{
		APIKey:      env.LoopsApiKey,
		APIEndpoint: env.LoopsApiEndpoint,
		FromEmail:   env.EmailFromEmail,
		FromName:    env.EmailFromName,
	})
}
