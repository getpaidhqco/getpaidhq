package service

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// --- fakes (self-contained for the usage tests) ---

type usageMeterRepo struct {
	port.MeterRepository
	byCode map[string]domain.BillableMetric
	byId   map[string]domain.BillableMetric
}

func (r *usageMeterRepo) FindByCode(_ context.Context, _, code string) (domain.BillableMetric, error) {
	if m, ok := r.byCode[code]; ok {
		return m, nil
	}
	return domain.BillableMetric{}, port.ErrNotFound
}

func (r *usageMeterRepo) FindById(_ context.Context, _, id string) (domain.BillableMetric, error) {
	if m, ok := r.byId[id]; ok {
		return m, nil
	}
	return domain.BillableMetric{}, port.ErrNotFound
}

type usageCustomerRepo struct {
	port.CustomerRepository
	byId  map[string]domain.Customer
	byExt map[string]domain.Customer
}

func (r *usageCustomerRepo) FindById(_ context.Context, _, id string) (domain.Customer, error) {
	if c, ok := r.byId[id]; ok {
		return c, nil
	}
	return domain.Customer{}, port.ErrNotFound
}

func (r *usageCustomerRepo) FindByExternalId(_ context.Context, _, ext string) (domain.Customer, error) {
	if c, ok := r.byExt[ext]; ok {
		return c, nil
	}
	return domain.Customer{}, port.ErrNotFound
}

type usageSubRepo struct {
	port.SubscriptionRepository
	metered []domain.Subscription
}

func (r *usageSubRepo) FindActiveMeteredForMeter(_ context.Context, _, _, _ string) ([]domain.Subscription, error) {
	return r.metered, nil
}

type usageEventStore struct {
	ingested  []domain.MeterEvent
	lastQuery port.UsageQuery
	count     int64
	dup       bool
}

func (s *usageEventStore) Ingest(_ context.Context, e domain.MeterEvent) (port.IngestResult, error) {
	s.ingested = append(s.ingested, e)
	return port.IngestResult{Id: e.Id, Duplicate: s.dup}, nil
}
func (s *usageEventStore) Count(_ context.Context, q port.UsageQuery) (int64, error) {
	s.lastQuery = q
	return s.count, nil
}
func (s *usageEventStore) UniqueCount(_ context.Context, q port.UsageQuery) (int64, error) {
	s.lastQuery = q
	return s.count, nil
}
func (s *usageEventStore) Sum(_ context.Context, q port.UsageQuery) (decimal.Decimal, error) {
	s.lastQuery = q
	return decimal.Zero, nil
}
func (s *usageEventStore) Max(_ context.Context, q port.UsageQuery) (decimal.Decimal, error) {
	s.lastQuery = q
	return decimal.Zero, nil
}
func (s *usageEventStore) Latest(_ context.Context, q port.UsageQuery) (decimal.Decimal, error) {
	s.lastQuery = q
	return decimal.Zero, nil
}
func (s *usageEventStore) WeightedSum(_ context.Context, q port.UsageQuery, _ decimal.Decimal) (decimal.Decimal, error) {
	s.lastQuery = q
	return decimal.Zero, nil
}

func countMeter() domain.BillableMetric {
	return domain.BillableMetric{OrgId: "org_1", Id: "met_1", Code: "api_calls", Aggregation: domain.AggregationCount}
}
func sumMeter() domain.BillableMetric {
	return domain.BillableMetric{OrgId: "org_1", Id: "met_2", Code: "gb", Aggregation: domain.AggregationSum, FieldName: "bytes"}
}

func newUsageSvc(m *usageMeterRepo, c *usageCustomerRepo, sub *usageSubRepo, es *usageEventStore) *UsageService {
	return NewUsageService(m, c, sub, es, &recordingPubSub{}, silentLogger{})
}

