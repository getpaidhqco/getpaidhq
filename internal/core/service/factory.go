package service

import (
	"context"
	"encoding/json"
	"payloop/internal/core/domain"
	"payloop/internal/core/port"
	"payloop/internal/adapter/checkout_com"
	"payloop/internal/adapter/paystack"
	"payloop/internal/lib"
)

// GatewayFactory creates payment gateway instances from stored PSP configuration.
type GatewayFactory struct {
	pspRepository     port.PspRepository
	settingRepository port.SettingRepository
	logger            port.Logger

	paystackWebhookParser paystack.WebhookParser
}

func NewGatewayFactory(
	pspRepository port.PspRepository,
	settingRepository port.SettingRepository,
	paystackWebhookParser paystack.WebhookParser,
	logger port.Logger,
) *GatewayFactory {
	return &GatewayFactory{
		pspRepository:         pspRepository,
		settingRepository:     settingRepository,
		logger:                logger,
		paystackWebhookParser: paystackWebhookParser,
	}
}

func (s *GatewayFactory) NewGateway(ctx context.Context, orgId string, id string) (domain.GatewayProvider, error) {
	psp, err := s.pspRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Errorf("Failed to get Gateway[%s] - %s", id, err.Error())
		return nil, err
	}

	setting, err := s.settingRepository.FindById(ctx, orgId, psp.Id, "settings")
	if err != nil {
		s.logger.Errorf("Failed to get settings for %s - %v", id, err)
		return nil, err
	}

	switch psp.PspId {
	case domain.Paystack:
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

	case domain.CheckoutDotCom:
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

func (s *GatewayFactory) NewWebhookParser(psp domain.Gateway) domain.WebhookParser {
	switch psp {
	case domain.Paystack:
		return s.paystackWebhookParser
	case domain.CheckoutDotCom:
		return checkout_com.NewWebhookParser(s.logger)
	default:
		return nil
	}
}
