package subscriptions

import (
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
