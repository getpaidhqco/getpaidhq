package aes_vault

import (
	"encoding/base64"
	"fmt"
	"log"
	"payloop/internal/domain/security"
	"payloop/internal/lib"

	"go.uber.org/fx"
)

// Module provides dependency injection for AES vault implementation
var Module = fx.Options(
	fx.Provide(NewAESVaultFromEnv),
)

// NewAESVaultFromEnv creates a new AES vault from environment variables
func NewAESVaultFromEnv(env lib.Env) (security.TokenVault, error) {
	key := env.TokenVaultAESKey
	if key == "" {
		return nil, fmt.Errorf("TOKEN_VAULT_AES_KEY is required for AES vault")
	}

	// Decode base64 key
	keyBytes, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return nil, fmt.Errorf("failed to decode AES key: %w", err)
	}

	// Ensure 32 bytes for AES-256
	if len(keyBytes) != 32 {
		return nil, fmt.Errorf("AES key must be 32 bytes for AES-256, got %d bytes", len(keyBytes))
	}

	vault, err := NewAESTokenVault(string(keyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create AES vault: %w", err)
	}

	log.Printf("Successfully initialized AES token vault")
	return vault, nil
}