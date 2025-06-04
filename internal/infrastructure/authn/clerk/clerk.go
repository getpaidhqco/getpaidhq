package clerk

import (
	"context"
	clerkapi "github.com/clerkinc/clerk-sdk-go/clerk"
	apiauthn "payloop/internal/api/authn"
	"payloop/internal/application/lib/authn"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
	"time"
)

type ClerkClient struct {
	client             clerkapi.Client
	logger             logger.Logger
	metadataRepository repositories.MetadataStoreRepository
}

func NewClerkClient(env lib.Env, logger logger.Logger, metadataRepository repositories.MetadataStoreRepository) authn.AuthProvider {
	client, err := clerkapi.NewClient(env.ClerkSecretKey)
	if err != nil {
		logger.Error("Failed to create clerk client", "error", err)
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
	c.logger.Info("Handling Clerk webhook")
	return nil
}

// Implement AuthProvider interface

// CreateOrg creates a new organization in Clerk and adds the owner to it
func (c ClerkClient) CreateOrg(ctx context.Context, org entities.Org, ownerUserID string) (authn.CreateOrgResponse, error) {
	c.logger.Info("Creating organization in Clerk", "orgID", org.Id, "ownerUserID", ownerUserID)

	// Create the organization in Clerk
	createParams := clerkapi.CreateOrganizationParams{
		Name:      org.Name,
		CreatedBy: ownerUserID,
	}

	// Create the organization
	createdOrg, err := c.client.Organizations().Create(createParams)
	if err != nil {
		c.logger.Error("Failed to create organization in Clerk", "error", err, "orgID", org.Id)
		return authn.CreateOrgResponse{}, err
	}

	c.logger.Info("Organization created in Clerk", "orgID", createdOrg.ID, "name", org.Name)
	_, err = c.metadataRepository.Create(ctx, entities.MetadataStore{
		OrgId:      org.Id,
		ParentId:   org.Id,
		ParentType: "org",
		Key:        "clerk_org_id",
		Value:      createdOrg.ID,
		Namespace:  "clerk",
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	})
	if err != nil {
		c.logger.Error("Failed to store Clerk organization ID in metadata", "error", err, "orgID", org.Id, "clerkOrgID", createdOrg.ID)
		return authn.CreateOrgResponse{}, err
	}

	return authn.CreateOrgResponse{
		ExternalId: createdOrg.ID,
		Data:       createdOrg,
	}, nil
}

// AddUserToOrg adds a user to an organization with the specified role
func (c ClerkClient) AddUserToOrg(orgID string, userID string, role apiauthn.UserRole) error {
	c.logger.Info("Adding user to organization in Clerk", "orgID", orgID, "userID", userID, "role", role)

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
		c.logger.Error("Failed to add user to organization in Clerk", "error", err, "orgID", orgID, "userID", userID, "role", clerkRole)
		return err
	}

	c.logger.Info("User added to organization in Clerk", "orgID", orgID, "userID", userID, "role", clerkRole)

	return nil
}

// RemoveUserFromOrg removes a user from an organization
func (c ClerkClient) RemoveUserFromOrg(orgID, userID string) error {
	c.logger.Info("Removing user from organization in Clerk", "orgID", orgID, "userID", userID)

	// Delete the membership
	_, err := c.client.Organizations().DeleteMembership(orgID, userID)
	if err != nil {
		c.logger.Error("Failed to remove user from organization in Clerk", "error", err, "orgID", orgID, "userID", userID)
		return err
	}

	c.logger.Info("User removed from organization in Clerk", "orgID", orgID, "userID", userID)

	return nil
}

// DeleteOrg deletes an organization from Clerk
func (c ClerkClient) DeleteOrg(orgID string) error {
	c.logger.Info("Deleting organization in Clerk", "orgID", orgID)

	// Delete the organization
	_, err := c.client.Organizations().Delete(orgID)
	if err != nil {
		c.logger.Error("Failed to delete organization in Clerk", "error", err, "orgID", orgID)
		return err
	}

	c.logger.Info("Organization deleted in Clerk", "orgID", orgID)

	return nil
}

// mapUserRoleToClerkRole maps the application role to Clerk role
func mapUserRoleToClerkRole(role apiauthn.UserRole) string {
	switch role {
	case apiauthn.Admin:
		return "org:admin"
	case apiauthn.Owner:
		return "org:admin" // Map owner to admin in Clerk
	case apiauthn.Support:
		return "org:member" // Map support to member in Clerk
	default:
		return "org:member"
	}
}
