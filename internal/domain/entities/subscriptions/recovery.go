package subscriptions

import (
	"payloop/internal/domain/entities"
	"time"
)

type RetryInterval string

const (
	RetryIntervalDay  RetryInterval = "day"
	RetryIntervalWeek RetryInterval = "week"
)

type FailureAction string

const (
	FailureActionCancel       FailureAction = "cancel"
	FailureActionMarkUnpaid   FailureAction = "mark_unpaid"
	FailureActionLeavePastDue FailureAction = "past_due"
)

type RetryPolicy struct {
	RetryAttempts int           `json:"attempts"`
	RetryInterval RetryInterval `json:"interval"`
	RetryPeriod   int           `json:"retry_period"`
	FailureAction FailureAction `json:"failure_action"`
}

type RetryPolicyResponse struct {
	RetryDate time.Time
}

func (r RetryPolicy) GetNextCharge(subscription entities.Subscription) time.Time {
	if subscription.Retries >= r.RetryAttempts {
		return time.Time{}
	}
	var nextCharge time.Time
	base := subscription.RenewsAt

	// for now we just divvy the retry period by the number of retries
	retryPeriod := base.Add(time.Duration(r.RetryPeriod) * 24 * time.Hour)
	retryPeriodLeft := retryPeriod.Sub(time.Now())
	retriesLeft := r.RetryAttempts - subscription.Retries
	retry := float64(retryPeriodLeft) / float64(retriesLeft)

	nextCharge = base.Add(time.Duration(retry))

	return nextCharge
}
