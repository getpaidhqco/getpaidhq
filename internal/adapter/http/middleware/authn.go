package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"payloop/internal/core/port"
	"payloop/internal/lib"
)

// AuthnWrapperMiddleware combines authn middlewares so that token-based and API key auth
// can be used interchangeably.
type AuthnWrapperMiddleware struct {
	handler   lib.RequestHandler
	authnList []port.Authenticator
	logger    port.Logger
	env       lib.Env
}

// NewAuthnWrapperMiddleware creates a new AuthnWrapperMiddleware.
func NewAuthnWrapperMiddleware(
	authenticators []port.Authenticator,
	handler lib.RequestHandler,
	logger port.Logger,
	env lib.Env,
) AuthnWrapperMiddleware {
	return AuthnWrapperMiddleware{
		authnList: authenticators,
		handler:   handler,
		logger:    logger,
		env:       env,
	}
}

// Setup registers the authentication wrapper middleware on the gin engine.
func (m AuthnWrapperMiddleware) Setup() {
	m.logger.Info("setting up authn wrapper middleware")
	m.handler.Gin.Use(func(c *gin.Context) {
		isAuthenticated := false

		for _, authenticator := range m.authnList {
			token := c.GetHeader("Authorization")
			if token == "" {
				token = c.GetHeader("x-api-key")
			}

			user, err := authenticator.Authenticate(c.Request.Context(), token)
			if err != nil {
				// special case for onboarding required
				if errors.Is(err, port.ErrOnboardingRequired) &&
					c.Request.Method == http.MethodPost &&
					c.FullPath() == "/api/organizations" {
					isAuthenticated = true
					c.Set("user", user)
					break
				}
				continue
			}

			c.Set("user", user)
			isAuthenticated = true
			break
		}

		if !isAuthenticated {
			m.logger.Error("authentication failed", "message", "unauthorized access")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
			return
		}

		c.Next()
	})
}
