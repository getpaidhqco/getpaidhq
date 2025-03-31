package services

import (
	"context"
	"encoding/json"
	"payloop/internal/application/dto"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
)

type PspService struct {
	pspRepository     repositories.PspRepository
	settingRepository repositories.SettingRepository
	pubsub            events.PubSub
	logger            logger.Logger
}

func NewPspService(pspRepository repositories.PspRepository,
	settingRepository repositories.SettingRepository,
	logger logger.Logger,
	pubsub events.PubSub,
) interfaces.GatewayService {
	return PspService{
		pspRepository:     pspRepository,
		settingRepository: settingRepository,
		logger:            logger,
		pubsub:            pubsub,
	}
}

func (s PspService) CreateGateway(ctx context.Context, input dto.CreateGatewayInput) (entities.Gateway, error) {

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
		s.logger.Errorf("Failed to create psp - %e", err)
		return entities.Gateway{}, err
	}

	settingsJson, err := json.Marshal(input.Settings)
	if err != nil {
		s.logger.Errorf("Failed to marshal settings - %e", err)
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
		s.logger.Errorf("Failed to create psp settings - %e", err)
		return entities.Gateway{}, err
	}

	return psp, err
}
