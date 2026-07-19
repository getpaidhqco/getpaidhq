package port

import (
	"getpaidhq/internal/lib/ids"
	"time"

	"getpaidhq/internal/core/domain"
)

// CreateMeterInput is the input for MeterService.Create. A meter (BillableMetric)
// defines what usage to measure (Code, referenced by events) and how to add it up
// (Aggregation over an event Metadata FieldName).
type CreateMeterInput struct {
	OrgId         string
	Code          string
	Name          string
	Aggregation   domain.AggregationType
	FieldName     string
	CarryOver     bool
	RoundingMode  string
	RoundingScale int
	Filters       []domain.MetricFilter
	GroupBy       []string
	Metadata      map[string]string
}

// ToMetric builds a domain.BillableMetric from the input.
func (input CreateMeterInput) ToMetric() domain.BillableMetric {
	now := time.Now().UTC()
	return domain.BillableMetric{
		OrgId:         input.OrgId,
		Id:            ids.Generate("met"),
		Code:          input.Code,
		Name:          input.Name,
		Aggregation:   input.Aggregation,
		FieldName:     input.FieldName,
		CarryOver:     input.CarryOver,
		RoundingMode:  input.RoundingMode,
		RoundingScale: input.RoundingScale,
		Filters:       input.Filters,
		GroupBy:       input.GroupBy,
		Metadata:      input.Metadata,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}
