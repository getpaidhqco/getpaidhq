package postgres

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// billableMetricRow is the postgres on-the-wire shape of a BillableMetric.
type billableMetricRow struct {
	OrgId         string                 `gorm:"column:org_id;primaryKey"`
	Id            string                 `gorm:"column:id;primaryKey"`
	Code          string                 `gorm:"column:code"`
	Name          string                 `gorm:"column:name"`
	Aggregation   domain.AggregationType `gorm:"column:aggregation"`
	FieldName     string                 `gorm:"column:field_name"`
	Recurring     bool                   `gorm:"column:recurring"`
	RoundingMode  string                 `gorm:"column:rounding_mode"`
	RoundingScale int                    `gorm:"column:rounding_scale"`
	Metadata      map[string]string      `gorm:"column:metadata;serializer:json"`
	CreatedAt     time.Time              `gorm:"column:created_at"`
	UpdatedAt     time.Time              `gorm:"column:updated_at"`
}

func (billableMetricRow) TableName() string { return "billable_metrics" }

func (r billableMetricRow) toDomain() domain.BillableMetric {
	return domain.BillableMetric{
		OrgId:         r.OrgId,
		Id:            r.Id,
		Code:          r.Code,
		Name:          r.Name,
		Aggregation:   r.Aggregation,
		FieldName:     r.FieldName,
		Recurring:     r.Recurring,
		RoundingMode:  r.RoundingMode,
		RoundingScale: r.RoundingScale,
		Metadata:      r.Metadata,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
	}
}

func billableMetricRowFromDomain(m domain.BillableMetric) billableMetricRow {
	return billableMetricRow{
		OrgId:         m.OrgId,
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

func billableMetricRowsToDomain(rows []billableMetricRow) []domain.BillableMetric {
	out := make([]domain.BillableMetric, len(rows))
	for i, r := range rows {
		out[i] = r.toDomain()
	}
	return out
}
