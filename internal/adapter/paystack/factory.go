package paystack

import (
	"context"
	"encoding/json"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

type PaystackFactory struct {
	pspRepository     port.PspRepository
	settingRepository port.SettingRepository
	logger            port.Logger
}

func NewPaystackFactory(
	pspRepository port.PspRepository,
	settingRepository port.SettingRepository,
	logger port.Logger,
) PaystackFactory {
	return PaystackFactory{
		pspRepository:     pspRepository,
		settingRepository: settingRepository,
		logger:            logger,
	}
}

func (s PaystackFactory) New(ctx context.Context, orgId string) (domain.GatewayProvider, error) {

	psp, err := s.pspRepository.FindById(ctx, orgId, string(domain.Paystack))
	if err != nil {
		return nil, err
	}

	setting, err := s.settingRepository.FindById(ctx, orgId, psp.Id, "settings")
	if err != nil {
		return nil, err
	}
	var config PaystackConfig
	err = json.Unmarshal([]byte(setting.Value), &config)
	if err != nil {
		s.logger.Error("Failed to unmarshal setting value", "error", err)
		return nil, err
	}
	err = config.Validate()
	if err != nil {
		return nil, lib.NewCustomError(lib.ValidationError, "invalid Config", err)
	}

	gw := NewPaystackGateway(s.logger, config)

	return gw, nil
}
