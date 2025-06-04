package entities

import (
	"time"
)

// MetadataStore represents a key-value store for metadata associated with a specific resource
type MetadataStore struct {
	OrgId      string    `json:"org_id"`
	ParentId   string    `json:"parent_id"`   // Direct FK to any entity's ID
	ParentType string    `json:"parent_type"` // "org", "customer", "subscription", "payment", etc.
	Key        string    `json:"key"`         // "clerk_org_id", "stripe_customer_id", etc.
	Value      string    `json:"value"`       // Most metadata is string - keep it simple!
	Namespace  string    `json:"namespace"`   // Optional grouping - "external_ids", "settings", "custom_fields"
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}