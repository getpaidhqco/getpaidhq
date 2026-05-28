package apikey

import (
	"context"

	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

type ApiKeyMiddleware struct {
	apiKeyRepository port.ApiKeyRepository
	logger           port.Logger
	env              lib.Env
}

func NewApiKeyMiddleware(
	logger port.Logger,
	env lib.Env,
	apiKeyRepository port.ApiKeyRepository,
) port.Authenticator {
	return ApiKeyMiddleware{
		apiKeyRepository: apiKeyRepository,
		logger:           logger,
		env:              env,
	}
}

// Authenticate verifies an API key by hashing the incoming raw key
// with the configured server pepper, then looking up the resulting
// hash. The lookup is constant-time at the index level; the previous
// plaintext-equality lookup leaked existence via timing.
//
// On any failure we return the same opaque error — never confirm
// whether the key existed but had the wrong value.
func (m ApiKeyMiddleware) Authenticate(ctx context.Context, token string) (port.AuthUser, error) {
	if token == "" {
		return port.AuthUser{}, lib.NewCustomError(lib.AuthenticationError, "not allowed", nil)
	}

	keyHash, err := lib.HashApiKey(token, m.env.ApiKeyPepper)
	if err != nil {
		// Missing pepper is a server-config error, not a credential
		// problem — but surface as a generic authn failure so the
		// shape of the response doesn't help fingerprint our setup.
		m.logger.Error("api key hash failed (check API_KEY_PEPPER)", "err", err.Error())
		return port.AuthUser{}, lib.NewCustomError(lib.AuthenticationError, "not allowed", nil)
	}

	apiKey, err := m.apiKeyRepository.FindByKey(ctx, keyHash)
	if err != nil {
		return port.AuthUser{}, lib.NewCustomError(lib.AuthenticationError, "not allowed", nil)
	}

	return port.AuthUser{
		OrgId:       apiKey.OrgId,
		Id:          apiKey.Id,
		Email:       "",
		PrimaryRole: port.RoleAdmin,
		Roles:       []port.UserRole{port.RoleAdmin},
	}, nil
}
