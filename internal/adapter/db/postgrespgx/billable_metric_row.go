package postgrespgx

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// billableMetricRow is the postgres on-the-wire shape of a BillableMetric.
// field_name and rounding_mode are nullable TEXT (NULL ↔ ""); filters, group_by
// and metadata are nullable JSONB, mapped explicitly in the mappers.
type billableMetricRow struct {
	OrgId         string
	Id            string
	Code          string
	Name          string
	Aggregation   string
	FieldName     *string
	CarryOver     bool
	RoundingMode  *string
	RoundingScale int
	Filters       jsonCol[[]domain.MetricFilter]
	GroupBy       jsonCol[[]string]
	Metadata      jsonCol[map[string]string]
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

const billableMetricColumns = `org_id, id, code, name, aggregation, field_name, carry_over, rounding_mode, rounding_scale, filters, group_by, metadata, created_at, updated_at`

func (r *billableMetricRow) scanInto(s scanner) error {
	return s.Scan(&r.OrgId, &r.Id, &r.Code, &r.Name, &r.Aggregation, &r.FieldName,
		&r.CarryOver, &r.RoundingMode, &r.RoundingScale, &r.Filters, &r.GroupBy,
		&r.Metadata, &r.CreatedAt, &r.UpdatedAt)
}

func (r billableMetricRow) toDomain() domain.BillableMetric {
	return domain.BillableMetric{
		OrgId:         r.OrgId,
		Id:            r.Id,
		Code:          r.Code,
		Name:          r.Name,
		Aggregation:   domain.AggregationType(r.Aggregation),
		FieldName:     strOrEmpty(r.FieldName),
		CarryOver:     r.CarryOver,
		RoundingMode:  strOrEmpty(r.RoundingMode),
		RoundingScale: r.RoundingScale,
		Filters:       r.Filters.V,
		GroupBy:       r.GroupBy.V,
		Metadata:      r.Metadata.V,
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
		Aggregation:   string(m.Aggregation),
		FieldName:     nilIfEmpty(m.FieldName),
		CarryOver:     m.CarryOver,
		RoundingMode:  nilIfEmpty(m.RoundingMode),
		RoundingScale: m.RoundingScale,
		Filters:       newJSON(m.Filters),
		GroupBy:       newJSON(m.GroupBy),
		Metadata:      newJSON(m.Metadata),
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
