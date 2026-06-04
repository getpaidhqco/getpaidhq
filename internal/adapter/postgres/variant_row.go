package postgres

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// variantRow is the postgres on-the-wire shape of a Variant. Prices are NOT
// embedded — composition is a service-layer concern.
type variantRow struct {
	OrgId       string            `gorm:"column:org_id;primaryKey"`
	Id          string            `gorm:"column:id;primaryKey"`
	ProductId   string            `gorm:"column:product_id"`
	Name        string            `gorm:"column:name"`
	Description string            `gorm:"column:description"`
	Metadata    map[string]string `gorm:"column:metadata;serializer:json"`
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
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

func variantRowFromDomain(v domain.Variant) variantRow {
	return variantRow{
		OrgId:       v.OrgId,
		Id:          v.Id,
		ProductId:   v.ProductId,
		Name:        v.Name,
		Description: v.Description,
		Metadata:    v.Metadata,
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
