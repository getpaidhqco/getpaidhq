package postgres

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// priceRow is the postgres on-the-wire shape of a Price. Package-internal.
type priceRow struct {
	OrgId              string                 `gorm:"column:org_id;primaryKey"`
	Id                 string                 `gorm:"column:id;primaryKey"`
	VariantId          string                 `gorm:"column:variant_id"`
	Label              string                 `gorm:"column:label"`
	Category           domain.PriceCategory   `gorm:"column:category"`
	Scheme             domain.PriceScheme     `gorm:"column:scheme"`
	Cycles             int                    `gorm:"column:cycles"`
	Currency           domain.Currency        `gorm:"column:currency"`
	UnitPrice          int64                  `gorm:"column:unit_price"`
	MinPrice           int64                  `gorm:"column:min_price"`
	SuggestedPrice     int64                  `gorm:"column:suggested_price"`
	BillingInterval    domain.BillingInterval `gorm:"column:billing_interval"`
	BillingIntervalQty int                    `gorm:"column:billing_interval_qty"`
	TrialInterval      domain.BillingInterval `gorm:"column:trial_interval"`
	TrialIntervalQty   int                    `gorm:"column:trial_interval_qty"`
	TaxCode            string                 `gorm:"column:tax_code"`
	Metadata           map[string]string      `gorm:"column:metadata;serializer:json"`
	CreatedAt          time.Time              `gorm:"column:created_at"`
	UpdatedAt          time.Time              `gorm:"column:updated_at"`
}

func (priceRow) TableName() string { return "prices" }

func (r priceRow) toDomain() domain.Price {
	return domain.Price{
		OrgId:              r.OrgId,
		Id:                 r.Id,
		VariantId:          r.VariantId,
		Label:              r.Label,
		Category:           r.Category,
		Scheme:             r.Scheme,
		Cycles:             r.Cycles,
		Currency:           r.Currency,
		UnitPrice:          r.UnitPrice,
		MinPrice:           r.MinPrice,
		SuggestedPrice:     r.SuggestedPrice,
		BillingInterval:    r.BillingInterval,
		BillingIntervalQty: r.BillingIntervalQty,
		TrialInterval:      r.TrialInterval,
		TrialIntervalQty:   r.TrialIntervalQty,
		TaxCode:            r.TaxCode,
		Metadata:           r.Metadata,
		CreatedAt:          r.CreatedAt,
		UpdatedAt:          r.UpdatedAt,
	}
}

func priceRowFromDomain(p domain.Price) priceRow {
	return priceRow{
		OrgId:              p.OrgId,
		Id:                 p.Id,
		VariantId:          p.VariantId,
		Label:              p.Label,
		Category:           p.Category,
		Scheme:             p.Scheme,
		Cycles:             p.Cycles,
		Currency:           p.Currency,
		UnitPrice:          p.UnitPrice,
		MinPrice:           p.MinPrice,
		SuggestedPrice:     p.SuggestedPrice,
		BillingInterval:    p.BillingInterval,
		BillingIntervalQty: p.BillingIntervalQty,
		TrialInterval:      p.TrialInterval,
		TrialIntervalQty:   p.TrialIntervalQty,
		TaxCode:            p.TaxCode,
		Metadata:           p.Metadata,
		CreatedAt:          p.CreatedAt,
		UpdatedAt:          p.UpdatedAt,
	}
}

func priceRowsToDomain(rows []priceRow) []domain.Price {
	out := make([]domain.Price, len(rows))
	for i, row := range rows {
		out[i] = row.toDomain()
	}
	return out
}
