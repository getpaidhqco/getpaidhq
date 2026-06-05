package port

import (
	"time"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/lib"
)

// CreateMeterInput is the command input for MeterService.Create. A meter (BillableMetric)
// defines what usage to measure (Code, referenced by events) and how to add it up
// (Aggregation over an event Metadata FieldName).
type CreateMeterInput struct {
	OrgId         string
	Code          string
	Name          string
	Aggregation   domain.AggregationType
	FieldName     string
	Recurring     bool
	RoundingMode  string
	RoundingScale int
	Metadata      map[string]string
}

// ToMetric builds a domain.BillableMetric from the input.
func (input CreateMeterInput) ToMetric() domain.BillableMetric {
	now := time.Now().UTC()
	return domain.BillableMetric{
		OrgId:         input.OrgId,
		Id:            lib.GenerateId("met"),
		Code:          input.Code,
		Name:          input.Name,
		Aggregation:   input.Aggregation,
		FieldName:     input.FieldName,
		Recurring:     input.Recurring,
		RoundingMode:  input.RoundingMode,
		RoundingScale: input.RoundingScale,
		Metadata:      input.Metadata,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}
