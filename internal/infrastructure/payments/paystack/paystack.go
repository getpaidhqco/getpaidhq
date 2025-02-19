package paystack

import (
	"context"
	"encoding/json"
	"errors"
	paystacklib "github.com/mdwt/paystack-go"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/payment_providers"
	"payloop/internal/lib"
	"strconv"
)

var PAYSTACK = "Paystack"

type Paystack struct {
	logger logger.Logger
	env    lib.Env
}

func NewPaystackGateway(logger logger.Logger, env lib.Env) payment_providers.Gateway {
	if env.PaystackApiKey == "" {
		logger.Fatalf("Paystack API key is required")
	}
	return Paystack{
		env:    env,
		logger: logger,
	}
}

func (p Paystack) InitPayment(ctx context.Context, input payment_providers.InitPaymentCommand) (payment_providers.InitPaymentResponse, error) {
	cart := input.Cart
	currency := input.Cart.Currency
	reference := input.Order.Reference
	email := input.Customer.Email

	client := paystacklib.NewClient(p.env.PaystackApiKey)

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
		p.logger.Errorf("failed to init paystack client", err.Error())
		return payment_providers.InitPaymentResponse{}, err
	}
	p.logger.Info("created Paystack transaction", "reference", transaction.Reference, "code", transaction.AccessCode)
	return payment_providers.InitPaymentResponse{
		PspResponse: transaction,
	}, nil
}

func (p Paystack) ChargePayment(ctx context.Context, input payment_providers.ChargePaymentCommand) payment_providers.ChargePaymentResponse {
	client := paystacklib.NewClient(p.env.PaystackApiKey)
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

func (p Paystack) ParseWebhook(ctx context.Context, data []byte) (payment_providers.PaymentWebhookContext, error) {
	p.logger.Info("handling Paystack webhook", "data", string(data))

	var payload WebhookPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		p.logger.Errorf("failed to unmarshal webhook payload", err.Error())
		return payment_providers.PaymentWebhookContext{}, err
	}

	switch payload.Event {
	case "charge.success":
		webhook, err := p.parseChargeSuccess(payload.Data)
		if err != nil {
			p.logger.Errorf("failed to parse charge success", err.Error())
			return payment_providers.PaymentWebhookContext{}, err
		}
		return payment_providers.PaymentWebhookContext{
			OrgId:   webhook.Metadata.OrgID,
			OrderId: webhook.Metadata.OrderID,
			Psp:     PAYSTACK,
			Status:  "success",
			Payment: payment_providers.Payment{
				Currency:    webhook.Currency,
				Reference:   webhook.Reference,
				PspId:       strconv.FormatInt(webhook.ID, 10),
				Amount:      webhook.Amount,
				PaidAt:      webhook.PaidAt,
				PspFee:      webhook.Fees,
				PlatformFee: 0,
			},
			Customer: payment_providers.Customer{
				Id:        strconv.Itoa(webhook.Customer.ID),
				Email:     webhook.Customer.Email,
				FirstName: webhook.Customer.FirstName,
				LastName:  webhook.Customer.LastName,
				Phone:     webhook.Customer.Phone,
				PspId:     webhook.Customer.CustomerCode,
			},
			PaymentMethod: payment_providers.PaymentMethod{
				PspId:       webhook.Authorization.Signature,
				Name:        webhook.Authorization.Brand,
				Type:        webhook.Authorization.CardType,
				IsRecurring: webhook.Authorization.Reusable,
				Token:       webhook.Authorization.AuthorizationCode,
			},
			Type:    payment_providers.PaymentSuccess,
			RawData: data,
		}, nil

	case "charge.failed":
		p.logger.Info("charge failed")
	case "transfer.success":
		p.logger.Info("transfer success")
	case "transfer.failed":
		p.logger.Info("transfer failed")
	default:
		p.logger.Info("unknown event", "event", payload.Event)
	}

	return payment_providers.PaymentWebhookContext{}, errors.New("unknown event")
}

func (p Paystack) ValidateWebhook(ctx context.Context, data []byte) error {
	return nil
}

func (p Paystack) parseChargeSuccess(data interface{}) (TransactionSuccessful, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return TransactionSuccessful{}, errors.New("failed to marshal data to JSON")
	}

	var payload TransactionSuccessful
	if err := json.Unmarshal(jsonData, &payload); err != nil {
		return TransactionSuccessful{}, errors.New("failed to unmarshal JSON to TransactionSuccessful")
	}

	p.logger.Info("handling charge success", "reference", payload.Reference)
	return payload, nil
}
