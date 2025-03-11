package paystack

import (
	"context"
	"encoding/json"
	"errors"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/common"
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
	var payload WebhookPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		p.logger.Errorf("failed to unmarshal webhook payload", err.Error())
		return err
	}
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
			p.logger.Errorf("failed to parse charge success: %s", err.Error())
			return payment_providers.PaymentWebhookContext{}, err
		}

		webhookType := payment_providers.PaymentSuccess
		if webhook.Metadata.Type == "recurring" {
			// we can safely ignore recurring payments as the result is handled sync
			webhookType = payment_providers.RecurringSuccess
		}

		if webhook.Metadata.OrgID == "" {
			return payment_providers.PaymentWebhookContext{}, errors.New("missing org ID in webhook metadata")
		}
		if webhook.Metadata.OrderID == "" {
			p.logger.Errorf("missing order ID in webhook metadata")
			return payment_providers.PaymentWebhookContext{}, errors.New("missing order ID in webhook metadata")
		}

		return payment_providers.PaymentWebhookContext{
			Type:    webhookType,
			RawData: data,
			OrgId:   webhook.Metadata.OrgID,
			OrderId: webhook.Metadata.OrderID,
			Psp:     common.Paystack,
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
		return TransactionSuccessful{}, err
	}

	var payload TransactionSuccessful
	var metadata Metadata
	// metadata field is sometimes a string, and sometimes a struct
	var temp struct {
		Metadata json.RawMessage `json:"metadata"`
	}
	if err := json.Unmarshal(jsonData, &temp); err != nil {
		return TransactionSuccessful{}, err
	}

	// Try to unmarshal Metadata as a string
	var str string
	if err := json.Unmarshal(temp.Metadata, &str); err == nil {
		if str != "" {
			// It's a string, so marshal it back to a struct
			if err := json.Unmarshal([]byte(str), &metadata); err != nil {
				return TransactionSuccessful{}, err
			}
		}
	} else {
		// It's a struct, so use it directly
		if err := json.Unmarshal(temp.Metadata, &metadata); err != nil {
			return TransactionSuccessful{}, err
		}
	}

	// Unmarshal the rest of the payload
	if err := json.Unmarshal(jsonData, &payload); err != nil {
		return TransactionSuccessful{}, err
	}

	payload.Metadata = metadata

	p.logger.Info("handling charge success", "reference", payload.Reference)
	return payload, nil
}
