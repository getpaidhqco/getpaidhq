package subscriptions

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
	RetryAttempts    int           `json:"attempts"`
	RetryInterval    RetryInterval `json:"interval"`
	RetryIntervalQty int           `json:"interval_qty"`
	FailureAction    FailureAction `json:"failure_action"`
}
