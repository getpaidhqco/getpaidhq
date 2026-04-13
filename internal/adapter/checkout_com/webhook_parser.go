package checkout_com

import (
	"context"
	"encoding/json"
	"errors"
	"payloop/internal/core/domain"
	"payloop/internal/core/port"
)

type WebhookParserAdapter struct {
	logger port.Logger
}

func NewWebhookParser(logger port.Logger) WebhookParserAdapter {
	return WebhookParserAdapter{
		logger: logger,
	}
}

func (p WebhookParserAdapter) ValidateWebhook(ctx context.Context, data []byte) error {
	return nil
}

func (p WebhookParserAdapter) ParseWebhook(ctx context.Context, data []byte) (domain.PaymentWebhookContext, error) {
	p.logger.Info("[CheckoutDotCom] parsing webhook")

	var payload WebhookPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		p.logger.Errorf("failed to unmarshal webhook payload", err.Error())
		return domain.PaymentWebhookContext{}, err
	}

	switch payload.Type {
	case PaymentCapturedWebhook:
		_, err := parseData[PaymentCaptured](payload.Data)
		if err != nil {
			p.logger.Errorf("failed to parse charge success: %s", err.Error())
			return domain.PaymentWebhookContext{}, err
		}

		return domain.PaymentWebhookContext{
			Type:    domain.Noop,
			RawData: data,
		}, nil
	case PaymentApprovedWebhook:
		webhook, err := parseData[PaymentApproved](payload.Data)
		if err != nil {
			p.logger.Errorf("failed to parse charge success: %s", err.Error())
			return domain.PaymentWebhookContext{}, err
		}

		orgId := webhook.Metadata.OrgID
		orderId := webhook.Metadata.OrderID
		phase := webhook.Metadata.Phase

		if orgId == "" || orderId == "" {
			p.logger.Errorf("missing orgId or orderId")
			return domain.PaymentWebhookContext{}, errors.New("missing orgId or orderId")
		}
		if phase == "recurring" {
			p.logger.Debugf("Recurring charge webhook, ignoring")
			return domain.PaymentWebhookContext{
				Type: domain.Noop,
			}, nil
		}

		return domain.PaymentWebhookContext{
			Type:    domain.PaymentSuccess,
			OrgId:   orgId,
			OrderId: orderId,
			Psp:     domain.CheckoutDotCom,
			Status:  "success",
			Payment: domain.GatewayPayment{
				Currency:    webhook.Currency,
				Reference:   webhook.Reference,
				PspId:       webhook.ID,
				Amount:      int64(webhook.Amount),
				PaidAt:      webhook.ProcessedOn,
				PspFee:      0,
				PlatformFee: 0,
			},
			Customer: domain.GatewayCustomer{
				Id:        webhook.Customer.ID,
				Email:     webhook.Customer.Email,
				FirstName: "",
				LastName:  "",
				Phone:     "",
				PspId:     webhook.Customer.ID,
			},
			PaymentMethod: domain.GatewayPaymentMethod{
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
			return domain.PaymentWebhookContext{}, err
		}
		orgId := webhook.Metadata["org_id"]
		if orgId == "" {
			p.logger.Errorf("missing orgId ")
			return domain.PaymentWebhookContext{}, errors.New("missing orgId")
		}

		return domain.PaymentWebhookContext{
			Type:    domain.PaymentRefunded,
			OrgId:   orgId,
			OrderId: "",
			Psp:     domain.CheckoutDotCom,
			Status:  string(domain.PaymentStatusRefunded),
			Payment: domain.GatewayPayment{
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

	return domain.PaymentWebhookContext{}, errors.New("unknown event")
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
