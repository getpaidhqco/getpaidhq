package handler

import (
	"time"

	"getpaidhq/internal/core/domain"
)

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
	Metadata      map[string]string      `json:"metadata"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

func NewMeterResponse(m domain.BillableMetric) MeterResponse {
	return MeterResponse{
		Id:            m.Id,
		Code:          m.Code,
		Name:          m.Name,
		Aggregation:   m.Aggregation,
		FieldName:     m.FieldName,
		Recurring:     m.Recurring,
		RoundingMode:  m.RoundingMode,
		RoundingScale: m.RoundingScale,
		Metadata:      m.Metadata,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
}
