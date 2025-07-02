package services

import (
	"context"
	"errors"
	"payloop/internal/application/interfaces"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/prices"
	"payloop/internal/domain/repositories"
)

// TierCalculationService handles pricing calculations for tiered pricing models
type TierCalculationService struct {
	priceRepository repositories.PriceRepository
}

// NewTierCalculationService creates a new TierCalculationService
func NewTierCalculationService(priceRepository repositories.PriceRepository) interfaces.TierCalculationService {
	return &TierCalculationService{
		priceRepository: priceRepository,
	}
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

// CalculateTieredAmount calculates the amount for a given quantity and price
func (s *TierCalculationService) CalculateTieredAmount(ctx context.Context, quantity int, price entities.Price) (*interfaces.TierCalculationResult, error) {
	// Load tiers from database
	tiers, err := s.priceRepository.GetPriceTiers(ctx, price.OrgId, price.Id)
	if err != nil {
		return nil, err
	}

	var result *TierCalculationResult

	switch price.Scheme {
	case prices.Tiered, prices.Graduated:
		result = s.calculateCumulativeTiers(quantity, tiers)
	case prices.Volume:
		result = s.calculateVolumeTiers(quantity, tiers)
	case prices.Fixed:
		result = &TierCalculationResult{
			TotalAmount: price.UnitPrice * int64(quantity),
			Breakdown: []TierBreakdown{{
				Tier:      1,
				Quantity:  quantity,
				UnitPrice: price.UnitPrice,
				Amount:    price.UnitPrice * int64(quantity),
			}},
		}
	default:
		return nil, errors.New("unsupported pricing scheme")
	}

	// Convert internal result to interface result
	interfaceResult := &interfaces.TierCalculationResult{
		TotalAmount: result.TotalAmount,
		Breakdown:   make([]interfaces.TierBreakdown, len(result.Breakdown)),
	}

	for i, breakdown := range result.Breakdown {
		interfaceResult.Breakdown[i] = interfaces.TierBreakdown{
			Tier:        breakdown.Tier,
			Description: breakdown.Description,
			Quantity:    breakdown.Quantity,
			UnitPrice:   breakdown.UnitPrice,
			Amount:      breakdown.Amount,
		}
	}

	return interfaceResult, nil
}

// calculateCumulativeTiers calculates the amount for tiered/graduated pricing
func (s *TierCalculationService) calculateCumulativeTiers(quantity int, tiers []entities.PriceTier) *TierCalculationResult {
	var totalAmount int64
	var breakdown []TierBreakdown
	remainingQty := quantity

	for _, tier := range tiers {
		if remainingQty <= 0 {
			break
		}

		// Calculate tier quantity
		tierQty := remainingQty
		if tier.ToQty != nil && remainingQty > (*tier.ToQty-tier.FromQty+1) {
			tierQty = *tier.ToQty - tier.FromQty + 1
		}

		tierAmount := int64(tierQty) * tier.UnitPrice
		totalAmount += tierAmount

		breakdown = append(breakdown, TierBreakdown{
			Tier:        tier.Tier,
			Description: tier.Description,
			Quantity:    tierQty,
			UnitPrice:   tier.UnitPrice,
			Amount:      tierAmount,
		})

		remainingQty -= tierQty
	}

	return &TierCalculationResult{
		TotalAmount: totalAmount,
		Breakdown:   breakdown,
	}
}

// calculateVolumeTiers calculates the amount for volume pricing
func (s *TierCalculationService) calculateVolumeTiers(quantity int, tiers []entities.PriceTier) *TierCalculationResult {
	// Find the appropriate tier based on total quantity
	var selectedTier *entities.PriceTier

	for i := range tiers {
		tier := tiers[i]
		if quantity >= tier.FromQty && (tier.ToQty == nil || quantity <= *tier.ToQty) {
			selectedTier = &tier
			break
		}
	}

	if selectedTier == nil && len(tiers) > 0 {
		// Fallback to last tier
		selectedTier = &tiers[len(tiers)-1]
	}

	if selectedTier == nil {
		// No tiers defined, return zero
		return &TierCalculationResult{
			TotalAmount: 0,
			Breakdown:   []TierBreakdown{},
		}
	}

	totalAmount := int64(quantity) * selectedTier.UnitPrice

	return &TierCalculationResult{
		TotalAmount: totalAmount,
		Breakdown: []TierBreakdown{{
			Tier:        selectedTier.Tier,
			Description: selectedTier.Description,
			Quantity:    quantity,
			UnitPrice:   selectedTier.UnitPrice,
			Amount:      totalAmount,
		}},
	}
}
