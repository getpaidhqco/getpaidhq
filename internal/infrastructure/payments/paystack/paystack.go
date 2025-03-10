package paystack

import (
	"context"
	"encoding/json"
	"errors"
	paystacklib "github.com/mdwt/paystack-go"
	pscommon "github.com/mdwt/paystack-go/common"
	pserrors "github.com/mdwt/paystack-go/errors"
	"github.com/mdwt/paystack-go/transactions"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/common"
	"payloop/internal/domain/payment_providers"
	"strconv"
)

type Paystack struct {
	logger logger.Logger
	config PaystackConfig
}

type PaystackConfig struct {
	Type      string `json:"type"`
	ApiKey    string `json:"api_key"`
	ConnectId string `json:"connect_id"`
}

func NewPaystackGateway(logger logger.Logger, config PaystackConfig) payment_providers.Gateway {
	return Paystack{
		config: config,
		logger: logger,
	}
}

func (p Paystack) InitPayment(ctx context.Context, input payment_providers.InitPaymentCommand) (payment_providers.InitPaymentResponse, error) {
	cart := input.Cart
	currency := input.Cart.Currency
	reference := input.Order.Reference
	email := input.Customer.Email

	client := paystacklib.NewPaystackApi(paystacklib.Options{
		ApiKey:    p.config.ApiKey,
		ConnectId: p.config.ConnectId,
	})

	request := transactions.TransactionRequest{
		CallbackURL: "https://www.example.com",
		Reference:   reference,
		Currency:    currency,
		Amount:      float32(cart.Total),
		Email:       email,
		Metadata: pscommon.Metadata{
			"order_id": input.Order.Id,
			"cart_id":  input.Cart.Id,
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
		return payment_providers.InitPaymentResponse{}, err
	}
	p.logger.Info("created Paystack transaction", "reference", transaction.Reference, "code", transaction.AccessCode)
	return payment_providers.InitPaymentResponse{
		PspResponse: transaction,
	}, nil
}

func (p Paystack) ChargePayment(ctx context.Context, input payment_providers.ChargePaymentCommand) payment_providers.ChargePaymentResponse {
	client := paystacklib.NewPaystackApi(paystacklib.Options{
		ApiKey:    p.config.ApiKey,
		ConnectId: p.config.ConnectId,
	})
	p.logger.Infof("charging payment for connect account %s", p.config.ConnectId)

	customer := input.Customer
	paymentMethod := input.PaymentMethod

	request := transactions.ChargeAuthorizationRequest{
		Amount:            input.Amount,
		Email:             customer.Email,
		AuthorizationCode: paymentMethod.Token,
		Reference:         input.Reference,
		Currency:          input.Currency,
		Metadata: pscommon.Metadata{
			"org_id": input.OrgId,
			"type":   "recurring",
		},
	}

	jsonR, _ := json.Marshal(request)
	p.logger.Debugf("ChargeAuthorization: %s", jsonR)

	response, err := client.Transaction.ChargeAuthorization(ctx, request)
	if err != nil {
		p.logger.Errorf("failed to charge payment [%s]", err.Error())
		var paystackErr *pserrors.APIError
		if errors.As(err, &paystackErr) {
			return payment_providers.ChargePaymentResponse{
				Success:     false,
				Retryable:   true,
				Psp:         common.Paystack,
				PspResponse: paystackErr,
			}
		}

		return payment_providers.ChargePaymentResponse{
			Success:     false,
			Retryable:   false,
			Psp:         common.Paystack,
			PspResponse: err,
		}
	}

	p.logger.Info("charged payment", "response", response.GatewayResponse)
	return payment_providers.ChargePaymentResponse{
		Success:       true,
		Psp:           common.Paystack,
		PspId:         strconv.FormatInt(response.ID, 10),
		Reference:     response.Reference,
		AmountCharged: int64(response.Amount),
		Currency:      common.Currency(response.Currency),
		PaymentType:   response.Channel,
		PspResponse:   response,
	}
}
