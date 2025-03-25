package cart

import (
	"payloop/internal/domain/entities"
	"payloop/internal/infrastructure/cart/types"
)

func PriceToCartItemPrice(p entities.Price) Price {
	return Price{
		Id:                 p.Id,
		Category:           types.PriceCategory(p.Category),
		Scheme:             types.PriceScheme(p.Scheme),
		Currency:           string(p.Currency),
		Cycles:             int64(p.Cycles),
		UnitPrice:          p.UnitPrice,
		BillingInterval:    types.BillingInterval(p.BillingInterval),
		BillingIntervalQty: int64(p.BillingIntervalQty),
		TrialInterval:      types.BillingInterval(p.TrialInterval),
		TrialIntervalQty:   int64(p.TrialIntervalQty),
		TaxCode:            p.TaxCode,
	}
}
