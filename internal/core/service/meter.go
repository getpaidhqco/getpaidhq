package service

import (
	"context"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

// MeterService manages BillableMetric (meter) definitions. Narrow — no workflow
// engine. Meters are the configuration usage events reference by Code and that
// metered prices reference by Id.
type MeterService struct {
	meterRepository port.MeterRepository
	pubsub          port.PubSub
	logger          port.Logger
}

func NewMeterService(meterRepository port.MeterRepository, pubsub port.PubSub, logger port.Logger) *MeterService {
	return &MeterService{meterRepository: meterRepository, pubsub: pubsub, logger: logger}
}

var validAggregations = map[domain.AggregationType]bool{
	domain.AggregationCount:       true,
	domain.AggregationSum:         true,
	domain.AggregationMax:         true,
	domain.AggregationLatest:      true,
	domain.AggregationWeightedSum: true,
	domain.AggregationUniqueCount: true,
}

var validRoundingModes = map[string]bool{"": true, "round": true, "ceil": true, "floor": true}

// Create validates and stores a meter. Code must be unique per org (the repo's unique
// index enforces it). Every aggregation except count needs a FieldName — the event
// Metadata key it reads (the numeric value for sum/max/latest/weighted_sum, or the
// distinct key for unique_count).
func (s *MeterService) Create(ctx context.Context, in port.CreateMeterInput) (domain.BillableMetric, error) {
	if in.Code == "" {
		return domain.BillableMetric{}, lib.NewCustomError(lib.BadRequestError, "code is required", nil)
	}
	if !validAggregations[in.Aggregation] {
		return domain.BillableMetric{}, lib.NewCustomError(lib.BadRequestError, "unknown aggregation type", nil)
	}
	if in.Aggregation != domain.AggregationCount && in.FieldName == "" {
		return domain.BillableMetric{}, lib.NewCustomError(lib.BadRequestError, "field_name is required for this aggregation", nil)
	}
	if !validRoundingModes[in.RoundingMode] {
		return domain.BillableMetric{}, lib.NewCustomError(lib.BadRequestError, "rounding_mode must be one of round, ceil, floor", nil)
	}

	metric, err := s.meterRepository.Create(ctx, in.ToMetric())
	if err != nil {
		return domain.BillableMetric{}, err
	}
	_ = s.pubsub.Publish(in.OrgId, "meter.created", metric)
	return metric, nil
}

func (s *MeterService) Get(ctx context.Context, orgId, id string) (domain.BillableMetric, error) {
	return s.meterRepository.FindById(ctx, orgId, id)
}

func (s *MeterService) List(ctx context.Context, orgId string, p domain.Pagination) ([]domain.BillableMetric, int, error) {
	return s.meterRepository.Find(ctx, orgId, p)
}
