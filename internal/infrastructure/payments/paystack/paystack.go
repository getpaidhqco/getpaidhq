package paystack

import (
	"context"
	paystacklib "github.com/mdwt/paystack-go"
	"payloop/internal/domain/payment_providers"
	"payloop/internal/lib"
)

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
