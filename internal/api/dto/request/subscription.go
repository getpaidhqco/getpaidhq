package request

import (
	"payloop/internal/application/dto"
	"payloop/internal/domain/entities/subscriptions"
)

type CreateSubscriptionRequest struct {
	CustomerId      string                          `json:"customer_id" binding:"required"`
	PaymentMethodId string                          `json:"payment_method_id"`
	Currency        string                          `json:"currency"  binding:"required"`
	Items           []CreateSubscriptionItemRequest `json:"items" binding:"required,dive"`
	Metadata        map[string]string               `json:"metadata"`
}

// UpdateSubscriptionRequest handles subscription updates
type UpdateSubscriptionRequest struct {
	PaymentMethodId string            `json:"payment_method_id"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

// PauseSubscriptionRequest handles subscription pausing
type PauseSubscriptionRequest struct {
	PauseMode string `json:"pause_mode" binding:"required"`
	ResumeAt  string `json:"resume_at,omitempty"`
	Reason    string `json:"reason"`
}

type UpdateBillingAnchorRequest struct {
	// BillingAnchor the new billing anchor as a day between 1 and 31. If the day is not valid for the current month, it will be adjusted to the last day of the month.
	BillingAnchor int               `json:"billing_anchor" binding:"required,gte=1,lte=31"`
	ProrationMode dto.ProrationMode `json:"proration_mode" binding:"required,oneof=none credit_unused"`
}

// ResumeSubscriptionRequest handles subscription resuming
type ResumeSubscriptionRequest struct {
	ProrationMode  string                                   `json:"proration_mode"`
	ResumeBehavior subscriptions.SubscriptionResumeBehavior `json:"resume_behavior"`
}

// CancelSubscriptionRequest handles subscription cancellation
type CancelSubscriptionRequest struct {
	CancelMode       string `json:"cancel_mode" binding:"required"`
	ProrationMode    string `json:"proration_mode"`
	CancellationDate string `json:"cancellation_date,omitempty"`
	Reason           string `json:"reason"`
}

// ChangePlanRequest handles subscription plan changes
type ChangePlanRequest struct {
	NewVariantId  string `json:"new_variant_id" binding:"required"`
	NewPriceId    string `json:"new_price_id" binding:"required"`
	ProrationMode string `json:"proration_mode" binding:"oneof=none immediate credit_unused"`
	EffectiveDate string `json:"effective_date" binding:"oneof=immediate next_billing_cycle"`
	Reason        string `json:"reason"`
}

// ActivateSubscriptionRequest handles subscription activation
type ActivateSubscriptionRequest struct {
	PaymentMethodId string `json:"payment_method_id,omitempty"`
	Reason string `json:"reason"`
}
