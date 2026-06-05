package handler

import (
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// CreateMeterRequest is the body for POST /meters. A meter defines what usage to
// measure: events reference it by code, metered prices reference it by id.
type CreateMeterRequest struct {
	Code          string                 `json:"code" validate:"required,min=1,max=255"`
	Name          string                 `json:"name" validate:"required,min=1,max=255"`
	Aggregation   domain.AggregationType `json:"aggregation" validate:"required,oneof=count sum max latest weighted_sum unique_count"`
	FieldName     string                 `json:"field_name" validate:"omitempty,max=255"`
	Recurring     bool                   `json:"recurring"`
	RoundingMode  string                 `json:"rounding_mode" validate:"omitempty,oneof=round ceil floor"`
	RoundingScale int                    `json:"rounding_scale" validate:"omitempty,gte=0,lte=18"`
	Metadata      map[string]string      `json:"metadata"`
}

func (r CreateMeterRequest) ToInput(orgId string) port.CreateMeterInput {
	return port.CreateMeterInput{
		OrgId:         orgId,
		Code:          r.Code,
		Name:          r.Name,
		Aggregation:   r.Aggregation,
		FieldName:     r.FieldName,
		Recurring:     r.Recurring,
		RoundingMode:  r.RoundingMode,
		RoundingScale: r.RoundingScale,
		Metadata:      r.Metadata,
	}
}
