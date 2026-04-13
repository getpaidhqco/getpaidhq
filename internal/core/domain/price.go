package domain

import (
	"payloop/internal/lib"
	"time"
)

type Price struct {
	OrgId              string          `json:"org_id"`
	Id                 string          `json:"id"`
	VariantId          string          `json:"variant_id"`
	Label              string          `json:"label"`
	Category           PriceCategory   `json:"category"`
	Scheme             PriceScheme     `json:"scheme"`
	Cycles             int             `json:"cycles"`
	Currency           Currency        `json:"currency"`
	UnitPrice          int64           `json:"unit_price"`
	MinPrice           int64           `json:"min_price"`
	SuggestedPrice     int64           `json:"suggested_price"`
	BillingInterval    BillingInterval `json:"billing_interval"`
	BillingIntervalQty int             `json:"billing_interval_qty"`
	TrialInterval      BillingInterval `json:"trial_interval"`
	TrialIntervalQty   int             `json:"trial_interval_qty"`
	TaxCode            string          `json:"tax_code"`
	Metadata           map[string]string `json:"metadata"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

func NewPrice(orgId, variantId string, input CreatePriceInput) Price {
	if input.BillingInterval == "" {
		input.BillingInterval = BillingIntervalNone
	}
	if input.TrialInterval == "" {
		input.TrialInterval = BillingIntervalNone
	}

	return Price{
		OrgId:              orgId,
		Id:                 lib.GenerateId("price"),
		Label:              input.Label,
		VariantId:          variantId,
		Category:           input.Category,
		Scheme:             input.Scheme,
		Cycles:             input.Cycles,
		Currency:           Currency(input.Currency),
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

type CreatePriceInput struct {
	OrgId              string          `json:"org_id"`
	Label              string          `json:"label"`
	VariantId          string          `json:"variant_id"`
	Category           PriceCategory   `json:"category"`
	Scheme             PriceScheme     `json:"scheme"`
	Cycles             int             `json:"cycles"`
	Currency           string          `json:"currency"`
	UnitPrice          int64           `json:"unit_price"`
	MinPrice           int64           `json:"min_price"`
	SuggestedPrice     int64           `json:"suggested_price"`
	BillingInterval    BillingInterval `json:"billing_interval"`
	BillingIntervalQty int             `json:"billing_interval_qty"`
	TrialInterval      BillingInterval `json:"trial_interval"`
	TrialIntervalQty   int             `json:"trial_interval_qty"`
	TaxCode            string          `json:"tax_code"`
	Metadata           map[string]string `json:"metadata"`
}
