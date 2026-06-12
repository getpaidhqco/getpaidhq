package paystack

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	paystacklib "github.com/mdwt/paystack-go"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// ErrInvalidSignature is returned when the supplied X-Paystack-Signature
// header doesn't match the HMAC of the raw body. The caller (webhook
// service) treats this as a hard reject — no claim, no parsing.
var ErrInvalidSignature = errors.New("paystack: invalid webhook signature")

// ErrMissingWebhookSecret is returned when no secret is configured for
// signature verification. Fail-closed: without a secret we cannot
// distinguish a real Paystack event from a forged one, so we must
// reject everything rather than blindly trust the body.
var ErrMissingWebhookSecret = errors.New("paystack: no webhook secret configured (PAYSTACK_SECRET)")

type WebhookParser struct {
	logger            port.Logger
	paymentRepository port.PaymentRepository
	factory           PaystackFactory
	// secret is the Paystack merchant SECRET KEY (same value used as
	// the API auth token). Paystack signs webhook bodies with this
	// key using HMAC-SHA512.
	secret string
}

func NewWebhookParser(
	paymentRepository port.PaymentRepository,
	factory PaystackFactory,
	logger port.Logger,
	secret string,
) WebhookParser {
	return WebhookParser{
		logger:            logger,
		paymentRepository: paymentRepository,
		factory:           factory,
		secret:            secret,
	}
}

// ValidateWebhook verifies the X-Paystack-Signature header against an
// HMAC-SHA512 of the raw body computed with the merchant secret key.
// The comparison is constant-time. Returns ErrInvalidSignature on
// mismatch and ErrMissingWebhookSecret if no secret is configured.
func (p WebhookParser) ValidateWebhook(ctx context.Context, data []byte, signature string) error {
	if p.secret == "" {
		return ErrMissingWebhookSecret
	}
	if signature == "" {
		return ErrInvalidSignature
	}
	mac := hmac.New(sha512.New, []byte(p.secret))
	mac.Write(data)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return ErrInvalidSignature
	}
	return nil
}

func (p WebhookParser) ParseWebhook(ctx context.Context, data []byte) (domain.PaymentWebhookContext, error) {
	var payload WebhookPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		p.logger.Errorf("failed to unmarshal webhook payload: %s", err.Error())
		return domain.PaymentWebhookContext{}, err
	}

	switch payload.Event {
	case "charge.success":
		webhook, err := p.parseChargeSuccess(payload.Data)
		if err != nil {
			p.logger.Errorf("failed to parse charge success: %s", err.Error())
			return domain.PaymentWebhookContext{}, err
		}

		webhookType := domain.PaymentSuccess
		if webhook.Metadata["type"] == "recurring" {
			// we can safely ignore recurring payments as the result is handled sync
			webhookType = domain.RecurringSuccess
		}

		if orgID, ok := webhook.Metadata["org_id"]; !ok || orgID == "" {
			p.logger.Errorf("missing org_id in webhook metadata")
			return domain.PaymentWebhookContext{}, errors.New("missing org_id in webhook metadata")
		}
		if orderId, ok := webhook.Metadata["order_id"]; !ok || orderId == "" {
			p.logger.Errorf("missing order Id in webhook metadata")
			return domain.PaymentWebhookContext{}, errors.New("missing order_id in webhook metadata")
		}

		return domain.PaymentWebhookContext{
			Type:    webhookType,
			RawData: data,
			OrgId:   webhook.Metadata["org_id"].(string),
			OrderId: webhook.Metadata["order_id"].(string),
			Psp:     domain.Paystack,
			Status:  "success",
			Payment: domain.GatewayPayment{
				Currency:    webhook.Currency,
				Reference:   webhook.Reference,
				PspId:       strconv.FormatInt(webhook.ID, 10),
				Amount:      webhook.Amount,
				PaidAt:      webhook.PaidAt,
				PspFee:      webhook.Fees,
				PlatformFee: 0,
			},
			Customer: domain.GatewayCustomer{
				Id:        strconv.Itoa(webhook.Customer.ID),
				Email:     webhook.Customer.Email,
				FirstName: webhook.Customer.FirstName,
				LastName:  webhook.Customer.LastName,
				Phone:     webhook.Customer.Phone,
				PspId:     webhook.Customer.CustomerCode,
			},
			PaymentMethod: domain.GatewayPaymentMethod{
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
			return domain.PaymentWebhookContext{}, err
		}

		transactions, err := p.paymentRepository.ListByPspId(ctx, domain.Paystack, webhook.TransactionReference)
		if err != nil {
			p.logger.Errorf("failed to find ListByPspId: %s", err.Error())
			return domain.PaymentWebhookContext{}, err
		}
		if len(transactions) == 0 {
			p.logger.Debugf("No transaction with ref %s", webhook.TransactionReference)
			return domain.PaymentWebhookContext{}, errors.New("transaction not found")
		}

		var originalPayment domain.Payment
		for _, transaction := range transactions {
			p.logger.Debugf(`Checking transaction %s`, transaction.PspId)
			cli, err := p.factory.New(ctx, transaction.OrgId)
			if err != nil {
				p.logger.Errorf("failed to create paystack client: %s", err.Error())
				return domain.PaymentWebhookContext{}, err
			}
			paystack, ok := cli.(Paystack)
			if !ok {
				p.logger.Errorf("failed to assert cli to Paystack")
				return domain.PaymentWebhookContext{}, errors.New("invalid Paystack type")
			}

			paystackClient := paystacklib.NewPaystackApi(paystacklib.Options{
				ApiKey:    paystack.Config.ApiKey.Reveal(),
				ConnectId: paystack.Config.ConnectId,
			})
			refund, err := paystackClient.Refund.Fetch(ctx, webhook.ID)
			if err != nil {
				continue
			}
			p.logger.Debugf(`Found refund %d -> %s`, refund.ID, refund.Status)
			// we now have the original payment
			originalPayment = transaction
			break
		}

		return domain.PaymentWebhookContext{
			Type:    domain.PaymentRefunded,
			RawData: data,
			OrgId:   originalPayment.OrgId,
			OrderId: "",
			Psp:     domain.Paystack,
			Status:  "success",
			Payment: domain.GatewayPayment{
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

	return domain.PaymentWebhookContext{}, errors.New("unknown event")
}

func (p WebhookParser) parseChargeSuccess(data any) (TransactionSuccessful, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return TransactionSuccessful{}, err
	}

	var payload TransactionSuccessful
	var metadata map[string]any
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

func (p WebhookParser) parseRefundProcessed(data any) (RefundProcessed, error) {
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
