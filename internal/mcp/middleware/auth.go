package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"payloop/internal/api/authn"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
	"strings"
	"time"

	"github.com/clerk/clerk-sdk-go/v2/jwt"
)

// AuthContext holds the authenticated user and organization information
type AuthContext struct {
	User  authn.User
	OrgId string
}

// JWKSCache holds cached JWKS data
type JWKSCache struct {
	Keys      []JWK     `json:"keys"`
	ExpiresAt time.Time `json:"expires_at"`
}

// JWK represents a JSON Web Key
type JWK struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// AuthService handles authentication for MCP requests
type AuthService struct {
	apiKeyRepository   repositories.ApiKeyRepository
	metadataRepository repositories.MetadataStoreRepository
	logger             logger.Logger
	env                lib.Env
	jwksCache          *JWKSCache
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
		jwksCache:          nil, // Will be populated on first JWKS fetch
	}
}

// ExtractAuthFromMCPRequest extracts authentication information from MCP request arguments
func (s *AuthService) ExtractAuthFromMCPRequest(ctx context.Context, arguments map[string]any) (*AuthContext, error) {
	// Check for authorization token in arguments
	authTokenVal, exists := arguments["x-api-key"]
	if !exists {
		return nil, s.createMCPAuthError("missing_token", "Authentication required: x-api-key parameter missing. For OAuth 2.1 authentication, use 'Bearer {your_token}'")
	}

	authToken, ok := authTokenVal.(string)
	if !ok {
		return nil, s.createMCPAuthError("invalid_token_format", "Authentication required: x-api-key must be a string")
	}

	if authToken == "" {
		return nil, s.createMCPAuthError("empty_token", "Authentication required: x-api-key parameter is empty")
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
		return s.validateClerkToken(ctx, authToken)
	}

	// Try standard Bearer JWT tokens (Clerk session tokens)
	if strings.HasPrefix(authToken, "Bearer ") && !strings.HasPrefix(authToken, "Bearer pk_") {
		return s.validateClerkToken(ctx, authToken)
	}

	return nil, s.createMCPAuthError("invalid_token", "Invalid authentication token. Supported formats: 'Bearer pk_live_...' (API key), 'Bearer pk_test_...' (API key), or 'Bearer {clerk_session_token}' (OAuth 2.1)")
}

// validateClerkToken validates a Clerk session token and returns the authenticated user context
func (s *AuthService) validateClerkToken(ctx context.Context, authToken string) (*AuthContext, error) {
	// Remove Bearer prefix
	token := strings.TrimPrefix(authToken, "Bearer ")

	// Get Clerk secret key from environment
	clerkSecret := s.env.ClerkSecretKey
	if clerkSecret == "" {
		s.logger.Error("Clerk secret key not configured")
		return nil, s.createMCPAuthError("configuration_error", "Clerk authentication not configured on server")
	}

	// Validate JWT with Clerk's JWT verification
	claims, err := jwt.Verify(ctx, &jwt.VerifyParams{
		Token: token,
	})
	if err != nil {
		s.logger.Warn("Clerk token validation failed", "error", err.Error())
		return nil, s.createMCPAuthError("invalid_token", fmt.Sprintf("Clerk token validation failed: %s", err.Error()))
	}

	// Extract user information from claims
	userId := claims.Subject
	if userId == "" {
		s.logger.Warn("Missing user ID in Clerk token")
		return nil, s.createMCPAuthError("invalid_token", "Invalid token: missing user ID in claims")
	}

	// Extract email from claims
	// In Clerk SessionClaims, email is not directly available
	// You might need to make an additional API call to get user details
	email := ""

	// Extract organization ID from claims
	// Clerk stores the active organization ID as a string
	orgId := claims.ActiveOrganizationID

	// If still no orgId, we can't proceed as this is a multi-tenant system
	if orgId == "" {
		s.logger.Warn("No organization ID found in Clerk token", "userId", userId)
		return nil, s.createMCPAuthError("invalid_token", "Invalid token: missing organization information. Ensure you have selected an active organization in Clerk.")
	}

	// Extract roles - default to member
	// In Clerk, roles are typically stored in organization membership
	roles := []authn.UserRole{authn.Member}
	primaryRole := authn.Member

	// For now, we'll default to member role
	// In a production environment, you'd query the organization membership
	// to get the actual role of the user in the organization

	user := authn.User{
		OrgId:       orgId,
		Id:          userId,
		Email:       email,
		PrimaryRole: primaryRole,
		Roles:       roles,
	}

	s.logger.Debug("Successfully validated Clerk token", "userId", userId, "orgId", orgId, "email", email)

	return &AuthContext{
		User:  user,
		OrgId: orgId,
	}, nil
}

