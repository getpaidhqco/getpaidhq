package models

import (
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/domain/entities"
)

type Variant struct {
	OrgId       pgtype.Text       `json:"org_id"`
	Id          pgtype.Text       `json:"id"`
	ProductId   pgtype.Text       `json:"product_id"`
	Name        pgtype.Text       `json:"name"`
	Description pgtype.Text       `json:"description"`
	Metadata    map[string]string `json:"metadata"`
	Prices      []Price           `json:"prices"`
	CreatedAt   pgtype.Date       `json:"created_at"`
	UpdatedAt   pgtype.Date       `json:"updated_at"`
}

func (p *Variant) ToEntity() entities.Variant {

	return entities.Variant{
		OrgId:       p.OrgId.String,
		Id:          p.Id.String,
		ProductId:   p.ProductId.String,
		Name:        p.Name.String,
		Description: p.Description.String,
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
