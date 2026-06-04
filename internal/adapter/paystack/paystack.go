package paystack

import (
	"context"
	"encoding/json"
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

type PaystackConfig struct {
	Type      string `json:"type"`
	ApiKey    string `json:"api_key"`
	ConnectId string `json:"connect_id"`
}

func (c PaystackConfig) Validate() error {
	if c.ApiKey == "" {
		return errors.New("api_key is required")
	}
	return nil
}

func NewPaystackGateway(logger port.Logger, config PaystackConfig) domain.GatewayProvider {
	return Paystack{
		Config: config,
		logger: logger,
	}
}

func (p Paystack) InitPayment(ctx context.Context, input domain.InitPaymentCommand) (domain.InitPaymentResponse, error) {
	cart := input.Cart
	currency := cart.Data.Currency
	reference := input.Order.Reference
	email := input.Customer.Email

	client := paystacklib.NewPaystackApi(paystacklib.Options{
		ApiKey:    p.Config.ApiKey,
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
		return domain.InitPaymentResponse{}, err
	}
	p.logger.Info("created Paystack transaction", "reference", transaction.Reference, "code", transaction.AccessCode)
	return domain.InitPaymentResponse{
		PspResponse: transaction,
	}, nil
}

func (p Paystack) ChargePayment(ctx context.Context, input domain.ChargePaymentCommand) domain.ChargePaymentResponse {
	client := paystacklib.NewPaystackApi(paystacklib.Options{
		ApiKey:    p.Config.ApiKey,
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

	jsonR, _ := json.Marshal(request)
	p.logger.Debugf("ChargeAuthorization: %s", jsonR)

	response, err := client.Transaction.ChargeAuthorization(ctx, request)
	if err != nil {
		p.logger.Errorf("failed to charge payment [%s]", err.Error())
		if paystackErr, ok := errors.AsType[*pserrors.APIError](err); ok {

			if paystackErr.HTTPStatusCode == 429 {
				return domain.ChargePaymentResponse{
					Status:        domain.GatewayError,
					Retryable:     false,
					Psp:           domain.Paystack,
					ErrorReason:   paystackErr.Details.Message,
					ErrorCode:     strconv.Itoa(paystackErr.HTTPStatusCode),
					Currency:      domain.Currency(input.Currency),
					AmountCharged: input.Amount,
					PspResponse:   paystackErr,
				}
			}

			return domain.ChargePaymentResponse{
				Status:        domain.ChargePaymentStatusError,
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

		return domain.ChargePaymentResponse{
			Status:        domain.ChargePaymentStatusError,
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
	status := domain.ChargePaymentStatusSuccess
	//if response.Status == "queued" {
	//	status = domain.ChargePaymentStatusPending
	//}

	p.logger.Infof("ChargeAuthorization [%s][%s]", response.Status, response.Reference)
	return domain.ChargePaymentResponse{
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
