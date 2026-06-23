package service

import (
	"context"
	"encoding/json"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

// GatewayFactory creates payment gateway instances from stored PSP configuration.
// It uses a registry of GatewayAdapter implementations to avoid importing adapter
// packages directly. This is the single place stored gateway credentials are
// decrypted — the plaintext exists only inside domain.Secret values handed to
// the adapter, never on a response or logging path.
type GatewayFactory struct {
	pspRepository port.PspRepository
	cipher        port.SecretCipher
	logger        port.Logger
	adapters      map[domain.Gateway]port.GatewayAdapter
}

func NewGatewayFactory(
	pspRepository port.PspRepository,
	cipher port.SecretCipher,
	logger port.Logger,
	adapters map[domain.Gateway]port.GatewayAdapter,
) *GatewayFactory {
	return &GatewayFactory{
		pspRepository: pspRepository,
		cipher:        cipher,
		logger:        logger,
		adapters:      adapters,
	}
}

func (s *GatewayFactory) NewGateway(ctx context.Context, orgId string, id string) (port.PaymentGateway, error) {
	psp, err := s.pspRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Errorf("Failed to get Gateway[%s] - %s", id, err.Error())
		return nil, err
	}

	creds, err := DecryptGatewayCredentials(s.cipher, psp)
	if err != nil {
		s.logger.Errorf("Failed to open credentials for Gateway[%s] - %s", id, err.Error())
		return nil, err
	}

	adapter, ok := s.adapters[psp.PspId]
	if !ok {
		return nil, lib.NewCustomError(lib.BadRequestError, "Invalid payment processor", nil)
	}

	return adapter.CreateGateway(psp.Config, creds)
}

func (s *GatewayFactory) NewWebhookParser(psp domain.Gateway) domain.WebhookParser {
	adapter, ok := s.adapters[psp]
	if !ok {
		return nil
	}
	return adapter.CreateWebhookParser()
}

// DecryptGatewayCredentials opens a PspConfig's sealed credentials into
// Secret-typed values. Shared with adapter-side factories (e.g. the Paystack
// webhook parser) so decryption stays uniform. A gateway with no stored
// credentials (e.g. the in-memory test gateway) yields an empty map.
func DecryptGatewayCredentials(cipher port.SecretCipher, psp domain.PspConfig) (map[string]domain.Secret, error) {
	if psp.EncryptedCredentials == "" {
		return map[string]domain.Secret{}, nil
	}
	if cipher == nil {
		return nil, lib.NewCustomError(lib.InternalError, "SECRETS_ENCRYPTION_KEY is not configured; cannot open gateway credentials", nil)
	}
	plaintext, err := cipher.Decrypt(psp.OrgId, psp.Id, psp.EncryptedCredentials)
	if err != nil {
		return nil, err
	}
	var creds map[string]domain.Secret
	if err := json.Unmarshal(plaintext, &creds); err != nil {
		return nil, err
	}
	return creds, nil
}
