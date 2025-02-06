package entities

import (
	cart "github.com/mdwt/payloop-cart"
	"github.com/mdwt/payloop-cart/types"
	"payloop/internal/domain/entities/prices"
)

type Price struct {
	OrgId              string                 `json:"org_id"`
	Id                 string                 `json:"id"`
	Category           prices.PriceCategory   `json:"category"`
	Scheme             prices.PriceScheme     `json:"scheme"`
	Currency           string                 `json:"currency"`
	UnitPrice          int                    `json:"unit_price"`
	MinPrice           int                    `json:"min_price"`
	SuggestedPrice     int                    `json:"suggested_price"`
	BillingInterval    prices.BillingInterval `json:"billing_interval"`
	BillingIntervalQty int                    `json:"billing_interval_qty"`
	TrialInterval      prices.BillingInterval `json:"trial_interval"`
	TrialIntervalQty   int                    `json:"trial_interval_qty"`
	TaxCode            *string                `json:"tax_code"`
	Metadata           map[string]string      `json:"metadata"`
}

func (p Price) ToCartItemPrice() cart.Price {
	if p.TaxCode == nil {
		p.TaxCode = new(string)
	}
	
	return cart.Price{
		Id:                 p.Id,
		Category:           types.PriceCategory(p.Category),
		Scheme:             types.PriceScheme(p.Scheme),
		Currency:           p.Currency,
		UnitPrice:          p.UnitPrice,
		BillingInterval:    types.BillingInterval(p.BillingInterval),
		BillingIntervalQty: p.BillingIntervalQty,
		TrialInterval:      types.BillingInterval(p.TrialInterval),
		TrialIntervalQty:   p.TrialIntervalQty,
		TaxCode:            *p.TaxCode,
	}
}
