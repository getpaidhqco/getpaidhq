package clerk

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/jwt"
	"github.com/clerk/clerk-sdk-go/v2/user"
	"github.com/gin-gonic/gin"

	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

type ClerkMiddleware struct {
	handler            lib.RequestHandler
	logger             port.Logger
	env                lib.Env
	metadataRepository port.MetadataStoreRepository
}

func NewClerkMiddleware(
	handler lib.RequestHandler,
	logger port.Logger,
	env lib.Env,
	metadataRepository port.MetadataStoreRepository,
) port.Authenticator {
	clerk.SetKey(env.ClerkSecretKey)
	return ClerkMiddleware{
		handler:            handler,
		metadataRepository: metadataRepository,
		logger:             logger,
		env:                env,
	}
}

// Setup is retained for the legacy standalone-middleware pattern; the
// AuthnWrapperMiddleware now invokes Authenticate directly, so this is unused
// in the wired-up path but kept available for tests or alt wiring.
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
		authedUser, err := m.Authenticate(c.Request.Context(), token)
		if err != nil {
			if errors.Is(err, port.ErrOnboardingRequired) {
				c.Redirect(http.StatusFound, "/onboarding")
				return
			}
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "invalid token"})
			return
		}
		c.Set("user", authedUser)
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
	claims, err := jwt.Verify(ctx, &jwt.VerifyParams{
		Token: strings.TrimPrefix(token, "Bearer "),
	})
	if err != nil {
		return port.AuthUser{}, err
	}

	usr, err := user.Get(ctx, claims.Subject)
	if err != nil {
		m.logger.Error("Error fetching user from Clerk API", "error", err)
		return port.AuthUser{}, err
	}

	authedUser := port.AuthUser{
		OrgId:       claims.ActiveOrganizationID,
		Id:          claims.Subject,
		Email:       primaryEmail(usr),
		PrimaryRole: MapClerkRoleToUserRole(claims.ActiveOrganizationRole),
		Roles:       []port.UserRole{MapClerkRoleToUserRole(claims.ActiveOrganizationRole)},
	}

	if authedUser.OrgId == "" {
		// v2 JWTs put the active org in `o.id`; if it's missing the user
		// has no active org and needs to onboard / pick one.
		return authedUser, port.ErrOnboardingRequired
	}

	m.logger.Infof("Clerk auth resolved: user=%s org=%s role=%s", authedUser.Id, authedUser.OrgId, authedUser.PrimaryRole)
	return authedUser, nil
}

func primaryEmail(usr *clerk.User) string {
	if usr == nil {
		return ""
	}
	if usr.PrimaryEmailAddressID != nil {
		for _, e := range usr.EmailAddresses {
			if e != nil && e.ID == *usr.PrimaryEmailAddressID {
				return e.EmailAddress
			}
		}
	}
	if len(usr.EmailAddresses) > 0 && usr.EmailAddresses[0] != nil {
		return usr.EmailAddresses[0].EmailAddress
	}
	return ""
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
