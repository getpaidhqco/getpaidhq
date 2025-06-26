package security

import (
	"context"
	"fmt"
)

// TokenVault provides secure token encryption/decryption interface
type TokenVault interface {
	// Encrypt encrypts a plaintext token and returns the encrypted representation
	Encrypt(ctx context.Context, plaintext string) (string, error)
	
	// Decrypt decrypts an encrypted token back to plaintext
	Decrypt(ctx context.Context, ciphertext string) (string, error)
	
	// IsHealthy checks if the vault service is available
	IsHealthy(ctx context.Context) error
	
	// GetVaultType returns the type of vault implementation
	GetVaultType() VaultType
}

// VaultType represents different vault implementation types
type VaultType string

const (
	VaultTypeAES       VaultType = "aes"
	VaultTypeAWS       VaultType = "aws_secrets_manager"
	VaultTypeHashiCorp VaultType = "hashicorp_vault"
)

// VaultConfig contains configuration for vault implementations
type VaultConfig struct {
	Type           VaultType `json:"type"`
	AESKey         string    `json:"aes_key,omitempty"`
	AWSRegion      string    `json:"aws_region,omitempty"`
	AWSSecretPath  string    `json:"aws_secret_path,omitempty"`
}

// TokenMetadata contains metadata about encrypted tokens
type TokenMetadata struct {
	VaultType     VaultType `json:"vault_type"`
	EncryptedAt   int64     `json:"encrypted_at"`
	KeyVersion    string    `json:"key_version,omitempty"`
	SecretPath    string    `json:"secret_path,omitempty"`
}

// SecureToken represents an encrypted token with metadata
type SecureToken struct {
	EncryptedData string        `json:"encrypted_data"`
	Metadata      TokenMetadata `json:"metadata"`
}

// VaultError represents vault-specific errors
type VaultError struct {
	VaultType VaultType
	Operation string
	Err       error
}

func (e *VaultError) Error() string {
	return fmt.Sprintf("vault error [%s] during %s: %v", e.VaultType, e.Operation, e.Err)
}

func (e *VaultError) Unwrap() error {
	return e.Err
}

// NewVaultError creates a new vault error
func NewVaultError(vaultType VaultType, operation string, err error) error {
	return &VaultError{
		VaultType: vaultType,
		Operation: operation,
		Err:       err,
	}
}