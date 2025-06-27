package entities

import (
	"time"
)

// SubscriptionPlanChange represents a change in a subscription's plan
type SubscriptionPlanChange struct {
	Id              string    `json:"id"`
	OrgId           string    `json:"org_id"`
	SubscriptionId  string    `json:"subscription_id"`

	// Change details
	FromProductId   string    `json:"from_product_id"`
	FromVariantId   string    `json:"from_variant_id"`
	FromPriceId     string    `json:"from_price_id"`
	FromAmount      int64     `json:"from_amount"`

	ToProductId     string    `json:"to_product_id"`
	ToVariantId     string    `json:"to_variant_id"`
	ToPriceId       string    `json:"to_price_id"`
	ToAmount        int64     `json:"to_amount"`

	// Metadata
	ChangeType      string    `json:"change_type"` // "upgrade", "downgrade", "switch"
	EffectiveDate   time.Time `json:"effective_date"`
	ProrationMode   string    `json:"proration_mode"`
	ProrationAmount int64     `json:"proration_amount"`
	Reason          string    `json:"reason"`
	InitiatedBy     string    `json:"initiated_by"` // "customer", "admin", "system"
	Metadata        map[string]string `json:"metadata"`
	CreatedAt       time.Time `json:"created_at"`
}

// PlanDetails represents the details of a subscription plan for event publishing
type PlanDetails struct {
	ProductId string `json:"product_id"`
	VariantId string `json:"variant_id"`
	PriceId   string `json:"price_id"`
	Amount    int64  `json:"amount"`
}

// SubscriptionPlanChangedEvent represents an event that is emitted when a subscription plan is changed
type SubscriptionPlanChangedEvent struct {
	SubscriptionId  string      `json:"subscription_id"`
	CustomerId      string      `json:"customer_id"`
	FromPlan        PlanDetails `json:"from_plan"`
	ToPlan          PlanDetails `json:"to_plan"`
	EffectiveDate   time.Time   `json:"effective_date"`
	ProrationAmount int64       `json:"proration_amount"`
}