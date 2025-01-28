package paystack

import (
	"context"
	paystacklib "github.com/mdwt/paystack-go"
	"log"
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

func (p Paystack) InitPayment(ctx context.Context, input payment_providers.InitPaymentCommand) error {
	apiKey := "sk_test_e39ce23869e6e677121a5e6ef691a8c3d835f0bb"

	cart := input.Cart
	currency := input.Cart.Currency
	reference := input.Order.Reference
	email := input.Customer.Email

	// second param is an optional http client, allowing overriding of the HTTP client to use.
	// This is useful if you're running in a Google AppEngine environment
	// where the http.DefaultClient is not available.
	client := paystacklib.NewClient(apiKey, nil)

	request := paystacklib.TransactionRequest{
		CallbackURL: "https://www.example.com",
		Reference:   reference,
		Currency:    currency,
		Amount:      float32(cart.Total),
		Email:       email,
	}

	transaction, err := client.Transaction.Initialize(&request)

	log.Printf("created Paystack transaction", transaction)
	if err != nil {
		p.logger.Errorf("failed to init paystack client", err.Error())
		return err
	}

	return nil
}
