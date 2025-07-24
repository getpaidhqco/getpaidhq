package response

type PaymentLinkResponse struct {
	Id        string                 `json:"id"`
	Slug      string                 `json:"slug"`
	Data      map[string]interface{} `json:"data"`
	Config    map[string]interface{} `json:"config"`
	SingleUse bool                   `json:"single_use"`
	Status    string                 `json:"status"`
	CreatedAt string                 `json:"created_at"`
	UpdatedAt string                 `json:"updated_at"`
	UsedAt    string                 `json:"used_at,omitempty"`
	ExpiresAt string                 `json:"expires_at,omitempty"`
}

type PaymentLinkUsageResponse struct {
	Id           string                 `json:"id"`
	PaymentLinkId string                 `json:"payment_link_id"`
	SessionId    string                 `json:"session_id,omitempty"`
	CustomerId   string                 `json:"customer_id,omitempty"`
	EventType    string                 `json:"event_type"`
	IpAddress    string                 `json:"ip_address,omitempty"`
	UserAgent    string                 `json:"user_agent,omitempty"`
	Referer      string                 `json:"referer,omitempty"`
	Country      string                 `json:"country,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Timestamp    string                 `json:"timestamp"`
}

type PaymentLinkListResponse struct {
	Items []PaymentLinkResponse `json:"items"`
	Meta  Meta                  `json:"meta"`
}

type PaymentLinkUsageListResponse struct {
	Items []PaymentLinkUsageResponse `json:"items"`
	Meta  Meta                       `json:"meta"`
}
