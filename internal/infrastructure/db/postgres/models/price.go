package models

import (
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/prices"
)

type Price struct {
	OrgId              string            `json:"org_id"`
	Id                 string            `json:"id"`
	Category           string            `json:"category"`
	Scheme             string            `json:"scheme"`
	Cycles             int               `json:"cycles"`
	Currency           string            `json:"currency"`
	UnitPrice          int               `json:"unit_price"`
	MinPrice           int               `json:"min_price"`
	SuggestedPrice     int               `json:"suggested_price"`
	BillingInterval    string            `json:"billing_interval"`
	BillingIntervalQty int               `json:"billing_interval_qty"`
	TrialInterval      string            `json:"trial_interval"`
	TrialIntervalQty   int               `json:"trial_interval_qty"`
	TaxCode            string            `json:"tax_code"`
	Metadata           map[string]string `json:"metadata"`
	CreatedAt          pgtype.Date       `json:"created_at"`
	UpdatedAt          pgtype.Date       `json:"updated_at"`
}

func (p *Price) ToEntity() entities.Price {
	return entities.Price{
		OrgId:              p.OrgId,
		Id:                 p.Id,
		Category:           prices.PriceCategory(p.Category),
		Scheme:             prices.PriceScheme(p.Scheme),
		Cycles:             p.Cycles,
		Currency:           p.Currency,
		UnitPrice:          p.UnitPrice,
		MinPrice:           p.MinPrice,
		SuggestedPrice:     p.SuggestedPrice,
		BillingInterval:    prices.BillingInterval(p.BillingInterval),
		BillingIntervalQty: p.BillingIntervalQty,
		TrialInterval:      prices.BillingInterval(p.TrialInterval),
		TrialIntervalQty:   p.TrialIntervalQty,
		TaxCode:            p.TaxCode,
		Metadata:           p.Metadata,
		CreatedAt:          p.CreatedAt.Time,
		UpdatedAt:          p.UpdatedAt.Time,
	}
}
