package entities

import (
	"encoding/json"
	"time"
)

// DiscountType represents the type of discount (fixed amount or percentage).
type DiscountType string

const (
	DiscountTypeFixed      DiscountType = "fixed"
	DiscountTypePercentage DiscountType = "percentage"
)

// Discount represents a discount that can be applied to subscriptions, invoices, etc.
type Discount struct {
	Id             string          `json:"id"`
	OrgId          string          `json:"org_id"`
	Name           string          `json:"name"`
	Type           DiscountType    `json:"type"`           // DiscountTypeFixed or DiscountTypePercentage
	Value          int             `json:"value"`          // For fixed: amount in smallest currency unit. For percentage: 0-100 (e.g., 20 = 20%)
	Code           string          `json:"code,omitempty"` // Optional discount code
	StartsAt       time.Time       `json:"starts_at,omitempty"`
	EndsAt         time.Time       `json:"ends_at,omitempty"`
	MaxRedemptions int             `json:"max_redemptions,omitempty"` // Maximum number of times this discount can be redeemed
	Recurring      string          `json:"recurring"`                 // "once", "forever", "cycles"
	Cycles         int             `json:"cycles,omitempty"`          // Number of billing cycles when recurring is "cycles"
	Currency       string          `json:"currency,omitempty"`        // Required for fixed discounts
	Active         bool            `json:"active"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
}

// IsValid checks if the discount is valid based on its configuration
func (d Discount) IsValid() bool {
	now := time.Now()

	// Check if discount is active
	if !d.Active {
		return false
	}

	// Check start date if set
	if !d.StartsAt.IsZero() && now.Before(d.StartsAt) {
		return false
	}

	// Check end date if set
	if !d.EndsAt.IsZero() && now.After(d.EndsAt) {
		return false
	}

	return true
}

// IsFixedAmount returns true if the discount is a fixed amount discount
func (d Discount) IsFixedAmount() bool {
	return d.Type == DiscountTypeFixed
}

// IsPercentage returns true if the discount is a percentage discount
func (d Discount) IsPercentage() bool {
	return d.Type == DiscountTypePercentage
}

// CalculateDiscountAmount calculates the discount amount for a given amount
func (d Discount) CalculateDiscountAmount(amount int) int {
	if d.IsFixedAmount() {
		return d.Value
	}

	if d.IsPercentage() {
		return (amount * d.Value) / 100
	}

	return 0
}
