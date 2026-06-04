package domain

// ProrationBillingMode determines how proration is calculated for subscription changes.
type ProrationBillingMode string

const (
	ProratedImmediately       ProrationBillingMode = "prorated_immediately"
	ProratedNextBillingPeriod ProrationBillingMode = "prorated_next_billing_period"
	FullImmediately           ProrationBillingMode = "full_immediately"
	FullNextBillingPeriod     ProrationBillingMode = "full_next_billing_period"
	DoNotBill                 ProrationBillingMode = "do_not_bill"
)

// SubscriptionResumeBehavior determines how a paused subscription is resumed.
type SubscriptionResumeBehavior string

const (
	ContinueExistingBillingPeriod SubscriptionResumeBehavior = "continue_existing_billing_period"
	StartNewBillingPeriod         SubscriptionResumeBehavior = "start_new_billing_period"
)

type ProrationMode string

const (
	ProrationModeNone         ProrationMode = "none"
	ProrationModeCreditUnused ProrationMode = "credit_unused"
)

// Subscription API input types

type UpdateSubscriptionRequest struct {
	Id                   string             `json:"id"`
	Status               SubscriptionStatus `json:"status"`
	DefaultPaymentMethod string             `json:"default_payment_method"`
	Metadata             map[string]string  `json:"metadata"`
}

type ProcessSubscriptionChargeInput struct {
	Subscription Subscription `json:"subscription"`
}
