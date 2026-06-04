package postgres

import (
	"context"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"

	"gorm.io/gorm"
)

type MeterRepo struct {
	db *gorm.DB
}

func NewMeterRepo(db *gorm.DB) port.MeterRepository {
	return &MeterRepo{db: db}
}

func (r *MeterRepo) FindByCode(ctx context.Context, orgId, code string) (domain.BillableMetric, error) {
	var row billableMetricRow
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("code = ?", code).
		First(&row).Error
	if err != nil {
		return domain.BillableMetric{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *MeterRepo) Create(ctx context.Context, m domain.BillableMetric) (domain.BillableMetric, error) {
	m.Metadata = emptyIfNil(m.Metadata)
	row := billableMetricRowFromDomain(m)
	if err := dbFromCtx(ctx, r.db).Create(&row).Error; err != nil {
		return domain.BillableMetric{}, err
	}
	return r.FindByCode(ctx, m.OrgId, m.Code)
}

func (r *MeterRepo) Find(ctx context.Context, orgId string, p domain.Pagination) ([]domain.BillableMetric, int, error) {
	var rows []billableMetricRow
	var count int64
	if err := dbFromCtx(ctx, r.db).Model(&billableMetricRow{}).
		Scopes(OrgScope(orgId)).
		Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId), Paginate(p)).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return billableMetricRowsToDomain(rows), int(count), nil
}
