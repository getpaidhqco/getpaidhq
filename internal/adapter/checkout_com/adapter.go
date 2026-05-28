package checkout_com

import (
	"encoding/json"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

// Adapter implements port.GatewayAdapter for Checkout.com.
type Adapter struct {
	logger        port.Logger
	webhookSecret string
}

// NewAdapter wires the Checkout.com adapter. webhookSecret is the
// HMAC-SHA256 signing key from the Checkout.com dashboard. Empty
// secret = fail-closed (every webhook rejected).
func NewAdapter(logger port.Logger, webhookSecret string) *Adapter {
	return &Adapter{logger: logger, webhookSecret: webhookSecret}
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
	return NewWebhookParser(a.logger, a.webhookSecret)
}
