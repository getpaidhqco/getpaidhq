package dto

// RefundPaymentInput represents input for refunding a payment
type RefundPaymentInput struct {
    Amount int64  `json:"amount"`
    Reason string `json:"reason,omitempty"`
}

// ProcessPaymentInput represents input for processing a payment
type ProcessPaymentInput struct {
    Amount         int64             `json:"amount"`
    Currency       string            `json:"currency"`
    PaymentMethodId string           `json:"payment_method_id"`
    Description    string            `json:"description,omitempty"`
    Metadata       map[string]string `json:"metadata,omitempty"`
}