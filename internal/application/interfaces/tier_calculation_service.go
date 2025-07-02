package interfaces

import (
	"context"
	"payloop/internal/domain/entities"
)

// TierCalculationService handles pricing calculations for tiered pricing models
type TierCalculationService interface {
	// CalculateTieredAmount calculates the amount for a given quantity and price
	CalculateTieredAmount(ctx context.Context, quantity int, price entities.Price) (*TierCalculationResult, error)
}

// TierCalculationResult represents the result of a tier calculation
type TierCalculationResult struct {
	TotalAmount int64           `json:"total_amount"`
	Breakdown   []TierBreakdown `json:"breakdown"`
}

// TierBreakdown represents the breakdown of a tier calculation
type TierBreakdown struct {
	Tier        int    `json:"tier"`
	Description string `json:"description"`
	Quantity    int    `json:"quantity"`
	UnitPrice   int64  `json:"unit_price"`
	Amount      int64  `json:"amount"`
}