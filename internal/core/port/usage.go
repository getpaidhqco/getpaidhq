package port

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"getpaidhq/internal/core/domain"
)

// MeterRepository manages BillableMetric (meter) definitions — operational DB.
type MeterRepository interface {
	FindByCode(ctx context.Context, orgId, code string) (domain.BillableMetric, error)
	Create(ctx context.Context, m domain.BillableMetric) (domain.BillableMetric, error)
	Find(ctx context.Context, orgId string, p domain.Pagination) ([]domain.BillableMetric, int, error)
}

// IngestResult reports the outcome of ingesting one usage event.
type IngestResult struct {
	Id        string
	Duplicate bool // true if a resend with the same external_id was ignored
}

// UsageQuery scopes an aggregation. A row matches if customer_id = CustomerId OR
// external_customer_id = ExternalCustomerId (so usage recorded before the customer
// existed is still found). Time range is half-open [From, To).
type UsageQuery struct {
	OrgId      string
	MetricCode string
	FieldName  string // metadata key for sum/max/latest/weighted_sum; the distinct key for unique_count
	From, To   time.Time

	CustomerId         string
	ExternalCustomerId string

	// SubscriptionId set → only events attributed to it; blank → all of the
	// customer's events. IncludeUnattributed also folds in events with no
	// subscription_id (set when this is the customer's earliest metered sub).
	SubscriptionId      string
	IncludeUnattributed bool
}

// EventStore stores usage events and aggregates them. Swappable backend (Postgres
// today; ClickHouse later) in a separate usage datastore.
type EventStore interface {
	Ingest(ctx context.Context, e domain.MeterEvent) (IngestResult, error)
	Count(ctx context.Context, q UsageQuery) (int64, error)
	UniqueCount(ctx context.Context, q UsageQuery) (int64, error)
	Sum(ctx context.Context, q UsageQuery) (decimal.Decimal, error)
	Max(ctx context.Context, q UsageQuery) (decimal.Decimal, error)
	Latest(ctx context.Context, q UsageQuery) (decimal.Decimal, error)
	WeightedSum(ctx context.Context, q UsageQuery, initial decimal.Decimal) (decimal.Decimal, error)
}
