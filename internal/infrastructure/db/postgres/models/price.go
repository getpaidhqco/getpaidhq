package models

import (
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/domain/common"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/prices"
)

type Price struct {
	OrgId              pgtype.Text       `json:"org_id"`
	Id                 pgtype.Text       `json:"id"`
	VariantId          pgtype.Text       `json:"variant_id"`
	Category           pgtype.Text       `json:"category"`
	Scheme             pgtype.Text       `json:"scheme"`
	Cycles             pgtype.Int8       `json:"cycles"`
	Currency           pgtype.Text       `json:"currency"`
	UnitPrice          pgtype.Int8       `json:"unit_price"`
	MinPrice           pgtype.Int8       `json:"min_price"`
	SuggestedPrice     pgtype.Int8       `json:"suggested_price"`
	BillingInterval    pgtype.Text       `json:"billing_interval"`
	BillingIntervalQty pgtype.Int8       `json:"billing_interval_qty"`
	TrialInterval      pgtype.Text       `json:"trial_interval"`
	TrialIntervalQty   pgtype.Int8       `json:"trial_interval_qty"`
	TaxCode            pgtype.Text       `json:"tax_code"`
	Metadata           map[string]string `json:"metadata"`
	CreatedAt          pgtype.Date       `json:"created_at"`
	UpdatedAt          pgtype.Date       `json:"updated_at"`
}

func (p *Price) ToEntity() entities.Price {
	return entities.Price{
		OrgId:              p.OrgId.String,
		Id:                 p.Id.String,
		VariantId:          p.VariantId.String,
		Category:           prices.PriceCategory(p.Category.String),
		Scheme:             prices.PriceScheme(p.Scheme.String),
		Cycles:             int(p.Cycles.Int64),
		Currency:           common.Currency(p.Currency.String),
		UnitPrice:          p.UnitPrice.Int64,
		MinPrice:           p.MinPrice.Int64,
		SuggestedPrice:     p.SuggestedPrice.Int64,
		BillingInterval:    prices.BillingInterval(p.BillingInterval.String),
		BillingIntervalQty: int(p.BillingIntervalQty.Int64),
		TrialInterval:      prices.BillingInterval(p.TrialInterval.String),
		TrialIntervalQty:   int(p.TrialIntervalQty.Int64),
		TaxCode:            p.TaxCode.String,
		Metadata:           p.Metadata,
		CreatedAt:          p.CreatedAt.Time,
		UpdatedAt:          p.UpdatedAt.Time,
	}
}
