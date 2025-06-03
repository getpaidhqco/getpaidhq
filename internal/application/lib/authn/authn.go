package authn

import (
	"context"
	"payloop/internal/api/authn"
	"payloop/internal/domain/entities"
)

var PublicPaths = []string{"/api/health", "/api/notify", "/api/notify/cdc"}

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

type AuthProvider interface {
	CreateOrg(org entities.Org, ownerUserID string) error
	AddUserToOrg(orgID string, userID string, role authn.UserRole) error
	RemoveUserFromOrg(orgID, userID string) error
	DeleteOrg(orgID string) error
	HandleWebhook(data string) error
}
