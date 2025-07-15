package models

import (
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/domain/entities"
)

// PriceTier represents a pricing tier model for database operations
type PriceTier struct {
	OrgId       pgtype.Text   `json:"org_id"`
	PriceId     pgtype.Text   `json:"price_id"`
	Tier        pgtype.Int4   `json:"tier"`
	FromQty     pgtype.Int4   `json:"from_qty"`
	ToQty       pgtype.Int4   `json:"to_qty"`
	UnitPrice   pgtype.Int8   `json:"unit_price"`
	Description pgtype.Text   `json:"description"`
	CreatedAt   pgtype.Date   `json:"created_at"`
	UpdatedAt   pgtype.Date   `json:"updated_at"`
}

// ToEntity converts the model to a PriceTier entity
func (p *PriceTier) ToEntity() entities.PriceTier {
	var toQty *int
	if p.ToQty.Valid {
		val := int(p.ToQty.Int32)
		toQty = &val
	}

	return entities.PriceTier{
		OrgId:       p.OrgId.String,
		PriceId:     p.PriceId.String,
		Tier:        int(p.Tier.Int32),
		FromQty:     int(p.FromQty.Int32),
		ToQty:       toQty,
		UnitPrice:   p.UnitPrice.Int64,
		Description: p.Description.String,
		CreatedAt:   p.CreatedAt.Time,
		UpdatedAt:   p.UpdatedAt.Time,
	}
}