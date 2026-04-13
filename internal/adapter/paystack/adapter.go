package paystack

import (
	"encoding/json"
	"payloop/internal/core/domain"
	"payloop/internal/core/port"
	"payloop/internal/lib"
)

// Adapter implements port.GatewayAdapter for Paystack.
type Adapter struct {
	paymentRepo port.PaymentRepository
	pspRepo     port.PspRepository
	settingRepo port.SettingRepository
	logger      port.Logger
}

func NewAdapter(
	paymentRepo port.PaymentRepository,
	pspRepo port.PspRepository,
	settingRepo port.SettingRepository,
	logger port.Logger,
) *Adapter {
	return &Adapter{
		paymentRepo: paymentRepo,
		pspRepo:     pspRepo,
		settingRepo: settingRepo,
		logger:      logger,
	}
}

func (a *Adapter) CreateGateway(settingsJSON string) (domain.GatewayProvider, error) {
	var config PaystackConfig
	if err := json.Unmarshal([]byte(settingsJSON), &config); err != nil {
		return nil, err
	}
	if err := config.Validate(); err != nil {
		return nil, lib.NewCustomError(lib.ValidationError, "invalid config", err)
	}
	return NewPaystackGateway(a.logger, config), nil
}

func (a *Adapter) CreateWebhookParser() domain.WebhookParser {
	factory := NewPaystackFactory(a.pspRepo, a.settingRepo, a.logger)
	return NewWebhookParser(a.paymentRepo, factory, a.logger)
}
