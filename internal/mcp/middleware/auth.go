package middleware

import (
	"context"
	"errors"
	"payloop/internal/api/authn"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
	"strings"
)

// AuthContext holds the authenticated user and organization information
type AuthContext struct {
	User  authn.User
	OrgId string
}

// AuthService handles authentication for MCP requests
type AuthService struct {
	apiKeyRepository   repositories.ApiKeyRepository
	metadataRepository repositories.MetadataStoreRepository
	logger             logger.Logger
	env                lib.Env
}

// NewAuthService creates a new authentication service
func NewAuthService(
	logger logger.Logger,
	env lib.Env,
	apiKeyRepository repositories.ApiKeyRepository,
	metadataRepository repositories.MetadataStoreRepository,
) *AuthService {
	return &AuthService{
		apiKeyRepository:   apiKeyRepository,
		metadataRepository: metadataRepository,
		logger:             logger,
		env:                env,
	}
}

// ExtractAuthFromMCPRequest extracts authentication information from MCP request arguments
func (s *AuthService) ExtractAuthFromMCPRequest(ctx context.Context, arguments map[string]any) (*AuthContext, error) {
	// Check for authorization token in arguments
	authTokenVal, exists := arguments["x-api-key"]
	if !exists {
		return nil, errors.New("authentication required: authorization parameter missing")
	}

	authToken, ok := authTokenVal.(string)
	if !ok {
		return nil, errors.New("authentication required: authorization must be a string")
	}

	if authToken == "" {
		return nil, errors.New("authentication required: authorization parameter is empty")
	}

	// Try API key authentication first
	if strings.HasPrefix(authToken, "Bearer pk_live_") || strings.HasPrefix(authToken, "Bearer pk_test_") {
		token := strings.TrimPrefix(authToken, "Bearer ")

		// Find API key in repository
		apiKey, err := s.apiKeyRepository.FindByKey(ctx, token)
		if err == nil {
			user := authn.User{
				OrgId:       apiKey.OrgId,
				Id:          apiKey.Id,
				Email:       "",
				PrimaryRole: authn.Admin,
				Roles:       []authn.UserRole{authn.Admin},
			}

			return &AuthContext{
				User:  user,
				OrgId: user.OrgId,
			}, nil
		}
		s.logger.Warn("API key authentication failed", "error", err.Error())
	}

	// Try Clerk OAuth authentication
	if strings.HasPrefix(authToken, "Bearer clerk_") {
		// For Clerk authentication, we would need to implement the token validation
		// This would require importing the Clerk SDK and validating the token
		// For now, we'll return an error indicating that Clerk authentication is not implemented
		s.logger.Warn("Clerk authentication not implemented for MCP")
	}

	return nil, errors.New("invalid authentication token")
}
