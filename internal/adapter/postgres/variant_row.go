package postgres

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// variantRow is the postgres on-the-wire shape of a Variant. The Prices
// slice is populated via gorm Preload("Prices") at the row level; mapper
// converts to domain.Variant with []domain.Price.
type variantRow struct {
	OrgId       string            `gorm:"column:org_id;primaryKey"`
	Id          string            `gorm:"column:id;primaryKey"`
	ProductId   string            `gorm:"column:product_id"`
	Name        string            `gorm:"column:name"`
	Description string            `gorm:"column:description"`
	Metadata    map[string]string `gorm:"column:metadata;serializer:json"`
	Prices      []priceRow        `gorm:"foreignKey:VariantId,OrgId;references:Id,OrgId"`
	CreatedAt   time.Time         `gorm:"column:created_at"`
	UpdatedAt   time.Time         `gorm:"column:updated_at"`
}

func (variantRow) TableName() string { return "variants" }

func (r variantRow) toDomain() domain.Variant {
	return domain.Variant{
		OrgId:       r.OrgId,
		Id:          r.Id,
		ProductId:   r.ProductId,
		Name:        r.Name,
		Description: r.Description,
		Metadata:    r.Metadata,
		Prices:      priceRowsToDomain(r.Prices),
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

func variantRowFromDomain(v domain.Variant) variantRow {
	prices := make([]priceRow, len(v.Prices))
	for i, p := range v.Prices {
		prices[i] = priceRowFromDomain(p)
	}
	return variantRow{
		OrgId:       v.OrgId,
		Id:          v.Id,
		ProductId:   v.ProductId,
		Name:        v.Name,
		Description: v.Description,
		Metadata:    v.Metadata,
		Prices:      prices,
		CreatedAt:   v.CreatedAt,
		UpdatedAt:   v.UpdatedAt,
	}
}

func variantRowsToDomain(rows []variantRow) []domain.Variant {
	out := make([]domain.Variant, len(rows))
	for i, row := range rows {
		out[i] = row.toDomain()
	}
	return out
}
