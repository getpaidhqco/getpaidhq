package models

import (
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/domain/entities"
	"time"
)

// UsageEvent represents a raw usage event based on CloudEvents v1.0 specification
// Matches the schema defined in schemas/usage/schema.prisma
type UsageEvent struct {
	// Primary key fields
	OrgId string `json:"org_id"` // @map("org_id")
	Id    string `json:"id"`     // CloudEvent id field

	// Business context (enriched from CloudEvent subject)
	SubscriptionId     string `json:"subscription_id"`     // @map("subscription_id")
	SubscriptionItemId string `json:"subscription_item_id"` // @map("subscription_item_id")
	MeterId            string `json:"meter_id"`            // @map("meter_id")

	// CloudEvents v1.0 fields
	SpecVersion string                 `json:"spec_version"` // @map("spec_version")
	Type        string                 `json:"type"`         // @map("type")
	EventId     string                 `json:"event_id"`     // @map("event_id")
	Time        time.Time              `json:"time"`         // @map("time")
	Source      string                 `json:"source"`       // @map("source")
	Subject     string                 `json:"subject"`      // @map("subject")
	Data        map[string]interface{} `json:"data"`         // @map("data")

	// Audit
	ReceivedAt pgtype.Timestamp `json:"received_at"` // @map("received_at")
}

func (m *UsageEvent) ToEntity() entities.UsageEvent {
	return entities.UsageEvent{
		OrgId:              m.OrgId,
		Id:                 m.Id,
		SubscriptionId:     m.SubscriptionId,
		SubscriptionItemId: m.SubscriptionItemId,
		MeterId:            m.MeterId,
		SpecVersion:        m.SpecVersion,
		Type:               m.Type,
		EventId:            m.EventId,
		Time:               m.Time,
		Source:             m.Source,
		Subject:            m.Subject,
		Data:               m.Data,
		ReceivedAt:         m.ReceivedAt.Time,
	}
}

func UsageEventFromEntity(entity entities.UsageEvent) UsageEvent {
	return UsageEvent{
		OrgId:              entity.OrgId,
		Id:                 entity.Id,
		SubscriptionId:     entity.SubscriptionId,
		SubscriptionItemId: entity.SubscriptionItemId,
		MeterId:            entity.MeterId,
		SpecVersion:        entity.SpecVersion,
		Type:               entity.Type,
		EventId:            entity.EventId,
		Time:               entity.Time,
		Source:             entity.Source,
		Subject:            entity.Subject,
		Data:               entity.Data,
		ReceivedAt:         pgtype.Timestamp{Time: entity.ReceivedAt, Valid: !entity.ReceivedAt.IsZero()},
	}
}
