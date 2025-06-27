package models

import (
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/domain/entities"
)

// SubscriptionPlanChange represents a change in a subscription's plan
type SubscriptionPlanChange struct {
	Id              string      `json:"id"`
	OrgId           string      `json:"org_id"`
	SubscriptionId  string      `json:"subscription_id"`

	// Change details
	FromProductId   string      `json:"from_product_id"`
	FromVariantId   string      `json:"from_variant_id"`
	FromPriceId     string      `json:"from_price_id"`
	FromAmount      int64       `json:"from_amount"`

	ToProductId     string      `json:"to_product_id"`
	ToVariantId     string      `json:"to_variant_id"`
	ToPriceId       string      `json:"to_price_id"`
	ToAmount        int64       `json:"to_amount"`

	// Metadata
	ChangeType      string      `json:"change_type"` // "upgrade", "downgrade", "switch"
	EffectiveDate   pgtype.Date `json:"effective_date"`
	ProrationMode   string      `json:"proration_mode"`
	ProrationAmount int64       `json:"proration_amount"`
	Reason          pgtype.Text `json:"reason"`
	InitiatedBy     string      `json:"initiated_by"` // "customer", "admin", "system"
	Metadata        map[string]string `json:"metadata"`
	CreatedAt       pgtype.Date `json:"created_at"`
}

// ToEntity converts the model to an entity
func (s *SubscriptionPlanChange) ToEntity() entities.SubscriptionPlanChange {

	return entities.SubscriptionPlanChange{
		Id:              s.Id,
		OrgId:           s.OrgId,
		SubscriptionId:  s.SubscriptionId,
		FromProductId:   s.FromProductId,
		FromVariantId:   s.FromVariantId,
		FromPriceId:     s.FromPriceId,
		FromAmount:      s.FromAmount,
		ToProductId:     s.ToProductId,
		ToVariantId:     s.ToVariantId,
		ToPriceId:       s.ToPriceId,
		ToAmount:        s.ToAmount,
		ChangeType:      s.ChangeType,
		EffectiveDate:   s.EffectiveDate.Time,
		ProrationMode:   s.ProrationMode,
		ProrationAmount: s.ProrationAmount,
		Reason:          s.Reason.String,
		InitiatedBy:     s.InitiatedBy,
		Metadata:        s.Metadata,
		CreatedAt:       s.CreatedAt.Time,
	}
}
