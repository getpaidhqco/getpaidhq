package entities

import (
	"encoding/json"
	"time"
)

// DiscountRedemption represents a record of a discount being applied to a resource
type DiscountRedemption struct {
	Id             string          `json:"id"`
	OrgId          string          `json:"org_id"`
	DiscountId     string          `json:"discount_id"`
	CustomerId     string          `json:"customer_id"`
	ResourceType   string          `json:"resource_type"` // "subscription", "invoice", "payment", "checkout_session"
	ResourceId     string          `json:"resource_id"`
	DiscountAmount int             `json:"discount_amount"` // Amount saved in smallest currency unit
	Currency       string          `json:"currency"`
	CreatedAt      time.Time       `json:"created_at"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
}

// GetResourceIdentifier returns a string that uniquely identifies the resource
func (dr DiscountRedemption) GetResourceIdentifier() string {
	return dr.ResourceType + ":" + dr.ResourceId
}

// IsForResource checks if this redemption is for the specified resource
func (dr DiscountRedemption) IsForResource(resourceType, resourceId string) bool {
	return dr.ResourceType == resourceType && dr.ResourceId == resourceId
}

// IsForCustomer checks if this redemption is for the specified customer
func (dr DiscountRedemption) IsForCustomer(customerId string) bool {
	return dr.CustomerId == customerId
}

// IsForDiscount checks if this redemption is for the specified discount
func (dr DiscountRedemption) IsForDiscount(discountId string) bool {
	return dr.DiscountId == discountId
}