package postgres

import (
	"time"

	"github.com/shopspring/decimal"

	"getpaidhq/internal/core/domain"
)

// meterEventRow is the postgres on-the-wire shape of a MeterEvent.
type meterEventRow struct {
	OrgId              string            `gorm:"column:org_id;primaryKey"`
	Id                 string            `gorm:"column:id;primaryKey"`
	CustomerId         string            `gorm:"column:customer_id"`
	ExternalCustomerId string            `gorm:"column:external_customer_id"`
	MetricCode         string            `gorm:"column:metric_code"`
	SubscriptionId     string            `gorm:"column:subscription_id"`
	ExternalId         string            `gorm:"column:external_id"`
	Metadata           map[string]string `gorm:"column:metadata;serializer:json;type:jsonb"`
	Value              decimal.Decimal   `gorm:"column:value;type:numeric"`
	Timestamp          time.Time         `gorm:"column:timestamp"`
	CreatedAt          time.Time         `gorm:"column:created_at"`
}

func (meterEventRow) TableName() string { return "meter_events" }

func (r meterEventRow) toDomain() domain.MeterEvent {
	return domain.MeterEvent{
		OrgId:              r.OrgId,
		Id:                 r.Id,
		CustomerId:         r.CustomerId,
		ExternalCustomerId: r.ExternalCustomerId,
		MetricCode:         r.MetricCode,
		SubscriptionId:     r.SubscriptionId,
		ExternalId:         r.ExternalId,
		Metadata:           r.Metadata,
		Value:              r.Value,
		Timestamp:          r.Timestamp,
		CreatedAt:          r.CreatedAt,
	}
}

func meterEventRowFromDomain(e domain.MeterEvent) meterEventRow {
	return meterEventRow{
		OrgId:              e.OrgId,
		Id:                 e.Id,
		CustomerId:         e.CustomerId,
		ExternalCustomerId: e.ExternalCustomerId,
		MetricCode:         e.MetricCode,
		SubscriptionId:     e.SubscriptionId,
		ExternalId:         e.ExternalId,
		Metadata:           e.Metadata,
		Value:              e.Value,
		Timestamp:          e.Timestamp,
		CreatedAt:          e.CreatedAt,
	}
}
