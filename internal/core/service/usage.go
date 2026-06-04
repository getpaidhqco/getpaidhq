package service

import (
	"context"
	"errors"
	"time"

	"github.com/shopspring/decimal"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

// UsageService records usage events and aggregates them. Narrow — no workflow engine.
type UsageService struct {
	meterRepository port.MeterRepository
	eventStore      port.EventStore
	pubsub          port.PubSub
	logger          port.Logger
}

func NewUsageService(
	meterRepository port.MeterRepository,
	eventStore port.EventStore,
	pubsub port.PubSub,
	logger port.Logger,
) *UsageService {
	return &UsageService{
		meterRepository: meterRepository,
		eventStore:      eventStore,
		pubsub:          pubsub,
		logger:          logger,
	}
}

// RecordEvent validates + stores one usage event. Returns the ingest result
// (Duplicate=true when a resend with the same external_id was ignored).
func (s *UsageService) RecordEvent(ctx context.Context, in port.RecordEventInput) (port.IngestResult, error) {
	metric, err := s.meterRepository.FindByCode(ctx, in.OrgId, in.MetricCode)
	if err != nil {
		return port.IngestResult{}, lib.NewCustomError(lib.BadRequestError, "unknown metric code", err)
	}
	if in.CustomerId == "" && in.ExternalCustomerId == "" {
		return port.IngestResult{}, lib.NewCustomError(lib.BadRequestError, "customer_id or external_customer_id is required", nil)
	}

	value := decimal.Zero
	if metric.Aggregation != domain.AggregationCount && metric.Aggregation != domain.AggregationUniqueCount {
		raw, ok := in.Metadata[metric.FieldName]
		if !ok {
			return port.IngestResult{}, lib.NewCustomError(lib.BadRequestError, "metadata is missing the metric field "+metric.FieldName, nil)
		}
		v, perr := decimal.NewFromString(raw)
		if perr != nil {
			return port.IngestResult{}, lib.NewCustomError(lib.BadRequestError, "metric field "+metric.FieldName+" is not numeric", perr)
		}
		value = v
	}

	ts := in.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}

	event := domain.MeterEvent{
		OrgId:              in.OrgId,
		Id:                 lib.GenerateId("mev"),
		CustomerId:         in.CustomerId,
		ExternalCustomerId: in.ExternalCustomerId,
		MetricCode:         in.MetricCode,
		SubscriptionId:     in.SubscriptionId,
		ExternalId:         in.ExternalId,
		Metadata:           in.Metadata,
		Value:              value,
		Timestamp:          ts,
		CreatedAt:          time.Now().UTC(),
	}

	res, err := s.eventStore.Ingest(ctx, event)
	if err != nil {
		s.logger.Error("Failed to ingest usage event", "err", err.Error())
		return port.IngestResult{}, err
	}
	_ = s.pubsub.Publish(in.OrgId, "usage.recorded", event)
	return res, nil
}

// AggregateForPeriod adds up usage for the given query using the metric's aggregation,
// then applies the metric's rounding. The caller builds the UsageQuery (customer +
// window + optional subscription attribution).
func (s *UsageService) AggregateForPeriod(ctx context.Context, metric domain.BillableMetric, q port.UsageQuery) (decimal.Decimal, error) {
	q.FieldName = metric.FieldName
	q.MetricCode = metric.Code

	var (
		units decimal.Decimal
		err   error
	)
	switch metric.Aggregation {
	case domain.AggregationCount:
		var n int64
		n, err = s.eventStore.Count(ctx, q)
		units = decimal.NewFromInt(n)
	case domain.AggregationUniqueCount:
		var n int64
		n, err = s.eventStore.UniqueCount(ctx, q)
		units = decimal.NewFromInt(n)
	case domain.AggregationSum:
		units, err = s.eventStore.Sum(ctx, q)
	case domain.AggregationMax:
		units, err = s.eventStore.Max(ctx, q)
	case domain.AggregationLatest:
		units, err = s.eventStore.Latest(ctx, q)
	case domain.AggregationWeightedSum:
		units, err = s.eventStore.WeightedSum(ctx, q, decimal.Zero)
	default:
		return decimal.Zero, errors.New("unknown aggregation type: " + string(metric.Aggregation))
	}
	if err != nil {
		return decimal.Zero, err
	}
	return applyRounding(metric, units), nil
}

// UsageForSubscription aggregates a metered subscription's usage for [from, to),
// resolving the meter from the price. v1 includes unattributed events (the
// earliest-subscription disambiguation for multiple metered subs is deferred).
func (s *UsageService) UsageForSubscription(ctx context.Context, sub domain.Subscription, price domain.Price, from, to time.Time) (decimal.Decimal, error) {
	metric, err := s.meterRepository.FindById(ctx, sub.OrgId, price.BillableMetricId)
	if err != nil {
		return decimal.Zero, err
	}
	q := port.UsageQuery{
		OrgId:               sub.OrgId,
		CustomerId:          sub.CustomerId,
		From:                from,
		To:                  to,
		SubscriptionId:      sub.Id,
		IncludeUnattributed: true,
	}
	return s.AggregateForPeriod(ctx, metric, q)
}

func applyRounding(metric domain.BillableMetric, units decimal.Decimal) decimal.Decimal {
	scale := int32(metric.RoundingScale)
	switch metric.RoundingMode {
	case "ceil":
		return units.RoundCeil(scale)
	case "floor":
		return units.RoundFloor(scale)
	case "round":
		return units.Round(scale)
	default:
		return units
	}
}
