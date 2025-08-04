package middlewares

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	apiauthn "payloop/internal/api/authn"
	"payloop/internal/application/lib/authn"
	"payloop/internal/application/lib/logger"
	"payloop/internal/lib"
)

// AuthnWrapperMiddleware combines authn middlewares so that we can use token based and api key auth
type AuthnWrapperMiddleware struct {
	handler   lib.RequestHandler
	authnList []authn.Authenticator `group:"authenticators"`
	logger    logger.Logger
	env       lib.Env
}

func NewAuthnWrapperMiddleware(
	authenticators []authn.Authenticator,
	handler lib.RequestHandler,
	logger logger.Logger,
	env lib.Env,
) AuthnWrapperMiddleware {
	return AuthnWrapperMiddleware{
		authnList: authenticators,
		handler:   handler,
		logger:    logger,
		env:       env,
	}
}

// Setup sets up cors middleware
func (m AuthnWrapperMiddleware) Setup() {
	m.logger.Info("Setting up authn wrapper middleware")
	m.handler.Gin.Use(func(c *gin.Context) {
		if authn.IsPublicPath(c.Request.URL.Path) {
			c.Next()
			return
		}
		// Create a flag to track if authentication is successful
		isAuthenticated := false

		for _, authenticator := range m.authnList {
			// TODO
			// We need a way to extract the token from the request header without having the Authenticator know about the header
			// otherwise we code Gin dependency into the Authenticator interface.
			// For now, check both Authorization header and x-api-key header
			token := c.GetHeader("Authorization")
			if token == "" {
				// If Authorization header is empty, check x-api-key header
				token = c.GetHeader("x-api-key")
			}

			user, err := authenticator.Authenticate(c.Request.Context(), token)
			if err != nil {
				// special case for onboarding required
				if errors.Is(err, apiauthn.ErrOnboardingRequired) &&
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

		// If neither middleware authenticated, abort the request
		if !isAuthenticated {
			m.logger.Error("Authentication failed", "message", "unauthorized access")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
			return
		}

		// If authenticated, proceed to the next handler
		c.Next()
	})
}
