package entities

import (
	cart "github.com/mdwt/payloop-cart"
	"github.com/mdwt/payloop-cart/types"
	"payloop/internal/domain/common"
	"payloop/internal/domain/entities/prices"
	"payloop/internal/lib"
	"time"
)

type Price struct {
	OrgId              string                 `json:"org_id"`
	Id                 string                 `json:"id"`
	VariantId          string                 `json:"variant_id"`
	Label              string                 `json:"label"`
	Category           prices.PriceCategory   `json:"category"`
	Scheme             prices.PriceScheme     `json:"scheme"`
	Cycles             int                    `json:"cycles"`
	Currency           common.Currency        `json:"currency"`
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

// Factory function to create a Price with default values
func NewPrice(orgId, variantId string, input CreatePriceInput) Price {

	if input.BillingInterval == "" {
		input.BillingInterval = prices.BillingIntervalNone
	}
	if input.TrialInterval == "" {
		input.TrialInterval = prices.BillingIntervalNone
	}

	return Price{
		OrgId:              orgId,
		Id:                 lib.GenerateId("price"),
		Label:              input.Label,
		VariantId:          variantId,
		Category:           input.Category,
		Scheme:             input.Scheme,
		Cycles:             input.Cycles,
		Currency:           common.Currency(input.Currency),
		UnitPrice:          input.UnitPrice,
		MinPrice:           input.MinPrice,
		SuggestedPrice:     input.SuggestedPrice,
		BillingInterval:    input.BillingInterval,
		BillingIntervalQty: input.BillingIntervalQty,
		TrialInterval:      input.TrialInterval,
		TrialIntervalQty:   input.TrialIntervalQty,
		TaxCode:            input.TaxCode,
		Metadata:           input.Metadata,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
}

func (p Price) ToCartItemPrice() cart.Price {
	return cart.Price{
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

type CreatePriceInput struct {
	OrgId              string                 `json:"org_id"`
	Label              string                 `json:"label"`
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
}
