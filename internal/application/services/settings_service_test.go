package services_test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/stretchr/testify/assert"
	"payloop/internal/application/services"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"testing"
)

// MockLogger is a mock implementation of the logger.Logger interface
type MockLogger struct{}

func (l MockLogger) Debug(msg string, args ...any) {}
func (l MockLogger) Info(msg string, args ...any)  {}
func (l MockLogger) Warn(msg string, args ...any)  {}
func (l MockLogger) Error(msg string, args ...any) {}
func (l MockLogger) Fatal(msg string, args ...any) {}

func (l MockLogger) Debugf(template string, args ...interface{}) {}
func (l MockLogger) Infof(template string, args ...interface{})  {}
func (l MockLogger) Warnf(template string, args ...interface{})  {}
func (l MockLogger) Errorf(template string, args ...interface{}) {}
func (l MockLogger) Panicf(template string, args ...interface{}) {}
func (l MockLogger) Fatalf(template string, args ...interface{}) {}

func (l MockLogger) Sync() error { return nil }

// MockSettingRepository is a mock implementation of the SettingRepository interface
type MockSettingRepository struct {
	settings map[string]entities.Setting
}

func NewMockSettingRepository() repositories.SettingRepository {
	return &MockSettingRepository{
		settings: make(map[string]entities.Setting),
	}
}

func (m *MockSettingRepository) FindById(ctx context.Context, orgId string, parentId string, id string) (entities.Setting, error) {
	key := orgId + ":" + parentId + ":" + id
	setting, ok := m.settings[key]
	if !ok {
		return entities.Setting{}, errors.New("not found")
	}
	return setting, nil
}

func (m *MockSettingRepository) FindAll(ctx context.Context, orgId string, parentId string) ([]entities.Setting, error) {
	var result []entities.Setting
	for _, setting := range m.settings {
		if setting.OrgId == orgId && setting.ParentId == parentId {
			result = append(result, setting)
		}
	}
	return result, nil
}

func (m *MockSettingRepository) Create(ctx context.Context, entity entities.Setting) (entities.Setting, error) {
	key := entity.OrgId + ":" + entity.ParentId + ":" + entity.Id
	m.settings[key] = entity
	return entity, nil
}

func (m *MockSettingRepository) Update(ctx context.Context, entity entities.Setting) (entities.Setting, error) {
	key := entity.OrgId + ":" + entity.ParentId + ":" + entity.Id
	_, ok := m.settings[key]
	if !ok {
		return entities.Setting{}, errors.New("not found")
	}
	m.settings[key] = entity
	return entity, nil
}

func (m *MockSettingRepository) Delete(ctx context.Context, orgId string, parentId string, id string) error {
	key := orgId + ":" + parentId + ":" + id
	_, ok := m.settings[key]
	if !ok {
		return errors.New("not found")
	}
	delete(m.settings, key)
	return nil
}

func TestUpsertSetting(t *testing.T) {
	// Create mock repository and logger
	mockRepo := NewMockSettingRepository()
	mockLogger := MockLogger{}

	// Create settings service with mock dependencies
	settingsService := services.NewSettingsService(mockRepo, mockLogger)

	// Create context
	ctx := context.Background()

	// Test data
	orgId := "test_org"
	parentId := "test_parent"
	settingId := "test_setting"
	settingValue := `{"key1": "value1"}`

	// We'll use these values for testing

	// Test creating a new setting
	var value map[string]interface{}
	err := json.Unmarshal([]byte(settingValue), &value)
	assert.NoError(t, err)

	createdSetting, err := settingsService.UpsertSetting(ctx, orgId, parentId, settingId, value)
	assert.NoError(t, err)
	assert.Equal(t, orgId, createdSetting.OrgId)
	assert.Equal(t, parentId, createdSetting.ParentId)
	assert.Equal(t, settingId, createdSetting.Id)

	// Parse the JSON to compare it properly
	var createdValueMap map[string]interface{}
	err = json.Unmarshal([]byte(createdSetting.Value), &createdValueMap)
	assert.NoError(t, err)
	assert.Equal(t, "value1", createdValueMap["key1"])

	// Check that timestamps are set
	assert.NotZero(t, createdSetting.CreatedAt)
	assert.NotZero(t, createdSetting.UpdatedAt)

	// Update the setting with new values
	updatedValue := `{"key1": "value1", "key2": "value2"}`

	var updatedValueObj map[string]interface{}
	err = json.Unmarshal([]byte(updatedValue), &updatedValueObj)
	assert.NoError(t, err)

	// Test updating an existing setting
	updatedSetting, err := settingsService.UpsertSetting(ctx, orgId, parentId, settingId, updatedValueObj)
	assert.NoError(t, err)
	assert.Equal(t, orgId, updatedSetting.OrgId)
	assert.Equal(t, parentId, updatedSetting.ParentId)
	assert.Equal(t, settingId, updatedSetting.Id)

	// Parse the JSON to compare it properly
	var updatedValueMap map[string]interface{}
	err = json.Unmarshal([]byte(updatedSetting.Value), &updatedValueMap)
	assert.NoError(t, err)
	assert.Equal(t, "value1", updatedValueMap["key1"])
	assert.Equal(t, "value2", updatedValueMap["key2"])

	assert.Equal(t, createdSetting.CreatedAt, updatedSetting.CreatedAt)
	assert.True(t, updatedSetting.UpdatedAt.After(createdSetting.UpdatedAt) || 
		updatedSetting.UpdatedAt.Equal(createdSetting.UpdatedAt))

	// Test invalid JSON
	invalidSettingId := "invalid_json_setting"
	invalidJSON := `{"key": "value"` // Invalid JSON missing closing brace

	// Try to parse the invalid JSON
	var invalidValue interface{}
	err = json.Unmarshal([]byte(invalidJSON), &invalidValue)
	assert.Error(t, err)

	// Since we can't even parse the invalid JSON, we'll create a different test
	// We'll try to use a value that can't be properly marshaled to JSON
	invalidValue = make(chan int) // channels can't be marshaled to JSON

	_, err = settingsService.UpsertSetting(ctx, orgId, parentId, invalidSettingId, invalidValue)
	assert.Error(t, err)

	// Clean up - delete the test setting
	err = settingsService.DeleteSetting(ctx, orgId, parentId, settingId)
	assert.NoError(t, err)

	// Verify deletion
	_, err = settingsService.GetSettingRaw(ctx, orgId, parentId, settingId)
	assert.Error(t, err)
}
