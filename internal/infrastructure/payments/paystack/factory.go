package paystack

import (
	"context"
	"encoding/json"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/common"
	"payloop/internal/domain/payment_providers"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
)

type PaystackFactory struct {
	pspRepository     repositories.PspRepository
	settingRepository repositories.SettingRepository
	logger            logger.Logger
}

func NewPaystackFactory(
	pspRepository repositories.PspRepository,
	settingRepository repositories.SettingRepository,
	logger logger.Logger,
) PaystackFactory {
	return PaystackFactory{
		pspRepository:     pspRepository,
		settingRepository: settingRepository,
		logger:            logger,
	}
}

func (s PaystackFactory) New(ctx context.Context, orgId string) (payment_providers.Gateway, error) {

	psp, err := s.pspRepository.FindById(ctx, orgId, string(common.Paystack))
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
