package interfaces

import (
	"context"
	"payloop/internal/domain/entities"
	"time"
)

type BillingService interface {
	// Main billing calculation method
	CalculateBillingAmount(ctx context.Context, subscription entities.Subscription) (BillingCalculation, error)

	// Pricing model specific calculations
	CalculateTraditionalAmount(ctx context.Context, subscription entities.Subscription) (int64, error)
	CalculateUsageAmount(ctx context.Context, subscription entities.Subscription, period BillingPeriod) (int64, error)
	CalculateHybridAmount(ctx context.Context, subscription entities.Subscription, period BillingPeriod) (int64, error)

	// Billing adjustments
	CalculateProrationAdjustments(ctx context.Context, subscription entities.Subscription) (int64, error)
}

type BillingCalculation struct {
	BaseAmount      int64                    `json:"base_amount"`
	UsageAmount     int64                    `json:"usage_amount"`
	ProrationAmount int64                    `json:"proration_amount"`
	TotalAmount     int64                    `json:"total_amount"`
	Currency        string                   `json:"currency"`
	ItemBreakdown   []BillingItemBreakdown   `json:"item_breakdown"`
	UsageBreakdown  []UsageCalculationResult `json:"usage_breakdown"`
}

type BillingItemBreakdown struct {
	SubscriptionItemId string `json:"subscription_item_id"`
	Description        string `json:"description"`
	PriceCategory      string `json:"price_category"`
	Amount             int64  `json:"amount"`
}

type BillingPeriod struct {
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}

type UsageCalculationResult struct {
	SubscriptionItemId string  `json:"subscription_item_id"`
	UnitType           string  `json:"unit_type"`
	Quantity           float64 `json:"quantity"`
	UnitPrice          int64   `json:"unit_price"`
	Amount             int64   `json:"amount"`
	AggregationType    string  `json:"aggregation_type"`
}
