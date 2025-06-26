package aws_vault

import (
	"context"
	"fmt"
	"log"
	"payloop/internal/domain/security"
	"payloop/internal/lib"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"go.uber.org/fx"
)

// Module provides dependency injection for AWS vault implementation
var Module = fx.Options(
	fx.Provide(NewAWSVaultFromEnv),
)

// NewAWSVaultFromEnv creates a new AWS vault from environment variables
func NewAWSVaultFromEnv(env lib.Env) (security.TokenVault, error) {
	region := env.TokenVaultAWSRegion
	if region == "" {
		return nil, fmt.Errorf("TOKEN_VAULT_AWS_REGION is required for AWS vault")
	}

	secretPath := env.TokenVaultAWSPath
	if secretPath == "" {
		secretPath = "payloop/payment-tokens"
	}

	// Load AWS config
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := secretsmanager.NewFromConfig(cfg)
	vault := NewAWSSecretsVault(client, region, secretPath)

	// Test vault health on startup
	if err := vault.IsHealthy(context.Background()); err != nil {
		log.Printf("Warning: AWS Secrets Manager vault health check failed: %v", err)
		// Continue anyway for development, but log the warning
	}

	log.Printf("Successfully initialized AWS Secrets Manager token vault")
	return vault, nil
}