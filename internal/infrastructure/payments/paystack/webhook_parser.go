package paystack

import (
	"context"
	"encoding/json"
	"errors"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/payment_providers"
	"strconv"
)

type WebhookParser struct {
	logger logger.Logger
}

func NewWebhookParser(logger logger.Logger) WebhookParser {
	return WebhookParser{logger: logger}
}

func (p WebhookParser) ValidateWebhook(ctx context.Context, data []byte) error {
	return nil
}

func (p WebhookParser) ParseWebhook(ctx context.Context, data []byte) (payment_providers.PaymentWebhookContext, error) {
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

func (p WebhookParser) parseChargeSuccess(data interface{}) (TransactionSuccessful, error) {
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
