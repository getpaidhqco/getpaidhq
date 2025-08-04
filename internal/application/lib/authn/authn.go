package authn

import (
	"context"
	"payloop/internal/api/authn"
	"payloop/internal/domain/entities"
)

var PublicPaths = []string{"/api/health", "/api/notify", "/.well-known/oauth-authorization-server", "/.well-known/openid_configuration"}

func IsPublicPath(path string) bool {
	for _, publicPath := range PublicPaths {
		if path == publicPath {
			return true
		}
	}
	return false
}

type Authenticator interface {
	Setup()
	Authenticate(ctx context.Context, token string) (authn.User, error)
}

// CreateOrgResponse contains the response data from creating an organization
type CreateOrgResponse struct {
	ExternalId string
	Data       interface{}
}

type AuthProvider interface {
	CreateOrg(ctx context.Context, org entities.Org, ownerUserID string) (CreateOrgResponse, error)
	AddUserToOrg(orgID string, userID string, role authn.UserRole) error
	RemoveUserFromOrg(orgID, userID string) error
	DeleteOrg(orgID string) error
	HandleWebhook(data string) error
}
