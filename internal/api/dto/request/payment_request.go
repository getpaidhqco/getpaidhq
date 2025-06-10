package request

// RefundPaymentRequest represents the request to refund a payment
type RefundPaymentRequest struct {
	// Reason is the reason for the refund
	Reason string `json:"reason,omitempty" binding:"omitempty"`
	
	// Amount is the amount to refund. If not provided, the full payment amount will be refunded
	Amount int64 `json:"amount,omitempty" binding:"omitempty"`
}