package subscriptions

import (
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/payments"
)

type SubscriptionChargeInput struct {
	Subscription entities.Subscription `json:"subscription"`
	ChargeResult payments.ChargeResult `json:"charge_result"`
}

// ProrationBillingMode How we should handle proration calculation for changes made to a subscription or its items. Required when making changes that impact billing.
type ProrationBillingMode string

const (
	// ProratedImmediately:  calculates the prorated amount for the subscription changes based on the current billing cycle, then creates a transaction to collect immediately.
	ProratedImmediately ProrationBillingMode = "prorated_immediately"
	// ProratedNextBillingPeriod:  calculates the prorated amount for the subscription changes based on the current billing cycle, then scheduler them to be billed on the next renewal.
	ProratedNextBillingPeriod ProrationBillingMode = "prorated_next_billing_period"
	// FullImmediately:  does not calculate proration for the subscription changes, creating a transaction to collect for the full amount immediately.
	FullImmediately ProrationBillingMode = "full_immediately"
	// FullNextBillingPeriod:  does not calculate proration for the subscription changes, scheduling for the full amount for the changes to be billed on the next renewal.
	FullNextBillingPeriod ProrationBillingMode = "full_next_billing_period"
	// DoNotBill:  does not bill for the subscription changes.
	DoNotBill ProrationBillingMode = "do_not_bill"
)

// SubscriptionResumeBehavior How we should handle resuming a subscription that has been paused.
type SubscriptionResumeBehavior string

const (
	// When resuming, continue the existing billing period. If the customer resumes before the end date of the existing billing period, there's no immediate charge.
	// If after, an error is returned.
	ContinueExistingBillingPeriod SubscriptionResumeBehavior = "continue_existing_billing_period"

	// When resuming, start a new billing period. The current_billing_period.starts_at date is set to the resume date, and we immediately charge the full amount for the new billing period.
	StartNewBillingPeriod SubscriptionResumeBehavior = "start_new_billing_period"
)
