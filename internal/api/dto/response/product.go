package response

import (
	"payloop/internal/domain/entities"
	"time"
)

type Product struct {
	Id          string             `json:"id"`
	Name        string             `json:"name"`
	Description *string            `json:"description"`
	Metadata    *map[string]string `json:"metadata"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

func NewProductFromEntity(entity entities.Product) Product {
	return Product{
		Id:          entity.Id,
		Name:        entity.Name,
		Description: entity.Description,
		Metadata:    entity.Metadata,
		CreatedAt:   entity.CreatedAt,
		UpdatedAt:   entity.UpdatedAt,
	}
}
