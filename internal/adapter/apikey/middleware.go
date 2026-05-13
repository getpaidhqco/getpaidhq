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

func (m ApiKeyMiddleware) Authenticate(ctx context.Context, token string) (port.AuthUser, error) {
	if token == "" {
		return port.AuthUser{}, lib.NewCustomError(lib.AuthenticationError, "not allowed", nil)
	}

	apiKey, err := m.apiKeyRepository.FindByKey(ctx, token)
	if err != nil {
		return port.AuthUser{}, err
	}

	return port.AuthUser{
		OrgId:       apiKey.OrgId,
		Id:          apiKey.Id,
		Email:       "",
		PrimaryRole: port.RoleAdmin,
		Roles:       []port.UserRole{port.RoleAdmin},
	}, nil
}
