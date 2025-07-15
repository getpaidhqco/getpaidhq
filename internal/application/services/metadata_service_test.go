package services

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"payloop/internal/application/dto"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/testing/mocks"
	"testing"
)

// Define error for testing
var ErrNotFound = errors.New("not found")

// MockMetadataRepository is a mock implementation of the MetadataStoreRepository interface
type MockMetadataRepository struct {
	metadata map[string]entities.MetadataStore
}

func NewMockMetadataRepository() repositories.MetadataStoreRepository {
	return &MockMetadataRepository{
		metadata: make(map[string]entities.MetadataStore),
	}
}

func (m *MockMetadataRepository) FindByKey(ctx context.Context, orgId string, parentId string, key string) (entities.MetadataStore, error) {
	metadataKey := orgId + ":" + parentId + ":" + key
	metadata, ok := m.metadata[metadataKey]
	if !ok {
		return entities.MetadataStore{}, ErrNotFound
	}
	return metadata, nil
}

func (m *MockMetadataRepository) FindByParent(ctx context.Context, orgId string, parentId string) ([]entities.MetadataStore, error) {
	var result []entities.MetadataStore
	for _, metadata := range m.metadata {
		if metadata.OrgId == orgId && metadata.ParentId == parentId {
			result = append(result, metadata)
		}
	}
	return result, nil
}

func (m *MockMetadataRepository) FindByParentType(ctx context.Context, orgId string, parentType string, key string) ([]entities.MetadataStore, error) {
	var result []entities.MetadataStore
	for _, metadata := range m.metadata {
		if metadata.OrgId == orgId && metadata.ParentType == parentType && metadata.Key == key {
			result = append(result, metadata)
		}
	}
	return result, nil
}

func (m *MockMetadataRepository) FindByValue(ctx context.Context, orgId string, key string, value string) ([]entities.MetadataStore, error) {
	var result []entities.MetadataStore
	for _, metadata := range m.metadata {
		if metadata.OrgId == orgId && metadata.Key == key && metadata.Value == value {
			result = append(result, metadata)
		}
	}
	return result, nil
}

func (m *MockMetadataRepository) FindByValueWithoutOrg(ctx context.Context, key string, value string, parentType string) ([]entities.MetadataStore, error) {
	var result []entities.MetadataStore
	for _, metadata := range m.metadata {
		if metadata.Key == key && metadata.Value == value {
			if parentType == "" || metadata.ParentType == parentType {
				result = append(result, metadata)
			}
		}
	}
	return result, nil
}

func (m *MockMetadataRepository) Create(ctx context.Context, metadata entities.MetadataStore) (entities.MetadataStore, error) {
	metadataKey := metadata.OrgId + ":" + metadata.ParentId + ":" + metadata.Key
	m.metadata[metadataKey] = metadata
	return metadata, nil
}

func (m *MockMetadataRepository) Update(ctx context.Context, metadata entities.MetadataStore) (entities.MetadataStore, error) {
	metadataKey := metadata.OrgId + ":" + metadata.ParentId + ":" + metadata.Key
	_, ok := m.metadata[metadataKey]
	if !ok {
		return entities.MetadataStore{}, ErrNotFound
	}
	m.metadata[metadataKey] = metadata
	return metadata, nil
}

func (m *MockMetadataRepository) Delete(ctx context.Context, orgId string, parentId string, key string) error {
	metadataKey := orgId + ":" + parentId + ":" + key
	_, ok := m.metadata[metadataKey]
	if !ok {
		return ErrNotFound
	}
	delete(m.metadata, metadataKey)
	return nil
}

