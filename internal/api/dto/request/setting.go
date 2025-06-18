package request

// CreateSettingRequest is the request body for creating a setting
type CreateSettingRequest struct {
	ParentId string `json:"parent_id" binding:"required"`
	Id       string `json:"id" binding:"required"`
	Type     string `json:"value_type" binding:"required"`
	Value    string `json:"value" binding:"required"`
}

type FailureAction string

const (
	FailureActionCancel       FailureAction = "cancel"
	FailureActionMarkUnpaid   FailureAction = "mark_unpaid"
	FailureActionLeavePastDue FailureAction = "past_due"
)

// UpdateSubscriptionSettingRequest is the request body for updating subscription settings
type UpdateSubscriptionSettingRequest struct {
	EnableInvoicePdfs bool   `json:"enable_invoice_pdfs"`
	InvoicePrefix     string `json:"invoice_prefix"`
	EmailReminders    bool   `json:"email_reminders"`
	ReminderDays      int    `json:"reminder_days" binding:"gte=0,lte=30"`
	CancelOnFailure   bool   `json:"cancel_on_failure"`
	RetryPolicy       struct {
		RetryAttempts int           `json:"attempts"`
		RetryPeriod   int           `json:"retry_period"`
		FailureAction FailureAction `json:"failure_action"`
	} `json:"retry_policy,omitempty"`
}

// UpsertSettingRequest is the request body for upserting a setting
// Value is optional for partial updates
type UpsertSettingRequest struct {
	ParentId string `json:"parent_id" binding:"required"`
	Id       string `json:"id" binding:"required"`
	Type     string `json:"value_type"`
	Value    any    `json:"value"`
}
