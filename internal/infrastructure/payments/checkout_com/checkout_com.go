package checkout_com

import (
	"context"
	"encoding/json"
	"github.com/checkout/checkout-sdk-go"
	checkout_common "github.com/checkout/checkout-sdk-go/common"
	"github.com/checkout/checkout-sdk-go/configuration"
	cnas "github.com/checkout/checkout-sdk-go/nas"
	"github.com/checkout/checkout-sdk-go/payments"
	"github.com/checkout/checkout-sdk-go/payments/hosted"
	"github.com/checkout/checkout-sdk-go/payments/nas"
	"github.com/checkout/checkout-sdk-go/payments/nas/sources"
	payment_sessions "github.com/checkout/checkout-sdk-go/payments/sessions"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/common"
	"payloop/internal/domain/payment_providers"
)

var CHECKOUT_DOT_COM = "CheckoutDotCom"

type CheckoutDotCom struct {
	logger logger.Logger
	config CheckoutDotComConfig
	client *cnas.Api
}

type CheckoutDotComConfig struct {
	SecretKey           string `json:"secret_key"`
	ProcessingChannelId string `json:"processing_channel_id"`
}

func NewCheckoutDotComGateway(logger logger.Logger, config CheckoutDotComConfig) payment_providers.Gateway {
	api, _ := checkout.
		Builder().
		StaticKeys().
		WithSecretKey(config.SecretKey).
		WithEnvironment(configuration.Sandbox()). // or Environment.PRODUCTION
		Build()

	return CheckoutDotCom{
		config: config,
		logger: logger,
		client: api,
	}
}

func (p CheckoutDotCom) InitPayment(ctx context.Context, input payment_providers.InitPaymentCommand) (payment_providers.InitPaymentResponse, error) {
	reference := input.Order.Reference
	email := input.Customer.Email
	var options InitPaymentOptions

	if input.Options != nil {
		optionsJSON, err := json.Marshal(input.Options)
		if err != nil {
			p.logger.Error("failed to marshal options to JSON", "error", err)
			return payment_providers.InitPaymentResponse{}, err
		}
		err = json.Unmarshal(optionsJSON, &options)
		if err != nil {
			p.logger.Error("failed to unmarshal options from JSON", "error", err)
			return payment_providers.InitPaymentResponse{}, err
		}
	}

	switch options.Type {
	case "hosted":
		billing := payments.BillingInformation{
			Address: &checkout_common.Address{
				AddressLine1: "123 High St.",
				AddressLine2: "Flat 456",
				City:         "London",
				State:        "GB",
				Zip:          "SW1A 1AA",
				Country:      checkout_common.GB,
			},
		}
		customer := checkout_common.CustomerRequest{
			Email: email,
		}
		response, err := p.client.Hosted.CreateHostedPaymentsPageSession(hosted.HostedPaymentRequest{
			Amount:              int(input.Cart.Total),
			Currency:            checkout_common.Currency(input.Cart.Currency),
			PaymentType:         payments.Recurring,
			Billing:             &billing,
			Reference:           reference,
			Description:         reference,
			Customer:            &customer,
			ProcessingChannelId: p.config.ProcessingChannelId,
			DisplayName:         "",
			SuccessUrl:          "https://example.com/success",
			CancelUrl:           "https://example.com/failure",
			FailureUrl:          "https://example.com/failure",
			Metadata: map[string]interface{}{
				"order_id": input.Order.Id,
				"cart_id":  input.Cart.Id,
				"org_id":   input.OrgId,
				"phase":    "init",
			},
			Capture: true,
		})
		if err != nil {
			p.logger.Error("failed to request payment", "error", err)
			return payment_providers.InitPaymentResponse{}, err
		}

		p.logger.Info("created Checkout.com payment session", "response", response)
		return payment_providers.InitPaymentResponse{
			PspResponse: map[string]interface{}{
				"redirect": response.Links["redirect"].HRef,
			},
		}, nil

	default:
		billing := payments.BillingInformation{
			Address: &checkout_common.Address{
				AddressLine1: "123 High St.",
				AddressLine2: "Flat 456",
				City:         "London",
				State:        "GB",
				Zip:          "SW1A 1AA",
				Country:      checkout_common.GB,
			},
		}
		customer := checkout_common.CustomerRequest{
			Email: email,
		}

		flowRequest := payment_sessions.PaymentSessionsRequest{
			Amount:              input.Cart.Total,
			Currency:            checkout_common.Currency(input.Cart.Currency),
			PaymentType:         payments.Recurring,
			Billing:             &billing,
			Reference:           reference,
			Description:         reference,
			Customer:            &customer,
			ProcessingChannelId: p.config.ProcessingChannelId,
			Items:               nil,
			AmountAllocations:   nil,
			Risk:                nil,
			CustomerRetry:       nil,
			DisplayName:         "",
			SuccessUrl:          "https://example.com/success",
			FailureUrl:          "https://example.com/failure",
			Metadata: map[string]interface{}{
				"order_id": input.Order.Id,
				"cart_id":  input.Cart.Id,
				"org_id":   input.OrgId,
				"phase":    "init",
			},
			Capture: true,
		}

		response, err := p.client.PaymentSessions.RequestPaymentSessions(flowRequest)
		if err != nil {
			p.logger.Error("failed to request payment", "error", err)
			return payment_providers.InitPaymentResponse{}, err
		}

		p.logger.Info("created Checkout.com payment session", "response", response)
		return payment_providers.InitPaymentResponse{
			PspResponse: map[string]interface{}{
				"id":                    response.Id,
				"payment_session_token": response.PaymentSessionToken,
			},
		}, nil
	}

}

func (p CheckoutDotCom) ChargePayment(ctx context.Context, input payment_providers.ChargePaymentCommand) payment_providers.ChargePaymentResponse {
	//customer := input.Customer
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

	request := nas.PaymentRequest{
		Source:              source,
		Amount:              input.Amount,
		Currency:            checkout_common.Currency(input.Currency),
		Reference:           input.Reference,
		PaymentType:         "Recurring",
		MerchantInitiated:   true,
		Capture:             true,
		ProcessingChannelId: p.config.ProcessingChannelId,
		SuccessUrl:          "https://docs.checkout.com/success",
		FailureUrl:          "https://docs.checkout.com/failure",
		Metadata: map[string]interface{}{
			"order_id":        input.OrderId,
			"org_id":          input.OrgId,
			"subscription_id": input.SubscriptionId,
			"phase":           "recurring",
		},
	}

	p.logger.Infof("Recurring Checkout.com payment [%s][%s %s]", input.Reference, input.Currency, input.Amount)
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
