package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"time"
)

type SettingsService struct {
	settingRepository repositories.SettingRepository
	registry          SettingsRegistryInterface
	logger            logger.Logger
}

func NewSettingsService(
	settingRepository repositories.SettingRepository,
	registry SettingsRegistryInterface,
	logger logger.Logger,
) interfaces.SettingsService {
	return &SettingsService{
		settingRepository: settingRepository,
		registry:          registry,
		logger:            logger,
	}
}

func (s *SettingsService) GetSetting(ctx context.Context, orgId string, parentId string, id string, result interface{}) error {
	// Retrieve from database
	setting, err := s.settingRepository.FindById(ctx, orgId, parentId, id)
	if err != nil {
		s.logger.Error("Failed to get setting", "err", err.Error())
		return err
	}

	// Get validator from registry
	validator, err := s.registry.GetValidator(id)
	if err != nil {
		return fmt.Errorf("no validator found for setting type %s: %w", id, err)
	}

	// Determine the secure type based on the validator
	secureValue := s.getSecureTypeForValidator(id)
	if err := json.Unmarshal([]byte(setting.Value), secureValue); err != nil {
		s.logger.Error("Failed to unmarshal secure settings", "err", err.Error())
		return fmt.Errorf("failed to unmarshal secure settings: %w", err)
	}

	// Restore sensitive data
	decryptedValue, err := validator.RestoreSensitiveData(ctx, secureValue)
	if err != nil {
		s.logger.Error("Failed to restore sensitive data", "err", err.Error())
		return fmt.Errorf("failed to restore sensitive data: %w", err)
	}

	// Marshal and unmarshal to populate the result
	jsonData, err := json.Marshal(decryptedValue)
	if err != nil {
		s.logger.Error("Failed to marshal decrypted data", "err", err.Error())
		return fmt.Errorf("failed to marshal decrypted data: %w", err)
	}

	return json.Unmarshal(jsonData, result)
}

func (s *SettingsService) GetSettingRaw(ctx context.Context, orgId string, parentId string, id string) (interface{}, error) {
	setting, err := s.settingRepository.FindById(ctx, orgId, parentId, id)
	if err != nil {
		s.logger.Error("Failed to get setting", "err", err.Error())
		return entities.Setting{}, err
	}
	var value interface{}
	if err := json.Unmarshal([]byte(setting.Value), &value); err != nil {
		s.logger.Error("Failed to unmarshal setting value", "err", err.Error())
		return nil, errors.New("invalid setting value format")
	}

	return value, nil
}

func (s *SettingsService) ListSettings(ctx context.Context, orgId string, parentId string) ([]entities.Setting, error) {
	settings, err := s.settingRepository.FindAll(ctx, orgId, parentId)
	if err != nil {
		s.logger.Error("Failed to list settings", "err", err.Error())
		return nil, err
	}

	return settings, nil
}

func (s *SettingsService) CreateSetting(ctx context.Context, orgId string, parentId string, id string, value interface{}) (entities.Setting, error) {
	valueBytes, err := json.Marshal(value)
	if err != nil {
		s.logger.Error("Failed to marshal new setting value", "err", err.Error())
		return entities.Setting{}, errors.New("invalid new setting value format")
	}

	createdSetting, err := s.settingRepository.Create(ctx, entities.Setting{
		OrgId:    orgId,
		ParentId: parentId,
		Id:       id,
		Value:    string(valueBytes),
	})
	if err != nil {
		s.logger.Error("Failed to create setting", "err", err.Error())
		return entities.Setting{}, err
	}

	return createdSetting, nil
}

func (s *SettingsService) UpdateSetting(ctx context.Context, orgId string, parentId string, id string, value interface{}) (entities.Setting, error) {
	// Get the existing setting to preserve created_at
	existingSetting, err := s.settingRepository.FindById(ctx, orgId, parentId, id)
	if err != nil {
		s.logger.Error("Failed to get existing setting for update", "err", err.Error())
		return entities.Setting{}, err
	}

	valueBytes, err := json.Marshal(value)
	if err != nil {
		s.logger.Error("Failed to marshal new setting value", "err", err.Error())
		return entities.Setting{}, errors.New("invalid new setting value format")
	}

	existingSetting.UpdatedAt = time.Now().UTC()
	existingSetting.Value = string(valueBytes)

	updatedSetting, err := s.settingRepository.Update(ctx, existingSetting)
	if err != nil {
		s.logger.Error("Failed to update setting", "err", err.Error())
		return entities.Setting{}, err
	}

	return updatedSetting, nil
}

func (s *SettingsService) DeleteSetting(ctx context.Context, orgId string, parentId string, id string) error {
	err := s.settingRepository.Delete(ctx, orgId, parentId, id)
	if err != nil {
		s.logger.Error("Failed to delete setting", "err", err.Error())
		return err
	}

	return nil
}

// getSecureTypeForValidator returns the appropriate type for a given setting type
func (s *SettingsService) getSecureTypeForValidator(settingType string) interface{} {
	// Try to get validator from registry
	validator, err := s.registry.GetValidator(settingType)
	if err != nil {
		// If no validator is found, return a generic map
		return &map[string]interface{}{}
	}

	// Use the validator's default value as the type template
	return validator.GetDefaultValue()
}

func (s *SettingsService) UpsertSetting(ctx context.Context, orgId string, parentId string, id string, value interface{}) (entities.Setting, error) {
	var jsonValue []byte
	var err error

	// Get validator from registry
	validator, err := s.registry.GetValidator(id)
	if err != nil {
		// If no validator is found, proceed without validation
		s.logger.Info("No validator found for setting type, proceeding without validation", "setting_type", id)

		// Use the raw value directly
		jsonValue, err = json.Marshal(value)
		if err != nil {
			return entities.Setting{}, fmt.Errorf("failed to marshal settings: %w", err)
		}
	}
	// Validate the settings
	if err := validator.ValidateSettings(value); err != nil {
		return entities.Setting{}, fmt.Errorf("validation failed: %w", err)
	}

	// Prepare sensitive data for storage
	secureValue, err := validator.PrepareSensitiveData(ctx, value)
	if err != nil {
		return entities.Setting{}, fmt.Errorf("failed to secure sensitive data: %w", err)
	}

	// Marshal the secure value
	jsonValue, err = json.Marshal(secureValue)
	if err != nil {
		return entities.Setting{}, fmt.Errorf("failed to marshal settings: %w", err)
	}

	// Create or update the setting
	setting := entities.Setting{
		OrgId:    orgId,
		ParentId: parentId,
		Id:       id,
		Type:     id, // Use id as the type
		Value:    string(jsonValue),
	}

	// Try to find the existing setting
	_, err = s.settingRepository.FindById(ctx, orgId, parentId, id)

	// If the setting doesn't exist, create it
	if err != nil {
		return s.settingRepository.Create(ctx, setting)
	} else {
		setting.UpdatedAt = time.Now().UTC()
		return s.settingRepository.Update(ctx, setting)
	}
}
