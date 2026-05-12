package clerk

import (
	"context"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/organization"
	"github.com/clerk/clerk-sdk-go/v2/organizationmembership"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

// ClerkClient implements port.AuthProvider against the Clerk Backend API
// using the global-key style of clerk-sdk-go/v2.
type ClerkClient struct {
	logger             port.Logger
	metadataRepository port.MetadataStoreRepository
}

func NewClerkClient(env lib.Env, logger port.Logger, metadataRepository port.MetadataStoreRepository) port.AuthProvider {
	clerk.SetKey(env.ClerkSecretKey)
	return ClerkClient{
		logger:             logger,
		metadataRepository: metadataRepository,
	}
}

func (c ClerkClient) HandleWebhook(data string) error {
	c.logger.Info("Handling Clerk webhook")
	return nil
}

func (c ClerkClient) CreateOrg(ctx context.Context, org domain.Org, ownerUserID string) (port.CreateOrgResponse, error) {
	c.logger.Info("Creating organization in Clerk", "orgID", org.Id, "ownerUserID", ownerUserID)

	createdOrg, err := organization.Create(ctx, &organization.CreateParams{
		Name:      clerk.String(org.Name),
		CreatedBy: clerk.String(ownerUserID),
	})
	if err != nil {
		c.logger.Error("Failed to create organization in Clerk", "error", err, "orgID", org.Id)
		return port.CreateOrgResponse{}, err
	}

	return port.CreateOrgResponse{
		ExternalId: createdOrg.ID,
		Data:       createdOrg,
	}, nil
}

func (c ClerkClient) AddUserToOrg(orgID string, userID string, role port.UserRole) error {
	c.logger.Info("Adding user to organization in Clerk", "orgID", orgID, "userID", userID, "role", role)

	clerkRole := mapUserRoleToClerkRole(role)
	_, err := organizationmembership.Create(context.Background(), &organizationmembership.CreateParams{
		OrganizationID: orgID,
		UserID:         clerk.String(userID),
		Role:           clerk.String(clerkRole),
	})
	if err != nil {
		c.logger.Error("Failed to add user to organization in Clerk", "error", err, "orgID", orgID, "userID", userID, "role", clerkRole)
		return err
	}

	c.logger.Info("User added to organization in Clerk", "orgID", orgID, "userID", userID, "role", clerkRole)
	return nil
}

func (c ClerkClient) RemoveUserFromOrg(orgID, userID string) error {
	c.logger.Info("Removing user from organization in Clerk", "orgID", orgID, "userID", userID)

	_, err := organizationmembership.Delete(context.Background(), &organizationmembership.DeleteParams{
		OrganizationID: orgID,
		UserID:         userID,
	})
	if err != nil {
		c.logger.Error("Failed to remove user from organization in Clerk", "error", err, "orgID", orgID, "userID", userID)
		return err
	}

	c.logger.Info("User removed from organization in Clerk", "orgID", orgID, "userID", userID)
	return nil
}

func (c ClerkClient) DeleteOrg(orgID string) error {
	c.logger.Info("Deleting organization in Clerk", "orgID", orgID)

	_, err := organization.Delete(context.Background(), orgID)
	if err != nil {
		c.logger.Error("Failed to delete organization in Clerk", "error", err, "orgID", orgID)
		return err
	}

	c.logger.Info("Organization deleted in Clerk", "orgID", orgID)
	return nil
}

func mapUserRoleToClerkRole(role port.UserRole) string {
	switch role {
	case port.RoleAdmin, port.RoleOwner:
		return "org:admin"
	case port.RoleSupport:
		return "org:member"
	default:
		return "org:member"
	}
}
