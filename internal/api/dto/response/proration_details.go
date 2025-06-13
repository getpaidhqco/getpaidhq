package response

import (
	"payloop/internal/application/dto"
	"payloop/internal/domain/entities"
	"time"
)

// ProrationDetails is the response struct for proration details
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

// UpdateBillingAnchorResult is the response struct for update billing anchor result
type UpdateBillingAnchorResult struct {
	Subscription     Subscription     `json:"subscription"`
	ProrationDetails ProrationDetails `json:"proration_details"`
}

// NewProrationDetailsFromEntity creates a new ProrationDetails response from an entity
func NewProrationDetailsFromEntity(details entities.ProrationDetails) ProrationDetails {
	return ProrationDetails{
		CreditAmount:       details.CreditAmount,
		DaysCredited:       details.DaysCredited,
		CurrentPeriodStart: details.CurrentPeriodStart,
		CurrentPeriodEnd:   details.CurrentPeriodEnd,
		OldBillingAnchor:   details.OldBillingAnchor,
		NewBillingAnchor:   details.NewBillingAnchor,
		NewPeriodStart:     details.NewPeriodStart,
		NewPeriodEnd:       details.NewPeriodEnd,
	}
}


// NewProrationDetailsFromDto creates a new ProrationDetails response from a DTO
func NewProrationDetailsFromDto(details dto.ProrationDetails) ProrationDetails {
	return ProrationDetails{
		CreditAmount:       details.CreditAmount,
		DaysCredited:       details.DaysCredited,
		CurrentPeriodStart: details.CurrentPeriodStart,
		CurrentPeriodEnd:   details.CurrentPeriodEnd,
		OldBillingAnchor:   details.OldBillingAnchor,
		NewBillingAnchor:   details.NewBillingAnchor,
		NewPeriodStart:     details.NewPeriodStart,
		NewPeriodEnd:       details.NewPeriodEnd,
	}
}

// NewUpdateBillingAnchorResultFromDto creates a new UpdateBillingAnchorResult response from a DTO
func NewUpdateBillingAnchorResultFromDto(result dto.UpdateBillingAnchorResult) UpdateBillingAnchorResult {
	return UpdateBillingAnchorResult{
		Subscription:     NewSubscriptionFromEntity(result.Subscription),
		ProrationDetails: NewProrationDetailsFromDto(result.ProrationDetails),
	}
}
