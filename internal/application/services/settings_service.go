package services

import (
	"context"
	"encoding/json"
	"errors"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"time"
)

type SettingsService struct {
	settingRepository repositories.SettingRepository
	logger            logger.Logger
}

func NewSettingsService(
	settingRepository repositories.SettingRepository,
	logger logger.Logger,
) interfaces.SettingsService {
	return &SettingsService{
		settingRepository: settingRepository,
		logger:            logger,
	}
}

func (s *SettingsService) GetSetting(ctx context.Context, orgId string, parentId string, id string, result interface{}) error {
	setting, err := s.settingRepository.FindById(ctx, orgId, parentId, id)
	if err != nil {
		s.logger.Error("Failed to get setting", "err", err.Error())
		return err
	}

	// Unmarshal the setting value into the provided type
	if err := json.Unmarshal([]byte(setting.Value), result); err != nil {
		s.logger.Error("Failed to unmarshal setting value", "err", err.Error())
		return errors.New("invalid setting value format")
	}

	return nil
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

func (s *SettingsService) UpsertSetting(ctx context.Context, orgId string, parentId string, id string, value interface{}) (entities.Setting, error) {
	// Try to find the existing setting
	_, err := s.settingRepository.FindById(ctx, orgId, parentId, id)

	// If the setting doesn't exist, create it
	if err != nil {
		return s.CreateSetting(ctx, orgId, parentId, id, value)
	} else {
		return s.UpdateSetting(ctx, orgId, parentId, id, value)
	}
}
