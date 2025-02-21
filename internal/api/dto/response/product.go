package response

import (
	"payloop/internal/domain/entities"
	"time"
)

type Product struct {
	Id          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Variants    []Variant         `json:"variants,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

func NewProductFromEntity(entity entities.Product) Product {
	var variants []Variant
	for _, variant := range entity.Variants {
		variants = append(variants, NewVariantFromEntity(variant))
	}
	return Product{
		Id:          entity.Id,
		Name:        entity.Name,
		Description: entity.Description,
		Variants:    variants,
		Metadata:    entity.Metadata,
		CreatedAt:   entity.CreatedAt,
		UpdatedAt:   entity.UpdatedAt,
	}
}
