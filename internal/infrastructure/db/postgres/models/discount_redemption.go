package models

import (
	"encoding/json"
	"payloop/internal/domain/entities"
	"time"
)

// DiscountRedemption represents a record of a discount being applied to a resource in the database
type DiscountRedemption struct {
	Id             string          `json:"id"`
	OrgId          string          `json:"org_id"`
	DiscountId     string          `json:"discount_id"`
	CustomerId     string          `json:"customer_id"`
	ResourceType   string          `json:"resource_type"`
	ResourceId     string          `json:"resource_id"`
	DiscountAmount int             `json:"discount_amount"`
	Currency       string          `json:"currency"`
	CreatedAt      time.Time       `json:"created_at"`
	Metadata       json.RawMessage `json:"metadata"`
}

// ToEntity converts the model to an entity
func (dr DiscountRedemption) ToEntity() entities.DiscountRedemption {
	redemption := entities.DiscountRedemption{
		Id:             dr.Id,
		OrgId:          dr.OrgId,
		DiscountId:     dr.DiscountId,
		CustomerId:     dr.CustomerId,
		ResourceType:   dr.ResourceType,
		ResourceId:     dr.ResourceId,
		DiscountAmount: dr.DiscountAmount,
		Currency:       dr.Currency,
		CreatedAt:      dr.CreatedAt,
	}

	if len(dr.Metadata) > 0 {
		redemption.Metadata = dr.Metadata
	}

	return redemption
}