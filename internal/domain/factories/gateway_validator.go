package factories

import (
	"fmt"
	"payloop/internal/domain/common"
	"payloop/internal/domain/payment_providers"
	"payloop/internal/infrastructure/payments/checkout_com"
	"payloop/internal/infrastructure/payments/paystack"
)

// GatewayValidatorFactory creates validators for different payment gateways
type GatewayValidatorFactory struct{}

// NewGatewayValidatorFactory creates a new GatewayValidatorFactory
func NewGatewayValidatorFactory() GatewayValidatorFactory {
	return GatewayValidatorFactory{}
}

// GetValidator returns a validator for the specified gateway type
func (f GatewayValidatorFactory) GetValidator(gatewayType common.Gateway) (payment_providers.GatewayValidator, error) {
	switch gatewayType {
	case common.Paystack:
		return paystack.PaystackValidator{}, nil
	case common.CheckoutDotCom:
		return checkout_com.CheckoutDotComValidator{}, nil
	default:
		return nil, fmt.Errorf("unsupported gateway type: %s", gatewayType)
	}
}