package postgresgorm

import (
	"database/sql/driver"
	"fmt"
	"time"

	"getpaidhq/internal/core/domain"
)

// jsonbRaw stores pre-encoded JSON bytes in a jsonb column without a
// marshal/unmarshal round-trip.
type jsonbRaw []byte

func (j jsonbRaw) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return string(j), nil
}

func (j *jsonbRaw) Scan(src any) error {
	switch s := src.(type) {
	case nil:
		*j = nil
	case []byte:
		*j = append((*j)[:0], s...)
	case string:
		*j = []byte(s)
	default:
		return fmt.Errorf("jsonbRaw: unsupported source type %T", src)
	}
	return nil
}

type outboxEventRow struct {
	Id            int64      `gorm:"column:id;primaryKey"`
	EventId       string     `gorm:"column:event_id"`
	OrgId         string     `gorm:"column:org_id"`
	Topic         string     `gorm:"column:topic"`
	Payload       jsonbRaw   `gorm:"column:payload;type:jsonb"`
	Attempts      int        `gorm:"column:attempts"`
	NextAttemptAt *time.Time `gorm:"column:next_attempt_at"`
	LastError     *string    `gorm:"column:last_error"`
	PublishedAt   *time.Time `gorm:"column:published_at"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
}

func (outboxEventRow) TableName() string { return "outbox_events" }

func outboxEventRowFromDomain(ev domain.OutboxEvent) outboxEventRow {
	row := outboxEventRow{
		Id:            ev.Id,
		EventId:       ev.EventId,
		OrgId:         ev.OrgId,
		Topic:         ev.Topic,
		Payload:       jsonbRaw(ev.Payload),
		Attempts:      ev.Attempts,
		NextAttemptAt: ev.NextAttemptAt,
		PublishedAt:   ev.PublishedAt,
		CreatedAt:     ev.CreatedAt,
	}
	if ev.LastError != "" {
		row.LastError = &ev.LastError
	}
	return row
}

func (r outboxEventRow) toDomain() domain.OutboxEvent {
	ev := domain.OutboxEvent{
		Id:            r.Id,
		EventId:       r.EventId,
		OrgId:         r.OrgId,
		Topic:         r.Topic,
		Payload:       []byte(r.Payload),
		Attempts:      r.Attempts,
		NextAttemptAt: r.NextAttemptAt,
		PublishedAt:   r.PublishedAt,
		CreatedAt:     r.CreatedAt,
	}
	if r.LastError != nil {
		ev.LastError = *r.LastError
	}
	return ev
}
