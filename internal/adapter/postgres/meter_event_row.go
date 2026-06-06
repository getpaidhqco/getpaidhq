package postgres

import (
	"time"

	"github.com/shopspring/decimal"

	"getpaidhq/internal/core/domain"
)

// meterEventRow is the postgres on-the-wire shape of a MeterEvent. The optional id
// columns are nullable pointers: an absent id is stored as NULL, never "". This keeps
// "no value" unambiguous and lets the dedup index (external_id IS NOT NULL) and the
// customer match key cleanly off real values instead of an empty-string sentinel.
type meterEventRow struct {
	OrgId              string            `gorm:"column:org_id;primaryKey"`
	Id                 string            `gorm:"column:id;primaryKey"`
	CustomerId         *string           `gorm:"column:customer_id"`
	ExternalCustomerId *string           `gorm:"column:external_customer_id"`
	MetricCode         string            `gorm:"column:metric_code"`
	SubscriptionId     *string           `gorm:"column:subscription_id"`
	ExternalId         *string           `gorm:"column:external_id"`
	Metadata           map[string]string `gorm:"column:metadata;serializer:json;type:jsonb"`
	Value              decimal.Decimal   `gorm:"column:value;type:numeric"`
	Timestamp          time.Time         `gorm:"column:timestamp"`
	CreatedAt          time.Time         `gorm:"column:created_at"`
}

func (meterEventRow) TableName() string { return "meter_events" }

// nilIfEmpty guards against storing an empty string in an optional id column.
func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func strOrEmpty(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func (r meterEventRow) toDomain() domain.MeterEvent {
	return domain.MeterEvent{
		OrgId:              r.OrgId,
		Id:                 r.Id,
		CustomerId:         strOrEmpty(r.CustomerId),
		ExternalCustomerId: strOrEmpty(r.ExternalCustomerId),
		MetricCode:         r.MetricCode,
		SubscriptionId:     strOrEmpty(r.SubscriptionId),
		ExternalId:         strOrEmpty(r.ExternalId),
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
		CustomerId:         nilIfEmpty(e.CustomerId),
		ExternalCustomerId: nilIfEmpty(e.ExternalCustomerId),
		MetricCode:         e.MetricCode,
		SubscriptionId:     nilIfEmpty(e.SubscriptionId),
		ExternalId:         nilIfEmpty(e.ExternalId),
		Metadata:           e.Metadata,
		Value:              e.Value,
		Timestamp:          e.Timestamp,
		CreatedAt:          e.CreatedAt,
	}
}
