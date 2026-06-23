package paystack

import (
	"context"
	"errors"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"strconv"

	paystacklib "github.com/mdwt/paystack-go"
	pscommon "github.com/mdwt/paystack-go/common"
	pserrors "github.com/mdwt/paystack-go/errors"
	"github.com/mdwt/paystack-go/transactions"
)

type Paystack struct {
	logger port.Logger
	Config PaystackConfig
}

// PaystackConfig mirrors the stored gateway settings split: ApiKey is the
// merchant SECRET key and arrives Secret-typed from the credentials map —
// logging or marshaling this struct prints "[redacted]"; Reveal() only at
// SDK client construction. ConnectId/Type are non-secret config.
type PaystackConfig struct {
	Type      string
	ApiKey    domain.Secret
	ConnectId string
}

// ParseConfig builds a PaystackConfig from the stored config/credentials
// maps. Shared by the GatewayFactory path (adapter.CreateGateway) and the
// webhook-side PaystackFactory so the field mapping lives in one place.
// A secret mis-filed under config is NOT picked up — the gateway fails
// validation loudly instead of silently using a readable secret.
func ParseConfig(config map[string]string, credentials map[string]domain.Secret) (PaystackConfig, error) {
	c := PaystackConfig{
		Type:      config["type"],
		ApiKey:    credentials["api_key"],
		ConnectId: config["connect_id"],
	}
	return c, c.Validate()
}

func (c PaystackConfig) Validate() error {
	if c.ApiKey.IsZero() {
		return errors.New("api_key is required in credentials")
	}
	return nil
}

func NewPaystackGateway(logger port.Logger, config PaystackConfig) port.PaymentGateway {
	return Paystack{
		Config: config,
		logger: logger,
	}
}

func (p Paystack) InitPayment(ctx context.Context, input port.InitPaymentInput) (port.InitPaymentResponse, error) {
	cart := input.Cart
	currency := cart.Data.Currency
	reference := input.Order.Reference
	email := input.Customer.Email

	client := paystacklib.NewPaystackApi(paystacklib.Options{
		ApiKey:    p.Config.ApiKey.Reveal(),
		ConnectId: p.Config.ConnectId,
	})

	request := transactions.TransactionRequest{
		Reference: reference,
		Currency:  currency,
		Amount:    float32(cart.Total),
		Email:     email,
		Metadata: pscommon.Metadata{
			"order_id": input.Order.Id,
			"cart_id":  cart.Id,
			"org_id":   input.OrgId,
			"custom_fields": []pscommon.MetadataCustomField{{
				DisplayName:  "order_id",
				VariableName: "Order#",
				Value:        input.Order.Id,
			}},
		},
	}

	transaction, err := client.Transaction.Initialize(ctx, &request)
	if err != nil {
		p.logger.Errorf("failed to init paystack payment [%s]", err.Error())
		return port.InitPaymentResponse{}, err
	}
	p.logger.Info("created Paystack transaction", "reference", transaction.Reference, "code", transaction.AccessCode)
	return port.InitPaymentResponse{
		PspResponse: transaction,
	}, nil
}

func (p Paystack) ChargePayment(ctx context.Context, input port.ChargePaymentInput) port.ChargePaymentResponse {
	client := paystacklib.NewPaystackApi(paystacklib.Options{
		ApiKey:    p.Config.ApiKey.Reveal(),
		ConnectId: p.Config.ConnectId,
	})
	p.logger.Infof("charging payment for connect account %s", p.Config.ConnectId)

	customer := input.Customer
	paymentMethod := input.PaymentMethod

	request := transactions.ChargeAuthorizationRequest{
		Amount:            input.Amount,
		Email:             customer.Email,
		AuthorizationCode: paymentMethod.Token,
		Reference:         input.Reference,
		Currency:          input.Currency,
		Queue:             true,
		Metadata: pscommon.Metadata{
			"org_id":   input.OrgId,
			"order_id": input.OrderId,
			"type":     "recurring",
		},
	}

	// Never log the marshaled request — AuthorizationCode is the customer's
	// reusable card charge token and Email is PII; logging them would put
	// the log aggregator inside PCI scope. Correlation fields only.
	p.logger.Debugf("ChargeAuthorization reference=%s currency=%s amount=%d", request.Reference, request.Currency, request.Amount)

	response, err := client.Transaction.ChargeAuthorization(ctx, request)
	if err != nil {
		p.logger.Errorf("failed to charge payment [%s]", err.Error())
		if paystackErr, ok := errors.AsType[*pserrors.APIError](err); ok {

			if paystackErr.HTTPStatusCode == 429 {
				return port.ChargePaymentResponse{
					Status:        port.ChargePaymentStatusGatewayError,
					Retryable:     false,
					Psp:           domain.Paystack,
					ErrorReason:   paystackErr.Details.Message,
					ErrorCode:     strconv.Itoa(paystackErr.HTTPStatusCode),
					Currency:      domain.Currency(input.Currency),
					AmountCharged: input.Amount,
					PspResponse:   paystackErr,
				}
			}

			return port.ChargePaymentResponse{
				Status:        port.ChargePaymentStatusError,
				Retryable:     true,
				Psp:           domain.Paystack,
				ErrorReason:   paystackErr.Details.Message,
				ErrorCode:     strconv.Itoa(paystackErr.HTTPStatusCode),
				PspId:         "",
				Reference:     "",
				Currency:      domain.Currency(input.Currency),
				AmountCharged: input.Amount,
				PaymentType:   "",
				PspResponse:   paystackErr,
			}
		}

		return port.ChargePaymentResponse{
			Status:        port.ChargePaymentStatusError,
			Retryable:     true,
			Psp:           domain.Paystack,
			ErrorReason:   err.Error(),
			ErrorCode:     "500",
			PspId:         "",
			Reference:     "",
			Currency:      domain.Currency(input.Currency),
			AmountCharged: input.Amount,
			PaymentType:   "",
			PspResponse:   err,
		}
	}

	// check the status of the payment - "success" or "queued"
	status := port.ChargePaymentStatusSuccess
	//if response.Status == "queued" {
	//	status = port.ChargePaymentStatusPending
	//}

	p.logger.Infof("ChargeAuthorization [%s][%s]", response.Status, response.Reference)
	return port.ChargePaymentResponse{
		Status:        status,
		Psp:           domain.Paystack,
		PspId:         strconv.FormatInt(response.ID, 10),
		Reference:     response.Reference,
		AmountCharged: response.Amount,
		Currency:      domain.Currency(response.Currency),
		PaymentType:   response.Channel,
		PspResponse:   response,
	}
}
