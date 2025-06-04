package repositories

import (
	"context"
	"payloop/internal/domain/entities"
)

// MetadataStoreRepository defines the interface for metadata store operations
type MetadataStoreRepository interface {
	// FindByKey retrieves a metadata entry by org, parent, and key
	FindByKey(ctx context.Context, orgId string, parentId string, key string) (entities.MetadataStore, error)

	// FindByParent retrieves all metadata entries for a specific parent
	FindByParent(ctx context.Context, orgId string, parentId string) ([]entities.MetadataStore, error)

	// FindByParentType retrieves all metadata entries for a specific parent type with a specific key
	FindByParentType(ctx context.Context, orgId string, parentType string, key string) ([]entities.MetadataStore, error)

	// FindByValue retrieves all metadata entries with a specific key and value
	FindByValue(ctx context.Context, orgId string, key string, value string) ([]entities.MetadataStore, error)

	// FindByValueWithoutOrg retrieves all metadata entries with a specific key and value across all organizations
	// If parentType is provided, it will filter by parent type as well
	FindByValueWithoutOrg(ctx context.Context, key string, value string, parentType string) ([]entities.MetadataStore, error)

	// Create creates a new metadata entry
	Create(ctx context.Context, metadata entities.MetadataStore) (entities.MetadataStore, error)

	// Update updates an existing metadata entry
	Update(ctx context.Context, metadata entities.MetadataStore) (entities.MetadataStore, error)

	// Delete deletes a metadata entry
	Delete(ctx context.Context, orgId string, parentId string, key string) error
}
