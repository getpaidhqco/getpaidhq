package entities

import (
	"time"
)

// PriceTier represents a pricing tier for tiered, graduated, and volume pricing schemes
type PriceTier struct {
	OrgId       string    `json:"org_id"`
	PriceId     string    `json:"price_id"`
	Tier        int       `json:"tier"`        // Tier order (1, 2, 3...)
	FromQty     int       `json:"from_qty"`    // Starting quantity (inclusive)
	ToQty       *int      `json:"to_qty"`      // Ending quantity (inclusive, null = unlimited)
	UnitPrice   int64     `json:"unit_price"`  // Price per unit in cents
	Description string    `json:"description"` // Human readable description
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreatePriceTierInput represents the input for creating a price tier
type CreatePriceTierInput struct {
	OrgId       string `json:"org_id"`
	PriceId     string `json:"price_id"`
	Tier        int    `json:"tier"`        // Tier order (1, 2, 3...)
	FromQty     int    `json:"from_qty"`    // Starting quantity (inclusive)
	ToQty       *int   `json:"to_qty"`      // Ending quantity (inclusive, null = unlimited)
	UnitPrice   int64  `json:"unit_price"`  // Price per unit in cents
	Description string `json:"description"` // Human readable description
}

// NewPriceTier creates a new PriceTier entity
func NewPriceTier(input CreatePriceTierInput) PriceTier {
	return PriceTier{
		OrgId:       input.OrgId,
		PriceId:     input.PriceId,
		Tier:        input.Tier,
		FromQty:     input.FromQty,
		ToQty:       input.ToQty,
		UnitPrice:   input.UnitPrice,
		Description: input.Description,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
}