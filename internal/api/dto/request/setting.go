package request

// CreateSettingRequest is the request body for creating a setting
type CreateSettingRequest struct {
	ParentId string `json:"parent_id" binding:"required"`
	Id       string `json:"id" binding:"required"`
	Type     string `json:"value_type" binding:"required"`
	Value    string `json:"value" binding:"required"`
}

// UpdateSettingRequest is the request body for updating a setting
type UpdateSubscriptionSettingRequest struct {
	EmailReminders  bool `json:"email_reminders"`
	ReminderDays    int  `json:"reminder_days" binding:"gte=0,lte=30"`
	CancelOnFailure bool `json:"cancel_on_failure"`
}

// UpsertSettingRequest is the request body for upserting a setting
// Value is optional for partial updates
type UpsertSettingRequest struct {
	ParentId string `json:"parent_id" binding:"required"`
	Id       string `json:"id" binding:"required"`
	Type     string `json:"value_type"`
	Value    any    `json:"value"`
}
