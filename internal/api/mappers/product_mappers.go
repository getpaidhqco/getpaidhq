package mappers

import (
	"payloop/internal/api/dto/request"
	"payloop/internal/application/dto"
)

// ToCreateProductInput maps an API CreateProductRequest to an application CreateProductInput
func ToCreateProductInput(req request.CreateProductRequest) dto.CreateProductInput {
	variants := make([]dto.CreateProductVariantInput, len(req.Variants))
	for i, v := range req.Variants {
		prices := make([]dto.CreateProductPriceInput, len(v.Prices))
		for j, p := range v.Prices {
			tiers := make([]dto.CreatePriceTierInput, len(p.Tiers))
			for k, t := range p.Tiers {
				// Convert FromQty from int to int64
				fromQty := int64(t.FromQty)

				// Handle ToQty which is a pointer in the API DTO
				var toQty int64
				if t.ToQty != nil {
					toQty = int64(*t.ToQty)
				}

				tiers[k] = dto.CreatePriceTierInput{
					Tier:        t.Tier,
					FromQty:     fromQty,
					ToQty:       toQty,
					UnitPrice:   t.UnitPrice,
					Description: t.Description,
				}
			}

			prices[j] = dto.CreateProductPriceInput{
				Label:              p.Label,
				Category:           p.Category,
				Scheme:             p.Scheme,
				Cycles:             p.Cycles,
				Currency:           p.Currency,
				UnitPrice:          p.UnitPrice,
				MinPrice:           p.MinPrice,
				SuggestedPrice:     p.SuggestedPrice,
				BillingInterval:    p.BillingInterval,
				BillingIntervalQty: p.BillingIntervalQty,
				TrialInterval:      p.TrialInterval,
				TrialIntervalQty:   p.TrialIntervalQty,
				TaxCode:            p.TaxCode,
				HasUsage:           p.HasUsage,
				UsageType:          p.UsageType,
				UnitType:           p.UnitType,
				AggregationType:    p.AggregationType,
				PercentageRate:     p.PercentageRate,
				FixedFee:           p.FixedFee,
				OverageUnitPrice:   p.OverageUnitPrice,
				IncludedUsage:      p.IncludedUsage,
				UsageLimit:         p.UsageLimit,
				Tiers:              tiers,
				Metadata:           p.Metadata,
			}
		}

		variants[i] = dto.CreateProductVariantInput{
			Name:        v.Name,
			Description: v.Description,
			Metadata:    v.Metadata,
			Prices:      prices,
		}
	}

	return dto.CreateProductInput{
		Name:        req.Name,
		Description: req.Description,
		Metadata:    req.Metadata,
		Variants:    variants,
	}
}

// ToUpdateProductInput maps an API UpdateProductRequest to an application UpdateProductInput
func ToUpdateProductInput(req request.UpdateProductRequest) dto.UpdateProductInput {
	return dto.UpdateProductInput{
		Name:        req.Name,
		Description: req.Description,
		Metadata:    req.Metadata,
	}
}
