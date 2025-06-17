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
	settingType := "json"
	settingValue := `{"key1": "value1"}`

	// Create a new setting
	setting := entities.Setting{
		OrgId:    orgId,
		ParentId: parentId,
		Id:       settingId,
		Type:     settingType,
		Value:    settingValue,
	}

	// Test creating a new setting
	createdSetting, err := settingsService.UpsertSetting(ctx, setting)
	assert.NoError(t, err)
	assert.Equal(t, orgId, createdSetting.OrgId)
	assert.Equal(t, parentId, createdSetting.ParentId)
	assert.Equal(t, settingId, createdSetting.Id)
	assert.Equal(t, settingType, createdSetting.Type)
	assert.Equal(t, settingValue, createdSetting.Value)
	assert.False(t, createdSetting.CreatedAt.IsZero())
	assert.False(t, createdSetting.UpdatedAt.IsZero())

	// Update the setting with new values
	updatedValue := `{"key1": "value1", "key2": "value2"}`
	setting.Value = updatedValue

	// Test updating an existing setting
	updatedSetting, err := settingsService.UpsertSetting(ctx, setting)
	assert.NoError(t, err)
	assert.Equal(t, orgId, updatedSetting.OrgId)
	assert.Equal(t, parentId, updatedSetting.ParentId)
	assert.Equal(t, settingId, updatedSetting.Id)
	assert.Equal(t, settingType, updatedSetting.Type)

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
	invalidSetting := entities.Setting{
		OrgId:    orgId,
		ParentId: parentId,
		Id:       "invalid_json_setting",
		Type:     settingType,
		Value:    `{"key": "value"`, // Invalid JSON missing closing brace
	}
	_, err = settingsService.UpsertSetting(ctx, invalidSetting)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JSON")

	// Clean up - delete the test setting
	err = settingsService.DeleteSetting(ctx, orgId, parentId, settingId)
	assert.NoError(t, err)

	// Verify deletion
	_, err = settingsService.GetSettingRaw(ctx, orgId, parentId, settingId)
	assert.Error(t, err)
}
