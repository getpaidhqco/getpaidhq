package models

import (
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/prices"
)

type Price struct {
	OrgId              string            `json:"org_id"`
	Id                 string            `json:"id"`
	VariantId          string            `json:"variant_id"`
	Category           string            `json:"category"`
	Scheme             string            `json:"scheme"`
	Cycles             int               `json:"cycles"`
	Currency           string            `json:"currency"`
	UnitPrice          int               `json:"unit_price"`
	MinPrice           pgtype.Int8       `json:"min_price"`
	SuggestedPrice     pgtype.Int8       `json:"suggested_price"`
	BillingInterval    string            `json:"billing_interval"`
	BillingIntervalQty int               `json:"billing_interval_qty"`
	TrialInterval      string            `json:"trial_interval"`
	TrialIntervalQty   int               `json:"trial_interval_qty"`
	TaxCode            pgtype.Text       `json:"tax_code"`
	Metadata           map[string]string `json:"metadata"`
	CreatedAt          pgtype.Date       `json:"created_at"`
	UpdatedAt          pgtype.Date       `json:"updated_at"`
}

func (p *Price) ToEntity() entities.Price {
	return entities.Price{
		OrgId:              p.OrgId,
		Id:                 p.Id,
		VariantId:          p.VariantId,
		Category:           prices.PriceCategory(p.Category),
		Scheme:             prices.PriceScheme(p.Scheme),
		Cycles:             p.Cycles,
		Currency:           p.Currency,
		UnitPrice:          p.UnitPrice,
		MinPrice:           int(p.MinPrice.Int64),
		SuggestedPrice:     int(p.SuggestedPrice.Int64),
		BillingInterval:    prices.BillingInterval(p.BillingInterval),
		BillingIntervalQty: p.BillingIntervalQty,
		TrialInterval:      prices.BillingInterval(p.TrialInterval),
		TrialIntervalQty:   p.TrialIntervalQty,
		TaxCode:            p.TaxCode.String,
		Metadata:           p.Metadata,
		CreatedAt:          p.CreatedAt.Time,
		UpdatedAt:          p.UpdatedAt.Time,
	}
}