func TestMetadataService(t *testing.T) {
	mockRepo := NewMockMetadataRepository()
	logger := mocks.MockLogger{}
	service := NewMetadataService(mockRepo, &logger)

	ctx := context.Background()
	orgId := "org_123"
	parentId := "customer_456"
	parentType := "customer"
	key := "stripe_customer_id"
	value := "cus_123456"
	namespace := "external_ids"

	// Test Create
	createInput := dto.CreateMetadataInput{
		OrgId:      orgId,
		ParentId:   parentId,
		ParentType: parentType,
		Key:        key,
		Value:      value,
		Namespace:  namespace,
	}
	createOutput, err := service.Create(ctx, createInput)
	assert.NoError(t, err)
	assert.Equal(t, orgId, createOutput.OrgId)
	assert.Equal(t, parentId, createOutput.ParentId)
	assert.Equal(t, parentType, createOutput.ParentType)
	assert.Equal(t, key, createOutput.Key)
	assert.Equal(t, value, createOutput.Value)
	assert.Equal(t, namespace, createOutput.Namespace)

	// Test GetByKey
	getOutput, err := service.GetByKey(ctx, orgId, parentId, key)
	assert.NoError(t, err)
	assert.Equal(t, orgId, getOutput.OrgId)
	assert.Equal(t, parentId, getOutput.ParentId)
	assert.Equal(t, parentType, getOutput.ParentType)
	assert.Equal(t, key, getOutput.Key)
	assert.Equal(t, value, getOutput.Value)
	assert.Equal(t, namespace, getOutput.Namespace)

	// Test Update
	newValue := "cus_789012"
	updateInput := dto.UpdateMetadataInput{
		OrgId:     orgId,
		ParentId:  parentId,
		Key:       key,
		Value:     newValue,
		Namespace: namespace,
	}
	updateOutput, err := service.Update(ctx, updateInput)
	assert.NoError(t, err)
	assert.Equal(t, orgId, updateOutput.OrgId)
	assert.Equal(t, parentId, updateOutput.ParentId)
	assert.Equal(t, parentType, updateOutput.ParentType)
	assert.Equal(t, key, updateOutput.Key)
	assert.Equal(t, newValue, updateOutput.Value)
	assert.Equal(t, namespace, updateOutput.Namespace)

	// Test GetByParent
	parentOutput, err := service.GetByParent(ctx, orgId, parentId)
	assert.NoError(t, err)
	assert.Len(t, parentOutput, 1)
	assert.Equal(t, key, parentOutput[0].Key)
	assert.Equal(t, newValue, parentOutput[0].Value)

	// Test GetByParentType
	parentTypeOutput, err := service.GetByParentType(ctx, orgId, parentType, key)
	assert.NoError(t, err)
	assert.Len(t, parentTypeOutput, 1)
	assert.Equal(t, parentId, parentTypeOutput[0].ParentId)
	assert.Equal(t, newValue, parentTypeOutput[0].Value)

	// Test GetByValue
	valueOutput, err := service.GetByValue(ctx, orgId, key, newValue)
	assert.NoError(t, err)
	assert.Len(t, valueOutput, 1)
	assert.Equal(t, parentId, valueOutput[0].ParentId)
	assert.Equal(t, parentType, valueOutput[0].ParentType)

	// Test GetByValueWithoutOrg
	// First, create a metadata entry for a different org to test cross-org search
	otherOrgId := "org_789"
	otherParentId := "customer_012"
	createInput2 := dto.CreateMetadataInput{
		OrgId:      otherOrgId,
		ParentId:   otherParentId,
		ParentType: parentType,
		Key:        key,
		Value:      newValue,
		Namespace:  namespace,
	}
	_, err = service.Create(ctx, createInput2)
	assert.NoError(t, err)

	// Now test the GetByValueWithoutOrg method without parent type filter
	valueWithoutOrgOutput, err := service.GetByValueWithoutOrg(ctx, key, newValue, "")
	assert.NoError(t, err)
	assert.Len(t, valueWithoutOrgOutput, 2) // Should find entries from both orgs

	// Verify that we got entries from both orgs
	foundOrg1 := false
	foundOrg2 := false
	for _, metadata := range valueWithoutOrgOutput {
		if metadata.OrgId == orgId {
			foundOrg1 = true
		}
		if metadata.OrgId == otherOrgId {
			foundOrg2 = true
		}
	}
	assert.True(t, foundOrg1, "Should find metadata from first org")
	assert.True(t, foundOrg2, "Should find metadata from second org")

	// Test GetByValueWithoutOrg with parent type filter
	valueWithParentTypeOutput, err := service.GetByValueWithoutOrg(ctx, key, newValue, parentType)
	assert.NoError(t, err)
	assert.Len(t, valueWithParentTypeOutput, 2) // Should still find both entries since they have the same parent type

	// Create a metadata entry with a different parent type
	differentParentType := "payment_method"
	createInput3 := dto.CreateMetadataInput{
		OrgId:      otherOrgId,
		ParentId:   "pm_345",
		ParentType: differentParentType,
		Key:        key,
		Value:      newValue,
		Namespace:  namespace,
	}
	_, err = service.Create(ctx, createInput3)
	assert.NoError(t, err)

	// Test filtering by the original parent type
	filteredOutput, err := service.GetByValueWithoutOrg(ctx, key, newValue, parentType)
	assert.NoError(t, err)
	assert.Len(t, filteredOutput, 2) // Should only find the two entries with parentType="customer"

	// Test filtering by the different parent type
	differentFilteredOutput, err := service.GetByValueWithoutOrg(ctx, key, newValue, differentParentType)
	assert.NoError(t, err)
	assert.Len(t, differentFilteredOutput, 1) // Should only find the one entry with parentType="payment_method"
	assert.Equal(t, differentParentType, differentFilteredOutput[0].ParentType)

	// Test Delete
	err = service.Delete(ctx, orgId, parentId, key)
	assert.NoError(t, err)

	// Verify deletion
	_, err = service.GetByKey(ctx, orgId, parentId, key)
	assert.Error(t, err)
}
