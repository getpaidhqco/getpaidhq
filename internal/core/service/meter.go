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
	if err := validateFiltersAndGroups(in.Filters, in.GroupBy); err != nil {
		return domain.BillableMetric{}, err
	}
	if err := validateCarryOver(in); err != nil {
		return domain.BillableMetric{}, err
	}

	metric, err := s.meterRepository.Create(ctx, in.ToMetric())
	if err != nil {
		return domain.BillableMetric{}, err
	}
	_ = s.pubsub.Publish(ctx, in.OrgId, "meter.created", metric)
	return metric, nil
}

// validateCarryOver guards carry-over (stock) meters: the aggregation must be one
// of the standing-level readings, and filters/group_by have no defined replay
// semantics on a ledger. docs/internal/billing-model/stock-billing-architecture-impact.md §4.
func validateCarryOver(in port.CreateMeterInput) error {
	if !in.CarryOver {
		// A time-averaged quantity is a standing level — that's what carry_over
		// means. A flow weighted_sum would reset to zero each period and underbill
		// every quiet period, so it is forbidden outright.
		if in.Aggregation == domain.AggregationWeightedSum {
			return lib.NewCustomError(lib.BadRequestError, "weighted_sum requires carry_over: true", nil)
		}
		return nil
	}
	switch in.Aggregation {
	case domain.AggregationLatest, domain.AggregationMax,
		domain.AggregationUniqueCount, domain.AggregationWeightedSum:
	default:
		return lib.NewCustomError(lib.BadRequestError, "aggregation "+string(in.Aggregation)+" is not supported for carry-over meters", nil)
	}
	if len(in.Filters) > 0 {
		return lib.NewCustomError(lib.BadRequestError, "filters are not supported on carry-over meters", nil)
	}
	if len(in.GroupBy) > 0 {
		return lib.NewCustomError(lib.BadRequestError, "group_by is not supported on carry-over meters", nil)
	}
	return nil
}

// validateFiltersAndGroups guards the meter's filter (rate) and group (breakout)
// dimensions: each filter field is non-empty, unique, and has at least one non-empty
// value; group keys are non-empty and distinct from each other; v1 allows at most one
// group dimension. See docs/internal/usage-filters-and-groups.md.
func validateFiltersAndGroups(filters []domain.MetricFilter, groupBy []string) error {
	seenField := map[string]bool{}
	for _, f := range filters {
		if f.Field == "" {
			return lib.NewCustomError(lib.BadRequestError, "filter field is required", nil)
		}
		if seenField[f.Field] {
			return lib.NewCustomError(lib.BadRequestError, "duplicate filter field: "+f.Field, nil)
		}
		seenField[f.Field] = true
		if len(f.Values) == 0 {
			return lib.NewCustomError(lib.BadRequestError, "filter "+f.Field+" needs at least one value", nil)
		}
		seenVal := map[string]bool{}
		for _, v := range f.Values {
			if v == "" {
				return lib.NewCustomError(lib.BadRequestError, "filter "+f.Field+" has an empty value", nil)
			}
			if seenVal[v] {
				return lib.NewCustomError(lib.BadRequestError, "filter "+f.Field+" has a duplicate value: "+v, nil)
			}
			seenVal[v] = true
		}
	}
	if len(groupBy) > 1 {
		return lib.NewCustomError(lib.BadRequestError, "at most one group_by dimension is supported", nil)
	}
	seenKey := map[string]bool{}
	for _, k := range groupBy {
		if k == "" {
			return lib.NewCustomError(lib.BadRequestError, "group_by key is required", nil)
		}
		if seenKey[k] {
			return lib.NewCustomError(lib.BadRequestError, "duplicate group_by key: "+k, nil)
		}
		seenKey[k] = true
	}
	return nil
}

func (s *MeterService) Get(ctx context.Context, orgId, id string) (domain.BillableMetric, error) {
	return s.meterRepository.FindById(ctx, orgId, id)
}

func (s *MeterService) List(ctx context.Context, orgId string, p domain.Pagination) ([]domain.BillableMetric, int, error) {
	return s.meterRepository.Find(ctx, orgId, p)
}
