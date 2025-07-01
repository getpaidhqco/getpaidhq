package entities

import (
	"payloop/internal/lib"
	"time"
)

// SubscriptionItemStatus represents the status of a subscription item
type SubscriptionItemStatus string

const (
	SubscriptionItemStatusActive    SubscriptionItemStatus = "active"
	SubscriptionItemStatusPaused    SubscriptionItemStatus = "paused"
	SubscriptionItemStatusCancelled SubscriptionItemStatus = "cancelled"
	SubscriptionItemStatusPending   SubscriptionItemStatus = "pending"
)

// SubscriptionItem represents an individual product/service within a subscription
type SubscriptionItem struct {
	OrgId          string                `json:"org_id"`
	Id             string                `json:"id"`
	SubscriptionId string                `json:"subscription_id"`
	Subscription   *Subscription         `json:"-"`
	
	// Product/Price reference
	PriceId        string                `json:"price_id"`
	ProductId      string                `json:"product_id,omitempty"`
	VariantId      string                `json:"variant_id,omitempty"`
	
	// Item details
	Name           string                `json:"name"`
	Description    string                `json:"description,omitempty"`
	Status         SubscriptionItemStatus `json:"status"`
	
	// Quantity for fixed items
	Quantity       int                   `json:"quantity"`
	
	// Billing
	Amount         int64                 `json:"amount,omitempty"`
	Currency       string                `json:"currency"`
	
	// Usage configuration
	HasUsage       bool                  `json:"has_usage"`
	UsageType      string                `json:"usage_type,omitempty"`
	AggregationType string               `json:"aggregation_type,omitempty"`
	
	// Metadata
	Metadata       map[string]string     `json:"metadata,omitempty"`
	CreatedAt      time.Time             `json:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at"`
}

// NewSubscriptionItem creates a new subscription item
func NewSubscriptionItem(orgId, subscriptionId, priceId, name, currency string) SubscriptionItem {
	return SubscriptionItem{
		OrgId:          orgId,
		Id:             lib.GenerateId("si"),
		SubscriptionId: subscriptionId,
		PriceId:        priceId,
		Name:           name,
		Status:         SubscriptionItemStatusActive,
		Quantity:       1,
		Currency:       currency,
		HasUsage:       false,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
}

// SetMetadata merges the existing metadata with the specified values.
func (s *SubscriptionItem) SetMetadata(meta map[string]string) *SubscriptionItem {
	if s.Metadata == nil {
		s.Metadata = make(map[string]string)
	}
	for key, value := range meta {
		s.Metadata[key] = value
	}
	return s
}

// SetPaused sets the subscription item status to paused
func (s *SubscriptionItem) SetPaused() *SubscriptionItem {
	s.Status = SubscriptionItemStatusPaused
	s.UpdatedAt = time.Now().UTC()
	return s
}

// SetActive sets the subscription item status to active
func (s *SubscriptionItem) SetActive() *SubscriptionItem {
	s.Status = SubscriptionItemStatusActive
	s.UpdatedAt = time.Now().UTC()
	return s
}

// SetCancelled sets the subscription item status to cancelled
func (s *SubscriptionItem) SetCancelled() *SubscriptionItem {
	s.Status = SubscriptionItemStatusCancelled
	s.UpdatedAt = time.Now().UTC()
	return s
}

// IsActive checks if the subscription item is active
func (s *SubscriptionItem) IsActive() bool {
	return s.Status == SubscriptionItemStatusActive
}