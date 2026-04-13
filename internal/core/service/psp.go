package service

import (
	"context"
	"encoding/json"
	"payloop/internal/core/domain"
	"payloop/internal/core/port"
	"payloop/internal/lib"
)

type PspService struct {
	pspRepository     port.PspRepository
	settingRepository port.SettingRepository
	pubsub            port.PubSub
	logger            port.Logger
}

func NewPspService(
	pspRepository port.PspRepository,
	settingRepository port.SettingRepository,
	logger port.Logger,
	pubsub port.PubSub,
) *PspService {
	return &PspService{
		pspRepository:     pspRepository,
		settingRepository: settingRepository,
		logger:            logger,
		pubsub:            pubsub,
	}
}

func (s *PspService) CreateGateway(ctx context.Context, input port.CreateGatewayInput) (domain.PspConfig, error) {

	id := lib.GenerateId("psp")
	psp, err := s.pspRepository.Create(ctx,
		domain.PspConfig{
			OrgId:  input.OrgId,
			Id:     id,
			Name:   input.Name,
			PspId:  input.PspId,
			Active: true,
		})
	if err != nil {
		s.logger.Error("failed to create psp", "error", err)
		return domain.PspConfig{}, err
	}

	settingsJson, err := json.Marshal(input.Settings)
	if err != nil {
		s.logger.Error("failed to marshal settings", "error", err)
		return domain.PspConfig{}, err
	}

	// Create the psp settings
	_, err = s.settingRepository.Create(ctx, domain.Setting{
		OrgId:    input.OrgId,
		ParentId: id,
		Id:       "settings",
		Type:     "psp",
		Value:    string(settingsJson),
	})
	if err != nil {
		s.logger.Error("failed to create psp settings", "error", err)
		return domain.PspConfig{}, err
	}

	return psp, err
}
