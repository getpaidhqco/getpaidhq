package paystack

import (
	"encoding/json"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

// Adapter implements port.GatewayAdapter for Paystack.
type Adapter struct {
	paymentRepo   port.PaymentRepository
	pspRepo       port.PspRepository
	settingRepo   port.SettingRepository
	logger        port.Logger
	webhookSecret string
}

// NewAdapter wires the Paystack adapter. webhookSecret is the merchant
// SECRET KEY used to sign webhooks (HMAC-SHA512 of the raw body). An
// empty secret puts the parser in fail-closed mode — every webhook
// will be rejected. Source it from env.PaystackSecret at the wiring
// layer.
func NewAdapter(
	paymentRepo port.PaymentRepository,
	pspRepo port.PspRepository,
	settingRepo port.SettingRepository,
	logger port.Logger,
	webhookSecret string,
) *Adapter {
	return &Adapter{
		paymentRepo:   paymentRepo,
		pspRepo:       pspRepo,
		settingRepo:   settingRepo,
		logger:        logger,
		webhookSecret: webhookSecret,
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
	return NewWebhookParser(a.paymentRepo, factory, a.logger, a.webhookSecret)
}
