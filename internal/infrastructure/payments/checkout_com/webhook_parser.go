package checkout_com

import (
	"context"
	"encoding/json"
	"errors"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/common"
	"payloop/internal/domain/entities/payments"
	"payloop/internal/domain/payment_providers"
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
	p.logger.Info("[CheckoutDotCom] parsing webhook")

	var payload WebhookPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		p.logger.Errorf("failed to unmarshal webhook payload", err.Error())
		return payment_providers.PaymentWebhookContext{}, err
	}

	switch payload.Type {
	case PaymentCapturedWebhook:
		_, err := parseData[PaymentCaptured](payload.Data)
		if err != nil {
			p.logger.Errorf("failed to parse charge success: %s", err.Error())
			return payment_providers.PaymentWebhookContext{}, err
		}

		return payment_providers.PaymentWebhookContext{
			Type:    payment_providers.Noop,
			RawData: data,
		}, nil
	case PaymentApprovedWebhook:
		webhook, err := parseData[PaymentApproved](payload.Data)
		if err != nil {
			p.logger.Errorf("failed to parse charge success: %s", err.Error())
			return payment_providers.PaymentWebhookContext{}, err
		}

		orgId := webhook.Metadata.OrgID
		orderId := webhook.Metadata.OrderID
		phase := webhook.Metadata.Phase

		if orgId == "" || orderId == "" {
			p.logger.Errorf("missing orgId or orderId")
			return payment_providers.PaymentWebhookContext{}, errors.New("missing orgId or orderId")
		}
		if phase == "recurring" {
			p.logger.Debugf("Recurring charge webhook, ignoring")
			return payment_providers.PaymentWebhookContext{
				Type: payment_providers.Noop,
			}, nil
		}

		return payment_providers.PaymentWebhookContext{
			Type:    payment_providers.PaymentSuccess,
			OrgId:   orgId,
			OrderId: orderId,
			Psp:     common.CheckoutDotCom,
			Status:  "success",
			Payment: payment_providers.Payment{
				Currency:    webhook.Currency,
				Reference:   webhook.Reference,
				PspId:       webhook.ID,
				Amount:      int64(webhook.Amount),
				PaidAt:      webhook.ProcessedOn,
				PspFee:      0,
				PlatformFee: 0,
			},
			Customer: payment_providers.Customer{
				Id:        webhook.Customer.ID,
				Email:     webhook.Customer.Email,
				FirstName: "",
				LastName:  "",
				Phone:     "",
				PspId:     webhook.Customer.ID,
			},
			PaymentMethod: payment_providers.PaymentMethod{
				PspId:       webhook.Source.ID,
				Name:        webhook.Source.Name,
				Type:        webhook.Source.Type,
				IsRecurring: true,
				Token:       webhook.Source.ID,
			},
			RawData: data,
		}, nil

	case PaymentRefundedWebhook:
		webhook, err := parseData[PaymentRefunded](payload.Data)
		if err != nil {
			p.logger.Errorf("failed to parse PaymentRefundedWebhook: %s", err.Error())
			return payment_providers.PaymentWebhookContext{}, err
		}

		orgId := webhook.Metadata["org_id"]
		orderId := webhook.Metadata["order_id"]

		if orgId == "" || orderId == "" {
			p.logger.Errorf("missing orgId or orderId")
			return payment_providers.PaymentWebhookContext{}, errors.New("missing orgId or orderId")
		}

		return payment_providers.PaymentWebhookContext{
			Type:    payment_providers.PaymentRefunded,
			OrgId:   orgId,
			OrderId: orderId,
			Psp:     common.CheckoutDotCom,
			Status:  string(payments.PaymentStatusRefunded),
			Payment: payment_providers.Payment{
				Currency:    webhook.Currency,
				Reference:   webhook.Reference,
				PspId:       webhook.ID,
				Amount:      int64(webhook.Amount),
				PaidAt:      webhook.ProcessedOn,
				PspFee:      0,
				PlatformFee: 0,
			},
			RawData: data,
		}, nil
	default:
		p.logger.Info("unknown event", "event", payload.Type)
	}

	return payment_providers.PaymentWebhookContext{}, errors.New("unknown event")
}

func parseData[T WebhookData](data interface{}) (T, error) {
	var payload T
	jsonData, err := json.Marshal(data)
	if err != nil {
		return payload, err
	}

	if err := json.Unmarshal(jsonData, &payload); err != nil {
		return payload, err
	}
	return payload, nil
}
