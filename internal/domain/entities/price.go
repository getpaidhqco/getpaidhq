package entities

import cart "github.com/mdwt/payloop-cart"

type Price struct {
	AccountId          string             `json:"account_id"`
	Id                 string             `json:"id"`
	Category           string             `json:"category"`
	Scheme             string             `json:"scheme"`
	Currency           string             `json:"currency"`
	UnitPrice          int                `json:"unit_price"`
	MinPrice           *int               `json:"min_price"`
	SuggestedPrice     *int               `json:"suggested_price"`
	BillingInterval    *string            `json:"billing_interval"`
	BillingIntervalQty *int               `json:"billing_interval_qty"`
	TrialInterval      *string            `json:"trial_interval"`
	TrialIntervalQty   *int               `json:"trial_interval_qty"`
	TaxCode            *string            `json:"tax_code"`
	Metadata           *map[string]string `json:"metadata"`
}

func (p Price) ToCartItemPrice() cart.Price {
	if p.MinPrice == nil {
		p.MinPrice = new(int)
	}
	if p.SuggestedPrice == nil {
		p.SuggestedPrice = new(int)
	}
	if p.BillingInterval == nil {
		p.BillingInterval = new(string)
	}
	if p.BillingIntervalQty == nil {
		p.BillingIntervalQty = new(int)
	}
	if p.TrialInterval == nil {
		p.TrialInterval = new(string)
	}
	if p.TrialIntervalQty == nil {
		p.TrialIntervalQty = new(int)
	}
	if p.TaxCode == nil {
		p.TaxCode = new(string)
	}
	return cart.Price{
		Id:                 p.Id,
		Category:           p.Category,
		Scheme:             p.Scheme,
		Currency:           p.Currency,
		UnitPrice:          p.UnitPrice,
		BillingInterval:    *p.BillingInterval,
		BillingIntervalQty: *p.BillingIntervalQty,
		TrialInterval:      *p.TrialInterval,
		TrialIntervalQty:   *p.TrialIntervalQty,
		TaxCode:            *p.TaxCode,
	}
}
