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
	pspRepository     repositories.PspRepository
	settingRepository repositories.SettingRepository
	logger            logger.Logger

	paystackWehbookParser paystack.WebhookParser
}

func NewGatewayFactory(
	pspRepository repositories.PspRepository,
	settingRepository repositories.SettingRepository,
	paystackWehbookParser paystack.WebhookParser,
	logger logger.Logger,
) GatewayFactory {
	return GatewayFactory{
		pspRepository:         pspRepository,
		settingRepository:     settingRepository,
		logger:                logger,
		paystackWehbookParser: paystackWehbookParser,
	}
}

func (s GatewayFactory) NewGateway(ctx context.Context, orgId string, id string) (payment_providers.Gateway, error) {

	psp, err := s.pspRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Errorf("Failed to get  Gateway[%s] - %s", id, err.Error())
		return nil, err
	}

	setting, err := s.settingRepository.FindById(ctx, orgId, psp.Id, "settings")
	if err != nil {
		s.logger.Errorf("Failed to get settings for %s - %e", id, err)
		return nil, err
	}

	switch psp.PspId {
	case common.Paystack:
		var config paystack.PaystackConfig
		err = json.Unmarshal([]byte(setting.Value), &config)
		if err != nil {
			s.logger.Error("Failed to unmarshal setting value", "error", err)
			return nil, err
		}
		err = config.Validate()
		if err != nil {
			return nil, lib.NewCustomError(lib.ValidationError, "invalid config", err)
		}

		return paystack.NewPaystackGateway(s.logger, config), nil
	case common.CheckoutDotCom:
		var config checkout_com.CheckoutDotComConfig
		err = json.Unmarshal([]byte(setting.Value), &config)
		if err != nil {
			s.logger.Error("Failed to unmarshal setting value", "error", err)
			return nil, lib.NewCustomError(lib.BadRequestError, "Invalid payment processor", nil)
		}
		err = config.Validate()
		if err != nil {
			return nil, lib.NewCustomError(lib.ValidationError, "invalid config for CheckoutDotCom", err)
		}

		return checkout_com.NewCheckoutDotComGateway(s.logger, config), nil
	default:
		return nil, lib.NewCustomError(lib.BadRequestError, "Invalid payment processor", nil)
	}

}

func (s GatewayFactory) NewWebhookParser(psp common.Gateway) payment_providers.WebhookParser {

	switch psp {
	case common.Paystack:
		return s.paystackWehbookParser
	case common.CheckoutDotCom:
		return checkout_com.NewWebhookParser(s.logger)
	default:
		return nil
	}
}
