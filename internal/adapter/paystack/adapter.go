package paystack

import (
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

// Adapter implements port.GatewayAdapter for Paystack.
type Adapter struct {
	paymentRepo   port.PaymentRepository
	pspRepo       port.PspRepository
	cipher        port.SecretCipher
	logger        port.Logger
	webhookSecret string
}

// NewAdapter wires the Paystack adapter. webhookSecret is the merchant
// SECRET KEY used to sign webhooks (HMAC-SHA512 of the raw body). An
// empty secret puts the parser in fail-closed mode — every webhook
// will be rejected. Source it from env.PaystackSecret at the wiring
// layer. cipher opens stored gateway credentials for the webhook-side
// factory.
func NewAdapter(
	paymentRepo port.PaymentRepository,
	pspRepo port.PspRepository,
	cipher port.SecretCipher,
	logger port.Logger,
	webhookSecret string,
) *Adapter {
	return &Adapter{
		paymentRepo:   paymentRepo,
		pspRepo:       pspRepo,
		cipher:        cipher,
		logger:        logger,
		webhookSecret: webhookSecret,
	}
}

func (a *Adapter) CreateGateway(config map[string]string, credentials map[string]domain.Secret) (domain.GatewayProvider, error) {
	c, err := ParseConfig(config, credentials)
	if err != nil {
		return nil, lib.NewCustomError(lib.ValidationError, "invalid config", err)
	}
	return NewPaystackGateway(a.logger, c), nil
}

func (a *Adapter) CreateWebhookParser() domain.WebhookParser {
	factory := NewPaystackFactory(a.pspRepo, a.cipher, a.logger)
	return NewWebhookParser(a.paymentRepo, factory, a.logger, a.webhookSecret)
}
