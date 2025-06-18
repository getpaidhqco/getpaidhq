package settings

import "payloop/internal/domain/entities/subscriptions"

type Subscription struct {
	EmailReminders  bool                      `json:"email_reminders,omitempty" binding:"required"`
	ReminderDays    int                       `json:"reminder_days,omitempty" binding:"gte=0,lte=30"`
	CancelOnFailure bool                      `json:"cancel_on_failure,omitempty" binding:"required"`
	RetryPolicy     subscriptions.RetryPolicy `json:"retry_policy,omitempty"`
}
