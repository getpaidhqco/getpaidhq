package entities

import (
	"time"
)

type PaymentLinkUsage struct {
	Id           string    `json:"id"`
	OrgId        string    `json:"org_id"`
	PaymentLinkId string    `json:"payment_link_id"`
	SessionId    string    `json:"session_id,omitempty"`
	CustomerId   string    `json:"customer_id,omitempty"`
	EventType    string    `json:"event_type"`
	IpAddress    string    `json:"ip_address,omitempty"`
	UserAgent    string    `json:"user_agent,omitempty"`
	Referer      string    `json:"referer,omitempty"`
	Country      string    `json:"country,omitempty"`
	Metadata     []byte    `json:"metadata,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}