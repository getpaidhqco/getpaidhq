package domain

import (
	"payloop/internal/lib"
	"time"
)

type RetryInterval string

const (
	RetryIntervalMinute RetryInterval = "minute"
	RetryIntervalHour   RetryInterval = "hour"
	RetryIntervalDay    RetryInterval = "day"
	RetryIntervalWeek   RetryInterval = "week"
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

func (r RetryPolicy) GetNextCharge(subscription Subscription) time.Time {
	logger := lib.GetLogger()

	if subscription.Retries >= r.RetryAttempts {
		return time.Time{}
	}
	var nextCharge time.Time
	base := subscription.RenewsAt
	logger.Debug("calculating next retry charge", "subscriptionId", subscription.Id)
	logger.Debug("retries attempted", "retries", subscription.Retries)
	logger.Debug("base time", "base", base)

	var retryDuration time.Duration
	switch r.RetryInterval {
	case RetryIntervalMinute:
		retryDuration = time.Minute
	case RetryIntervalHour:
		retryDuration = time.Hour
	case RetryIntervalDay:
		retryDuration = 24 * time.Hour
	case RetryIntervalWeek:
		retryDuration = 7 * 24 * time.Hour
	default:
		retryDuration = 24 * time.Hour
	}

	retryUntil := base.Add(time.Duration(r.RetryPeriod) * retryDuration)
	logger.Debug("retry until", "retryUntil", retryUntil)

	retryPeriod := time.Duration(r.RetryPeriod) * retryDuration / time.Duration(r.RetryAttempts-subscription.Retries)

	nextCharge = base.Add(retryPeriod)
	logger.Debug("charge date for retry", "retryNumber", subscription.Retries+1, "nextCharge", nextCharge)

	return nextCharge
}
