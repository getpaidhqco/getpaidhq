package response

import "time"

// CloudEventUsageResponse represents the API response for CloudEvent usage recording
type CloudEventUsageResponse struct {
	EventId            string    `json:"event_id"`            // Internal event ID
	OriginalEventId    string    `json:"original_event_id"`   // Original CloudEvent ID
	SubscriptionItemId string    `json:"subscription_item_id"`
	Type               string    `json:"type"`                // CloudEvent type
	Status             string    `json:"status"`              // "recorded", "processing", "calculated"
	RecordedAt         time.Time `json:"recorded_at"`
	Message            string    `json:"message,omitempty"`
}