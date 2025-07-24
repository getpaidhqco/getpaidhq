package request

type CreatePaymentLinkRequest struct {
	Slug      string                 `json:"slug" binding:"required"`
	Data      map[string]interface{} `json:"data"`
	Config    map[string]interface{} `json:"config" binding:"required"`
	SingleUse bool                   `json:"single_use"`
	ExpiresAt string                 `json:"expires_at"`
}

type UpdatePaymentLinkRequest struct {
	Slug      string                 `json:"slug"`
	Data      map[string]interface{} `json:"data"`
	Config    map[string]interface{} `json:"config"`
	SingleUse bool                   `json:"single_use"`
	Status    string                 `json:"status"`
	ExpiresAt string                 `json:"expires_at"`
}

type RecordPaymentLinkUsageRequest struct {
	PaymentLinkId string                 `json:"payment_link_id" binding:"required"`
	SessionId     string                 `json:"session_id"`
	CustomerId    string                 `json:"customer_id"`
	EventType     string                 `json:"event_type" binding:"required"`
	IpAddress     string                 `json:"ip_address"`
	UserAgent     string                 `json:"user_agent"`
	Referer       string                 `json:"referer"`
	Country       string                 `json:"country"`
	Metadata      map[string]interface{} `json:"metadata"`
}