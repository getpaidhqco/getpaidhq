package dto

import (
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/subscriptions"
)

// CreateSubscriptionItemInput represents input for creating a subscription item
type CreateSubscriptionItemInput struct {
	PriceId     string            `json:"price_id" jsonschema:"required,description=Price ID to use"`
	Name        string            `json:"name" jsonschema:"required,description=Name of the subscription item"`
	Description string            `json:"description,omitempty" jsonschema:"description=Description of the subscription item"`
	Quantity    int               `json:"quantity" jsonschema:"minimum=1,description=Quantity of the subscription item"`
	Metadata    map[string]string `json:"metadata,omitempty" jsonschema:"description=Additional metadata as key-value pairs"`
}

// CreateSubscriptionInput represents input for creating a subscription
type CreateSubscriptionInput struct {
	CustomerId      string                       `json:"customer_id" jsonschema:"required,description=Customer ID for the subscription"`
	PaymentMethodId string                       `json:"payment_method_id,omitempty" jsonschema:"description=Payment method ID (optional, will use default if not provided)"`
	Currency        string                       `json:"currency" jsonschema:"required,description=Currency for the subscription"`
	Items           []CreateSubscriptionItemInput `json:"items" jsonschema:"required,description=Items to include in the subscription"`
	Metadata        map[string]string            `json:"metadata,omitempty" jsonschema:"description=Additional metadata as key-value pairs"`
}

// UpdateSubscriptionInput represents input for updating a subscription
type UpdateSubscriptionInput struct {
	SubscriptionId       string                      `json:"subscription_id" jsonschema:"required,description=Subscription ID to update"`
	Status               entities.SubscriptionStatus `json:"status,omitempty" jsonschema:"enum=trial,enum=active,enum=past_due,enum=non_renewing,enum=paused,enum=unpaid,enum=cancelled,enum=pending,enum=expired,enum=completed,enum=error,description=Subscription status"`
	DefaultPaymentMethod string                      `json:"default_payment_method,omitempty" jsonschema:"description=Default payment method ID"`
	Metadata             map[string]string           `json:"metadata,omitempty" jsonschema:"description=Additional metadata as key-value pairs"`
}

// PauseSubscriptionInput represents input for pausing a subscription
type PauseSubscriptionInput struct {
	SubscriptionId string `json:"subscription_id" jsonschema:"required,description=Subscription ID to pause"`
	Reason         string `json:"reason,omitempty" jsonschema:"description=Reason for pausing the subscription"`
}

// ResumeSubscriptionInput represents input for resuming a subscription
type ResumeSubscriptionInput struct {
	SubscriptionId string                                   `json:"subscription_id" jsonschema:"required,description=Subscription ID to resume"`
	ResumeBehavior subscriptions.SubscriptionResumeBehavior `json:"resume_behavior,omitempty" jsonschema:"enum=immediate,enum=next_billing_cycle,description=When to resume the subscription"`
}

// CancelSubscriptionInput represents input for cancelling a subscription
type CancelSubscriptionInput struct {
	SubscriptionId string `json:"subscription_id" jsonschema:"required,description=Subscription ID to cancel"`
	Reason         string `json:"reason,omitempty" jsonschema:"description=Reason for cancelling the subscription"`
	Immediate      bool   `json:"immediate,omitempty" jsonschema:"description=Whether to cancel immediately or at period end"`
}

// ChangePlanInput represents input for changing a subscription's plan
type ChangePlanInput struct {
	SubscriptionId string `json:"subscription_id" jsonschema:"required,description=Subscription ID to change"`
	NewVariantId   string `json:"new_variant_id" jsonschema:"required,description=New product variant ID"`
	NewPriceId     string `json:"new_price_id" jsonschema:"required,description=New price ID"`
	ProrationMode  string `json:"proration_mode,omitempty" jsonschema:"enum=none,enum=immediate,enum=credit_unused,description=How to handle proration"`
	EffectiveDate  string `json:"effective_date,omitempty" jsonschema:"enum=immediate,enum=next_billing_cycle,description=When to apply the change"`
	Reason         string `json:"reason,omitempty" jsonschema:"description=Reason for the plan change"`
}

// SubscriptionListFilters represents filters for listing subscriptions
type SubscriptionListFilters struct {
	Page       int    `json:"page,omitempty" jsonschema:"minimum=1,description=Page number for pagination (default: 1)"`
	Limit      int    `json:"limit,omitempty" jsonschema:"minimum=1,maximum=100,description=Number of items per page (default: 20, max: 100)"`
	Status     string `json:"status,omitempty" jsonschema:"enum=trial,enum=active,enum=past_due,enum=non_renewing,enum=paused,enum=unpaid,enum=cancelled,enum=pending,enum=expired,enum=completed,enum=error,description=Filter by subscription status"`
	CustomerId string `json:"customer_id,omitempty" jsonschema:"description=Filter by customer ID"`
	SortBy     string `json:"sort_by,omitempty" jsonschema:"enum=created_at,enum=updated_at,enum=next_billing_date,description=Field to sort by"`
	SortDir    string `json:"sort_direction,omitempty" jsonschema:"enum=asc,enum=desc,description=Sort direction"`
}
