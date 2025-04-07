package apikey

import (
	"context"
	"github.com/gin-gonic/gin"
	"net/http"
	apiauthn "payloop/internal/api/authn"
	"payloop/internal/application/lib/authn"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
)

type ApiKeyMiddleware struct {
	handler          lib.RequestHandler
	apiKeyRepository repositories.ApiKeyRepository
	logger           logger.Logger
	env              lib.Env
}

func NewApiKeyMiddleware(
	handler lib.RequestHandler,
	logger logger.Logger,
	env lib.Env,
	apiKeyRepository repositories.ApiKeyRepository,
) authn.Authenticator {

	return ApiKeyMiddleware{
		apiKeyRepository: apiKeyRepository,
		handler:          handler,
		logger:           logger,
		env:              env,
	}
}

// Setup sets up apiKey middleware
func (m ApiKeyMiddleware) Setup() {
	m.logger.Info("Setting up apiKey middleware")
	m.handler.Gin.Use(func(c *gin.Context) {
		if isPublicPath(c.Request.URL.Path) {
			c.Next()
			return
		}

		apiKey := c.GetHeader("x-api-key")

		user, err := m.Authenticate(c.Request.Context(), apiKey)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "invalid api key"})
			return
		}
		c.Set("user", user)
		c.Next()
	})
}

func isPublicPath(path string) bool {
	for _, publicPath := range authn.PublicPaths {
		if path == publicPath {
			return true
		}
	}
	return false
}

func (m ApiKeyMiddleware) Authenticate(ctx context.Context, token string) (apiauthn.User, error) {
	m.logger.Debugf("ApiKey Auth: [%s]", token[:5])

	apiKey, err := m.apiKeyRepository.FindByKey(ctx, token)
	if err != nil {
		return apiauthn.User{}, err
	}

	return apiauthn.User{
		OrgId:       apiKey.OrgId,
		Id:          apiKey.Id,
		Email:       "",
		PrimaryRole: apiauthn.Admin,
		Roles:       []apiauthn.UserRole{apiauthn.Admin},
	}, nil
}
