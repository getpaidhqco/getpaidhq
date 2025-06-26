package aws_vault

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"payloop/internal/domain/security"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
)

// AWSSecretsVault implements TokenVault using AWS Secrets Manager
type AWSSecretsVault struct {
	client     *secretsmanager.Client
	region     string
	secretPath string
}

// NewAWSSecretsVault creates a new AWS Secrets Manager vault
func NewAWSSecretsVault(client *secretsmanager.Client, region, secretPath string) *AWSSecretsVault {
	return &AWSSecretsVault{
		client:     client,
		region:     region,
		secretPath: secretPath,
	}
}

// TokenSecret represents a stored token in AWS Secrets Manager
type TokenSecret struct {
	Token       string                `json:"token"`
	Metadata    security.TokenMetadata `json:"metadata"`
	CreatedAt   string                `json:"created_at"`
	UpdatedAt   string                `json:"updated_at"`
}

// Encrypt stores the token in AWS Secrets Manager and returns a reference ID
func (v *AWSSecretsVault) Encrypt(ctx context.Context, plaintext string) (string, error) {
	if plaintext == "" {
		return "", security.NewVaultError(security.VaultTypeAWS, "encrypt", fmt.Errorf("plaintext cannot be empty"))
	}

	// Create a unique secret name
	secretName := fmt.Sprintf("%s/payment-token-%s", v.secretPath, generateSecureID())
	
	// Create the secret payload
	secret := TokenSecret{
		Token: plaintext,
		Metadata: security.TokenMetadata{
			VaultType:   security.VaultTypeAWS,
			EncryptedAt: time.Now().Unix(),
			SecretPath:  secretName,
		},
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	
	secretValue, err := json.Marshal(secret)
	if err != nil {
		return "", security.NewVaultError(security.VaultTypeAWS, "encrypt", fmt.Errorf("failed to marshal secret: %w", err))
	}
	
	// Store in AWS Secrets Manager
	_, err = v.client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
		Name:         aws.String(secretName),
		SecretString: aws.String(string(secretValue)),
		Description:  aws.String("Payment method token for Payloop"),
		Tags: []types.Tag{
			{
				Key:   aws.String("Application"),
				Value: aws.String("Payloop"),
			},
			{
				Key:   aws.String("Purpose"),
				Value: aws.String("PaymentToken"),
			},
			{
				Key:   aws.String("CreatedAt"),
				Value: aws.String(time.Now().UTC().Format(time.RFC3339)),
			},
		},
	})
	
	if err != nil {
		return "", security.NewVaultError(security.VaultTypeAWS, "encrypt", fmt.Errorf("failed to store secret: %w", err))
	}
	
	// Create secure token reference
	secureToken := security.SecureToken{
		EncryptedData: secretName, // Store the secret name/path as the encrypted data
		Metadata: security.TokenMetadata{
			VaultType:   security.VaultTypeAWS,
			EncryptedAt: time.Now().Unix(),
			SecretPath:  secretName,
		},
	}

	// Serialize the token reference
	tokenBytes, err := json.Marshal(secureToken)
	if err != nil {
		return "", security.NewVaultError(security.VaultTypeAWS, "encrypt", fmt.Errorf("failed to marshal token reference: %w", err))
	}

	return base64.StdEncoding.EncodeToString(tokenBytes), nil
}

// Decrypt retrieves the token from AWS Secrets Manager using the reference ID
func (v *AWSSecretsVault) Decrypt(ctx context.Context, ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", security.NewVaultError(security.VaultTypeAWS, "decrypt", fmt.Errorf("ciphertext cannot be empty"))
	}

	// Decode the token reference
	tokenBytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", security.NewVaultError(security.VaultTypeAWS, "decrypt", fmt.Errorf("failed to decode token reference: %w", err))
	}

	// Unmarshal the secure token reference
	var secureToken security.SecureToken
	if err := json.Unmarshal(tokenBytes, &secureToken); err != nil {
		return "", security.NewVaultError(security.VaultTypeAWS, "decrypt", fmt.Errorf("failed to unmarshal token reference: %w", err))
	}

	// Verify vault type
	if secureToken.Metadata.VaultType != security.VaultTypeAWS {
		return "", security.NewVaultError(security.VaultTypeAWS, "decrypt", fmt.Errorf("invalid vault type: %s", secureToken.Metadata.VaultType))
	}

	secretName := secureToken.EncryptedData

	// Retrieve from AWS Secrets Manager
	result, err := v.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	})
	
	if err != nil {
		return "", security.NewVaultError(security.VaultTypeAWS, "decrypt", fmt.Errorf("failed to retrieve secret: %w", err))
	}
	
	// Parse the secret
	var secret TokenSecret
	err = json.Unmarshal([]byte(*result.SecretString), &secret)
	if err != nil {
		return "", security.NewVaultError(security.VaultTypeAWS, "decrypt", fmt.Errorf("failed to unmarshal secret: %w", err))
	}
	
	return secret.Token, nil
}

// IsHealthy checks if AWS Secrets Manager is accessible
func (v *AWSSecretsVault) IsHealthy(ctx context.Context) error {
	// Try to list secrets to verify connectivity
	_, err := v.client.ListSecrets(ctx, &secretsmanager.ListSecretsInput{
		MaxResults: aws.Int32(1),
		Filters: []types.Filter{
			{
				Key:    types.FilterNameStringTypeTagKey,
				Values: []string{"Application"},
			},
		},
	})
	
	if err != nil {
		return security.NewVaultError(security.VaultTypeAWS, "health_check", fmt.Errorf("failed to connect to AWS Secrets Manager: %w", err))
	}
	
	return nil
}

// GetVaultType returns the vault type
func (v *AWSSecretsVault) GetVaultType() security.VaultType {
	return security.VaultTypeAWS
}

// CleanupExpiredTokens removes expired token secrets (utility method)
func (v *AWSSecretsVault) CleanupExpiredTokens(ctx context.Context, maxAge time.Duration) error {
	// List all Payloop payment token secrets
	secrets, err := v.client.ListSecrets(ctx, &secretsmanager.ListSecretsInput{
		Filters: []types.Filter{
			{
				Key:    types.FilterNameStringTypeTagKey,
				Values: []string{"Application"},
			},
			{
				Key:    types.FilterNameStringTypeTagValue,
				Values: []string{"Payloop"},
			},
		},
	})
	
	if err != nil {
		return fmt.Errorf("failed to list secrets: %w", err)
	}
	
	now := time.Now()
	for _, secret := range secrets.SecretList {
		if secret.CreatedDate != nil && now.Sub(*secret.CreatedDate) > maxAge {
			// Schedule secret for deletion
			_, err := v.client.DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{
				SecretId:                   secret.Name,
				RecoveryWindowInDays:       aws.Int64(7), // 7-day recovery window
				ForceDeleteWithoutRecovery: aws.Bool(false),
			})
			
			if err != nil {
				// Log error but continue with other secrets
				continue
			}
		}
	}
	
	return nil
}

// generateSecureID creates a cryptographically secure random ID
func generateSecureID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if random generation fails
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return base64.URLEncoding.EncodeToString(bytes)[:22] // Remove padding
}