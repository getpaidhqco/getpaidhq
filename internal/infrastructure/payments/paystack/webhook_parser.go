package paystack

import (
	"context"
	"encoding/json"
	"errors"
	paystacklib "github.com/mdwt/paystack-go"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/common"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/payment_providers"
	"payloop/internal/domain/repositories"
	"strconv"
	"time"
)

type WebhookParser struct {
	logger            logger.Logger
	paymentRepository repositories.PaymentRepository
	factory           PaystackFactory
}

func NewWebhookParser(
	paymentRepository repositories.PaymentRepository,
	factory PaystackFactory,
	logger logger.Logger,
) WebhookParser {
	return WebhookParser{
		logger:            logger,
		paymentRepository: paymentRepository,
		factory:           factory,
	}
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
			return payment_providers.PaymentWebhookContext{}, errors.New("missing org Id in webhook metadata")
		}
		if webhook.Metadata.OrderID == "" {
			p.logger.Errorf("missing order Id in webhook metadata")
			return payment_providers.PaymentWebhookContext{}, errors.New("missing order Id in webhook metadata")
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

	case "refund.processed":
		webhook, err := p.parseRefundProcessed(payload.Data)
		if err != nil {
			p.logger.Errorf("failed to parse: %s", err.Error())
			return payment_providers.PaymentWebhookContext{}, err
		}

		transactions, err := p.paymentRepository.ListByPspId(ctx, common.Paystack, webhook.TransactionReference)
		if err != nil {
			p.logger.Errorf("failed to find ListByPspId: %s", err.Error())
			return payment_providers.PaymentWebhookContext{}, err
		}
		if len(transactions) == 0 {
			p.logger.Errorf("failed to find transaction: %s", err.Error())
			return payment_providers.PaymentWebhookContext{}, errors.New("transaction not found")
		}

		var originalPayment entities.Payment
		for _, transaction := range transactions {
			p.logger.Debugf(`Checking transaction %s`, transaction.PspId)
			cli, err := p.factory.New(ctx, transaction.OrgId)
			if err != nil {
				p.logger.Errorf("failed to create paystack client: %s", err.Error())
				return payment_providers.PaymentWebhookContext{}, err
			}
			paystack, ok := cli.(Paystack)
			if !ok {
				p.logger.Errorf("failed to assert cli to Paystack")
				return payment_providers.PaymentWebhookContext{}, errors.New("invalid Paystack type")
			}

			paystackClient := paystacklib.NewPaystackApi(paystacklib.Options{
				ApiKey:    paystack.Config.ApiKey,
				ConnectId: paystack.Config.ConnectId,
			})
			refund, err := paystackClient.Refund.Fetch(ctx, webhook.ID)
			if err != nil {
				continue
			}
			p.logger.Debugf(`Found refund %s -> %s`, refund.ID, refund.Status)
			// we now have the original payment
			originalPayment = transaction
			break
		}

		return payment_providers.PaymentWebhookContext{
			Type:    payment_providers.PaymentRefunded,
			RawData: data,
			OrgId:   originalPayment.OrgId,
			OrderId: "",
			Psp:     common.Paystack,
			Status:  "success",
			Payment: payment_providers.Payment{
				Currency:    webhook.Currency,
				Reference:   originalPayment.Reference,
				PspId:       originalPayment.PspId,
				Amount:      webhook.Amount,
				PaidAt:      time.Now().UTC(),
				PspFee:      0,
				PlatformFee: 0,
			},
		}, nil

	case "charge.failed":
		p.logger.Info("charge failed")
	case "transfer.success":
		p.logger.Info("transfer success")
	case "transfer.failed":
		p.logger.Info("transfer failed")

		p.logger.Info("transfer failed")
	default:
		p.logger.Warnf("unknown event %s", payload.Event)
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

func (p WebhookParser) parseRefundProcessed(data interface{}) (RefundProcessed, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return RefundProcessed{}, err
	}

	var payload RefundProcessed
	// Unmarshal the rest of the payload
	if err := json.Unmarshal(jsonData, &payload); err != nil {
		return RefundProcessed{}, err
	}

	return payload, nil
}
