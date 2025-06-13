package dto

import (
	"payloop/internal/domain/entities"
	"time"
)

// ProrationMode defines how proration is applied when updating a subscription's billing anchor
type ProrationMode string

const (
	// ProrationModeNone: No proration is applied. The subscription will not be prorated for any changes made.
	// This is the default behavior.
	ProrationModeNone ProrationMode = "none"

	// ProrationModeCreditUnused: Proration is applied by crediting unused time. The subscription will be prorated for any changes made.
	ProrationModeCreditUnused ProrationMode = "credit_unused"
)

// UpdateBillingAnchorInput contains the input parameters for updating a subscription's billing anchor
type UpdateBillingAnchorInput struct {
	OrgId         string        `json:"org_id"`
	Id            string        `json:"id"`
	BillingAnchor int           `json:"billing_anchor"`
	ProrationMode ProrationMode `json:"proration_mode"`
}

// ProrationDetails contains the details of a proration calculation
type ProrationDetails struct {
	CreditAmount       int       `json:"credit_amount"`
	DaysCredited       int       `json:"days_credited"`
	CurrentPeriodStart time.Time `json:"current_period_start"`
	CurrentPeriodEnd   time.Time `json:"current_period_end"`
	OldBillingAnchor   int       `json:"old_billing_anchor,omitempty"`
	NewBillingAnchor   int       `json:"new_billing_anchor,omitempty"`
	NewPeriodStart     time.Time `json:"new_period_start,omitempty"`
	NewPeriodEnd       time.Time `json:"new_period_end,omitempty"`
}

// UpdateBillingAnchorResult contains the updated subscription and proration details
type UpdateBillingAnchorResult struct {
	Subscription     entities.Subscription `json:"subscription"`
	ProrationDetails ProrationDetails      `json:"proration_details"`
}