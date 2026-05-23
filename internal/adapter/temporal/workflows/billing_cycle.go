package workflows

import (
	"time"

	temporalio "go.temporal.io/sdk/temporal"
	temporal "go.temporal.io/sdk/workflow"

	"getpaidhq/internal/adapter/temporal/activities"
	"getpaidhq/internal/core/domain"
)

// BillingCycleWorkflow runs a single charge attempt. Mirrors
// internal/adapter/hatchet/workflows/billing_cycle.go.
//
// Long backoff with effectively unlimited retries while transient gateway
// errors persist. The caller (SubscriptionWorkflow) inspects the returned
// ChargeResult and handles Pending → webhook wait, then delegates to the
// SubscriptionService to apply the result.
func BillingCycleWorkflow(ctx temporal.Context, input BillingCycleInput) (domain.ChargeResult, error) {
	var act *activities.OrderActivities

	actCtx := temporal.WithActivityOptions(ctx, temporal.ActivityOptions{
		StartToCloseTimeout: 60 * time.Second,
		RetryPolicy: &temporalio.RetryPolicy{
			InitialInterval:    2 * time.Minute,
			BackoffCoefficient: 1.2,
			MaximumAttempts:    50,
			MaximumInterval:    10 * time.Minute,
		},
	})

	var result domain.ChargeResult
	if err := temporal.ExecuteActivity(actCtx, act.ChargeCustomerForBillingPeriod, input.Subscription).
		Get(actCtx, &result); err != nil {
		return domain.ChargeResult{}, err
	}
	return result, nil
}
