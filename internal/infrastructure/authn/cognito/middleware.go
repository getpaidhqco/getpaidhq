package cognito

import (
	"errors"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"net/http"
	"payloop/internal/api/authn"
	"payloop/internal/application/lib/logger"
	"payloop/internal/lib"
	"strings"
)

type CognitoMiddleware struct {
	handler lib.RequestHandler
	logger  logger.Logger
	env     lib.Env
	client  Cognito
}

func NewCognitoMiddleware(handler lib.RequestHandler, logger logger.Logger, env lib.Env) CognitoMiddleware {

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

// Setup sets up cognito middleware
func (m CognitoMiddleware) Setup() {
	m.logger.Info("Setting up cognito middleware")
	m.handler.Gin.Use(m.client.Authorize)
}

func (cog *Cognito) Authorize(c *gin.Context) {
	tokenHeader, err := tokenFromAuthHeader(c.Request)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "invalid Authorization header"})
		return
	}
	token, err := cog.VerifyToken(tokenHeader)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "invalid token"})
		return
	}

	orgId := token.Claims.(jwt.MapClaims)["custom:company"].(string)
	userId := token.Claims.(jwt.MapClaims)["sub"].(string)
	email := token.Claims.(jwt.MapClaims)["email"].(string)
	roles := token.Claims.(jwt.MapClaims)["cognito:groups"].([]interface{})

	var roleStrings []authn.UserRole
	for _, role := range roles {
		roleStrings = append(roleStrings, authn.UserRole(strings.ToLower(role.(string))))
	}

	if orgId == "" {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "invalid token"})
		return
	}

	c.Set("token", token)
	c.Set("user", authn.NewUser(orgId, userId, email, roleStrings))
	c.Next()
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