func TestUsageService_RecordEvent(t *testing.T) {
	meters := &usageMeterRepo{
		byCode: map[string]domain.BillableMetric{"api_calls": countMeter(), "gb": sumMeter()},
		byId:   map[string]domain.BillableMetric{"met_1": countMeter(), "met_2": sumMeter()},
	}
	customers := &usageCustomerRepo{
		byId:  map[string]domain.Customer{"cus_1": {OrgId: "org_1", Id: "cus_1", ExternalId: "ext-1"}},
		byExt: map[string]domain.Customer{"ext-1": {OrgId: "org_1", Id: "cus_1", ExternalId: "ext-1"}},
	}

	t.Run("unknown metric is rejected", func(t *testing.T) {
		es := &usageEventStore{}
		svc := newUsageSvc(meters, customers, &usageSubRepo{}, es)
		_, err := svc.RecordEvent(context.Background(), port.RecordEventInput{OrgId: "org_1", MetricCode: "nope", CustomerId: "cus_1"})
		if err == nil {
			t.Fatal("expected error")
		}
		if len(es.ingested) != 0 {
			t.Error("must not ingest on validation failure")
		}
	})

	t.Run("missing customer identity is rejected", func(t *testing.T) {
		es := &usageEventStore{}
		svc := newUsageSvc(meters, customers, &usageSubRepo{}, es)
		_, err := svc.RecordEvent(context.Background(), port.RecordEventInput{OrgId: "org_1", MetricCode: "api_calls"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("unknown customer_id is 404-style rejected", func(t *testing.T) {
		es := &usageEventStore{}
		svc := newUsageSvc(meters, customers, &usageSubRepo{}, es)
		_, err := svc.RecordEvent(context.Background(), port.RecordEventInput{OrgId: "org_1", MetricCode: "api_calls", CustomerId: "ghost"})
		if err == nil {
			t.Fatal("expected error")
		}
		if len(es.ingested) != 0 {
			t.Error("must not ingest")
		}
	})

	t.Run("count event stored", func(t *testing.T) {
		es := &usageEventStore{}
		svc := newUsageSvc(meters, customers, &usageSubRepo{}, es)
		res, err := svc.RecordEvent(context.Background(), port.RecordEventInput{OrgId: "org_1", MetricCode: "api_calls", CustomerId: "cus_1"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(es.ingested) != 1 || res.Id == "" {
			t.Fatal("event should be ingested")
		}
	})

	t.Run("sum event missing field is rejected", func(t *testing.T) {
		es := &usageEventStore{}
		svc := newUsageSvc(meters, customers, &usageSubRepo{}, es)
		_, err := svc.RecordEvent(context.Background(), port.RecordEventInput{OrgId: "org_1", MetricCode: "gb", CustomerId: "cus_1", Metadata: map[string]string{"other": "1"}})
		if err == nil {
			t.Fatal("expected error for missing metric field")
		}
	})

	t.Run("external_customer_id resolves to internal id", func(t *testing.T) {
		es := &usageEventStore{}
		svc := newUsageSvc(meters, customers, &usageSubRepo{}, es)
		_, err := svc.RecordEvent(context.Background(), port.RecordEventInput{OrgId: "org_1", MetricCode: "api_calls", ExternalCustomerId: "ext-1"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := es.ingested[0].CustomerId; got != "cus_1" {
			t.Errorf("expected resolved customer_id cus_1, got %q", got)
		}
	})

	t.Run("unknown external_customer_id is accepted as orphan", func(t *testing.T) {
		es := &usageEventStore{}
		svc := newUsageSvc(meters, customers, &usageSubRepo{}, es)
		_, err := svc.RecordEvent(context.Background(), port.RecordEventInput{OrgId: "org_1", MetricCode: "api_calls", ExternalCustomerId: "ext-unknown"})
		if err != nil {
			t.Fatalf("orphan event should be accepted: %v", err)
		}
		if es.ingested[0].CustomerId != "" {
			t.Error("orphan event keeps customer_id empty")
		}
	})

	t.Run("subscription not metered for meter is rejected", func(t *testing.T) {
		es := &usageEventStore{}
		// metered set is empty → the named subscription is not valid for this meter.
		svc := newUsageSvc(meters, customers, &usageSubRepo{}, es)
		_, err := svc.RecordEvent(context.Background(), port.RecordEventInput{OrgId: "org_1", MetricCode: "api_calls", CustomerId: "cus_1", SubscriptionId: "sub_x"})
		if err == nil {
			t.Fatal("expected rejection of subscription not metered for meter")
		}
	})

	t.Run("subscription metered for meter is accepted", func(t *testing.T) {
		es := &usageEventStore{}
		subs := &usageSubRepo{metered: []domain.Subscription{{OrgId: "org_1", Id: "sub_x", CustomerId: "cus_1"}}}
		svc := newUsageSvc(meters, customers, subs, es)
		_, err := svc.RecordEvent(context.Background(), port.RecordEventInput{OrgId: "org_1", MetricCode: "api_calls", CustomerId: "cus_1", SubscriptionId: "sub_x"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if es.ingested[0].SubscriptionId != "sub_x" {
			t.Error("attribution should be stored")
		}
	})
}

func TestUsageService_UsageForSubscription_Attribution(t *testing.T) {
	meters := &usageMeterRepo{byId: map[string]domain.BillableMetric{"met_1": countMeter()}}
	customers := &usageCustomerRepo{byId: map[string]domain.Customer{
		"cus_1": {OrgId: "org_1", Id: "cus_1", ExternalId: "ext-1"},
	}}
	price := domain.Price{OrgId: "org_1", Id: "price_1", Category: domain.PriceCategorySubscription, BillableMetricId: "met_1"}
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0)

	subEarliest := domain.Subscription{OrgId: "org_1", Id: "sub_a", CustomerId: "cus_1"}
	subLater := domain.Subscription{OrgId: "org_1", Id: "sub_b", CustomerId: "cus_1"}
	ordered := []domain.Subscription{subEarliest, subLater} // FindActiveMeteredForMeter returns earliest-first

	t.Run("earliest metered sub folds in unattributed usage", func(t *testing.T) {
		es := &usageEventStore{count: 5}
		svc := newUsageSvc(meters, customers, &usageSubRepo{metered: ordered}, es)
		_, err := svc.UsageForSubscription(context.Background(), subEarliest, price, from, to)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !es.lastQuery.IncludeUnattributed {
			t.Error("earliest sub should include unattributed usage")
		}
		if es.lastQuery.ExternalCustomerId != "ext-1" {
			t.Errorf("external customer id should be filled, got %q", es.lastQuery.ExternalCustomerId)
		}
	})

	t.Run("non-earliest metered sub excludes unattributed usage", func(t *testing.T) {
		es := &usageEventStore{count: 5}
		svc := newUsageSvc(meters, customers, &usageSubRepo{metered: ordered}, es)
		_, err := svc.UsageForSubscription(context.Background(), subLater, price, from, to)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if es.lastQuery.IncludeUnattributed {
			t.Error("non-earliest sub must NOT include unattributed usage (avoids double-billing)")
		}
	})
}
