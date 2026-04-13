package checkout_com

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/checkout/checkout-sdk-go"
	checkout_common "github.com/checkout/checkout-sdk-go/common"
	"github.com/checkout/checkout-sdk-go/configuration"
	checkout_errors "github.com/checkout/checkout-sdk-go/errors"
	cnas "github.com/checkout/checkout-sdk-go/nas"
	"github.com/checkout/checkout-sdk-go/payments"
	"github.com/checkout/checkout-sdk-go/payments/hosted"
	"github.com/checkout/checkout-sdk-go/payments/nas"
	"github.com/checkout/checkout-sdk-go/payments/nas/sources"
	payment_sessions "github.com/checkout/checkout-sdk-go/payments/sessions"
	"payloop/internal/core/domain"
	"payloop/internal/core/port"
)

type CheckoutDotCom struct {
	logger port.Logger
	config CheckoutDotComConfig
	client *cnas.Api
}

type CheckoutDotComConfig struct {
	SecretKey           string `json:"secret_key"`
	ProcessingChannelId string `json:"processing_channel_id"`
}

func (c CheckoutDotComConfig) Validate() error {
	return nil
}

func NewCheckoutDotComGateway(logger port.Logger, config CheckoutDotComConfig) domain.GatewayProvider {
	api, _ := checkout.
		Builder().
		StaticKeys().
		WithSecretKey(config.SecretKey).
		WithEnvironment(configuration.Sandbox()). // or Env.PRODUCTION
		Build()

	return CheckoutDotCom{
		config: config,
		logger: logger,
		client: api,
	}
}

func (p CheckoutDotCom) InitPayment(ctx context.Context, input domain.InitPaymentCommand) (domain.InitPaymentResponse, error) {
	reference := input.Order.Reference
	email := input.Customer.Email
	var options InitPaymentOptions

	if input.Options != nil {
		optionsJSON, err := json.Marshal(input.Options)
		if err != nil {
			p.logger.Error("failed to marshal options to JSON", "error", err)
			return domain.InitPaymentResponse{}, err
		}
		err = json.Unmarshal(optionsJSON, &options)
		if err != nil {
			p.logger.Error("failed to unmarshal options from JSON", "error", err)
			return domain.InitPaymentResponse{}, err
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
		// TODO: restore cart field access once cart type is resolved (input.Cart is interface{})
		response, err := p.client.Hosted.CreateHostedPaymentsPageSession(hosted.HostedPaymentRequest{
			Amount:              0, // int(input.Cart.Total),
			Currency:            "", // checkout_common.Currency(input.Cart.Currency),
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
				// "cart_id":  input.Cart.Id,
				"org_id": input.OrgId,
				"phase":  "init",
			},
			Capture: true,
		})
		if err != nil {
			p.logger.Error("failed to request payment", "error", err)
			return domain.InitPaymentResponse{}, err
		}

		p.logger.Info("created Checkout.com payment session", "response", response)
		return domain.InitPaymentResponse{
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

		// TODO: restore cart field access once cart type is resolved (input.Cart is interface{})
		flowRequest := payment_sessions.PaymentSessionsRequest{
			Amount:              0, // input.Cart.Total,
			Currency:            "", // checkout_common.Currency(input.Cart.Currency),
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
				// "cart_id":  input.Cart.Id,
				"org_id": input.OrgId,
				"phase":  "init",
			},
			Capture: true,
		}

		response, err := p.client.PaymentSessions.RequestPaymentSessions(flowRequest)
		if err != nil {
			p.logger.Error("failed to request payment", "error", err)
			return domain.InitPaymentResponse{}, err
		}

		p.logger.Info("created Checkout.com payment session", "response", response)
		return domain.InitPaymentResponse{
			PspResponse: map[string]interface{}{
				"id":                    response.Id,
				"payment_session_token": response.PaymentSessionToken,
			},
		}, nil
	}

}

func (p CheckoutDotCom) ChargePayment(ctx context.Context, input domain.ChargePaymentCommand) domain.ChargePaymentResponse {
	//customer := input.Customer
	paymentMethod := input.PaymentMethod

	// API Keys
	api, err := checkout.
		Builder().
		StaticKeys().
		WithSecretKey(p.config.SecretKey).
		WithEnvironment(configuration.Sandbox()). // or Env.PRODUCTION
		Build()
	if err != nil {
		p.logger.Error("failed to build checkout.com api", "error", err)
		return domain.ChargePaymentResponse{
			Status: domain.ChargePaymentStatusError,
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

	requestJSON, _ := json.Marshal(request)
	p.logger.Info("request JSON", "request", string(requestJSON))

	p.logger.Info("recurring checkout.com payment", "reference", input.Reference, "currency", input.Currency, "amount", input.Amount)
	p.logger.Info("paymentMethod", "paymentMethod", paymentMethod.Token)
	response, err := api.Payments.RequestPayment(request, nil)
	if err != nil {
		p.logger.Error("error charging payment", "error", err)
		errjson, _ := json.Marshal(err)
		p.logger.Error("errjson", "errjson", string(errjson))
		var capierr checkout_errors.CheckoutAPIError
		if errors.As(err, &capierr) {
			p.logger.Error("checkout api error", "errorType", capierr.Data.ErrorType)
		}
		return domain.ChargePaymentResponse{
			Status: domain.ChargePaymentStatusError,
		}
	}

	p.logger.Info("charged payment", "id", response.Id, "reference", response.Reference, "responseSummary", response.ResponseSummary)
	return domain.ChargePaymentResponse{
		Status:        domain.ChargePaymentStatusSuccess,
		Psp:           domain.CheckoutDotCom,
		PspId:         response.Id,
		Reference:     response.Reference,
		AmountCharged: response.Amount,
		Currency:      domain.Currency(response.Currency),
		PaymentType:   string(response.Source.ResponseCardSource.Type),
		PspResponse:   response,
	}
}
