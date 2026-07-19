package postgrespgx

import (
	"time"

	"getpaidhq/internal/core/domain"
)

const outboxEventColumns = `id, event_id, org_id, topic, payload, attempts, next_attempt_at, last_error, published_at, created_at`

type outboxEventRow struct {
	Id            int64
	EventId       string
	OrgId         string
	Topic         string
	Payload       []byte
	Attempts      int
	NextAttemptAt *time.Time
	LastError     *string
	PublishedAt   *time.Time
	CreatedAt     time.Time
}

func (r *outboxEventRow) scanInto(s scanner) error {
	return s.Scan(&r.Id, &r.EventId, &r.OrgId, &r.Topic, &r.Payload,
		&r.Attempts, &r.NextAttemptAt, &r.LastError, &r.PublishedAt, &r.CreatedAt)
}

func (r outboxEventRow) toDomain() domain.OutboxEvent {
	ev := domain.OutboxEvent{
		Id:            r.Id,
		EventId:       r.EventId,
		OrgId:         r.OrgId,
		Topic:         r.Topic,
		Payload:       r.Payload,
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
