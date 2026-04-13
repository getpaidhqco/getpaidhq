package service

import (
	"context"
	"payloop/internal/core/domain"
	"payloop/internal/core/port"
	"payloop/internal/lib"
)

// GatewayFactory creates payment gateway instances from stored PSP configuration.
// It uses a registry of GatewayAdapter implementations to avoid importing adapter packages directly.
type GatewayFactory struct {
	pspRepository     port.PspRepository
	settingRepository port.SettingRepository
	logger            port.Logger
	adapters          map[domain.Gateway]port.GatewayAdapter
}

func NewGatewayFactory(
	pspRepository port.PspRepository,
	settingRepository port.SettingRepository,
	logger port.Logger,
	adapters map[domain.Gateway]port.GatewayAdapter,
) *GatewayFactory {
	return &GatewayFactory{
		pspRepository:     pspRepository,
		settingRepository: settingRepository,
		logger:            logger,
		adapters:          adapters,
	}
}

func (s *GatewayFactory) NewGateway(ctx context.Context, orgId string, id string) (domain.GatewayProvider, error) {
	psp, err := s.pspRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("failed to get gateway", "gatewayId", id, "error", err)
		return nil, err
	}

	setting, err := s.settingRepository.FindById(ctx, orgId, psp.Id, "settings")
	if err != nil {
		s.logger.Error("failed to get settings", "gatewayId", id, "error", err)
		return nil, err
	}

	adapter, ok := s.adapters[psp.PspId]
	if !ok {
		return nil, lib.NewCustomError(lib.BadRequestError, "Invalid payment processor", nil)
	}

	return adapter.CreateGateway(setting.Value)
}

func (s *GatewayFactory) NewWebhookParser(psp domain.Gateway) domain.WebhookParser {
	adapter, ok := s.adapters[psp]
	if !ok {
		return nil
	}
	return adapter.CreateWebhookParser()
}
