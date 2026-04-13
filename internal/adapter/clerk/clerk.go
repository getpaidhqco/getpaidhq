package clerk

import (
	"context"
	clerkapi "github.com/clerkinc/clerk-sdk-go/clerk"
	"payloop/internal/core/domain"
	"payloop/internal/core/port"
	"payloop/internal/lib"
)

type ClerkClient struct {
	client             clerkapi.Client
	logger             port.Logger
	metadataRepository port.MetadataStoreRepository
}

func NewClerkClient(env lib.Env, logger port.Logger, metadataRepository port.MetadataStoreRepository) port.AuthProvider {
	client, err := clerkapi.NewClient(env.ClerkSecretKey)
	if err != nil {
		logger.Error("failed to create clerk client", "error", err)
		panic(err)
	}

	return ClerkClient{
		client:             client,
		logger:             logger,
		metadataRepository: metadataRepository,
	}
}

func (c ClerkClient) HandleWebhook(data string) error {
	// Implement the logic to handle webhooks from Clerk
	// This could involve processing events like user creation, deletion, etc.
	// For now, we will just log that the webhook handler is called.
	c.logger.Info("handling clerk webhook")
	return nil
}

// Implement AuthProvider interface

// CreateOrg creates a new organization in Clerk and adds the owner to it
func (c ClerkClient) CreateOrg(ctx context.Context, org domain.Org, ownerUserID string) (port.CreateOrgResponse, error) {
	c.logger.Info("creating organization in clerk", "orgId", org.Id, "ownerUserID", ownerUserID)

	// Create the organization in Clerk
	createParams := clerkapi.CreateOrganizationParams{
		Name:      org.Name,
		CreatedBy: ownerUserID,
	}

	// Create the organization
	createdOrg, err := c.client.Organizations().Create(createParams)
	if err != nil {
		c.logger.Error("failed to create organization in clerk", "error", err, "orgId", org.Id)
		return port.CreateOrgResponse{}, err
	}

	return port.CreateOrgResponse{
		ExternalId: createdOrg.ID,
		Data:       createdOrg,
	}, nil
}

// AddUserToOrg adds a user to an organization with the specified role
func (c ClerkClient) AddUserToOrg(orgID string, userID string, role port.UserRole) error {
	c.logger.Info("adding user to organization in clerk", "orgId", orgID, "userID", userID, "role", role)

	// Map the application role to Clerk role
	clerkRole := mapUserRoleToClerkRole(role)

	// Create membership params
	createParams := clerkapi.CreateOrganizationMembershipParams{
		UserID: userID,
		Role:   clerkRole,
	}

	// Create the membership
	_, err := c.client.Organizations().CreateMembership(orgID, createParams)
	if err != nil {
		c.logger.Error("failed to add user to organization in clerk", "error", err, "orgId", orgID, "userID", userID, "role", clerkRole)
		return err
	}

	c.logger.Info("user added to organization in clerk", "orgId", orgID, "userID", userID, "role", clerkRole)

	return nil
}

// RemoveUserFromOrg removes a user from an organization
func (c ClerkClient) RemoveUserFromOrg(orgID, userID string) error {
	c.logger.Info("removing user from organization in clerk", "orgId", orgID, "userID", userID)

	// Delete the membership
	_, err := c.client.Organizations().DeleteMembership(orgID, userID)
	if err != nil {
		c.logger.Error("failed to remove user from organization in clerk", "error", err, "orgId", orgID, "userID", userID)
		return err
	}

	c.logger.Info("user removed from organization in clerk", "orgId", orgID, "userID", userID)

	return nil
}

// DeleteOrg deletes an organization from Clerk
func (c ClerkClient) DeleteOrg(orgID string) error {
	c.logger.Info("deleting organization in clerk", "orgId", orgID)

	// Delete the organization
	_, err := c.client.Organizations().Delete(orgID)
	if err != nil {
		c.logger.Error("failed to delete organization in clerk", "error", err, "orgId", orgID)
		return err
	}

	c.logger.Info("organization deleted in clerk", "orgId", orgID)

	return nil
}

// mapUserRoleToClerkRole maps the application role to Clerk role
func mapUserRoleToClerkRole(role port.UserRole) string {
	switch role {
	case port.RoleAdmin:
		return "org:admin"
	case port.RoleOwner:
		return "org:admin" // Map owner to admin in Clerk
	case port.RoleSupport:
		return "org:member" // Map support to member in Clerk
	default:
		return "org:member"
	}
}
