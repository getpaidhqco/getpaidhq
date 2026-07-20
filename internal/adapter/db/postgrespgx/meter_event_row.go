package postgrespgx

import (
	"time"

	"github.com/shopspring/decimal"

	"getpaidhq/internal/core/domain"
)

// meterEventRow is the postgres on-the-wire shape of a MeterEvent. The optional
// id columns (customer_id, external_customer_id, subscription_id, external_id)
// are nullable pointers: an absent id is stored as NULL, never "". This keeps
// "no value" unambiguous and lets the dedup unique index and the customer match
// key work off real values instead of an empty-string sentinel.
type meterEventRow struct {
	OrgId              string
	Id                 string
	CustomerId         *string
	ExternalCustomerId *string
	MetricCode         string
	SubscriptionId     *string
	// ExternalId is the dedup key; NULL when absent (never ""). The composite
	// unique index (org_id, external_id) dedups real ids; NULLs are distinct in
	// Postgres, so absent-id events are never deduped.
	ExternalId *string
	Metadata   jsonCol[map[string]string]
	Value      decimal.Decimal
	Timestamp  time.Time
	CreatedAt  time.Time
}

// meterEventColumns is the column list, in insert/select order.
const meterEventColumns = `org_id, id, customer_id, external_customer_id, metric_code, subscription_id, external_id, metadata, value, timestamp, created_at`

func (r *meterEventRow) scanInto(s scanner) error {
	return s.Scan(&r.OrgId, &r.Id, &r.CustomerId, &r.ExternalCustomerId, &r.MetricCode,
		&r.SubscriptionId, &r.ExternalId, &r.Metadata, &r.Value, &r.Timestamp, &r.CreatedAt)
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
		Metadata:           r.Metadata.V,
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
		Metadata:           newJSON(emptyIfNil(e.Metadata)),
		Value:              e.Value,
		Timestamp:          e.Timestamp,
		CreatedAt:          e.CreatedAt,
	}
}
