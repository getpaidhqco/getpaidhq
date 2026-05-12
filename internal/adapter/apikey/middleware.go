package apikey

import (
	"context"
	"github.com/gin-gonic/gin"
	"net/http"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
	"slices"
)

type ApiKeyMiddleware struct {
	handler          lib.RequestHandler
	apiKeyRepository port.ApiKeyRepository
	logger           port.Logger
	env              lib.Env
}

func NewApiKeyMiddleware(
	handler lib.RequestHandler,
	logger port.Logger,
	env lib.Env,
	apiKeyRepository port.ApiKeyRepository,
) port.Authenticator {

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
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "not allowed"})
			return
		}
		c.Set("user", user)
		c.Next()
	})
}

func isPublicPath(path string) bool {
	return slices.Contains(port.PublicPaths, path)
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
