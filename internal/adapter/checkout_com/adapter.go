package checkout_com

import (
	"encoding/json"
	"payloop/internal/core/domain"
	"payloop/internal/core/port"
	"payloop/internal/lib"
)

// Adapter implements port.GatewayAdapter for Checkout.com.
type Adapter struct {
	logger port.Logger
}

func NewAdapter(logger port.Logger) *Adapter {
	return &Adapter{logger: logger}
}

func (a *Adapter) CreateGateway(settingsJSON string) (domain.GatewayProvider, error) {
	var config CheckoutDotComConfig
	if err := json.Unmarshal([]byte(settingsJSON), &config); err != nil {
		return nil, err
	}
	if err := config.Validate(); err != nil {
		return nil, lib.NewCustomError(lib.ValidationError, "invalid config for CheckoutDotCom", err)
	}
	return NewCheckoutDotComGateway(a.logger, config), nil
}

func (a *Adapter) CreateWebhookParser() domain.WebhookParser {
	return NewWebhookParser(a.logger)
}
