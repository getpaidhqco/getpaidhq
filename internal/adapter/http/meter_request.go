package handler

import (
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// MeterFilterRequest declares one filterable dimension of a meter: a metadata key and
// the enumerated values that each get their own priced metered Price. A metered Price
// selects one value via filter_field/filter_value; the default charge bills NOT IN
// these. See docs/internal/usage-filters-and-groups.md.
type MeterFilterRequest struct {
	Field  string   `json:"field" validate:"required,min=1,max=255"`
	Values []string `json:"values" validate:"required,min=1,dive,min=1,max=255"`
}

// CreateMeterRequest is the body for POST /meters. A meter defines what usage to
// measure: events reference it by code, metered prices reference it by id.
type CreateMeterRequest struct {
	Code          string                 `json:"code" validate:"required,min=1,max=255"`
	Name          string                 `json:"name" validate:"required,min=1,max=255"`
	Aggregation   domain.AggregationType `json:"aggregation" validate:"required,oneof=count sum max latest weighted_sum unique_count"`
	FieldName     string                 `json:"field_name" validate:"omitempty,max=255"`
	CarryOver     bool                   `json:"carry_over"`
	RoundingMode  string                 `json:"rounding_mode" validate:"omitempty,oneof=round ceil floor"`
	RoundingScale int                    `json:"rounding_scale" validate:"omitempty,gte=0,lte=18"`
	// Filters are the rate dimensions (each value gets its own metered Price). GroupBy
	// are open breakout dimensions (metadata keys): usage is split into one invoice
	// line per discovered value, all at the Price's single rate. v1 honours one group
	// key. (usage-filters-and-groups.md.)
	Filters  []MeterFilterRequest `json:"filters" validate:"omitempty,dive"`
	GroupBy  []string             `json:"group_by" validate:"omitempty,max=1,dive,min=1,max=255"`
	Metadata map[string]string    `json:"metadata"`
}

func (r CreateMeterRequest) ToInput(orgId string) port.CreateMeterInput {
	var filters []domain.MetricFilter
	for _, f := range r.Filters {
		filters = append(filters, domain.MetricFilter{Field: f.Field, Values: f.Values})
	}
	return port.CreateMeterInput{
		OrgId:         orgId,
		Code:          r.Code,
		Name:          r.Name,
		Aggregation:   r.Aggregation,
		FieldName:     r.FieldName,
		CarryOver:     r.CarryOver,
		RoundingMode:  r.RoundingMode,
		RoundingScale: r.RoundingScale,
		Filters:       filters,
		GroupBy:       r.GroupBy,
		Metadata:      r.Metadata,
	}
}
