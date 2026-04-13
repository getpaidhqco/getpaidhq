package port

import (
	"context"
	"errors"
	"payloop/internal/core/domain"
)

// ErrOnboardingRequired is returned when a user needs to complete onboarding.
var ErrOnboardingRequired = errors.New("onboarding required")

// UserRole represents the role of a user.
type UserRole string

const (
	RoleAdmin   UserRole = "admin"
	RoleSupport UserRole = "support"
	RoleOwner   UserRole = "owner"
	RoleMember  UserRole = "member"
)

// AuthUser represents an authenticated user in the system.
type AuthUser struct {
	OrgId       string     `json:"org_id"`
	Id          string     `json:"id"`
	Email       string     `json:"email"`
	PrimaryRole UserRole   `json:"primary_role"`
	Roles       []UserRole `json:"roles"`
}

func NewAuthUser(orgId, id, email string, roles []UserRole) AuthUser {
	return AuthUser{
		OrgId:       orgId,
		Id:          id,
		Email:       email,
		PrimaryRole: GetPrimaryRole(roles),
		Roles:       roles,
	}
}

func GetPrimaryRole(roles []UserRole) UserRole {
	rolesRank := map[UserRole]int{
		RoleAdmin:   4,
		RoleSupport: 3,
		RoleOwner:   2,
		RoleMember:  1,
	}

	primaryRole := RoleMember
	for _, role := range roles {
		if rolesRank[role] > rolesRank[primaryRole] {
			primaryRole = role
		}
	}
	return primaryRole
}

var PublicPaths = []string{"/api/health", "/api/notify", "/api/notify/cdc"}

func IsPublicPath(path string) bool {
	for _, publicPath := range PublicPaths {
		if path == publicPath {
			return true
		}
	}
	return false
}

// Authenticator validates tokens and returns an authenticated user.
type Authenticator interface {
	Setup()
	Authenticate(ctx context.Context, token string) (AuthUser, error)
}

// CreateOrgResponse contains the response data from creating an organization.
type CreateOrgResponse struct {
	ExternalId string
	Data       interface{}
}

// AuthProvider manages external auth provider operations (Clerk, Cognito, etc.).
type AuthProvider interface {
	CreateOrg(ctx context.Context, org domain.Org, ownerUserID string) (CreateOrgResponse, error)
	AddUserToOrg(orgID string, userID string, role UserRole) error
	RemoveUserFromOrg(orgID, userID string) error
	DeleteOrg(orgID string) error
	HandleWebhook(data string) error
}
