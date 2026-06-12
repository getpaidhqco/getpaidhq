package cognito

import (
	"context"
	"errors"
	"strings"

	"github.com/dgrijalva/jwt-go"

	"getpaidhq/internal/core/port"
)

type CognitoMiddleware struct {
	logger port.Logger
	client Cognito
}

func NewCognitoMiddleware(logger port.Logger, cfg Config) port.Authenticator {

	client, err := NewCognitoClient(cfg)
	if err != nil {
		logger.Error("Error initializing cognito client", "error", err)
		panic(err)
	}

	return CognitoMiddleware{
		logger: logger,
		client: client,
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
