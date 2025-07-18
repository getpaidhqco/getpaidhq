package services

import (
	"context"
	"encoding/json"
	"fmt"
	"payloop/internal/application/dto"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/common"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/factories"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
)

type PspService struct {
	pspRepository     repositories.PspRepository
	settingRepository repositories.SettingRepository
	pubsub            events.NotificationPublisher
	logger            logger.Logger
}

func NewPspService(pspRepository repositories.PspRepository,
	settingRepository repositories.SettingRepository,
	logger logger.Logger,
	pubsub events.NotificationPublisher,
) interfaces.GatewayService {
	return PspService{
		pspRepository:     pspRepository,
		settingRepository: settingRepository,
		logger:            logger,
		pubsub:            pubsub,
	}
}

func (s PspService) GetSettingsSchema(ctx context.Context, gatewayType string) (interface{}, error) {
	validatorFactory := factories.NewGatewayValidatorFactory()
	validator, err := validatorFactory.GetValidator(common.Gateway(gatewayType))
	if err != nil {
		s.logger.Errorf("Failed to get validator - %v", err)
		return nil, fmt.Errorf("invalid gateway type: %w", err)
	}

	schema := validator.GetSettingsSchema()
	return schema, nil
}

func (s PspService) CreateGateway(ctx context.Context, input dto.CreateGatewayInput) (entities.Gateway, error) {
	// Get validator for the gateway type
	validatorFactory := factories.NewGatewayValidatorFactory()
	validator, err := validatorFactory.GetValidator(input.PspId)
	if err != nil {
		s.logger.Errorf("Failed to get validator - %v", err)
		return entities.Gateway{}, fmt.Errorf("invalid gateway type: %w", err)
	}

	// Validate settings
	if err := validator.ValidateSettings(input.Settings); err != nil {
		s.logger.Errorf("Invalid settings - %v", err)
		return entities.Gateway{}, fmt.Errorf("invalid settings: %w", err)
	}

	id := lib.GenerateId("psp")
	psp, err := s.pspRepository.Create(ctx,
		entities.Gateway{
			OrgId:  input.OrgId,
			Id:     id,
			Name:   input.Name,
			PspId:  input.PspId,
			Active: true,
		})
	if err != nil {
		s.logger.Errorf("Failed to create psp - %v", err)
		return entities.Gateway{}, err
	}

	settingsJson, err := json.Marshal(input.Settings)
	if err != nil {
		s.logger.Errorf("Failed to marshal settings - %v", err)
		return entities.Gateway{}, err
	}

	// Create the psp settings
	_, err = s.settingRepository.Create(ctx, entities.Setting{
		OrgId:    input.OrgId,
		ParentId: id,
		Id:       "settings",
		Type:     "psp",
		Value:    string(settingsJson),
	})
	if err != nil {
		s.logger.Errorf("Failed to create psp settings - %v", err)
		return entities.Gateway{}, err
	}

	return psp, err
}

func (s PspService) GetGateway(ctx context.Context, orgId string, id string) (entities.Gateway, map[string]string, error) {
	// Get the gateway
	gateway, err := s.pspRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Errorf("Failed to get gateway - %v", err)
		return entities.Gateway{}, nil, err
	}

	// Get the settings
	setting, err := s.settingRepository.FindById(ctx, orgId, id, "settings")
	if err != nil {
		s.logger.Errorf("Failed to get gateway settings - %v", err)
		return entities.Gateway{}, nil, err
	}

	// Parse the settings
	var settings map[string]string
	if err := json.Unmarshal([]byte(setting.Value), &settings); err != nil {
		s.logger.Errorf("Failed to unmarshal settings - %v", err)
		return entities.Gateway{}, nil, err
	}

	return gateway, settings, nil
}

func (s PspService) ListGateways(ctx context.Context, filter dto.GatewayFilter) ([]entities.Gateway, error) {
	// Get all gateways for the organization
	gateways, err := s.pspRepository.FindAll(ctx, filter.OrgId)
	if err != nil {
		s.logger.Errorf("Failed to list gateways - %v", err)
		return nil, err
	}

	return gateways, nil
}

func (s PspService) UpdateGateway(ctx context.Context, input dto.UpdateGatewayInput) (entities.Gateway, error) {
	// Get the gateway
	gateway, err := s.pspRepository.FindById(ctx, input.OrgId, input.Id)
	if err != nil {
		s.logger.Errorf("Failed to get gateway - %v", err)
		return entities.Gateway{}, err
	}

	// Get validator for the gateway type
	validatorFactory := factories.NewGatewayValidatorFactory()
	validator, err := validatorFactory.GetValidator(gateway.PspId)
	if err != nil {
		s.logger.Errorf("Failed to get validator - %v", err)
		return entities.Gateway{}, fmt.Errorf("invalid gateway type: %w", err)
	}

	// Validate settings
	if err := validator.ValidateSettings(input.Settings); err != nil {
		s.logger.Errorf("Invalid settings - %v", err)
		return entities.Gateway{}, fmt.Errorf("invalid settings: %w", err)
	}

	// Update the gateway
	gateway.Name = input.Name
	updatedGateway, err := s.pspRepository.Update(ctx, gateway)
	if err != nil {
		s.logger.Errorf("Failed to update gateway - %v", err)
		return entities.Gateway{}, err
	}

	// Marshal the settings
	settingsJson, err := json.Marshal(input.Settings)
	if err != nil {
		s.logger.Errorf("Failed to marshal settings - %v", err)
		return entities.Gateway{}, err
	}

	// Get the settings
	setting, err := s.settingRepository.FindById(ctx, input.OrgId, input.Id, "settings")
	if err != nil {
		s.logger.Errorf("Failed to get gateway settings - %v", err)
		return entities.Gateway{}, err
	}

	// Update the settings
	setting.Value = string(settingsJson)
	_, err = s.settingRepository.Update(ctx, setting)
	if err != nil {
		s.logger.Errorf("Failed to update gateway settings - %v", err)
		return entities.Gateway{}, err
	}

	return updatedGateway, nil
}

func (s PspService) DeleteGateway(ctx context.Context, orgId string, id string) error {
	// Delete the settings
	err := s.settingRepository.Delete(ctx, orgId, id, "settings")
	if err != nil {
		s.logger.Errorf("Failed to delete gateway settings - %v", err)
		return err
	}

	// Delete the gateway
	err = s.pspRepository.Delete(ctx, orgId, id)
	if err != nil {
		s.logger.Errorf("Failed to delete gateway - %v", err)
		return err
	}

	return nil
}
