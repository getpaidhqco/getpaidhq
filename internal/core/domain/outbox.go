package domain

import "time"

// OutboxEvent is a durably-queued domain event awaiting broker delivery.
// Payload holds the encoded PubSubPayload envelope. There is no status field:
// pending, failed and published states are derived from PublishedAt/Attempts.
type OutboxEvent struct {
	Id            int64
	EventId       string
	OrgId         string
	Topic         string
	Payload       []byte
	Attempts      int
	NextAttemptAt *time.Time
	LastError     string
	PublishedAt   *time.Time
	CreatedAt     time.Time
}
