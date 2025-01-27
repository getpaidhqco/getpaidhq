package paystack

import (
	paystacklib "github.com/mdwt/paystack-go"
	"log"
	"payloop/internal/domain/payment_providers"
	"payloop/internal/lib"
)

type PaystackProvider struct {
	logger lib.Logger
}

func NewPaystackProvider(logger lib.Logger) payment_providers.PaymentProvider {
	return PaystackProvider{
		logger: logger,
	}
}

func (p PaystackProvider) InitPayment() error {
	apiKey := "sk_test_e39ce23869e6e677121a5e6ef691a8c3d835f0bb"

	// second param is an optional http client, allowing overriding of the HTTP client to use.
	// This is useful if you're running in a Google AppEngine environment
	// where the http.DefaultClient is not available.
	client := paystacklib.NewClient(apiKey, nil)

	transaction, err := client.Transaction.Initialize(&paystacklib.TransactionRequest{
		CallbackURL: "https://www.example.com",
		Reference:   "1123",
		Currency:    "ZAR",
		Amount:      10000,
		Email:       "test+1@checkoutjoy.com",
	})

	log.Printf("created transaction", transaction)
	if err != nil {
		return err
	}

	return nil
}
