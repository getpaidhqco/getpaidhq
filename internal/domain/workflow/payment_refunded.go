package workflow

import (
	"payloop/internal/domain/entities"
)

type PaymentRefundedPayload struct {
	refund entities.Refund
}
