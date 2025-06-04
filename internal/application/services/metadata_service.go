package services

import (
	"context"
	"payloop/internal/application/dto"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"time"
)

// MetadataService provides operations for managing metadata
type MetadataService struct {
	metadataRepository repositories.MetadataStoreRepository
	logger             logger.Logger
}

// NewMetadataService creates a new metadata service
func NewMetadataService(
	metadataRepository repositories.MetadataStoreRepository,
	logger logger.Logger,
) MetadataService {
	return MetadataService{
		metadataRepository: metadataRepository,
		logger:             logger,
	}
}

// Create creates a new metadata entry
func (s MetadataService) Create(ctx context.Context, input dto.CreateMetadataInput) (entities.MetadataStore, error) {
	s.logger.Debug("Creating metadata", "input", input)

	metadata, err := s.metadataRepository.Create(ctx, entities.MetadataStore{
		OrgId:      input.OrgId,
		ParentId:   input.ParentId,
		ParentType: input.ParentType,
		Key:        input.Key,
		Value:      input.Value,
		Namespace:  input.Namespace,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})
	if err != nil {
		s.logger.Error("Failed to create metadata", "err", err)
		return entities.MetadataStore{}, err
	}

	return metadata, nil
}

// Update updates an existing metadata entry
func (s MetadataService) Update(ctx context.Context, input dto.UpdateMetadataInput) (entities.MetadataStore, error) {
	s.logger.Debug("Updating metadata", "input", input)

	// First, get the existing metadata to preserve the parent type
	existing, err := s.metadataRepository.FindByKey(ctx, input.OrgId, input.ParentId, input.Key)
	if err != nil {
		s.logger.Error("Failed to find metadata for update", "err", err)
		return entities.MetadataStore{}, err
	}

	// Update the metadata
	metadata, err := s.metadataRepository.Update(ctx, entities.MetadataStore{
		OrgId:      input.OrgId,
		ParentId:   input.ParentId,
		ParentType: existing.ParentType, // Preserve the parent type
		Key:        input.Key,
		Value:      input.Value,
		Namespace:  input.Namespace,
		UpdatedAt:  time.Now(),
	})
	if err != nil {
		s.logger.Error("Failed to update metadata", "err", err)
		return entities.MetadataStore{}, err
	}

	return metadata, nil
}

// GetByKey retrieves a metadata entry by key
func (s MetadataService) GetByKey(ctx context.Context, orgId string, parentId string, key string) (entities.MetadataStore, error) {
	s.logger.Debug("Getting metadata by key", "orgId", orgId, "parentId", parentId, "key", key)

	metadata, err := s.metadataRepository.FindByKey(ctx, orgId, parentId, key)
	if err != nil {
		s.logger.Error("Failed to get metadata by key", "err", err)
		return entities.MetadataStore{}, err
	}

	return metadata, nil
}

// GetByParent retrieves all metadata entries for a parent
func (s MetadataService) GetByParent(ctx context.Context, orgId string, parentId string) ([]entities.MetadataStore, error) {
	s.logger.Debug("Getting metadata by parent", "orgId", orgId, "parentId", parentId)

	metadataList, err := s.metadataRepository.FindByParent(ctx, orgId, parentId)
	if err != nil {
		s.logger.Error("Failed to get metadata by parent", "err", err)
		return nil, err
	}

	return metadataList, nil
}

// GetByParentType retrieves all metadata entries for a parent type with a specific key
func (s MetadataService) GetByParentType(ctx context.Context, orgId string, parentType string, key string) ([]entities.MetadataStore, error) {
	s.logger.Debug("Getting metadata by parent type", "orgId", orgId, "parentType", parentType, "key", key)

	metadataList, err := s.metadataRepository.FindByParentType(ctx, orgId, parentType, key)
	if err != nil {
		s.logger.Error("Failed to get metadata by parent type", "err", err)
		return nil, err
	}

	return metadataList, nil
}

// GetByValue retrieves all metadata entries with a specific key and value
func (s MetadataService) GetByValue(ctx context.Context, orgId string, key string, value string) ([]entities.MetadataStore, error) {
	s.logger.Debug("Getting metadata by value", "orgId", orgId, "key", key, "value", value)

	metadataList, err := s.metadataRepository.FindByValue(ctx, orgId, key, value)
	if err != nil {
		s.logger.Error("Failed to get metadata by value", "err", err)
		return nil, err
	}

	return metadataList, nil
}

// GetByValueWithoutOrg retrieves all metadata entries with a specific key and value across all organizations
// If parentType is provided, it will filter by parent type as well
func (s MetadataService) GetByValueWithoutOrg(ctx context.Context, key string, value string, parentType string) ([]entities.MetadataStore, error) {
	s.logger.Debug("Getting metadata by value without org", "key", key, "value", value, "parentType", parentType)

	metadataList, err := s.metadataRepository.FindByValueWithoutOrg(ctx, key, value, parentType)
	if err != nil {
		s.logger.Error("Failed to get metadata by value without org", "err", err)
		return nil, err
	}

	return metadataList, nil
}

// Delete deletes a metadata entry
func (s MetadataService) Delete(ctx context.Context, orgId string, parentId string, key string) error {
	s.logger.Debug("Deleting metadata", "orgId", orgId, "parentId", parentId, "key", key)

	err := s.metadataRepository.Delete(ctx, orgId, parentId, key)
	if err != nil {
		s.logger.Error("Failed to delete metadata", "err", err)
		return err
	}

	return nil
}
