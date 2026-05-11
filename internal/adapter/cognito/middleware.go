package cognito

import (
	"context"
	"errors"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"net/http"
	"payloop/internal/core/port"
	"payloop/internal/lib"
	"strings"
)

type CognitoMiddleware struct {
	handler lib.RequestHandler
	logger  port.Logger
	env     lib.Env
	client  Cognito
}

func NewCognitoMiddleware(handler lib.RequestHandler, logger port.Logger, env lib.Env) port.Authenticator {

	client, err := NewCognitoClient(env)
	if err != nil {
		logger.Error("Error initializing cognito client", "error", err)
		panic(err)
	}

	return CognitoMiddleware{
		handler: handler,
		logger:  logger,
		env:     env,
		client:  client,
	}
}

func (m CognitoMiddleware) Authenticate(ctx context.Context, token string) (port.AuthUser, error) {
	t, err := m.client.VerifyToken(token)
	if err != nil {
		return port.AuthUser{}, err
	}
	orgId := t.Claims.(jwt.MapClaims)["custom:company"].(string)
	userId := t.Claims.(jwt.MapClaims)["sub"].(string)
	email := t.Claims.(jwt.MapClaims)["email"].(string)
	roles := t.Claims.(jwt.MapClaims)["cognito:groups"].([]any)

	var roleStrings []port.UserRole
	for _, role := range roles {
		roleStrings = append(roleStrings, port.UserRole(strings.ToLower(role.(string))))
	}

	if orgId == "" {
		return port.AuthUser{}, errors.New("invalid token")
	}

	return port.NewAuthUser(orgId, userId, email, roleStrings), nil
}

// Setup sets up cognito middleware
func (m CognitoMiddleware) Setup() {
	m.logger.Info("Setting up cognito middleware")
	m.handler.Gin.Use(func(c *gin.Context) {
		if port.IsPublicPath(c.Request.URL.Path) {
			c.Next()
			return
		}

		tokenHeader, err := tokenFromAuthHeader(c.Request)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "invalid Authorization header"})
			return
		}

		user, err := m.Authenticate(c.Request.Context(), tokenHeader)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "not allowed"})
			return
		}
		c.Set("user", user)
		c.Next()
	})
}

func tokenFromAuthHeader(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("no token")
	}

	parts := strings.Fields(authHeader)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return "", errors.New("invalid Authorization header format")
	}

	return parts[1], nil
}
