package postgres

import (
	"context"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"

	"gorm.io/gorm"
)

type OrgRepo struct {
	db *gorm.DB
}

func NewOrgRepo(db *gorm.DB) port.OrgRepository {
	return &OrgRepo{db: db}
}

func (r *OrgRepo) Create(ctx context.Context, entity domain.Org) (domain.Org, error) {
	row := orgRowFromDomain(entity)
	if err := dbFromCtx(ctx, r.db).Create(&row).Error; err != nil {
		return domain.Org{}, err
	}
	var created orgRow
	if err := dbFromCtx(ctx, r.db).Where("id = ?", entity.Id).First(&created).Error; err != nil {
		return domain.Org{}, translateErr(err)
	}
	return created.toDomain(), nil
}

// ListIds returns all org ids. The billing sweep fans out to every tenant and
// relies on the per-org FindDueForBilling to gate on subscription status, so
// filtering orgs here would silently drop billable subscriptions in
// trial/other-status orgs. Excluding terminal/suspended orgs is a future
// refinement that would be layered here if needed.
func (r *OrgRepo) ListIds(ctx context.Context) ([]string, error) {
	var ids []string
	err := dbFromCtx(ctx, r.db).
		Model(&orgRow{}).
		Pluck("id", &ids).Error
	return ids, err
}
