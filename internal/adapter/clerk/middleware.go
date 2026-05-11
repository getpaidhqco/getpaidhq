package clerk

import (
	"context"
	"errors"
	clerkapi "github.com/clerkinc/clerk-sdk-go/clerk"
	"github.com/gin-gonic/gin"
	"net/http"
	"payloop/internal/core/port"
	"payloop/internal/lib"
	"slices"
	"strings"
)

type ClerkMiddleware struct {
	handler            lib.RequestHandler
	logger             port.Logger
	env                lib.Env
	client             clerkapi.Client
	metadataRepository port.MetadataStoreRepository
}

func NewClerkMiddleware(
	handler lib.RequestHandler,
	logger port.Logger,
	env lib.Env,
	metadataRepository port.MetadataStoreRepository,
) port.Authenticator {

	client, err := clerkapi.NewClient(env.ClerkSecretKey)
	if err != nil {
		logger.Error("Error initializing clerk client", "error", err)
		panic(err)
	}

	return ClerkMiddleware{
		handler:            handler,
		metadataRepository: metadataRepository,
		logger:             logger,
		env:                env,
		client:             client,
	}
}

// Setup sets up clerk middleware
func (m ClerkMiddleware) Setup() {
	m.logger.Info("Setting up clerk middleware")
	m.handler.Gin.Use(func(c *gin.Context) {

		if isPublicPath(c.Request.URL.Path) {
			c.Next()
			return
		}

		token, err := tokenFromAuthHeader(c.Request)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "invalid Authorization header"})
			return
		}
		user, err := m.Authenticate(c.Request.Context(), token)
		if err != nil {
			if errors.Is(err, port.ErrOnboardingRequired) {
				// TODO make this URL configurable
				c.Redirect(302, "/onboarding")
				return
			}
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "invalid token"})
			return
		}
		c.Set("user", user)
		c.Next()
	})
}

func isPublicPath(path string) bool {
	return slices.Contains(port.PublicPaths, path)
}

func MapClerkRoleToUserRole(role string) port.UserRole {
	switch role {
	case "org:admin":
		return port.RoleAdmin
	default:
		return port.RoleMember
	}
}

func (m ClerkMiddleware) Authenticate(ctx context.Context, token string) (port.AuthUser, error) {
	session, err := m.client.VerifyToken(strings.TrimPrefix(token, "Bearer "))
	if err != nil {
		return port.AuthUser{}, err
	}

	// Log the session information
	m.logger.Infof("Clerk Auth: [%s][%s][%s]", session.ActiveOrganizationID, session.Claims.Subject, token)
	user, err := m.client.Users().Read(session.Claims.Subject)
	if err != nil {
		m.logger.Error("Error fetching user from Clerk API", "error", err)
		return port.AuthUser{}, err
	}
	m.logger.Infof("Clerk Auth: [%s][%s][%+v]", session.ActiveOrganizationID, session.Claims.Subject, user)

	// If the organization ID is not in the token, try to fetch the user's organization memberships
	clerkOrgId := session.ActiveOrganizationID
	orgRole := session.ActiveOrganizationRole

	authedUser := port.AuthUser{
		OrgId:       clerkOrgId,
		Id:          session.Claims.Subject,
		Email:       user.EmailAddresses[0].EmailAddress,
		PrimaryRole: MapClerkRoleToUserRole(orgRole),
		Roles:       []port.UserRole{MapClerkRoleToUserRole(orgRole)},
	}

	if authedUser.OrgId == "" {
		m.logger.Info("Organization ID not found in token, fetching from Clerk API")
		// Fetch the user's organization memberships
		memberships, err := m.client.Users().ListMemberships(clerkapi.ListMembershipsParams{
			UserID: session.Claims.Subject,
		})
		if err != nil {
			m.logger.Error("Error fetching user organization memberships", "error", err)
			return port.AuthUser{}, err
		}
		if len(memberships.Data) > 0 {
			// Use the first organization as the active one
			authedUser.OrgId = memberships.Data[0].Organization.ID
			authedUser.PrimaryRole = MapClerkRoleToUserRole(memberships.Data[0].Role)
			authedUser.Roles = []port.UserRole{MapClerkRoleToUserRole(memberships.Data[0].Role)}
		} else {
			m.logger.Error("No organization memberships found for user", "user_id", session.Claims.Subject)
			return authedUser, port.ErrOnboardingRequired
		}
	}

	m.logger.Infof("Found org ID from metadata: %s with role: %s", clerkOrgId, orgRole)

	return authedUser, nil
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
