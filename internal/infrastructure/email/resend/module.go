package resend

import (
	"go.uber.org/fx"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/email_providers"
	"payloop/internal/lib"
)

// Module provides the Resend email provider
var Module = fx.Options(
	fx.Provide(NewResendProviderFromConfig),
)

// NewResendProviderFromConfig creates a new Resend email provider from configuration
func NewResendProviderFromConfig(logger logger.Logger, env lib.Env) (email_providers.Provider, error) {
	// Check if email provider is configured
	if env.EmailProvider == "" {
		logger.Warn("Email provider not configured, using no-op provider")
		return nil, nil
	}

	// Check if the configured provider is Resend
	if env.EmailProvider != "resend" {
		logger.Warn("Resend provider not configured, using no-op provider")
		return nil, nil
	}

	// Check if Resend API key is configured
	if env.ResendApiKey == "" {
		logger.Warn("Resend API key not specified, using no-op provider")
		return nil, nil
	}

	// Get from email from environment or use default
	fromEmail := env.EmailFromEmail
	if fromEmail == "" {
		fromEmail = "no-reply@getpaidhq.co"
	}

	logger.Info("Initializing Resend email provider with SDK", "api_key_set", env.ResendApiKey != "", "from_email", fromEmail)

	return NewResendProvider(logger, ResendConfig{
		APIKey:    env.ResendApiKey,
		FromEmail: fromEmail,
	})
}