package workflow

import (
	"payloop/internal/domain/entities"
)

type OrderCompletedPayload struct {
	Order   entities.Order   `json:"order"`
	Payment entities.Payment `json:"payment"`
}