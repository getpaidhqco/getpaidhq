package entities

import (
	"context"
	"fmt"
	"payloop/internal/domain/security"
	"time"
)

// SecurePaymentMethod provides secure token handling for payment methods
type SecurePaymentMethod struct {
	PaymentMethod
	tokenVault security.TokenVault
}

// NewSecurePaymentMethod creates a payment method with secure token handling
func NewSecurePaymentMethod(pm PaymentMethod, vault security.TokenVault) SecurePaymentMethod {
	return SecurePaymentMethod{
		PaymentMethod: pm,
		tokenVault:    vault,
	}
}

// SetToken securely stores the payment token
func (spm *SecurePaymentMethod) SetToken(ctx context.Context, plainTextToken string) error {
	if spm.tokenVault == nil {
		return fmt.Errorf("token vault not configured")
	}

	if plainTextToken == "" {
		return fmt.Errorf("token cannot be empty")
	}

	encryptedToken, err := spm.tokenVault.Encrypt(ctx, plainTextToken)
	if err != nil {
		return fmt.Errorf("failed to encrypt token: %w", err)
	}

	spm.Token = encryptedToken
	spm.UpdatedAt = time.Now().UTC()
	return nil
}

// GetToken securely retrieves the decrypted payment token
func (spm SecurePaymentMethod) GetToken(ctx context.Context) (string, error) {
	if spm.tokenVault == nil {
		return "", fmt.Errorf("token vault not configured")
	}

	if spm.Token == "" {
		return "", fmt.Errorf("no token stored")
	}

	plainTextToken, err := spm.tokenVault.Decrypt(ctx, spm.Token)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt token: %w", err)
	}

	return plainTextToken, nil
}

// IsTokenValid checks if the stored token can be decrypted
func (spm SecurePaymentMethod) IsTokenValid(ctx context.Context) bool {
	_, err := spm.GetToken(ctx)
	return err == nil
}

// RedactedToken returns a redacted version of the token for logging/display
func (spm SecurePaymentMethod) RedactedToken() string {
	if len(spm.Token) <= 8 {
		return "****"
	}
	return spm.Token[:4] + "****" + spm.Token[len(spm.Token)-4:]
}

// ToEntity returns the underlying PaymentMethod entity
func (spm SecurePaymentMethod) ToEntity() PaymentMethod {
	return spm.PaymentMethod
}

// GetVaultType returns the vault type used for this payment method
func (spm SecurePaymentMethod) GetVaultType() security.VaultType {
	if spm.tokenVault == nil {
		return ""
	}
	return spm.tokenVault.GetVaultType()
}

// GetTokenVault returns the token vault used by this payment method
func (spm SecurePaymentMethod) GetTokenVault() security.TokenVault {
	return spm.tokenVault
}

// TokenSummary returns a summary of the token for debugging/auditing
func (spm SecurePaymentMethod) TokenSummary() map[string]interface{} {
	return map[string]interface{}{
		"has_token":         spm.Token != "",
		"token_length":      len(spm.Token),
		"vault_type":        spm.GetVaultType(),
		"redacted":          spm.RedactedToken(),
		"payment_method_id": spm.Id,
		"psp":               spm.Psp,
		"updated_at":        spm.UpdatedAt,
	}
}

// PaymentMethodSecurityService wraps standard PaymentMethod operations with security
type PaymentMethodSecurityService struct {
	vault security.TokenVault
}

// NewPaymentMethodSecurityService creates a new payment method security service
func NewPaymentMethodSecurityService(vault security.TokenVault) *PaymentMethodSecurityService {
	return &PaymentMethodSecurityService{vault: vault}
}

// CreateSecurePaymentMethod creates a new payment method with encrypted token
func (pms *PaymentMethodSecurityService) CreateSecurePaymentMethod(ctx context.Context, pm PaymentMethod, token string) (SecurePaymentMethod, error) {
	// Ensure the payment method has an ID
	if pm.Id == "" {
		return SecurePaymentMethod{}, fmt.Errorf("payment method ID is required")
	}

	// Ensure status is set if not already
	if pm.Status == "" {
		return SecurePaymentMethod{}, fmt.Errorf("payment method ID is required")
	}

	// Set or update timestamps
	now := time.Now().UTC()
	if pm.CreatedAt.IsZero() {
		pm.CreatedAt = now
	}
	pm.UpdatedAt = now

	securePM := NewSecurePaymentMethod(pm, pms.vault)
	err := (&securePM).SetToken(ctx, token)
	if err != nil {
		return SecurePaymentMethod{}, fmt.Errorf("failed to set secure token: %w", err)
	}

	return securePM, nil
}

// LoadSecurePaymentMethod loads an existing payment method with vault access
func (pms *PaymentMethodSecurityService) LoadSecurePaymentMethod(pm PaymentMethod) SecurePaymentMethod {
	return NewSecurePaymentMethod(pm, pms.vault)
}

// UpdateToken updates the token for an existing payment method
func (pms *PaymentMethodSecurityService) UpdateToken(ctx context.Context, pm SecurePaymentMethod, newToken string) error {
	return (&pm).SetToken(ctx, newToken)
}

// ValidateTokenAccess validates that a token can be successfully decrypted
func (pms *PaymentMethodSecurityService) ValidateTokenAccess(ctx context.Context, pm SecurePaymentMethod) error {
	_, err := pm.GetToken(ctx)
	return err
}

// MigrateToNewVault migrates a payment method token from one vault to another
func (pms *PaymentMethodSecurityService) MigrateToNewVault(ctx context.Context, pm SecurePaymentMethod, newVault security.TokenVault) (SecurePaymentMethod, error) {
	// Get the current token
	plainToken, err := pm.GetToken(ctx)
	if err != nil {
		return SecurePaymentMethod{}, fmt.Errorf("failed to decrypt token with current vault: %w", err)
	}

	// Create a new secure payment method with the new vault
	newSecurePM := NewSecurePaymentMethod(pm.PaymentMethod, newVault)

	// Set the token with the new vault
	err = (&newSecurePM).SetToken(ctx, plainToken)
	if err != nil {
		return SecurePaymentMethod{}, fmt.Errorf("failed to encrypt token with new vault: %w", err)
	}

	return newSecurePM, nil
}
