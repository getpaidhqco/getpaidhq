package aes_vault

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"payloop/internal/domain/security"
	"time"
)

// AESTokenVault implements TokenVault using AES-256-GCM encryption
type AESTokenVault struct {
	key        []byte
	keyVersion string
}

// NewAESTokenVault creates a new AES token vault
func NewAESTokenVault(key string) (*AESTokenVault, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes for AES-256, got %d bytes", len(key))
	}
	
	return &AESTokenVault{
		key:        []byte(key),
		keyVersion: "v1", // Could be derived from key hash or config
	}, nil
}

// Encrypt encrypts a payment token using AES-256-GCM
func (v *AESTokenVault) Encrypt(ctx context.Context, plaintext string) (string, error) {
	if plaintext == "" {
		return "", security.NewVaultError(security.VaultTypeAES, "encrypt", fmt.Errorf("plaintext cannot be empty"))
	}

	block, err := aes.NewCipher(v.key)
	if err != nil {
		return "", security.NewVaultError(security.VaultTypeAES, "encrypt", fmt.Errorf("failed to create cipher: %w", err))
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", security.NewVaultError(security.VaultTypeAES, "encrypt", fmt.Errorf("failed to create GCM: %w", err))
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", security.NewVaultError(security.VaultTypeAES, "encrypt", fmt.Errorf("failed to generate nonce: %w", err))
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	encryptedData := base64.StdEncoding.EncodeToString(ciphertext)

	// Create secure token with metadata
	secureToken := security.SecureToken{
		EncryptedData: encryptedData,
		Metadata: security.TokenMetadata{
			VaultType:   security.VaultTypeAES,
			EncryptedAt: time.Now().Unix(),
			KeyVersion:  v.keyVersion,
		},
	}

	// Serialize the complete token
	tokenBytes, err := json.Marshal(secureToken)
	if err != nil {
		return "", security.NewVaultError(security.VaultTypeAES, "encrypt", fmt.Errorf("failed to marshal token: %w", err))
	}

	return base64.StdEncoding.EncodeToString(tokenBytes), nil
}

// Decrypt decrypts a payment token using AES-256-GCM
func (v *AESTokenVault) Decrypt(ctx context.Context, ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", security.NewVaultError(security.VaultTypeAES, "decrypt", fmt.Errorf("ciphertext cannot be empty"))
	}

	// Decode the token
	tokenBytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", security.NewVaultError(security.VaultTypeAES, "decrypt", fmt.Errorf("failed to decode token: %w", err))
	}

	// Unmarshal the secure token
	var secureToken security.SecureToken
	if err := json.Unmarshal(tokenBytes, &secureToken); err != nil {
		return "", security.NewVaultError(security.VaultTypeAES, "decrypt", fmt.Errorf("failed to unmarshal token: %w", err))
	}

	// Verify vault type
	if secureToken.Metadata.VaultType != security.VaultTypeAES {
		return "", security.NewVaultError(security.VaultTypeAES, "decrypt", fmt.Errorf("invalid vault type: %s", secureToken.Metadata.VaultType))
	}

	// Decode the encrypted data
	data, err := base64.StdEncoding.DecodeString(secureToken.EncryptedData)
	if err != nil {
		return "", security.NewVaultError(security.VaultTypeAES, "decrypt", fmt.Errorf("failed to decode encrypted data: %w", err))
	}

	block, err := aes.NewCipher(v.key)
	if err != nil {
		return "", security.NewVaultError(security.VaultTypeAES, "decrypt", fmt.Errorf("failed to create cipher: %w", err))
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", security.NewVaultError(security.VaultTypeAES, "decrypt", fmt.Errorf("failed to create GCM: %w", err))
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", security.NewVaultError(security.VaultTypeAES, "decrypt", fmt.Errorf("ciphertext too short"))
	}

	nonce, encrypted := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return "", security.NewVaultError(security.VaultTypeAES, "decrypt", fmt.Errorf("failed to decrypt: %w", err))
	}

	return string(plaintext), nil
}

// IsHealthy checks if the AES vault is functioning properly
func (v *AESTokenVault) IsHealthy(ctx context.Context) error {
	// Test encryption and decryption with a known value
	testValue := "health_check_test"
	
	encrypted, err := v.Encrypt(ctx, testValue)
	if err != nil {
		return security.NewVaultError(security.VaultTypeAES, "health_check", fmt.Errorf("failed to encrypt test value: %w", err))
	}
	
	decrypted, err := v.Decrypt(ctx, encrypted)
	if err != nil {
		return security.NewVaultError(security.VaultTypeAES, "health_check", fmt.Errorf("failed to decrypt test value: %w", err))
	}
	
	if decrypted != testValue {
		return security.NewVaultError(security.VaultTypeAES, "health_check", fmt.Errorf("decrypted value does not match original"))
	}
	
	return nil
}

// GetVaultType returns the vault type
func (v *AESTokenVault) GetVaultType() security.VaultType {
	return security.VaultTypeAES
}

// GenerateAESKey generates a random 32-byte AES key for development/setup
func GenerateAESKey() string {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		panic(fmt.Sprintf("Failed to generate AES key: %v", err))
	}
	return base64.StdEncoding.EncodeToString(key)
}