// getClerkJWKS fetches JWKS from Clerk's well-known endpoint
func (s *AuthService) getClerkJWKS(ctx context.Context) (*JWKSCache, error) {
	// Check if we have a valid cached JWKS
	if s.jwksCache != nil && time.Now().Before(s.jwksCache.ExpiresAt) {
		return s.jwksCache, nil
	}

	// Get Clerk domain from environment
	clerkDomain := s.env.ClerkDomain
	if clerkDomain == "" {
		return nil, fmt.Errorf("clerk domain not configured")
	}

	// Ensure the domain starts with https://
	if !strings.HasPrefix(clerkDomain, "http://") && !strings.HasPrefix(clerkDomain, "https://") {
		clerkDomain = "https://" + clerkDomain
	}

	// Construct the JWKS URL
	jwksURL := clerkDomain + "/.well-known/jwks.json"

	resp, err := http.Get(jwksURL)
	if err != nil {
		s.logger.Error("Failed to fetch JWKS from Clerk", "error", err.Error(), "url", jwksURL)
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		s.logger.Error("JWKS endpoint returned non-200 status", "status", resp.StatusCode, "url", jwksURL)
		return nil, fmt.Errorf("JWKS endpoint returned status %d", resp.StatusCode)
	}

	var jwksResponse struct {
		Keys []JWK `json:"keys"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&jwksResponse); err != nil {
		s.logger.Error("Failed to decode JWKS response", "error", err.Error())
		return nil, fmt.Errorf("failed to decode JWKS: %w", err)
	}

	// Cache JWKS for 1 hour
	cache := &JWKSCache{
		Keys:      jwksResponse.Keys,
		ExpiresAt: time.Now().Add(time.Hour),
	}

	s.jwksCache = cache
	s.logger.Debug("Successfully fetched and cached JWKS", "keyCount", len(jwksResponse.Keys))

	return cache, nil
}

// CreateUnauthorizedChallenge creates a proper WWW-Authenticate challenge response
// This follows OAuth 2.1 and RFC 6750 standards for resource server authentication challenges
func (s *AuthService) CreateUnauthorizedChallenge() map[string]string {
	// For MCP OAuth 2.1, we need to provide information about where to get authorization
	// This should point to our discovery endpoint which redirects to Clerk

	// Construct the challenge header
	// Format: Bearer realm="resource-server", authorization_uri="https://your-server/.well-known/oauth-authorization-server"

	challenge := map[string]string{
		"WWW-Authenticate": `Bearer realm="payloop-mcp", error="invalid_token", error_description="The access token is missing or invalid"`,
		"Link":             `</.well-known/oauth-authorization-server>; rel="authorization_server"`,
	}

	s.logger.Debug("Created OAuth 2.1 challenge response")

	return challenge
}

// createMCPAuthError creates a structured authentication error for MCP responses
func (s *AuthService) createMCPAuthError(errorCode, description string) error {
	return fmt.Errorf("authentication_error:%s:%s:discovery_endpoint:/.well-known/oauth-authorization-server", errorCode, description)
}
