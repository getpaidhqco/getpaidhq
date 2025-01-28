package paystack

import (
	"context"
	"encoding/json"
	"errors"
	paystacklib "github.com/mdwt/paystack-go"
	"payloop/internal/domain/payment_providers"
	"payloop/internal/lib"
)

var PAYSTACK = "Paystack"

type Paystack struct {
	logger lib.Logger
}

func NewPaystackGateway(logger lib.Logger) payment_providers.Gateway {
	return Paystack{
		logger: logger,
	}
}

func (p Paystack) InitPayment(ctx context.Context, input payment_providers.InitPaymentCommand) (payment_providers.InitPaymentResponse, error) {
	apiKey := "sk_test_e39ce23869e6e677121a5e6ef691a8c3d835f0bb"

	cart := input.Cart
	currency := input.Cart.Currency
	reference := input.Order.Reference
	email := input.Customer.Email

	client := paystacklib.NewClient(apiKey)

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

func (p Paystack) ParseWebhook(ctx context.Context, data []byte) (payment_providers.PaymentWebhookContext, error) {
	p.logger.Info("handling Paystack webhook", "data", string(data))

	var payload WebhookPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		p.logger.Errorf("failed to unmarshal webhook payload", err.Error())
		return err
	}

	switch payload.Event {
	case "charge.success":
		webhook, err := p.parseChargeSuccess(payload.Data)
		if err != nil {
			p.logger.Errorf("failed to parse charge success", err.Error())
			return err
		}
		return payment_providers.PaymentWebhookContext{
			OrgId:   webhook.Metadata.OrgID,
			OrderId: webhook.Metadata.OrderID,
			Psp:     PAYSTACK,
			Status:  "success",
			RawData: []byte(payload.Data.(string)),
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

	return nil
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
