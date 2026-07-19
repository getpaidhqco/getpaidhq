package postgresgorm

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// productRow is the postgres on-the-wire shape of a Product. Variants are NOT
// embedded — composition is a service-layer concern.
type productRow struct {
	OrgId       string               `gorm:"column:org_id;primaryKey"`
	Id          string               `gorm:"column:id;primaryKey"`
	Name        string               `gorm:"column:name"`
	Description string               `gorm:"column:description"`
	Status      domain.ProductStatus `gorm:"column:status"`
	ArchivedAt  *time.Time           `gorm:"column:archived_at"`
	Metadata    map[string]string    `gorm:"column:metadata;serializer:json"`
	CreatedAt   time.Time            `gorm:"column:created_at"`
	UpdatedAt   time.Time            `gorm:"column:updated_at"`
}

func (productRow) TableName() string { return "products" }

func (r productRow) toDomain() domain.Product {
	return domain.Product{
		OrgId:       r.OrgId,
		Id:          r.Id,
		Name:        r.Name,
		Description: r.Description,
		Status:      r.Status,
		ArchivedAt:  r.ArchivedAt,
		Metadata:    r.Metadata,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

func productRowFromDomain(p domain.Product) productRow {
	return productRow{
		OrgId:       p.OrgId,
		Id:          p.Id,
		Name:        p.Name,
		Description: p.Description,
		Status:      p.Status,
		ArchivedAt:  p.ArchivedAt,
		Metadata:    p.Metadata,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

func productRowsToDomain(rows []productRow) []domain.Product {
	out := make([]domain.Product, len(rows))
	for i, row := range rows {
		out[i] = row.toDomain()
	}
	return out
}
