package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// EventStore is the Postgres backend for usage events. It uses the operational DB
// handle by default (the USAGE_DATABASE_URL-unset fallback); a separate usage DB or
// the ClickHouse backend can be swapped in behind the port.EventStore interface.
type EventStore struct {
	db *gorm.DB
}

func NewEventStore(db *gorm.DB) port.EventStore {
	return &EventStore{db: db}
}

func (s *EventStore) Ingest(ctx context.Context, e domain.MeterEvent) (port.IngestResult, error) {
	e.Metadata = emptyIfNil(e.Metadata)
	row := meterEventRowFromDomain(e)
	err := dbFromCtx(ctx, s.db).Create(&row).Error
	if err != nil {
		// Dedup: a resend with the same external_id hits the partial unique index.
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return port.IngestResult{Id: e.Id, Duplicate: true}, nil
		}
		return port.IngestResult{}, err
	}
	return port.IngestResult{Id: e.Id}, nil
}

// scope applies the shared WHERE: org + metric + half-open window + match either
// customer id + optional subscription attribution.
func (s *EventStore) scope(ctx context.Context, q port.UsageQuery) *gorm.DB {
	tx := dbFromCtx(ctx, s.db).Model(&meterEventRow{}).
		Where("org_id = ? AND metric_code = ?", q.OrgId, q.MetricCode).
		Where("timestamp >= ? AND timestamp < ?", q.From, q.To).
		Where("(customer_id = ? OR external_customer_id = ?)", q.CustomerId, q.ExternalCustomerId)
	if q.SubscriptionId != "" {
		if q.IncludeUnattributed {
			tx = tx.Where("(subscription_id = ? OR subscription_id = '')", q.SubscriptionId)
		} else {
			tx = tx.Where("subscription_id = ?", q.SubscriptionId)
		}
	}
	return tx
}

func (s *EventStore) Count(ctx context.Context, q port.UsageQuery) (int64, error) {
	var n int64
	err := s.scope(ctx, q).Count(&n).Error
	return n, err
}

func (s *EventStore) UniqueCount(ctx context.Context, q port.UsageQuery) (int64, error) {
	var n int64
	err := s.scope(ctx, q).Select("COUNT(DISTINCT metadata->>?)", q.FieldName).Row().Scan(&n)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return n, err
}

func (s *EventStore) Sum(ctx context.Context, q port.UsageQuery) (decimal.Decimal, error) {
	return s.scalar(ctx, q, "COALESCE(SUM(value),0)")
}

func (s *EventStore) Max(ctx context.Context, q port.UsageQuery) (decimal.Decimal, error) {
	return s.scalar(ctx, q, "COALESCE(MAX(value),0)")
}

func (s *EventStore) Latest(ctx context.Context, q port.UsageQuery) (decimal.Decimal, error) {
	var out decimal.Decimal
	err := s.scope(ctx, q).Select("value").Order("timestamp DESC").Limit(1).Row().Scan(&out)
	if errors.Is(err, sql.ErrNoRows) {
		return decimal.Zero, nil
	}
	return out, err
}

// WeightedSum (value averaged over time) needs a window query; deferred (spec phase 5).
func (s *EventStore) WeightedSum(ctx context.Context, q port.UsageQuery, initial decimal.Decimal) (decimal.Decimal, error) {
	return decimal.Zero, errors.New("weighted_sum aggregation not implemented")
}

func (s *EventStore) scalar(ctx context.Context, q port.UsageQuery, expr string) (decimal.Decimal, error) {
	var out decimal.Decimal
	err := s.scope(ctx, q).Select(expr).Row().Scan(&out)
	if errors.Is(err, sql.ErrNoRows) {
		return decimal.Zero, nil
	}
	return out, err
}
