package factories

import (
	"context"
	"encoding/json"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/payment_providers"
	"payloop/internal/domain/repositories"
	"payloop/internal/infrastructure/payments/paystack"
	"payloop/internal/lib"
)

type GatewayFactory struct {
	settingRepository repositories.SettingRepository
	logger            logger.Logger
}

func NewGatewayFactory(
	settingRepository repositories.SettingRepository,
	logger logger.Logger,
) GatewayFactory {
	return GatewayFactory{
		settingRepository: settingRepository,
		logger:            logger,
	}
}

func (s GatewayFactory) NewGateway(ctx context.Context, orgId string, id string) (payment_providers.Gateway, error) {
	setting, err := s.settingRepository.FindById(ctx, orgId, "payment_processors", id)
	if err != nil {
		s.logger.Error("Failed to get setting", "error", err)
		return nil, err
	}

	switch id {
	case "Paystack":
		var config paystack.PaystackConfig
		err = json.Unmarshal([]byte(setting.Value), &config)
		if err != nil {
			s.logger.Error("Failed to unmarshal setting value", "error", err)
			return nil, err
		}

		return paystack.NewPaystackGateway(s.logger, config), nil
	default:
		return nil, lib.NewCustomError(lib.BadRequestError, "Invalid payment processor", nil)
	}

}
