package response

import (
	"payloop/internal/domain/entities"
	"time"
)

// PriceTier represents a pricing tier in the response
type PriceTier struct {
	Tier        int       `json:"tier"`
	FromQty     int       `json:"from_qty"`
	ToQty       *int      `json:"to_qty"`
	UnitPrice   int64     `json:"unit_price"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// NewPriceTierFromEntity creates a new PriceTier response from an entity
func NewPriceTierFromEntity(entity entities.PriceTier) PriceTier {
	return PriceTier{
		Tier:        entity.Tier,
		FromQty:     entity.FromQty,
		ToQty:       entity.ToQty,
		UnitPrice:   entity.UnitPrice,
		Description: entity.Description,
		CreatedAt:   entity.CreatedAt,
		UpdatedAt:   entity.UpdatedAt,
	}
}

// NewPriceTiersFromEntities creates a slice of PriceTier responses from entities
func NewPriceTiersFromEntities(entities []entities.PriceTier) []PriceTier {
	tiers := make([]PriceTier, len(entities))
	for i, entity := range entities {
		tiers[i] = NewPriceTierFromEntity(entity)
	}
	return tiers
}