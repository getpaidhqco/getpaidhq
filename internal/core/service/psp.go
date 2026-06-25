package service

import (
	"context"
	"encoding/json"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

type PspService struct {
	pspRepository port.PspRepository
	cipher        port.SecretCipher
	pubsub        port.PubSub
	logger        port.Logger
}

func NewPspService(
	pspRepository port.PspRepository,
	cipher port.SecretCipher,
	logger port.Logger,
	pubsub port.PubSub,
) *PspService {
	return &PspService{
		pspRepository: pspRepository,
		cipher:        cipher,
		logger:        logger,
		pubsub:        pubsub,
	}
}

// CreateGateway stores a PSP configuration. The non-secret Config map is
// stored readable; Credentials are sealed with the SecretCipher (AAD-bound
// to org + gateway id) before the row is written, so no plaintext secret
// ever reaches the database. Nothing is written to the settings table —
// gateway credentials are deliberately NOT readable back through any API.
func (s *PspService) CreateGateway(ctx context.Context, input port.CreateGatewayInput) (domain.PspConfig, error) {
	if len(input.Credentials) == 0 {
		return domain.PspConfig{}, lib.NewCustomError(lib.BadRequestError, "credentials are required", nil)
	}

	id := lib.GenerateId("psp")

	credsJson, err := json.Marshal(domain.RevealMap(input.Credentials))
	if err != nil {
		return domain.PspConfig{}, err
	}
	envelope, err := s.cipher.Encrypt(input.OrgId, id, credsJson)
	if err != nil {
		s.logger.Errorf("Failed to encrypt gateway credentials - %s", err.Error())
		return domain.PspConfig{}, err
	}

	psp, err := s.pspRepository.Create(ctx,
		domain.PspConfig{
			OrgId:                input.OrgId,
			Id:                   id,
			Name:                 input.Name,
			PspId:                input.PspId,
			Active:               true,
			Config:               input.Config,
			EncryptedCredentials: envelope,
		})
	if err != nil {
		s.logger.Errorf("Failed to create psp - %s", err.Error())
		return domain.PspConfig{}, err
	}

	return psp, nil
}
