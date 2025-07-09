package events

import (
	"time"
)

// RawUsageRecordedEvent represents a raw usage event for storage
type RawUsageRecordedEvent struct {
	BaseEvent

	// Original CloudEvent data
	Data interface{} `json:"data"`

	// Enriched context (resolved from subject)
	OrgId              string `json:"org_id"`
	SubscriptionId     string `json:"subscription_id"`
	SubscriptionItemId string `json:"subscription_item_id"`
	MeterId            string `json:"meter_id"`

	// Processing metadata
	ReceivedAt time.Time `json:"received_at"`
}

// Event type constant for raw usage events
const (
	RawUsageRecorded = "raw_usage.recorded"
)
