package clerk

import (
	"context"
	"strings"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/jwt"
	"github.com/clerk/clerk-sdk-go/v2/user"

	"getpaidhq/internal/core/port"
)

type ClerkMiddleware struct {
	logger             port.Logger
	metadataRepository port.MetadataStoreRepository
}

// NewClerkMiddleware authenticates Clerk bearer tokens. secretKey is the
// Clerk backend API key (CLERK_SECRET); the clerk-sdk-go v2 global-key style
// means it is set process-wide here.
func NewClerkMiddleware(
	logger port.Logger,
	secretKey string,
	metadataRepository port.MetadataStoreRepository,
) port.Authenticator {
	clerk.SetKey(secretKey)
	return ClerkMiddleware{
		metadataRepository: metadataRepository,
		logger:             logger,
	}
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
