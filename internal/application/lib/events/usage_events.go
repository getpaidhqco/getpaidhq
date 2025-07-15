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
	OrgId   string `json:"org_id"`
	MeterId string `json:"meter_id"`
	Id      string `json:"id"`
	Source  string `json:"source"`
	Subject string `json:"subject"`

	// Processing metadata
	ReceivedAt time.Time `json:"received_at"`
}

// Event type constant for raw usage events
const (
	RawUsageRecorded = "raw_usage.recorded"
)
