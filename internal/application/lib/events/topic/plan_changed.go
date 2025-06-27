package topic

import (
	"payloop/internal/domain/entities"
	"time"
)

// SubscriptionPlanChangedEvent represents the event data for a subscription plan change
type SubscriptionPlanChangedEvent struct {
	SubscriptionId  string    `json:"subscription_id"`
	CustomerId      string    `json:"customer_id"`
	FromPlan        PlanInfo  `json:"from_plan"`
	ToPlan          PlanInfo  `json:"to_plan"`
	EffectiveDate   time.Time `json:"effective_date"`
	ProrationAmount int64     `json:"proration_amount"`
	ChangeType      string    `json:"change_type"`
	Timestamp       time.Time `json:"timestamp"`
}

// PlanInfo represents the details of a subscription plan
type PlanInfo struct {
	ProductId string `json:"product_id"`
	VariantId string `json:"variant_id"`
	PriceId   string `json:"price_id"`
	Amount    int64  `json:"amount"`
}

// NewSubscriptionPlanChangedEvent creates a new event for a subscription plan change
func NewSubscriptionPlanChangedEvent(
	subscription entities.Subscription,
	planChange entities.SubscriptionPlanChange,
) SubscriptionPlanChangedEvent {
	return SubscriptionPlanChangedEvent{
		SubscriptionId:  subscription.Id,
		CustomerId:      subscription.CustomerId,
		FromPlan: PlanInfo{
			ProductId: planChange.FromProductId,
			VariantId: planChange.FromVariantId,
			PriceId:   planChange.FromPriceId,
			Amount:    planChange.FromAmount,
		},
		ToPlan: PlanInfo{
			ProductId: planChange.ToProductId,
			VariantId: planChange.ToVariantId,
			PriceId:   planChange.ToPriceId,
			Amount:    planChange.ToAmount,
		},
		EffectiveDate:   planChange.EffectiveDate,
		ProrationAmount: planChange.ProrationAmount,
		ChangeType:      planChange.ChangeType,
		Timestamp:       time.Now().UTC(),
	}
}