package handler

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// MeterFilterResponse is the API shape of one MetricFilter (a rate dimension).
type MeterFilterResponse struct {
	Field  string   `json:"field"`
	Values []string `json:"values"`
}

// MeterResponse is the API shape of a BillableMetric (meter).
type MeterResponse struct {
	Id            string                 `json:"id"`
	Code          string                 `json:"code"`
	Name          string                 `json:"name"`
	Aggregation   domain.AggregationType `json:"aggregation"`
	FieldName     string                 `json:"field_name"`
	Recurring     bool                   `json:"recurring"`
	RoundingMode  string                 `json:"rounding_mode"`
	RoundingScale int                    `json:"rounding_scale"`
	Filters       []MeterFilterResponse  `json:"filters"`
	GroupBy       []string               `json:"group_by"`
	Metadata      map[string]string      `json:"metadata"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

func NewMeterResponse(m domain.BillableMetric) MeterResponse {
	filters := make([]MeterFilterResponse, 0, len(m.Filters))
	for _, f := range m.Filters {
		filters = append(filters, MeterFilterResponse{Field: f.Field, Values: f.Values})
	}
	return MeterResponse{
		Id:            m.Id,
		Code:          m.Code,
		Name:          m.Name,
		Aggregation:   m.Aggregation,
		FieldName:     m.FieldName,
		Recurring:     m.Recurring,
		RoundingMode:  m.RoundingMode,
		RoundingScale: m.RoundingScale,
		Filters:       filters,
		GroupBy:       m.GroupBy,
		Metadata:      m.Metadata,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
}
