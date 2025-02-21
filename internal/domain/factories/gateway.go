package factories

import (
	"context"
	"encoding/json"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/common"
	"payloop/internal/domain/payment_providers"
	"payloop/internal/domain/repositories"
	"payloop/internal/infrastructure/payments/checkout_com"
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

func (s GatewayFactory) NewGateway(ctx context.Context, orgId string, id common.Gateway) (payment_providers.Gateway, error) {
	setting, err := s.settingRepository.FindById(ctx, orgId, "payment_processors", string(id))
	if err != nil {
		s.logger.Errorf("Failed to get [payment_processors][%s] - %e", id, err)
		return nil, err
	}

	switch id {
	case common.Paystack:
		var config paystack.PaystackConfig
		err = json.Unmarshal([]byte(setting.Value), &config)
		if err != nil {
			s.logger.Error("Failed to unmarshal setting value", "error", err)
			return nil, err
		}

		return paystack.NewPaystackGateway(s.logger, config), nil
	case common.CheckoutDotCom:
		var config checkout_com.CheckoutDotComConfig
		err = json.Unmarshal([]byte(setting.Value), &config)
		if err != nil {
			s.logger.Error("Failed to unmarshal setting value", "error", err)
			return nil, err
		}

		return checkout_com.NewCheckoutDotComGateway(s.logger, config), nil
	default:
		return nil, lib.NewCustomError(lib.BadRequestError, "Invalid payment processor", nil)
	}

}

func (s GatewayFactory) NewWebhookParser(psp common.Gateway) payment_providers.WebhookParser {
	switch psp {
	case common.Paystack:
		return paystack.NewWebhookParser(s.logger)
	case common.CheckoutDotCom:
		return checkout_com.NewWebhookParser(s.logger)
	default:
		return nil
	}
}
