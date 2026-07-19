package port

import (
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/lib/ids"
	"time"
)

// CreatePriceInput is the input for PriceService.Create.
type CreatePriceInput struct {
	OrgId              string
	Label              string
	VariantId          string
	Category           domain.PriceCategory
	Scheme             domain.PriceScheme
	Cycles             int
	Currency           string
	UnitPrice          int64
	UnitCount          int // units UnitPrice buys (fixed scheme); <= 1 = per single unit
	MinPrice           int64
	SuggestedPrice     int64
	BillingInterval    domain.BillingInterval
	BillingIntervalQty int
	TrialInterval      domain.BillingInterval
	TrialIntervalQty   int
	TaxCode            string
	BillableMetricId   string             // set when Category == metered
	Tiers              []domain.PriceTier // rate bands for graduated / volume schemes
	FilterField        string             // metered: scopes the price to one slice of its meter
	FilterValue        string             // the filter value; empty with FilterField set = default charge
	ProrateOnIncrease  bool               // carry-over weighted_sum: prorate mid-period adds
	CreditOnDecrease   bool               // carry-over weighted_sum: credit mid-period removes
	Metadata           map[string]string
}

// ToPrice constructs a domain.Price from the input. Replaces the old domain.NewPrice
// factory, which would have required domain to reference CreatePriceInput.
func (input CreatePriceInput) ToPrice(orgId, variantId string) domain.Price {
	billingInterval := input.BillingInterval
	if billingInterval == "" {
		billingInterval = domain.BillingIntervalNone
	}
	trialInterval := input.TrialInterval
	if trialInterval == "" {
		trialInterval = domain.BillingIntervalNone
	}
	return domain.Price{
		OrgId:              orgId,
		Id:                 ids.Generate("price"),
		Label:              input.Label,
		VariantId:          variantId,
		Category:           input.Category,
		Scheme:             input.Scheme,
		Cycles:             input.Cycles,
		Currency:           domain.Currency(input.Currency),
		UnitPrice:          input.UnitPrice,
		UnitCount:          max(1, input.UnitCount),
		MinPrice:           input.MinPrice,
		SuggestedPrice:     input.SuggestedPrice,
		BillingInterval:    billingInterval,
		BillingIntervalQty: input.BillingIntervalQty,
		TrialInterval:      trialInterval,
		TrialIntervalQty:   input.TrialIntervalQty,
		TaxCode:            input.TaxCode,
		BillableMetricId:   input.BillableMetricId,
		Tiers:              input.Tiers,
		FilterField:        input.FilterField,
		FilterValue:        input.FilterValue,
		ProrateOnIncrease:  input.ProrateOnIncrease,
		CreditOnDecrease:   input.CreditOnDecrease,
		Metadata:           input.Metadata,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
}

// CreateProductPriceInput is a price within a CreateProductVariantInput.
type CreateProductPriceInput struct {
	Label              string
	Category           domain.PriceCategory
	Scheme             domain.PriceScheme
	Cycles             int
	Currency           string
	UnitPrice          int64
	UnitCount          int // units UnitPrice buys (fixed scheme); <= 1 = per single unit
	MinPrice           int64
	SuggestedPrice     int64
	BillingInterval    domain.BillingInterval
	BillingIntervalQty int
	TrialInterval      domain.BillingInterval
	TrialIntervalQty   int
	TaxCode            string
	BillableMetricId   string
	Tiers              []domain.PriceTier
	FilterField        string
	FilterValue        string
	ProrateOnIncrease  bool
	CreditOnDecrease   bool
	Metadata           map[string]string
}
