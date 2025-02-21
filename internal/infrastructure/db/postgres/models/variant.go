package models

import (
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/domain/entities"
)

type Variant struct {
	OrgId       string            `json:"org_id"`
	Id          string            `json:"id"`
	ProductId   string            `json:"product_id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Metadata    map[string]string `json:"metadata"`
	Prices      []Price           `json:"prices"`
	CreatedAt   pgtype.Date       `json:"created_at"`
	UpdatedAt   pgtype.Date       `json:"updated_at"`
}

func (p *Variant) ToEntity() entities.Variant {

	return entities.Variant{
		OrgId:       p.OrgId,
		Id:          p.Id,
		ProductId:   p.ProductId,
		Name:        p.Name,
		Description: p.Description,
		Metadata:    p.Metadata,
		Prices:      convertPricesToEntities(p.Prices),
		CreatedAt:   p.CreatedAt.Time,
		UpdatedAt:   p.UpdatedAt.Time,
	}
}
func convertPricesToEntities(prices []Price) []entities.Price {
	var entityPrices []entities.Price
	for _, price := range prices {
		entityPrices = append(entityPrices, price.ToEntity())
	}
	return entityPrices
}
