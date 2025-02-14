package models

import "payloop/internal/domain/entities"

// A subscription model
// swagger:model subscription
type Subscription struct {
	// the ID of the subscription
	// required: true
	ID int64 `json:"id"`

	// the customer that owns the subscription
	// required: true
	CustomerId string `json:"customer_id"`

	// the customer that owns the subscription
	// required: true
	Status          entities.SubscriptionStatus `json:"status"`
	PaymentMethodId string                      `json:"payment_method_id,omitempty"`
}

// An SubscriptionResponse response model. This is used for returning a response with a single order as body.
//
// swagger:response subscriptionResponse
type SubscriptionResponse struct {
	// in: body
	Payload *Subscription `json:"subscription"`
}
