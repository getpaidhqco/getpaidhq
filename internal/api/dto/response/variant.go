package response

import (
	"payloop/internal/domain/entities"
	"time"
)

type Variant struct {
	Id        string    `json:"id"`
	Name      string    `json:"name"`
	Prices    []Price   `json:"prices,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewVariantFromEntity(entity entities.Variant) Variant {
	var prices []Price
	for _, price := range entity.Prices {
		prices = append(prices, NewPriceFromEntity(price))
	}
	return Variant{
		Id:        entity.Id,
		Name:      entity.Name,
		Prices:    prices,
		CreatedAt: entity.CreatedAt,
		UpdatedAt: entity.UpdatedAt,
	}
}
