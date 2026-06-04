package postgres

import (
	"context"

	"gorm.io/gorm"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type OrgRepo struct {
	db *gorm.DB
}

func NewOrgRepo(db *gorm.DB) port.OrgRepository {
	return &OrgRepo{db: db}
}

func (r *OrgRepo) Create(ctx context.Context, entity domain.Org) (domain.Org, error) {
	err := dbFromCtx(ctx, r.db).Create(&entity).Error
	if err != nil {
		return domain.Org{}, err
	}
	var created domain.Org
	err = dbFromCtx(ctx, r.db).
		Where("id = ?", entity.Id).
		First(&created).Error
	return created, translateErr(err)
}

// ListIds returns all org ids. The billing sweep fans out to every tenant and
// relies on the per-org FindDueForBilling to gate on subscription status, so
// filtering orgs here would silently drop billable subscriptions in
// trial/other-status orgs. Excluding terminal/suspended orgs is a future
// refinement that would be layered here if needed.
func (r *OrgRepo) ListIds(ctx context.Context) ([]string, error) {
	var ids []string
	err := dbFromCtx(ctx, r.db).
		Model(&domain.Org{}).
		Pluck("id", &ids).Error
	return ids, err
}
