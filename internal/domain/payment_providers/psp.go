package payment_providers

type PaymentProvider interface {
	InitPayment() error
}
