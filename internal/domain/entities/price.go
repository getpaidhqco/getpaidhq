package entities

import (
	cart "github.com/mdwt/payloop-cart"
	"github.com/mdwt/payloop-cart/types"
	"payloop/internal/domain/entities/prices"
	"time"
)

type Price struct {
	OrgId              string                 `json:"org_id"`
	Id                 string                 `json:"id"`
	VariantId          string                 `json:"variant_id"`
	Category           prices.PriceCategory   `json:"category"`
	Scheme             prices.PriceScheme     `json:"scheme"`
	Cycles             int                    `json:"cycles"`
	Currency           string                 `json:"currency"`
	UnitPrice          int64                  `json:"unit_price"`
	MinPrice           int64                  `json:"min_price"`
	SuggestedPrice     int64                  `json:"suggested_price"`
	BillingInterval    prices.BillingInterval `json:"billing_interval"`
	BillingIntervalQty int                    `json:"billing_interval_qty"`
	TrialInterval      prices.BillingInterval `json:"trial_interval"`
	TrialIntervalQty   int                    `json:"trial_interval_qty"`
	TaxCode            string                 `json:"tax_code"`
	Metadata           map[string]string      `json:"metadata"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

func (p Price) ToCartItemPrice() cart.Price {
	return cart.Price{
		Id:                 p.Id,
		Category:           types.PriceCategory(p.Category),
		Scheme:             types.PriceScheme(p.Scheme),
		Currency:           p.Currency,
		Cycles:             int64(p.Cycles),
		UnitPrice:          p.UnitPrice,
		BillingInterval:    types.BillingInterval(p.BillingInterval),
		BillingIntervalQty: int64(p.BillingIntervalQty),
		TrialInterval:      types.BillingInterval(p.TrialInterval),
		TrialIntervalQty:   int64(p.TrialIntervalQty),
		TaxCode:            p.TaxCode,
	}
}
