package models

import (
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/domain/entities"
)

type Product struct {
	OrgId       string            `json:"org_id"`
	Id          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Variants    []Variant         `json:"variants"`
	Metadata    map[string]string `json:"metadata"`
	CreatedAt   pgtype.Date       `json:"created_at"`
	UpdatedAt   pgtype.Date       `json:"updated_at"`
}

func (p *Product) ToEntity() entities.Product {
	var variants []entities.Variant
	for _, variant := range p.Variants {
		variants = append(variants, variant.ToEntity())
	}
	return entities.Product{
		OrgId:       p.OrgId,
		Id:          p.Id,
		Name:        p.Name,
		Description: p.Description,
		Variants:    variants,
		Metadata:    p.Metadata,
		CreatedAt:   p.CreatedAt.Time,
		UpdatedAt:   p.UpdatedAt.Time,
	}
}
