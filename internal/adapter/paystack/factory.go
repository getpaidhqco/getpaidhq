package paystack

import (
	"context"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
	"getpaidhq/internal/lib"
)

// PaystackFactory builds a Paystack gateway for an org from its stored PSP
// row — the webhook-side twin of service.GatewayFactory. Credentials are
// decrypted via the shared service helper and stay Secret-typed inside
// PaystackConfig.
type PaystackFactory struct {
	pspRepository port.PspRepository
	cipher        port.SecretCipher
	logger        port.Logger
}

func NewPaystackFactory(
	pspRepository port.PspRepository,
	cipher port.SecretCipher,
	logger port.Logger,
) PaystackFactory {
	return PaystackFactory{
		pspRepository: pspRepository,
		cipher:        cipher,
		logger:        logger,
	}
}

func (s PaystackFactory) New(ctx context.Context, orgId string) (domain.GatewayProvider, error) {

	psp, err := s.pspRepository.FindById(ctx, orgId, string(domain.Paystack))
	if err != nil {
		return nil, err
	}

	creds, err := service.DecryptGatewayCredentials(s.cipher, psp)
	if err != nil {
		s.logger.Error("Failed to open gateway credentials", "error", err)
		return nil, err
	}

	config, err := ParseConfig(psp.Config, creds)
	if err != nil {
		return nil, lib.NewCustomError(lib.ValidationError, "invalid Config", err)
	}

	return NewPaystackGateway(s.logger, config), nil
}
