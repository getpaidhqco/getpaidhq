package models

import (
	"encoding/json"
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/domain/entities"
	"time"
)

// Discount represents a discount in the database
type Discount struct {
	Id             string          `json:"id"`
	OrgId          string          `json:"org_id"`
	Name           string          `json:"name"`
	Type           string          `json:"type"`
	Value          int             `json:"value"`
	Code           pgtype.Text     `json:"code"`
	StartsAt       pgtype.Timestamptz `json:"starts_at"`
	EndsAt         pgtype.Timestamptz `json:"ends_at"`
	MaxRedemptions int             `json:"max_redemptions"`
	Recurring      string          `json:"recurring"`
	Cycles         int             `json:"cycles"`
	Currency       string          `json:"currency"`
	Active         bool            `json:"active"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	Metadata       json.RawMessage `json:"metadata"`
}

// ToEntity converts the model to an entity
func (d Discount) ToEntity() entities.Discount {
	discount := entities.Discount{
		Id:             d.Id,
		OrgId:          d.OrgId,
		Name:           d.Name,
		Type:           entities.DiscountType(d.Type),
		Value:          d.Value,
		MaxRedemptions: d.MaxRedemptions,
		Recurring:      d.Recurring,
		Cycles:         d.Cycles,
		Currency:       d.Currency,
		Active:         d.Active,
		CreatedAt:      d.CreatedAt,
		UpdatedAt:      d.UpdatedAt,
	}

	if d.Code.Valid {
		discount.Code = d.Code.String
	}

	if d.StartsAt.Valid {
		discount.StartsAt = d.StartsAt.Time
	}

	if d.EndsAt.Valid {
		discount.EndsAt = d.EndsAt.Time
	}

	if len(d.Metadata) > 0 {
		discount.Metadata = d.Metadata
	}

	return discount
}
