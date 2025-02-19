package paystack

import (
	"context"
	"errors"
	paystacklib "github.com/mdwt/paystack-go"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/payment_providers"
	"strconv"
)

var PAYSTACK = "Paystack"

type Paystack struct {
	logger logger.Logger
	config PaystackConfig
}

type PaystackConfig struct {
	ApiKey string `json:"api_key"`
}

func NewPaystackGateway(logger logger.Logger, config PaystackConfig) payment_providers.Gateway {
	return Paystack{
		config: config,
		logger: logger,
	}
}

func (p Paystack) InitPayment(ctx context.Context, input payment_providers.InitPaymentCommand) (payment_providers.InitPaymentResponse, error) {
	cart := input.Cart
	currency := input.Cart.Currency
	reference := input.Order.Reference
	email := input.Customer.Email

	client := paystacklib.NewClient(p.config.ApiKey)

	request := paystacklib.TransactionRequest{
		CallbackURL: "https://www.example.com",
		Reference:   reference,
		Currency:    currency,
		Amount:      float32(cart.Total),
		Email:       email,
		Metadata: paystacklib.Metadata{
			"order_id": input.Order.Id,
			"cart_id":  input.Cart.Id,
			"org_id":   input.OrgId,
			"custom_fields": []paystacklib.MetadataCustomField{{
				DisplayName:  "order_id",
				VariableName: "Order#",
				Value:        input.Order.Id,
			}},
		},
	}

	transaction, err := client.Transaction.Initialize(ctx, &request)
	if err != nil {
		p.logger.Errorf("failed to init paystack payment [%s]", err.Error())
		return payment_providers.InitPaymentResponse{}, err
	}
	p.logger.Info("created Paystack transaction", "reference", transaction.Reference, "code", transaction.AccessCode)
	return payment_providers.InitPaymentResponse{
		PspResponse: transaction,
	}, nil
}

func (p Paystack) ChargePayment(ctx context.Context, input payment_providers.ChargePaymentCommand) payment_providers.ChargePaymentResponse {
	client := paystacklib.NewClient(p.config.ApiKey)
	customer := input.Customer
	paymentMethod := input.PaymentMethod

	request := paystacklib.ChargeAuthorizationRequest{
		Amount:            input.Amount,
		Email:             customer.Email,
		AuthorizationCode: paymentMethod.Token,
		Reference:         input.Reference,
		Currency:          input.Currency,
		Metadata:          nil,
	}

	response, err := client.Transaction.ChargeAuthorization(ctx, request)
	if err != nil {
		p.logger.Errorf("failed to charge payment [%s]", err.Error())
		var paystackErr *paystacklib.APIError
		if errors.As(err, &paystackErr) {
			return payment_providers.ChargePaymentResponse{
				Success:     false,
				Retryable:   true,
				Psp:         PAYSTACK,
				PspResponse: paystackErr,
			}
		}

		return payment_providers.ChargePaymentResponse{
			Success:     false,
			Retryable:   false,
			Psp:         PAYSTACK,
			PspResponse: err,
		}
	}

	p.logger.Info("charged payment", "response", response.GatewayResponse)
	return payment_providers.ChargePaymentResponse{
		Success:       true,
		Psp:           PAYSTACK,
		PspId:         strconv.FormatInt(response.ID, 10),
		Reference:     response.Reference,
		AmountCharged: response.Amount,
		Currency:      response.Currency,
		PaymentType:   response.Channel,
		PspResponse:   response,
	}
}
