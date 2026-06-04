package postgres

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// productRow is the postgres on-the-wire shape of a Product. The Variants
// slice is populated via gorm Preload("Variants.Prices") at the row level.
type productRow struct {
	OrgId       string            `gorm:"column:org_id;primaryKey"`
	Id          string            `gorm:"column:id;primaryKey"`
	Name        string            `gorm:"column:name"`
	Description string            `gorm:"column:description"`
	Variants    []variantRow      `gorm:"foreignKey:ProductId,OrgId;references:Id,OrgId"`
	Metadata    map[string]string `gorm:"column:metadata;serializer:json"`
	CreatedAt   time.Time         `gorm:"column:created_at"`
	UpdatedAt   time.Time         `gorm:"column:updated_at"`
}

func (productRow) TableName() string { return "products" }

func (r productRow) toDomain() domain.Product {
	return domain.Product{
		OrgId:       r.OrgId,
		Id:          r.Id,
		Name:        r.Name,
		Description: r.Description,
		Variants:    variantRowsToDomain(r.Variants),
		Metadata:    r.Metadata,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

func productRowFromDomain(p domain.Product) productRow {
	variants := make([]variantRow, len(p.Variants))
	for i, v := range p.Variants {
		variants[i] = variantRowFromDomain(v)
	}
	return productRow{
		OrgId:       p.OrgId,
		Id:          p.Id,
		Name:        p.Name,
		Description: p.Description,
		Variants:    variants,
		Metadata:    p.Metadata,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}
