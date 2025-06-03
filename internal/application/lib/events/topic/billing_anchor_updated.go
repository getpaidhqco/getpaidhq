package topic

import "time"

// ProrationDetails contains the calculated proration information
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

// SubscriptionBillingAnchorChangedData contains the event-specific data
type SubscriptionBillingAnchorChangedData struct {
	OldBillingAnchor int               `json:"old_billing_anchor"`
	NewBillingAnchor int               `json:"new_billing_anchor"`
	ProrationMode    string            `json:"proration_mode"`
	ProrationDetails *ProrationDetails `json:"proration_details,omitempty"`
	EffectiveDate    time.Time         `json:"effective_date"`
}

// SubscriptionBillingAnchorChangedEvent represents the complete event structure
type SubscriptionBillingAnchorChangedEvent struct {
	Event          string                               `json:"event"`
	SubscriptionID string                               `json:"subscription_id"`
	OrgID          string                               `json:"org_id"`
	Data           SubscriptionBillingAnchorChangedData `json:"data"`
	CreatedAt      time.Time                            `json:"created_at"`
}

// NewSubscriptionBillingAnchorChangedEvent creates a new billing anchor changed event
func NewSubscriptionBillingAnchorChangedEvent(
	subscriptionID,
	orgID string,
	oldAnchor,
	newAnchor int,
	prorationMode string,
	prorationDetails *ProrationDetails,
) *SubscriptionBillingAnchorChangedEvent {
	return &SubscriptionBillingAnchorChangedEvent{
		Event:          "subscription.billing_anchor_changed",
		SubscriptionID: subscriptionID,
		OrgID:          orgID,
		Data: SubscriptionBillingAnchorChangedData{
			OldBillingAnchor: oldAnchor,
			NewBillingAnchor: newAnchor,
			ProrationMode:    prorationMode,
			ProrationDetails: prorationDetails,
			EffectiveDate:    time.Now(),
		},
		CreatedAt: time.Now(),
	}
}
