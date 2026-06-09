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
	FindById(ctx context.Context, orgId, id string) (domain.BillableMetric, error)
	Create(ctx context.Context, m domain.BillableMetric) (domain.BillableMetric, error)
	Find(ctx context.Context, orgId string, p domain.Pagination) ([]domain.BillableMetric, int, error)
}

// IngestStatus is the outcome of accepting/writing one usage event.
type IngestStatus string

const (
	IngestRecorded  IngestStatus = "recorded"  // written, new
	IngestDuplicate IngestStatus = "duplicate" // resend with a seen external_id, ignored at write
	IngestAccepted  IngestStatus = "accepted"  // durably queued; the write happens asynchronously
	IngestRejected  IngestStatus = "rejected"  // validation failed; never written (batch ingest only)
)

// IngestResult reports the outcome of ingesting one usage event. Error is set
// only for a Rejected result (the validation reason); it is empty otherwise.
type IngestResult struct {
	Id     string
	Status IngestStatus
	Error  string
}

// EventIngestor accepts a fully-validated usage event for durable storage. The
// synchronous adapter is the EventStore itself (direct write); the JetStream adapter
// publishes durably and a background consumer drains into the EventStore. Validation
// has already happened in the service — an ingestor MUST NOT re-validate.
type EventIngestor interface {
	Ingest(ctx context.Context, e domain.MeterEvent) (IngestResult, error)
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

	// Filter scopes the aggregation to one slice of the meter (a metered Price's
	// FilterField/FilterValue). When FilterField is set:
	//   - FilterExclude non-empty → default/catch-all: metadata->>field NOT IN exclude
	//     (and field absent), so unclassified usage is billed exactly once;
	//   - else FilterValue → metadata->>field = value.
	// FilterField == "" applies no filter. See usage-filters-and-groups.md.
	FilterField   string
	FilterValue   string
	FilterExclude []string
}

// GroupedUsage is one segment of a grouped aggregation: a single value of the group
// dimension and its aggregated quantity (e.g. {Key:"project", Value:"acme", Quantity:1000}).
type GroupedUsage struct {
	Key      string
	Value    string
	Quantity decimal.Decimal
}

// EventStore stores usage events and aggregates them. Swappable backend (Postgres
// today; ClickHouse later) in a separate usage datastore.
type EventStore interface {
	Ingest(ctx context.Context, e domain.MeterEvent) (IngestResult, error)
	// IngestBatch writes many events in one round trip; results align by index with
	// events. Used by the async consumer for throughput.
	IngestBatch(ctx context.Context, events []domain.MeterEvent) ([]IngestResult, error)
	Count(ctx context.Context, q UsageQuery) (int64, error)
	UniqueCount(ctx context.Context, q UsageQuery) (int64, error)
	Sum(ctx context.Context, q UsageQuery) (decimal.Decimal, error)
	Max(ctx context.Context, q UsageQuery) (decimal.Decimal, error)
	Latest(ctx context.Context, q UsageQuery) (decimal.Decimal, error)
	WeightedSum(ctx context.Context, q UsageQuery, initial decimal.Decimal) (decimal.Decimal, error)
	// AggregateGrouped aggregates q (honouring its filter) partitioned by a single
	// metadata key, returning one GroupedUsage per distinct value. Supports count, sum,
	// unique_count, max; latest/weighted_sum return an error (need window queries).
	AggregateGrouped(ctx context.Context, q UsageQuery, agg domain.AggregationType, groupKey string) ([]GroupedUsage, error)
}
