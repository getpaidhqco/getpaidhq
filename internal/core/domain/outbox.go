package domain

import "time"

// OutboxEvent is one durably-queued domain event awaiting delivery to the
// message broker. Payload holds the fully-encoded PubSubPayload envelope so
// the relay publishes the stored bytes verbatim (no double-wrapping).
//
// There is no status field — state is derived: pending (PublishedAt nil and
// Attempts below the relay's max), failed / left for inspection (PublishedAt
// nil, Attempts at max), published (PublishedAt set).
type OutboxEvent struct {
	Id            int64  // insertion order; assigned by the database
	EventId       string // evt_<id> from the envelope
	OrgId         string
	Topic         string
	Payload       []byte // encoded PubSubPayload envelope
	Attempts      int
	NextAttemptAt *time.Time
	LastError     string
	PublishedAt   *time.Time
	CreatedAt     time.Time
}
