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
	Metadata    map[string]string `json:"metadata"`
	CreatedAt   pgtype.Date       `json:"created_at"`
	UpdatedAt   pgtype.Date       `json:"updated_at"`
}

func (p *Product) ToEntity() entities.Product {
	return entities.Product{
		OrgId:       p.OrgId,
		Id:          p.Id,
		Name:        p.Name,
		Description: p.Description,
		Metadata:    p.Metadata,
		CreatedAt:   p.CreatedAt.Time,
		UpdatedAt:   p.UpdatedAt.Time,
	}
}
