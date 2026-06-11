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
	UnitCount          int                    `gorm:"column:unit_count"`
	MinPrice           int64                  `gorm:"column:min_price"`
	SuggestedPrice     int64                  `gorm:"column:suggested_price"`
	BillingInterval    domain.BillingInterval `gorm:"column:billing_interval"`
	BillingIntervalQty int                    `gorm:"column:billing_interval_qty"`
	TrialInterval      domain.BillingInterval `gorm:"column:trial_interval"`
	TrialIntervalQty   int                    `gorm:"column:trial_interval_qty"`
	TaxCode            string                 `gorm:"column:tax_code"`
	BillableMetricId   string                 `gorm:"column:billable_metric_id"`
	Tiers              []domain.PriceTier     `gorm:"column:tiers;serializer:json"`
	FilterField        string                 `gorm:"column:filter_field"`
	FilterValue        string                 `gorm:"column:filter_value"`
	ProrateOnIncrease  bool                   `gorm:"column:prorate_on_increase"`
	CreditOnDecrease   bool                   `gorm:"column:credit_on_decrease"`
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
		UnitCount:          r.UnitCount,
		MinPrice:           r.MinPrice,
		SuggestedPrice:     r.SuggestedPrice,
		BillingInterval:    r.BillingInterval,
		BillingIntervalQty: r.BillingIntervalQty,
		TrialInterval:      r.TrialInterval,
		TrialIntervalQty:   r.TrialIntervalQty,
		TaxCode:            r.TaxCode,
		BillableMetricId:   r.BillableMetricId,
		Tiers:              r.Tiers,
		FilterField:        r.FilterField,
		FilterValue:        r.FilterValue,
		ProrateOnIncrease:  r.ProrateOnIncrease,
		CreditOnDecrease:   r.CreditOnDecrease,
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
		UnitCount:          p.UnitCount,
		MinPrice:           p.MinPrice,
		SuggestedPrice:     p.SuggestedPrice,
		BillingInterval:    p.BillingInterval,
		BillingIntervalQty: p.BillingIntervalQty,
		TrialInterval:      p.TrialInterval,
		TrialIntervalQty:   p.TrialIntervalQty,
		TaxCode:            p.TaxCode,
		BillableMetricId:   p.BillableMetricId,
		Tiers:              p.Tiers,
		FilterField:        p.FilterField,
		FilterValue:        p.FilterValue,
		ProrateOnIncrease:  p.ProrateOnIncrease,
		CreditOnDecrease:   p.CreditOnDecrease,
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
