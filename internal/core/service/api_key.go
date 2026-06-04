package service

import (
	"context"
	"time"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

// ApiKeyService manages programmatic API keys (the `x-api-key` header
// path). Keys are bound to an org; the raw secret is surfaced exactly
// once at creation and never persisted. See lib.HashApiKey for the
// pepper-keyed HMAC used at rest.
type ApiKeyService struct {
	repo   port.ApiKeyRepository
	logger port.Logger

	// pepper is the server-side HMAC key applied before storage. Without
	// it Create cannot mint a key (HashApiKey returns ErrMissingApiKeyPepper).
	// Sourced from env.ApiKeyPepper at wiring time.
	pepper string
}

func NewApiKeyService(repo port.ApiKeyRepository, pepper string, logger port.Logger) *ApiKeyService {
	return &ApiKeyService{repo: repo, logger: logger, pepper: pepper}
}

// Create mints a new key for the org. The returned port.CreatedApiKey pairs
// the persisted ApiKey aggregate with the plaintext raw secret — the raw
// secret is surfaced ONCE here and never persisted, never logged, never
// re-derivable from the row.
func (s *ApiKeyService) Create(ctx context.Context, orgId string, name string) (port.CreatedApiKey, error) {
	secret, err := lib.GenerateApiKeySecret()
	if err != nil {
		return port.CreatedApiKey{}, err
	}
	keyId := lib.GenerateId("sk")
	rawKey := keyId + "_" + secret
	keyHash, err := lib.HashApiKey(rawKey, s.pepper)
	if err != nil {
		s.logger.Error("API key hash failed (check API_KEY_PEPPER)", "err", err.Error())
		return port.CreatedApiKey{}, err
	}

	created, err := s.repo.Create(ctx, domain.ApiKey{
		OrgId:     orgId,
		Id:        keyId,
		Name:      name,
		KeyHash:   keyHash,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		s.logger.Error("Failed to create API key", "org_id", orgId, "err", err)
		return port.CreatedApiKey{}, err
	}
	return port.CreatedApiKey{ApiKey: created, Key: rawKey}, nil
}

func (s *ApiKeyService) List(ctx context.Context, orgId string, p domain.Pagination) ([]domain.ApiKey, int, error) {
	return s.repo.List(ctx, orgId, p)
}

func (s *ApiKeyService) Delete(ctx context.Context, orgId string, id string) error {
	return s.repo.Delete(ctx, orgId, id)
}
