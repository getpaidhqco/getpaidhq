package events

import (
	"payloop/internal/domain/entities"
)

// OrderCompletedEvent represents the payload for order.completed event
type OrderCompletedEvent struct {
	Order   entities.Order   `json:"order"`
	Payment entities.Payment `json:"payment"`
}