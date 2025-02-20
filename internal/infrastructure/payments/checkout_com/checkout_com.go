package checkout_com

import (
	"context"
	"github.com/checkout/checkout-sdk-go"
	checkout_common "github.com/checkout/checkout-sdk-go/common"
	"github.com/checkout/checkout-sdk-go/configuration"
	"github.com/checkout/checkout-sdk-go/payments"
	"github.com/checkout/checkout-sdk-go/payments/nas"
	"github.com/checkout/checkout-sdk-go/payments/nas/sources"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/common"
	"payloop/internal/domain/payment_providers"
)

var CHECKOUT_DOT_COM = "CheckoutDotCom"

type CheckoutDotCom struct {
	logger logger.Logger
	config CheckoutDotComConfig
}

type CheckoutDotComConfig struct {
	SecretKey string `json:"secret_key"`
}

func NewCheckoutDotComGateway(logger logger.Logger, config CheckoutDotComConfig) payment_providers.Gateway {
	return CheckoutDotCom{
		config: config,
		logger: logger,
	}
}

func (p CheckoutDotCom) InitPayment(ctx context.Context, input payment_providers.InitPaymentCommand) (payment_providers.InitPaymentResponse, error) {
	cart := input.Cart
	currency := input.Cart.Currency
	reference := input.Order.Reference
	email := input.Customer.Email

	// API Keys
	api, err := checkout.
		Builder().
		StaticKeys().
		WithSecretKey(p.config.SecretKey).
		WithEnvironment(configuration.Sandbox()). // or Environment.PRODUCTION
		Build()
	if err != nil {
		p.logger.Error("failed to build checkout.com api", "error", err)
		return payment_providers.InitPaymentResponse{}, err
	}

	flowRequest := nas.PaymentRequest{
		Source:    sources.NewRequestCardSource(),
		Amount:    int64(cart.Total),
		Currency:  checkout_common.Currency(currency),
		Reference: reference,
		Capture:   true,
		Customer: &checkout_common.CustomerRequest{
			Email: email,
		},
		SuccessUrl: "https://www.example.com/success",
		FailureUrl: "https://www.example.com/failure",
	}

	response, err := api.Payments.RequestPayment(flowRequest, nil)
	if err != nil {
		p.logger.Error("failed to request payment", "error", err)
		return payment_providers.InitPaymentResponse{}, err
	}

	p.logger.Info("created Checkout.com payment session", "response", response)
	return payment_providers.InitPaymentResponse{
		PspResponse: response,
	}, nil

}

func (p CheckoutDotCom) ChargePayment(ctx context.Context, input payment_providers.ChargePaymentCommand) payment_providers.ChargePaymentResponse {
	customer := input.Customer
	paymentMethod := input.PaymentMethod

	// API Keys
	api, err := checkout.
		Builder().
		StaticKeys().
		WithSecretKey(p.config.SecretKey).
		WithEnvironment(configuration.Sandbox()). // or Environment.PRODUCTION
		Build()
	if err != nil {
		p.logger.Error("failed to build checkout.com api", "error", err)
		return payment_providers.ChargePaymentResponse{
			Success: false,
		}
	}

	source := sources.NewRequestIdSource()
	source.Id = paymentMethod.Token

	sender := nas.NewRequestIndividualSender()
	sender.FirstName = customer.Name
	sender.LastName = customer.Name
	sender.Address = &checkout_common.Address{
		AddressLine1: "123 High St.",
		AddressLine2: "Flat 456",
		City:         "London",
		State:        "GB",
		Zip:          "SW1A 1AA",
		Country:      checkout_common.GB,
	}

	request := nas.PaymentRequest{
		Source:    source,
		Amount:    10,
		Currency:  checkout_common.GBP,
		Reference: "reference",
		Capture:   false,
		ThreeDsRequest: &payments.ThreeDsRequest{
			Enabled:            true,
			ChallengeIndicator: checkout_common.NoChallengeRequested,
		},
		ProcessingChannelId: "processing_channel_id",
		SuccessUrl:          "https://docs.checkout.com/success",
		FailureUrl:          "https://docs.checkout.com/failure",
		Sender:              sender,
	}

	response, err := api.Payments.RequestPayment(request, nil)
	if err != nil {
		return payment_providers.ChargePaymentResponse{
			Success: false,
		}
	}

	p.logger.Info("charged payment", "response", response)
	return payment_providers.ChargePaymentResponse{
		Success:       true,
		Psp:           common.CheckoutDotCom,
		PspId:         response.Id,
		Reference:     response.Reference,
		AmountCharged: response.Amount,
		Currency:      common.Currency(response.Currency),
		PaymentType:   string(response.Source.ResponseCardSource.Type),
		PspResponse:   response,
	}
}
